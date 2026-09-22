# Track C — Sentinel Name Corrections (Report)

- **Date:** 2026-09-18
- **Mode:** BUILD
- **Status:** Complete
- **Commit:** `7cbc084`

## Summary

Renamed the public WebSocket sentinel errors from `ErrStreamDisconnected`/`ErrStreamReconnected` to `ErrWSDisconnected`/`ErrWSReconnected`, matching the internal definitions and ADR 0016 documentation. The old names are retained as deprecated aliases.

## Files Changed

| File | Change |
|------|--------|
| `pkg/ibkr/errors.go` | Added `ErrWSDisconnected`/`ErrWSReconnected` as primary sentinels; marked `ErrStream*` as deprecated aliases |
| `pkg/ibkr/ws_test.go` | Updated 2 test references from `ErrStreamReconnected` → `ErrWSReconnected` |
| `docs/ERRORS.md` | Updated sentinel code block and paragraph; added deprecated alias note |
| `docs/STREAMING.md` | Updated 4 references from `ErrStream*` → `ErrWS*` |
| `docs/runs/2026-09-18-track-c-sentinel-rename/` | Run artifacts (plan.md, todos.md) |

## Verification

- `make check` — PASS (gofmt, go vet, all tests)
- `check_money.py` — PASS (no float fields in pkg/ibkr)
- `go test -race -count=1 ./...` — PASS (all packages)
- `grep 'ErrStream*' pkg/` — only deprecated aliases remain

## Follow-ups

- Track D (godoc sprint), Track E (unexport wrappers), Track F (RELEASING.md + STABILITY.md), Track G (tag v0.2.0) remain.
