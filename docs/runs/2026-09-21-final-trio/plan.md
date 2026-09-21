# Plan — Community Seeding + Real-World Examples + v1.0.1 Patch

- **Run:** `docs/runs/2026-09-21-final-trio/`
- **Date:** 2026-09-21
- **Mode:** BUILD

## Goal

Execute three independent tasks: community seeding (S), two targeted examples
(S each), and a v1.0.1 patch release for codegen precision fixes already on
`main`.

## Context

- Repo at `e0fefa7`, v1.0.0 released, spec v2.40.0 (no drift)
- CONTRIBUTING.md (236 lines) has architecture tour but lacks "good-first-issue" guidance
- 10 examples exist (6 mock, 4 live) — no error-handling or OAuth2 example
- 3 bug-fix commits since v1.0.0 (`f6c0422`) are in `[Unreleased]` — ready to ship
- No regressions found; fixes are type-precision corrections

## Task Breakdown

| ID | Task | Role | Size | Depends |
|----|------|------|------|---------|
| **Community** |||||
| C1 | Add "good first issue" section + labels guidance to CONTRIBUTING.md | docs | S | — |
| C2 | Post Discussions welcome announcement | docs | S | C1 |
| C3 | Commit + push | release | S | C1,C2 |
| **Examples** |||||
| E1 | Add `examples/mock/error-handling.go` — retry, error types, graceful degradation | backend | S | — |
| E2 | Add `examples/live/oauth2-flow.go` — OAuth2 token acquisition + refresh | backend | S | — |
| E3 | Update `examples/README.md` with new entries | docs | S | E1,E2 |
| E4 | Commit + push | release | S | E1,E2,E3 |
| **v1.0.1 Release** |||||
| V1 | Update CHANGELOG: move `[Unreleased]` → `[1.0.1]` with compare links | docs | S | — |
| V2 | Tag `v1.0.1` + push tag to both remotes | release | S | V1 |
| V3 | Patch GitHub release body from CHANGELOG | release | S | V2 |
| V4 | Create Gitee release entry (manual — no CLI/token) | manual | — | V2 |

## Execution Order

```
V1+V2 FIRST (tag existing HEAD before other commits land)
  → C1+C2+E1+E2+E3 in parallel
    → C3+E4+V3+V4 (commit artifacts, patch release body)
```

## Risks

1. **Tagging on shared branch** — other tracks push after V1 but before V2.
   Mitigation: execute V1+V2 immediately and sequentially.
2. **Discussions seeding** — use `scripts/seed_discussions.go` (if it exists)
   or post manually via GitHub web UI.
3. **OAuth2 example** — needs live paper account; verify against mock first.

## Verification

- `go build ./...` and `go vet ./...` after examples
- `make docs-check` after CONTRIBUTING.md changes
- `git tag -l v1.0.1` confirms tag exists
- `gh release view v1.0.1` confirms release body
