# Report: WebSocket Shutdown Reliability

- **Run:** `docs/runs/2026-09-24-ws-shutdown/`
- **Base commit:** `945182e`
- **Feature commit:** pending release commit
- **Close-out commit:** pending release commit
- **Status:** complete

## Summary

Fixed the WebSocket shutdown hang introduced by D4 and hardened the connection
lifecycle. The change is internal and additive; no exported signature changed.

## Changes

- Initialize `lastUpdated` when constructing `WSConn`.
- Move sequence tracking into `recordSequence` with guaranteed mutex release.
- Preserve duplicate/out-of-order sequence semantics and `WSGapError`.
- Give each WebSocket connection an owned cancelable context.
- Make explicit close cancel I/O and force-close the socket.
- Discard a newly dialed connection if close began while dialing.
- Reserve market-data field `6509` consistently.
- Add silent-peer, sequence, public-close, and repeated-run regressions.
- Corrected the prior run report: the failure was a D4 regression, not a
  base-commit lock cycle.
- Tightened `scripts/check_money.py` to inspect exported public struct fields
  and ignore generated code, unexported adapters, and function bodies.

## Verification

- `go test ./... -count=1 -timeout=15m` — pass
- Focused WebSocket tests with `-count=20` — pass
- `go vet ./...` — pass
- `gofmt` and `git diff --check` — pass
- Local `-race` verification unavailable: Go reports `CGO_ENABLED=0`, and no C
  compiler is installed in the Windows environment.

## Follow-up

P2 remains the next recommended maintenance phase: make `money-check` precise
and green. P3 (public token refresh) and P4 (unified streaming events) remain
backlog items.
