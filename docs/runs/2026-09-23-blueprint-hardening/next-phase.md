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
| Windows WS `Close` deadlock | Resolved in v1.0.4 | D4 left `lastUpdated` nil; the first sequence frame panicked while holding `WSConn.mu`, and close then blocked. Sequence tracking, owned I/O cancellation, force-close, and late-reconnect cleanup were added. |
| `check_money.py` false positives | Resolved in v1.0.4 | The scanner now checks exported fields on exported structs in hand-written code and ignores generated code, unexported adapters, and function bodies. |
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

**Status:** Done in `v1.0.4`. The scanner now checks exported fields on
exported structs in hand-written `pkg/ibkr` and `internal`, skips generated
client code, unexported adapters, and function bodies, and keeps explicit
non-money exceptions for observability aggregates and the legacy bank
instruction ID. `python scripts/check_money.py` passes.

### P3 — Public OAuth2 token-refresh surface

**Status:** Done. `RESTSurface.ForceRefresh(ctx)` and `Invalidate()` are
available with automatic refresh preserved, refresh-token rotation retained,
and stale in-flight results prevented from repopulating the cache.

### P4 — Unified streaming event API
**Objective:** Add a single `SubscribeEvents` (or manager-scoped) entry point
that returns all typed events (account, portfolio, order, notification, user)
plus a documented `execution` (`str`) event if the gateway emits it.
**Why now:** D3/D7 expose typed channels per subscription; a unified surface is
more ergonomic for event-driven consumers.
**Effort:** M
**Dependencies:** D3, D7.
**Risks:** Additive; must preserve existing per-channel behavior.

## 4. Completed Follow-Up

P1 was completed in the `2026-09-24-ws-shutdown` run. The confirmed failure was
a D4 regression rather than a base-commit lock-ordering defect. The follow-up
fixed sequence-map initialization, made close interrupt active I/O, prevented
late reconnect publication, and added Windows-oriented regression tests.

The next recommended work is P4: add a unified typed streaming event entry
point.

## 5. Open Questions

1. **Run index location:** should `docs/runs/index.md` become the single index
   (with archived runs linked from it), or keep the archive index authoritative?
2. **`money-check` scope:** should it also assert on `client/` generated types,
   or only on hand-written `pkg/ibkr`/`internal`?
