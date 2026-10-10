# Phase D — Feature Gaps (P2) — Detailed Plan

- **Run:** `docs/runs/2026-09-23-blueprint-hardening/`
- **Depends on:** Phase C (C1 ws tests) for D1/D3
- **Purpose:** Close the blueprint capability gaps: order state machine +
  duplicate protection, bracket/OCA/multi-leg, typed WS events, gap detection,
  delayed-data/permissions, ClientOrderID, and account/portfolio streaming.

**Sequencing:** D1 → D2 · D3 → D4 → D7 · D5, D6 independent.
**Breaking-change rule:** add fields/APIs; do not break existing signatures
without explicit approval (else a v2 bump).

---

## D1 — Order state machine + duplicate-submission protection

- **Role:** architect · **Size:** L · **Deps:** C1 · **Wave:** 4

**Objective:** No `OrderState` type/transitions exist; only
`SubmitResult.Accepted()` (`pkg/ibkr/trade.go:82`). No client-order-ID registry,
so a retried/duplicated submit can create two orders (ADR 0009 relies on
no-retry, but the blueprint asks for explicit modeling).

**Files:**
- Read: `pkg/ibkr/trade.go`, `docs/adr/0009-no-auto-retry-orders.md`,
  `docs/design/09-orders-and-confirmation.md`
- Create: `pkg/ibkr/orderstate.go` (+ transitions + `cOID` registry)
- Modify: `pkg/ibkr/trade.go` (submit path consults registry; records cOID)

**Approach:** Define an explicit `OrderState` enum + legal transitions
(Submitted → Accepted → Filled/Partial/Cancelled/Rejected). Maintain a
client-side cOID registry to reject duplicate submits (return existing order
reference). Keep additive; do not change `Submit` signature without approval.

**Acceptance:** State transitions validated; duplicate submit with same cOID is
rejected/deduped; tests cover transitions + dup protection.

**Verification:** `go test -race ./pkg/ibkr/...`

---

## D2 — Bracket/OCA/conditional + multi-leg orders + TIF validation

- **Role:** backend · **Size:** L · **Deps:** D1 · **Wave:** 4

**Objective:** Generated client supports parent/child order structures
(`client/client.gen.go`), but public `OrderRequest` (`pkg/ibkr/trade.go:20-41`)
and `orderTicketJSON` (`:486-498`) have no `ParentID`/`IsSingleGroup`; TIF
constants exist (`ids.go:55-61`) with no validation; no multi-leg.

**Files:**
- Read: `pkg/ibkr/trade.go`, `pkg/ibkr/builders.go`, `pkg/ibkr/ids.go`
- Modify: `pkg/ibkr/trade.go`, `pkg/ibkr/builders.go`, `pkg/ibkr/orderstate.go`

**Approach:** Add additive fields/methods for bracket (parent + TP/SL children),
OCA groups, conditional orders, and multi-leg (combo) orders. Add TIF validation
in the builder/validator.

**Acceptance:** New order flows build valid tickets and are tested against the
mock gateway; existing single-leg flow unchanged.

**Verification:** `go test -race ./pkg/ibkr/...`

**Open:** additive-only, or may we break `OrderRequest`? (plan §Open Questions)

---

## D3 — Typed order/execution/portfolio WS events

- **Role:** backend · **Size:** M/L · **Deps:** C1 · **Wave:** 4

**Objective:** `ntf`/`sor`/`usr` frames are raw `[]byte`
(`internal/ws.go:448-493`, `pkg/ibkr/ws.go:42-53`); no typed order/execution/
portfolio events.

**Files:**
- Read: `internal/ws.go` (`parseSystemFrame`, `dispatch`), `pkg/ibkr/ws.go`,
  `internal/mockgateway/fixtures.go`
- Create: typed event types (`pkg/ibkr/events.go`)
- Modify: `internal/ws.go`, `pkg/ibkr/ws.go`

**Approach:** Define typed event structs for order status, executions, and
portfolio/account updates; parse in `internal/ws.go`; expose via `Subscription`
channels. Keep the existing raw accessor for compatibility.

**Acceptance:** Typed events delivered + tested (incl. malformed frames → no panic).

**Verification:** `go test -race ./internal/... ./pkg/ibkr/...`

---

## D4 — WS sequence/gap detection

- **Role:** backend · **Size:** M · **Deps:** D3 · **Wave:** 4

**Objective:** No sequence/gap detection; explicitly delegated to the caller
(`docs/design/05-streaming.md:58-59`, `docs/STREAMING.md:93-94`).

**Files:**
- Modify: `internal/ws.go`, `pkg/ibkr/ws.go` (expose gap signal)

**Approach:** Track the server sequence/`_updated` where available; emit a gap
signal (error/channel) when a discontinuity is detected; trigger reconciliation
hook.

**Acceptance:** Injected gap produces a signal; test added.

**Verification:** `go test -race ./internal/... ./pkg/ibkr/...`

---

## D5 — Delayed-data flags + market-data permissions

- **Role:** backend · **Size:** M · **Deps:** — · **Wave:** 4

**Objective:** Snapshot (`pkg/ibkr/marketdata.go:74-99`) and streaming
(`pkg/ibkr/ws.go:250-311`) don't surface delayed-data flags or permission/
entitlement status.

**Files:**
- Read: `pkg/ibkr/marketdata.go`, `pkg/ibkr/ws.go`, `client/client.gen.go`
  (snapshot/field schemas)
- Modify: `pkg/ibkr/marketdata.go`, `pkg/ibkr/ws.go`

**Approach:** Add delayed-data + permission fields to `Snapshot`/`Update` (or a
companion status type). Parse from the generated response where present.

**Acceptance:** Fields exposed and populated in tests.

**Verification:** `go test ./pkg/ibkr/...`

---

## D6 — `ClientOrderID` round-trip

- **Role:** backend · **Size:** S · **Deps:** — · **Wave:** 4

**Objective:** `cOID` is sent (`pkg/ibkr/trade.go:496`) but not parsed back;
`Order`/`OrderStatus` (`:84-116`) lack `ClientOrderID`, and there's no
X-request-id ↔ order linkage.

**Files:**
- Modify: `pkg/ibkr/trade.go` (add field + parse from responses)

**Approach:** Add `ClientOrderID` to `Order`/`OrderStatus`; populate from the
broker response.

**Acceptance:** Field populated in tests.

**Verification:** `go test ./pkg/ibkr/...`

---

## D7 — Account/portfolio streaming (NEW — user decision 2026-09-23)

- **Role:** backend · **Size:** M/L · **Deps:** D3 · **Wave:** 4

**Objective:** The public API only supports market-data streaming
(`Subscribe(conids, fields)`); account/portfolio/order-push streaming is not
exposed. The blueprint requires normalized account, order, execution, and
portfolio events. **Implement** (not defer).

**Files:**
- Read: `pkg/ibkr/ws.go`, `internal/ws.go`, `docs/STREAMING.md`,
  `internal/mockgateway/stream.go`, `internal/mockgateway/fixtures.go`
- Modify: `pkg/ibkr/ws.go` (new subscribe API for account/order/portfolio
  channels), `internal/ws.go` (subscription registry + dispatch)
- Modify/Create: `internal/mockgateway/stream.go` (emit account/portfolio frames
  for tests)

**Approach:** Add a subscription API (e.g. `SubscribeAccount`, `SubscribeOrders`,
`SubscribePortfolio` or a unified `SubscribeEvents`) that registers IBKR account
update requests and routes typed events (from D3) to the caller. Update the mock
gateway to emit deterministic account/portfolio/order frames so this is testable
without a live account.

**Acceptance:** Account/portfolio/order events stream as typed events from the
mock gateway; unsubscribe/reconnect works; tests under `-race`.

**Verification:** `go test -race ./internal/... ./pkg/ibkr/...`

**Docs:** followed by E8.

---

## Phase D Gate

All new features tested under `-race`; no breaking change to existing exported
signatures without approval; `docs/STREAMING.md` + examples updated (E7/E8).
