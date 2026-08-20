package application

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	dlq "github.com/example/event-contract-quality-gateway/internal/deadletter/domain"
	ing "github.com/example/event-contract-quality-gateway/internal/ingestion/domain"
	lineage "github.com/example/event-contract-quality-gateway/internal/lineage/domain"
)

// FindEvents provides deterministic cursor pagination over accepted and
// dead-lettered events. Cursor values are event IDs, so callers can safely
// resume a page after transient network failures.
func (s *Service) FindEvents(_ context.Context, filter ing.EventFilter, limit int, cursor string) (ing.EventPage, error) {
	if filter.Status != "" && filter.Status != ing.Accepted && filter.Status != ing.Rejected && filter.Status != ing.DeadLettered {
		return ing.EventPage{}, fmt.Errorf("unsupported event status %q", filter.Status)
	}
	if limit < 1 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]ing.Event, 0, len(s.events))
	for _, event := range s.events {
		if !s.matchesEvent(event, filter) {
			continue
		}
		items = append(items, cloneEvent(event))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].ReceivedAt.Equal(items[j].ReceivedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].ReceivedAt.Before(items[j].ReceivedAt)
	})
	start := cursorIndex(items, cursor)
	if start > len(items) {
		start = len(items)
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	page := ing.EventPage{Items: items[start:end], Total: len(items), Limit: limit}
	page.NextCursor = ing.NextCursor(items, end)
	return page, nil
}

func (s *Service) matchesEvent(event ing.Event, filter ing.EventFilter) bool {
	if filter.TenantID != "" && event.TenantID != filter.TenantID {
		return false
	}
	if filter.ContractID != "" && event.ContractID != filter.ContractID {
		return false
	}
	if filter.PartitionKey != "" && event.PartitionKey != filter.PartitionKey {
		return false
	}
	if !filter.OccurredAfter.IsZero() && !event.OccurredAt.After(filter.OccurredAfter) {
		return false
	}
	if !filter.OccurredBefore.IsZero() && !event.OccurredAt.Before(filter.OccurredBefore) {
		return false
	}
	if !filter.ReceivedAfter.IsZero() && !event.ReceivedAt.After(filter.ReceivedAfter) {
		return false
	}
	if !filter.ReceivedBefore.IsZero() && !event.ReceivedAt.Before(filter.ReceivedBefore) {
		return false
	}
	if filter.Search != "" {
		needle := strings.ToLower(filter.Search)
		if !strings.Contains(strings.ToLower(event.ID), needle) && !strings.Contains(strings.ToLower(event.ContractID), needle) {
			return false
		}
	}
	if filter.Status != "" && sEventStatus(event.ID, s.seen) != filter.Status {
		return false
	}
	return true
}

func sEventStatus(eventID string, seen map[string]ing.Receipt) ing.Status {
	for _, receipt := range seen {
		if receipt.EventID == eventID {
			return receipt.Status
		}
	}
	return ""
}

func cursorIndex(items []ing.Event, cursor string) int {
	if cursor == "" {
		return 0
	}
	for i, item := range items {
		if item.ID == cursor {
			return i + 1
		}
	}
	return 0
}

func cloneEvent(event ing.Event) ing.Event {
	event.Payload = cloneFields(event.Payload)
	return event
}

func cloneFields(fields map[string]any) map[string]any {
	if fields == nil {
		return nil
	}
	clone := make(map[string]any, len(fields))
	for key, value := range fields {
		clone[key] = cloneValue(value)
	}
	return clone
}

func cloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneFields(typed)
	case map[string]string:
		out := make(map[string]string, len(typed))
		for key, item := range typed {
			out[key] = item
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = cloneValue(typed[i])
		}
		return out
	case []string:
		return append([]string(nil), typed...)
	case []map[string]any:
		out := make([]map[string]any, len(typed))
		for i := range typed {
			out[i] = cloneFields(typed[i])
		}
		return out
	default:
		return value
	}
}

func (s *Service) GetEvent(_ context.Context, tenant, eventID string) (ing.Event, ing.Receipt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	event, ok := s.events[eventID]
	if !ok || event.TenantID != tenant {
		return ing.Event{}, ing.Receipt{}, fmt.Errorf("event not found")
	}
	for _, receipt := range s.seen {
		if receipt.EventID == eventID {
			return cloneEvent(event), receipt, nil
		}
	}
	return cloneEvent(event), ing.Receipt{EventID: eventID}, nil
}

func (s *Service) Summary(_ context.Context, tenant, contractID string) ing.EventSummary {
	s.mu.Lock()
	defer s.mu.Unlock()
	var summary ing.EventSummary
	summary.TenantID, summary.ContractID = tenant, contractID
	for _, event := range s.events {
		if event.TenantID != tenant || (contractID != "" && event.ContractID != contractID) {
			continue
		}
		summary.Total++
		if summary.FirstReceived.IsZero() || event.ReceivedAt.Before(summary.FirstReceived) {
			summary.FirstReceived = event.ReceivedAt
		}
		if event.ReceivedAt.After(summary.LastReceived) {
			summary.LastReceived = event.ReceivedAt
		}
		receipt := receiptFor(event.ID, s.seen)
		if receipt.Status == ing.Accepted {
			summary.Accepted++
		} else if receipt.Status == ing.DeadLettered {
			summary.DeadLettered++
		}
		if receipt.Duplicate {
			summary.Duplicate++
		}
		if attempts := s.attempts[event.ID]; len(attempts) > 0 {
			for _, attempt := range attempts {
				summary.AverageLatency += float64(attempt.Duration.Microseconds()) / 1000
			}
		}
	}
	if summary.Total > 0 {
		summary.AverageLatency /= float64(summary.Total)
	}
	return summary
}

func receiptFor(eventID string, seen map[string]ing.Receipt) ing.Receipt {
	for _, receipt := range seen {
		if receipt.EventID == eventID {
			return receipt
		}
	}
	return ing.Receipt{}
}

// ReplayMany processes independent dead letters and reports every result,
// allowing operators to retry a batch without losing successful items when a
// single event is malformed.
func (s *Service) ReplayMany(ctx context.Context, ids []string, actor string) []ing.ReplayResult {
	if actor == "" {
		actor = "operator"
	}
	results := make([]ing.ReplayResult, 0, len(ids))
	for _, id := range ids {
		if strings.TrimSpace(id) == "" {
			results = append(results, ing.ReplayResult{LetterID: id, Error: "letter id is required"})
			continue
		}
		results = append(results, s.replayOne(ctx, id, actor))
	}
	return results
}

// ReplayManyForTenant applies the same batch semantics while enforcing the
// tenant boundary at the service layer. HTTP handlers should use this method
// instead of trying to filter IDs in transport code.
func (s *Service) ReplayManyForTenant(ctx context.Context, tenant string, ids []string, actor string) []ing.ReplayResult {
	if actor == "" {
		actor = "operator"
	}
	results := make([]ing.ReplayResult, 0, len(ids))
	for _, id := range ids {
		if strings.TrimSpace(id) == "" {
			results = append(results, ing.ReplayResult{LetterID: id, Error: "letter id is required"})
			continue
		}
		s.mu.Lock()
		letter, exists := s.letters[id]
		s.mu.Unlock()
		if !exists || letter.TenantID != tenant {
			results = append(results, ing.ReplayResult{LetterID: id, Error: "dead letter not found"})
			continue
		}
		results = append(results, s.replayOne(ctx, id, actor))
	}
	return results
}

func (s *Service) replayOne(ctx context.Context, id, actor string) ing.ReplayResult {
	receipt, err := s.Replay(ctx, id, actor)
	result := ing.ReplayResult{LetterID: id, Receipt: receipt}
	if err != nil {
		result.Error = err.Error()
	}
	return result
}

func (s *Service) ListLetters(filter dlq.State, tenant string) []dlq.Letter {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]dlq.Letter, 0, len(s.letters))
	for _, letter := range s.letters {
		if tenant != "" && letter.TenantID != tenant {
			continue
		}
		if filter != "" && letter.State != filter {
			continue
		}
		letter.Payload = cloneFields(letter.Payload)
		letter.Audit = append([]dlq.Audit(nil), letter.Audit...)
		out = append(out, letter)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.Before(out[j].UpdatedAt) })
	return out
}

func (s *Service) LineageSnapshot(eventID string) (lineage.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.lineages[eventID]
	if !ok {
		return lineage.Record{}, fmt.Errorf("lineage not found")
	}
	record.ConsumerIDs = append([]string(nil), record.ConsumerIDs...)
	record.ParentEventIDs = append([]string(nil), record.ParentEventIDs...)
	record.Attempts = append([]lineage.Attempt(nil), record.Attempts...)
	return record, nil
}

func (s *Service) PruneExpiredLineage(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	removed := 0
	for id, record := range s.lineages {
		if !record.RetainUntil.IsZero() && now.After(record.RetainUntil) {
			delete(s.lineages, id)
			removed++
		}
	}
	return removed
}
