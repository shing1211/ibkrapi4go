# todos.md — Spec Drift Watch

- **Run:** `2026-09-21-spec-drift-watch`
- **Mode:** BUILD

| ID | Task | Role | Status | Depends On | Acceptance |
|----|------|------|--------|-----------|-----------|
| S1 | Write `scripts/check_spec_version.py` | backend | todo | — | Script fetches spec, prints version, exits 0=current/1=drifted |
| S2 | Add `.github/workflows/spec-drift.yml` | devops | todo | S1 | Workflow on schedule; creates issue on drift |
| S3 | Verify script against live spec | tester | todo | S1 | Script runs; output verified |
| S4 | Commit + push to both remotes | release | todo | S2,S3 | Both remotes at same commit |
