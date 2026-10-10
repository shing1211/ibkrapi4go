// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

func poolGateway() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/api/logout", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{}`)
	})
	mux.HandleFunc("/v1/api/iserver/auth/ssodh/init", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{}`)
	})
	mux.HandleFunc("/v1/api/iserver/auth/status", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"authenticated":true,"established":true,"connected":true}`)
	})
	mux.HandleFunc("/v1/api/tickle", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"session":"test-session-token"}`)
	})
	mux.HandleFunc("/v1/api/iserver/accounts", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"accounts":["U1234567"]}`)
	})
	return httptest.NewServer(mux)
}

func TestTransportPool_BasicCreation(t *testing.T) {
	srv := poolGateway()
	defer srv.Close()

	pool, err := ibkr.NewTransportPool(
		ibkr.WithGatewayURL(srv.URL),
		ibkr.WithRateLimit(10, 20),
	)
	if err != nil {
		t.Fatalf("NewTransportPool: %v", err)
	}
	defer pool.Close()

	client, err := pool.Client()
	if err != nil {
		t.Fatalf("pool.Client: %v", err)
	}
	defer client.Close()

	if client.GatewayURL() != srv.URL {
		t.Errorf("GatewayURL = %q; want %q", client.GatewayURL(), srv.URL)
	}
}

func TestTransportPool_MultipleClientsShareTransport(t *testing.T) {
	srv := poolGateway()
	defer srv.Close()

	pool, err := ibkr.NewTransportPool(
		ibkr.WithGatewayURL(srv.URL),
	)
	if err != nil {
		t.Fatalf("NewTransportPool: %v", err)
	}
	defer pool.Close()

	c1, err := pool.Client()
	if err != nil {
		t.Fatalf("pool.Client 1: %v", err)
	}
	c2, err := pool.Client()
	if err != nil {
		t.Fatalf("pool.Client 2: %v", err)
	}

	// Both clients must share the same *http.Client.
	if c1.HTTPClient() != c2.HTTPClient() {
		t.Error("clients do not share the same HTTPClient")
	}

	c1.Close()
	c2.Close()
}

func TestTransportPool_CloseReleasesSharedResources(t *testing.T) {
	var logoutCalled atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/api/logout", func(w http.ResponseWriter, r *http.Request) {
		logoutCalled.Add(1)
		fmt.Fprint(w, `{}`)
	})
	mux.HandleFunc("/v1/api/iserver/auth/ssodh/init", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{}`)
	})
	mux.HandleFunc("/v1/api/iserver/auth/status", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"authenticated":true,"established":true,"connected":true}`)
	})
	mux.HandleFunc("/v1/api/tickle", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"session":"tok"}`)
	})
	customSrv := httptest.NewServer(mux)
	defer customSrv.Close()

	pool, err := ibkr.NewTransportPool(
		ibkr.WithGatewayURL(customSrv.URL),
	)
	if err != nil {
		t.Fatalf("NewTransportPool: %v", err)
	}

	c1, err := pool.Client()
	if err != nil {
		t.Fatalf("pool.Client 1: %v", err)
	}
	c2, err := pool.Client()
	if err != nil {
		t.Fatalf("pool.Client 2: %v", err)
	}

	// Initialize the session so it's authenticated and the tickle loop runs.
	if err := c1.Session().Initialize(t.Context()); err != nil {
		t.Fatalf("Session.Initialize: %v", err)
	}

	// Close pool first — resources stay alive because clients are open.
	pool.Close()

	// Closing the first client decrements refcount but doesn't tear down.
	c1.Close()

	// Closing the last client tears down the session (triggers logout).
	c2.Close()

	// Give a moment for async logout.
	if n := logoutCalled.Load(); n < 1 {
		t.Errorf("logout called %d times; want >= 1", n)
	}
}

func TestTransportPool_ClosePoolFirstThenClients(t *testing.T) {
	srv := poolGateway()
	defer srv.Close()

	pool, err := ibkr.NewTransportPool(
		ibkr.WithGatewayURL(srv.URL),
	)
	if err != nil {
		t.Fatalf("NewTransportPool: %v", err)
	}

	c1, _ := pool.Client()
	c2, _ := pool.Client()

	// Close pool first — clients still work (resources alive).
	pool.Close()
	c1.Close()
	c2.Close()
}

func TestTransportPool_ClosedPoolRejectsClients(t *testing.T) {
	srv := poolGateway()
	defer srv.Close()

	pool, err := ibkr.NewTransportPool(
		ibkr.WithGatewayURL(srv.URL),
	)
	if err != nil {
		t.Fatalf("NewTransportPool: %v", err)
	}
	pool.Close()

	_, err = pool.Client()
	if err == nil {
		t.Fatal("pool.Client on closed pool = nil; want error")
	}
}

func TestTransportPool_IdempotentClose(t *testing.T) {
	srv := poolGateway()
	defer srv.Close()

	pool, err := ibkr.NewTransportPool(
		ibkr.WithGatewayURL(srv.URL),
	)
	if err != nil {
		t.Fatalf("NewTransportPool: %v", err)
	}

	if err := pool.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := pool.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestTransportPool_PoolClientCloseIsIdempotent(t *testing.T) {
	srv := poolGateway()
	defer srv.Close()

	pool, err := ibkr.NewTransportPool(
		ibkr.WithGatewayURL(srv.URL),
	)
	if err != nil {
		t.Fatalf("NewTransportPool: %v", err)
	}
	defer pool.Close()

	c, err := pool.Client()
	if err != nil {
		t.Fatalf("pool.Client: %v", err)
	}

	if err := c.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestTransportPool_WithPerClientOptions(t *testing.T) {
	srv := poolGateway()
	defer srv.Close()

	pool, err := ibkr.NewTransportPool(
		ibkr.WithGatewayURL(srv.URL),
	)
	if err != nil {
		t.Fatalf("NewTransportPool: %v", err)
	}
	defer pool.Close()

	c1, err := pool.Client(ibkr.WithUserAgent("client-1"))
	if err != nil {
		t.Fatalf("pool.Client 1: %v", err)
	}
	c2, err := pool.Client(ibkr.WithUserAgent("client-2"))
	if err != nil {
		t.Fatalf("pool.Client 2: %v", err)
	}

	// Both share the same HTTP client transport (from the pool).
	if c1.HTTPClient() != c2.HTTPClient() {
		t.Error("clients do not share the same HTTPClient")
	}

	c1.Close()
	c2.Close()
}

func TestTransportPool_ConcurrentClients(t *testing.T) {
	srv := poolGateway()
	defer srv.Close()

	pool, err := ibkr.NewTransportPool(
		ibkr.WithGatewayURL(srv.URL),
	)
	if err != nil {
		t.Fatalf("NewTransportPool: %v", err)
	}
	defer pool.Close()

	const N = 20
	var wg sync.WaitGroup
	clients := make([]*ibkr.Client, N)

	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			c, err := pool.Client()
			if err != nil {
				t.Errorf("pool.Client %d: %v", idx, err)
				return
			}
			clients[idx] = c
		}(i)
	}
	wg.Wait()

	// All clients share the same HTTP client.
	for i := 1; i < N; i++ {
		if clients[i] != nil && clients[0] != nil {
			if clients[i].HTTPClient() != clients[0].HTTPClient() {
				t.Errorf("client %d does not share HTTPClient with client 0", i)
			}
		}
	}

	// Close all concurrently.
	var wgClose sync.WaitGroup
	for i := 0; i < N; i++ {
		if clients[i] != nil {
			wgClose.Add(1)
			go func(c *ibkr.Client) {
				defer wgClose.Done()
				c.Close()
			}(clients[i])
		}
	}
	wgClose.Wait()
}

func TestTransportPool_BadOptionReturnsError(t *testing.T) {
	_, err := ibkr.NewTransportPool(
		ibkr.WithGatewayURL(""),
	)
	if err == nil {
		t.Fatal("NewTransportPool with empty URL = nil; want error")
	}
}

func TestNewClient_StillWorksWithoutPool(t *testing.T) {
	srv := poolGateway()
	defer srv.Close()

	c, err := ibkr.NewClient(
		ibkr.WithGatewayURL(srv.URL),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	if c.GatewayURL() != srv.URL {
		t.Errorf("GatewayURL = %q; want %q", c.GatewayURL(), srv.URL)
	}
}
