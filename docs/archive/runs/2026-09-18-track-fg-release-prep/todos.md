# Tracks F + G — Release Prep & Tag v0.2.0

| ID | Task | Role | Status | Depends On | Acceptance |
|----|------|------|--------|------------|------------|
| F1 | Update `docs/RELEASING.md` | docs | done | — | No stale refs |
| F2 | Create `docs/STABILITY.md` | docs | done | — | File exists, links to ADR 0015 |
| G1 | Update `CHANGELOG.md` — v0.2.0 entry | docs | done | — | All changes since v0.1.1 captured |
| G2 | Update version refs across docs | docs | done | — | No stale v0.1.1 refs in docs/ |
| G3 | Tag v0.2.0 + push to both remotes | release | done | F1, F2, G1, G2 | Both remotes updated, tag exists |
