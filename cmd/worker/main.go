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
	if ctx == nil {
		ctx = context.Background()
	}
	if ticker == nil {
		slog.Warn("worker has no maintenance ticker")
		return
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("worker stopped")
			return
		case <-ticker.C:
			runMaintenance(ctx)
		}
	}
}

func runMaintenance(ctx context.Context) {
	if err := ctx.Err(); err != nil {
		slog.Info("maintenance skipped", "error", err)
		return
	}
	slog.Info("maintenance tick")
}
