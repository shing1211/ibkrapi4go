# Next phase

## Needs a decision

- **A time-based budget is still not a thing.** The window is a rolling **count** over
  the last `size` outcomes, and now recovers as documented. "5 failures in 60s" is
  still not expressible - a caller approximates it by choosing `size` to cover the
  period of interest, which depends on request rate and is therefore a guess. This is
  a new public option with new semantics, not a bug fix, and it wants a brainstorm and
  an ADR before code. Releasing it would be a minor version.
- **A loose `size`/`budget` ratio is a footgun.** With `budget=3, size=100` the budget
  stays satisfied for ~97 further outcomes, so the breaker keeps re-opening and the
  consecutive threshold still looks broken. The behaviour is correct and now tested, but
  `WithCircuitBreakerBudget` could reject or warn on a ratio that is unlikely to be
  intended. Left alone deliberately: guessing at intent is what made the withdrawn
  `check_money.py` gate unusable.
- **`internal/fake` - delete or rewire.** Unchanged from the last run. The in-package
  `testClock` already covers what the fake would have been for.
- **The coverage floor** has moved 58 -> 60 -> 62 across four releases while coverage
  went 63.5% -> 66.6%. Pin it, or commit to ratcheting per batch. Deferred for six runs.

## Blocked on a live account

- **D15** - `listTaxDocumentsAvailable` with `year` omitted, against a real gateway.
- **`submitModelPortfolioOrder` collision** - FA-enabled paper account.

## Available next, no credentials needed

- **`stream.go`'s remaining gap** - `serveWS` branches and the `handleSubscribe` family.
- **`cmd/ibkr` at 33%** - the largest number, and a design change: every non-help path
  builds a live client, so it needs a client-factory seam first.

## Process notes carried forward

- **Prove the behaviour before reasoning about the source.** Three runs in a row
  described this defect by reading `errorBudget` and got the cause wrong the same way.
  A 15-line probe settled in one run what three readings did not.
- **A test named for the opposite of what it asserts misleads every reader.**
  `TestErrorBudget_EvictsOldEntries` asserted that entries were *not* evicted, and that
  mismatch is why the time-window question was answered wrongly twice. It is renamed.
- **Overstate nothing, in either direction.** The probe proved the threshold drops to 1;
  calling that "permanent" was wrong, because a loose ratio legitimately keeps the
  budget spent for many more outcomes. Both halves are now in the test comments.
- **Check the test's parameters against the contract before believing its failure.** The
  first regression test failed after a correct fix because `size=5, budget=3` genuinely
  should trip. A test that fails against correct code is usually the test's fault.
- **An unobservable test should say so.** The nil-clock test cannot detect the zero-time
  fallback because the observable behaviour is identical; its doc comment says that
  rather than implying coverage.
