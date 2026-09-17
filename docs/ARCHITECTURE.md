# Architecture

Design decisions, tradeoffs, and implementation rationale.
Status: **Pre-alpha — all 185 operations implemented (115 CPAPI + 70 IB REST).**

## Dependencies

| Library | Purpose |
|---------|---------|
| `github.com/coder/websocket` | WebSocket client — context-aware, idiomatic, maintained |
| `golang.org/x/time/rate` | Rate limiting (token bucket) |
| `github.com/oapi-codegen/runtime` | Runtime helpers used by generated code |
| `go.uber.org/goleak` | Goroutine leak detection (test-only) |
| `oapi-codegen` (CLI) | OpenAPI → Go codegen (build-time only) |

No web frameworks. No DI frameworks. Standard `net/http` for all HTTP client
operations. New dependencies require an [ADR](./adr/).

## Why this SDK exists

Interactive Brokers offers a REST + WebSocket API but no official Go SDK. This
project fills that gap with a typed, OpenAPI-driven, idiomatic Go client.

## Design principles

1. **OpenAPI-first.** Types are generated from the official spec; the index is
   regenerated, not hand-maintained. Tradeoff: the spec has defects, patched in
   `scripts/patch_spec.py` (see [CODEGEN.md](./CODEGEN.md)).
2. **Two surfaces, explicitly.** `/v1/api` (`ssoBearer`) and `/gw/api/*`
   (`oauth2Bearer`) are distinct and both implemented. See
   [ADR 0001](./adr/0001-two-api-surfaces.md) and
   [ADR 0011](./adr/0011-oauth2-surface.md).
3. **Idiomatic Go, not OpenAPI-native.** Generated code is a starting point; the
   public API wraps it in managers.
4. **Session is a state machine.** See [SESSIONS.md](./SESSIONS.md).
5. **WebSocket is channel-based.** See [STREAMING.md](./STREAMING.md).
6. **Money is never `float64`.** See [ADR 0008](./adr/0008-numeric-precision.md).
7. **Order mutations never retry automatically.** See
   [ADR 0009](./adr/0009-no-auto-retry-orders.md).

## Layering

```
pkg/ibkr/          Public API: Client, managers, options, errors
      │
      ▼
internal/          transport, session, oauth, ratelimit, retry, ws, metrics
      │
      ▼
client/            Generated OpenAPI types + HTTP client (DO NOT EDIT)
```

`internal/mockgateway` is a server-side test/development aid, not part of the
request path above; `cmd/ibkr-mock-gateway` and `examples/mock` use it.

Generated code never leaks to callers. Managers adapt generated request/response
types to stable public types (see [design/04-generated-wrapping.md](./design/04-generated-wrapping.md)).

## Package structure

```
ibkrapi4go/
├── client/            # Generated (DO NOT EDIT)
├── pkg/ibkr/
│   ├── client.go      # Client, NewClient, options, Close, accessors
│   ├── auth.go        # SessionManager
│   ├── account.go     # AccountManager
│   ├── portfolio.go   # PortfolioManager
│   ├── contract.go    # TradeManager + contract lookups
│   ├── trade.go       # TradeManager (orders)
│   ├── marketdata.go  # MarketDataManager
│   ├── trading_accounts.go  # TradingAccountManager
│   ├── alerts.go      # AlertManager
│   ├── events.go      # ForecastManager (event contracts)
│   ├── scanner.go     # ScannerManager
│   ├── allocation.go  # AllocationManager
│   ├── models.go      # ModelManager
│   ├── notifications.go # FYIManager
│   ├── oauth1.go      # OAuthManager (OAuth1)
│   ├── watchlists.go  # WatchlistManager
│   ├── performance.go # PerformanceManager
│   ├── rest.go        # RESTSurface (accounts, statements, requests, tax docs, confirmations, tax vouchers)
│   ├── rest_*.go      # REST banking, utilities, balances, restrictions, SSO, echo
│   ├── response.go    # netDo + json.Number decoding helpers
│   ├── ids.go         # ConID, Field, Side, OrderType, TimeInForce
│   ├── pagination.go  # PositionIterator
│   ├── ws.go          # streaming Subscription (coder/websocket)
│   ├── oauth.go       # OAuth2 options for the REST surface
│   └── doc.go
├── internal/
│   ├── transport.go     # middleware chain (request id, UA, auth, errors, timeout)
│   ├── session.go       # state machine + tickle
│   ├── ws.go            # WebSocket connection management
│   ├── oauth.go         # OAuth2 token source (secret + private_key_jwt)
│   ├── jwt.go           # RS256 JWT assertions for the OAuth2 surface
│   ├── ratelimit.go     # per-endpoint + global token buckets
│   ├── retry.go         # safe-method retry + Retry-After
│   ├── observability.go # request logging, redaction, telemetry hooks
│   ├── breaker.go       # optional circuit breaker
│   ├── errors.go        # *Error type, sentinels, ConfigError
│   └── mockgateway/     # in-repo mock of both API surfaces (test/dev aid)
├── cmd/                 # standalone binaries (ibkr-mock-gateway)
├── examples/            # runnable examples (mock)
├── scripts/
└── docs/
```

## Client lifecycle

```
NewClient(options...)
   │
   ▼
SessionManager.Initialize(ctx)   → tickle goroutine starts
   │
   ▼
Manager calls (Account, Portfolio, Trade, MarketData)
   │
   ▼
Client.Close()                   → tickle stops, logout, ws closed
```

## Transport

All requests flow through a `http.RoundTripper` chain (see
[design/01-transport.md](./design/01-transport.md)):

```
Request
  → request ID + User-Agent
  → auth header injection (bearer)
  → logging + telemetry hooks
  → circuit breaker (optional)
  → retry (safe methods only; honors Retry-After)
  → per-endpoint rate limiter
  → global rate limiter
  → per-request timeout (when the caller sets none)
  → HTTP call
  → error parsing (IBKR envelope → *ibkr.Error)
Response
```

## Error handling

See [ERRORS.md](./ERRORS.md). All API failures surface as `*ibkr.Error` with a
code, message, HTTP status, and request metadata not containing secrets.

## Testing strategy

See [TESTING.md](./TESTING.md): unit tests against `httptest`, manager and
WebSocket tests against the in-repo mock gateway
([`internal/mockgateway`](./MOCK-GATEWAY.md)), integration tests against a paper
gateway, and a codegen reproducibility check.

## Non-goals (v1)

TWS/FIX protocols, account opening/KYC, and GraphQL. See
[ROADMAP.md](./ROADMAP.md). The `/gw/*` OAuth2 surface was added in Phases 5–6
and is no longer a non-goal.

## Future

- Multiple concurrent sessions / gateways.
- Optional higher-level convenience helpers.

---

*See [adr/](./adr/) for the decision records that back this document.*
