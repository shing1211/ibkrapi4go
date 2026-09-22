# Plan: Phase 9 — Correctness & Conformance Closure

- **Run:** `2026-09-17-correctness-closure`
- **Mode:** BUILD
- **Repo:** `github.com/shing1211/ibkrapi4go`
- **Base commit:** `eb34ca1` (close-out of mock-gateway run)

## Goal
Fix critical correctness issues in the SDK discovered during the mock-gateway development:
D1 (generated-client nil-`interface{}` panics) and D2 (8 REST wrapper/decode mismatches).

## Discovered issues

### D1 — Generated-client nil-`interface{}` panic
Wrappers `TradeManager.GetConidsByExchange`, `TradeManager.GetContractInfo`, and
`FYIManager.GetAllFYIs` panic before any HTTP call because the generated request
builders pass a nil `interface{}` param to `runtime.StyleParamWithOptions` with no
nil guard (`client/client.gen.go:44409` for `GetConidsByExchange`).

**Root cause:** The spec-patch pipeline produces `interface{}` with `omitempty` for
optional non-pointer params, but the `oapi-codegen` request-builder template does not
guard nil for non-pointer interface types.

**Fix location:** `scripts/patch_spec.py` + `make codegen` (AGENTS.md rule 1: do not
edit `client/client.gen.go` directly). Blocked locally by missing `oapi-codegen`
(`make tools` fails without it).

**Scope:** All generated request builders that pass `interface{}` params to
`runtime.StyleParamWithOptions` without a nil guard. Must identify all affected
operations.

### D2 — 8 REST wrapper/model decode mismatches
Driving every REST wrapper against the mock surfaced 8 pre-existing decode issues:

| Operation | Issue |
|-----------|-------|
| `TaxDocuments.Generate` | Wrapper re-reads a body already consumed/closed by `ParseCreateTaxDocumentsResponse` |
| `Utilities.Enumerations` | Op returns `[]string` but generated model may differ |
| `Utilities.ComplexAssetTransferBrokers` | Shape mismatch |
| `Utilities.RequiredForms` | Shape mismatch |
| `TaxVouchers.CreateRequests` | Shape mismatch |
| `TaxVouchers.ActiveCountries` | Shape mismatch |
| `TaxVouchers.AvailableYears` | Shape mismatch |
| `TaxVouchers.Dividends` | Shape mismatch |

**Fix:** Either correct the wrapper to match the spec/generated-model shape, or
correct the fixture to match the wrapper's expectations — whichever is correct per
the SPEC and the generated client.

## Approach
- **D1:** Read `client/client.gen.go` to enumerate ALL request builders with the nil
  interface pattern. Modify `scripts/patch_spec.py` to add explicit nil guards in
  the generated code (or use a post-processing step). Verify with `make codegen`
  if toolchain available, otherwise patch `client/client.gen.go` directly and flag
  for `make codegen-verify` once toolchain is available.
- **D2:** For each op, read the generated client model vs the wrapper decode logic
  and determine the correct shape. Fix wrappers (not fixtures — fixtures match
  spec; wrappers are SDK-level code we own).

## Task breakdown
| ID | Objective | Role | Depends | Acceptance | Verify |
|----|-----------|-------|---------|-----------|--------|
| T1 | D1: enumerate all nil-interface builder panics | backend | — | full list of affected ops; nil guard patch in patch_spec.py OR direct gen.go patch | compile + test affected wrappers |
| T2 | D1: apply nil guard fix | backend | T1 | affected ops compile + pass | `make check` |
| T3 | D2: fix 8 REST wrapper mismatches | backend | T2 | 8 ops decode correctly against mock | `make check`, integration test |
| T4 | Docs sync | docs | T3 | no stale refs; docs-check pass | `make docs-check` |
| T5 | Release: commit + push both remotes | release | T4 | both remotes at new commit | `git push` |
| T6 | Next-phase planning | planner | T5 | `next-phase.md` | — |
| T7 | Close-out report + index | orchestrator | T6 | `report.md` + index | — |

## Assumptions
- `make tools` is NOT available locally (oapi-codegen not installed); D1 fix
  will patch `client/client.gen.go` directly as a pragmatic fallback per the
  codegen-verify gap, or the fix targets the patch script only.
- D2 fixes are wrapper-side only (fixtures match spec).
- AGENTS.md rule 1 (`client/client.gen.go` is never edited directly) may need
  to be locally suspended for this emergency fix if `make codegen` is unavailable.

## Risks
- **D1 toolchain gap:** without `oapi-codegen`, the fix may require direct
  `client/client.gen.go` edits; this is a one-time correctness fix, not
  ongoing codegen.
- **D2 shape ambiguity:** in some cases the spec, the generated model, and the
  wrapper may all disagree; need a human decision on which is correct.

## Verification (global)
`make check` · `make test-race` · `make docs-check` · `make license-check`
