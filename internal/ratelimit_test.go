// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestEndpointKey(t *testing.T) {
	cases := []struct {
		method, path, want string
	}{
		{"GET", "/v1/api/portfolio/U1234567/summary", "GET /v1/api/portfolio/{}/summary"},
		{"GET", "/v1/api/iserver/account/265598/orders", "GET /v1/api/iserver/account/{}/orders"},
		{"GET", "/v1/api/iserver/account/order/status/12345", "GET /v1/api/iserver/account/order/status/{}"},
		{"GET", "/v1/api/iserver/accounts", "GET /v1/api/iserver/accounts"},
		{"POST", "/v1/api/portfolio/U1234567/positions/invalidate", "POST /v1/api/portfolio/{}/positions/invalidate"},
	}
	for _, tc := range cases {
		if got := EndpointKey(tc.method, tc.path); got != tc.want {
			t.Errorf("EndpointKey(%q, %q) = %q; want %q", tc.method, tc.path, got, tc.want)
		}
	}
}

func TestLimiter_BurstThenThrottle(t *testing.T) {
	l := NewLimiter(10, 3, 0) // 10 rps, burst 3
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < 3; i++ {
		if err := l.Wait(ctx, "GET", "/v1/api/iserver/accounts"); err != nil {
			t.Fatalf("burst wait %d: %v", i, err)
		}
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Errorf("burst consumed %v; want near-instant", elapsed)
	}

	start = time.Now()
	if err := l.Wait(ctx, "GET", "/v1/api/iserver/accounts"); err != nil {
		t.Fatalf("throttled wait: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 50*time.Millisecond {
		t.Errorf("post-burst wait took %v; want >= 50ms (throttled)", elapsed)
	}
}

func TestLimiter_LogsLongWait(t *testing.T) {
	var buf bytes.Buffer
	l := NewLimiter(5, 1, 0) // 200ms/req, burst 1
	l.Logger = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := context.Background()

	if err := l.Wait(ctx, "GET", "/v1/api/portfolio/U1234567/summary"); err != nil {
		t.Fatalf("warmup: %v", err)
	}
	if err := l.Wait(ctx, "GET", "/v1/api/portfolio/U1234567/summary"); err != nil {
		t.Fatalf("throttled wait: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "ibkr.ratelimit wait") || !strings.Contains(out, "/v1/api/portfolio/{}/summary") {
		t.Errorf("log output = %q; want ratelimit wait with normalized path", out)
	}
}

func TestLimiter_SteadyState(t *testing.T) {
	l := NewLimiter(100, 1, 0) // 10ms/req
	ctx := context.Background()
	// Consume the burst.
	if err := l.Wait(ctx, "GET", "/v1/api/iserver/accounts"); err != nil {
		t.Fatalf("warmup: %v", err)
	}
	start := time.Now()
	const n = 4
	for i := 0; i < n; i++ {
		if err := l.Wait(ctx, "GET", "/v1/api/iserver/accounts"); err != nil {
			t.Fatalf("wait %d: %v", i, err)
		}
	}
	// n-1 intervals of ~10ms remain after the burst.
	min := time.Duration(n-1) * 10 * time.Millisecond
	if elapsed := time.Since(start); elapsed < min {
		t.Errorf("steady-state elapsed %v; want >= %v", elapsed, min)
	}
}

func TestLimiter_ContextCancelled(t *testing.T) {
	l := NewLimiter(1, 1, 0)
	ctx := context.Background()
	if err := l.Wait(ctx, "GET", "/v1/api/iserver/accounts"); err != nil {
		t.Fatalf("warmup: %v", err)
	}
	cctx, cancel := context.WithCancel(ctx)
	cancel()
	if err := l.Wait(cctx, "GET", "/v1/api/iserver/accounts"); err == nil {
		t.Error("Wait with cancelled context = nil; want error")
	}
}

func TestLimiter_PerEndpointIsolation(t *testing.T) {
	l := NewLimiter(10, 1, 0)
	ctx := context.Background()
	// Consume the burst on endpoint A.
	if err := l.Wait(ctx, "GET", "/v1/api/iserver/accounts"); err != nil {
		t.Fatalf("A: %v", err)
	}
	// Endpoint B has its own bucket and should not be throttled by A.
	start := time.Now()
	if err := l.Wait(ctx, "GET", "/v1/api/portfolio/accounts"); err != nil {
		t.Fatalf("B: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Errorf("independent endpoint waited %v; want near-instant", elapsed)
	}
}

// TestLimiter_LogoutIsNotAuthPaced pins that the best-effort teardown call does not
// sit in the auth bucket.
//
// /v1/api/logout used to match isAuthPath, so with a 1 rps burst-1 auth bucket a
// logout issued right after a session init waited ~997ms for a token. Every
// CLI invocation pays that twice - once before auth/status, once at exit - and the
// exit wait is the sharper problem, because Session.Close documents its logout as
// best-effort and then blocks process exit on it.
//
// The auth bucket exists to pace credential-touching endpoints. Logout touches no
// credential, its result is discarded, and a 429 there costs nothing, so it belongs
// in the ordinary per-endpoint bucket where WithRateLimit governs it.
func TestLimiter_LogoutIsNotAuthPaced(t *testing.T) {
	l := NewLimiter(1000, 100, 0)
	ctx := context.Background()

	// Consume the auth token exactly as a session init does.
	if err := l.Wait(ctx, "POST", "/v1/api/iserver/auth/ssodh/init"); err != nil {
		t.Fatalf("auth warmup: %v", err)
	}

	start := time.Now()
	if err := l.Wait(ctx, "POST", "/v1/api/logout"); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if elapsed := time.Since(start); elapsed >= 500*time.Millisecond {
		t.Errorf("logout waited %v; want no auth-bucket wait, since it is best-effort", elapsed)
	}
}

// TestLimiter_AuthPathsAreSlow still pins the pacing itself, and the two tests above
// are deliberately paired: one removes a path from the auth bucket, the other guards
// the paths that remain in it. TestLimiter_AuthPathsAreSlow is unchanged.
func TestLimiter_AuthPathsAreSlow(t *testing.T) {
	l := NewLimiter(1000, 100, 0)
	ctx := context.Background()
	if err := l.Wait(ctx, "POST", "/v1/api/iserver/auth/ssodh/init"); err != nil {
		t.Fatalf("auth warmup: %v", err)
	}
	// The auth bucket is a fixed 1 rps with burst 1.
	start := time.Now()
	if err := l.Wait(ctx, "POST", "/v1/api/iserver/auth/status"); err != nil {
		t.Fatalf("auth wait: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 500*time.Millisecond {
		t.Errorf("auth wait took %v; want >= 500ms (1 rps)", elapsed)
	}
}
