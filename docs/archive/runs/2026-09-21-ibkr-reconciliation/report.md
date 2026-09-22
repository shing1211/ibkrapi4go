# Run Report — 2026-09-21-ibkr-reconciliation

## Mode: BUILD

## Summary

Two phases executed in this run:
1. **Reconciliation** — WS system frame routing, decodeJSON consistency fix, defect 5 spec patch
2. **Codegen P7** — Full regeneration from v2.40.0 spec with defects 5/6/7 applied

---

## Phase 1: Reconciliation (commits f51ae41, c6b6840)

### Shipped

| ID | Task | Outcome |
|----|------|---------|
| R1 | WS dispatch frame routing | Done — `Subscription.SystemUpdates()` channel for `sts`/`ntf`/`sor`/`usr` |
| R4 | `decodeJSON` consistency in rest_accounts.go | Done — 7 `json.Unmarshal` → `decodeJSONBytes` |
| R6 | SPEC.md date update | Done — 2026-09-21 |
| R2 | `patch_spec.py` defect 5 (float→int for ID fields) | Done — written but not yet applied |

### Not Needed

| ID | Reason |
|----|--------|
| R3 | `accountSummaryRaw` already uses `json.Number`, `toPublic()` converts to string |
| R5 | All high-value wrappers already exist (PnL, SetActiveAccount, Cancel, Modify, etc.) |

---

## Phase 2: Codegen P7 (commit f6c0422)

### Shipped

| ID | Task | Outcome |
|----|------|---------|
| P7-1 | Run codegen with defect 5 | Done — 41 ID fields float→int64, no `Conid float32` |
| P7-2 | Audit float64 money fields, defect 7 | Done — 16 money fields identified and patched |
| P7-2b | Re-patch + re-codegen | Done — defects 5+6+7 applied, spec v2.40.0 |
| P7-3 | `codegen-verify` | PASSED |
| P7-4 | Update CHANGELOG | Done — unreleased section added |

### Fixes Applied

**`patch_spec.py` defects (5/6/7):**
- Defect 5: 41 fields (conid, clientInstructionId, instructionId, etc.) `float→int64`
- Defect 6: `twsInvestDivestResponse` → `TwsInvestDivestResponseData` (collision fix for v2.40.0)
- Defect 7: 16 money fields `float64→string` (SMA, Balance, BuyingPower, NetLiquidationValue, etc.)

**`pkg/ibkr/`:**
- `portfolio.go`: `nil` params for `GetPositionByConid`/`GetPortfolioLedger`/`GetPortfolioSummary`
- `rest_banking.go`: `float32` → `int` for all banking IDs; `strToInt` helper added

**Spec:**
- `specs/ibkr_spec.json` refreshed to v2.40.0

---

## Verification

- `go build ./...` ✅
- `go vet ./...` ✅
- `go test ./internal/... ./pkg/ibkr/...` ✅
- `codegen-verify` ✅ (committed matches fresh generation)

---

## Commits

| Commit | Description |
|--------|-------------|
| `f51ae41` | feat(ws): expose system frames via SystemUpdates channel |
| `c6b6840` | docs: close out ibkr-reconciliation run |
| `f6c0422` | fix: codegen defects 5/6/7 + WS system frame routing |

Pushed to `origin/main` and `gitee/main`.
