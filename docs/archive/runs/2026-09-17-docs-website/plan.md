# Plan: Docs Website

- **Run:** `2026-09-17-docs-website`
- **Mode:** BUILD
- **Repo:** `github.com/shing1211/ibkrapi4go`
- **Base commit:** `919b576`

## Goal
Serve the SDK documentation at a branded GitHub Pages URL
(`https://shing1211.github.io/ibkrapi4go/`) from the existing `docs/` folder,
without adding new toolchain dependencies (no Node.js, no Python, no build step).

## Approach: GitHub Pages from `main:/docs`

**How it works:**
- GitHub Pages can serve directly from a branch/folder: `Settings → Pages → Source:
  main branch, `/docs (root)` folder`.
- `docs/index.html` becomes the landing page at the Pages root URL.
- All `docs/*.md` files become top-level pages (clean URLs).
- GitHub renders `.md` files as HTML automatically — no Jekyll needed, no build step.

**What to create:**
1. `docs/index.html` — branded landing page: SDK name, description, quick links to
   key docs (SPEC.md, README, ARCHITECTURE.md, MOCK-GATEWAY.md, OBSERVABILITY.md,
   AUTH.md), links to GitHub repo and issue tracker. Uses pure HTML/CSS (no JS,
   no external CDN — keeps it self-contained).
2. README badge: add GitHub Pages URL badge to all 6 README translations.

**What to update:**
- `docs/SPEC.md`, `docs/ARCHITECTURE.md`, etc. — add `docs/` prefix to relative
  links? No — GitHub Pages serves `docs/` as root, so all relative links within
  docs/*.md already work correctly (they're relative to `docs/`).

**GitHub Pages setup (manual — requires repo owner):**
`Settings → Pages → Source: Deploy from a branch → main branch, `/docs` folder → Save`.
URL will be: `https://shing1211.github.io/ibkrapi4go/`

## Scope constraints
- **ADR 004 compliance:** No new runtime or build-time dependencies. Pure HTML/CSS only.
- **No new CI workflows** — GitHub Pages serves the docs folder directly.
- **No build step** — GitHub renders `.md` files natively.
- **No Jekyll** — GitHub Pages handles markdown rendering without Jekyll processing.

## Task breakdown
| ID | Objective | Role | Depends | Acceptance |
|----|-----------|------|---------|-----------|
| W1 | `docs/index.html`: branded landing page (HTML/CSS only) | devops | — | file valid HTML; local browser renders correctly |
| W2 | README: add GitHub Pages badge to all 6 translations | docs | W1 | badge in README*.md files |
| W3 | Docs sync: CHANGELOG, ROADMAP | docs | W2 | `make docs-check` pass |
| W4 | Release: commit + push | release | W3 | both remotes at new commit |
| W5 | Next-phase planning | planner | W4 | `next-phase.md` |
| W6 | Close-out report + index | orchestrator | W5 | `report.md` + index line |

## Manual step (out of scope for automation — repo owner only)
Enable GitHub Pages: `Settings → Pages → Source: main branch, /docs folder`.

## Verification (global)
`make check` · `make test-race` · `make docs-check` · `make license-check`

## Notes
- `docs/` already has `make docs-check` verifying all internal links resolve
- GitHub Pages renders `.md` → HTML automatically
- `docs/index.html` takes priority over `docs/README.md` at the Pages root
