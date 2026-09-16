# Design 01 — Transport

The transport is the single path through which every HTTP request flows. It is an
`http.RoundTripper` chain so it composes with `net/http` and is testable.

## Chain

```
request
  └─ requestID → userAgent → auth → rateLimit(perEndpoint) → rateLimit(global)
       → http.Transport.Do
       → retry (safe methods only)
       → errorDecode
response / *ibkr.Error
```

Order matters: auth runs before rate limiting so a rate-limited retry reuses the
same credential; error decoding runs last so it sees the final status.

## Interfaces

```go
type RoundTripper interface {
    RoundTrip(*http.Request) (*http.Response, error)
}

type Middleware func(next RoundTripper) RoundTripper
```

## Responsibilities

| Layer | Does |
|-------|------|
| requestID | attaches `X-Request-ID` (UUID) and stores it in context for error correlation |
| userAgent | sets `User-Agent` (`ibkrapi4go/<version>`) if unset |
| auth | injects `Authorization: Bearer <token>` from the credential provider |
| rateLimit (per-endpoint) | token bucket keyed by `METHOD templatedPath` |
| rateLimit (global) | client-wide token bucket |
| retry | exponential backoff + jitter; honoring `Retry-After`; safe methods only |
| errorDecode | maps status + IBKR envelope to `*ibkr.Error` |

## Rules

- Never modify the caller's request body for retries without buffering it first
  (safe methods have no body; unsafe methods are not retried, which sidesteps this).
- Always honor the request context; cancellation must abort in-flight I/O.
- Redact `Authorization`, `Cookie`, and token-bearing headers from all logs/errors.
- The generated client is given this `http.Client`; it does not build its own.

## Testing

- `httptest.Server` captures requests; assert headers, ids, and ordering.
- Rate limiter uses an injectable clock.
- Retry tests use a server that fails N times then succeeds, asserting attempt
  count and that unsafe methods are single-attempt.
