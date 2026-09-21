// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestFaultInjection_GatewayTimeout504(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGatewayTimeout)
		json.NewEncoder(w).Encode(map[string]string{"error": "timeout", "message": "gateway timeout"})
	}))
	defer srv.Close()

	tp := NewClientTransport(srv.Client().Transport, TransportConfig{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/v1/api/test", nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	e := ResponseError(resp)
	if e == nil {
		t.Fatal("ResponseError returned nil for 504")
	}
	if e.HTTPStatus != http.StatusGatewayTimeout {
		t.Errorf("HTTPStatus = %d; want %d", e.HTTPStatus, http.StatusGatewayTimeout)
	}
	if e.Code != "timeout" {
		t.Errorf("Code = %q; want %q", e.Code, "timeout")
	}
}

func TestFaultInjection_RateLimit429_WithRetryAfter(t *testing.T) {
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	policy := RetryPolicy{
		MaxAttempts:   3,
		BaseDelay:     10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		Jitter:        false,
		RetryOnStatus: []int{429},
	}
	tp := NewClientTransport(srv.Client().Transport, TransportConfig{Retry: policy})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/v1/api/test", nil)
	start := time.Now()
	resp, err := tp.RoundTrip(req)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()

	if calls.Load() != 2 {
		t.Errorf("calls = %d; want 2 (retry after 429)", calls.Load())
	}
	if elapsed < 900*time.Millisecond {
		t.Errorf("elapsed = %v; want >= 900ms (Retry-After honored)", elapsed)
	}
}

func TestFaultInjection_AuthFailure401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized", "message": "not authenticated"})
	}))
	defer srv.Close()

	tp := NewClientTransport(srv.Client().Transport, TransportConfig{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/v1/api/test", nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	e := ResponseError(resp)
	if e == nil {
		t.Fatal("ResponseError returned nil for 401")
	}
	if !errors.Is(e, ErrSessionExpired) {
		t.Errorf("errors.Is(e, ErrSessionExpired) = false; want true")
	}
	if e.HTTPStatus != http.StatusUnauthorized {
		t.Errorf("HTTPStatus = %d; want %d", e.HTTPStatus, http.StatusUnauthorized)
	}
}

func TestFaultInjection_ServerError500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "INTERNAL", "message": "server error"})
	}))
	defer srv.Close()

	tp := NewClientTransport(srv.Client().Transport, TransportConfig{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/v1/api/test", nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	e := ResponseError(resp)
	if e == nil {
		t.Fatal("ResponseError returned nil for 500")
	}
	if e.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("HTTPStatus = %d; want %d", e.HTTPStatus, http.StatusInternalServerError)
	}
	if e.Code != "INTERNAL" {
		t.Errorf("Code = %q; want %q", e.Code, "INTERNAL")
	}
}

func TestFaultInjection_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `not valid json{{{`)
	}))
	defer srv.Close()

	tp := NewClientTransport(srv.Client().Transport, TransportConfig{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/v1/api/test", nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	e := ResponseError(resp)
	if e == nil {
		t.Fatal("ResponseError returned nil for malformed JSON")
	}
	if e.HTTPStatus != http.StatusBadRequest {
		t.Errorf("HTTPStatus = %d; want %d", e.HTTPStatus, http.StatusBadRequest)
	}
}

func TestFaultInjection_EmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	tp := NewClientTransport(srv.Client().Transport, TransportConfig{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/v1/api/test", nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	e := ResponseError(resp)
	if e == nil {
		t.Fatal("ResponseError returned nil for empty body 500")
	}
	if e.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("HTTPStatus = %d; want %d", e.HTTPStatus, http.StatusInternalServerError)
	}
}

func TestFaultInjection_ConnectionRefused(t *testing.T) {
	tp := NewClientTransport(http.DefaultTransport, TransportConfig{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:1", nil)
	_, err := tp.RoundTrip(req)
	if err == nil {
		t.Fatal("RoundTrip: want connection refused error")
	}
}

func TestFaultInjection_ContextCancelledMidRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tp := NewClientTransport(srv.Client().Transport, TransportConfig{})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/v1/api/test", nil)

	start := time.Now()
	_, err := tp.RoundTrip(req)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("RoundTrip: want context error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v; want context error", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("elapsed = %v; want prompt cancellation", elapsed)
	}
}

func TestFaultInjection_502WithRetry(t *testing.T) {
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	policy := RetryPolicy{
		MaxAttempts:   3,
		BaseDelay:     time.Millisecond,
		MaxDelay:      5 * time.Millisecond,
		Jitter:        false,
		RetryOnStatus: []int{502},
	}
	tp := NewClientTransport(srv.Client().Transport, TransportConfig{Retry: policy})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/v1/api/test", nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()

	if calls.Load() != 3 {
		t.Errorf("calls = %d; want 3 (retried 502s then succeeded)", calls.Load())
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d; want 200", resp.StatusCode)
	}
}

func TestFaultInjection_503WithRetry(t *testing.T) {
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	policy := RetryPolicy{
		MaxAttempts:   3,
		BaseDelay:     time.Millisecond,
		MaxDelay:      5 * time.Millisecond,
		Jitter:        false,
		RetryOnStatus: []int{503},
	}
	tp := NewClientTransport(srv.Client().Transport, TransportConfig{Retry: policy})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/v1/api/test", nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()

	if calls.Load() != 3 {
		t.Errorf("calls = %d; want 3 (retried 503s then succeeded)", calls.Load())
	}
}

func TestFaultInjection_504WithRetry(t *testing.T) {
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	policy := RetryPolicy{
		MaxAttempts:   3,
		BaseDelay:     time.Millisecond,
		MaxDelay:      5 * time.Millisecond,
		Jitter:        false,
		RetryOnStatus: []int{504},
	}
	tp := NewClientTransport(srv.Client().Transport, TransportConfig{Retry: policy})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/v1/api/test", nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()

	if calls.Load() != 3 {
		t.Errorf("calls = %d; want 3 (retried 504s then succeeded)", calls.Load())
	}
}

func TestFaultInjection_RetryExhausted(t *testing.T) {
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "SERVER_ERROR", "message": "persistent failure"})
	}))
	defer srv.Close()

	policy := RetryPolicy{
		MaxAttempts:   3,
		BaseDelay:     time.Millisecond,
		MaxDelay:      5 * time.Millisecond,
		Jitter:        false,
		RetryOnStatus: []int{500},
	}
	tp := NewClientTransport(srv.Client().Transport, TransportConfig{Retry: policy})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/v1/api/test", nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	if calls.Load() != 3 {
		t.Errorf("calls = %d; want 3 (all retries exhausted)", calls.Load())
	}
	e := ResponseError(resp)
	if e == nil {
		t.Fatal("ResponseError returned nil after retry exhaustion")
	}
	if e.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("HTTPStatus = %d; want %d", e.HTTPStatus, http.StatusInternalServerError)
	}
}

func TestFaultInjection_POSTNotRetried(t *testing.T) {
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	policy := RetryPolicy{
		MaxAttempts:   3,
		BaseDelay:     time.Millisecond,
		MaxDelay:      5 * time.Millisecond,
		Jitter:        false,
		RetryOnStatus: []int{500},
	}
	tp := NewClientTransport(srv.Client().Transport, TransportConfig{Retry: policy})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+"/v1/api/test", strings.NewReader("{}"))
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()

	if calls.Load() != 1 {
		t.Errorf("calls = %d; want 1 (POST not retried per ADR 0009)", calls.Load())
	}
}

func TestFaultInjection_IBKRErrorEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-request-id", "req-123")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ibkrErrorEnvelope{
			Error:   "INVALID",
			Message: "invalid conid",
			Code:    "321",
		})
	}))
	defer srv.Close()

	tp := NewClientTransport(srv.Client().Transport, TransportConfig{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/v1/api/test", nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	e := ResponseError(resp)
	if e == nil {
		t.Fatal("ResponseError returned nil for IBKR error envelope")
	}
	if e.HTTPStatus != http.StatusBadRequest {
		t.Errorf("HTTPStatus = %d; want %d", e.HTTPStatus, http.StatusBadRequest)
	}
	if e.Code != "321" {
		t.Errorf("Code = %q; want %q", e.Code, "321")
	}
	if e.Message != "invalid conid" {
		t.Errorf("Message = %q; want %q", e.Message, "invalid conid")
	}
	if e.RequestID != "req-123" {
		t.Errorf("RequestID = %q; want %q", e.RequestID, "req-123")
	}
}

func TestFaultInjection_403MapsToNotAuthenticated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	tp := NewClientTransport(srv.Client().Transport, TransportConfig{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/v1/api/test", nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	e := ResponseError(resp)
	if e == nil {
		t.Fatal("ResponseError returned nil for 403")
	}
	if !errors.Is(e, ErrNotAuthenticated) {
		t.Errorf("errors.Is(e, ErrNotAuthenticated) = false; want true")
	}
}

func TestFaultInjection_404MapsToNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	tp := NewClientTransport(srv.Client().Transport, TransportConfig{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/v1/api/test", nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	e := ResponseError(resp)
	if e == nil {
		t.Fatal("ResponseError returned nil for 404")
	}
	if !errors.Is(e, ErrNotFound) {
		t.Errorf("errors.Is(e, ErrNotFound) = false; want true")
	}
}

func TestFaultInjection_SuccessPassesThrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	tp := NewClientTransport(srv.Client().Transport, TransportConfig{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/v1/api/test", nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d; want 200", resp.StatusCode)
	}
	if e := ResponseError(resp); e != nil {
		t.Errorf("ResponseError = %v; want nil for 200", e)
	}
}

func TestFaultInjection_TransportErrorPropagates(t *testing.T) {
	expectedErr := errors.New("simulated transport failure")
	base := RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, expectedErr
	})
	tp := NewClientTransport(base, TransportConfig{})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.test/x", nil)
	_, err := tp.RoundTrip(req)
	if !errors.Is(err, expectedErr) {
		t.Errorf("err = %v; want %v", err, expectedErr)
	}
}

func TestFaultInjection_TimeoutMiddlewareAppliesDeadline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tp := NewClientTransport(srv.Client().Transport, TransportConfig{Timeout: 50 * time.Millisecond})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/v1/api/test", nil)

	start := time.Now()
	_, err := tp.RoundTrip(req)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("RoundTrip: want timeout error")
	}
	if elapsed > 1*time.Second {
		t.Errorf("elapsed = %v; want prompt timeout", elapsed)
	}
}

func TestFaultInjection_ContextCancelledDuringRetryBackoff(t *testing.T) {
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	policy := RetryPolicy{
		MaxAttempts:   5,
		BaseDelay:     500 * time.Millisecond,
		MaxDelay:      500 * time.Millisecond,
		Jitter:        false,
		RetryOnStatus: []int{500},
	}
	tp := NewClientTransport(srv.Client().Transport, TransportConfig{Retry: policy})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/v1/api/test", nil)

	start := time.Now()
	_, err := tp.RoundTrip(req)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("RoundTrip: want error from context cancellation")
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("elapsed = %v; want prompt cancellation during backoff", elapsed)
	}
}

func TestFaultInjection_HeadersPreservedOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "value")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "err", "message": "msg"})
	}))
	defer srv.Close()

	tp := NewClientTransport(srv.Client().Transport, TransportConfig{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/v1/api/test", nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	if v := resp.Header.Get("X-Custom"); v != "value" {
		t.Errorf("X-Custom = %q; want %q (custom headers lost on error)", v, "value")
	}
}
