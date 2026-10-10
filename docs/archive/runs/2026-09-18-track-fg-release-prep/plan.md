# Tracks F + G — Release Prep & Tag v0.2.0

- **Date:** 2026-09-18
- **Mode:** BUILD
- **Status:** In Progress

## Goal

Update RELEASING.md, create STABILITY.md, write the v0.2.0 CHANGELOG entry, and tag + push v0.2.0 to both remotes.

## Task Breakdown

| ID | Task | Role | Size | Acceptance |
|----|------|------|------|------------|
| F1 | Update `docs/RELEASING.md` | docs | S | No stale refs |
| F2 | Create `docs/STABILITY.md` | docs | S | File exists, links to ADR 0015 |
| G1 | Update `CHANGELOG.md` — v0.2.0 entry | docs | M | All changes since v0.1.1 captured |
| G2 | Update version refs across docs | docs | S | No stale v0.1.1 refs in docs/ |
| G3 | Tag v0.2.0 + push to both remotes | release | S | Both remotes updated, tag exists |

## Verification

```bash
make check
grep -r 'v0.1.1' docs/ CHANGELOG.md
git tag -l | grep v0.2.0
```
