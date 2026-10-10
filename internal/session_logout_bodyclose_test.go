// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

// Connection-reuse regression test for httpAPI.logout.
//
// WHAT THIS MEASURES
// A dropped response body is not observable from Go code directly — logout
// discards the *http.Response entirely, so there is nothing left on it to Close
// and nothing for a caller to forget. It is observable from the server side:
// net/http returns a connection to the keep-alive pool only once the response
// body has been read to EOF *and* closed, so a call that breaks either half
// forces the next call to dial again. This file counts accepted TCP
// connections across a sequence of repeated logout calls and requires the count
// to stay flat.
//
// The endpoint must answer with a NON-EMPTY body for the measurement to mean
// anything. net/http pools a zero-length-body response eagerly, before the
// caller ever touches Body, so a leak on such a response is invisible. The mock
// gateway fixture for /v1/api/logout is `{}` (internal/mockgateway/fixtures.go),
// so the test server answers with exactly that.
//
// TestHTTPAPI_Logout_ReuseConnections_DetectsLeak is the non-vacuity check: it
// repeats the same measurement against deliberately unclosed bodies and
// requires the counter to climb, so a green result above cannot be an artifact
// of a blind instrument. Its shape is the pre-fix logout.

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// logoutConnCounter counts TCP connections accepted by a test server, split by
// the http.ConnState transition that marks a brand-new connection. Counting
// StateNew therefore measures connection reuse directly.
type logoutConnCounter struct {
	mu    sync.Mutex
	conns int
}

func (c *logoutConnCounter) hook(_ net.Conn, state http.ConnState) {
	if state != http.StateNew {
		return
	}
	c.mu.Lock()
	c.conns++
	c.mu.Unlock()
}

func (c *logoutConnCounter) get() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conns
}

func (c *logoutConnCounter) reset() {
	c.mu.Lock()
	c.conns = 0
	c.mu.Unlock()
}

// logoutReuseServer starts an httptest server wired to a logoutConnCounter that
// answers /v1/api/logout with a small non-empty JSON body, matching the mock
// gateway fixture.
func logoutReuseServer(t *testing.T) (*httptest.Server, *logoutConnCounter) {
	t.Helper()
	counter := &logoutConnCounter{}
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/api/logout" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}))
	srv.Config.ConnState = counter.hook
	srv.Start()
	t.Cleanup(srv.Close)
	return srv, counter
}

// TestHTTPAPI_Logout_ReuseConnections asserts that httpAPI.logout returns its
// connection to the keep-alive pool, i.e. that it drains and closes the logout
// response body instead of dropping it.
func TestHTTPAPI_Logout_ReuseConnections(t *testing.T) {
	srv, counter := logoutReuseServer(t)

	client := &http.Client{}
	api := newHTTPAPI(client, srv.URL)

	// Warm up: the first call establishes the connection. Measuring from zero
	// would fold that setup cost into the result.
	if err := api.logout(context.Background()); err != nil {
		t.Fatalf("warmup logout: %v", err)
	}

	// Ordered so that the server and its idle connections are torn down before
	// the goroutine check runs. t.Cleanup would run after these defers, which
	// would leave the httptest server's Serve goroutine live during the check.
	defer settleGoroutines(t)
	defer client.CloseIdleConnections()
	defer srv.Close()

	const iterations = 8
	counter.reset()
	for i := 0; i < iterations; i++ {
		if err := api.logout(context.Background()); err != nil {
			t.Fatalf("logout (iteration %d): %v", i, err)
		}
	}

	got, want := counter.get(), iterations
	t.Logf("%d sequential logout calls reused %d new connection(s)", want, got)
	if got >= want {
		t.Errorf("server accepted %d new connections for %d sequential logout calls "+
			"(want strictly fewer, i.e. reuse): the logout response body is not "+
			"drained and closed, so the connection is dropped instead of returned "+
			"to the keep-alive pool",
			got, want)
	}
}

// TestHTTPAPI_Logout_ReuseConnections_DetectsLeak is the non-vacuity check for
// TestHTTPAPI_Logout_ReuseConnections. It runs the same measurement against a
// client that deliberately drops every response body, and requires the counter
// to climb. If this test ever stops failing, the instrument has gone blind and
// the test above would pass for the wrong reason. The body handling below is
// exactly the pre-fix logout: the response is discarded unread.
func TestHTTPAPI_Logout_ReuseConnections_DetectsLeak(t *testing.T) {
	srv, counter := logoutReuseServer(t)

	client := &http.Client{}
	leaked := make([]*http.Response, 0, 8)
	t.Cleanup(func() {
		// The bodies are deliberately still open: that is the behaviour under
		// test, and it is what makes the counter climb. Clean them up afterwards
		// so the package's goleak TestMain does not inherit this test's abandoned
		// connections. Cleanup runs after the assertion below, so the measurement
		// still sees every body unclosed.
		client.CloseIdleConnections()
		for _, resp := range leaked {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	})

	const iterations = 8
	for i := 0; i < iterations; i++ {
		// Same request httpAPI.logout builds, with the response discarded unread.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			srv.URL+"/v1/api/logout", nil)
		if err != nil {
			cancel()
			t.Fatalf("NewRequestWithContext: %v", err)
		}
		resp, err := client.Do(req)
		cancel()
		if err != nil {
			t.Fatalf("Do (iteration %d): %v", i, err)
		}
		leaked = append(leaked, resp)
	}

	if got := counter.get(); got < iterations {
		t.Errorf("server accepted %d new connections for %d unclosed-body logout calls; "+
			"expected the counter to detect the leak", got, iterations)
	}
}
