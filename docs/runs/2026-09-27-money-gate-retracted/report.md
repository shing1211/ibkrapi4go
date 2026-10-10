# Report: a recommendation withdrawn, and 7 fields closed

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `22051aa` (`v1.1.18`)
- **Outcome**: complete, uncommitted pending approval

## The recommendation being tested

The previous run recommended extending `check_money.py` with a rule that fails the
build when a money assertion uses a value a `float32` renders unchanged. The
argument was prevention: that blind spot hid two real money bugs in this repository,
and a rule would have caught both at authoring time.

It was a reasonable recommendation. The implementation is not available cheaply, and
this run found that out before shipping it.

## Formulation 1: lint the literals

Rule: in a test file, a decimal literal on a line that references money and compares
it must change under a `float32`.

It flags **27 literals**, and the sample shows why most are wrong:

```go
if sum.TotalCashValue != "100.25" {                              // flagged 100.25
if positions[0].UnrealizedPnL != "49.875" {                       // flagged 49.875
if *first.TransferPrice != "150.25" {                             // flagged 150.25
```

All three are string comparisons against `string`-typed public fields. A `float32`
regression there is a **compile error**, not a silent round: the type does the work,
and the literal's float32-representability is irrelevant. The rule is measuring the
wrong thing - it sees a money name and a decimal, and cannot tell whether a float is
involved at all.

It is also **not catching the target**. The one real case, `0.007` in the
tax-voucher precision fixture, sits on a fixture line rather than a comparison, so a
comparison-scoped rule never examines it. Too noisy, and blind where it matters.

## Formulation 2: key off the type

Rule: every monetary `json.Number` field in a `*Raw` decode struct must have a test
asserting it with a `float32`-changing value.

This is the correct invariant. There are **25 such fields** in `accountSummaryRaw`,
`positionRaw` and `ledgerRaw`, and they are precisely the ones where a `float32`
regression would round silently.

The invariant is sound; the implementation is not. Answering "does this field have a
lossy-value assertion" requires following `const` declarations and table-driven
loops. Scanned line by line, the prototype reported **16 fields as uncovered that
are in fact covered** - `SMA`, `AvailableFunds`, `EquityWithLoanValue` and others are
asserted in a table where the literal is a named `const` declared elsewhere. A gate
that reports 16 false negatives while appearing rigorous is worse than useless.

## Why nothing shipped

A lint with 27 false positives teaches a team to ignore it. A lint with false
negatives teaches a team to trust it. Both failure modes are more expensive than the
recurrence they prevent, and one of them is invisible. `check_money.py` is
unmodified.

The honest conclusion is that the rule needs to resolve the *type* of the field under
assertion, not the literal beside it - which means following the `toPublic` mappings
and the raw struct definitions. That is a small analysis, not a lint.

## What shipped instead

The gap itself, closed: 7 of the 28 monetary `json.Number` fields across the three
responses had no `float32`-changing assertion and now do - `balance`,
`excessLiquidity`, `initialMargin`, `regTLoan`, `regTMargin`, `securitiesGVP` and
the ledger's `stockOptionMarketValue`.

All seven are mutation-verified individually, each confirmed to fail on the
assertion:

| Field | Reported under a `float32` decode |
|---|---|
| `balance` | `16777218` (want `16777217.89`) |
| `excessLiquidity` | `12345679` (want `12345678.91`) |
| `initialMargin` | `33554432` (want `33554432.55`) |
| `regTLoan` | `33554432` (want `33554432.55`) |
| `regTMargin` | `12345679` (want `12345678.91`) |
| `securitiesGVP` | `33554432` (want `33554432.55`) |
| `stockoptionmarketvalue` | `12345.679` (want `12345.6789`) |

`money_precision_test.go` is now exhaustive over the monetary `json.Number` fields
of these three responses, and a comment says so - so a field added later without a
lossy-value assertion is visibly out of place rather than silently missing. That
explicit list is the closest sound thing to a gate available without real analysis.

## Two process notes

- **My PowerShell debugging attempt corrupted `account.go`** and I restored it with
  `git checkout`. The first mutation script then failed again by rewriting
  `r.Balance.String()` into `r.Balancef32String(...)` - a string overlap, not a test
  problem. The fix was to stop doing import surgery and put the mutation helper in a
  temporary file. Three tooling failures before the first meaningful signal, all
  recorded because the same mistakes are available to the next attempt.
- **A recommendation is a hypothesis like any other.** Last turn's gate was proposed
  confidently and was wrong on inspection. Two of my own premises in this run were
  also wrong before measurement: that string-typed assertions were vacuous, and that
  the type-keyed rule was implementable as a scan.

## Verification

All gates green: `gofmt -l`, `go build ./...`, `go vet ./...`, `go test ./...`,
`go test -race`, `golangci-lint run` and `--tests=false`, `check_money.py`,
`check_design`, `check_links.py`, `check_i18n.py`. Coverage 66.3% against 62%.
Working tree contains only the test file and untracked scratch, restored after each
mutation.

## Production code

None. One test file, the changelog, and run artifacts.
