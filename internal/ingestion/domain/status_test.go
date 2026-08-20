package domain

import (
	"sync"
	"testing"
)

func TestStatusForReasonsDeadLettersFailures(t *testing.T) {
	start := make(chan struct{})
	statuses := make(chan Status, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			statuses <- StatusForReasons([]string{"schema version is not published"})
		}()
	}
	close(start)
	workers.Wait()
	close(statuses)
	for got := range statuses {
		if got != DeadLettered {
			t.Fatalf("status=%s", got)
		}
	}
}
