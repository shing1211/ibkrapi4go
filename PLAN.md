# Implementation Plan — ibkr-sdk

> Phased plan for building the complete Interactive Brokers Web API Go SDK.
> Generated: 2026-09-16. Scope: Full (185 endpoints, 443 schemas).

---

## Overview

| Phase | Focus | Endpoints | Duration |
|-------|-------|-----------|----------|
| **0** | Repo setup, docs, codegen infrastructure | — | Day 1 |
| **1** | Core: Auth, Session, Client, Account, Portfolio | ~60 | Day 1–2 |
| **2** | Trading: Orders, Contracts, Market Data | ~55 | Day 2–3 |
| **3** | Streaming: WebSocket, real-time data | ~4 + WS | Day 3–4 |
| **4** | Advanced: FA, Scanner, Alerts, Reports, Banking | ~66 | Day 4–5 |
| **5** | Polish: rate limiting, retries, tracing, tests | all | Day 5–6 |

---

## Phase 0 — Repository Setup *(Day 1)*

### 0.1 Repository Structure

```
ibkr-sdk/
├── client/              # Generated OpenAPI types + HTTP client
│   ├── client.gen.go    # Auto-generated from OpenAPI spec
│   └── types.gen.go     # Auto-generated schema types
├── pkg/ibkr/            # Public SDK surface
│   ├── client.go        # Main client entry point
│   ├── auth.go          # Authentication & session management
│   ├── account.go       # Account operations
│   ├── portfolio.go     # Portfolio & positions
│   ├── trade.go         # Order management
│   ├── marketdata.go    # Market data
│   ├── ws.go            # WebSocket client
│   └── doc.go
├── internal/
│   ├── http.go          # HTTP client with middleware
│   ├── session.go       # Session state machine
│   ├── ratelimit.go     # Rate limiter
│   └── retry.go         # Retry logic
├── docs/
│   ├── ARCHITECTURE.md
│   └── SESSIONS.md
├── scripts/
│   └── codegen.sh       # OpenAPI → Go codegen
├── test/
│   └── integration_test.go
├── examples/
│   ├── account_list.go
│   ├── portfolio_positions.go
│   ├── place_order.go
│   ├── market_data_snapshot.go
│   └── ws_stream.go
├── SPEC.md
├── PLAN.md
├── README.md
├── ARCHITECTURE.md
├── CONTRIBUTING.md
├── LICENSE
├── go.mod
└── go.sum
```

### 0.2 Codegen Infrastructure

**Problem:** `oapi-codegen` fails on `/gw/api/v1/balances/query` due to path param mismatch.

**Workaround:** Patch the spec before codegen:
1. Remove the problematic path from the spec JSON
2. Generate types for the remaining 184 endpoints
3. Manually define the missing type

```bash
# scripts/codegen.sh
python3 scripts/patch_spec.py /tmp/ibkr_spec.json > specs/ibkr_patched.json
oapi-codegen \
  --package=ibkr \
  --generate=types,client \
  specs/ibkr_patched.json > client/client.gen.go
# Manual patch for /balances/query
```

### 0.3 Module Init

```bash
go mod init github.com/shing1211/ibkrapi4go
go get github.com/google/wire/...   # dependency injection
go get github.com/gorilla/websocket  # WebSocket
go get golang.org/x/time/rate        # rate limiting
go get github.com/stretchr/testify   # testing
```

---

## Phase 1 — Core SDK *(Day 1–2)*

### 1.1 Client Entry Point

```go
// pkg/ibkr/client.go
type Client struct {
    gatewayURL string
    http       *http.Client
    session    *Session
    auth       *AuthManager
    account    *AccountManager
    portfolio  *PortfolioManager
}

func NewClient(opts ...ClientOption) (*Client, error)
func (c *Client) Close() error
```

### 1.2 Session Management

- `POST /v1/api/tickle` — 60-second heartbeat
- `POST /v1/api/logout` — graceful disconnect
- Auto-reconnect on gateway restart
- Token stored in memory, refreshed on 401

### 1.3 Authentication

- **SSO**: `POST /v1/api/iserver/auth/ssodh/init` → tickle token
- **OAuth2**: `POST /v1/api/oauth/access_token`
- **SAML/SSO Browser**: `POST /gw/api/v1/sso-sessions`

### 1.4 Account Management *(14 endpoints)*

```
GET  /gw/api/v1/accounts              → List all accounts
GET  /gw/api/v1/accounts/{id}/details → Account details
GET  /v1/api/iserver/accounts         → Brokerage accounts
GET  /v1/api/iserver/account/{id}/summary → Account summary
GET  /v1/api/iserver/account/{id}/summary/balances → Balance summary
GET  /v1/api/iserver/account/{id}/summary/margins → Margin summary
GET  /v1/api/iserver/account/{id}/summary/available_funds → Funds
GET  /v1/api/iserver/account/{id}/summary/market_value → Market value
```

### 1.5 Portfolio *(21 endpoints)*

```
GET  /v1/api/portfolio/accounts              → List all portfolio accounts
GET  /v1/api/portfolio/{accountId}/positions → All positions (paginated)
GET  /v1/api/portfolio/{accountId}/position/{conid} → Single position
GET  /v1/api/portfolio/{accountId}/ledger    → Portfolio ledger
GET  /v1/api/portfolio/{accountId}/summary   → Portfolio summary
GET  /v1/api/portfolio/{accountId}/allocation → Asset allocation
POST /v1/api/portfolio/{accountId}/positions/invalidate → Refresh cache
```

### 1.6 Phase 1 Deliverables

- [ ] `pkg/ibkr/client.go` — main client
- [ ] `internal/session.go` — tickle heartbeat
- [ ] `pkg/ibkr/auth.go` — SSO + OAuth2
- [ ] `pkg/ibkr/account.go` — account operations
- [ ] `pkg/ibkr/portfolio.go` — portfolio + positions
- [ ] `internal/http.go` — HTTP client with error handling
- [ ] Unit tests for session state machine

---

## Phase 2 — Trading *(Day 2–3)*

### 2.1 Order Management *(11 endpoints)*

```
POST /v1/api/iserver/account/{accountId}/orders       → Submit new order
POST /v1/api/iserver/account/{accountId}/orders/whatif → Preview margin impact
POST /v1/api/iserver/account/{accountId}/order/{orderId} → Modify order
DELETE /v1/api/iserver/account/{accountId}/order/{orderId} → Cancel order
GET  /v1/api/iserver/account/orders          → Get all open orders
GET  /v1/api/iserver/account/order/status/{orderId} → Order status
GET  /v1/api/iserver/account/trades          → Trade history
```

### 2.2 Contract / Instrument Search *(17 endpoints)*

```
POST /v1/api/iserver/secdef/search → Search by symbol (body)
GET  /v1/api/iserver/secdef/search → Search by symbol (query)
GET  /v1/api/iserver/contract/{conid}/info → Instrument info
GET  /v1/api/iserver/contract/{conid}/info-and-rules → Info + rules
GET  /v1/api/iserver/contract/{conid}/algos → Available algos
POST /v1/api/iserver/contract/rules → Contract rules
GET  /v1/api/iserver/secdef/info    → Contract details
GET  /v1/api/iserver/secdef/strikes → Strike prices
GET  /v1/api/iserver/secdef/bond-filters → Bond filters
GET  /v1/api/trsrv/stocks           → Stock by symbol
GET  /v1/api/trsrv/futures          → Futures by symbol
GET  /v1/api/trsrv/secdef           → Instrument definition
GET  /v1/api/contract/trading-schedule → Trading schedule
```

### 2.3 Market Data *(4 endpoints)*

```
GET /v1/api/iserver/marketdata/snapshot    → Live snapshot
GET /v1/api/iserver/marketdata/history     → Historical OHLC
POST /v1/api/iserver/marketdata/unsubscribe → Unsubscribe
GET  /v1/api/iserver/marketdata/unsubscribeall → Unsubscribe all
```

### 2.4 Phase 2 Deliverables

- [ ] `pkg/ibkr/trade.go` — order operations
- [ ] `pkg/ibkr/contract.go` — instrument search
- [ ] `pkg/ibkr/marketdata.go` — market data snapshot + history
- [ ] Order pre-flight validation
- [ ] Unit tests for order building

---

## Phase 3 — WebSocket Streaming *(Day 3–4)*

### 3.1 WebSocket Connection

```
GET /v1/api/ws → Upgrade to WebSocket
```

- Connect to `/v1/api/ws`
- Send JSON subscribe messages
- Receive typed updates on Go channels

### 3.2 Streaming Protocol

```go
// Subscribe to market data
type WSSubscribe struct {
    ID         int      `json:"id"`
    Method     string   `json:"method"`
    Params     WSParams `json:"params"`
}

type WSParams struct {
    ConIDs []int  `json:"conids"`
    Fields []string `json:"fields"`  // e.g. ["31","83","86","88"]
}

// Receive updates
type WSMarketUpdate struct {
    ConID int     `json:"conid"`
    Field int     `json:"field"`
    Value string  `json:"value"`
    TS    int64   `json:"ts"`
}
```

### 3.3 Channel-based API

```go
// idiomatic Go channel subscription
ch, err := cli.MarketData().Subscribe(ctx, conid, []string{"31", "83", "86"})
for {
    select {
    case update := <-ch:
        fmt.Printf("BID: %s ASK: %s\n", update.Bid, update.Ask)
    case <-ctx.Done():
        return
    }
}
```

### 3.4 Phase 3 Deliverables

- [ ] `pkg/ibkr/ws.go` — WebSocket client
- [ ] `internal/ws.go` — connection management
- [ ] Channel-based subscriptions (market data, orders, notifications)
- [ ] Auto-reconnect with backoff
- [ ] Integration test against paper account

---

## Phase 4 — Advanced APIs *(Day 4–5)*

### 4.1 Financial Advisor *(18 endpoints)*

- Model portfolios: `GET/POST /v1/api/fa/model/*`
- Allocation groups: `GET/POST/PUT/DELETE /v1/api/iserver/account/allocation/*`
- FA presets: `GET/POST /v1/api/fa/fa-preset/*`

### 4.2 Scanner *(2 endpoints)*

```
POST /v1/api/iserver/scanner/run   → Run market scanner
GET  /v1/api/iserver/scanner/params → Scanner parameters
```

### 4.3 Alerts *(6 endpoints)*

```
POST /v1/api/iserver/account/{accountId}/alert       → Create alert
GET  /v1/api/iserver/account/{accountId}/alerts      → List alerts
DELETE /v1/api/iserver/account/{accountId}/alert/{alertId} → Delete
```

### 4.4 Watchlists *(4 endpoints)*

```
GET  /v1/api/iserver/watchlists    → All watchlists
GET  /v1/api/iserver/watchlist     → Single watchlist
POST /v1/api/iserver/watchlist     → Create watchlist
DELETE /v1/api/iserver/watchlist   → Delete watchlist
```

### 4.5 Reports & Statements *(6 endpoints)*

```
POST /gw/api/v1/statements         → Generate statement
GET  /gw/api/v1/statements/available → Available statements
POST /gw/api/v1/trade-confirmations → Fetch trade confirmations
POST /gw/api/v1/tax-documents      → Fetch tax documents
```

### 4.6 Banking *(20 endpoints)*

```
POST /gw/api/v1/external-cash-transfers → Cash transfer
POST /gw/api/v1/external-asset-transfers → Position transfer (ACATS)
POST /gw/api/v1/bank-instructions       → Bank instructions
```

### 4.7 Portfolio Analyst *(4 endpoints)*

```
POST /v1/api/pa/performance   → Account performance
POST /v1/api/pa/allperiods    → All time periods
POST /v1/api/pa/allocation    → Portfolio allocation
POST /v1/api/pa/transactions  → Transaction history
```

### 4.8 Other *(remainder)*

- OAuth 1.0a, SSO Browser Sessions, SSO Sessions
- FYI Notifications
- Restrictions / PTC
- Event Contracts (forecast)
- Tax Vouchers
- Enumerations, Forms, Client Instructions

---

## Phase 5 — Polish & Hardening *(Day 5–6)*

### 5.1 Rate Limiting

- Per-endpoint rate limit: **10 req/sec**
- Global rate limit: **50 req/sec**
- Exponential backoff on `429 Too Many Requests`
- Token bucket algorithm

### 5.2 Retry Logic

- Max 3 retries with jitter
- Idempotency keys on POST/PUT/DELETE
- Circuit breaker on persistent failures

### 5.3 Observability

- OpenTelemetry tracing (opt-in)
- Structured logging (`slog`)
- Metrics: request count, latency, error rate

### 5.4 Testing

- Unit tests: mock HTTP round-tripper (`httptest`)
- Integration tests: against paper account
- Spec compliance tests: compare generated types to spec

### 5.5 Documentation

- All exported types documented
- godoc on pkg/ibkr
- `scripts/codegen.sh` documented
- `examples/` coverage

---

## Verification Strategy

| Stage | Method | Account Required |
|-------|--------|-----------------|
| Phase 0–1 | Unit tests + Mock Server (`prism`) | No |
| Phase 2 | `httptest` round-tripper | No |
| Phase 3 | WebSocket echo test | No |
| Phase 4 | Integration tests | **Paper account** |
| Phase 5 | Full integration | Paper → live |

---

## Out of Scope

- TWS API (separate protocol, not REST)
- FIX protocol
- Account opening / KYC workflows
- Non-REST endpoints (this spec only)

---

*Last updated: 2026-09-16*
