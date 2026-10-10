# Phase B — CI/CD & Release (P1) — Detailed Plan

- **Run:** `docs/runs/2026-09-23-blueprint-hardening/`
- **Depends on:** Phase A (A1 fixes the build so CI can be green)
- **Purpose:** Make the CI gates real (lint, coverage, matrix) and ship proper
  release artifacts (binaries, checksums, provenance, SBOM).

---

## B1 — Run `golangci-lint` in CI + enable `gosec`

- **Role:** devops · **Size:** S · **Deps:** A1 · **Wave:** 1

**Objective:** `.golangci.yml` exists with `errcheck, govet, staticcheck, unused,
ineffassign, misspell, gofmt, goimports, bodyclose, contextcheck, errorlint,
nolintlint, revive, unconvert` — but **no workflow or Make target runs it**, and
`gosec` is absent.

**Files:**
- Modify: `.golangci.yml` (add `gosec`), `.github/workflows/ci.yml` (add lint job/step),
  `Makefile` (add `lint` target)

**Approach:** Add a `lint` job using `golangci/golangci-lint-action` pinned to a
version; enable `gosec`. Keep `client/*.gen.go` excluded.

**Acceptance:** CI fails on an injected lint error; passes on clean tree.

**Verification:** run `golangci-lint run` locally (or via action) on clean tree.

---

## B2 — Consolidate duplicate release workflows

- **Role:** devops · **Size:** S · **Deps:** — · **Wave:** 1

**Objective:** Both `release.yml` and `release-automation.yml` fire on `v*` and
both create a GitHub Release + push Gitee → race/duplicates.

**Files:**
- Read: `.github/workflows/release.yml`, `.github/workflows/release-automation.yml`
- Modify: merge into one workflow (keep semver tag validation + changelog +
  release + Gitee push in a single pipeline), delete the other.

**Acceptance:** Exactly one workflow runs on a `v*` tag push.

**Verification:** review workflow triggers; dry-run via `workflow_dispatch` if available.

---

## B3 — GoReleaser multi-arch + checksums + provenance

- **Role:** devops · **Size:** M · **Deps:** B2 · **Wave:** 1

**Objective:** No `.goreleaser.yml` exists; no cross-compilation; no checksums or
provenance. `CHANGELOG.md` claims "SLSA-level provenance generation" that is not
implemented.

**Files:**
- Create: `.goreleaser.yml`
- Modify: the consolidated release workflow (B2), `CHANGELOG.md` (correct claim)
- Read: `cmd/ibkr/`, `cmd/ibkr-mock-gateway/` (binary targets)

**Approach:** Build `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`,
`windows/amd64`; emit `checksums.txt`; add `actions/attest-build-provenance` (or
cosign) for signed provenance. Attach artifacts to the GitHub Release.

**Acceptance:** A tag produces downloadable binaries + checksums + provenance
attestation.

**Verification:** GoReleaser config validates (`goreleaser check`); snapshot build.

---

## B4 — Coverage threshold + `-coverpkg`

- **Role:** devops · **Size:** S · **Deps:** A1 · **Wave:** 1

**Objective:** Coverage is uploaded to Codecov with `fail_ci_if_error: false`; no
threshold; `-coverpkg` missing so `internal/ws.go` isn't credited.

**Files:**
- Create/Modify: `.github/codecov.yml` (project + patch thresholds)
- Modify: `.github/workflows/ci.yml` (add `-coverpkg=./...`, set
  `fail_ci_if_error: true` or add a `make coverage-check` gate)

**Approach:** Start with a realistic floor (e.g. project 55–60%) and ratchet it
up as Phase C adds tests. Document the target (85%+ eventual).

**Acceptance:** CI fails when coverage drops below the configured floor.

**Verification:** `make` coverage target; codecov config lint.

---

## B5 — OS/Go-version test matrix

- **Role:** devops · **Size:** S · **Deps:** A1 · **Wave:** 1

**Objective:** All jobs are `ubuntu-latest` + Go 1.26, but `docs/RELEASING.md`
promises "latest two minor releases" on "Linux, macOS, Windows".

**Files:**
- Modify: `.github/workflows/ci.yml` (add `strategy.matrix`: `os: [ubuntu-latest,
  macos-latest, windows-latest]`, `go: ['1.25','1.26']` where supported)

**Acceptance:** Tests run green on all matrix cells (or documented exclusions).

**Verification:** matrix job output.

---

## B6 — DCO enforcement + dependency-review

- **Role:** devops · **Size:** S · **Deps:** — · **Wave:** 1

**Objective:** DCO sign-off is required (`CONTRIBUTING.md`, PR template) but not
enforced; no `dependency-review-action` on PRs.

**Files:**
- Create: `.github/workflows/dco.yml`, `.github/workflows/dependency-review.yml`

**Acceptance:** Unsigned commits fail the DCO check; new vulnerable deps are flagged.

**Verification:** workflow presence + dry PR behavior.

---

## B7 — Real secret scanner + CycloneDX SBOM

- **Role:** devops · **Size:** M · **Deps:** B6 · **Wave:** 1

**Objective:** Secret scanning is a hand-rolled regex grep; SBOM is a plain-text
dep listing (`scripts/sbom-gen.sh`), neither standard nor attached to releases.

**Files:**
- Modify: `.github/workflows/supply-chain.yml` (use `gitleaks`/`trufflehog` action;
  generate CycloneDX via `cyclonedx-gomod`), `scripts/sbom-gen.sh`
- Modify: release workflow to attach SBOM

**Acceptance:** Standard CycloneDX SBOM produced and attached; secret scanner
detects a seeded test secret.

**Verification:** run SBOM gen locally; scanner on a fake-secret fixture.

---

## E9 — Enforce GoDoc in CI

- **Role:** devops · **Size:** S · **Deps:** B1 · **Wave:** 1

**Objective:** `revive` `exported` rule is enabled but warning-only and never run.

**Files:**
- Modify: `.golangci.yml` (raise `revive` exported severity to error),
  `.github/workflows/ci.yml` (covered by B1 lint job)

**Acceptance:** An undocumented exported symbol fails CI.

**Verification:** injected undocumented export fails lint.

---

## Phase B Gate

One release workflow; lint + coverage + matrix gates active; a dry-run/snapshot
release produces binaries + checksums + SBOM. `make check` still green.
