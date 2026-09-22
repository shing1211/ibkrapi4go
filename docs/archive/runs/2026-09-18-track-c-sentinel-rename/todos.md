# Track C — Sentinel Name Corrections

| ID | Task | Role | Status | Depends On | Acceptance |
|----|------|------|--------|------------|------------|
| T1 | Update `errors.go` — add `ErrWSDisconnected`/`ErrWSReconnected`, deprecate `ErrStream*` | backend | done | — | `go build ./...` passes |
| T2 | Update `ws_test.go` — `ErrStreamReconnected` → `ErrWSReconnected` | tester | done | T1 | `go test -race ./pkg/ibkr/` passes |
| T3 | Update `docs/ERRORS.md` — sentinel block + text references | docs | done | — | No stale `ErrStream*` in docs |
| T4 | Update `docs/STREAMING.md` — all `ErrStream*` → `ErrWS*` | docs | done | — | No stale `ErrStream*` in docs |
| T5 | Verify — full `make check` + race tests | tester | done | T1–T4 | All green |
| T6 | Commit and push to origin + gitee | release | done | T5 | Both remotes updated |
