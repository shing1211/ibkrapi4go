# Todos: Docs Website

Run: `docs/runs/2026-09-17-docs-website/`
Plan: `plan.md` · Status legend: `todo` · `doing` · `done`

| ID | Task | Role | Status | Depends On | Acceptance |
|----|------|------|--------|------------|------------|
| W1 | `docs/index.html`: branded landing page (HTML/CSS only) | devops | done | — | valid HTML, browser renders, no external deps |
| W2 | README: add GitHub Pages badge to all 6 translations | docs | done | W1 | badge in all 6 README files |
| W3 | Docs sync: CHANGELOG, ROADMAP | docs | doing | W2 | `make docs-check` pass |
| W4 | Release: commit + push | release | todo | W3 | both remotes at new commit |
| W5 | Next-phase planning | planner | todo | W4 | `next-phase.md` |
| W6 | Close-out report + index | orchestrator | todo | W5 | `report.md` + index line |

## Manual step (out of scope)
`Settings → Pages → Source: main branch, /docs folder` (repo owner only)

## Notes
- Base commit: `919b576`
- Pure HTML/CSS only — no JS, no external CDN, no new deps
- GitHub Pages renders .md files natively — no build step needed
