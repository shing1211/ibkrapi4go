# Report: 39 money fields that had no test proving they keep their digits

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `18af8d5` (`v1.1.17`)
- **Outcome**: complete, uncommitted pending approval

## The class of defect

An assertion comparing a money value against a literal a `float32` renders
unchanged cannot detect a `float32` regression. It passes either way, so it looks
like a precision test while proving nothing.

This has produced two real bugs in this repository. v1.1.9 found a `float32` decode
in the tax-voucher response. v1.1.13 found the request direction on deposit,
withdraw and internal transfer. The v1.1.13 commit message states the pattern
plainly: the three existing assertions used 250.75, 500.25 and 1000.50, "all of
which are exactly representable as a float32, so they passed whether or not the
amount had been rounded".

## The audit

53 decimal literals across the four money-bearing test files. Checking each for
whether `float32` changes it:

| | count |
|---|---|
| changes under `float32` - deliberate | 2 |
| renders unchanged - undetectable | 38 |

The 38 are not automatically a problem, and most are not:

- **Banking and transfer money is `string`-typed** (`tx.Amount`, `TransferPrice`),
  and the wire carries quoted strings. A `float32` regression there is a compile
  error, not silent rounding. The type does the work.
- **`1234.5600` on the account summary** does change under `float32` (1234.56), so
  the existing endtoend assertion is not vacuous.

The genuine gap is the `json.Number` response fields. `accountSummaryRaw` declares
15 monetary fields as `json.Number` with the comment *"decodes monetary fields as
json.Number to preserve decimal precision"*; `portfolio.go` declares 17 more.
`toPublic` calls `.String()` on each, so a caller gets the gateway's exact digits -
or, under a `float32` regression, a rounded string.

**39 money fields with no test using a value `float32` would corrupt.** The
tax-voucher path has exactly such a test, added in v1.1.9. These three did not.

## The tests

`money_precision_test.go` covers all three responses with values above 2^24, where
a 24-bit mantissa rounds to a whole unit:

- **Account summary** - 8 top-level fields plus both nested `cashBalances` values.
  The nested pair matters separately: a slice element that regressed would leave
  every top-level field correct.
- **Positions** - `avgCost`, `avgPrice`, `mktPrice`, `mktValue`, `realizedPnl`,
  `unrealizedPnl`.
- **Ledger** - the currency-keyed map and six values within it.

## A subtlety my own guard caught

The first draft used `0.007` as the fractional case, reasoning that it sits below
`float32` resolution. The guard in the test rejected it:

> float32 renders 0.007 back as "0.007", so no string assertion could detect a
> float32 regression; this value does not prove the precision fix

`0.007` *is* lossy in binary - the nearest `float32` is 0.00700000022 - but Go's
`strconv.FormatFloat(..., -1, 32)` produces the shortest string that round-trips
to that `float32`, and that string is `"0.007"`. The value is corrupted and the
text is identical, so string comparison is blind to it.

The helper therefore checks **detectability** rather than exact
representability, and rejects any value that survives the round trip unchanged.
`12345.6789` replaced `0.007` - its float32 renders as `12345.6787`.

This also means the existing tax-voucher test's `fee` assertion carries no proof,
even though it sits next to two assertions that do. Worth revisiting; not changed
here because it is not wrong, only weaker than it looks.

## Checked, and recorded as not-a-bug

The mock gateway sends the account summary's monetary fields as **quoted JSON
strings**, while the struct declares them `json.Number` - which looks like it should
fail to decode. A probe confirmed it does not: `encoding/json` accepts a quoted
string into `json.Number` and preserves the digits, so `"1234.5600"` arrives as
`"1234.5600"`.

My recollection of the decoder's rules was wrong and the probe corrected it.
Recorded so the question is not raised again as a suspected bug.

## Mutation verification

Five mutations, each emulating a `float32` decode in the production conversion.
All five caught **on the assertion**, verified individually so that none was
passing for a build error:

| Mutation | Reported damage |
|---|---|
| account summary `SMA` | `SMA = "12345679"; want "12345678.91"` |
| account summary nested `cashBalances[0].balance` | `"33554432"; want "33554432.55"` |
| positions `avgCost` | `avgCost = "16777218"; want "16777217.89"` |
| positions `unrealizedPnl` | `unrealizedPnl = "67108864"; want "67108865.13"` |
| ledger `cashbalance` | `cashbalance = "16777218"; want "16777217.89"` |

Note that `16777218` is 2^24 rounded to the nearest even representable value - the
exact failure the test exists to catch, and one that reads as plausible at a glance.

## Verification

All gates green: `gofmt -l`, `go build ./...`, `go vet ./...`, `go test ./...`,
`go test -race`, `golangci-lint run` and `--tests=false`, `check_money.py`,
`check_design`, `check_links.py`, `check_i18n.py`, `check_spec_version.py`.
Coverage 66.3% against a 62% floor. The temporary probe file was removed.

`codegen-verify` not re-run: no spec or generated-code change.

## Production code

None. One new test file, the changelog, and run artifacts. The account summary,
positions and ledger decode exactly as they did - these tests document behaviour
that was already correct and make it impossible to regress silently.
