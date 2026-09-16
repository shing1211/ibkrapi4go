# Architecture — ibkr-sdk

> Design decisions, tradeoffs, and implementation rationale.
> Status: **Draft — Phase 0**

---

## Why This SDK Exists

Interactive Brokers offers a modern REST + WebSocket API, but **no official Go SDK**.
Community efforts exist in Python, Java, Rust — but not Go. This SDK fills that gap
with a fully-typed, OpenAPI-driven, idiomatic Go interface.

---

## Design Principles

### 1. OpenAPI-First
The SDK types are generated directly from the [official OpenAPI 3.0 spec](https://api.ibkr.com/gw/api/v3/api-docs).
This guarantees type-level alignment with the API and eliminates manual type maintenance
for the 443 schemas and 185 endpoints.

**Tradeoff:** The spec has minor bugs (e.g., `/balances/query` param mismatch).
These are patched in `scripts/patch_spec.py` before codegen, not worked around manually afterward.

### 2. FutuAPI-Compatible Layout
Follows the same package layout as [futuapi4go](https://github.com/shing1211/futuapi4go):
`pkg/ibkr/` as the public surface, `client/` for generated code, `internal/` for implementation details.
Developers already familiar with futuapi4go will find ibkr-sdk immediately intuitive.

### 3. Idiomatic Go, Not OpenAPI-Native
The generated client is a starting point, not the end product. We wrap it in a
domain-specific API (`cli.Account().List()`, `cli.Portfolio().Positions()`) that
hides HTTP details and provides typed Go semantics.

### 4. Session is State Machine
Authentication in IBKR is session-based (tickle heartbeat every 60s). The `Session`
struct manages: login → tickle loop → token refresh → logout. Callers never manage
session state directly.

### 5. WebSocket is Channel-Based
Go's concurrency model maps perfectly to streaming data. Market data updates arrive
as typed structs on Go channels. Subscribers use `for select` loops — familiar, natural Go.

---

## Package Structure

```
ibkr-sdk/
├── client/           # Generated from OpenAPI spec (DO NOT EDIT)
│   ├── client.gen.go # HTTP client with all endpoints
│   └── types.gen.go  # 443 schema structs
├── pkg/ibkr/         # Public SDK surface
│   ├── client.go     # NewClient, WithOption, exported managers
│   ├── auth.go       # SSO, OAuth2, token management
│   ├── account.go    # AccountManager
│   ├── portfolio.go  # PortfolioManager
│   ├── trade.go      # TradeManager (orders, contracts)
│   ├── marketdata.go # MarketDataManager
│   ├── ws.go         # WebSocket client
│   └── doc.go
├── internal/
│   ├── http.go       # HTTP client wrapper + middleware
│   ├── session.go    # Session state machine
│   ├── ratelimit.go  # Token bucket rate limiter
│   ├── retry.go      # Exponential backoff retry
│   └── ws.go         # WebSocket connection management
├── scripts/
│   ├── codegen.sh    # Generate client/ from spec
│   └── patch_spec.py # Fix spec bugs before codegen
└── test/
    └── integration_test.go
```

---

## Client Lifecycle

```
NewClient()
    ↓
[Session: DISCONNECTED]
    ↓
Auth().SSO() or Auth().OAuth2()
    ↓
[Session: AUTHENTICATED] ← tickle goroutine starts (60s interval)
    ↓
API calls (Account, Portfolio, Trade, MarketData)
    ↓
Client.Close()
    ↓
[Session: CLOSED] ← tickle goroutine stops, logout sent
```

---

## HTTP Client Design

### Middleware Stack (in order)

```
Request
  → Auth header injection (bearer token)
  → Rate limiter (token bucket, per-endpoint 10 req/sec)
  → Request ID / idempotency key injection
  → HTTP call
  → Retry on 429 / 5xx (exponential backoff, max 3)
  → Error parsing (IBKR error format → typed error)
  → Response
```

### Error Handling

IBKR API errors follow this shape:
```json
{
  "error": "some.error.code",
  "message": "Human-readable message",
  "details": {}
}
```

All API errors are wrapped as `*ibkr.Error` with:
- `Code string` — machine-readable error code
- `Message string` — human-readable
- `HTTPStatus int` — raw HTTP status

### Retry Policy

| Condition | Action |
|-----------|--------|
| 429 Too Many Requests | Retry with `Retry-After` header or exponential backoff |
| 500 Internal Server Error | Retry max 3 times with jitter |
| 401 Unauthorized | Trigger session refresh, retry once |
| 429 on auth endpoint | Do NOT retry — rate limit is 1 req/sec for SSO |

---

## WebSocket Architecture

### Connection Flow

```
1. HTTP GET /v1/api/ws → upgrade to WebSocket
2. Send subscribe message:
   {"id": 1, "method": "subscribe", "params": {"conids": [265598], "fields": ["31","83","86"]}}
3. Receive updates as JSON on the WebSocket connection
4. Goroutine dispatches to typed Go channels
```

### Channel API Design

```go
// Subscribe — returns a typed channel
updates, cancel := cli.WS().SubscribeMarketData(ctx, conid, []string{"31","83","86"})
defer cancel()

for {
    select {
    case <-ctx.Done():
        return
    case update := <-updates:
        // update is *MarketDataUpdate, fully typed
        fmt.Println(update)
    }
}
```

### Auto-Reconnect

WebSocket connection is monitored by a heartbeat goroutine. On disconnect:
1. Exponential backoff (1s, 2s, 4s, 8s, max 30s)
2. Reconnect and re-subscribe to all active channels
3. Re-send any pending subscription requests

---

## Rate Limiting

IBKR enforces **10 requests per second per endpoint** globally, and lower limits on
some endpoints (auth: 1/sec, tickle: no limit explicitly stated).

Implementation: **token bucket algorithm** per endpoint.

```go
type Limiter struct {
    buckets map[string]*rate.Limiter  // per-endpoint
    mu      sync.Mutex
    global  *rate.Limiter             // global 50 req/sec
}
```

On `429`: parse `Retry-After` header, wait that duration before retry.

---

## Testing Strategy

### Unit Tests
- Mock `*http.Client` using `httptest.NewServer`
- Test request serialization, response parsing, error handling
- No network, no account required

### Integration Tests
- Requires running Client Portal Gateway
- Use **paper account** credentials (never production keys in tests)
- `IBKR_GATEWAY`, `IBKR_USERNAME`, `IBKR_PASSWORD` env vars
- Tests are tagged: `//go:build integration`

### Spec Compliance Tests
- Re-run codegen and diff output to detect spec drift
- Run as CI check on every spec update

---

## Security Considerations

### Secrets Management
- API credentials **never** stored in config files
- Use environment variables or IBKR's own credential system
- Token stored in memory only — not persisted to disk

### HTTPS Only
- All API calls over HTTPS (mandatory by IBKR for production)
- Local gateway (`localhost:5000`) uses self-signed cert — handled by default TLS verification

### Rate Limit as Security
- Rate limiting prevents accidental credential lockout from too many requests
- Circuit breaker prevents cascade failures

---

## Future Considerations

### v2 — TWS API Support
Separate `ibkr-tws` package for the proprietary TWS socket protocol.
Not in scope for v1 — different protocol, different expertise required.

### v2 — Connection Pooling
Support multiple concurrent sessions (multi-account, multi-gateway).
Currently designed for single-session use.

### v2 — GraphQL?
IBKR does not currently offer GraphQL. Monitor for future API additions.

---

*Last updated: 2026-09-16*
