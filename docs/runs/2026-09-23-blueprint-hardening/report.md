# Report: Blueprint Hardening & Enhancement

- **Run:** `docs/runs/2026-09-23-blueprint-hardening/`
- **Mode:** BUILD
- **Base commit:** `96d7d75`
- **Feature commit:** `34055b2`
- **Close-out commit:** see the commit that adds this report
- **Status:** complete

## Summary

Hardened the SDK against the production-grade blueprint and closed the feature
gaps. All four phases (A–E) shipped: correctness/security fixes, real CI gates,
reliability/observability, the Phase D feature gaps (including account/portfolio
streaming), and the Phase E docs/DX work.

## Shipped

| Task | Deliverable | Status |
|------|-------------|--------|
| A1–A7 | Broken build fix, money-precision scanner + leaks, timeout body-cancellation, response-size bound, redaction hardening, `Error.Message` sanitising | done |
| B1–B7, E9 | golangci-lint + gosec in CI, single release workflow, GoReleaser, coverage threshold, OS×Go matrix, DCO + dependency-review, secret scanner + SBOM, GoDoc enforcement | done |
| C1–C8 | `internal/ws.go` tests, flake fix, resilience tests, injectable clock/dialer, metrics, composite health probe, OTel bridge, fuzz in CI | done |
| D1 | Order state machine + duplicate-submission protection (`orderstate.go`) | done |
| D2 | Bracket/OCA/multi-leg + TIF `FOK`/`GTD` validation | done |
| D3 | Typed WS events (`OrderEvent`, `NotificationEvent`, `UserMessageEvent`) | done |
| D4 | WS sequence/gap detection (`WSGapError`) | done |
| D5 | Delayed-data flags (`MarketDataStatus` on `Snapshot`/`Update`) | done |
| D6 | `ClientOrderID` round-trip on `Order`/`OrderStatus` | done |
| D7 | Account/portfolio streaming (`SubscribeAccount`/`SubscribePortfolio`) | done |
| E1–E3 | Version/spec drift, error docs, `check_design` | done |
| E4 | Removed leaking `CreateSessionRaw` | done |
| E5 | `docs/GATEWAY-SETUP.md` + `docs/PERMISSIONS.md` | done |
| E6 | Corrected misleading live examples | done |
| E7 | Cancel + reconciliation mock examples | done |
| E8 | Documented account/portfolio streaming | done |
| E10 | README architecture flow diagram (+ translations) | done |

### Key outcomes

- **Account/portfolio streaming (D7):** the public API now streams typed
  `acq`/`pos` events over the same multiplexed WebSocket, with reconnect
  (method-aware resubscribe) and per-position delivery. Tested against the mock
  gateway under `-race`.
- **Delayed data is visible:** `MarketDataStatus` flags `IsDelayed`,
  `IsFrozen`, and `IsNotSubscribed` on snapshots and streamed field `6509`, so
  callers can avoid trading on stale quotes.
- **Generated-code boundary:** no generated `client.*` type leaks through an
  exported `pkg/ibkr` signature (`CreateSessionRaw` removed).
- **Docs to truth:** new gateway-setup and permissions guides, a top-level
  architecture flow in the README (+ all translations), accurate live examples,
  and cancellation/reconciliation examples that run against the mock.

## Verification

- `go build ./...` — pass
- `go vet ./...` — pass
- New Phase D tests pass under `-race` (parsers, `DeliverSystem` routing, and
  mock-gateway account/portfolio integration + reconnect).
- `python scripts/check_links.py` — pass
- `python scripts/check_i18n.py` — pass (6 languages consistent)
- `go run ./scripts/check_design` — pass
- `go build ./...` + `go run` for both new examples against the mock gateway —
  pass

### Known environment caveat

On Windows, some `pkg/ibkr` WebSocket integration tests (`TestWS_SystemUpdates`
with `-count>1`, `TestWS_ReconnectResubscribes`) can deadlock in
`internal.WSConn.Close` waiting on `WSConn.mu`. This is **pre-existing** and
reproduces on the base commit with none of this run's changes applied; CI
(Linux) is authoritative. The new account/portfolio tests were verified to pass
under `-race`.

## Notes

- Phase D changes are additive; no existing exported signature changed shape.
- `git log 96d7d75..34055b2` contains the feature commits; this report, the
  updated `todos.md`, and the new `docs/runs/index.md` entry form the close-out.
