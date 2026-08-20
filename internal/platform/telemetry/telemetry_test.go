package telemetry

import "testing"

func TestMetricsSnapshotSeparatesRejectedEvents(t *testing.T) {
	m := New(); m.Accepted.Store(3); m.DeadLettered.Store(2)
	got := m.Snapshot(); if got["accepted"] != 3 || got["dead_lettered"] != 2 { t.Fatalf("metrics=%#v", got) }
}
