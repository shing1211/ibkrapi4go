# Next phase

## Needs a live account — gates the biggest open question

- **Does IBKR actually rate-limit auth endpoints, and at what rate?** ADR 0018 is
  Accepted with this open. The 1 rps default is a client-side precaution; the spec is
  silent, the mock imposes no limit, and no ADR states it. If auth is not specially
  limited, the right change is to **drop** the bucket rather than raise its rate, and
  `TestLimiter_AuthPathsAreSlow` goes with it. This is the same trip that would unblock
  D15, so it is worth doing once rather than twice.
- **D15** - `listTaxDocumentsAvailable` with `year` omitted.
- **`submitModelPortfolioOrder` collision** - FA-enabled paper account.

## Needs a decision

- **`--rest` is inert.** `cfg.RestGatewayURL` is parsed, stored, printed and settable,
  and `pkg/ibkr` has no option to receive it. Either add `WithRestGatewayURL` or remove
  the flag, the field and the `ibkr config` output. The v1.1.26 next-phase said this;
  no action has been taken.
- **`internal/fake` — delete or leave.** `testClock` in `internal/` covers what the fake
  was for. It's on `ALLOWED` in `check_internal_refs.py`. Unchanged for five runs.
  Decide and act.
- **Coverage floor policy deferred nine times.** Moved 58 → 60 → 62 across releases
  while actual coverage went 63.5% → 69.6%. Current `pkg/ibkr` is at 61.8% (cache-stale?
  verify with `go clean -testcache` before treating as below floor). Pin the floor or
  retire the rule.
- **`parseGlobalFlags` returns `cmdIdx` one past** the first non-flag, against its own
  doc comment and its test's comment. No production caller uses it. Fix the code or fix
  the comment — seven runs of noting it without acting.
- **`LedgerCurrency` has no json tags** on its `json.Number` fields, unlike every other
  JSON command in `portfolio.go`. CLI output emits Go field names
  (`NetLiquidationValue`) instead of wire names (`netliquidationvalue`).

## Available next, no credentials needed

*(All three resolved since v1.1.27 — no further action required.)*
- ~~`config` subcommand at 0%~~ → covered at 84.3% in `b3ef5e6`.
- ~~`cmd/ibkr-mock-gateway` at 0%~~ → covered at 90.4% in `b3ef5e6` / `281be72`.
- ~~`ibkrPrintln` and `truncate` at 66.7%~~ → both 100% after `b3ef5e6`.

## Process notes carried forward

- **Verify coverage numbers from a clean run.** `go test -cover` caches; stale numbers
  have misled this list twice. Always `go clean -testcache` before reading a coverage
  number as a fact.
- **Check for an existing signal before adding machinery.**
- **A counter is not a clock.** A wall-clock gate is flaky and gets muted.
- **Verify which case catches the regression, not which case should.** Mutation first.
- **Check the metric's attributes before keying a snapshot.**
- **Re-run the mutation after editing the test.**
- **Assert on the flow, not the helper.**
