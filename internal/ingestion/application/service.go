package application

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	contract "github.com/example/event-contract-quality-gateway/internal/contract/domain"
	dlq "github.com/example/event-contract-quality-gateway/internal/deadletter/domain"
	ing "github.com/example/event-contract-quality-gateway/internal/ingestion/domain"
	lineage "github.com/example/event-contract-quality-gateway/internal/lineage/domain"
	quality "github.com/example/event-contract-quality-gateway/internal/quality/domain"
	"sort"
	"sync"
	"time"
)

type ContractLookup interface {
	Find(context.Context, string, string) (contract.Contract, error)
}
type Service struct {
	contracts ContractLookup
	secret    string
	maxQuota  int
	mu        sync.Mutex
	seen      map[string]ing.Receipt
	events    map[string]ing.Event
	attempts  map[string][]ing.Attempt
	letters   map[string]dlq.Letter
	lineages  map[string]lineage.Record
	rules     map[string][]quality.Rule
	catalog   *quality.Catalog
	usage     map[string]quota
	ids       func() string
	now       func() time.Time
}
type quota struct {
	start time.Time
	count int
}

func New(contracts ContractLookup, secret string, maxQuota int, ids func() string) *Service {
	return &Service{contracts: contracts, secret: secret, maxQuota: maxQuota, seen: map[string]ing.Receipt{}, events: map[string]ing.Event{}, attempts: map[string][]ing.Attempt{}, letters: map[string]dlq.Letter{}, lineages: map[string]lineage.Record{}, rules: map[string][]quality.Rule{}, catalog: quality.NewCatalog(10), usage: map[string]quota{}, ids: ids, now: func() time.Time { return time.Now().UTC() }}
}
func (s *Service) SetRules(contractID string, rules []quality.Rule) error {
	for _, r := range rules {
		if e := r.Validate(); e != nil {
			return e
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	version := s.catalog.NextVersion(contractID)
	if err := s.catalog.Put(quality.RuleSet{ContractID: contractID, Version: version, Rules: rules, Active: true, CreatedAt: s.now(), CreatedBy: "api"}); err != nil {
		return err
	}
	s.rules[contractID] = append([]quality.Rule(nil), rules...)
	return nil
}
func (s *Service) Publish(ctx context.Context, e ing.Event) (ing.Receipt, error) {
	start := s.now()
	if e.ID == "" {
		e.ID = s.ids()
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = start
	}
	e.ReceivedAt = start
	if e.IdempotencyKey == "" {
		return ing.Receipt{}, fmt.Errorf("Idempotency-Key is required")
	}
	s.mu.Lock()
	if r, ok := s.seen[e.TenantID+"/"+e.IdempotencyKey]; ok {
		r.Duplicate = true
		s.mu.Unlock()
		return r, nil
	}
	s.mu.Unlock()
	if err := s.consumeQuota(e.TenantID, start); err != nil {
		return ing.Receipt{}, err
	}
	reasons := s.validate(ctx, e)
	status := ing.StatusForReasons(reasons)
	receipt := ing.Receipt{EventID: e.ID, Status: status, AttemptID: s.ids(), Reasons: reasons, ProcessingDuration: s.now().Sub(start)}
	attempt := ing.Attempt{ID: receipt.AttemptID, EventID: e.ID, Number: 1, Status: status, Reasons: reasons, Duration: receipt.ProcessingDuration, At: s.now()}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events[e.ID] = e
	s.seen[e.TenantID+"/"+e.IdempotencyKey] = receipt
	s.attempts[e.ID] = append(s.attempts[e.ID], attempt)
	summary := "passed"
	if len(reasons) > 0 {
		summary = "failed: " + reasons[0]
		letter := dlq.Letter{ID: s.ids(), TenantID: e.TenantID, EventID: e.ID, Reason: reasons[0], Payload: e.Payload, State: dlq.Pending, MaxAttempts: 3, CreatedAt: s.now(), UpdatedAt: s.now()}
		letter.Audit = []dlq.Audit{{At: s.now(), Action: "created", Actor: "gateway", Detail: "validation failure"}}
		s.letters[letter.ID] = letter
	}
	s.lineages[e.ID] = lineage.Record{EventID: e.ID, ContractID: e.ContractID, ValidationSummary: summary, Attempts: []lineage.Attempt{{ID: attempt.ID, Stage: "validation", Outcome: string(status), At: attempt.At, Duration: attempt.Duration, Detail: summary}}, RetainUntil: s.now().Add(30 * 24 * time.Hour), CreatedAt: s.now()}
	return receipt, nil
}
func (s *Service) consumeQuota(tenant string, now time.Time) error {
	q := s.usage[tenant]
	if q.start.IsZero() || now.Sub(q.start) >= time.Minute {
		q = quota{start: now}
	}
	if q.count >= s.maxQuota {
		return fmt.Errorf("tenant quota exceeded")
	}
	time.Sleep(time.Microsecond)
	q.count++
	s.usage[tenant] = q
	return nil
}
func (s *Service) validate(ctx context.Context, e ing.Event) []string {
	out := []string{}
	if e.TenantID == "" {
		return []string{"tenant is required"}
	}
	if e.ContractID == "" {
		return []string{"contract id is required"}
	}
	if len(e.Payload) == 0 {
		out = append(out, "payload is required")
	}
	if s.secret != "" && !validSignature(s.secret, e) {
		out = append(out, "invalid signature")
	}
	if s.now().Sub(e.OccurredAt) > 24*time.Hour || e.OccurredAt.Sub(s.now()) > 5*time.Minute {
		out = append(out, "event timestamp outside allowed window")
	}
	c, err := s.contracts.Find(ctx, e.TenantID, e.ContractID)
	if err != nil {
		return append(out, "contract not found")
	}
	var version *contract.Version
	for i := range c.Versions {
		if c.Versions[i].Number == e.SchemaVersion {
			version = &c.Versions[i]
			break
		}
	}
	if version == nil || version.Lifecycle != contract.Published {
		return append(out, "schema version is not published")
	}
	for _, issue := range quality.ValidatePayload(e.Payload, version.Schema.Fields) {
		out = append(out, "field "+issue.Field+": "+issue.Reason+" (expected "+issue.Expected+", got "+issue.Actual+")")
	}
	s.mu.Lock()
	rules := append([]quality.Rule(nil), s.rules[e.ContractID]...)
	s.mu.Unlock()
	for _, r := range rules {
		if !r.Enabled {
			continue
		}
		res := r.Execute(e.Payload)
		if !res.Passed {
			out = append(out, "rule "+r.ID+": "+res.Reason)
		}
	}
	sort.Strings(out)
	return out
}
func validSignature(secret string, e ing.Event) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(e.ID + "|" + e.ContractID + "|" + e.IdempotencyKey))
	return hmac.Equal([]byte(e.Signature), []byte(hex.EncodeToString(mac.Sum(nil))))
}
func (s *Service) GetLineage(id string) (lineage.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.lineages[id]
	if !ok {
		return lineage.Record{}, fmt.Errorf("event lineage not found")
	}
	return v, nil
}
func (s *Service) Attempts(id string) []ing.Attempt {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]ing.Attempt(nil), s.attempts[id]...)
}
func (s *Service) Quality(id string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.attempts[id]
	if !ok {
		return nil, fmt.Errorf("event not found")
	}
	if len(a) == 0 {
		return []string{}, nil
	}
	return append([]string(nil), a[len(a)-1].Reasons...), nil
}
func (s *Service) Replay(ctx context.Context, letterID, actor string) (ing.Receipt, error) {
	s.mu.Lock()
	l, ok := s.letters[letterID]
	if !ok {
		s.mu.Unlock()
		return ing.Receipt{}, fmt.Errorf("dead letter not found")
	}
	if !l.BeginReplay(actor, s.now()) {
		s.mu.Unlock()
		return ing.Receipt{}, fmt.Errorf("dead letter cannot be replayed")
	}
	e := s.events[l.EventID]
	e.IdempotencyKey = s.ids()
	s.letters[letterID] = l
	s.mu.Unlock()
	r, err := s.Publish(ctx, e)
	s.mu.Lock()
	l = s.letters[letterID]
	l.FinishReplay(err == nil && r.Status == ing.Accepted, actor, s.now())
	s.letters[letterID] = l
	s.mu.Unlock()
	return r, err
}
func (s *Service) Letters(tenant string) []dlq.Letter {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []dlq.Letter{}
	for _, v := range s.letters {
		if v.TenantID == tenant {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}
