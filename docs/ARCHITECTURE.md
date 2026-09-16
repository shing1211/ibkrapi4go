# Architecture

Design decisions, tradeoffs, and implementation rationale.
Status: **Pre-alpha — v1 targets CPAPI.**

## Dependencies

| Library | Purpose |
|---------|---------|
| `github.com/coder/websocket` | WebSocket client — context-aware, idiomatic, maintained |
| `golang.org/x/time/rate` | Rate limiting (token bucket) |
| `github.com/oapi-codegen/runtime` | Runtime helpers used by generated code |
| `github.com/stretchr/testify` | Test assertions (test-only) |
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
   (`oauth2Bearer`) are distinct; v1 targets the former. See
   [ADR 0001](./adr/0001-two-api-surfaces.md).
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
internal/          transport, session, ratelimit, retry, ws
      │
      ▼
client/            Generated OpenAPI types + HTTP client (DO NOT EDIT)
```

Generated code never leaks to callers. Managers adapt generated request/response
types to stable public types (see [design/04-generated-wrapping.md](./design/04-generated-wrapping.md)).

## Package structure

```
ibkrapi4go/
├── client/            # Generated (DO NOT EDIT)
├── pkg/ibkr/
│   ├── client.go      # NewClient, options, Close
│   ├── auth.go        # SessionManager
│   ├── account.go     # AccountManager
│   ├── portfolio.go   # PortfolioManager
│   ├── trade.go       # TradeManager
│   ├── marketdata.go  # MarketDataManager
│   ├── ws.go          # streaming
│   └── doc.go
├── internal/
│   ├── transport.go   # http.RoundTripper middleware chain
│   ├── session.go     # state machine + tickle
│   ├── ratelimit.go   # token buckets
│   ├── retry.go       # retry policy (safe methods only)
│   └── ws.go          # connection management
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
  → per-endpoint rate limiter
  → global rate limiter
  → HTTP call (with context deadline)
  → 401 → mark session expired (no auto-login)
  → 429/5xx → retry only for idempotent methods
  → error parsing (IBKR envelope → *ibkr.Error)
Response
```

## Error handling

See [ERRORS.md](./ERRORS.md). All API failures surface as `*ibkr.Error` with a
code, message, HTTP status, and request metadata not containing secrets.

## Testing strategy

See [TESTING.md](./TESTING.md): unit tests against `httptest`, WebSocket tests
against `wstest`/local server, integration tests against a paper gateway, and a
codegen reproducibility check.

## Non-goals (v1)

TWS/FIX protocols, account opening/KYC, the `/gw/*` OAuth2 surface, and
GraphQL. See [ROADMAP.md](./ROADMAP.md).

## Future

- v2: `/gw/*` OAuth2 surface + refresh.
- v2: multiple concurrent sessions / gateways.

---

*See [adr/](./adr/) for the decision records that back this document.*
