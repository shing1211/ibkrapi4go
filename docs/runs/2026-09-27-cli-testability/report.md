# Report: CLI testability and the coverage floor

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `c09e01d` (`v1.1.13`)
- **Outcome**: complete, uncommitted pending approval

## What changed

`cmd/ibkr` no longer reads the process. An `env` struct carries argv, stdout,
stderr, and a client factory from `run` down; the six subcommands take it
explicitly. Direct `os` references now exist only in `main.go` and `env.go`.

The three `append(os.Args[:2], args[1:]...)` hand-offs are gone. That expression
was an aliasing write into the real argv, so it is removed rather than repaired.

`dispatch(e, cmd)` was split out of `run` so the routing table is testable
directly, and `newClient` is a field so a test can substitute a factory.

## The defect the refactor removed

`runOrders`, `runPortfolio`, and `runConfig` passed a subcommand's arguments down
by mutating `os.Args` in place. Consequences:

- Dispatching one command changed the arguments every later command saw.
- The test harness reads `os.Args`, so the corruption was observable as test
  breakage, not only as runtime misbehaviour.
- This is why these commands had no tests. It was not an untested-code gap so
  much as an untestable-code gap.

`TestOrders_ArgsAreNotLeakedBetweenInvocations` now asserts on `os.Args` itself,
before and after, across two invocations with different arguments. It is
mutation-verified: reinserting the original `append` into the dispatch makes it
fail with the mutation message.

An earlier draft of that test asserted on each `env`'s own argument copy. That
proved nothing - a copy is trivially unmodified, and the mutation harness caught
it only via an unrelated compile error. The assertion is on the process vector now
because that is the thing the bug damaged.

## A near-miss worth recording

The `gosec` G602 findings were fixed by replacing six flag loops with
`for i, a := range args`. That form compiles, reads cleanly, and keeps the `i++`
that skips a consumed value - but **assigning to a range variable does not advance
the iteration**. Each iteration reassigns `i`, so the parser stopped at the first
flag's value and silently dropped every later flag: `-gateway g -rest r positions`
would parse `gateway=g` and discard `-rest r` entirely.

`TestArgParsing_ConsecutiveFlagsBothApply` was written to catch exactly that, and
does: the mutation reports `rest=""` and command index `2` where `5` is required.
The loops are three-clause, with the bounds moved into `argAt` / `argValue`.

Worth noting the sequence: the mutation check is what turned a plausible-looking
refactor into a caught defect. Both mutations in this run were found by the
harness rather than by reading the diff.

## Structural fix rather than suppression

`gosec` reports `args[i]` inside a `switch` as G602 even when the loop condition
bounds `i` - it does not follow a bound held outside the indexing expression. A
`//nolint:gosec` silenced the finding in the normal profile and then made
`nolintlint` report the directive as unused in the `--tests=false` profile, where
`gosec` reported nothing at all. The two profiles disagree, so no single
annotation satisfies both.

Fixed instead by moving the bound adjacent to the index, inside `argAt` and
`argValue`, which is the form `gosec` does verify. Sixteen copies of
`if i+1 < len(args)` collapsed into two helpers, and both profiles are clean with
no suppression.

## Coverage

| | before | after |
|---|---|---|
| total | 62.0% | **63.5%** |
| CI floor | 58% | **60%** |

`cmd/ibkr` went from untested to 20 top-level tests. The floor was raised to 60%
rather than to the new 63.5%, to keep some headroom for future work.

Note the 58 -> 60 step is small on purpose. The figure is honest: the 1.1.12
audit removed several seconds of fixed `time.Sleep` waits that had been
incidentally keeping background goroutines running during the test, so the
earlier number over-stated what the suite really exercised.

## Two self-inflicted findings

- `(*env).realStdout` was written as a "fallback to the real process" escape hatch
  and never used. The `--tests=false` gate added in 1.1.13 caught it within
  minutes of being written.
- A test fixture used the typo `sumary` to exercise the unknown-subcommand path.
  `misspell` correctly flagged it, since a misspelling check has no way to know the
  string is deliberate. Replaced with `nope`.

## Verification

All gates green:

- `gofmt -l` clean, `go build ./...`, `go vet ./...`
- `go test ./...` and `go test -race ./pkg/ibkr/ ./internal/... ./cmd/ibkr/`
- `golangci-lint run` and `golangci-lint run --tests=false` - 0 issues each
- `check_money.py` (55 files), `check_design` (9/9 docs), `check_links.py`,
  `check_i18n.py` (6 languages), `check_spec_version.py`

`codegen-verify` was not re-run: the spec, `scripts/patch_spec.py`, and
`client/*.gen.go` were not touched by this run.

## Not done, and why

Both remain blocked on credentials, exactly as the previous run recorded:

- **D15 live verification** - calling `listTaxDocumentsAvailable` with `year`
  omitted needs a real account.
- **Mutating model-endpoint dispatch** - `submitModelPortfolioOrder` is
  deliberately unrouted, because its path collides with `submitNewOrder`
  (191 of 192 fixture ops are routed; the duplicate `getTradingSchedule` row is a
  fixture artifact). Confirming the collision is real needs an FA-enabled paper
  account.

WebSocket `contextcheck` stays excluded: `Subscription.Close(ctx)` is a breaking
API change, and the current signature is deliberate for a handle closed from a
`defer`.
