// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// rateLimitIdleBeforeSweep is how long a bucket must be idle before it can be
// evicted to bound memory.
const rateLimitIdleBeforeSweep = 5 * time.Minute

// Limiter applies a per-endpoint token bucket plus a client-wide bucket. A
// request waits for both. Auth/session endpoints use a fixed 1 req/s bucket.
type Limiter struct {
	mu       sync.Mutex
	buckets  map[string]*rate.Limiter
	lastUsed map[string]time.Time
	global   *rate.Limiter
	auth     *rate.Limiter

	rps   rate.Limit
	burst int

	// Logger receives wait diagnostics. Never nil after NewLimiter.
	Logger *slog.Logger
}

// NewLimiter builds a limiter. rps<=0 disables per-endpoint limiting;
// globalRPS<=0 disables the global bucket. burst<=0 defaults to 2*rps.
func NewLimiter(rps float64, burst int, globalRPS float64) *Limiter {
	l := &Limiter{
		buckets:  map[string]*rate.Limiter{},
		lastUsed: map[string]time.Time{},
		Logger:   NopLogger(),
	}
	if rps > 0 {
		if burst <= 0 {
			burst = int(2 * rps)
		}
		if burst < 1 {
			burst = 1
		}
		l.rps = rate.Limit(rps)
		l.burst = burst
	}
	if globalRPS > 0 {
		b := int(2 * globalRPS)
		if b < 1 {
			b = 1
		}
		l.global = rate.NewLimiter(rate.Limit(globalRPS), b)
	}
	l.auth = rate.NewLimiter(rate.Limit(1), 1)
	return l
}

// Wait blocks until the request is permitted by the applicable buckets or the
// context is done. Waits longer than 100ms are logged at Debug.
func (l *Limiter) Wait(ctx context.Context, method, path string) error {
	if l == nil {
		return nil
	}
	start := time.Now()
	err := l.wait(ctx, method, path)
	if waited := time.Since(start); waited > 100*time.Millisecond {
		l.Logger.Debug("ibkr.ratelimit wait",
			"method", method,
			"path", normalizePath(path),
			"duration", waited.String(),
		)
	}
	return err
}

func (l *Limiter) wait(ctx context.Context, method, path string) error {
	if l.global != nil {
		if err := l.global.Wait(ctx); err != nil {
			return err
		}
	}
	if isAuthPath(path) {
		return l.auth.Wait(ctx)
	}
	if l.rps <= 0 {
		return nil
	}
	return l.endpoint(method, path).Wait(ctx)
}

func (l *Limiter) endpoint(method, path string) *rate.Limiter {
	key := EndpointKey(method, path)
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()
	if lim, ok := l.buckets[key]; ok {
		l.lastUsed[key] = now
		return lim
	}
	l.sweepLocked(now)
	lim := rate.NewLimiter(l.rps, l.burst)
	l.buckets[key] = lim
	l.lastUsed[key] = now
	return lim
}

// sweepLocked evicts buckets idle longer than the sweep window.
func (l *Limiter) sweepLocked(now time.Time) {
	for k, last := range l.lastUsed {
		if now.Sub(last) > rateLimitIdleBeforeSweep {
			delete(l.buckets, k)
			delete(l.lastUsed, k)
		}
	}
}

// EndpointKey normalizes a concrete path into a stable rate-limit key, replacing
// dynamic segments (ids) with "{}" so per-resource calls share a bucket.
func EndpointKey(method, path string) string {
	return method + " " + normalizePath(path)
}

// normalizePath replaces dynamic path segments (ids) with "{}".
func normalizePath(path string) string {
	segs := strings.Split(strings.Trim(path, "/"), "/")
	for i, s := range segs {
		if isDynamicSegment(s) {
			segs[i] = "{}"
		}
	}
	return "/" + strings.Join(segs, "/")
}

// isDynamicSegment reports whether a path segment looks like an identifier
// (account id, conid, order id, UUID) rather than a static route segment.
func isDynamicSegment(s string) bool {
	if s == "" {
		return false
	}
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}
	// Account ids like U1234567.
	if len(s) >= 3 && (s[0] == 'U' || s[0] == 'D' || s[0] == 'F') {
		if _, err := strconv.Atoi(s[1:]); err == nil {
			return true
		}
	}
	// UUID-shaped (e.g. reply ids).
	if len(s) >= 32 && strings.Count(s, "-") >= 3 {
		return true
	}
	return false
}

// isAuthPath reports whether the path is a session/auth endpoint paced at 1 rps.
func isAuthPath(path string) bool {
	return strings.Contains(path, "/iserver/auth/") ||
		strings.Contains(path, "/v1/api/tickle") ||
		strings.Contains(path, "/v1/api/logout") ||
		strings.Contains(path, "/sso/validate")
}

// RateLimit returns a middleware that waits on l before dispatching.
func RateLimit(l *Limiter) func(http.RoundTripper) http.RoundTripper {
	return func(base http.RoundTripper) http.RoundTripper {
		if l == nil {
			return base
		}
		return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if err := l.Wait(req.Context(), req.Method, req.URL.Path); err != nil {
				return nil, err
			}
			return base.RoundTrip(req)
		})
	}
}
