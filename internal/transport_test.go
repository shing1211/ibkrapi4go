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
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	code := m.Run()
	time.Sleep(50 * time.Millisecond)
	if code == 0 {
		goleak.Find()
	}
	os.Exit(code)
}

func TestTransport_PassthroughSwitchingProtocols(t *testing.T) {
	base := RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusSwitchingProtocols,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader("upgraded")),
		}, nil
	})
	tp := NewClientTransport(base, TransportConfig{})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.test/ws", nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d; want 101", resp.StatusCode)
	}
	if resp.Header.Get("X-ibkr-error") != "" {
		t.Errorf("101 was treated as an error: %q", resp.Header.Get("X-ibkr-error"))
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != "upgraded" {
		t.Errorf("body = %q; want %q (upgrade connection was mangled)", body, "upgraded")
	}
}

func TestTransport_RequestID(t *testing.T) {
	var capturedID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = r.Header.Get("X-request-id")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	var idCounter atomic.Int64
	tp := NewClientTransport(srv.Client().Transport, TransportConfig{
		RequestID: func() string {
			return "req-" + string(rune('0'+idCounter.Add(1)))
		},
	})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()
	if capturedID == "" {
		t.Errorf("Request ID not injected; got %q", capturedID)
	}
}

func TestTransport_UserAgent(t *testing.T) {
	var capturedUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tp := NewClientTransport(srv.Client().Transport, TransportConfig{UserAgent: "test-agent/1.0"})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()
	if capturedUA != "test-agent/1.0" {
		t.Errorf("User-Agent = %q; want %q", capturedUA, "test-agent/1.0")
	}
}

func TestTransport_Auth(t *testing.T) {
	var capturedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tp := NewClientTransport(srv.Client().Transport, TransportConfig{
		AuthHeader: "Authorization",
		Token:      func() (string, bool) { return "Bearer secret-token", true },
	})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()
	if capturedAuth != "Bearer secret-token" {
		t.Errorf("Authorization = %q; want %q", capturedAuth, "Bearer secret-token")
	}
}

func TestTransport_Auth_NotSet(t *testing.T) {
	sent := atomic.Bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			sent.Store(true)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tp := NewClientTransport(srv.Client().Transport, TransportConfig{
		AuthHeader: "Authorization",
		Token:      func() (string, bool) { return "", false },
	})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()
	if sent.Load() {
		t.Error("Authorization header sent when token provider returned ok=false")
	}
}

func TestTransport_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tp := NewClientTransport(srv.Client().Transport, TransportConfig{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Errorf("Status = %d; want 2xx", resp.StatusCode)
	}
}

func TestTransport_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "BAD_REQ", "message": "invalid parameter"})
	}))
	defer srv.Close()

	tp := NewClientTransport(srv.Client().Transport, TransportConfig{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	err2 := ResponseError(resp)
	if err2 == nil {
		t.Fatal("ResponseError returned nil for error response")
	}
	if err2.HTTPStatus != http.StatusBadRequest {
		t.Errorf("HTTPStatus = %d; want %d", err2.HTTPStatus, http.StatusBadRequest)
	}
	if err2.Code != "BAD_REQ" {
		t.Errorf("Code = %q; want %q", err2.Code, "BAD_REQ")
	}
	if !errors.Is(err2, ErrInvalidRequest) {
		t.Errorf("errors.Is(err, ErrInvalidRequest) = false; want true")
	}
}

func TestTransport_401_SessionExpired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-request-id", "test-req-id")
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	tp := NewClientTransport(srv.Client().Transport, TransportConfig{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	err2 := ResponseError(resp)
	if err2 == nil {
		t.Fatal("ResponseError returned nil for 401")
	}
	if !errors.Is(err2, ErrSessionExpired) {
		t.Errorf("errors.Is(err, ErrSessionExpired) = false; want true")
	}
	if err2.RequestID != "test-req-id" {
		t.Errorf("RequestID = %q; want %q", err2.RequestID, "test-req-id")
	}
}

func TestTransport_429_RateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	tp := NewClientTransport(srv.Client().Transport, TransportConfig{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	err2 := ResponseError(resp)
	if err2 == nil {
		t.Fatal("ResponseError returned nil for 429")
	}
	if !errors.Is(err2, ErrRateLimited) {
		t.Errorf("errors.Is(err, ErrRateLimited) = false; want true")
	}
}

func TestTransport_ErrorResponse_UnknownEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "internal server error plain text")
	}))
	defer srv.Close()

	tp := NewClientTransport(srv.Client().Transport, TransportConfig{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	err2 := ResponseError(resp)
	if err2 == nil {
		t.Fatal("ResponseError returned nil for non-JSON error")
	}
	if err2.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("HTTPStatus = %d; want %d", err2.HTTPStatus, http.StatusInternalServerError)
	}
}

func TestTransport_TimeoutAppliedWhenNoDeadline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tp := NewClientTransport(srv.Client().Transport, TransportConfig{Timeout: 20 * time.Millisecond})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	_, err := tp.RoundTrip(req)
	if err == nil {
		t.Fatal("RoundTrip: want timeout error")
	}
}

func TestTransport_TimeoutRespectsCallerDeadline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Middleware timeout is shorter than the caller's, but must not override it.
	tp := NewClientTransport(srv.Client().Transport, TransportConfig{Timeout: time.Nanosecond})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	resp, err := tp.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()
}

func TestChain(t *testing.T) {
	var order []string

	mw1 := func(base http.RoundTripper) http.RoundTripper {
		return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			order = append(order, "mw1-in")
			resp, err := base.RoundTrip(req)
			order = append(order, "mw1-out")
			return resp, err
		})
	}
	mw2 := func(base http.RoundTripper) http.RoundTripper {
		return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			order = append(order, "mw2-in")
			resp, err := base.RoundTrip(req)
			order = append(order, "mw2-out")
			return resp, err
		})
	}

	chained := Chain(http.DefaultTransport, mw1, mw2)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	_, _ = chained.RoundTrip(req)

	want := []string{"mw1-in", "mw2-in", "handler", "mw2-out", "mw1-out"}
	for i, w := range want {
		if i < len(order) && order[i] != w {
			t.Errorf("order[%d] = %q; want %q", i, order[i], w)
		}
	}
}
