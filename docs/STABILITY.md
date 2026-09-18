# Stability

This document describes the public API surface and stability guarantees of the
`ibkrapi4go` SDK. For the full rationale, see [ADR 0015](./adr/0015-stability.md).

## Public API

The public API is **all exported symbols in `pkg/ibkr/`** — types, functions,
constants, and variables whose name begins with an uppercase letter. This
includes:

- All manager types and their exported methods (`AccountManager`,
  `TradeManager`, `PortfolioManager`, etc.)
- All option functions (`WithGatewayURL`, `WithOAuth2ClientCredentials`, etc.)
- All exported types used as request/response values (structs, enums, aliases)
- All sentinel error variables (`ErrNotAuthenticated`, etc.)
- All exported constants (`DefaultGatewayURL`, `TimeInForceDay`, etc.)

### Not part of the public API

- `client/*.gen.go` — generated code, never hand-edited
- `internal/` — may change at any time
- Any unexported symbol
- `Client` fields — access only through options and accessors

## Stability levels

| Level | Description |
|-------|-------------|
| **Stable** | Will not change incompatibly in v1.x |
| **Experimental** | May change; indicated by `// Experimental:` in godoc |
| **Internal** | Not part of the public API; may change at any time |

No symbol is currently marked Experimental. If a new API is added and its
stability cannot yet be guaranteed, it must carry `// Experimental:` in its
godoc until promoted to Stable.

## Breaking changes

A change is **breaking** if it causes a compile error or changes runtime
behaviour for a correct caller.

### Breaking

- Removing or renaming an exported symbol
- Changing the signature of an exported function (parameter or return types)
- Removing or renaming fields of an exported struct
- Changing the type of an exported struct field
- Adding a new required parameter to a function
- Changing the meaning of a return value
- Adding a new error variant to a function that callers switch on

### Not breaking

- Adding new optional parameters with zero values
- Adding new fields to structs
- Adding new methods to types
- Adding new exported symbols
- Changing internal implementation

## Deprecation policy

To remove or rename a stable symbol:

1. Add `// Deprecated: <reason> (remove in vX.Y.0)` to the godoc and rename the
   symbol.
2. Keep the old name as a type alias or wrapper for **at least one minor
   release**.
3. Announce the deprecation in the CHANGELOG entry for that release.
4. Remove in the next minor release.

## Pre-1.0 rules

While `MAJOR == 0` (current: v0.x):

- The API may change in any **minor** release (x.1.0, x.2.0, …)
- Breaking changes are called out explicitly in the CHANGELOG
- Tags may be deleted and rewritten if needed

After v1.0.0, breaking changes require a **major** bump and a new import
path (`/v2`, etc.).
