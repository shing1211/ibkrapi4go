# Next Phase — Post-Codegen P7

## What Was Done

This run completed the reconciliation (WS system frames, decodeJSON consistency) and the full codegen P7 cycle — defects 5/6/7 applied, spec refreshed to v2.40.0, `client/client.gen.go` regenerated and verified.

## Gaps & Deferred

1. **json.RawMessage for union types**: ~38 responses use `json.RawMessage` — bypasses compile-time type safety for `oneOf`/`anyOf` schemas. Requires oapi-codegen `generate` options for `allOf`/`oneOf` or a custom decoder.
2. **OTel histogram units**: OTel histogram calls lack explicit unit strings (USD, shares). Easy fix, low priority.
3. **MultiClient docs**: `examples/multi-account/main.go` exists but no written documentation.
4. **Mock gateway `sor` frames**: After R1 (WS system frame routing), the mock gateway should emit `sor` frames when orders are submitted — not yet implemented.
5. **Benchmark baseline**: `benchmark.baseline` needs updating after significant changes.
6. **float64 in other schemas**: 17 money fields fixed in defects 5/6/7, but the spec may have more. Full audit of all schemas could yield additional fixes.

## Candidate Next Phases

| # | Title | Objective | Why Now | Effort | Dependencies |
|---|-------|-----------|---------|--------|--------------|
| 1 | **Union type codegen** | Replace `json.RawMessage` unions with typeddiscriminated unions using oapi-codegen `generate` options | 38 responses lack type safety; high correctness impact | L | None |
| 2 | **OTel unit strings** | Add explicit unit parameters to OTel histogram calls | Better observability; consistent metric labels | S | None |
| 3 | **Mock `sor` frames** | Mock gateway emits order status frames when orders submitted | R1 exposed this gap; better e2e testing | M | R1 WS routing |
| 4 | **MultiClient written docs** | Document MultiClient usage pattern in docs | Added in P6, no written guide yet | S | MultiClient |
| 5 | **Full money-field re-audit** | Re-audit all schemas for additional float64 money fields after v2.40.0 refresh | spec changed; may have new violations | M | None |
| 6 | **Benchmark refresh** | Update `benchmark.baseline` after codegen + refactor changes | No current baseline | S | None |

## Recommended Next Phase

**Phase P8: Union type codegen + OTel units**

Both are small, independent, and high-value. Spawn two sequential sub-agents.

**Proposed task breakdown:**

| ID | Task | Role |
|----|------|------|
| P8-1 | Investigate oapi-codegen options for `allOf`/`oneOf` union types | backend |
| P8-2 | Add unit strings to all OTel histogram calls | backend |
| P8-3 | Update `contrib/otel/README.md` if OTel API changed | docs |
| P8-4 | Commit + push | release |

## Open Questions

1. Is union type codegen a priority vs. other items?
2. Should the mock `sor` frame implementation be part of P8 or deferred?
