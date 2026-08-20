package domain

import (
	"testing"
	"time"
)

func TestFailedReplayAlwaysAuditsCompletion(t *testing.T) {
	l := Letter{State: Replaying, Attempts: 1, MaxAttempts: 3, Audit: []Audit{{Action: "started"}}}
	l.FinishReplay(false, "operator", time.Unix(10, 0))
	if len(l.Audit) != 2 || l.Audit[1].Action != "replay_failed" { t.Fatalf("audit=%#v", l.Audit) }
}
