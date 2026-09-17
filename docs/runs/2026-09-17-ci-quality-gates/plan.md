# Plan: CI Quality Gates + Developer Experience

- **Run:** `2026-09-17-ci-quality-gates`
- **Mode:** BUILD
- **Repo:** `github.com/shing1211/ibkrapi4go`
- **Base commit:** `5037b9b`

## Goal
Add coverage badge, pre-commit CI gate, GitHub Discussions enablement, and FUNDING.yml.
These are pure CI/developer-experience improvements with no new runtime dependencies.

## Scope
- **Coverage badge:** `go test -coverprofile=coverage.out ./...` in CI → codecov.io →
  badge in README.
- **Pre-commit CI gate:** `.github/workflows/pre-commit.yml` running `gofmt -s -l .`,
  `go vet ./...`, `make check` on every push and PR.
- **GitHub Discussions:** enable on repo; add SUPPORT.md or README link.
- **`FUNDING.yml`:** `.github/FUNDING.yml` with GitHub Sponsors / custom links.

## Assumptions
- codecov.io token not required for open-source repos (using unauthenticated upload).
- GitHub Discussions can be enabled via API or manually; API approach preferred.
- Pre-commit gate is a separate workflow, not a blocking git hook.

## Task breakdown
| ID | Objective | Role | Depends | Acceptance |
|----|-----------|------|---------|-----------|
| N1 | Coverage badge: coverprofile in CI, codecov upload | devops | — | coverage.out in CI artifacts |
| N2 | Coverage badge: add to README | docs | N1 | badge renders correctly |
| N3 | Pre-commit CI gate: pre-commit.yml workflow | devops | — | workflow runs on push/PR |
| N4 | FUNDING.yml | devops | — | file committed; renders on GitHub |
| N5 | GitHub Discussions | meta | — | Discussions enabled |
| N6 | Docs sync + release | docs + release | N2 | `make docs-check` pass; pushed |
| N7 | Next-phase planning | planner | N6 | `next-phase.md` |
| N8 | Close-out report + index | orchestrator | N7 | `report.md` + index line |

## Verification (global)
`make check` · `make test-race` · `make docs-check` · `make license-check`
