// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestBreaker_DisabledReturnsNil(t *testing.T) {
	if NewBreaker(0, time.Second) != nil {
		t.Error("NewBreaker(0) != nil; want nil (disabled)")
	}
	if CircuitBreaker(nil)(http.DefaultTransport) == nil {
		t.Error("CircuitBreaker(nil) returned nil RoundTripper")
	}
}

func TestBreaker_OpensAndHalfOpens(t *testing.T) {
	b := NewBreaker(2, 40*time.Millisecond)

	if err := b.Allow(); err != nil {
		t.Fatalf("Allow (closed): %v", err)
	}
	b.Record(nil, 500)
	if err := b.Allow(); err != nil {
		t.Fatalf("Allow (1 failure): %v", err)
	}
	b.Record(nil, 500)

	if err := b.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("Allow after threshold = %v; want ErrCircuitOpen", err)
	}

	time.Sleep(60 * time.Millisecond)
	if err := b.Allow(); err != nil {
		t.Fatalf("Allow (half-open probe): %v", err)
	}
	// A second concurrent probe is rejected.
	if err := b.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("second probe = %v; want ErrCircuitOpen", err)
	}

	b.Record(nil, 200) // probe succeeds -> closed
	if err := b.Allow(); err != nil {
		t.Fatalf("Allow after recovery: %v", err)
	}
}

func TestBreaker_LogsTransitions(t *testing.T) {
	var buf bytes.Buffer
	b := NewBreaker(1, 40*time.Millisecond)
	b.Logger = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	b.Record(nil, 500) // opens
	time.Sleep(60 * time.Millisecond)
	if err := b.Allow(); err != nil { // half-open probe
		t.Fatalf("Allow (half-open): %v", err)
	}
	b.Record(nil, 200) // closes

	out := buf.String()
	for _, want := range []string{"ibkr.breaker open", "ibkr.breaker half-open", "ibkr.breaker closed"} {
		if !strings.Contains(out, want) {
			t.Errorf("log output missing %q; got %q", want, out)
		}
	}
}

func TestBreaker_SuccessResetsCount(t *testing.T) {
	b := NewBreaker(3, time.Second)
	b.Record(nil, 500)
	b.Record(nil, 500)
	b.Record(nil, 200) // reset
	b.Record(nil, 500)
	if err := b.Allow(); err != nil {
		t.Fatalf("Allow after reset+1 failure: %v", err)
	}
}

func TestCircuitBreakerMiddleware_ShortCircuits(t *testing.T) {
	var calls atomic.Int64
	base := RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		return &http.Response{StatusCode: 500, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	rt := NewClientTransport(base, TransportConfig{Breaker: NewBreaker(1, time.Minute)})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.test/x", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("first RoundTrip: %v", err)
	}
	resp.Body.Close()

	_, err = rt.RoundTrip(req)
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("second RoundTrip = %v; want ErrCircuitOpen", err)
	}
	if calls.Load() != 1 {
		t.Errorf("base calls = %d; want 1 (short-circuited)", calls.Load())
	}
}

func TestBreaker_ErrorBudget_TripsOnBudget(t *testing.T) {
	b := NewBreaker(100, time.Minute)
	b.SetErrorBudget(3, 10)

	for i := 0; i < 2; i++ {
		b.Record(nil, 500)
		if err := b.Allow(); err != nil {
			t.Fatalf("Allow after %d failures: %v", i+1, err)
		}
	}
	// Third failure exhausts the budget.
	b.Record(nil, 500)
	if err := b.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("Allow after budget exhaustion = %v; want ErrCircuitOpen", err)
	}
}

func TestBreaker_ErrorBudget_DoesNotTripBelowBudget(t *testing.T) {
	b := NewBreaker(100, time.Minute)
	b.SetErrorBudget(3, 10)

	b.Record(nil, 500)
	b.Record(nil, 200) // success resets consecutive but budget still counts
	b.Record(nil, 500)

	if err := b.Allow(); err != nil {
		t.Fatalf("Allow with 2 budget failures: %v; want nil", err)
	}
}

func TestBreaker_ErrorBudget_NilBudgetIsNoop(t *testing.T) {
	b := NewBreaker(100, time.Minute)
	// No budget set — consecutive threshold is 100.
	for i := 0; i < 99; i++ {
		b.Record(nil, 500)
	}
	if err := b.Allow(); err != nil {
		t.Fatalf("Allow after 99 failures (threshold=100): %v; want nil", err)
	}
}

func TestBreaker_SetErrorBudget_NilReceiver(t *testing.T) {
	var b *Breaker
	b.SetErrorBudget(5, 10) // should not panic
}

func TestErrorBudget_EvictsOldEntries(t *testing.T) {
	eb := newErrorBudget(2, 5)
	now := time.Now()
	eb.record(now.Add(-10 * time.Second))
	eb.record(now.Add(-9 * time.Second))
	eb.record(now) // third entry; evict should not remove since size=5
	if len(eb.window) != 3 {
		t.Fatalf("window len = %d; want 3", len(eb.window))
	}
}
