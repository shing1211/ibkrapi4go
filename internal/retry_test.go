// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func retryPolicy(max int) RetryPolicy {
	return RetryPolicy{
		MaxAttempts:   max,
		BaseDelay:     time.Millisecond,
		MaxDelay:      5 * time.Millisecond,
		Jitter:        false,
		RetryOnStatus: []int{429, 500, 502, 503, 504},
	}
}

func syntheticBase(statuses []int, header func(i int, h http.Header)) (*atomic.Int64, http.RoundTripper) {
	var calls atomic.Int64
	base := RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		i := int(calls.Add(1)) - 1
		if i >= len(statuses) {
			i = len(statuses) - 1
		}
		h := http.Header{}
		if header != nil {
			header(i, h)
		}
		return &http.Response{
			StatusCode: statuses[i],
			Header:     h,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	})
	return &calls, base
}

func doGET(t *testing.T, rt http.RoundTripper) *http.Response {
	t.Helper()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.test/x", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	return resp
}

func TestRetry_GetRetriesThenSucceeds(t *testing.T) {
	calls, base := syntheticBase([]int{500, 200}, nil)
	rt := NewClientTransport(base, TransportConfig{Retry: retryPolicy(3)})
	resp := doGET(t, rt)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d; want 200", resp.StatusCode)
	}
	if calls.Load() != 2 {
		t.Errorf("attempts = %d; want 2", calls.Load())
	}
}

func TestRetry_ExhaustsAttempts(t *testing.T) {
	calls, base := syntheticBase([]int{500}, nil)
	rt := NewClientTransport(base, TransportConfig{Retry: retryPolicy(3)})
	resp := doGET(t, rt)
	defer resp.Body.Close()
	if calls.Load() != 3 {
		t.Errorf("attempts = %d; want 3", calls.Load())
	}
	if e := ResponseError(resp); e == nil || e.HTTPStatus != 500 {
		t.Errorf("ResponseError = %v; want 500", e)
	}
}

func TestRetry_PostNotRetried(t *testing.T) {
	calls, base := syntheticBase([]int{500}, nil)
	rt := NewClientTransport(base, TransportConfig{Retry: retryPolicy(3)})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://example.test/x", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()
	if calls.Load() != 1 {
		t.Errorf("POST attempts = %d; want 1 (ADR 0009)", calls.Load())
	}
}

func TestRetry_401NotRetried(t *testing.T) {
	calls, base := syntheticBase([]int{401}, nil)
	rt := NewClientTransport(base, TransportConfig{Retry: retryPolicy(3)})
	resp := doGET(t, rt)
	resp.Body.Close()
	if calls.Load() != 1 {
		t.Errorf("401 attempts = %d; want 1", calls.Load())
	}
}

func TestRetry_HonorsRetryAfter(t *testing.T) {
	calls, base := syntheticBase([]int{429, 200}, func(i int, h http.Header) {
		if i == 0 {
			h.Set("Retry-After", "1")
		}
	})
	rt := NewClientTransport(base, TransportConfig{Retry: retryPolicy(3)})
	start := time.Now()
	resp := doGET(t, rt)
	elapsed := time.Since(start)
	resp.Body.Close()
	if calls.Load() != 2 {
		t.Fatalf("attempts = %d; want 2", calls.Load())
	}
	if elapsed < 900*time.Millisecond {
		t.Errorf("elapsed = %v; want >= 900ms (Retry-After honored)", elapsed)
	}
}

func TestRetry_ContextCancelledMidBackoff(t *testing.T) {
	_, base := syntheticBase([]int{500}, nil)
	p := RetryPolicy{MaxAttempts: 5, BaseDelay: 2 * time.Second, MaxDelay: 2 * time.Second, RetryOnStatus: []int{500}}
	rt := NewClientTransport(base, TransportConfig{Retry: p})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.test/x", nil)

	start := time.Now()
	_, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatal("want error after context cancellation")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("returned after %v; want prompt cancellation", elapsed)
	}
}

func TestParseRetryAfter(t *testing.T) {
	if d, ok := parseRetryAfter("2"); !ok || d != 2*time.Second {
		t.Errorf("parseRetryAfter(2) = %v, %v; want 2s, true", d, ok)
	}
	future := time.Now().Add(3 * time.Second).UTC().Format(http.TimeFormat)
	if d, ok := parseRetryAfter(future); !ok || d < 2*time.Second {
		t.Errorf("parseRetryAfter(date) = %v, %v; want ~3s, true", d, ok)
	}
	if _, ok := parseRetryAfter("not-a-date"); ok {
		t.Error("parseRetryAfter(invalid) ok = true; want false")
	}
}
