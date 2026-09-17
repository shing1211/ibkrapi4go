# 0015 — Public API Surface and Stability Contract

- Status: Accepted
- Date: 2026-09-18

## Context

The project has been developing under pre-1.0 semver rules, which allow the
public API to change in minor releases. Before declaring a stable v1.0.0, the
public API surface must be explicitly defined so that:

- Contributors know which symbols are guaranteed and which are internal.
- Users can reason about upgrade paths.
- Breaking changes can be identified and classified correctly.

## Decision

### What constitutes the public API

The public API is **all exported symbols in `pkg/ibkr/`** — specifically every
type, function, constant, and variable whose name begins with an uppercase
letter. This includes:

- All manager types and their exported methods (`AccountManager`,
  `TradeManager`, `PortfolioManager`, etc.).
- All option functions (`WithGatewayURL`, `WithOAuth2ClientCredentials`, etc.).
- All exported types used as request/response values (structs, enums, aliases).
- All sentinel error variables (`ErrNotAuthenticated`, etc.).
- All exported constants (`DefaultGatewayURL`, `TimeInForceDay`, etc.).

**NOT part of the public API:**

- `client/*.gen.go` — generated code, never hand-edited (AGENTS.md rule 1).
- `internal/` — by convention, internal packages may change at any time.
- Any symbol whose godoc comment says it is internal.
- `Client` fields — access only through options and accessors.

### Stability levels

| Level | Description | Who |
|-------|-------------|-----|
| **Stable** | Will not change incompatibly in v1.x | Users |
| **Experimental** | May change; indicated by godoc `// Experimental:` | Users |
| **Internal** | Not part of the public API; may change at any time | Developers |

No symbol is currently marked Experimental. If a new API is added and its
stability cannot yet be guaranteed, it must carry `// Experimental:` in its
godoc until promoted to Stable.

### Breaking changes

A change is **breaking** if it causes a compile error or changes runtime
behaviour for a correct caller. The following are breaking:

- Removing or renaming an exported symbol.
- Changing the signature of an exported function (parameter or return types).
- Removing or renaming fields of an exported struct.
- Changing the type of an exported struct field.
- Adding a new required parameter to a function.
- Changing the meaning of a return value.
- Adding a new error variant to a function that callers switch on.

The following are **NOT breaking**:

- Adding new optional parameters with zero values.
- Adding new fields to structs (unless the struct is a request type that would
  serialize with extra fields — callers using `json.Marshal` are unaffected).
- Adding new methods to types.
- Adding new exported symbols.
- Changing internal implementation.

### Deprecation policy

To remove or rename a stable symbol:

1. Add `// Deprecated: <reason> (remove in vX.Y.0)` to the godoc and rename the
   symbol.
2. Keep the old name as a type alias or wrapper for **at least one minor
   release**.
3. Announce the deprecation in the CHANGELOG entry for that release.
4. Remove in the next minor release.

### Pre-1.0 rules

While `MAJOR == 0` (current: v0.x):

- The API may change in any **minor** release (x.1.0, x.2.0, …).
- Breaking changes are called out explicitly in the CHANGELOG.
- Tags may be deleted and rewritten if needed.
- After v1.0.0, breaking changes require a **major** bump and a new import
  path (`/v2`, etc.).

## Consequences

- Any contributor adding, removing, or renaming an exported symbol must verify
  the change is not breaking under the rules above.
- The CHANGELOG must clearly call out breaking changes with a `### Breaking`
  sub-section.
- `make check` passes, but `make codegen-verify` is the guard against
  accidental generated-code surface changes.

## Alternatives considered

- **Only document internal packages**: Rejected — users import `pkg/ibkr/`
  and need to know what is guaranteed.
- **Semantic import versioning** (`/v2`, `/v3`): Not yet needed; would be
  adopted only at v1.0.0 if the API stabilises.
