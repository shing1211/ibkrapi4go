# Design 09 — Orders and confirmation

Order submission is the highest-risk part of the SDK. This document specifies the
flow and safety rules.

## IBKR order flow

1. **Submit** — `POST /v1/api/iserver/account/{accountId}/orders`.
   IBKR may respond `200` with one or more **reply items** that require
   confirmation (e.g. price-cap warnings, order-size confirmations).
2. **Confirm** — `POST /v1/api/iserver/reply/{replyId}` with the reply id set.
   A submission can take multiple confirmation rounds.
3. **Result** — once accepted, the order has an `order_id`; status is queried via
   `GET /v1/api/iserver/account/order/status/{orderId}`.
4. **Modify / Cancel** — `POST .../order/{orderId}` / `DELETE .../order/{orderId}`.

## Safety rules

- **No automatic retry** of submit/modify/cancel/confirm
  ([../adr/0009](../adr/0009-no-auto-retry-orders.md)).
- On an **ambiguous** outcome (timeout), the SDK does **not** resubmit. It returns
  an error directing the caller to reconcile via open orders / order status.
- Confirmation is **explicit**: the manager does not silently auto-confirm
  warnings. The caller decides.

## Public API (planned)

```go
type OrderRequest struct {
    ConID      ConID
    Side       Side        // Buy | Sell
    Quantity   string      // never float64
    OrderType  OrderType   // Market | Limit | Stop | ...
    LimitPrice string      // empty for market orders
    TimeInForce TimeInForce
    // ...
}

type SubmitResult struct {
    OrderID   string
    Status    string
    Replies   []Reply     // non-empty => confirmation required
}

type Reply struct {
    ID      string
    Message string
    // ...
}

// Submit places an order. If Result.Replies is non-empty the order is NOT yet
// accepted; call Confirm with the desired reply ids.
func (m *TradeManager) Submit(ctx context.Context, account AccountID, req OrderRequest) (*SubmitResult, error)

// Confirm sends the reply id(s) to advance a pending order.
func (m *TradeManager) Confirm(ctx context.Context, replyID string, confirmed bool) (*SubmitResult, error)
```

## Idempotency warning

IBKR offers no idempotency key. Two submits of the same `OrderRequest` create two
orders. The SDK documents this prominently and never retries.

## Reconciliation

After an ambiguous submit, callers should:

```go
orders, err := cli.Trade().OpenOrders(ctx)
// check whether the intended order exists before resubmitting
```

The SDK may provide a helper `Trade().Reconcile(ctx, account, fingerprint)`.

## Error mapping

Implemented in `pkg/ibkr/trade.go`:

| Outcome | Error |
|---------|-------|
| Rejected by IBKR | `*ibkr.Error` wrapping `ErrOrderRejected` (with code/message) |
| Requires confirmation | returned in `SubmitResult.Replies` (not an error) |
| Ambiguous timeout | `*ibkr.Error` with `Code == "ambiguous"`; reconcile via `OpenOrders` |

The wire body is built by hand (`orderTicketJSON`) with money/quantity as
strings, because the generated request type marshals them as `float32`.

## Tests

- Submit returning a reply → assert no order id and that `Confirm` is required.
- Submit timeout → assert exactly one attempt and an ambiguous error.
- Modify/cancel → assert exactly one attempt, no retry on 5xx.
- Money fields in `OrderRequest` are strings.
