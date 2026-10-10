# Todos: CI Quality Gates + Developer Experience

Run: `docs/runs/2026-09-17-ci-quality-gates/`
Plan: `plan.md` · Status legend: `todo` · `doing` · `done` · `cancelled`

| ID | Task | Role | Status | Depends On | Acceptance |
|----|------|------|--------|------------|------------|
| N1 | Coverage badge: coverprofile in CI + codecov upload | devops | done | — | coverage.out in CI artifacts |
| N2 | Coverage badge: add to README | docs | done | N1 | badge renders correctly |
| N3 | Pre-commit CI gate: pre-commit.yml workflow | devops | done | — | workflow runs on push/PR |
| N4 | FUNDING.yml | devops | done | — | file committed; renders on GitHub |
| N5 | GitHub Discussions | meta | cancelled | — | Requires manual GitHub admin action |
| N6 | Docs sync + release | docs + release | doing | N2 | `make docs-check` pass; pushed |
| N7 | Next-phase planning | planner | N6 | `next-phase.md` |
| N8 | Close-out report + index | orchestrator | N7 | `report.md` + index line |

## Notes
- Base commit: `5037b9b`
- codecov.io: no token needed for open-source repos (unauthenticated upload)
- Pre-commit is a separate CI workflow, not a git hook
- N5: repo owner must enable Discussions in GitHub repo Settings → General → Features
