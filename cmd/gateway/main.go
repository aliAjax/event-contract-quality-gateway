package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	contractapp "github.com/example/event-contract-quality-gateway/internal/contract/application"
	ingestapp "github.com/example/event-contract-quality-gateway/internal/ingestion/application"
	"github.com/example/event-contract-quality-gateway/internal/platform/config"
	"github.com/example/event-contract-quality-gateway/internal/platform/httpapi"
	"github.com/example/event-contract-quality-gateway/internal/platform/store"
	"github.com/example/event-contract-quality-gateway/internal/platform/telemetry"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg, e := config.Load()
	if e != nil {
		slog.Error("invalid configuration", "error", e)
		os.Exit(1)
	}
	ids := func() string { b := make([]byte, 16); _, _ = rand.Read(b); return hex.EncodeToString(b) }
	repo := store.NewMemoryContracts()
	contracts := contractapp.New(repo, ids)
	ingestion := ingestapp.New(repo, cfg.HMACSecret, cfg.DefaultQuota, ids)
	api := httpapi.NewServer(cfg, contracts, ingestion, telemetry.New())
	server := &http.Server{Addr: cfg.Address, Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		slog.Info("gateway listening", "address", cfg.Address, "environment", cfg.Environment)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("shutdown failed", "error", err)
		os.Exit(1)
	}
	slog.Info("gateway stopped")
}
