# Run Report — 2026-09-21-docs-reconciliation

## Mode: BUILD

## Summary

Reconciled markdown documentation against the actual codebase. The repo had
advanced to v1.0.0 (and intermediate v0.2.0, v0.3.0 releases) but documentation
still referenced v0.1.0–v0.2.0. Fixed 11 files across version references, status
badges, and stale claims.

---

## Shipped

| ID | Task | Outcome |
|----|------|---------|
| A1 | Audit all .md files for version/stale references | Done — fix list produced (14 files, 7 categories) |
| A2 | `pkg/ibkr/doc.go` Version → v1.0.0 | Done — constant updated; package comment corrected |
| A3 | `README.md` v1.0.0 (badge + text + release row) | Done — stable badge, alpha removed, v1.0.0 |
| A4 | `docs/ROADMAP.md` latest release → v1.0.0 | Done |
| A5 | 5 README translations → v1.0.0 + sync banner | Done — all 5 updated |
| A6 | Verify `docs/STABILITY.md` accuracy | Done — Pre-1.0 rules block flagged as stale (handled in A6b) |
| A6b | Fix `docs/RELEASING.md` + `docs/ARCHITECTURE.md` | Done — alpha/pre-alpha removed |
| A7 | Fix `CHANGELOG.md` [Unreleased] compare link | Done — v0.3.0→v1.0.0 |
| A8 | Verify links + build/vet | Done — go build/vet pass; i18n consistent; no new broken links |

---

## Files Changed (11 total)

| File | Changes |
|------|---------|
| `pkg/ibkr/doc.go` | Version v0.1.0→v1.0.0; package comment now covers both CPAPI + IB REST |
| `README.md` | Stable badge (not alpha); alpha text removed; v0.2.0→v1.0.0 |
| `README.zh-Hans.md` | v0.2.0→v1.0.0; banner b7f2b81→81df4c0 |
| `README.zh-Hant.md` | v0.2.0→v1.0.0; banner b7f2b81→81df4c0 |
| `README.ja.md` | v0.2.0→v1.0.0; banner b7f2b81→81df4c0 |
| `README.ko.md` | v0.2.0→v1.0.0; banner b7f2b81→81df4c0 |
| `README.es.md` | v0.2.0→v1.0.0; banner b7f2b81→81df4c0 |
| `docs/ROADMAP.md` | Latest release v0.2.0→v1.0.0 |
| `docs/ARCHITECTURE.md` | Status Pre-alpha→Stable |
| `docs/RELEASING.md` | Alpha/until-v1.0.0 rules removed |
| `CHANGELOG.md` | [Unreleased] compare link v0.3.0→v1.0.0 |

---

## Verification

- `go build ./...` ✅
- `go vet ./...` ✅
- `check_i18n` ✅ (6 languages consistent)
- `check_links` ✅ (no new broken links introduced; pre-existing dangling links in
  `docs/runs/2026-09-20-next-15-phases/plan.md` are historical and unrelated)

---

## Commits

| Commit | Description |
|--------|-------------|
| `8af8404` | docs: reconcile docs to v1.0.0 — version, status, stability contract |

Pushed to `origin/main` and `gitee/main`.
