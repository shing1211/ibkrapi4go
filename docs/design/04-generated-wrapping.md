# Design 04 — Generated-code wrapping

Generated code is an implementation detail. Callers must never depend on it.

## Boundary rule

```
pkg/ibkr (public types)  ←─ adapters ─→  client (generated)
```

- `client/*.gen.go` is imported **only** by `pkg/ibkr` and `internal/`.
- No exported symbol in `pkg/ibkr` may be a generated type.
- Public structs are hand-written and stable; generated structs may change
  whenever the spec changes.

## Why

- A spec update must not break callers' code.
- Generated types are awkward (pointers for optional fields, `oneOf` unions as
  `json.RawMessage`, nullable wrappers).
- Public types can carry domain semantics (validated ids, string money).

## Adapter pattern

```go
// generated (unstable)
type accountSummaryGenerated struct { ... }

// public (stable)
type AccountSummary struct {
    AccountID AccountID
    NetLiq    string   // money: string, never float64
    Currency  string
    // ...
}

func toAccountSummary(g accountSummaryGenerated) AccountSummary { ... }
```

## Optional and nullable fields

- Generated optional fields arrive as pointers or nullable wrappers; adapters
  normalize them into `Option[T]`-style presence (or a zero + `HasX bool`).
- `oneOf`/`anyOf` unions are modeled as a tagged union in public types, not as
  raw generated types.

## Enums

Generated string enums are mapped to public named string types with constants.
Unknown values are preserved (not dropped) so forward compatibility holds.

## Enforcement

- `client` may only be imported from `pkg/ibkr` and `internal`. This is enforced
  by review, not by a linter (see `.golangci.yml`), to avoid false positives in
  tests.
- Reviewers reject PRs that leak generated types through exported signatures.

## Never edit generated code

Change `scripts/patch_spec.py` and regenerate. See
[../CODEGEN.md](../CODEGEN.md) and [../adr/0003](../adr/0003-openapi-codegen.md).
