package main

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestSignalWaitReturnsCallerCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background()); cancel()
	if err := waitForSignal(ctx, make(chan os.Signal)); !errors.Is(err, context.Canceled) { t.Fatalf("err=%v", err) }
}
