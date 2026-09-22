# Next Phase — Post-Codegen P7 / P8 Results

## What Was Done

This run completed the reconciliation (WS system frames, decodeJSON consistency) and the full codegen P7 cycle — defects 5/6/7 applied, spec refreshed to v2.40.0, `client/client.gen.go` regenerated and verified.

P8 investigated two items:
- **Union type codegen (P8-1):** oapi-codegen generates `json.RawMessage` + `As*`/`From*` methods for `oneOf`/`anyOf` — this is working as designed. Go cannot express compile-time safe unions natively. **No action.**
- **OTel unit strings (P8-2):** Already correct — `MetricHTTPDuration` and `MetricRateLimitWaitMS` both use `"ms"`. **No action.**

## Gaps & Deferred

1. ~~json.RawMessage for union types~~ — oapi-codegen working as designed; no fix needed
2. ~~OTel histogram units~~ — Already correct; no fix needed
3. **MultiClient docs**: `examples/multi-account/main.go` exists but no written documentation
4. **Mock gateway `sor` frames**: After R1 (WS system frame routing), the mock gateway should emit `sor` frames when orders are submitted — not yet implemented
5. **Benchmark baseline**: `benchmark.baseline` needs updating after significant changes
6. **float64 in other schemas**: 17 money fields fixed in defects 5/6/7, but the spec may have more. Full audit could yield additional fixes
7. **Money-field ADR 0008 audit in codegen**: After defect 7, some float64 money fields may remain — needs re-check against regenerated client

## Candidate Next Phases

| # | Title | Objective | Why Now | Effort | Dependencies |
|---|-------|-----------|---------|--------|--------------|
| 1 | **Mock `sor` frames** | Mock gateway emits order status (`sor`) frames when orders are submitted via mock REST API | R1 exposed this gap; enables full e2e order testing via mock | M | R1 WS system frames |
| 2 | **MultiClient written docs** | Document MultiClient usage in `examples/multi-account/README.md` | Added in P6, never documented | S | MultiClient |
| 3 | **Benchmark refresh** | Update `benchmark.baseline` after codegen + refactor changes | No current baseline after large changes | S | None |
| 4 | **Full money-field re-audit** | Re-check all schemas for float64 money fields after v2.40.0 refresh + defect 7 | spec changed; may have new violations | M | None |
| 5 | **ADR 0008 codegen audit** | Audit regenerated client for remaining float64 in money-field positions | Some money fields may still be float64 in generated code | M | Defect 7 |

## Recommended Next Phase

**Phase P9: Mock Gateway `sor` Frames + MultiClient Docs Sprint**

Both are small, independent, and high-value completeness improvements.

**Proposed task breakdown:**

| ID | Task | Role | Acceptance |
|----|------|------|------------|
| P9-1 | Mock gateway emits `sor` frames on order submission | backend | `go test ./internal/mockgateway/...` passes; WS subscription receives `sor` after mock order submit |
| P9-2 | Write MultiClient usage docs in `examples/multi-account/README.md` | docs | README covers setup, parallel fan-out, error handling, example output |
| P9-3 | Update SPEC.md date if any endpoint changes | docs | Last synced updated to 2026-09-21 |
| P9-4 | Commit + push | release | Both remotes updated |

## Open Questions

1. **Mock `sor` timing:** Should the mock emit `sor` frames synchronously (immediate) or with a small delay (simulates real IBKR async behavior)?
2. **MultiClient docs scope:** Should the README also cover `MultiClient` in `pkg/ibkr/multiclient.go`, or just the `examples/multi-account/main.go` example?
