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

- **`cmd/ibkr` at 33.0%** - the largest real number, and **not** the design change
  carried forward from `breaker-clock`. That note said every non-help path builds a
  live client and "needs a client-factory seam first". The seam exists: `env.go` has
  `newClient func() (*ibkr.Client, error)`, doc-commented as "the seam a test replaces
  to point the command at a mock gateway", and all **9** call sites go through
  `e.newClient()` - `accounts.go`, `orders.go` (x3), `portfolio.go` (x3), `positions.go`
  and `stream.go`. `newClientFromArgs` already honours `--gateway`, so a test can aim it
  at an `httptest` server.

  The real gap is that **no test replaces it** - zero references to `newClient` in
  `cmd/ibkr/*_test.go`. Everything past client construction in those 6 files is
  therefore unexecuted. The blocker is smaller than the previous run recorded: build the
  tests, not the seam.
- **`internal/mockgateway` is ready to back them** at 89.4%, and `internal` is at
  80.4%, so the pieces under `cmd/ibkr` are the well-covered ones.
- **The `stream.go` gap recorded by `breaker-clock` does not exist.** There is no
  `internal/stream.go`, and `serveWS` and `handleSubscribe` are not defined anywhere in
  `internal/`, `pkg/ibkr/` or `cmd/` - the name matches neither the WebSocket code in
  `internal/ws.go` (737 lines, with 908 + 528 lines of existing tests beside it) nor
  `cmd/ibkr/stream.go`, which is the 98-line `stream` CLI command. Superseded, not
  carried forward.

## Process notes carried forward

- **Verify a claim against the tree before carrying it into the next backlog.** Two of
  the five items inherited from `breaker-clock` were wrong on arrival: one pointed at a
  file that does not exist and named two functions that do not exist, and one called for
  a seam that had existed since 1.1.14. A backlog is a list of assumptions about the
  current state, and an unverified one is worse than a short one - it makes the next run
  spend its budget re-deriving what is true. Cheap to check, and this run's own output
  was the second occurrence of the failure.

  The correction to the item above is itself an instance. Its first draft said "all 5
  call sites" and listed five files, read off a truncated search that stopped at twelve
  matches. The real count is 9 across 6 files. A wrong number in a backlog is not a
  smaller error than a wrong claim about a file's existence - both are guesses that
  read like measurements, and a reader cannot tell them apart.
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
