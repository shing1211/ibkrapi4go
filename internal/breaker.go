// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Circuit breaker states, as reported by MetricBreakerState.
const (
	breakerClosed   = 0
	breakerHalfOpen = 1
	breakerOpen     = 2
)

// Breaker is an optional circuit breaker. After threshold consecutive failures
// it short-circuits for cooldown, then allows a single half-open probe.
type Breaker struct {
	threshold int
	cooldown  time.Duration

	// Logger receives state-transition diagnostics. Never nil after NewBreaker.
	Logger *slog.Logger

	metrics Metrics

	mu          sync.Mutex
	consecutive int
	openUntil   time.Time
	probing     bool

	budget *errorBudget
}

// NewBreaker returns a breaker, or nil when threshold<=0 (disabled).
func NewBreaker(threshold int, cooldown time.Duration) *Breaker {
	if threshold <= 0 {
		return nil
	}
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	return &Breaker{threshold: threshold, cooldown: cooldown, Logger: NopLogger()}
}

// errorBudget tracks failures in a sliding window. When the budget is
// exhausted (failures >= threshold within the window), the breaker should trip.
type errorBudget struct {
	mu     sync.Mutex
	window []time.Time
	size   int
	budget int
}

// newErrorBudget returns a sliding-window error budget, or nil when budget<=0
// (disabled). size is the maximum number of entries tracked; budget is the
// failure count that exhausts the budget.
func newErrorBudget(budget, size int) *errorBudget {
	if budget <= 0 || size <= 0 {
		return nil
	}
	return &errorBudget{
		window: make([]time.Time, 0, size),
		size:   size,
		budget: budget,
	}
}

// record adds a failure timestamp and returns true when the budget is
// exhausted (i.e. the breaker should open).
func (eb *errorBudget) record(now time.Time) bool {
	if eb == nil {
		return false
	}
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.window = append(eb.window, now)
	eb.evict(now)
	return len(eb.window) >= eb.budget
}

// evict removes entries that have slid out of the window.
func (eb *errorBudget) evict(now time.Time) {
	// Bound memory: keep at most size entries.
	if len(eb.window) > eb.size {
		eb.window = eb.window[len(eb.window)-eb.size:]
	}
}

// SetErrorBudget installs a sliding-window error budget on the breaker.
// When the budget is exhausted the breaker opens. budget<=0 disables the
// budget. size controls the maximum number of tracked entries and should be
// >= budget. It is safe to call before concurrent use.
func (b *Breaker) SetErrorBudget(budget, size int) {
	if b == nil {
		return
	}
	b.mu.Lock()
	b.budget = newErrorBudget(budget, size)
	b.mu.Unlock()
}

// SetMetrics installs a metrics sink and records the current state. It is safe
// to call after construction and before concurrent use.
func (b *Breaker) SetMetrics(m Metrics) {
	if b == nil {
		return
	}
	b.mu.Lock()
	b.metrics = m
	state, label := breakerClosed, "closed"
	if !b.openUntil.IsZero() {
		if b.probing {
			state, label = breakerHalfOpen, "half-open"
		} else {
			state, label = breakerOpen, "open"
		}
	}
	b.mu.Unlock()
	setGauge(context.Background(), m, MetricBreakerState, float64(state), Attr{Key: "state", Value: label})
}

// Allow reports whether a request may proceed. It returns ErrCircuitOpen while
// the breaker is open.
func (b *Breaker) Allow() error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	if b.openUntil.IsZero() {
		b.mu.Unlock()
		return nil
	}
	if time.Now().Before(b.openUntil) {
		b.mu.Unlock()
		return ErrCircuitOpen
	}
	// Cooldown elapsed: allow one probe.
	if b.probing {
		b.mu.Unlock()
		return ErrCircuitOpen
	}
	b.probing = true
	metrics := b.metrics
	b.mu.Unlock()
	b.Logger.Info("ibkr.breaker half-open")
	setGauge(context.Background(), metrics, MetricBreakerState, breakerHalfOpen, Attr{Key: "state", Value: "half-open"})
	return nil
}

// Record updates the breaker with the outcome of a request.
func (b *Breaker) Record(err error, status int) {
	if b == nil {
		return
	}
	b.mu.Lock()
	if breakerFailure(err, status) {
		wasProbing := b.probing
		b.probing = false
		b.consecutive++
		transitioned := false

		// Check consecutive threshold.
		consecutiveTrip := b.consecutive >= b.threshold

		// Check sliding-window error budget.
		budgetTrip := false
		if b.budget != nil && !consecutiveTrip {
			budgetTrip = b.budget.record(time.Now())
		}

		if consecutiveTrip || budgetTrip {
			if b.openUntil.IsZero() || wasProbing {
				transitioned = true
			}
			b.openUntil = time.Now().Add(b.cooldown)
		}
		consecutive, threshold, cooldown := b.consecutive, b.threshold, b.cooldown
		metrics := b.metrics
		b.mu.Unlock()
		if transitioned {
			b.Logger.Warn("ibkr.breaker open",
				"consecutive", consecutive,
				"threshold", threshold,
				"cooldown", cooldown.String(),
			)
			setGauge(context.Background(), metrics, MetricBreakerState, breakerOpen, Attr{Key: "state", Value: "open"})
		}
		return
	}
	wasOpen := !b.openUntil.IsZero()
	b.consecutive = 0
	b.openUntil = time.Time{}
	b.probing = false
	metrics := b.metrics
	b.mu.Unlock()
	if wasOpen {
		b.Logger.Info("ibkr.breaker closed")
		setGauge(context.Background(), metrics, MetricBreakerState, breakerClosed, Attr{Key: "state", Value: "closed"})
	}
}

// breakerFailure reports whether an outcome counts as a failure.
func breakerFailure(err error, status int) bool {
	if err != nil {
		return true
	}
	return status >= 500
}

// CircuitBreaker returns a middleware that short-circuits when b is open. A nil
// breaker is a no-op.
func CircuitBreaker(b *Breaker) func(http.RoundTripper) http.RoundTripper {
	return func(base http.RoundTripper) http.RoundTripper {
		if b == nil {
			return base
		}
		return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if err := b.Allow(); err != nil {
				return nil, err
			}
			resp, err := base.RoundTrip(req)
			status := 0
			if resp != nil {
				status = resp.StatusCode
			}
			b.Record(err, status)
			return resp, err
		})
	}
}
