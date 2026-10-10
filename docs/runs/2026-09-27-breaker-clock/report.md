# Report: a package orphaned by a design change, and the seam it was written for

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `2bc985c` (`v1.1.21`)
- **Outcome**: complete, uncommitted pending approval

## The claim that turned out to be wrong

Two consecutive runs reported the offline backlog empty. That claim came from
per-package coverage, where `internal/fake` showed 0.0% and was dismissed as "seven
one-line setters, not worth it" - a description belonging to `options.go`, not to
this package.

A 0% package is not a coverage gap. It is either dead code or an untested
dependency, and both are findings.

## `internal/fake` is orphaned, not merely unused

- 7 files, **no importer anywhere in the module**, no test, no design-doc reference.
  The only mention outside the source is a historical changelog line.
- `internal/` is not importable from outside the module, so no external user can
  reach it either. It cannot be a published testing aid.
- It predates `internal/mockgateway` and was superseded by it.

But the interesting part is *why* it is dead:

```go
// internal/clock.go
type Clock struct {
	nowFunc    func() time.Time      // unexported
	tickerFunc func(time.Duration) *Ticker
	afterFunc  func(time.Duration) <-chan time.Time
}
// no exported constructor sets these
```

`Breaker.SetClock(c *Clock)` takes a concrete `*internal.Clock`. Only code **inside**
`internal` can build one with a custom `nowFunc`. And `internal/fake.Clock` is a
**different type** whose doc comment says it "implements internal.Clock" - but
`internal.Clock` is a struct, not an interface, so there is nothing to implement and
nothing that can be passed to `SetClock`.

The seam moved from an interface to a struct with unexported fields, and the fake
was never updated. That is what orphaned it: not neglect, a design change.

## What ships

**Deterministic breaker tests**, using the seam as intended. The existing tests in
this package wait on wall-clock time - `time.Sleep(60 * time.Millisecond)` against a
40 ms cooldown - which is slower and racy under load, and is precisely what
`SetClock` exists to remove. The new tests drive an in-package clock:

- the full closed → open → half-open → closed cycle, with both sides of the
  boundary pinned: still open one nanosecond before the cooldown elapses, probe
  allowed exactly at it;
- a failed probe restarts the cooldown, so the circuit reopens immediately;
- the error budget can open the circuit on its own, below the consecutive threshold.

**The error budget's real semantics**, which contradict its documentation. It is
described as a "sliding window", and `evict` as removing "entries that have slid out
of the window" - but `evict` is passed `now` and **never reads it**. It bounds the
slice by count only. So the budget is a count of recent failures with no time
component at all.

That matters operationally: configuring "5 failures in 60s" does not yield 5 failures
in 60s, it yields 5 failures out of the last N recorded, however far apart they were.

The existing test is *named* `TestErrorBudget_EvictsOldEntries` while its own body
asserts the opposite - "evict should not remove since size=5". That mismatch is
probably why two audits of this file read as though a time window existed. It is left
in place; renaming it is a separate call.

Correcting the behaviour is a real change to how a trading client trips its circuit
breaker, and AGENTS.md requires an ADR for that. This run pins the present behaviour
and records the discrepancy.

## Mutation verification

Four mutations, all caught on the assertion:

| Mutation | Caught by |
|---|---|
| cooldown boundary becomes exclusive | `Allow at cooldown = circuit breaker open; want the probe allowed` |
| a failed probe does not restart the cooldown | `Allow after a failed probe` |
| the budget trips one failure early | `record #2 an hour apart = true; want false` |
| `evict` no longer bounds by count | `window holds 20 entries after 20 records; want 4` |

**One could not be caught, and the test says so.** Making a nil clock fall back to
the zero time instead of the real clock is invisible: `openUntil` becomes
`zeroTime + cooldown`, which is still in the future relative to `zeroTime`, so the
breaker refuses requests exactly as before. Distinguishing them needs an hour of real
time to pass - the thing `SetClock` exists to avoid. `TestBreaker_SetClockNilFallsBackToRealTime`
is kept because it still pins that a nil clock does not panic and keeps the circuit
open, and its doc comment states plainly that it cannot detect that regression.

## The `internal/fake` decision

Not taken here. The changelog records the package as an intended v1.0.0 feature, so
removing it is a judgement call rather than a cleanup, and the two options are not
equivalent:

- **Delete it.** It is unreachable, unused, and incompatible with the current seam.
  ~7.6 KB of the shipped module goes away, and the historical changelog line stays
  accurate as history.
- **Rewire it.** Give `internal.Clock` an exported constructor, or make it an
  interface, and `fake.Clock` becomes usable by `internal`'s own tests. That is a
  design change with its own trade-offs, and on the evidence here the in-package
  `testClock` already does the job with less machinery.

Either way, `internal/fake` being at 0% and invisible to the linter is itself a gap:
`unused` does not report it, because every identifier it declares is exported and
`internal/fake` is a package rather than a symbol.

## Verification

All gates green: `gofmt -l`, `go build ./...`, `go vet ./...`, `go test ./...`,
`go test -race`, `golangci-lint run` and `--tests=false`, `check_money.py`. Coverage
66.5% -> **66.6%**.

One transient build failure during the sweep was the known temp-executable collision
in this environment, and passed on retry.

## Production code

None. Two test files, the changelog, and run artifacts.
