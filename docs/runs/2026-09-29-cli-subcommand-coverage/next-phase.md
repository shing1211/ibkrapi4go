# Next phase

## Needs a decision

- **Every CLI command costs ~2.0s of fixed latency.** Measured against a mock gateway
  that answers instantly: `Session.Initialize` 1.007s, `cli.Close` 995ms. The mock's
  recorder shows 4 requests and the auth poll succeeding first try, so it is *not* the
  one-second poll penalty fixed in `internal/session.go`; it is in the tickle-start and
  logout paths. This is the largest user-visible defect found so far and it is
  unexplained. Start with `internal/session.go`'s `startTickle` and the `Close`/logout
  path, both of which are on the happy path of every single command.
- **`--rest` is inert and now says so.** `cfg.RestGatewayURL` is parsed, stored, printed
  and settable, and `pkg/ibkr` has no option to receive it. Either add
  `WithRestGatewayURL` to the SDK or remove the flag, the config field and the
  `ibkr config` output. Leaving it as a commented dead assignment is honest but not
  finished.
- **`internal/fake` - delete or rewire.** Unchanged. The in-package `testClock` already
  covers what the fake would have been for, and it is still on the `ALLOWED` list.
- **The coverage floor** has moved 58 -> 60 -> 62 across five releases while coverage went
  63.5% -> 69.5%. Pin it, or commit to ratcheting per batch. Deferred for seven runs.
- **`parseGlobalFlags` returns `cmdIdx` one past** the first non-flag, contradicting its
  doc comment and its test's comment. No production caller uses it, so it can be fixed by
  either changing the return or changing the two comments. Cheap, and it is the kind of
  mismatch that misleads the next reader the same way `TestErrorBudget_EvictsOldEntries` did.
- **`portfolio ledger` emits Go field names** while every other JSON command emits
  lowerCamelCase, because `LedgerCurrency` carries no json tags. A one-line-per-field fix,
  but it changes output that a script may already parse.

## Blocked on a live account

- **D15** - `listTaxDocumentsAvailable` with `year` omitted, against a real gateway.
- **`submitModelPortfolioOrder` collision** - FA-enabled paper account.

## Available next, no credentials needed

- **`cmd/ibkr-mock-gateway` at 0%.** A thin `main` wrapper over `internal/mockgateway`,
  which is at 89.4%. The cmd/ibkr harness in this run shows the pattern: build the
  package once and mount its handler.
- **Remaining `cmd/ibkr` gap** - the 24% still uncovered is mostly `completion.go`
  (207 lines) and the flag-combination paths the new table-driven tests do not enumerate.
- **A config file can no longer mask a broken flag path.** Worth remembering when writing
  the next harness: this run's first version put the gateway URL in the config, which
  worked, and would have hidden the fact that no global flag was honoured at all.

## Process notes carried forward

- **Check a backlog item against the tree before acting on it.** Two of five items
  inherited into `v1.1.24` were wrong: one named a file that does not exist and two
  functions that are not defined, and one called for a seam that had shipped a year
  earlier. The wrong one changed the shape of the task from architectural to bounded.
- **A test that cannot fail is worse than no test.** The first test written for the
  `--account` fix hardcoded its expected value inside the fake it was asserting against.
- **When a test fails, ask whether the test is wrong before the code is.** Two of the
  failures in this run were bad assertions: a JSON command that does not print a ticker
  column, and a struct with no json tags that encodes with Go field names.
- **Prove a claim by running it, not by reading it.** The global-flag bug was visible in
  `parseGlobalFlags` within a minute of finding it, but the first probe - a throwaway test
  printing the three invocation shapes - turned it into a measured table in one run.
- **Leave a comment where a reader will hit the gap.** `--rest` is inert; the comment at
  the omission says why and what the real fix is, instead of leaving a plausible-looking
  assignment that reads as working.
