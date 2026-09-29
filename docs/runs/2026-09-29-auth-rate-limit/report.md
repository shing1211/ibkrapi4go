# Report: a precaution that cost 2s per process, one of which bought nothing

- **Date**: 2026-09-29
- **Mode**: BUILD
- **Baseline**: `e1fe64d` (`v1.1.25`)
- **Outcome**: complete

Every CLI invocation cost ~2.0s against a gateway that answers instantly. The previous
run had measured that and recorded a cause it did not have. This run found it.

## The cause, in one log line

`internal/ratelimit.go:67` builds a hardcoded auth bucket:

```go
l.auth = rate.NewLimiter(rate.Limit(1), 1)
```

One request per second, burst 1, not configurable. `isAuthPath` matched
`/iserver/auth/`, `/v1/api/tickle`, `/v1/api/logout` and `/sso/validate`. With burst 1,
the first auth call takes the only token and each subsequent one waits for a refill.

The client's own DEBUG log names it, which is how it was found in a single run:

```
msg="ibkr.ratelimit wait" method=GET  path=/v1/api/iserver/auth/status duration=991.4075ms
msg="ibkr.ratelimit wait" method=POST path=/v1/api/logout              duration=997.3873ms
```

## Finding it took four probes, and three of them ruled something out

The previous run had measured `Session.Initialize` at 1.007s and `cli.Close` at 995ms
against a mock. Two readings pointed at the wrong place:

- `internal/session.go` has a 1s poll ticker, and its comment describes a *previous*
  run fixing exactly this shape of penalty. So the natural suspect was the poll loop.
- Timing each phase gave `Initialize` 1.007s and `Close` 995ms - suspiciously equal,
  and suspiciously like that ticker.

Timing each *request*, and the gap between one response and the next request, killed
that theory. The init request was served in 0s and the auth status request arrived
998ms later. The wait was before the call, not after a failed one.

The mock's recorder showed 4 requests - init, auth/status, accounts, logout - with no
second auth/status, and `Recorder.Record` appends unconditionally, so there was no
de-duplication hiding a second poll. The poll genuinely succeeds first try: **the 1s
buys no information at all.** It is pure waiting for a token.

Ruled out along the way, in order: the poll ticker, the zero-value `internal.Clock`
(its `NewTicker` falls back to `time.NewTicker`), the `httpAPI` methods (plain
`http.Client.Do`), and recorder de-duplication. A DEBUG log then answered it in one run.

## Two waits, two different problems

They came apart cleanly once measured, and only one of them is a bug.

**The logout wait is a bug.** `Session.Close` documents its logout as best-effort and
discards the result, then blocks process exit for a second waiting for a token to
issue a call whose answer is thrown away. `/v1/api/logout` touches no credential, and a
429 on it costs nothing. It is no longer an auth path.

**The auth status wait is not a bug.** Pacing auth endpoints is a deliberate precaution,
pinned by `TestLimiter_AuthPathsAreSlow` and documented in `RATE-LIMITING.md`. It is
simply the wrong shape for a one-shot process, so it became configurable rather than
changed.

## Result

| | Initialize | Close | Total |
|---|---|---|---|
| Before | 1.004s | 998ms | **2.002s** |
| After, no opt-in | 1.004s | 1ms | **1.005s** |
| After + `WithAuthRateLimit(50, 10)` | 4ms | 1ms | **0.004s** |

The unconditional half needs no caller action. The second half exists for callers that
want it, and the default is unchanged, so `TestLimiter_AuthPathsAreSlow` needed no
edit - which is the check that the default really did not move.

## I was wrong about the rationale

I told the user "no ADR, no recorded rationale" for the 1 rps default. That was wrong,
and the mistake was procedural: I grepped `docs/adr/` and `docs/design/` and never
opened `docs/RATE-LIMITING.md`, which lists the figure in a table at line 12 and states
the reason at lines 3-4 - *"IBKR enforces request pacing. The SDK applies client-side
limits to avoid triggering them."* `pkg/ibkr/client.go:74` points at that file.

The ADR is written on the corrected premise. The behaviour was documented all along;
what was missing is configurability, and a *source* for the "IBKR enforces pacing"
claim, which the doc asserts without one.

That doc also **contradicted my change** - line 32 listed `/logout` among the auth
paths - so it is updated in the same change, along with the algorithm list, a new
short-lived-processes section and the testing list. A doc that disagrees with the code
is worse than no doc, and `design-check` is the gate that exists for it.

## Version number

This adds a public option, which strict semver would call a minor. It is released as a
patch on the authority of ADR 0015, whose non-breaking list includes "Adding new
exported symbols" (line 69) and "Adding new optional parameters with zero values"
(line 65). `WithAuthRateLimit` is a new exported symbol with a default that preserves
today's behaviour exactly.

## The open question, unchanged

Does IBKR's gateway actually rate-limit auth endpoints, and at what rate? Nothing in the
repository answers it: the spec is silent, the mock imposes no limit, and no ADR states
it. If the answer is that auth is not specially limited, the correct change is to drop
the auth bucket rather than raise its rate, and `TestLimiter_AuthPathsAreSlow` goes
with it. That needs a live gateway, and it is recorded in ADR 0018 as Accepted with the
question open.

## One interaction documented rather than fixed

The limiter is only constructed when per-endpoint or global limiting is enabled, so
`WithRateLimit(0)` together with `WithGlobalRateLimit(0)` removes auth pacing as well.
`WithAuthRateLimit` can relax pacing but cannot add it to a client that has turned the
limiter off. That is on the option's doc comment rather than restructured here, since
changing the construction guard would alter behaviour for callers who disable limiting
today.
