// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Breaker is an optional circuit breaker. After threshold consecutive failures
// it short-circuits for cooldown, then allows a single half-open probe.
type Breaker struct {
	threshold int
	cooldown  time.Duration

	// Logger receives state-transition diagnostics. Never nil after NewBreaker.
	Logger *slog.Logger

	mu          sync.Mutex
	consecutive int
	openUntil   time.Time
	probing     bool
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

// Allow reports whether a request may proceed. It returns ErrCircuitOpen while
// the breaker is open.
func (b *Breaker) Allow() error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.openUntil.IsZero() {
		return nil
	}
	if time.Now().Before(b.openUntil) {
		return ErrCircuitOpen
	}
	// Cooldown elapsed: allow one probe.
	if b.probing {
		return ErrCircuitOpen
	}
	b.probing = true
	b.Logger.Info("ibkr.breaker half-open")
	return nil
}

// Record updates the breaker with the outcome of a request.
func (b *Breaker) Record(err error, status int) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if breakerFailure(err, status) {
		wasProbing := b.probing
		b.probing = false
		b.consecutive++
		if b.consecutive >= b.threshold {
			if b.openUntil.IsZero() || wasProbing {
				b.Logger.Warn("ibkr.breaker open",
					"consecutive", b.consecutive,
					"threshold", b.threshold,
					"cooldown", b.cooldown.String(),
				)
			}
			b.openUntil = time.Now().Add(b.cooldown)
		}
		return
	}
	wasOpen := !b.openUntil.IsZero()
	b.consecutive = 0
	b.openUntil = time.Time{}
	b.probing = false
	if wasOpen {
		b.Logger.Info("ibkr.breaker closed")
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
