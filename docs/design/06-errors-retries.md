# Design 06 — Errors and retries

Implementation-level design. Public error semantics are in [../ERRORS.md](../ERRORS.md).

## Error construction

Every failure is wrapped with operation context:

```go
type Error struct {
    Op         string
    Code       string
    Message    string
    HTTPStatus int
    RequestID  string
    Err        error
}
```

`errorDecode` (transport) builds it from the status and the IBKR envelope. Manager
methods add `Op` (e.g. `"Account.List"`).

## Retry policy

Implemented in `internal/retry.go` (`RetryPolicy`, `Retry` middleware), exposed
as `ibkr.RetryPolicy` / `WithRetryPolicy`:

```go
type RetryPolicy struct {
    MaxAttempts   int           // default 3
    BaseDelay     time.Duration // default 200ms
    MaxDelay      time.Duration // default 5s
    Jitter        bool          // default true (full jitter)
    RetryOnStatus []int         // default 429,500,502,503,504
}
```

Rules:

1. **Safe methods only**: `GET`, `HEAD`, `OPTIONS`.
2. **Order/instruction mutations never retried** ([../adr/0009](../adr/0009-no-auto-retry-orders.md)).
3. `429`: honor `Retry-After` (seconds or HTTP date) if present; else exponential
   backoff.
4. `5xx`/transport error: exponential backoff with full jitter.
5. `401`: never retried here — the session layer handles re-initialization.
6. Total attempts capped by `MaxAttempts`.

## Backoff

`delay = min(MaxDelay, BaseDelay * 2^(attempt-1))`, then, if jitter, multiplied by
`rand(0,1]` (full jitter). `Retry-After`, when present, takes precedence.

## Ambiguous outcomes

If a request may have reached the server but the response was lost (timeout on an
unsafe method), the SDK returns a distinct error instructing the caller to
reconcile — e.g. query open orders — rather than resubmitting. For order
submission this is mandatory, not optional.

## Circuit breaker

Optional, disabled by default. When enabled: after N consecutive transport
failures, short-circuit for a cooldown window, then half-open probe. Threshold and
cooldown are configurable.

## Test matrix

| Scenario | Expectation |
|----------|-------------|
| GET, 500 then 200 | 2 attempts, success |
| POST, 500 | 1 attempt, error |
| Order submit, timeout | 1 attempt, reconcile error |
| 429 with `Retry-After: 2` | waits ~2s, then retry |
| GET, 401 | no retry, `ErrSessionExpired` |
| Context cancelled mid-backoff | returns `context.Canceled` promptly |
