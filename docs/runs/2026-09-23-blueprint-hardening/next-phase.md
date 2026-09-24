# Next Phase: Post-Hardening Directions

- **Run:** `2026-09-23-blueprint-hardening`
- **Date:** 2026-09-23
- **Base commit:** `96d7d75`

## 1. What Was Completed

All four phases shipped. Correctness/security defects are fixed (build, timeout
body cancellation, response-size bounds, redaction), CI gates are real
(golangci-lint + gosec, coverage threshold, OS×Go matrix, DCO, secret scan +
SBOM, GoDoc), reliability/observability landed (WS tests, resilience tests,
injectable clock, metrics, health probe, OTel bridge, fuzzing), the Phase D
feature gaps are closed (order state machine, bracket/OCA, typed WS events, gap
detection, delayed-data flags, `ClientOrderID` round-trip, account/portfolio
streaming), and the docs are back to truth (gateway/permissions guides, README
architecture diagram, accurate examples, streaming docs). See `report.md`.

## 2. Open / Near-Term Items

| Item | Severity | Notes |
|------|----------|-------|
| Windows WS `Close` deadlock | Medium | `internal.WSConn.Close` can block on `WSConn.mu` (contended with `readLoop`/`writeLoop`/`pingLoop`) on Windows. Pre-existing; reproduces on the base commit. CI (Linux) passes. Blocks reliable local Windows test runs. |
| `check_money.py` false positives | Medium | Flags exported fields on **unexported** adapter structs (e.g. `taxVoucherRaw.DivAmount *float32`, `rest_banking` raw structs) and function bodies containing `{`. `make money-check` is not green although public money types are strings (ADR 0008). |
| Spec drift watch | Low | Spec pinned at v2.40.0; no scheduled check for v2.41.0+. |
| No public token-refresh API | Low | `internal.TokenSource.ForceRefresh` is not exposed on `RESTSurface`; the OAuth2 example can only describe automatic refresh. |
| `docs/runs/` vs `docs/archive/runs/` | Low | New runs live in `docs/runs/`; older runs are archived. Confirm the long-term location and index ownership. |

## 3. Candidate Next Phases

### P1 — Fix the Windows WebSocket `Close` deadlock
**Objective:** Remove the lock-ordering hazard in `internal.WSConn.Close` /
`currentConn` so `go test ./...` is reliable on Windows.
**Why now:** It causes flaky/hanging local runs and hides real regressions for
Windows contributors.
**Effort:** M
**Dependencies:** Existing `internal/ws_resilience_test.go` + mock gateway.
**Risks:** WS lifecycle is subtle; changes must keep reconnect/gap semantics
intact and pass `-race` on all three OSes.

### P2 — Make `money-check` precise and green
**Objective:** Restrict `scripts/check_money.py` to exported struct fields of
exported types (skip unexported adapter structs and function bodies), then clear
any genuine remaining float money fields.
**Why now:** `make check` includes `money-check`; a noisy gate erodes trust and
masks real ADR 0008 violations.
**Effort:** S
**Dependencies:** A2/A3 (already done).
**Risks:** Must still fail on a seeded `Money float64` export (keep the A2 test).

### P3 — Public OAuth2 token-refresh surface
**Objective:** Expose an explicit `ForceRefresh`/`Invalidate` on `RESTSurface`
(wrapping `internal.TokenSource.ForceRefresh`) and update the OAuth2 example to
use it.
**Why now:** Callers occasionally need to force a rotation (credential change,
diagnostics); today it is unreachable.
**Effort:** S
**Dependencies:** C7 (OTel) not required.
**Risks:** Additive API; document that automatic refresh remains the default.

### P4 — Unified streaming event API
**Objective:** Add a single `SubscribeEvents` (or manager-scoped) entry point
that returns all typed events (account, portfolio, order, notification, user)
plus a documented `execution` (`str`) event if the gateway emits it.
**Why now:** D3/D7 expose typed channels per subscription; a unified surface is
more ergonomic for event-driven consumers.
**Effort:** M
**Dependencies:** D3, D7.
**Risks:** Additive; must preserve existing per-channel behavior.

## 4. Recommended Next Phase

**Recommended: P1 (fix the Windows WebSocket `Close` deadlock).**

It is the only item that currently blocks a core verification command for a
supported OS. Fixing it restores confidence in `go test ./...` everywhere and
costs little, whereas P2–P4 are quality-of-life improvements that do not block
day-to-day work.

**Draft task breakdown:**

| ID | Objective | Role | Depends | Acceptance |
|----|-----------|------|---------|-----------|
| W1 | Reproduce the `Close`/`currentConn` lock cycle deterministically | tester | — | failing test on Windows |
| W2 | Fix lock ordering (pass the conn in, or snapshot under lock without re-locking) | backend | W1 | `go test -race ./...` stable on Windows |
| W3 | Run the full matrix locally (or in CI) | qa | W2 | Linux/macOS/Windows green |
| W4 | Docs/CHANGELOG sync + release | docs/release | W3 | both remotes synced |

## 5. Open Questions

1. **Run index location:** should `docs/runs/index.md` become the single index
   (with archived runs linked from it), or keep the archive index authoritative?
2. **`money-check` scope:** should it also assert on `client/` generated types,
   or only on hand-written `pkg/ibkr`/`internal`?
