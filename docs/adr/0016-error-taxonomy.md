# 0016 — Error Handling Philosophy

- Status: Accepted
- Date: 2026-09-18

## Context

The SDK surfaces errors from two sources: the local client configuration/state
and the remote IBKR gateway. Callers need a consistent, predictable way to
detect, classify, and handle errors. This ADR documents the philosophy
established in the implementation but not yet written down.

## Decision

### Error types

The SDK defines two categories of errors:

**`*ibkr.Error`** — Remote API failures. Every non-2xx response from the
gateway, and every transport-level failure, is wrapped as `*ibkr.Error`. This
type is the primary error interface callers interact with.

```go
type Error struct {
    Op        string  // operation that failed, e.g. "Account.List"
    Code      string  // IBKR error code, e.g. "1100" or "code not available"
    Message   string  // human-readable description
    HTTPStatus int    // HTTP status code, 0 if not an HTTP response
    RequestID string  // gateway-assigned request ID, if available
    Err       error   // underlying error, if any
}
```

**`*ibkr.ConfigError`** — Invalid client configuration. Returned by
`NewClient` when required options are missing or malformed. This is a fatal
error — the client cannot operate.

```go
type ConfigError struct {
    Field   string
    Message string
}
```

**Sentinel errors** — Package-level variables for common error conditions.
These are values of `*ibkr.Error` and are used in `errors.Is` comparisons:

| Sentinel | When it is returned |
|----------|---------------------|
| `ErrNotAuthenticated` | Session not initialised, or tickle heartbeat expired |
| `ErrSessionExpired` | Gateway reported the session as expired |
| `ErrRateLimited` | Gateway returned a 429; the SDK will retry automatically for safe methods |
| `ErrNotFound` | The requested resource does not exist (404) |
| `ErrInvalidRequest` | Malformed request; the SDK should not have sent it |
| `ErrOrderRejected` | The gateway rejected an order submission |
| `ErrClosed` | The client has been closed; no further operations are valid |
| `ErrStreamingLimit` | The concurrent streaming subscription limit is reached |
| `ErrWSDisconnected` | The WebSocket transport was disconnected (see ADR 0006) |
| `ErrWSReconnected` | The WebSocket transport reconnected automatically |
| `ErrCircuitOpen` | The circuit breaker is open for this operation |

### Error wrapping

Errors are wrapped with `fmt.Errorf` and `%w` to preserve the error chain, so
callers can use `errors.Is` and `errors.As`:

```go
if errors.Is(err, ibkr.ErrNotAuthenticated) { ... }
var ibkrErr *ibkr.Error
if errors.As(err, &ibkrErr) { fmt.Println(ibkrErr.Op, ibkrErr.Code) }
```

`ConfigError` is returned as a plain error; callers use `errors.As`.

### When errors are returned vs logged

| Situation | Behaviour |
|-----------|-----------|
| Terminal failure (e.g. `ErrClosed`) | Return error; caller must handle |
| Non-terminal error on a best-effort operation | Log at `Warn` level; do not return |
| Retryable error (rate limit, 5xx on safe method) | Return after retries exhausted |
| Order mutation failure | Return immediately; no auto-retry (ADR 0009) |
| SDK misconfiguration | Return `*ConfigError` from `NewClient` |

### Panic conditions

The SDK does not panic in normal operation. Panics indicate a programming
error in the SDK itself and should be reported as bugs. Goroutine leaks are
detected by `goleak` in tests.

## Consequences

- Callers can use `errors.Is` and `errors.As` reliably against all SDK errors.
- The sentinel error names must be consistent with their actual meaning; see
  Track C naming fixes in the v0.2.0 release notes.
- No new error types should be added without an ADR.

## Alternatives considered

- **Distinct error types per manager**: Rejected — the single `*ibkr.Error`
  type with an `Op` field is sufficient and simpler for callers.
- **Error codes as typed constants**: Deferred — a future ADR may enumerate
  the IBKR error code set.
