package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Address          string
	ShutdownTimeout  time.Duration
	MaxBodyBytes     int64
	WorkerCount      int
	DefaultQuota     int
	HMACSecret       string
	Environment      string
	ReadinessEnabled bool
}

func Load() (Config, error) {
	c := Config{Address: env("ECQG_ADDRESS", ":8080"), ShutdownTimeout: duration("ECQG_SHUTDOWN_TIMEOUT", 15*time.Second), MaxBodyBytes: integer64("ECQG_MAX_BODY_BYTES", 1<<20), WorkerCount: integer("ECQG_WORKER_COUNT", 4), DefaultQuota: integer("ECQG_DEFAULT_QUOTA", 1000), HMACSecret: os.Getenv("ECQG_HMAC_SECRET"), Environment: env("ECQG_ENV", "development"), ReadinessEnabled: boolean("ECQG_READINESS_ENABLED", true)}
	if c.MaxBodyBytes < 1024 {
		return Config{}, fmt.Errorf("max body bytes must be at least 1024")
	}
	if c.WorkerCount < 1 || c.WorkerCount > 128 {
		return Config{}, fmt.Errorf("worker count must be between 1 and 128")
	}
	if c.DefaultQuota < 1 {
		return Config{}, fmt.Errorf("default quota must be positive")
	}
	return c, nil
}
func env(k, d string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return d
}
func integer(k string, d int) int {
	v, e := strconv.Atoi(env(k, strconv.Itoa(d)))
	if e != nil {
		return d
	}
	return v
}
func integer64(k string, d int64) int64 {
	v, e := strconv.ParseInt(env(k, strconv.FormatInt(d, 10)), 10, 64)
	if e != nil {
		return d
	}
	return v
}
func duration(k string, d time.Duration) time.Duration {
	v, e := time.ParseDuration(env(k, d.String()))
	if e != nil {
		return d
	}
	return v
}
func boolean(k string, d bool) bool {
	v, e := strconv.ParseBool(env(k, strconv.FormatBool(d)))
	if e != nil {
		return d
	}
	return v
}
