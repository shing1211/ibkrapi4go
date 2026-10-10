# Plan: audit money assertions for the flaw that hid two real bugs

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `18af8d5` (`v1.1.17`)

## Objective

Fifth request to "plan and implement remaining blocking items". The blocked list
has not moved and cannot move from here:

- D15 live verification - needs a real account
- confirming the `submitModelPortfolioOrder` collision - needs an FA paper account

The remaining offline coverage work is small and mostly mechanical (`options.go` is
seven one-line setters). Adding coverage for its own sake is a poor use of a run,
so this one looks for a *defect class* instead - specifically the one that has
already produced two real money bugs in this repository.

## The defect class

An assertion comparing a money value against a literal that a `float32` can render
unchanged cannot detect a `float32` regression. The test passes either way, so it
proves nothing about precision while looking like a precision test.

This is not hypothetical here. v1.1.9 found a `float32` decode in the tax-voucher
response, and v1.1.13 found the request direction on deposit, withdraw and
internal transfer. Both were sitting under assertions that passed regardless. The
v1.1.13 commit message says it plainly: the three existing assertions used 250.75,
500.25 and 1000.50, "all of which are exactly representable as a float32, so they
passed whether or not the amount had been rounded".

## The audit

Rather than assume, the suite was scanned. Across the four money-bearing test files
there are 53 decimal literals. Checking each for whether `float32` renders it
unchanged:

- **2 places** use a value that changes under `float32`. Both are deliberate, and
  both were added by the two fixes above.
- **38 literals** in `rest_banking_amount_test.go`, `rest_banking_e2e_test.go`,
  `rest_taxvouchers_e2e_test.go` and `rest_transfers_test.go` render unchanged.

So the question becomes: which of those 38 sit on a path where `float32` would
round silently? That narrows it sharply.

**Banking and transfer money is `string`-typed** - `tx.Amount`, `TransferPrice` -
and the wire fixtures send quoted strings. A `float32` regression there is a
compile error, not silent rounding. Those assertions are not vacuous; the type
does the work. And `1234.5600` on the account summary does change under `float32`
(1234.56), so the existing endtoend assertion is not vacuous either.

The real gap is narrower and sharper: **`json.Number` response fields.**
`accountSummaryRaw` declares 15 monetary fields as `json.Number` with the comment
"decodes monetary fields as json.Number to preserve decimal precision", and
`portfolio.go` declares 17 more. `toPublic` then calls `.String()` on each, so the
caller receives the gateway's exact digits - or, under a `float32` regression, a
rounded string. **39 money fields, and no test uses a value `float32` would
corrupt.**

The tax-voucher path has exactly this test, added in v1.1.9. The account summary,
positions and ledger do not.

## What this run implements

`money_precision_test.go` - precision tests for all three responses, using values
above 2^24 where a 24-bit mantissa rounds to a whole unit, plus a fractional case.
Each value is required to change under the `float32` path, and each test is
mutation-verified.

## A subtlety worth recording

The first draft used `0.007` as the "small value" case, on the reasoning that it is
below `float32` resolution. The guard in the test caught it: `0.007` is lossy in
binary but the nearest `float32` formats back as exactly `"0.007"` under Go's
shortest-round-trip formatting, so **no string comparison could ever detect it**.

The test therefore checks *detectability* rather than exact representability, and
rejects any value that survives the `float32` round trip unchanged. The tax
voucher test's `fee` assertion has this same weakness, and is worth revisiting.

## Also checked, and recorded as not-a-bug

The mock gateway sends the account summary's monetary fields as quoted JSON
strings, while the struct declares them `json.Number`, which looks like it should
fail to decode. It does not: `encoding/json` accepts a quoted string into
`json.Number` and preserves the digits. My recollection said otherwise; the probe
said otherwise, and the probe wins. Recorded so nobody re-raises it.

## Constraints held

- No credentials used.
- No production behaviour changed. One new test file, the changelog, and run
  artifacts.
- No public API change.
