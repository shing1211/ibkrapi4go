# Errors

## Error type

Every API failure surfaces as `*ibkr.Error`:

```go
type Error struct {
    Op         string      // logical operation, e.g. "Account.List"
    Code       string      // IBKR error code, if provided
    Message    string      // human-readable, safe to log
    HTTPStatus int         // raw HTTP status (0 if transport-level)
    RequestID  string      // per-request correlation id
    Err        error       // underlying error, if any
}

func (e *Error) Error() string
func (e *Error) Unwrap() error
```

`Message` is sanitized: tokens, cookies, and `Authorization` headers are never
included.

## Sentinels

```go
var (
    ErrNotAuthenticated = errors.New("ibkr: gateway not authenticated")
    ErrSessionExpired   = errors.New("ibkr: session expired")
    ErrRateLimited      = errors.New("ibkr: rate limited")
    ErrNotFound         = errors.New("ibkr: not found")
    ErrInvalidRequest   = errors.New("ibkr: invalid request")
    ErrOrderRejected    = errors.New("ibkr: order rejected")
    ErrClosed           = errors.New("ibkr: client closed")
    ErrStreamingLimit   = errors.New("ibkr: streaming limit exceeded")
    ErrStreamDisconnected = errors.New("ibkr: ws: disconnected")
    ErrStreamReconnected  = errors.New("ibkr: ws: reconnected")
    ErrCircuitOpen        = errors.New("ibkr: circuit breaker open")
)
```

`ErrStreamDisconnected` and `ErrStreamReconnected` are delivered on a streaming
subscription's `Errors()` channel to signal a dropped connection and a
subsequent successful reconnect (see [STREAMING.md](./STREAMING.md)).

Match with `errors.Is` / `errors.As`:

```go
var e *ibkr.Error
if errors.As(err, &e) && e.HTTPStatus == 401 {
    // session problem
}
```

## HTTP status mapping

| Status | Sentinel | Retry? |
|-------:|----------|--------|
| 400 | `ErrInvalidRequest` | No |
| 401 | `ErrSessionExpired` | No (re-initiate session) |
| 403 | `ErrNotAuthenticated` | No |
| 404 | `ErrNotFound` | No |
| 409 | — (carry message) | No |
| 429 | `ErrRateLimited` | Yes, honor `Retry-After` |
| 500/502/503/504 | — | Yes, **idempotent methods only** |
| transport/DNS/TLS | — | Yes, **idempotent methods only** |

## Retry safety

See [design/06-errors-retries.md](./design/06-errors-retries.md) and
[ADR 0009](./adr/0009-no-auto-retry-orders.md):

- **Safe (retryable):** `GET`, `HEAD`, `OPTIONS`.
- **Unsafe (never auto-retried):** `POST`, `PUT`, `PATCH`, `DELETE`.
- Order/instruction **mutations are never retried automatically**, regardless of
  the HTTP method.
- Retries use exponential backoff with full jitter; default budget 3 attempts
  for safe methods only.

## Ambiguous outcomes

When an order mutation (submit/modify/cancel) times out, the request may or may
not have reached IBKR. The SDK never resubmits. It returns an `*Error` with
`Code == "ambiguous"` and a message directing the caller to reconcile via
`Trade().OpenOrders` / `Trade().OrderStatus` before retrying. See
[design/09](./design/09-orders-and-confirmation.md).

## OAuth2 token errors

Token acquisition on the IB REST surface returns an `*Error` with
`Op == "OAuth.Token"` and the token endpoint's HTTP status. A 401 maps to
`ErrSessionExpired`. Tokens are refreshed on the next call rather than retried
blindly (see [adr/0011](./adr/0011-oauth2-surface.md)).

## IBKR error envelope

IBKR returns errors in a few shapes; the SDK normalizes them:

```json
{ "error": "code", "message": "text", "details": {} }
```

Some endpoints return a `200` with an embedded error/reply array (notably order
submission, which returns reply items requiring confirmation). Those are handled
in the manager layer and surfaced as `ErrOrderRejected` or a reply handle — see
[design/09-orders-and-confirmation.md](./design/09-orders-and-confirmation.md).

## Logging

Errors log at `warn`/`error` with `Op`, `Code`, `HTTPStatus`, `RequestID`.
Request/response bodies are logged at `debug` only, with redaction applied.
