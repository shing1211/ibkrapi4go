# Report: the json.Number class, closed

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `560611f` (`v1.1.19`)
- **Outcome**: complete, uncommitted pending approval

## Why the backlog is where it is

Two things, stated once rather than re-derived each run.

**The blocked items cannot move.** D15 needs a real account; the
`submitModelPortfolioOrder` gateway-side collision needs an FA-enabled paper
account. No local change alters that.

**The offline backlog is now empty of anything substantive.** Three consecutive runs
took the `json.Number` class from two real bugs to full coverage, and one of them
spent its budget establishing that the obvious gate could not be written soundly.
This run closes the last of the class, which is the honest description of what is
left.

## What was covered

`number_precision_test.go`, three responses that had no precision test:

| decode | shape | fields |
|---|---|---|
| historical bars | 5 scalars in one anonymous struct | open, high, low, close, volume |
| option strikes | two `[]json.Number` slices | call, put |
| contract info | one scalar | multiplier |

The three are worth separating rather than merging into the money file:

- **Bars** are four prices and a count, all stringified through `.String()`. A
  `float32` regression would round all five, and the count is included because it
  travels the identical path.
- **Strikes are a slice**, which is a different decode target from a scalar. An
  element that regressed would leave the surrounding structure intact, so the
  assertions check the values *and* the length - a slice that silently lost an
  element would otherwise pass a value-only check.
- **The multiplier is a scale factor, not a price.** A rounding error there misstates
  the *size* of a contract rather than its cost, which is a quieter failure than a
  wrong price and a good reason to assert it.

The `float32Loses` guard from `money_precision_test.go` is reused, so the new file
rejects values a `float32` renders unchanged at authoring time - the same discipline
that caught `0.007` in an earlier draft and now prevents the mistake rather than
recording it.

## A fixture mistake worth recording

`TestContractInfo_MultiplierPrecision` set its fixture on `OpGetContractInfo` and
the multiplier came back `"1"`. `Trade.ContractInfo` is labelled
`"Trade.ContractInfo"` but calls the **`getInstrumentInfo`** endpoint. The SDK's op
label and the route it hits are different things, and a test keyed off the label
silently reads the default fixture rather than failing loudly - the assertion caught
it, but only because it was asserting a distinctive value.

That is a small trap for the next person writing a fixture here, and the test now
carries a comment saying which route it actually is.

## Mutation verification

Five mutations, each emulating a `float32` decode in the production conversion, all
caught on the assertion:

| path | reported under a `float32` decode |
|---|---|
| bar open | `16777218` (want `16777217.89`) |
| bar close | `67108864` (want `67108865.13`) |
| bar volume | `12345.679` (want `12345.6789`) |
| strike slice element | `["16777218", "12345679"]` |
| contract multiplier | `12345.679` (want `12345.6789`) |

The helper lived in a temporary test file rather than being injected through an
import rewrite - the approach that broke twice in the previous run.

## The class is closed

| response | status |
|---|---|
| account summary | 1.1.18, 1.1.19 |
| positions | 1.1.18 |
| ledger | 1.1.18, 1.1.19 |
| tax voucher dividends | 1.1.9 |
| historical bars, strikes, multiplier | this run |

No `json.Number` field that reaches a caller as a string is now left without a
`float32`-changing assertion. That is the last substantive item reachable without
credentials.

## Verification

All gates green: `gofmt -l`, `go build ./...`, `go vet ./...`, `go test ./...`,
`go test -race`, `golangci-lint run` and `--tests=false`, `check_money.py`,
`check_design`, `check_links.py`, `check_i18n.py`, `check_spec_version.py`.
Coverage 66.3% against a 62% floor.

## Production code

None. One new test file, the changelog, and run artifacts.
