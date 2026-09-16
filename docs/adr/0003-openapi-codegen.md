# 0003 — Generate types and client with oapi-codegen + spec patching

- Status: Accepted
- Date: 2026-09-16

## Context

Hand-writing ~443 schema types and an HTTP client for 185 operations is
infeasible and drift-prone. The upstream OpenAPI spec is the source of truth.

However, the published spec (v2.39.0) does **not** generate cleanly. Measured
defects:

- 27 operations whose declared path parameters do not match their path template.
- 1 duplicate `operationId` (`getTradingSchedule`).
- 2 Go type-name collisions (`ErrorResponse`/`errorResponse`, `User`/`user`).

## Decision

- Generate with `oapi-codegen` (pinned).
- Apply a deterministic, idempotent patch step (`scripts/patch_spec.py`) **before**
  generation, addressing the three defect classes.
- Commit the generated `client/*.gen.go`; never edit it by hand.
- Verify reproducibility with `make codegen-verify`.
- Do not redistribute the spec; fetch it at build time into gitignored `specs/`.

## Consequences

- Regeneration is deterministic and reviewable.
- Spec changes are handled by adjusting the patch script, not generated output.
- CI fails if committed generated code drifts from the regenerated output.
- Measured result: generation exits 0 and the output compiles (see
  [../CODEGEN.md](../CODEGEN.md)).
