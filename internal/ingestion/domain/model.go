package domain

import "time"

type Event struct {
	ID             string         `json:"id"`
	TenantID       string         `json:"tenant_id"`
	ContractID     string         `json:"contract_id"`
	SchemaVersion  int            `json:"schema_version"`
	IdempotencyKey string         `json:"idempotency_key"`
	PartitionKey   string         `json:"partition_key"`
	OccurredAt     time.Time      `json:"occurred_at"`
	Payload        map[string]any `json:"payload"`
	Signature      string         `json:"signature,omitempty"`
	ReceivedAt     time.Time      `json:"received_at"`
}
type Status string

const (
	Accepted     Status = "accepted"
	Rejected     Status = "rejected"
	DeadLettered Status = "dead_lettered"
)

type Attempt struct {
	ID       string        `json:"id"`
	EventID  string        `json:"event_id"`
	Number   int           `json:"number"`
	Status   Status        `json:"status"`
	Reasons  []string      `json:"reasons,omitempty"`
	Duration time.Duration `json:"duration"`
	At       time.Time     `json:"at"`
}
type Receipt struct {
	EventID            string        `json:"event_id"`
	Status             Status        `json:"status"`
	Duplicate          bool          `json:"duplicate"`
	AttemptID          string        `json:"attempt_id"`
	Reasons            []string      `json:"reasons,omitempty"`
	ProcessingDuration time.Duration `json:"processing_duration"`
}
