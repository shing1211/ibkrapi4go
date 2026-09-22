# Report: D1 Root-Cause Fix — Retype Null Query Params

- **Run:** `docs/runs/2026-09-17-d1-root-cause/`
- **Mode:** BUILD
- **Base commit:** `5857f03`
- **Release commit:** `b94e677` — `fix(codegen): retype null query params to string at spec level`
- **Remotes:** `origin/main` + `gitee/main` both at `b94e677`
- **Published in:** `v0.1.1` (release commit `b7f2b81`; GitHub Release + Gitee tag)
- **Status:** complete

## Shipped

| Task | Deliverable | Status |
|------|-------------|--------|
| D1 | Enumerated 22 inline query params with `type: null` across 7 operations | done |
| D2 | `patch_spec.py` defect 4 (`type: null` → `type: string`); `patch_gen.py` retired to a no-op; codegen scripts updated | done |
| D3 | `client/client.gen.go` regenerated; 2 SDK callers adapted | done |
| D4 | Full verification: build, vet, tests, codegen-verify, money/docs/license | done |
| D5 | Docs sync + release to both remotes | done |
| D6 | Run docs, index entry, next-phase | done |

### Key outcomes

- **Root cause fixed, not worked around.** `scripts/patch_spec.py` now retypes
  inline query parameter schemas that omit `type` to `type: string`. `oapi-codegen`
  emits concrete `string` (or named string enum) types instead of a bare
  `interface{}`, so `runtime.StyleParamWithOptions` can never receive a nil
  interface value.
- **Guards eliminated.** `client/client.gen.go` no longer contains any
  `// FIX: guard nil interface{}` blocks. Optional params are generated as
  `*string` with the standard nil guard already present in the oapi-codegen
  template; required params are plain `string`.
- **Type safety improved as a side effect.** `GetContractInfoParamsRight`
  (`C`/`P`) and `GetTradingSchedule2ParamsAssetClass` (`STK`, `OPT`, …) are now
  typed string enums with `Valid()` methods.
- **`patch_gen.py` is a documented no-op**, retained only for backward
  compatibility; both `codegen.sh` and `validate_codegen.sh` no longer invoke it.
- **Public API unchanged.** `TradeManager.GetTradingScheduleBySymbol` and
  `FYIManager.ModifyFYIEmails` keep their signatures; only the internal literal
  construction changed (`client.GetTradingSchedule2ParamsAssetClass(assetClass)`
  and `strconv.FormatBool(enabled)`).

### Diff summary

| File | Change |
|------|--------|
| `client/client.gen.go` | regenerated (139 insertions, 68 deletions) |
| `scripts/patch_spec.py` | + defect 4 |
| `scripts/patch_gen.py` | −105 lines (no-op) |
| `scripts/codegen.sh`, `scripts/validate_codegen.sh` | drop `patch_gen.py` call |
| `pkg/ibkr/contract.go`, `pkg/ibkr/notifications.go` | adapt 2 callers |
| `AGENTS.md`, `docs/CODEGEN.md`, `CHANGELOG.md` | docs |

## Verification (final)

`go build ./...` · `go vet ./...` · `go test ./... -count=1` · codegen-verify
(committed matches a fresh generation byte-for-byte) · `scripts/check_money.py` ·
`scripts/check_links.py` · `scripts/check_i18n.py` · `addlicense -check` — all pass.

Coverage guard: still 185/185.

## Actuals vs plan

All planned tasks (D1–D6) completed. No scope dropped. One implementation
correction en route: the first attempt mapped typeless schemas to
`type: object, additionalProperties: {}`, which produced `map[string]any` and
broke the two SDK callers that pass a `string`/`bool`; retyping to `type: string`
was the correct mapping.
