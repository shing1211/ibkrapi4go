# Reconciliation Plan — Docs vs. Code

- **Run:** `docs/runs/2026-09-21-docs-reconciliation/`
- **Date:** 2026-09-21
- **Mode:** BUILD

## Goal

Audit and fix all discrepancies between markdown documentation and the actual
codebase. Focus on version references, feature claims, and package descriptions.

## Findings (from prior audit)

| Area | Documentation Says | Actual State | Severity |
|------|-------------------|--------------|----------|
| `pkg/ibkr/doc.go` Version | `v0.1.0` | v1.0.0 (latest tag) | High |
| README.md release row | `v0.2.0` | v1.0.0 (latest tag) | High |
| docs/ROADMAP.md latest | `v0.2.0` | v1.0.0 (latest tag) | High |
| README translations | `v0.1.1` (last sync) | v1.0.0 (latest tag) | High |
| `doc.go` description | "v1 targets CPAPI surface only" | Both CPAPI + IB REST wrapped | Medium |

## Assumptions

1. v1.0.0 is the correct latest release tag.
2. All code fixes from `2026-09-21-ibkr-reconciliation` are already on `main`.
3. The "docs reconciliation" run is purely documentation — no code changes.

## Approach

**Single-pass audit + parallel fixes.** One reviewer sub-agent audits all docs
for version references and stale claims. One docs sub-agent then applies all
corrections in parallel across the affected files.

## Task Breakdown

| ID | Objective | Role | Depends | Size | Acceptance |
|----|-----------|------|---------|------|-----------|
| A1 | Audit all .md files for version/stale references; produce a fix list | reviewer | — | M | Fix list with file:line:issue per item |
| A2 | Update `pkg/ibkr/doc.go` Version constant to `v1.0.0` | backend | A1 | S | Constant = "v1.0.0"; doc comment updated |
| A3 | Update `README.md` release row + any version badges to v1.0.0 | docs | A1 | S | README reflects v1.0.0 |
| A4 | Update `docs/ROADMAP.md` latest release to v1.0.0 | docs | A1 | S | ROADMAP reflects v1.0.0 |
| A5 | Update all 5 README translations to match English v1.0.0 | docs | A3 | S | Translations consistent |
| A6 | Verify `docs/STABILITY.md` content matches current API surface | reviewer | A2 | S | STABILITY doc accurate |
| A7 | Verify all internal doc links resolve | tester | A3,A4,A5 | S | No broken links |
| A8 | Commit + push to both remotes | release | A2,A3,A4,A5,A6,A7 | S | Both remotes at same commit |

## Risks

1. Translation files may have other drift beyond version — flagged by A1.
2. `doc.go` package comment may need rewriting (mentions "v1 targets CPAPI only").
3. Run index update must happen after push.

## Order of Work

A1 (sequential, feeds all others) → A2–A7 (parallel) → A8 (sequential)
