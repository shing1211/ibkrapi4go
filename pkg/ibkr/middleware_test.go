// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// recordingTransport records requests that pass through it.
type recordingTransport struct {
	base   http.RoundTripper
	called []string
	mu     sync.Mutex
}

func (t *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.called = append(t.called, req.Header.Get("X-Trace"))
	t.mu.Unlock()
	return t.base.RoundTrip(req)
}

func (t *recordingTransport) traceValues() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]string, len(t.called))
	copy(out, t.called)
	return out
}

// tagMiddleware adds a header value to each request.
func tagMiddleware(tag string) func(http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		return &tagTransport{next: next, tag: tag}
	}
}

type tagTransport struct {
	next http.RoundTripper
	tag  string
}

func (t *tagTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("X-Trace", t.tag)
	return t.next.RoundTrip(req)
}

func TestMiddleware_AppliedInOrder(t *testing.T) {
	// Server records what arrives.
	var mu sync.Mutex
	var traces []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		traces = append(traces, r.Header.Get("X-Trace"))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithTickleInterval(time.Hour),
		WithTransportMiddleware(tagMiddleware("first")),
		WithTransportMiddleware(tagMiddleware("second")),
		WithTransportMiddleware(tagMiddleware("third")),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/v1/api/test", nil)
	resp, err := cli.HTTPClient().Transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()

	mu.Lock()
	defer mu.Unlock()
	// Middleware is applied left-to-right in the Chain call. WithTransportMiddleware
	// appends, so the slice is [first, second, third]. Chain applies ms[0] outermost,
	// so first executes first on the way in, writing "first" last — the innermost
	// wins. Order: first → second → third (each overwrites), so "third" arrives.
	if len(traces) != 1 || traces[0] != "third" {
		t.Errorf("trace = %v; want [third]", traces)
	}
}

func TestMiddleware_NoMiddleware_Unchanged(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v := r.Header.Get("X-Trace"); v != "" {
			t.Errorf("unexpected X-Trace = %q", v)
		}
		w.WriteHeader(http.StatusOK)
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

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/v1/api/test", nil)
	resp, err := cli.HTTPClient().Transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()
}

func TestMiddleware_ConcurrentSafety(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithTickleInterval(time.Hour),
		WithTransportMiddleware(tagMiddleware("safe")),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/v1/api/test", nil)
			resp, err := cli.HTTPClient().Transport.RoundTrip(req)
			if err != nil {
				t.Errorf("RoundTrip: %v", err)
				return
			}
			resp.Body.Close()
		}()
	}
	wg.Wait()
}
