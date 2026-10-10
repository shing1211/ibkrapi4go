# todos.md — Docs Reconciliation

- **Run:** `2026-09-21-docs-reconciliation`
- **Mode:** BUILD

| ID | Task | Role | Status | Depends On | Acceptance |
|----|------|------|--------|-----------|-----------|
| A1 | Audit all .md files for version/stale references | reviewer | todo | — | Fix list with file:line:issue |
| A2 | Update `pkg/ibkr/doc.go` Version to v1.0.0 | backend | todo | A1 | Constant = "v1.0.0" |
| A3 | Update README.md release row to v1.0.0 | docs | todo | A1 | README reflects v1.0.0 |
| A4 | Update docs/ROADMAP.md latest release to v1.0.0 | docs | todo | A1 | ROADMAP reflects v1.0.0 |
| A5 | Update all 5 README translations to v1.0.0 | docs | todo | A3 | Translations match English |
| A6 | Verify docs/STABILITY.md content accuracy | reviewer | todo | A2 | STABILITY doc accurate |
| A7 | Verify all internal doc links resolve | tester | A3,A4,A5 | No broken links |
| A8 | Commit + push to both remotes | release | A2,A3,A4,A5,A6,A7 | Both remotes updated |
