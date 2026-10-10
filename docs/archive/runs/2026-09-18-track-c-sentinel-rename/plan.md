# Track C — Sentinel Name Corrections

- **Date:** 2026-09-18
- **Mode:** BUILD
- **Status:** In Progress

## Goal

Rename the public WebSocket sentinel errors from `ErrStreamDisconnected`/`ErrStreamReconnected` to `ErrWSDisconnected`/`ErrWSReconnected`, matching the internal definitions and ADR 0016 documentation. The old names become deprecated aliases.

## Scope

- `pkg/ibkr/errors.go` — primary sentinels + deprecated aliases
- `pkg/ibkr/ws_test.go` — update test references
- `docs/ERRORS.md` — update sentinel block
- `docs/STREAMING.md` — update all references

## Approach

**Option chosen:** Add correct `ErrWS*` names as primary; mark `ErrStream*` deprecated with alias. This is a pre-1.0 package so breaking changes in minor releases are acceptable per semver, but keeping the deprecated alias for one release is safer for any downstream users.

## Task Breakdown

| ID | Task | Role | Size | Acceptance |
|----|------|------|------|------------|
| T1 | Update `errors.go` — add `ErrWSDisconnected`/`ErrWSReconnected`, deprecate `ErrStream*` | backend | S | `go build ./...` passes |
| T2 | Update `ws_test.go` — `ErrStreamReconnected` → `ErrWSReconnected` | tester | S | `go test -race ./pkg/ibkr/` passes |
| T3 | Update `docs/ERRORS.md` — sentinel block + text references | docs | S | No stale `ErrStream*` in docs |
| T4 | Update `docs/STREAMING.md` — all `ErrStream*` → `ErrWS*` | docs | S | No stale `ErrStream*` in docs |
| T5 | Verify — full `make check` + `check_money.py` + race tests | tester | S | All green |
| T6 | Commit and push to origin + gitee | release | S | Both remotes updated |

## Risks

- **Low:** Downstream callers using `ErrStream*` will get deprecation warnings (via `// Deprecated` godoc) but won't break at compile time since aliases remain.
- **Low:** ADR 0016 already documents `ErrWS*` names — no ADR changes needed.

## Verification

```bash
make check                        # gofmt + go vet + tests
python3 scripts/check_money.py    # no float fields in pkg/ibkr
go test -race -count=1 ./...      # full race-tested suite
grep -r 'ErrStreamDisconnected\|ErrStreamReconnected' pkg/  # only in deprecated aliases
```
