# Next Phase: CI Quality Gates + Developer Experience

## Context

The 2026-09-17 benchmarks+fuzz run (`c97a027`) shipped:
- `pkg/ibkr/benchmark_test.go` — 20 JSON encode/decode benchmarks
- `internal/benchmark_test.go` — HTTP RT, WS, session init benchmarks
- `pkg/ibkr/fuzz_test.go` — 47 fuzz functions
- `internal/fuzz_test.go` — 185 op fixture decode fuzzing
- `benchmark.baseline` + `scripts/bench_compare.go` — CI regression gate
- `.github/workflows/ci.yml` — benchmarks job integrated

Remaining backlog (ROADMAP.md): **coverage badge, pre-commit hooks, GitHub Discussions, FUNDING.yml, docs website**.

## Recommended next phase: Coverage Badge + Pre-commit Hooks

### Why this bundle?
- Both are pure CI/developer-experience improvements with no new runtime dependencies.
- Coverage badge requires adding `go test -coverprofile` to CI; this surfaces which
  packages have weakest test coverage and guides future test writing.
- Pre-commit hooks (using `.git/hooks/pre-commit` or a minimal CI gate) catch
  `gofmt`, `go vet`, `make check` regressions before they reach CI.
- Both are low-risk, bounded (S–M effort), and have high visibility.

### Scope

| Item | Description |
|------|-------------|
| **Coverage badge** | Add `go test -coverprofile=coverage.out ./...` to CI, upload to
  `codecov.io` or `coveralls.io` via GitHub Actions `upload-artifact` step; add
  codecov badge to README. No new deps. |
| **Pre-commit CI gate** | Add a `.github/workflows/pre-commit.yml` that runs
  `gofmt -s -l .`, `go vet ./...`, `make check` on every push and PR. Fast
  (~30s). Catches regressions before CI. |
| **GitHub Discussions** | Enable GitHub Discussions on the repo; add a `SUPPORT.md`
  or link to discussions in README. |
| **`FUNDING.yml`** | Add `.github/FUNDING.yml` with GitHub Sponsors / custom links. |

### What's NOT in this phase
- Docs website (requires hosting, out of scope for SDK correctness)
- GitHub Sponsors until funding infrastructure is set up

## Risks
- Coverage badge may encourage gaming (aiming for % rather than correctness).
  Mitigation: focus on regression coverage (diff-based) not absolute %.
- Pre-commit CI adds a parallel job; must not slow down existing CI.

## Task breakdown (draft)

| ID | Objective | Role |
|----|-----------|------|
| N1 | Coverage badge: add `coverprofile` to CI, integrate codecov | devops |
| N2 | Coverage badge: add `[![Coverage]` to README | docs |
| N3 | Pre-commit CI gate: `.github/workflows/pre-commit.yml` | devops |
| N4 | `FUNDING.yml` | devops |
| N5 | Enable GitHub Discussions | meta |
| N6 | Docs sync + release | docs + release |
| N7 | Next-phase planning | planner |
| N8 | Close-out report + index | orchestrator |

## Run folder
`docs/runs/<date>-ci-quality-gates/`
