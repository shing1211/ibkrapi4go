# Report: audit every fixed wait in the test suite

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `cc343ab` (`v1.1.11`)
- **Scope**: item 2 of the post-v1.1.10 recommendation — audit the 21 `time.Sleep`
  sites rather than assuming they were all benign.

## Method

The v1.1.10 fix established a distinction worth applying everywhere: **is this
wait for an event that is guaranteed to happen, or is it for time to pass?**

- A sleep standing in for a *guaranteed* event is a defect. It cannot fail for the
  right reason, only when the machine is slow.
- A sleep waiting for a cooldown, an injected latency, or the *absence* of
  something is legitimate. You cannot wait on an absence, and a time-based state
  transition has no event to wait for.

The first pass found 21 `time.Sleep` sites. Reading them one by one showed that
`time.Sleep` was only half the problem: five more sites were `select` blocks whose
*only* cases were `time.After(N)` and `ctx.Done()` — a sleep wearing a costume,
and invisible to a grep for `time.Sleep`. A small script identified them by
checking whether any arm receives from a channel that carries the thing under
test.

## Result: 11 of 26 were defects

| File | Test | Was | Now |
|------|------|-----|-----|
| `session_test.go` | HappyPath | sleep 20 ms | drive one `tickleRound` |
| `session_test.go` | TickleFailure_Expires | **sleep 250 ms** (5× the 50 ms interval) | drive two rounds |
| `session_test.go` | ReauthorizeAfterTickleFailure | **sleep 250 ms** | drive one round |
| `session_test.go` | TokenCopy | sleep 20 ms | drive one round |
| `session_test.go` | StartTickle_Idempotent | sleep 20 ms | manual clock, count tickers |
| `session_test.go` | TickleGoroutine_NoLeak | sleep 20 ms | wait on the round counter |
| `ws_test.go` | SubscribeDeliversToCorrectSink | `time.After(500 ms)` | wait on both sinks |
| `ws_test.go` | SystemFrameDeliversToSystemSink | `time.After(500 ms)` | wait on the frame |
| `ws_resilience_test.go` | DuplicateUpdatedSequence | `time.After(1 s)` | wait for 2 updates |
| `ws_resilience_test.go` | OutOfOrderSequence | `time.After(1 s)` | wait for 3 frames |
| `ws_resilience_test.go` | NTFAndSORFrames | `time.After(1 s)` | wait for 2 frames |

The remaining 15 are legitimate and were left alone: circuit-breaker cooldowns
(`breaker_test.go`, `metrics_test.go`), injected latency in the fault-injection and
timeout tests, the OAuth tests' deliberate `ts.delay` and its 200 ms
prove-the-call-did-*not*-return window, the goroutine-settle poll and its
slow-shutdown fixture, and the `goleak` settle in `TestMain`.

## One test was not just flaky, it was vacuous

`TestSession_StartTickle_Idempotent` slept 20 ms and then compared two token
reads. `tickleFn` always returned the same token, so the two reads were equal
**whether or not a second tickle loop had been started**. The assertion could not
fail.

It now injects a manual clock, ticks once, and asserts exactly one ticker and one
tickle round. The clock hands each `NewTicker` call its **own** channel, so if two
loops were running one tick would be consumed by both and the count would be 2 —
which is exactly the bug the test exists to catch. Confirmed by breaking the
idempotency guard in `startTickle`:

```
session_test.go:478: tickers = 2; want 1 - a second startTickle must not start a second loop
session_test.go:481: tickle rounds = 2; want 1
```

A clock that handed every loop the same channel would have hidden that.

## The audit found a production defect

Every session test reported 1–2 s. The cause was not the tests: `Initialize` did

```go
clock.Sleep(1 * time.Second)          // before the first poll
ticker := clock.NewTicker(1 * time.Second)
for { ... case <-ticker.C: fetchAuthStatus() }
```

so **every session initialization paid a fixed one-second penalty** before it
could discover a session that was already established. The loop now polls first
and waits second, which converges at the same rate — a session that is not up yet
falls through to the wait — and takes the second off the happy path.

All the session tests dropped from 1–2 s to 0.00 s as a side effect, which is the
clearest evidence the second was never doing anything. `Clock.Sleep` became unused
and was removed, along with its `sleepFunc` hook.

## One unreproduced failure, reported

During verification a single run of the `internal` suite failed, but the output
was not captured. It did not recur in 37 subsequent runs, including 10 full-repo
runs at `GOMAXPROCS=2`. The next full-repo run then failed
`TestWS_NTFAndSORFrames` with "expected system frames, got none" — which is
exactly the defect class this audit targets, in a test that slept a full second
before looking.

So the earlier unreproduced failure is plausibly one of the same, now fixed. It is
recorded here rather than quietly dropped, because the honest claim is "no
observed failure remains" and not "no failure ever occurred".

## Coverage: a false alarm worth recording

One reading came back at **55.0%**, five points below the 58% floor, which would
have broken CI. The hypothesis was that removing ~5 s of waiting had removed the
incidental execution of background goroutines.

That was wrong. Measured three times after the tree settled, coverage is **60.0%**,
identical to HEAD, and `internal` alone is 79.6% both before and after. The 55.0%
reading came from a run that overlapped edits in progress. No regression, but the
check was worth doing rather than assuming.

## Verification

| Check | Result |
|-------|--------|
| `go build ./...` / `go vet ./...` / `gofmt -l .` | clean |
| `go test ./...` | pass |
| `go test -race` | clean |
| full-repo runs at `GOMAXPROCS=2` | 0 failures in 10 |
| goroutine-owning session tests | 0 failures in 25 |
| session tests under `-race`, `GOMAXPROCS=1` | 0 failures in 12 |
| bare fixed waits remaining | **0** (script-checked) |
| coverage | 60.0%, unchanged |
| `golangci-lint` / `check_money` / `check_design` / links / i18n / SPDX | pass |

## Not addressed

- The 15 legitimate sleeps stay, with the reasoning recorded above rather than
  left for the next reader to re-derive.
- Nothing in this run adds production behaviour; the one production change is the
  `Initialize` poll-order fix, which is a latency improvement and a removal of an
  unused clock hook.
