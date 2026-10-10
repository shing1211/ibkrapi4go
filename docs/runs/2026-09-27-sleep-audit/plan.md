# Plan: audit every fixed wait in the test suite

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `cc343ab` (`v1.1.11`)

## Objective

Audit the 21 `time.Sleep` sites flagged in the previous run's `next-phase.md`.
The v1.1.10 fix gave a reusable rule — a wait for a *guaranteed event* is a
defect, a wait for *time to pass* is not — and this run applies it to every site
rather than to the one that happened to fail.

## Approach

1. **Classify, don't count.** Reading each site against the rule produces a verdict
   per site, not a number. A test that is merely slow is not the target; one whose
   assertion can only fail on a slow machine is.
2. **Widen the search.** `time.Sleep` was the starting point, not the boundary. A
   `select` whose only arms are `time.After(N)` and `ctx.Done()` is the same defect
   in different clothing, so the audit looks for that shape too.
3. **Prefer driving to waiting.** Where the code under test can be stepped
   directly, do that; where a real loop must run, wait on a signal it emits.
4. **Prove each new assertion can fail.** A converted test that cannot fail is the
   same defect wearing better clothes.

## What the approach found that a tally would not

- One test was **vacuous**, not flaky: it compared two reads of a value that could
  not differ, so it could never have caught the bug it names.
- The slow tests pointed at a **production defect**: a one-second sleep before the
  first poll in `Session.Initialize`, taxing every session start.

Both were invisible to "the tests take 2 seconds".

## Tasks

| # | Task | Outcome |
|---|------|---------|
| 1 | Classify 21 `time.Sleep` sites | 6 defects, 15 legitimate |
| 2 | Find `time.After`-only selects | 5 further defects |
| 3 | Convert session tests | drive `tickleRound`; manual clock for the loop tests |
| 4 | Convert WebSocket tests | wait on delivery signals |
| 5 | Make the vacuous assertion real | mutation-proven to catch a double-start |
| 6 | Fix `Initialize` poll order | 1s off every session start |
| 7 | Remove orphaned `Clock.Sleep` | with its `sleepFunc` hook |
| 8 | Re-verify coverage | a 55.0% reading proved to be an artifact of measuring mid-edit |

## Ordering note

The production fix in task 6 came *after* the test conversions, not before. The
tests had to stop sleeping before the wasted second became visible as a latency
cost rather than as test runtime.
