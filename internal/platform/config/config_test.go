package config

import "testing"

func TestLoadRejectsNonPositiveShutdownTimeout(t *testing.T) {
	t.Setenv("ECQG_SHUTDOWN_TIMEOUT", "0s")
	if _, err := Load(); err == nil { t.Fatal("zero shutdown timeout was accepted") }
}
