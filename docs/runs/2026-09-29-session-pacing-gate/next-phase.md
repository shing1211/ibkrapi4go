# Next phase

Refreshed after v1.1.29 and the two commits that followed it. Six items that had
been carried across runs are now closed; the ones that remain are the ones that
actually need a decision or a credential.

## Resolved since v1.1.28

- **`--rest` is inert** → removed in v1.1.29. The flag, the `rest` config key, the
  `config show` line, the `config set` case and the help text are gone. A persisted
  `rest_gateway_url` still loads, because `loadConfig` ignores unknown fields.
- **`internal/fake` — delete or leave** → deleted in v1.1.29, seven files. `ALLOWED`
  in `check_internal_refs.py` is now empty, so the exemption list is a live decision
  again rather than a standing exception.
- **`LedgerCurrency` has no json tags** → fixed in `9d110b4` (struct) and v1.1.29
  (the CLI assertion that encoded the old output).
- **`check_design` verified 5 manager counts out of 15** → fixed in `7fb2179`. Every
  manager in `pkg/ibkr` is now checked against a one-row-per-manager doc table, and
  a missing row fails the build. Five wrong numbers were corrected and
  `SessionManager`, which had no row at all, was added.
- **`ForecastManager` has no methods** → the five wrappers landed in `83a20cc`.
  `ForecastCategories` returns `json.RawMessage` on purpose: the category tree is a
  map and the generated type flattens it, so a model built from it would describe a
  payload IBKR does not send.
- **`parseGlobalFlags` returns `cmdIdx` one past the first non-flag, against its own
  comments** → the behaviour was right and one comment was wrong. `TestParseGlobalFlags_
  StopsAtFirstNonFlag` called the value "the index of orders" when `orders` is at
  index 1 and the value is 2. Corrected, and the function doc now says why the
  return exists and why stopping at the first non-flag is load-bearing.

## Needs a decision

- **The CI `lint & security` job is red on `main`, and has been.** `golangci-lint run`
  reports six `nolintlint` findings, all unused `//nolint:gosec` directives, in
  `internal/mockgateway/oauth.go`, `internal/mockgateway/session.go`,
  `pkg/ibkr/rest_accounts_test.go`, and `scripts/check_design/red_test.go`. The
  directives guard gosec's file-permission rule, which the current config evidently
  does not enable. Either re-enable the rule so the directives are needed, or delete
  six of them. **Not fixed in `7fb2179` on purpose**: deleting a directive that CI's
  gosec version does need would turn a lint failure into a build failure, and which
  half is right depends on whether the project wants the rule enforced.
- **Coverage floor policy, deferred across nine runs.** This one is answerable from
  measurement rather than opinion. CI enforces 62% against the cross-package
  aggregate, which measures **72.0%** — so the floor is pinned, green, and has ten
  points of headroom. `pkg/ibkr` on its own was 61.8% and is now **62.0%** after the
  forecast tests, the first time that package has cleared 62%. The live question is
  only whether `pkg/ibkr` deserves its own floor, and whether a threshold with ten
  points of headroom pins anything.

## Needs a live account

- **Does IBKR actually rate-limit auth endpoints, and at what rate?** ADR 0018 is
  Accepted with this open. The 1 rps default is a client-side precaution; the spec is
  silent, the mock imposes no limit, and no ADR states it. If auth is not specially
  limited, the right change is to **drop** the bucket rather than raise its rate, and
  `TestLimiter_AuthPathsAreSlow` goes with it. This is the same trip that would unblock
  D15, so it is worth doing once rather than twice.
- **D15** — `listTaxDocumentsAvailable` with `year` omitted.
- **`submitModelPortfolioOrder` collision** — FA-enabled paper account.

## Process notes carried forward

- **Verify coverage numbers from a clean run.** `go test -cover` caches; stale numbers
  have misled this list more than once. Always `go clean -testcache` before reading a
  coverage number as a fact. (Re-confirmed here: the 61.8% reading was real, not stale.)
- **Mutation-test a gate, and mutation-test the tests.** Two gates in this repo passed
  for nine runs because they compared nothing, and one new test passed against the
  exact bug it targeted because Go formats floats with the shortest round-tripping
  representation — an assertion on `0.1` cannot tell text-preservation from a
  `float64`. Use a value the type cannot represent.
- **Check for an existing signal before adding machinery.**
- **A counter is not a clock.** A wall-clock gate is flaky and gets muted.
- **Verify which case catches the regression, not which case should.**
- **Check the metric's attributes before keying a snapshot.**
- **Re-run the mutation after editing the test.**
- **Assert on the flow, not the helper.**
- **Read coverage and lint scope before believing either number.** The coverage floor
  and the lint job measure different populations than the package they sound like
  they measure, and both were worth more than a re-run would have suggested.
