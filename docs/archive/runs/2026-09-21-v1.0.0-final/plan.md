# P5 — v1.0.0 Final Stabilization

- **Run:** `2026-09-21-v1.0.0-final`
- **Date:** 2026-09-21
- **Mode:** BUILD

## Goal

Final API audit, update CHANGELOG, tag v1.0.0, and push to both remotes.

## Tasks

| ID | Task | Status |
|----|------|--------|
| V1 | Update CHANGELOG.md with v1.0.0 entry | done |
| V2 | Run go build ./... && go vet ./... && go test | done |
| V3 | Tag v1.0.0 | done |
| V4 | Push to both remotes | done |

## What changed in v1.0.0

- Breaking changes from v0.3.0 (57 Get-prefix removals, float32→int64 banking IDs, 14 initialism fixes)
- Stability hardening (S1–S5): goroutine safety, backpressure, resilience, fault injection, performance baseline
- Ecosystem (E1–E5): CLI tool, migration guide, API reference site, release automation, supply-chain security
- Architectural (A1–A5): interface segregation, typed builders, middleware plugin, unified pagers, multi-client transport
- OpenTelemetry support (P2): contrib/otel package
- Live examples (P3): live-portfolio, options-chain, screener
- Community (P4): seeded discussions
