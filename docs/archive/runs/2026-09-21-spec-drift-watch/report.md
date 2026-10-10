# Run Report — 2026-09-21-spec-drift-watch

## Mode: BUILD

## Summary

Implemented a spec drift watch: a Python script that compares the live IBKR
OpenAPI spec version against the pinned v2.40.0, and a GitHub Actions workflow
that runs it weekly (Monday 9am UTC) and opens a GitHub issue when drift is
detected.

---

## Shipped

| ID | Task | Outcome |
|----|------|---------|
| S1 | Write `scripts/check_spec_version.py` | Done — stdlib only, exits 0/1/2, verified against live spec |
| S2 | Add `.github/workflows/spec-drift.yml` | Done — weekly + manual trigger, creates issue on drift |
| S3 | Verify script against live spec | Done — live spec v2.40.0 matches pinned; exit 0 |
| S4 | Commit + push | Done — both remotes updated |

---

## Files Changed (2) + Added (4 artifacts)

| File | Change |
|------|--------|
| `.github/workflows/spec-drift.yml` | Improved: weekly schedule, `issues:write` permission, fixed output parsing, `::error` on exit 2 |
| `scripts/check_spec_version.py` | Confirmed correct (stdlib, urllib, exit 0/1/2) |
| `docs/runs/2026-09-21-spec-drift-watch/plan.md` | New |
| `docs/runs/2026-09-21-spec-drift-watch/todos.md` | New |

---

## Verification

- `python scripts/check_spec_version.py` → `Live version: 2.40.0`, `Pinned version: v2.40.0`, `Status: OK`, exit 0 ✅
- Network error → exit 2 ✅
- Workflow file: 74 lines, no syntax errors, `issues:write` permission, weekly+cron triggers

---

## Commits

| Commit | Description |
|--------|-------------|
| `6e2be91` | feat(spec): improve spec-drift workflow and script |

Pushed to `origin/main` and `gitee/main`.
