# 0008 — Money and quantities are never `float64`

- Status: Accepted
- Date: 2026-09-16

## Context

IBKR frequently returns monetary values, quantities, and prices as **JSON
strings** specifically to preserve decimal precision (e.g. `"1234.5600"`).
Decoding these into `float64` silently loses precision and can corrupt order
prices and quantities. Some generated types may map numeric schemas to Go
floating-point types by default.

## Decision

- Represent money, prices, and quantities as `string` or `json.Number` — **never
  `float64`** — in public types.
- Provide explicit helpers for arithmetic/conversion where needed, with documented
  rounding behavior; do not implicitly convert to binary floating point.
- Configure/enforce this in codegen (`x-go-type` overrides where the spec would
  otherwise produce floats) and review generated types for any `float64` on money
  fields.

## Consequences

- No silent precision loss when placing orders or reading balances.
- Consumers must parse strings/`json.Number` themselves or use provided helpers.
- A CI/lint check flags new `float64` usages on money-like fields.
- Slightly more verbose API for numeric fields.
