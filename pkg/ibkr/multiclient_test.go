// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMultiClient_NewMultiClient(t *testing.T) {
	_, err := NewMultiClient(nil)
	if err == nil {
		t.Fatal("NewMultiClient(nil) = nil; want error")
	}
	if _, ok := err.(*ConfigError); !ok {
		t.Fatalf("err type = %T; want *ConfigError", err)
	}

	_, err = NewMultiClient([]*Client{})
	if err == nil {
		t.Fatal("NewMultiClient([]) = nil; want error")
	}
}

func TestMultiClient_Add(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/api/iserver/accounts":
			auth := r.Header.Get("Authorization")
			if auth == "Bearer token1" {
				w.Write([]byte(`{"accounts":["A"],"aliases":{"A":"Account A"}}`))
			} else {
				w.Write([]byte(`{"accounts":["B"],"aliases":{"B":"Account B"}}`))
			}
		case "/v1/api/portfolio2/A/positions":
			w.Write([]byte(`[{"conId":8314,"contractDesc":"AAPL","quantity":"100","avgCost":"150.00","mktValue":"16000","unrealizedPnL":"1000","realizedPnL":"0"}]`))
		case "/v1/api/portfolio2/B/positions":
			w.Write([]byte(`[{"conId":4154,"contractDesc":"MSFT","quantity":"50","avgCost":"300.00","mktValue":"16000","unrealizedPnL":"500","realizedPnL":"0"}]`))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	makeClient := func(token string) *Client {
		cli, err := NewClient(
			WithGatewayURL(srv.URL),
			WithTickleInterval(time.Hour),
		)
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		return cli
	}

	mc, err := NewMultiClient([]*Client{makeClient("token1")})
	if err != nil {
		t.Fatalf("NewMultiClient: %v", err)
	}
	mc.Add(makeClient("token2"))

	clients := mc.Clients()
	if len(clients) != 2 {
		t.Fatalf("len(clients) = %d; want 2", len(clients))
	}
}

func TestMultiClient_Accounts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/api/iserver/accounts" {
			w.Write([]byte(`{"accounts":["A","B"],"aliases":{"A":"Account A","B":"Account B"}}`))
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
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

	mc, err := NewMultiClient([]*Client{cli})
	if err != nil {
		t.Fatalf("NewMultiClient: %v", err)
	}

	accounts, err := mc.Accounts(context.Background())
	if err != nil {
		t.Fatalf("Accounts: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("len(accounts) = %d; want 2", len(accounts))
	}
	if accounts[0].ID != "A" || accounts[1].ID != "B" {
		t.Fatalf("accounts = %v; want [A, B]", accounts)
	}
}

func TestMultiClient_Accounts_MultiClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/api/iserver/accounts" {
			w.Write([]byte(`{"accounts":["A1"],"aliases":{"A1":"Acct A1"}}`))
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	cli1, _ := NewClient(WithGatewayURL(srv.URL), WithTickleInterval(time.Hour))
	cli2, _ := NewClient(WithGatewayURL(srv.URL), WithTickleInterval(time.Hour))
	defer func() { cli1.Close(); cli2.Close() }()

	mc, _ := NewMultiClient([]*Client{cli1, cli2})

	accounts, err := mc.Accounts(context.Background())
	if err != nil {
		t.Fatalf("Accounts: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("len(accounts) = %d; want 2 (A1 from each of 2 clients)", len(accounts))
	}
}

func TestMultiClient_Positions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/api/iserver/accounts":
			w.Write([]byte(`{"accounts":["A1"],"aliases":{"A1":"Acct A1"}}`))
		case "/v1/api/portfolio2/A1/positions":
			w.Write([]byte(`[{"conId":8314,"contractDesc":"AAPL","quantity":"100","avgCost":"150.00","mktValue":"16000","unrealizedPnL":"1000","realizedPnL":"0"}]`))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cli, _ := NewClient(WithGatewayURL(srv.URL), WithTickleInterval(time.Hour))
	defer cli.Close()

	mc, _ := NewMultiClient([]*Client{cli})

	positions, err := mc.Positions(context.Background())
	if err != nil {
		t.Fatalf("Positions: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("len(positions) = %d; want 1", len(positions))
	}
	if positions[0].ConID != 8314 {
		t.Fatalf("positions[0].ConID = %d; want 8314", positions[0].ConID)
	}
}

func TestMultiClient_Close(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/api/iserver/accounts" {
			w.Write([]byte(`{"accounts":["A"],"aliases":{"A":"Acct A"}}`))
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	cli, _ := NewClient(WithGatewayURL(srv.URL), WithTickleInterval(time.Hour))
	mc, _ := NewMultiClient([]*Client{cli})

	err := mc.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
}
