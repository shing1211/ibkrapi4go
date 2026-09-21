// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFaultInjection_SDK_GatewayTimeout504(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGatewayTimeout)
		json.NewEncoder(w).Encode(map[string]string{"error": "timeout", "message": "gateway timeout"})
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithTickleInterval(time.Hour),
		WithRetryPolicy(RetryPolicy{MaxAttempts: 1}),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	if err := cli.Session().Initialize(context.Background()); err == nil {
		t.Fatal("Session.Initialize: want error for 504")
	}
}

func TestFaultInjection_SDK_RateLimit429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": "rate_limited", "message": "too many requests"})
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithTickleInterval(time.Hour),
		WithRetryPolicy(RetryPolicy{MaxAttempts: 1}),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	_, err = cli.Account().List(context.Background())
	if err == nil {
		t.Fatal("Account.List: want error for 429")
	}
	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("err = %T; want *Error", err)
	}
	if e.HTTPStatus != http.StatusTooManyRequests {
		t.Errorf("HTTPStatus = %d; want %d", e.HTTPStatus, http.StatusTooManyRequests)
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("errors.Is(err, ErrRateLimited) = false; want true")
	}
}

func TestFaultInjection_SDK_AuthFailure401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized", "message": "not authenticated"})
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithTickleInterval(time.Hour),
		WithRetryPolicy(RetryPolicy{MaxAttempts: 1}),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	_, err = cli.Account().List(context.Background())
	if err == nil {
		t.Fatal("Account.List: want error for 401")
	}
	if !errors.Is(err, ErrSessionExpired) {
		t.Errorf("errors.Is(err, ErrSessionExpired) = false; want true")
	}
	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("err = %T; want *Error", err)
	}
	if e.HTTPStatus != http.StatusUnauthorized {
		t.Errorf("HTTPStatus = %d; want %d", e.HTTPStatus, http.StatusUnauthorized)
	}
}

func TestFaultInjection_SDK_ServerError500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "INTERNAL", "message": "server error"})
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithTickleInterval(time.Hour),
		WithRetryPolicy(RetryPolicy{MaxAttempts: 1}),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	_, err = cli.Account().List(context.Background())
	if err == nil {
		t.Fatal("Account.List: want error for 500")
	}
	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("err = %T; want *Error", err)
	}
	if e.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("HTTPStatus = %d; want %d", e.HTTPStatus, http.StatusInternalServerError)
	}
}

func TestFaultInjection_SDK_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `not valid json{{{`)
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithTickleInterval(time.Hour),
		WithRetryPolicy(RetryPolicy{MaxAttempts: 1}),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	_, err = cli.Account().List(context.Background())
	if err == nil {
		t.Fatal("Account.List: want decode error for malformed JSON")
	}
	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("err = %T; want *Error", err)
	}
	if !strings.Contains(e.Message, "decode response") {
		t.Errorf("Message = %q; want contains 'decode response'", e.Message)
	}
}

func TestFaultInjection_SDK_ConnectionRefused(t *testing.T) {
	cli, err := NewClient(
		WithGatewayURL("http://127.0.0.1:1"),
		WithTickleInterval(time.Hour),
		WithRetryPolicy(RetryPolicy{MaxAttempts: 1}),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	_, err = cli.Account().List(context.Background())
	if err == nil {
		t.Fatal("Account.List: want connection error")
	}
}

func TestFaultInjection_SDK_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithTickleInterval(time.Hour),
		WithRetryPolicy(RetryPolicy{MaxAttempts: 1}),
		WithRequestTimeout(0),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err = cli.Account().List(ctx)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("Account.List: want context error")
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("elapsed = %v; want prompt cancellation", elapsed)
	}
}

func TestFaultInjection_SDK_RetryThenSuccess(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"error": "UNAVAILABLE", "message": "temporarily unavailable"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"accounts":["U1234567"],"aliases":{"U1234567":"Main"}}`)
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithTickleInterval(time.Hour),
		WithRetryPolicy(RetryPolicy{
			MaxAttempts:   3,
			BaseDelay:     time.Millisecond,
			MaxDelay:      5 * time.Millisecond,
			Jitter:        false,
			RetryOnStatus: []int{503},
		}),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	accounts, err := cli.Account().List(context.Background())
	if err != nil {
		t.Fatalf("Account.List: %v", err)
	}
	if len(accounts) != 1 || accounts[0].ID != "U1234567" {
		t.Errorf("accounts = %+v; want [U1234567]", accounts)
	}
	if calls != 3 {
		t.Errorf("calls = %d; want 3 (2 retries + success)", calls)
	}
}

func TestFaultInjection_SDK_RetryExhausted(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "SERVER_ERROR", "message": "persistent failure"})
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithTickleInterval(time.Hour),
		WithRetryPolicy(RetryPolicy{
			MaxAttempts:   3,
			BaseDelay:     time.Millisecond,
			MaxDelay:      5 * time.Millisecond,
			Jitter:        false,
			RetryOnStatus: []int{500},
		}),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	_, err = cli.Account().List(context.Background())
	if err == nil {
		t.Fatal("Account.List: want error after retry exhaustion")
	}
	if calls != 3 {
		t.Errorf("calls = %d; want 3 (all retries exhausted)", calls)
	}
	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("err = %T; want *Error", err)
	}
	if e.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("HTTPStatus = %d; want %d", e.HTTPStatus, http.StatusInternalServerError)
	}
}

func TestFaultInjection_SDK_403NotAuthenticated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithTickleInterval(time.Hour),
		WithRetryPolicy(RetryPolicy{MaxAttempts: 1}),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	_, err = cli.Account().List(context.Background())
	if err == nil {
		t.Fatal("Account.List: want error for 403")
	}
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("errors.Is(err, ErrNotAuthenticated) = false; want true")
	}
}

func TestFaultInjection_SDK_404NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithTickleInterval(time.Hour),
		WithRetryPolicy(RetryPolicy{MaxAttempts: 1}),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	_, err = cli.Account().List(context.Background())
	if err == nil {
		t.Fatal("Account.List: want error for 404")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("errors.Is(err, ErrNotFound) = false; want true")
	}
}

func TestFaultInjection_SDK_101NotError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusSwitchingProtocols)
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithTickleInterval(time.Hour),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/v1/api/ws", nil)
	resp, err := cli.HTTPClient().Transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Errorf("StatusCode = %d; want 101", resp.StatusCode)
	}
}

func TestFaultInjection_SDK_SessionStatusWith504(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithTickleInterval(time.Hour),
		WithRetryPolicy(RetryPolicy{MaxAttempts: 1}),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	_, err = cli.Session().Status(context.Background())
	if err == nil {
		t.Fatal("Session.Status: want error for 504")
	}
}

func TestFaultInjection_SDK_AccountSummaryWith401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithTickleInterval(time.Hour),
		WithRetryPolicy(RetryPolicy{MaxAttempts: 1}),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	_, err = cli.Account().Summary(context.Background(), "U1234567")
	if err == nil {
		t.Fatal("Account.Summary: want error for 401")
	}
	if !errors.Is(err, ErrSessionExpired) {
		t.Errorf("errors.Is(err, ErrSessionExpired) = false; want true")
	}
}

func TestFaultInjection_SDK_ContractSearchWith500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "INTERNAL", "message": "search failed"})
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithTickleInterval(time.Hour),
		WithRetryPolicy(RetryPolicy{MaxAttempts: 1}),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	_, err = cli.Trade().SearchContracts(context.Background(), "AAPL")
	if err == nil {
		t.Fatal("SearchContracts: want error for 500")
	}
	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("err = %T; want *Error", err)
	}
	if e.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("HTTPStatus = %d; want %d", e.HTTPStatus, http.StatusInternalServerError)
	}
}
