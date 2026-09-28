// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaxBytes_DefaultCapIsApplied(t *testing.T) {
	// A default-configured transport had no cap at all: the middleware was only
	// added when MaxResponseBytes > 0, which left defaultMaxResponseBytes
	// unreferenced. Serving a small body cannot detect that, so this serves one
	// byte past the default cap and requires the typed error - the body is
	// streamed rather than buffered so the test does not allocate 32MB.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.CopyN(w, zeroReader{}, defaultMaxResponseBytes+1)
	}))
	defer srv.Close()

	rt := NewClientTransport(srv.Client().Transport, TransportConfig{})
	resp, err := rt.RoundTrip(mustReq(t, srv.URL))
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	if _, err := io.ReadAll(resp.Body); !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("read err = %v; want ErrResponseTooLarge - the default cap of %d bytes must be applied without opt-in",
			err, int64(defaultMaxResponseBytes))
	}
}

// zeroReader yields an unbounded stream of NUL bytes without allocating.
type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

// TestMaxBytes_UpgradeResponseIsNotWrapped is a regression test. Applying the cap
// unconditionally - which is what wiring up defaultMaxResponseBytes did - wraps
// the body of a 101 Switching Protocols response, and the WebSocket library then
// refuses to dial with "response body is not a io.ReadWriteCloser". Every
// WebSocket connection in the SDK broke. The upgrade body must pass through
// untouched.
func TestMaxBytes_UpgradeResponseIsNotWrapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Error("test server does not support hijacking")
			return
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Errorf("Hijack: %v", err)
			return
		}
		defer conn.Close()
		_, _ = io.WriteString(conn, "HTTP/1.1 101 Switching Protocols\r\n"+
			"Upgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
	}))
	defer srv.Close()

	rt := NewClientTransport(srv.Client().Transport, TransportConfig{})
	resp, err := rt.RoundTrip(mustReq(t, srv.URL))
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d; want 101", resp.StatusCode)
	}
	// The body must still satisfy the io.ReadWriteCloser the WebSocket library
	// requires of a hijacked connection.
	if _, ok := resp.Body.(io.ReadWriteCloser); !ok {
		t.Fatalf("upgrade body is %T; want an io.ReadWriteCloser so the WebSocket dial can take over the connection",
			resp.Body)
	}
}

func TestMaxBytes_ExceedingCapReturnsTypedError(t *testing.T) {
	const limit = 64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, strings.Repeat("a", 4096))
	}))
	defer srv.Close()

	rt := NewClientTransport(srv.Client().Transport, TransportConfig{MaxResponseBytes: limit})
	resp, err := rt.RoundTrip(mustReq(t, srv.URL))
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	// Truncation used to be silent here, so the caller received exactly `limit`
	// bytes and decoded them as a short body.
	if _, err := io.ReadAll(resp.Body); !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("read err = %v; want ErrResponseTooLarge", err)
	}
}

func TestMaxBytes_ExactFitIsNotAnError(t *testing.T) {
	// A body exactly at the cap must succeed: the reader probes for one extra
	// byte, and must not mistake EOF for overflow.
	const limit = 32
	body := strings.Repeat("a", limit)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, body)
	}))
	defer srv.Close()

	rt := NewClientTransport(srv.Client().Transport, TransportConfig{MaxResponseBytes: limit})
	resp, err := rt.RoundTrip(mustReq(t, srv.URL))
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("exact fit must not error: %v", err)
	}
	if string(got) != body {
		t.Errorf("body = %q (%d bytes); want %q (%d bytes)", got, len(got), body, len(body))
	}
}

func mustReq(t *testing.T, url string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	return req
}
