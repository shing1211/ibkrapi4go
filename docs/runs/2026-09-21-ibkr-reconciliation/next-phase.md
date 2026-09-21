# Next Phase — Post-Reconciliation

## What Was Done

This run added WS system-frame routing (R1), fixed decodeJSON consistency in rest_accounts.go (R4), added defect 5 to patch_spec.py for float32→int64 ConID fields (R2-fix), and updated SPEC.md (R6). All changes verified and pushed.

## Gaps & Deferred

1. **Codegen not regenerated**: defect 5 is in patch_spec.py but `client/client.gen.go` still has `float32` for ID fields. `make codegen` + `make codegen-verify` needed.
2. **float64 money in generated AccountSummary**: `SMA`, `AccruedInterest`, etc. in `AccountSummaryResponse` use `float64` in generated code — ADR 0008 violation, but wrapped at SDK boundary.
3. **float32 InstructionIDs**: `clientInstructionId`, `instructionId` etc. still `float32` — defect 5 handles them in spec, but codegen not yet run.
4. **json.RawMessage for union types**: ~38 responses use `json.RawMessage` — bypasses compile-time type safety. Requires a different approach (custom decoder or union type generation).

## Candidate Next Phases

| # | Title | Objective | Why Now | Effort | Dependencies |
|---|-------|-----------|---------|--------|--------------|
| 1 | **Codegen regeneration** | Run `make codegen` to apply defect 5 and regenerate `client/gen.go` | Defect 5 is committed but not active | S | patch_spec.py defect 5 |
| 2 | **Money-field ADR 0008 audit** | Audit all generated float64 money fields, patch spec to use string, regenerate | ADR 0008 violations in 15+ fields | M | Codegen (phase 1) |
| 3 | **Union type codegen** | Investigate oapi-codegen `generate` option for `allOf`/`oneOf` to replace json.RawMessage | 38 union responses lack type safety | L | Codegen (phase 1) |
| 4 | **OTel histogram units** | Add explicit unit strings (USD, shares) to OTel histogram calls | Consistency and better observability | S | — |
| 5 | **MultiClient docs** | Document MultiClient usage with examples | Added in P6, no example yet | S | — |
| 6 | **Mock gateway order fills** | Mock WS `sor` frame emission when order submitted | R1 exposed this gap | M | R1 WS routing |
| 7 | **Benchmarks** | Add `scripts/bench_compare.go` benchmarks for key operations | No performance baseline | M | — |

## Recommended Next Phase

**Phase P7: Codegen regeneration + ADR 0008 money audit**

Apply the committed defect 5 fix, then audit and patch all `float64` money fields in the spec.

**Proposed task breakdown:**

| ID | Task | Role |
|----|------|------|
| P7-1 | Run `make codegen` and resolve any diff | backend |
| P7-2 | Audit generated float64 money fields, add defect 6 to patch_spec.py | backend |
| P7-3 | Run `make codegen-verify` | tester |
| P7-4 | Update CHANGELOG | docs |
| P7-5 | Commit + push | release |

## Open Questions

1. Should `make codegen` be run now (it's a destructive regeneration step)?
2. Should the float64 money fields be patched at spec level, or is the current SDK-level wrapping sufficient?
