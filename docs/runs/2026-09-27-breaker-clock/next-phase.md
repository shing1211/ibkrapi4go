# Next phase

## Needs a decision

- **`internal/fake` - delete or rewire.** Unreachable, unused, and incompatible with
  the seam it was written for: `internal.Clock` became a struct with unexported
  fields, so `fake.Clock` cannot be passed to `SetClock`. The changelog records it as
  an intended v1.0.0 feature, so this is a judgement call. On the evidence here the
  in-package `testClock` already covers what the fake would have been for.
- **`TestErrorBudget_EvictsOldEntries` is misnamed.** Its body asserts that evict does
  *not* remove old entries. The name is the opposite of the behaviour, which is
  likely why two audits misread the file. Renaming it is a one-line change.
- **The coverage floor** has moved 58 -> 60 -> 62 across three releases while coverage
  went 63.5% -> 66.6%. Pin it, or commit to ratcheting per batch. Deferred for five
  runs.

## Needs an ADR, not a test

- **The error budget has no time component.** `evict` is passed `now` and never reads
  it; the window is a count, not a duration. "5 failures in 60s" currently means "5
  failures out of the last N". Correcting it changes when a trading client trips its
  circuit breaker, which is exactly the kind of decision AGENTS.md says needs an ADR
  and a minor version. `TestErrorBudget_CountsFailuresNotElapsedTime` pins the
  present behaviour so the discrepancy is visible in the suite.

## Blocked on a live account

- **D15** - `listTaxDocumentsAvailable` with `year` omitted, against a real gateway.
- **`submitModelPortfolioOrder` collision** - FA-enabled paper account.

## Available next, no credentials needed

- **An unreferenced-`internal`-package gate.** `internal/fake` sat at 0% and
  invisible to `unused` for the module's whole life, because every identifier it
  declares is exported. A check that every package under `internal/` is imported
  somewhere in the module would catch that class directly, and unlike the withdrawn
  `check_money.py` gate it has no heuristics and no false positives: the rule is
  "each `internal/<pkg>` appears in at least one import". Worth writing whichever way
  the `internal/fake` decision goes.
- **`stream.go`'s remaining gap** - `serveWS` branches and the `handleSubscribe`
  family. Smaller, and the parts left are the protocol's odder shapes.
- **`cmd/ibkr` at 33%** - the largest number, and a design change: every non-help
  path builds a live client, so it needs a client-factory seam first.

## Process notes carried forward

- **Verify "the backlog is empty" before saying it twice.** It was asserted for two
  runs from a coverage table, and the one 0% package in that table had been
  misdescribed as belonging to a different file.
- **A package at 0% is a finding, not a gap.** Either dead code or an untested
  dependency; both need a decision.
- **A test named for the opposite of what it asserts will mislead every reader.**
  `TestErrorBudget_EvictsOldEntries` asserts that entries are *not* evicted, and that
  mismatch is why the time-window question was answered wrongly twice.
- **An unobservable test should say so.** The nil-clock test cannot detect the
  zero-time fallback because the observable behaviour is identical; its doc comment
  says that rather than implying coverage.
