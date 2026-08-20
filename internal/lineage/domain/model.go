package domain

import "time"

type Record struct {
	EventID           string    `json:"event_id"`
	ContractID        string    `json:"contract_id"`
	ProducerID        string    `json:"producer_id"`
	ConsumerIDs       []string  `json:"consumer_ids"`
	ParentEventIDs    []string  `json:"parent_event_ids"`
	ValidationSummary string    `json:"validation_summary"`
	Attempts          []Attempt `json:"attempts"`
	RetainUntil       time.Time `json:"retain_until"`
	CreatedAt         time.Time `json:"created_at"`
}
type Attempt struct {
	ID       string        `json:"id"`
	Stage    string        `json:"stage"`
	Outcome  string        `json:"outcome"`
	At       time.Time     `json:"at"`
	Duration time.Duration `json:"duration"`
	Detail   string        `json:"detail"`
}
