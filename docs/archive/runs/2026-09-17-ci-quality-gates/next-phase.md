# Next Phase: Docs Website

## Context

The 2026-09-17 ci-quality-gates run (`47c354b`) shipped:
- Coverage badge in CI + README (all 6 translations)
- Pre-commit CI gate (`.github/workflows/pre-commit.yml`)
- `FUNDING.yml` with GitHub Sponsors link
- GitHub Discussions: **requires manual action** by repo owner in GitHub Settings → Features

**Remaining backlog (ROADMAP.md):** docs website.

## Recommended next phase: Docs Website

### Why a docs website?
The project has rich documentation (`docs/` with 20+ files: SPEC.md, ARCHITECTURE.md, AUTH.md, SESSIONS.md, ERRORS.md, LOGGING.md, OBSERVABILITY.md, MOCK-GATEWAY.md, RATE-LIMITING.md, STREAMING.md, ADR 0001–0014, design contracts, etc.) but they live in the repo and are only accessible via GitHub's web UI. A docs site makes the SDK more accessible to new users and provides a branded home page.

### Options (choose simplest)

| Option | Approach | Pros | Cons |
|--------|----------|------|------|
| A | `docs/` served via GitHub Pages from `gh-pages` branch | Zero cost; uses existing docs | Clunky URLs (`/ibkrapi4go/docs/...`); no search |
| B | VitePress / Hugo / MkDocs on GitHub Pages | Professional; fast; search | New dependency; requires theme/styling |
| C | GitBook / ReadMe.io (free for open source) | Hosted; collaborative | External service; requires account |
| D | Docusaurus | Popular in OSS; good i18n | Heavy; steeper setup |

### Recommended: Option A (GitHub Pages from `gh-pages`)
Rationale: zero new dependencies, zero cost, minimal maintenance. The `docs/` folder structure is already clean and link-resolved. A `gh-pages` branch with `index.html` redirect or a lightweight static site generator would suffice.

### Scope for this phase
1. Create `docs/` navigation (`_sidebar.md` or similar) if using a static generator
2. Set up `gh-pages` branch with redirect or minimal static site
3. Enable GitHub Pages in repo settings (requires repo owner — manual step)
4. Add docs site link to README (top badge area)

### What's NOT in this phase
- Full VitePress/Hugo setup (too heavy for the current doc volume)
- Multi-language docs site (translations are in README files; site would need i18n plugin)
- Blog / changelog page

## Risks
- GitHub Pages setup requires repo owner to enable it in settings (manual).
- Option A (redirect) is minimal; a proper static site would be better long-term.

## Task breakdown (draft)

| ID | Objective | Role |
|----|-----------|------|
| M1 | GitHub Pages setup: create `gh-pages` branch or static site | devops |
| M2 | Add docs site link to README badges | docs |
| M3 | Docs sync + release | docs + release |
| M4 | Next-phase planning | planner |
| M5 | Close-out report + index | orchestrator |
