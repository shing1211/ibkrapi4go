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
	"strings"
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
	failSubmit   atomic.Bool
	failOrder    atomic.Bool
	submitReply  atomic.Bool
	submitCalls  atomic.Int64
	orderCalls   atomic.Int64

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
		p := r.URL.Path
		switch {
		// --- session ---
		case p == "/v1/api/iserver/auth/ssodh/init":
			fmt.Fprint(w, `{"authenticated":true,"established":true}`)
		case p == "/v1/api/iserver/auth/status":
			fmt.Fprint(w, `{"authenticated":true,"established":true,"connected":true}`)
		case p == "/v1/api/tickle":
			fmt.Fprint(w, `{"session":"tok-123"}`)
		case p == "/v1/api/logout":
			fmt.Fprint(w, `{}`)

		// --- account ---
		case p == "/v1/api/iserver/accounts":
			fmt.Fprint(w, `{"accounts":["U1234567","U7654321"],"aliases":{"U1234567":"Main"}}`)
		case p == "/v1/api/iserver/account/pnl/partitioned":
			fmt.Fprint(w, `{"upnl":{"U1234567.Core":{"dpl":"1.5","el":"2","mv":"3","nl":"4","rowType":"1","upl":"5"}}}`)
		case p == "/v1/api/iserver/account/U1234567/summary":
			if gw.unauthorized.Load() {
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprint(w, `{"error":"not authenticated"}`)
				return
			}
			fmt.Fprint(w, `{"accountType":"INDIVIDUAL","netLiquidationValue":"1234.5600","totalCashValue":"100.25","availableFunds":"900.10","SMA":"1200.0000","cashBalances":[{"currency":"USD","balance":"50.5","settledCash":"40.0"}]}`)

		// --- contracts ---
		case p == "/v1/api/iserver/secdef/search":
			fmt.Fprint(w, `[{"conid":265598,"symbol":"AAPL","companyName":"Apple Inc","secType":"STK","description":"Apple Inc","exchange":"NASDAQ"}]`)
		case p == "/v1/api/iserver/contract/265598/info":
			fmt.Fprint(w, `{"con_id":265598,"symbol":"AAPL","company_name":"Apple Inc","currency":"USD","exchange":"NASDAQ","instrument_type":"STK","local_symbol":"AAPL","multiplier":1,"expiry_full":"","cusip":"037833100","category":"Technology","industry":"Consumer Electronics"}`)
		case p == "/v1/api/iserver/contract/rules":
			fmt.Fprint(w, `{"algoEligible":true,"allOrNoneEligible":true,"canTradeAcctIds":["U1234567"],"defaultSize":100,"limitPrice":0.01,"orderTypes":["MKT","LMT"],"tifTypes":["DAY","GTC"],"TIF":"DAY","negativeCapable":false,"preview":true}`)
		case p == "/v1/api/iserver/secdef/strikes":
			fmt.Fprint(w, `{"call":[150,155],"put":[145,140]}`)

		// --- portfolio positions ---
		case p == "/v1/api/portfolio/accounts":
			fmt.Fprint(w, `[{"accountId":"U1234567","accountTitle":"Main","currency":"USD","accountAlias":"Main"},{"accountId":"U7654321","accountTitle":"Second","currency":"USD"}]`)
		case p == "/v1/api/portfolio/subaccounts":
			fmt.Fprint(w, `[{"accountId":"U1234567","accountTitle":"Main","currency":"USD"}]`)
		case p == "/v1/api/portfolio2/U1234567/positions":
			fmt.Fprint(w, `[{"acctId":"U1234567","conid":265598,"contractDesc":"AAPL","assetClass":"STK","currency":"USD","position":"10.5","avgCost":"145.25","mktPrice":"150.00","mktValue":"1575.00","unrealizedPnl":"49.875"}]`)
		case p == "/v1/api/portfolio/U1234567/positions/invalidate" && r.Method == http.MethodPost:
			fmt.Fprint(w, `{"status":"success"}`)
		case strings.HasPrefix(p, "/v1/api/portfolio/U1234567/positions/"):
			page := strings.TrimPrefix(p, "/v1/api/portfolio/U1234567/positions/")
			switch page {
			case "0":
				fmt.Fprint(w, `[{"acctId":"U1234567","conid":265598,"contractDesc":"AAPL","assetClass":"STK","currency":"USD","position":"10.5","avgCost":"145.25","mktPrice":"150.00","mktValue":"1575.00","unrealizedPnl":"49.875"},{"acctId":"U1234567","conid":8314,"contractDesc":"MSFT","assetClass":"STK","currency":"USD","position":"5","avgCost":"300.10","mktPrice":"310.00","mktValue":"1550.00","unrealizedPnl":"49.5"}]`)
			case "1":
				fmt.Fprint(w, `[{"acctId":"U1234567","conid":1,"contractDesc":"SPY","assetClass":"STK","currency":"USD","position":"2","avgCost":"400.00","mktPrice":"410.00","mktValue":"820.00","unrealizedPnl":"20"}]`)
			default:
				fmt.Fprint(w, `[]`)
			}
		case p == "/v1/api/portfolio/U1234567/position/265598":
			fmt.Fprint(w, `[{"acctId":"U1234567","conid":265598,"contractDesc":"AAPL","assetClass":"STK","currency":"USD","position":"10.5","avgCost":"145.25","mktPrice":"150.00","mktValue":"1575.00","unrealizedPnl":"49.875"}]`)
		case p == "/v1/api/portfolio/U1234567/ledger":
			fmt.Fprint(w, `{"USD":{"acctcode":"U1234567","currency":"USD","cashbalance":"100.25","netliquidationvalue":"1575.00","stockmarketvalue":"1575.00","unrealizedpnl":"49.875","realizedpnl":"0"}}`)
		case p == "/v1/api/portfolio/U1234567/allocation":
			fmt.Fprint(w, `{"assetClass":{"long":{"STK":1000.5},"short":{}},"sector":{"long":{"Technology":1000.5},"short":{}},"group":{"long":{},"short":{}}}`)
		case p == "/v1/api/portfolio/U1234567/summary":
			fmt.Fprint(w, `{"netliquidation":{"amount":1575.00,"currency":"USD"},"totalcashvalue":{"amount":100.25,"currency":"USD"}}`)
		case p == "/v1/api/portfolio/U1234567/meta":
			fmt.Fprint(w, `{"accountId":"U1234567","accountTitle":"Main","currency":"USD","accountAlias":"Main"}`)

		// --- market data ---
		case p == "/v1/api/iserver/marketdata/snapshot":
			fmt.Fprint(w, `[{"conid":265598,"31":"150.25","84":"150.20","_updated":1564652478,"server_id":"q0"}]`)
		case p == "/v1/api/iserver/marketdata/history":
			fmt.Fprint(w, `{"symbol":"AAPL","data":[{"t":1564652478,"o":"150.1","h":"151.2","l":"149.9","c":"150.9","v":"1000"}]}`)
		case p == "/v1/api/iserver/marketdata/unsubscribe" && r.Method == http.MethodPost:
			fmt.Fprint(w, `{"status":"success"}`)
		case p == "/v1/api/iserver/marketdata/unsubscribeall":
			fmt.Fprint(w, `{}`)

		// --- orders ---
		case p == "/v1/api/iserver/account/U1234567/orders" && r.Method == http.MethodPost:
			gw.submitCalls.Add(1)
			if gw.failSubmit.Load() {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, `{"error":"internal error"}`)
				return
			}
			if gw.submitReply.Load() {
				fmt.Fprint(w, `[{"id":"reply-1","message":["Confirm this order"],"messageIds":["o354"]}]`)
				return
			}
			fmt.Fprint(w, `[{"order_id":"999","order_status":"PreSubmitted"}]`)
		case strings.HasPrefix(p, "/v1/api/iserver/reply/"):
			fmt.Fprint(w, `[{"order_id":"999","order_status":"PreSubmitted"}]`)
		case p == "/v1/api/iserver/account/U1234567/orders/whatif":
			fmt.Fprint(w, `{"amount":{"initial":"1000.50","maintenance":"800.25"}}`)
		case p == "/v1/api/iserver/account/U1234567/order/999" && r.Method == http.MethodPost:
			gw.orderCalls.Add(1)
			if gw.failOrder.Load() {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, `{"error":"internal error"}`)
				return
			}
			fmt.Fprint(w, `[{"order_id":"999","order_status":"PreSubmitted"}]`)
		case p == "/v1/api/iserver/account/U1234567/order/999" && r.Method == http.MethodDelete:
			gw.orderCalls.Add(1)
			if gw.failOrder.Load() {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, `{"error":"internal error"}`)
				return
			}
			fmt.Fprint(w, `{"order_id":"999","msg":"Request was submitted"}`)
		case p == "/v1/api/iserver/account/orders":
			fmt.Fprint(w, `{"orders":[{"orderId":"999","account":"U1234567","conid":265598,"ticker":"AAPL","orderType":"LMT","side":"BUY","status":"PreSubmitted","timeInForce":"DAY","totalSize":"10","filledQuantity":"0","remainingQuantity":"10","price":"150.00","avgPrice":"0"}]}`)
		case strings.HasPrefix(p, "/v1/api/iserver/account/order/status/"):
			fmt.Fprint(w, `{"order_id":"999","order_status":"PreSubmitted","conid":265598,"side":"BUY","filled_quantity":"0","remaining_quantity":"10","average_price":"0"}`)
		case p == "/v1/api/iserver/account/trades":
			fmt.Fprint(w, `[{"order_id":"999","execution_id":"exec-1","account":"U1234567","conid":265598,"symbol":"AAPL","side":"B","size":"10","price":"150.00","commission":"1.00","net_amount":"1499.00","trade_time_r":"1564652478000"}]`)

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
