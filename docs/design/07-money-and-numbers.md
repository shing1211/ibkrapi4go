# Design 07 — Money and numbers

Implements [ADR 0008](../adr/0008-numeric-precision.md): money, prices, and
quantities are **never `float64`**.

## Representation

| Concept | Type | Notes |
|---------|------|-------|
| Money amount | `string` | exactly as returned by IBKR, e.g. `"1234.5600"` |
| Price | `string` | never parse to float for display/storage |
| Quantity / size | `string` or `json.Number` | fractional quantities exist |
| Percentage | `string` | e.g. `"1.23"` |
| Counts (pages, ids) | `int` | non-monetary only |

Public fields use `string` by default. `json.Number` is used internally where a
value may be numeric or string in JSON.

## Why strings

IBKR returns decimals as JSON strings to preserve precision. `float64` is binary
floating point and cannot represent many decimal values exactly; decoding a price
into `float64` can alter it. For order prices and quantities that is unacceptable.

## Helpers (not yet provided)

Money and quantities are currently carried as raw `string` values and compared by
the caller; the SDK does not ship arithmetic helpers. If/when they are added,
the intended shape is:

```go
package amounts

// Add returns a + b as a decimal string. No binary floats are used.
func Add(a, b string) (string, error)

// Mul returns a * b as a decimal string.
func Mul(a, b string) (string, error)

// Compare returns -1, 0, or 1.
func Compare(a, b string) (int, error)

// IsZero reports whether the decimal is zero.
func IsZero(s string) bool
```

If arbitrary-precision decimal math is needed, it is implemented over
`math/big.Rat`/`big.Int` internally, preserving the string representation. A
decimal library may be evaluated via ADR if the standard library proves
insufficient.

> Arithmetic helpers are not yet implemented. For now, callers should use
> `strconv.ParseDecimal` from the standard library or a decimal package for
> calculations.

## Codegen enforcement

- Where the spec would generate `float64` for a monetary field, add an
  `x-go-type` override to `string` (recorded in `scripts/patch_spec.py` or
  `oapi-codegen.yaml`).
- `scripts/check_money.py` (run by `make check`) enforces this on `pkg/ibkr`:
  it fails the build if any exported struct field under `pkg/ibkr` is `float32`/`float64`
  and its name matches money patterns (`Price`, `Amount`, `Qty`, `Quantity`,
  `Balance`, `Cash`, `NetLiq`). A review checklist item: any `float64` on a
  money/price/quantity field is a bug.

## Formatting vs. value

Formatting (e.g. two-decimal display) happens at the edge, from the string value.
The SDK never rounds money implicitly.

## Tests

- Round-trip: string in → string out, byte-identical for untouched values.
- Arithmetic helpers tested against known decimal cases (including values that
  `float64` would corrupt).
- Lint/AST check: no `float64` fields under `pkg/ibkr` whose names match money
  patterns (`Price`, `Amount`, `Qty`, `Quantity`, `Balance`, `Cash`, `NetLiq`).
