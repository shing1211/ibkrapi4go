// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shing1211/ibkrapi4go/internal/mockgateway"
)

// TestSessionLifecycle_RateLimitWaits pins what a one-shot client pays to open and
// close a session.
//
// This is a regression test for a defect that passed every gate in this repository
// for two releases: every CLI invocation cost ~2.0s because the hardcoded 1 req/s
// auth bucket blocked twice - once before the auth status poll, and once before the
// best-effort logout, which blocked process exit for a call whose result is
// discarded. Nothing caught it because nothing asserted on pacing, and no linter
// reads wall clock.
//
// It asserts on the rate limiter's own counter rather than on elapsed time. A
// wall-clock assertion would be flaky on a loaded CI runner, and a gate that is wrong
// often gets muted - the same fate as the withdrawn check_money.py gate. The counter
// is exact.
//
// The two cases are paired on purpose, in the same shape as
// TestLimiter_AuthPathsAreSlow and TestLimiter_LogoutIsNotAuthPaced:
//
//   - default:    exactly one wait, the deliberate auth pacing. This is the case that
//     pins the v1.1.26 fix. Mutation-checked: putting /v1/api/logout back into the auth
//     set makes this report 2 waits and fail. The limiter test proves the
//     classification; this proves the real client flow takes that path.
//   - opted out:  a caller that raises the auth rate gets a wait-free lifecycle. Note
//     this case does NOT catch the logout regression - with a burst of 10 it absorbs
//     both auth calls. It pins the opt-out's promise, not the fix.
//
// Both assert the four requests, so neither case can pass by making no requests at
// all - a zero-wait assertion on an empty run is trivially true.
func TestSessionLifecycle_RateLimitWaits(t *testing.T) {
	cases := []struct {
		name      string
		opts      []Option
		wantWaits int64
	}{
		// Listed first in the comment order, and first here: this is the case that
		// catches a regression.
		{"default auth pacing", nil, 1},
		{"opted out of auth pacing", []Option{WithAuthRateLimit(50, 10)}, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(mockgateway.New().Handler())
			t.Cleanup(srv.Close)

			m := NewInMemoryMetrics()
			opts := append([]Option{WithGatewayURL(srv.URL), WithMetrics(m)}, tc.opts...)
			cli, err := NewClient(opts...)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			ctx := context.Background()

			if err := cli.Session().Initialize(ctx); err != nil {
				t.Fatalf("Initialize: %v", err)
			}
			if _, err := cli.Account().List(ctx); err != nil {
				t.Fatalf("List: %v", err)
			}
			if err := cli.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}

			snap := m.Snapshot()

			// Vacuity guard: the flow has to have actually run.
			if got := countSeries(snap.Counters, MetricHTTPRequests); got != 4 {
				t.Errorf("http requests = %d; want 4 (init, auth status, accounts, logout)", got)
			}
			for _, want := range []string{"ssodh/init", "auth/status", "iserver/accounts", "logout"} {
				if !sawPath(snap.Counters, want) {
					t.Errorf("no request to a path containing %q; the lifecycle did not run", want)
				}
			}

			if got := snap.Counters[SeriesKey(MetricRateLimitWaits)]; got != tc.wantWaits {
				t.Errorf("rate-limit waits = %d; want %d", got, tc.wantWaits)
			}
		})
	}
}

// countSeries sums every counter recorded under a metric name, regardless of the
// attributes. Series keys look like "name|k1=v1,k2=v2", so a bare SeriesKey(name)
// only matches observations that carried no attributes - which MetricRateLimitWaits
// does, but MetricHTTPRequests does not.
func countSeries(counters map[string]int64, name string) int64 {
	prefix := name + "|"
	var total int64
	for key, v := range counters {
		if key == name || strings.HasPrefix(key, prefix) {
			total += v
		}
	}
	return total
}

// sawPath reports whether any http-request series mentions path. The path attribute is
// part of the series key, so this is a substring test over the keys.
func sawPath(counters map[string]int64, path string) bool {
	prefix := MetricHTTPRequests + "|"
	for key := range counters {
		if strings.HasPrefix(key, prefix) && strings.Contains(key, path) {
			return true
		}
	}
	return false
}
