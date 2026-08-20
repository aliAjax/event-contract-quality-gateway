package main

import (
	"context"
	"testing"
	"time"
)

func TestRunWorkerStopsWhenContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background()); ticker := time.NewTicker(time.Hour); defer ticker.Stop()
	done := make(chan struct{}); go func() { runWorker(ctx, ticker); close(done) }(); cancel()
	select { case <-done: case <-time.After(100 * time.Millisecond): t.Fatal("worker ignored cancellation") }
}
