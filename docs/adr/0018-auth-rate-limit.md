# 0018: Auth rate limiting is a client-side precaution, and is configurable

- **Status**: Accepted (with an open question)
- **Date**: 2026-09-29
- **Relates to**: [RATE-LIMITING.md](../RATE-LIMITING.md), [0016-error-taxonomy.md](./0016-error-taxonomy.md)

## Context

`internal/ratelimit.go` paces auth and session endpoints with a hardcoded
`rate.NewLimiter(rate.Limit(1), 1)` - one request per second, burst 1. It is not
configurable, and the doc comment calls it fixed.

That bucket is not a measured property of IBKR's gateway. `RATE-LIMITING.md` says so
itself: *"These figures are **defaults**, not guarantees from IBKR."* The client
already has a second mechanism for the same concern - ADR 0016 documents
`ErrRateLimited` with automatic retry on safe methods, honouring `Retry-After` - so
there are two answers to "do not hammer auth", and only one of them was configurable.

The consequence was invisible until something measured it. A short-lived process that
initialises a session and exits pays twice:

| Request | Auth bucket | Wait |
|---------|-------------|------|
| `POST /v1/api/iserver/auth/ssodh/init` | takes the only token | - |
| `GET /v1/api/iserver/auth/status` | empty, waits for refill | ~991ms |
| `POST /v1/api/logout` | empty, waits for refill | ~997ms |

That is ~2.0s added to every invocation of the `ibkr` CLI, and to any library caller
that opens and closes a session. The client's own DEBUG log names the cause
(`ibkr.ratelimit wait`), which is how it was found. Against a local gateway the auth
status poll succeeds on the first try, so the wait buys no information at all.

The second wait is the sharper defect. `Session.Close` documents its logout as
best-effort and discards the result, then blocks process exit for a second waiting
for a token to issue a call whose answer is thrown away.

## Decision

1. **`/v1/api/logout` is no longer paced as an auth endpoint.** It touches no
   credential, its result is discarded, and a 429 on it costs nothing. It uses the
   ordinary per-endpoint bucket, where `WithRateLimit` governs it. This is
   unconditional and changes no real request - only the wait before one whose result
   is ignored. Measured: `Close` 998ms -> 1ms.

2. **The auth bucket becomes configurable via `WithAuthRateLimit(rps, burst)`.** The
   default stays 1 req/s, burst 1, so nothing changes for a caller who does nothing.
   `rps<=0` disables auth pacing.

The limiter is only constructed when per-endpoint or global limiting is enabled, so
this option can relax auth pacing but cannot add it to a client that has turned the
limiter off entirely.

`TestLimiter_AuthPathsAreSlow` is unchanged and still pins the pacing for the paths
that remain. It is paired with `TestLimiter_LogoutIsNotAuthPaced`, which pins the one
that was removed: one test guards what stays in the bucket, the other guards what left
it, so neither can change without the other being noticed.

## Why the default was not raised

Relaxing the default would trade a 2s CLI for 429s against a real gateway, and there is
no evidence either way about what the gateway tolerates. The default stays until
someone measures it. The option exists so the decision can be made per call site
rather than once, globally, by whoever guesses.

## Open question

**Does IBKR's gateway actually rate-limit auth endpoints, and at what rate?** Nothing
in the repository answers this: the spec is silent, the mock gateway imposes no limit,
and the ADRs never state it. `RATE-LIMITING.md` asserts that "IBKR enforces request
pacing" without a source.

This ADR is Accepted with that question open, and should be revisited when someone can
observe a real gateway. If the answer is that auth endpoints are not specially limited,
the correct change is to drop the auth bucket rather than to raise its rate, and
`TestLimiter_AuthPathsAreSlow` goes with it.

## Consequences

- Every CLI invocation and every open-then-close library session is ~1.0s faster with
  no opt-in.
- A short-lived process can opt out entirely: measured 2.002s -> 0.004s.
- A long-running client should leave the default alone. It is not paying the cost this
  ADR removes, because it makes auth calls minutes apart rather than back to back.
- `RATE-LIMITING.md` had listed `/logout` among the auth paths; it is updated here and
  in the same change, since a doc that contradicts the code is worse than no doc.
