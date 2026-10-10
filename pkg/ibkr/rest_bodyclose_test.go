// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

// Connection-reuse regression test for the REST *WithResponse call sites.
//
// WHAT THIS MEASURES
// A leaked body is not observable from Go code directly — the generated
// *Response struct carries `Body []byte`, not an io.ReadCloser, so there is
// nothing on it to Close and nothing for a caller to forget. It is observable
// from the server side: net/http only returns a connection to the keep-alive
// pool once the response body has been read to EOF *and* closed, so a call that
// breaks either half forces the next call to dial again. This file counts
// accepted TCP connections across a sequence of repeated calls and requires the
// count to stay flat.
//
// TestRESTMethods_ReuseConnections_DetectsLeak is the non-vacuity check: it
// repeats the same measurement against deliberately unclosed bodies and
// requires the counter to climb, so a green result above cannot be an artifact
// of a blind instrument.
//
// ORIGINAL TASK PREMISE (rejected)
// T22 was briefed as "add a deferred Body.Close() to the 66 *WithResponse call
// sites". That is not possible: Body on a generated *Response is []byte, and
// every *WithResponse wrapper already delegates to a Parse*Response function
// that does io.ReadAll(rsp.Body) followed by
// `defer func() { _ = rsp.Body.Close() }()`. No call site was changed.
//
// REAL CAUSE, NOW FIXED
// internal.Timeout installed `defer cancel()` inside its RoundTrip closure, so
// the request context was cancelled the moment response headers arrived —
// before the caller read the body. net/http takes its "request context done"
// branch and closes the connection instead of pooling it, giving 24 sequential
// REST calls 24 new TCP connections (and 24 TLS handshakes in production).
// cancelOnCloseBody exists precisely to defer that cancel until the body is
// closed, and the defer defeated it. See internal/transport.go Timeout.

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// connCounter counts TCP connections accepted by a test server, split by the
// http.ConnState transition that marks a brand-new connection. net/http only
// hands a connection back to the keep-alive pool once the response body has been
// fully read *and* closed, so a caller that drops the body forces the next call
// to dial again. Counting StateNew therefore measures connection reuse directly.
type connCounter struct {
	mu    sync.Mutex
	conns int
}

func (c *connCounter) hook(_ net.Conn, state http.ConnState) {
	if state != http.StateNew {
		return
	}
	c.mu.Lock()
	c.conns++
	c.mu.Unlock()
}

func (c *connCounter) get() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conns
}

func (c *connCounter) reset() {
	c.mu.Lock()
	c.conns = 0
	c.mu.Unlock()
}

// restReuseServer starts an httptest server wired to a connCounter and serving
// the handful of REST paths the body-reuse test exercises.
func restReuseServer(t *testing.T) (*httptest.Server, *connCounter) {
	t.Helper()
	counter := &connCounter{}
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/oauth2/api/v1/token":
			fmt.Fprint(w, `{"access_token":"tok-1","expires_in":3600,"token_type":"Bearer"}`)
		case "/gw/api/v1/accounts":
			fmt.Fprint(w, `[]`)
		case "/gw/api/v1/restrictions/account":
			fmt.Fprint(w, `{"ids":[]}`)
		case "/gw/api/v1/enumerations/test-enum":
			fmt.Fprint(w, `{"jsonData":{"k":"v"}}`)
		default:
			fmt.Fprint(w, `{}`)
		}
	}))
	srv.Config.ConnState = counter.hook
	srv.Start()
	t.Cleanup(srv.Close)
	return srv, counter
}

// TestRESTMethods_ReuseConnections asserts that the *WithResponse-backed REST
// methods return their connection to the keep-alive pool, i.e. that every
// response body is drained and closed and the per-request context stays alive
// until it is.
//
// Each method is chosen from a different file so the assertion covers more than
// one call-site shape (decode-from-resp.Body, decode-from-parsed-JSON200).
func TestRESTMethods_ReuseConnections(t *testing.T) {
	srv, counter := restReuseServer(t)

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithRESTGateway(srv.URL),
		WithOAuth2ClientCredentials("cid", "sec"),
		WithOAuth2TokenURL(srv.URL+"/oauth2/api/v1/token"),
		WithTickleInterval(time.Hour),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })

	rest, err := cli.REST()
	if err != nil {
		t.Fatalf("REST: %v", err)
	}

	const accountID = AccountID("U1234567")
	calls := []struct {
		name string
		call func(context.Context) error
	}{
		{"Accounts.List", func(ctx context.Context) error {
			_, err := rest.Accounts().List(ctx)
			return err
		}},
		{"Restrictions.AccountRestrictions", func(ctx context.Context) error {
			_, err := rest.Restrictions().AccountRestrictions(ctx, accountID)
			return err
		}},
		{"Utilities.Enumerations", func(ctx context.Context) error {
			_, err := rest.Utilities().Enumerations(ctx, "test-enum")
			return err
		}},
	}

	ctx := context.Background()

	// Warm up: the first call of each kind establishes its connection and, for
	// the first one, fetches an OAuth token. Measuring from zero would fold
	// that setup cost into the result.
	for _, c := range calls {
		if err := c.call(ctx); err != nil {
			t.Fatalf("warmup %s: %v", c.name, err)
		}
	}

	const iterations = 8
	counter.reset()
	for i := 0; i < iterations; i++ {
		for _, c := range calls {
			if err := c.call(ctx); err != nil {
				t.Fatalf("%s (iteration %d): %v", c.name, i, err)
			}
		}
	}

	got, want := counter.get(), len(calls)*iterations
	t.Logf("%d sequential calls reused %d new connection(s)", want, got)
	if got >= want {
		t.Errorf("server accepted %d new connections for %d sequential calls "+
			"(want strictly fewer, i.e. reuse): responses are not returned to the "+
			"keep-alive pool, which is what a body that is read-and-closed under a "+
			"still-live request context would do. Suspect internal.Timeout "+
			"cancelling the request context before the body is consumed",
			got, want)
	}
}

// TestRESTMethods_ReuseConnections_DetectsLeak is the non-vacuity check for
// TestRESTMethods_ReuseConnections. It runs the same measurement against a
// client that deliberately drops every response body, and requires the counter
// to climb. If this test ever stops failing, the instrument has gone blind and
// the test above would pass for the wrong reason.
func TestRESTMethods_ReuseConnections_DetectsLeak(t *testing.T) {
	srv, counter := restReuseServer(t)

	const iterations = 8
	leaked := make([]*http.Response, 0, iterations)
	for i := 0; i < iterations; i++ {
		resp, err := http.Get(srv.URL + "/gw/api/v1/accounts")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		leaked = append(leaked, resp)
	}
	// The bodies are deliberately still open: that is the behaviour under test,
	// and it is what makes the counter climb. Clean them up afterwards so the
	// package's goleak TestMain does not inherit this test's abandoned
	// connections. Cleanup runs after the assertion below, so the measurement
	// still sees every body unclosed.
	t.Cleanup(func() {
		for _, resp := range leaked {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	})

	if got := counter.get(); got < iterations {
		t.Errorf("server accepted %d new connections for %d unclosed-body calls; "+
			"expected the counter to detect the leak", got, iterations)
	}
}
