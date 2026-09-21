# Run Report — 2026-09-21-ibkr-reconciliation

## Mode: BUILD

## Shipped

| ID | Task | Outcome |
|----|------|---------|
| R1 | WS dispatch frame routing | ✅ Done — new `SystemUpdates()` channel on `Subscription`, non-breaking |
| R2-fix | patch_spec.py defect 5 | ✅ Done — 5 ID fields (conid, clientInstructionId, instructionId, etc.) float→integer |
| R4 | decodeJSON consistency | ✅ Done — 7 json.Unmarshal calls replaced with decodeJSONBytes |
| R6 | SPEC.md date | ✅ Done — updated to 2026-09-21 |

## Not Needed

| ID | Task | Reason |
|----|------|--------|
| R3 | AccountSummary float64 wrapper | Already correct — `accountSummaryRaw` uses `json.Number`, `toPublic()` converts to string |
| R5 | Add CPAPI wrappers | All high-value endpoints already exist (PnL, SetActiveAccount, Cancel, Modify, etc.) |

## Deferred

- **Codegen regeneration**: defect 5 is in patch_spec.py but client/gen.go has NOT been regenerated. Next run should `make codegen` to apply the fix.
- **float64 money fields in generated code**: `SMA`, `AccruedInterest`, etc. in `AccountSummaryResponse` — ADR 0008 violations in generated code. Wrapped at SDK boundary in account.go; not fixed at source.
- **float32 ConID in generated code**: Already partially fixed via defect 5 in spec; regeneration needed.

## Verification

- `go build ./...` ✅
- `go vet ./...` ✅
- `go test ./internal/... ./pkg/ibkr/...` ✅

## Commit

`f51ae41` — feat(ws): expose system frames (sts/ntf/sor/usr) via SystemUpdates channel
- Pushed to `origin/main` and `gitee/main`
