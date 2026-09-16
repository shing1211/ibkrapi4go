// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	code := m.Run()
	time.Sleep(50 * time.Millisecond)
	if code == 0 {
		if err := goleak.Find(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	os.Exit(code)
}

type gateway struct {
	*httptest.Server
	unauthorized atomic.Bool

	mu      sync.Mutex
	headers http.Header
}

func newGateway(t *testing.T) *gateway {
	t.Helper()
	gw := &gateway{}
	gw.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gw.mu.Lock()
		gw.headers = r.Header.Clone()
		gw.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v1/api/iserver/auth/ssodh/init":
			fmt.Fprint(w, `{"authenticated":true,"established":true}`)
		case r.URL.Path == "/v1/api/iserver/auth/status":
			fmt.Fprint(w, `{"authenticated":true,"established":true,"connected":true}`)
		case r.URL.Path == "/v1/api/tickle":
			fmt.Fprint(w, `{"session":"tok-123"}`)
		case r.URL.Path == "/v1/api/iserver/accounts":
			fmt.Fprint(w, `{"accounts":["U1234567","U7654321"],"aliases":{"U1234567":"Main"}}`)
		case r.URL.Path == "/v1/api/iserver/account/pnl/partitioned":
			fmt.Fprint(w, `{"upnl":{"U1234567.Core":{"dpl":"1.5","el":"2","mv":"3","nl":"4","rowType":"1","upl":"5"}}}`)
		case r.URL.Path == "/v1/api/iserver/account/U1234567/summary":
			if gw.unauthorized.Load() {
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprint(w, `{"error":"not authenticated"}`)
				return
			}
			fmt.Fprint(w, `{"accountType":"INDIVIDUAL","netLiquidationValue":"1234.5600","totalCashValue":"100.25","availableFunds":"900.10","SMA":"1200.0000","cashBalances":[{"currency":"USD","balance":"50.5","settledCash":"40.0"}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"error":"unknown path"}`)
		}
	}))
	t.Cleanup(gw.Close)
	return gw
}

func (g *gateway) lastHeaders() http.Header {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.headers
}

func TestEndToEnd_SessionAndAccount(t *testing.T) {
	gw := newGateway(t)
	cli, err := NewClient(WithGatewayURL(gw.URL), WithTickleInterval(time.Hour))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	ctx := context.Background()
	if err := cli.Session().Initialize(ctx); err != nil {
		t.Fatalf("Session.Initialize: %v", err)
	}
	if got := cli.Session().State(); got != StateAuthenticated {
		t.Fatalf("state = %v; want AUTHENTICATED", got)
	}

	status, err := cli.Session().Status(ctx)
	if err != nil {
		t.Fatalf("Session.Status: %v", err)
	}
	if !status.Authenticated || !status.Established {
		t.Errorf("status = %+v; want authenticated+established", status)
	}

	accounts, err := cli.Account().List(ctx)
	if err != nil {
		t.Fatalf("Account.List: %v", err)
	}
	if len(accounts) != 2 || accounts[0].ID != "U1234567" || accounts[0].Alias != "Main" {
		t.Errorf("accounts = %+v; want 2 with U1234567 alias Main", accounts)
	}

	if h := gw.lastHeaders(); h.Get("X-request-id") == "" {
		t.Error("X-request-id header not set on outbound request")
	}
	if h := gw.lastHeaders(); h.Get("User-Agent") == "" {
		t.Error("User-Agent header not set on outbound request")
	}

	sum, err := cli.Account().Summary(ctx, "U1234567")
	if err != nil {
		t.Fatalf("Account.Summary: %v", err)
	}
	if sum.NetLiquidationValue != "1234.5600" {
		t.Errorf("NetLiquidationValue = %q; want %q (precision lost)", sum.NetLiquidationValue, "1234.5600")
	}
	if sum.TotalCashValue != "100.25" {
		t.Errorf("TotalCashValue = %q; want 100.25", sum.TotalCashValue)
	}
	if len(sum.CashBalances) != 1 || sum.CashBalances[0].SettledCash != "40.0" {
		t.Errorf("CashBalances = %+v; want settledCash 40.0", sum.CashBalances)
	}

	rows, err := cli.Account().PnL(ctx)
	if err != nil {
		t.Fatalf("Account.PnL: %v", err)
	}
	if len(rows) != 1 || rows[0].AccountID != "U1234567" || rows[0].Nl != "4" {
		t.Errorf("rows = %+v; want U1234567 nl=4", rows)
	}

	// Closing stops the tickle goroutine; goleak verifies no leaks at exit.
	if err := cli.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := cli.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if _, err := cli.Account().List(ctx); !errors.Is(err, ErrClosed) {
		t.Errorf("List after Close = %v; want ErrClosed", err)
	}
}

func TestEndToEnd_UnauthorizedMapsSentinel(t *testing.T) {
	gw := newGateway(t)
	gw.unauthorized.Store(true)
	cli, err := NewClient(WithGatewayURL(gw.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	_, err = cli.Account().Summary(context.Background(), "U1234567")
	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("err = %v; want ErrSessionExpired", err)
	}
	var e *Error
	if !errors.As(err, &e) || e.HTTPStatus != http.StatusUnauthorized {
		t.Errorf("err = %+v; want *Error with HTTPStatus 401", e)
	}
}

func TestNewClient_BadGatewayURL(t *testing.T) {
	_, err := NewClient(WithGatewayURL("not a url"))
	var ce *ConfigError
	if !errors.As(err, &ce) {
		t.Fatalf("err = %v; want *ConfigError", err)
	}
}
