# Report: the gate for a defect that passed every gate

- **Date**: 2026-09-29
- **Mode**: BUILD
- **Baseline**: `46d4535` (`v1.1.26`)
- **Outcome**: complete

Test-only release. One new file, no production change.

## The gap this closes

The 2.0s-per-invocation defect fixed in `v1.1.26` was live for two releases. It was not
caught by a test, a linter, a checker, a design gate, or a review, because nothing in the
repository asserted on request pacing. Every gate was green the whole time.

Fixing the symptom without adding a signal leaves the class open.

## The first proposal was wrong

`v1.1.26`'s `next-phase.md` said:

> **A CLI latency check in CI.** This run found a 2s regression-class defect that no
> test, gate or linter would ever have caught, because nothing asserted on wall clock.
> A single test that an Initialize-then-Close cycle completes well inside a second would
> have caught it at any point in the last two releases.

That is a wall-clock assertion, and it is flaky. A loaded CI runner can blow any budget,
including one with generous headroom. A gate that is wrong gets muted rather than fixed -
this repository has already retracted `check_money.py` twice, in both directions, for
exactly that. Shipping the flaky version would have been worse than shipping nothing,
because it would have looked like coverage.

The replacement asserts on a counter the rate limiter already records:

```go
// internal/ratelimit.go:130
if needsWait {
    incrCounter(ctx, l.metrics, MetricRateLimitWaits, 1)
    observeHistogram(ctx, l.metrics, MetricRateLimitWaitMS, ...)
}
```

`InMemoryMetrics` and its `Snapshot().Counters[SeriesKey(...)]` reader already exist and
are already used by `pkg/ibkr/metrics_test.go`, so the test needed no new machinery at
all - I had planned a ~10-line sink before checking.

## Two things the design got wrong, both caught by running it

**The vacuity guard would have read zero.** I planned to assert
`Counters[SeriesKey(MetricHTTPRequests)] == 4`. But `MetricHTTPRequests` is recorded with
`method`, `path` and `status` attributes, so its series keys look like
`ibkr.http.requests|method=GET,path=/v1/api/iserver/accounts,status=200` and a bare key
matches nothing. The guard would have failed for the wrong reason, or worse, been
"fixed" by loosening it until it passed. It now sums across attribute-bearing keys and
separately asserts each of the four paths was hit, which is a stronger guard than the one
originally planned.

**The case I said caught the fix does not.** I told the user the "opted out" case was
"the v1.1.26 fix at the level that matters". The mutation run disproved it: with
`WithAuthRateLimit(50, 10)` the burst of 10 absorbs both auth calls, so that case still
passed with the defect reintroduced. The **default** case is the one that catches it -
`ibkr.ratelimit.waits` goes to 2. The cases are reordered so the regression-catching one
runs first, and the comment now says explicitly that the opted-out case does not catch
the regression, so nobody later "simplifies" by deleting the default case.

## Mutation result

Putting `/v1/api/logout` back into `isAuthPath`:

```
session_ratelimit_test.go:91: rate-limit waits = 2; want 1
--- FAIL: TestSessionLifecycle_RateLimitWaits/default_auth_pacing
```

Source restored, and the mutation re-verified after the comment and ordering edits so
the pass was not an artefact of the changes that followed it.

## What this does and does not buy

It catches a pacing regression in the session lifecycle. It does **not** cover the
opted-out configuration regressing, and it does not exercise `newClientFromArgs` - the
test builds the client directly, so it covers the SDK, not the CLI's argument wiring. A
CLI-shaped variant would couple a unit-level test to the command layer for little gain,
and the CLI's own subcommand tests already cover that wiring.

## No CI step

Deliberately. The test lives in the normal `go test ./...` sweep that CI already runs. A
gate that exists in only one CI job is a gate someone eventually drops, and this
repository's checker history says a gate with friction gets ignored rather than fixed.
