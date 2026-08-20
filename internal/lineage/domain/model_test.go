package domain

import "testing"

func TestSnapshotCopiesAttempts(t *testing.T) {
	r := Record{Attempts: []Attempt{{ID: "a1", Stage: "validate"}}}
	s := r.Snapshot(); s.Attempts[0].Stage = "corrupt"
	if r.Attempts[0].Stage != "validate" { t.Fatalf("attempt alias escaped: %#v", r.Attempts) }
}
