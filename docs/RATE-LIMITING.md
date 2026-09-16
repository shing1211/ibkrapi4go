# Rate Limiting

IBKR enforces request pacing. The SDK applies client-side limits to avoid
triggering them, and backs off correctly when it does.

## Limits

| Scope | Default | Option | Env |
|-------|--------:|--------|-----|
| Per endpoint (path+method) | 10 req/s | `WithRateLimit(rps, burst)` | `IBKR_RATE_LIMIT` |
| Client-wide | 50 req/s | `WithGlobalRateLimit(rps)` | `IBKR_GLOBAL_RATE_LIMIT` |
| Auth/session endpoints | 1 req/s | fixed | — |

Burst defaults to `2×rps` for the per-endpoint limiter unless overridden.

These figures are **defaults**, not guarantees from IBKR. Individual endpoints
may be stricter; the backoff path handles that.

## Algorithm

Token bucket (`golang.org/x/time/rate`), one bucket per endpoint key plus one
global bucket. A request waits for **both** buckets.

Implemented in `internal/ratelimit.go` (`NewLimiter`, `Limiter.Wait`) and the
`RateLimit` transport middleware; configured with `WithRateLimit` /
`WithGlobalRateLimit` or `IBKR_RATE_LIMIT` / `IBKR_GLOBAL_RATE_LIMIT`.

- Keys are **normalized** paths (`METHOD /v1/api/portfolio/{}/summary`): numeric,
  account-id, and UUID-shaped segments are replaced with `{}`, so per-account
  calls share a bucket.
- Buckets are created lazily and swept when idle to bound memory.
- Auth/session paths (`/iserver/auth/*`, `/tickle`, `/logout`, `/sso/validate`)
  use a fixed 1 req/s bucket.
- `Wait` respects the caller's context; cancellation returns
  `context.Canceled`/`context.DeadlineExceeded`.
- `WithRateLimit(0, …)` disables per-endpoint limiting; `WithGlobalRateLimit(0)`
  disables the global bucket.

## Backoff on 429

1. Read `Retry-After` (seconds or HTTP date). If present, wait that long.
2. Otherwise use exponential backoff with full jitter.
3. Retry only if the request is **safe** (see [ERRORS.md](./ERRORS.md)).
4. Cap total attempts (default 3).
5. If still `429`, return `ErrRateLimited` and pause that endpoint's bucket.

## Head-of-line blocking

The global limiter can stall unrelated endpoints. To bound this, the global
bucket uses a larger burst (default `2×` its rate) and callers may opt out with
`WithGlobalRateLimit(0)` (disables the global bucket) if they manage pacing
elsewhere.

## Testing

- Unit test: steady-state throughput ≤ configured rps within tolerance.
- Unit test: burst up to `burst`, then throttles.
- Unit test: `429` + `Retry-After: 2` delays ~2s (fake clock).
- Unit test: unsafe methods are not retried on `429`.

## Interaction with streaming

WebSocket subscriptions have their own IBKR limits (conids/fields per
subscription, total subscriptions). Those are documented in
[STREAMING.md](./STREAMING.md) and are separate from HTTP pacing.
