package domain

import "time"

type EventFilter struct {
	TenantID       string
	ContractID     string
	Status         Status
	PartitionKey   string
	OccurredAfter  time.Time
	OccurredBefore time.Time
	ReceivedAfter  time.Time
	ReceivedBefore time.Time
	Search         string
}

type EventPage struct {
	Items      []Event `json:"items"`
	NextCursor string  `json:"next_cursor,omitempty"`
	Total      int     `json:"total"`
	Limit      int     `json:"limit"`
}

type EventSummary struct {
	TenantID       string    `json:"tenant_id"`
	ContractID     string    `json:"contract_id"`
	Total          int       `json:"total"`
	Accepted       int       `json:"accepted"`
	DeadLettered   int       `json:"dead_lettered"`
	Duplicate      int       `json:"duplicate"`
	AverageLatency float64   `json:"average_latency_ms"`
	FirstReceived  time.Time `json:"first_received,omitempty"`
	LastReceived   time.Time `json:"last_received,omitempty"`
}

type ReplayRequest struct {
	LetterIDs []string `json:"letter_ids"`
	Actor     string   `json:"actor"`
}

type ReplayResult struct {
	LetterID string  `json:"letter_id"`
	Receipt  Receipt `json:"receipt"`
	Error    string  `json:"error,omitempty"`
}

func NextCursor(items []Event, end int) string {
	if end <= 0 || end >= len(items) {
		return ""
	}
	return items[end-1].ID
}
