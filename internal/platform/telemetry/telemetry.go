package telemetry

import (
	"expvar"
	"sync/atomic"
	"time"
)

type Metrics struct {
	Accepted        atomic.Int64
	Rejected        atomic.Int64
	DeadLettered    atomic.Int64
	ReplaySucceeded atomic.Int64
	Started         time.Time
}

func New() *Metrics {
	m := &Metrics{Started: time.Now().UTC()}
	expvar.Publish("events_accepted", expvar.Func(func() any { return m.Accepted.Load() }))
	expvar.Publish("events_rejected", expvar.Func(func() any { return m.Rejected.Load() }))
	expvar.Publish("events_deadlettered", expvar.Func(func() any { return m.DeadLettered.Load() }))
	return m
}
