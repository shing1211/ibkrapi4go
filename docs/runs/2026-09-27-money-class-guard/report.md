# Run: close the float32 money class

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `04162cb` (`v1.1.10`)
- **Scope**: item 1 of the post-v1.1.10 recommendation — turn the narrow D14 fix
  into a closed class.

## Why this was worth doing, and what it was *not*

The measurement that motivated it: the generated client holds **426 `float32`
fields, 198 of them monetary** — `balance`, `baseMktValue`,
`allowedTransferAmountToMaster`, `interest`, `fee` and so on. `check_money.py`
passed anyway, and the reason is structural rather than accidental: it inspects
exported struct fields under `pkg/ibkr`, while `client/` is excluded from lint
outright. The gate was blind to all 198.

That could have been a finding of live corruption. It was not. Tracing where
money actually reaches a caller showed the opposite:

- `rawToString` extracts money through `json.Number`, so the raw-decode path
  (110 of 193 operations) preserves the gateway's digits.
- After v1.1.9, `float32ToStr` had **zero production call sites**.
- The only other `Float64()` calls in the SDK convert `json.Number` to `int` for
  counts and ConIDs, not money.

So there is no live money bug. What there is, is a **trap**: 198 reachable-by-
accident fields, one dead helper that renders them lossily, and no gate on the
path between them. The work here is preventative, and the report says so rather
than claiming a save.

## The gate's blind spot

`check_money.py` had exactly one rule: no exported struct field is a binary
float. That is a *declaration* rule. The v1.1.9 defect was entirely inside a
function body — `float32ToStr` had a clean signature and quietly rounded
`16777217` to `"16777216"`. No declaration rule can see that.

## What changed

`check_money.py` gains a second rule for production `pkg/ibkr` code, rejecting two
vectors:

- `strconv.FormatFloat` — the direct call, and
- **any helper that takes a binary float and returns a string** — how the same
  defect returns once the direct call is removed. This is the vector that matters,
  because it also catches a helper nothing calls yet.

`ALLOWED_FLOAT_FORMATS` is the escape hatch; an entry must state why. It is empty.

### Both vectors were confirmed to bite

Each was reintroduced into `rest.go` in turn and the gate was run:

```
caught: direct FormatFloat call (the original D14 shape)
   pkg/ibkr/rest.go:912: strconv.FormatFloat formats a binary float; a money string
       must come from a quoted string field or json.Number so the gateway's digits survive
caught: helper reintroduced without a direct FormatFloat
   pkg/ibkr/rest.go:908: f32ToStr() takes a binary float and returns a string; money
       must not be rendered through a float mantissa (ADR 0008)
```

The script's own `_self_test` gained six cases covering both vectors and the
sanctioned `json.Number` path, so a broken scanner fails before it reports
anything.

## Dead code, and why `unused` missed it

`float32ToStr` had no production caller left. It survived anyway, because
`run.tests: true` in the lint config makes `unused` count a function as used when
only a test references it — so the sole thing keeping a lossy money formatter alive
was a test asserting the rounding it caused, whose own comment claimed
"this one stays because float32ToStr is still live". Both the helper and that
test are gone, and the new rule closes the shape of the blind spot.

## Documentation corrected, not just appended

`07-money-and-numbers.md` described the gate as failing "if any exported struct
field ... is `float32`/`float64` **and its name matches money patterns**
(`Price`, `Amount`, `Qty`, `Quantity`, `Balance`, `Cash`, `NetLiq`)". The name
matching does not exist — the check rejects *every* exported float field against a
small allowlist. The doc also omitted the float-formatting rule entirely. Both
corrected, since code is authoritative for these documents.

## Verification

| Check | Result |
|-------|--------|
| `check_money.py` | OK, 55 files; self-test passes |
| both guard vectors reintroduced | **caught**, with specific diagnostics |
| `checkMoneyNumberFields` mutation | still catches `TaxVoucherDTO.divAmount` regaining `float32` |
| `check_design` | 9 of 9 documents verified |
| `golangci-lint run` | 0 issues |
| `go build ./...` / `go vet ./...` / `gofmt -l .` | clean |
| `go test ./...` | pass |
| `go test -race` | clean |
| links / i18n / spec version / SPDX | pass |
| codegen drift | 0 lines |

## Not addressed

- **The 198 latent `float32` fields are still `float32` in the generated client.**
  Retyping them all is a large, high-risk spec change that would need per-field
  knowledge of whether the gateway quotes the value; the spec says `number` for
  most, and `string` for a documented minority. That is a project, not a patch.
  What is now guaranteed is that none of them can reach a caller through a float.
- **The 21 remaining `time.Sleep` sites in tests are still unaudited.** The
  250 ms ones in `session_test.go` are the most suspicious. Bundled with the
  coverage work.
- **`unused` still cannot see production code kept alive only by a test** for
  shapes the new money rule does not cover. The general blind spot remains; this
  release only closes the money-shaped instance of it.
