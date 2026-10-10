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

Public fields use `string` by default. `json.Number` is used internally in two
cases: where a value may be numeric or string in JSON, and where a field is
always numeric but a generated binary float would round it. The tax voucher's
`divAmount`, `withHeldAmount`, `fee` and `quantity` are the second case: the
gateway sends them as bare JSON numbers, so retyping them to `string` would make
the decode fail outright, and a `float32` mantissa is 24 bits - it silently
rounds anything above 2^24 (16777216), which an aggregate withholding figure can
exceed. `json.Number` accepts the number and preserves the gateway's own digits.

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
  `oapi-codegen.yaml`). Where the gateway sends the field as a bare JSON number,
  the override is `json.Number` instead: `string` would fail to decode, because
  Go cannot unmarshal a number into a string.
- `scripts/check_money.py` (run by `make check` and in CI) enforces this on
  `pkg/ibkr` and `internal` with two rules.

  **No exported money field is a binary float.** Any exported field on an
  exported struct that is `float32`/`float64` fails the build, with a short
  allowlist for the observability aggregates and one non-monetary request field.
  There is no name matching: every exported float field has to be justified.

  **Money is not rendered through a float.** The field rule inspects
  declarations, so it cannot see function bodies - and that is where the
  tax-voucher defect lived, where a decoded `float32` was re-rendered with
  `strconv.FormatFloat` and the cents silently vanished above 2^24. The gate
  therefore also rejects `strconv.FormatFloat` in production `pkg/ibkr` code, and
  any helper that takes a binary float and returns a string, which is how the
  same defect returns once the direct call is removed. The only sanctioned
  sources of a money string are a quoted `string` field and `json.Number` via
  `rawToString`/`jsonNumberToStr`; both preserve the gateway's own digits.

  A genuine need for float formatting is an explicit entry in
  `ALLOWED_FLOAT_FORMATS` in that script, which has to say why.

## Formatting vs. value

Formatting (e.g. two-decimal display) happens at the edge, from the string value.
The SDK never rounds money implicitly.

## Tests

- Round-trip: string in → string out, byte-identical for untouched values.
- Arithmetic helpers tested against known decimal cases (including values that
  `float64` would corrupt).
- Lint/AST check: no `float64` fields under `pkg/ibkr` whose names match money
  patterns (`Price`, `Amount`, `Qty`, `Quantity`, `Balance`, `Cash`, `NetLiq`).
