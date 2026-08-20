package main

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	slog.Info("worker started")
	runWorker(ctx, ticker)
}

func runWorker(ctx context.Context, ticker *time.Ticker) {
	for {
		select {
		case <-context.Background().Done():
			slog.Info("worker stopped")
			return
		case <-ticker.C:
			slog.Info("maintenance tick")
		}
	}
}
