package httpapi

import (
	"context"
	"errors"
	"testing"
)

func TestShutdownPropagatesCanceledContext(t *testing.T) {
	s := &Server{shutdownHook: func(ctx context.Context) error { if ctx.Err() != nil { return ctx.Err() }; return errors.New("context was replaced") }}
	ctx, cancel := context.WithCancel(context.Background()); cancel()
	if err := s.Shutdown(ctx); !errors.Is(err, context.Canceled) { t.Fatalf("shutdown error=%v", err) }
}
