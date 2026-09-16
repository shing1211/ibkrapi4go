# 0010 — Project naming and module path

- Status: Accepted
- Date: 2026-09-16

## Context

Earlier documents used three different names for the same project: the repository
`ibkrapi4go`, a product title `ibkr-sdk`, and the Go package `ibkr`. This creates
confusion in docs, install instructions, and import paths.

## Decision

- **Module path:** `github.com/shing1211/ibkrapi4go` (stable; unchanged).
- **Go package:** `ibkr` (imported as `ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"`).
- **Display name / project title:** **`ibkrapi4go`** everywhere in docs. The name
  `ibkr-sdk` is retired.
- Repository directory names in examples use `ibkrapi4go`.

## Consequences

- One consistent name across README, docs, and repository.
- Users import `.../pkg/ibkr` while the project is called `ibkrapi4go`; this is
  normal (package name ≠ module path) and is documented.
- Renaming the module path later would be a breaking change requiring `/vN`
  versioning (see [../RELEASING.md](../RELEASING.md)).
