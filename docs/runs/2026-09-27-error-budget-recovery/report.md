# Report: a budget that could be spent once and never recovered

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `f95f4c4` (`v1.1.23`)
- **Outcome**: complete

## The defect

`errorBudget.window` was a `[]time.Time` that was appended to on every failure and
**never cleared**. The success path in `Record` reset `consecutive`, `openUntil` and
`probing` - and not the budget:

```go
// internal/breaker.go, before
b.consecutive = 0
b.openUntil = time.Time{}
b.probing = false
// no budget reset
```

So once `len(window) >= budget` was true it stayed true for the life of the client.
The window only shrank past `size`, which at the recommended `size=100, budget=3` means
the budget stayed spent until roughly 97 *further* outcomes.

The observable effect is that after the budget trips once, the configured consecutive
threshold stops meaning anything: one more failure re-opens the circuit.

## What the probe showed

Throwaway test, no production code touched. `threshold=10`, `budget=3`, `size=100`:

| Step | Window | State |
|------|--------|-------|
| 3 failures | 3 | open (budget exhausted) |
| cooldown elapses, probe succeeds | 3 | closed |
| **1 more failure** | 4 | **open** |

So the effective threshold after recovery was **1**, not 10. `size=3` and `size=10000`
behaved identically at three failures, which is the tell that `size` was only bounding
memory, not defining a window.

## Two corrections to my own reasoning

**The first regression test was wrong.** I wrote it with `size=5, budget=3` and asserted
that one success clears the budget. It does not: the window holds 4 failures in 4
outcomes, which legitimately trips. The fix was right and the test was wrong, which is
the only reason the second version uses `size == budget`.

**The severity was overstated.** The spike concluded the threshold was "permanently"
reduced to 1. After the fix, recovery genuinely happens - but a loose ratio recovers
slowly, and with `budget=3, size=100` the budget legitimately stays satisfied for ~97
more outcomes, so a breaker configured that way still keeps re-opening. That is the
configured contract, not the defect returning. It is now written into the test comments
so the next reader does not re-report it.

## The fix

The window holds **outcomes**, not failures:

```go
// before
window []time.Time
func (eb *errorBudget) record(now time.Time) bool { ...; return len(eb.window) >= eb.budget }
// after
window []bool // true for a failure, most recent last
func (eb *errorBudget) record(failed bool) bool { ...; return failures >= eb.budget }
```

Three parts, each mutation-verified:

1. `Record` records successes too (`b.budget.record(false)`), which is what evicts old
   failures.
2. The window stays bounded, dropping the oldest outcome.
3. The trip condition counts failures within the window, not entries in it.

The dead `now` parameter went with it - `evict(now)` never read it, which is the direct
reason the type comment could say "sliding" while nothing ever slid.

A failure is now recorded even when the consecutive threshold already tripped. Skipping
it made the window understate how many failures actually happened.

## Tests that encoded the defect

- `TestErrorBudget_EvictsOldEntries` asserted that entries are **not** removed. The name
  is the opposite of the behaviour, and the previous run's `next-phase.md` already
  flagged that mismatch as the likely reason the time-window question was answered
  wrongly twice. Renamed to `TestErrorBudget_WindowIsBoundedAndKeepsTheNewest` and
  rewritten to assert what it says.
- `TestErrorBudget_CountsFailuresNotElapsedTime` pinned "failures an hour apart still
  exhaust the budget". That is true after the fix too, but as a headline it described
  the symptom. Replaced by tests that pin recovery directly.
- `TestBreaker_SetErrorBudgetTripsTheBreaker` was the useful one: the full suite caught
  it failing, because `size=3` with interleaved successes genuinely no longer trips. Its
  intent - the budget can open the circuit below the consecutive threshold - is valid
  and is preserved with `size=5`, where all three failures really are inside the
  window.

## Verification

The regression test was written first and seen **fail** on the unfixed code.

Mutation results, each of which must be caught:

| Mutation | Result |
|----------|--------|
| Successes never enter the window (the original defect) | caught |
| Window never bounded, so old failures stay forever | caught |
| Counts outcomes instead of failures (old `len()` logic) | caught |

Gates: gofmt, build, vet, `go test ./... -count=1`, race, money, internal-refs, design,
links, i18n. Coverage **66.6%** against the 62% CI floor (66.7% before - the
difference is the deleted probe's tests, which were the defect's own documentation).

## Versioning note

The previous run's `next-phase.md` said correcting this "changes when a trading client
trips its circuit breaker, which is exactly the kind of decision AGENTS.md says needs an
ADR and a minor version". That was that run's judgement; AGENTS.md requires an ADR for
dependencies and for retry policy, and sets no rule here. It is released as a patch
because the public contract text is unchanged - it already promised "within the last
size outcomes" - no signature moved, and only opt-in callers are affected. The
behaviour change is recorded below rather than hidden, since that judgement is arguable
and a reader should be able to disagree with it.
