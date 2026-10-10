# Plan: close the last of the json.Number class

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `560611f` (`v1.1.19`)

## Objective

Seventh request to "plan and implement remaining blocking items". Two things are
true of the backlog, and both need saying plainly rather than repeating.

**The blocked list cannot move.** D15 needs a real account; the
`submitModelPortfolioOrder` collision needs an FA-enabled paper account. Nothing
written in a local checkout changes that.

**The offline work is down to its last item.** The previous two runs closed the
monetary `json.Number` fields and then declined to ship a gate. What remains, and
what this run does, is the rest of the same class.

## The class, completed

Every `json.Number` field in the SDK that reaches a caller as a string is at risk
from a `float32` regression: the digits would round silently, with no compile error
and no decode failure. Two real instances have already been found this way - the
tax-voucher response (v1.1.9) and the banking request (v1.1.13).

Covered so far:

| response | fields | run |
|---|---|---|
| account summary | 14 monetary | 1.1.18, 1.1.19 |
| positions | 6 | 1.1.18 |
| ledger | 7 | 1.1.18, 1.1.19 |
| tax voucher dividends | 4 | 1.1.9 |

Still open, and the whole of what is left:

- `marketdata.go` - five `json.Number` fields in an anonymous struct behind
  `MarketData().History`: open, high, low, close and volume. Four prices and a
  count.
- `contract.go` - `Call []json.Number` and `Put []json.Number` behind
  `Trade().Strikes`, option strike prices, plus `Multiplier` on the contract info
  response.

Two of these are structurally new, which is the reason they are worth a run of their
own rather than being folded into the money file:

- **The bar decode is a scalar set**, five fields in one anonymous struct, all
  stringified through `.String()`.
- **The strike decode is a slice.** A slice is a different decode target from a
  scalar: an element that regressed would leave the surrounding structure intact,
  so the assertions check values *and* length.
- **The multiplier is a scale factor, not a price.** A rounding error there would
  misstate the size of a contract rather than its cost, which is a quieter and
  worse failure than a wrong price.

## What this run implements

`number_precision_test.go`, reusing the `float32Loses` guard from
`money_precision_test.go` so the new file cannot quietly stop proving anything:
values that a `float32` renders unchanged are rejected at authoring time rather than
accepted as decorative.

## Constraints held

- No credentials used.
- No production behaviour changed. One new test file, the changelog, run artifacts.
- No public API change.
