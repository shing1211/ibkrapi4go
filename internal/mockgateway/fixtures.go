// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package mockgateway

import (
	"net/http"
	"sync"
)

// Operation IDs are the canonical opIds from docs/SPEC.md.
const (
	OpInitializeSession          = "initializeSession"
	OpGetBrokerageStatus         = "getBrokerageStatus"
	OpGetSessionToken            = "getSessionToken"
	OpLogout                     = "logout"
	OpGetSessionValidation       = "getSessionValidation"
	OpGetBrokerageAccounts       = "getBrokerageAccounts"
	OpGetPnl                     = "getPnl"
	OpGetAccountSummary          = "getAccountSummary"
	OpGetContractSymbols         = "getContractSymbols"
	OpGetContractSymbolsFromBody = "getContractSymbolsFromBody"
	OpGetInstrumentInfo          = "getInstrumentInfo"
	OpGetContractRules           = "getContractRules"
	OpGetContractStrikes         = "getContractStrikes"
	OpGetAllAccounts             = "getAllAccounts"
	OpGetAllSubaccounts          = "getAllSubaccounts"
	OpGetUncachedPositions       = "getUncachedPositions"
	OpInvalidatePositionCache    = "invalidatePositionCache"
	OpGetPaginatedPositions      = "getPaginatedPositions"
	OpGetPositionByConid         = "getPositionByConid"
	OpGetPortfolioLedger         = "getPortfolioLedger"
	OpGetAssetAllocation         = "getAssetAllocation"
	OpGetPortfolioSummary        = "getPortfolioSummary"
	OpGetPortfolioMetadata       = "getPortfolioMetadata"
	OpGetMdSnapshot              = "getMdSnapshot"
	OpGetMdHistory               = "getMdHistory"
	OpCloseMdStream              = "closeMdStream"
	OpCloseAllMdStreams          = "closeAllMdStreams"
	OpSubmitNewOrder             = "submitNewOrder"
	OpConfirmOrderReply          = "confirmOrderReply"
	OpPreviewMarginImpact        = "previewMarginImpact"
	OpModifyOpenOrder            = "modifyOpenOrder"
	OpCancelOpenOrder            = "cancelOpenOrder"
	OpGetOpenOrders              = "getOpenOrders"
	OpGetOrderStatus             = "getOrderStatus"
	OpGetTradeHistory            = "getTradeHistory"
)

// Fixture is the canned response for one operation.
type Fixture struct {
	// Status is the HTTP status. Zero means 200 OK.
	Status int
	// Body is the raw JSON body.
	Body string
	// Dynamic, when set, computes the response from the request and takes
	// precedence over Status and Body.
	Dynamic func(*Request) (int, string)
}

// Fixtures is a registry of canned responses keyed by operation ID.
type Fixtures struct {
	mu sync.RWMutex
	m  map[string]Fixture
}

// DefaultFixtures returns the built-in fixture set. Money and quantity fields
// are JSON strings, per ADR 0008.
func DefaultFixtures() *Fixtures {
	f := &Fixtures{m: make(map[string]Fixture)}

	// --- session/auth ---
	f.Set(OpInitializeSession, Fixture{Body: `{"authenticated":true,"established":true}`})
	f.Set(OpGetBrokerageStatus, Fixture{Body: `{"authenticated":true,"established":true,"connected":true}`})
	f.Set(OpGetSessionToken, Fixture{Body: `{"session":"tok-123"}`})
	f.Set(OpLogout, Fixture{Body: `{}`})
	f.Set(OpGetSessionValidation, Fixture{Body: `{"valid":true,"message":"ok"}`})

	// --- account ---
	f.Set(OpGetBrokerageAccounts, Fixture{
		Body: `{"accounts":["U1234567","U7654321"],"aliases":{"U1234567":"Main"}}`,
	})
	f.Set(OpGetPnl, Fixture{
		Body: `{"upnl":{"U1234567.Core":{"dpl":"1.5","el":"2","mv":"3","nl":"4","rowType":"1","upl":"5"}}}`,
	})
	f.Set(OpGetAccountSummary, Fixture{
		Body: `{"accountType":"INDIVIDUAL","netLiquidationValue":"1234.5600","totalCashValue":"100.25","availableFunds":"900.10","SMA":"1200.0000","cashBalances":[{"currency":"USD","balance":"50.5","settledCash":"40.0"}]}`,
	})

	// --- contracts ---
	contractSearch := `[{"conid":265598,"symbol":"AAPL","companyName":"Apple Inc","secType":"STK","description":"Apple Inc","exchange":"NASDAQ"}]`
	f.Set(OpGetContractSymbols, Fixture{Body: contractSearch})
	f.Set(OpGetContractSymbolsFromBody, Fixture{Body: contractSearch})
	f.Set(OpGetInstrumentInfo, Fixture{
		Body: `{"con_id":265598,"symbol":"AAPL","company_name":"Apple Inc","currency":"USD","exchange":"NASDAQ","instrument_type":"STK","local_symbol":"AAPL","multiplier":"1","expiry_full":"","cusip":"037833100","category":"Technology","industry":"Consumer Electronics"}`,
	})
	f.Set(OpGetContractRules, Fixture{
		Body: `{"algoEligible":true,"allOrNoneEligible":true,"canTradeAcctIds":["U1234567"],"defaultSize":"100","limitPrice":"0.01","orderTypes":["MKT","LMT"],"tifTypes":["DAY","GTC"],"TIF":"DAY","negativeCapable":false,"preview":true}`,
	})
	f.Set(OpGetContractStrikes, Fixture{Body: `{"call":["150","155"],"put":["145","140"]}`})

	// --- portfolio ---
	f.Set(OpGetAllAccounts, Fixture{
		Body: `[{"accountId":"U1234567","accountTitle":"Main","currency":"USD","accountAlias":"Main"},{"accountId":"U7654321","accountTitle":"Second","currency":"USD"}]`,
	})
	f.Set(OpGetAllSubaccounts, Fixture{
		Body: `[{"accountId":"U1234567","accountTitle":"Main","currency":"USD"}]`,
	})
	f.Set(OpGetUncachedPositions, Fixture{
		Body: `[{"acctId":"U1234567","conid":265598,"contractDesc":"AAPL","assetClass":"STK","currency":"USD","position":"10.5","avgCost":"145.25","mktPrice":"150.00","mktValue":"1575.00","unrealizedPnl":"49.875"}]`,
	})
	f.Set(OpInvalidatePositionCache, Fixture{Body: `{"status":"success"}`})
	f.Set(OpGetPaginatedPositions, Fixture{Dynamic: paginatedPositions})
	f.Set(OpGetPositionByConid, Fixture{
		Body: `[{"acctId":"U1234567","conid":265598,"contractDesc":"AAPL","assetClass":"STK","currency":"USD","position":"10.5","avgCost":"145.25","mktPrice":"150.00","mktValue":"1575.00","unrealizedPnl":"49.875"}]`,
	})
	f.Set(OpGetPortfolioLedger, Fixture{
		Body: `{"USD":{"acctcode":"U1234567","currency":"USD","cashbalance":"100.25","netliquidationvalue":"1575.00","stockmarketvalue":"1575.00","unrealizedpnl":"49.875","realizedpnl":"0"}}`,
	})
	f.Set(OpGetAssetAllocation, Fixture{
		Body: `{"assetClass":{"long":{"STK":"1000.5"},"short":{}},"sector":{"long":{"Technology":"1000.5"},"short":{}},"group":{"long":{},"short":{}}}`,
	})
	f.Set(OpGetPortfolioSummary, Fixture{
		Body: `{"netliquidation":{"amount":"1575.00","currency":"USD"},"totalcashvalue":{"amount":"100.25","currency":"USD"}}`,
	})
	f.Set(OpGetPortfolioMetadata, Fixture{
		Body: `{"accountId":"U1234567","accountTitle":"Main","currency":"USD","accountAlias":"Main"}`,
	})

	// --- market data ---
	f.Set(OpGetMdSnapshot, Fixture{
		Body: `[{"conid":265598,"31":"150.25","84":"150.20","_updated":1564652478,"server_id":"q0"}]`,
	})
	f.Set(OpGetMdHistory, Fixture{
		Body: `{"symbol":"AAPL","data":[{"t":1564652478,"o":"150.1","h":"151.2","l":"149.9","c":"150.9","v":"1000"}]}`,
	})
	f.Set(OpCloseMdStream, Fixture{Body: `{"status":"success"}`})
	f.Set(OpCloseAllMdStreams, Fixture{Body: `{}`})

	// --- orders ---
	f.Set(OpSubmitNewOrder, Fixture{Body: `[{"order_id":"999","order_status":"PreSubmitted"}]`})
	f.Set(OpConfirmOrderReply, Fixture{Body: `[{"order_id":"999","order_status":"PreSubmitted"}]`})
	f.Set(OpPreviewMarginImpact, Fixture{
		Body: `{"amount":{"initial":"1000.50","maintenance":"800.25"}}`,
	})
	f.Set(OpModifyOpenOrder, Fixture{Body: `[{"order_id":"999","order_status":"PreSubmitted"}]`})
	f.Set(OpCancelOpenOrder, Fixture{Body: `{"order_id":"999","msg":"Request was submitted"}`})
	f.Set(OpGetOpenOrders, Fixture{
		Body: `{"orders":[{"orderId":"999","account":"U1234567","conid":265598,"ticker":"AAPL","orderType":"LMT","side":"BUY","status":"PreSubmitted","timeInForce":"DAY","totalSize":"10","filledQuantity":"0","remainingQuantity":"10","price":"150.00","avgPrice":"0"}]}`,
	})
	f.Set(OpGetOrderStatus, Fixture{
		Body: `{"order_id":"999","order_status":"PreSubmitted","conid":265598,"side":"BUY","filled_quantity":"0","remaining_quantity":"10","average_price":"0"}`,
	})
	f.Set(OpGetTradeHistory, Fixture{
		Body: `[{"order_id":"999","execution_id":"exec-1","account":"U1234567","conid":265598,"symbol":"AAPL","side":"B","size":"10","price":"150.00","commission":"1.00","net_amount":"1499.00","trade_time_r":1564652478000}]`,
	})

	registerCPAPIFixtures(f)
	registerRESTFixtures(f)

	return f
}

// paginatedPositions returns the position page for the requested pageId. Page 0
// holds AAPL and MSFT, page 1 holds SPY, and every other page is empty.
func paginatedPositions(req *Request) (int, string) {
	switch req.Params["pageId"] {
	case "0":
		return http.StatusOK, `[{"acctId":"U1234567","conid":265598,"contractDesc":"AAPL","assetClass":"STK","currency":"USD","position":"10.5","avgCost":"145.25","mktPrice":"150.00","mktValue":"1575.00","unrealizedPnl":"49.875"},{"acctId":"U1234567","conid":8314,"contractDesc":"MSFT","assetClass":"STK","currency":"USD","position":"5","avgCost":"300.10","mktPrice":"310.00","mktValue":"1550.00","unrealizedPnl":"49.5"}]`
	case "1":
		return http.StatusOK, `[{"acctId":"U1234567","conid":1,"contractDesc":"SPY","assetClass":"STK","currency":"USD","position":"2","avgCost":"400.00","mktPrice":"410.00","mktValue":"820.00","unrealizedPnl":"20"}]`
	default:
		return http.StatusOK, `[]`
	}
}

// Get returns the fixture registered for op.
func (f *Fixtures) Get(op string) (Fixture, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	fx, ok := f.m[op]
	return fx, ok
}

// All returns a copy of all registered fixtures.
func (f *Fixtures) All() map[string]Fixture {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := make(map[string]Fixture, len(f.m))
	for k, v := range f.m {
		out[k] = v
	}
	return out
}

// Set registers or replaces the fixture for op.
func (f *Fixtures) Set(op string, fx Fixture) {
	f.mu.Lock()
	if f.m == nil {
		f.m = make(map[string]Fixture)
	}
	f.m[op] = fx
	f.mu.Unlock()
}
