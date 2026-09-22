# release-plan.md — Release Plan

- **Run:** `2026-09-20-next-15-phases`
- **Date:** 2026-09-20
- **Mode:** BUILD (PLAN run — no actual release)

## Pre-flight

Before any release, the following must pass:

```bash
go vet ./...
gofmt -d pkg/ibkr internal
go test ./...
make license-check
make docs-check
```

Working tree must have only intended changes. Must be on `main` branch, up to
date with both `origin/main` and `gitee/main`.

## Phase releases

Since this is a PLAN run covering 15 phases across multiple topics, there is no
single release. Each phase may produce its own commit. The release cadence is:

### Per-phase commits (during BUILD execution)

Each completed phase produces a conventional commit:

| Phase | Commit type | Scope |
|-------|-------------|-------|
| S1 Goroutine safety | `fix:internal` | `internal/` |
| A1 Interface segregation | `refactor:internal` | `internal/` |
| A2 Typed builders | `feat:pkg` | `pkg/ibkr/` |
| S2 Backpressure | `fix:internal` | `internal/` |
| E1 CLI tool | `feat:cmd` | `cmd/` |
| A3 Middleware | `feat:pkg` | `pkg/ibkr/` |
| E2 Migration guide | `docs:docs` | `docs/` |
| S3 Production resilience | `feat:internal` | `internal/` |
| S4 Test quality | `test:internal` | `internal/` |
| A4 Pager types | `refactor:pkg` | `pkg/ibkr/` |
| E3 API reference site | `docs:docs` | `docs/` |
| E4 Release automation | `ci:github` | `.github/workflows/` |
| E5 Supply-chain security | `ci:github` | `.github/workflows/` |
| S5 Performance | `perf:internal` | `internal/` |
| A5 Multi-client transport | `feat:pkg` | `pkg/ibkr/`, `internal/` |

### v1.0.0 release

After all 15 phases complete, the final release:

```bash
# Update CHANGELOG.md: move Unreleased → v1.0.0, date today
# Tag:
git tag -s v1.0.0 -m "v1.0.0"
# Push:
git push origin main --tags
git push gitee  main --tags
```

## Release artifacts

Each phase execution produces:

- Commit with conventional message referencing run folder and task ID
- Files changed (per `todos.md` acceptance criteria)
- No force-push, no rebase of shared branches

## Post-release (Gitee mirror)

After GitHub push succeeds:

```bash
git push gitee main --tags
```

Verify both remotes show the new commit hash.

## Safety gates

- No `git force-push`
- No rebase of `main` or any shared branch
- Stop and report on any remote rejection
- Release only on clean `main` — no open PRs blocking
