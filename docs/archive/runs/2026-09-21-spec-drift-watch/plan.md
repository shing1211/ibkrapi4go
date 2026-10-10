# Plan — Spec Drift Watch

- **Run:** `docs/runs/2026-09-21-spec-drift-watch/`
- **Date:** 2026-09-21
- **Mode:** BUILD

## Goal

Add a scheduled CI job and script that detects when IBKR releases a new
OpenAPI spec version (beyond pinned v2.40.0) and reports it — without
auto-regenerating or auto-committing anything.

## Assumptions

1. The spec is fetched from IBKR's public OpenAPI endpoint at build time.
2. The pinned spec version is currently v2.40.0 (in `specs/ibkr_spec.json`).
3. No new dependencies may be added (per AGENTS.md hard rule 4).
4. CI is GitHub Actions; no changes to Gitee CI.

## Approach

| # | Approach | Description | Risk |
|---|----------|-------------|------|
| A | **Scheduled workflow** | GitHub Actions `workflow_dispatch` + `schedule: cron` runs a Python script that fetches the live spec version and compares to v2.40.0; opens a GitHub issue if newer | Low |
| B | **PR comment bot** | Same script, but posts a PR comment instead of opening an issue | Higher complexity |
| C | **CI-fail on drift** | Same script, but exits 1 (fails CI) if spec has advanced | Too noisy |

**Recommended: Approach A** — Scheduled workflow that opens a GitHub issue.
Low complexity, no new deps, actionable but non-blocking.

## Task Breakdown

| ID | Objective | Role | Depends | Size | Acceptance |
|----|-----------|------|---------|------|-----------|
| S1 | Write `scripts/check_spec_version.py` to fetch live spec version | backend | — | S | Script fetches spec, prints version, exits 0=current/1=drifted |
| S2 | Add `.github/workflows/spec-drift.yml` scheduled workflow | devops | S1 | S | Workflow runs on schedule, calls script, creates issue on drift |
| S3 | Verify script works against live spec | tester | S1 | S | Script runs; output verified |
| S4 | Commit + push to both remotes | release | S2,S3 | S | Both remotes updated |

## Order of Work

S1 (sequential) → S3 (parallel with S2) → S4 (sequential)
