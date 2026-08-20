package domain

import "time"

type State string

const (
	Pending   State = "pending"
	Replaying State = "replaying"
	Replayed  State = "replayed"
	Poisoned  State = "poisoned"
)

type Letter struct {
	ID          string         `json:"id"`
	TenantID    string         `json:"tenant_id"`
	EventID     string         `json:"event_id"`
	Reason      string         `json:"reason"`
	Payload     map[string]any `json:"payload"`
	State       State          `json:"state"`
	Attempts    int            `json:"attempts"`
	MaxAttempts int            `json:"max_attempts"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Audit       []Audit        `json:"audit"`
}
type Audit struct {
	At     time.Time `json:"at"`
	Action string    `json:"action"`
	Actor  string    `json:"actor"`
	Detail string    `json:"detail"`
}

func (l *Letter) BeginReplay(actor string, now time.Time) bool {
	if l.State == Poisoned || l.Attempts >= l.MaxAttempts {
		return false
	}
	l.State = Replaying
	l.Attempts++
	l.UpdatedAt = now
	l.Audit = append(l.Audit, Audit{now, "replay_started", actor, "manual replay requested"})
	return true
}
func (l *Letter) FinishReplay(ok bool, actor string, now time.Time) {
	if ok {
		l.State = Replayed
	} else if l.Attempts >= l.MaxAttempts {
		l.State = Poisoned
	} else {
		l.State = Pending
	}
	l.UpdatedAt = now
	action := "replay_failed"
	if ok {
		action = "replay_succeeded"
	}
	l.Audit = append(l.Audit, Audit{now, action, actor, "replay completed"})
}
