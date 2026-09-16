# 0009 — Never auto-retry order mutations

- Status: Accepted
- Date: 2026-09-16

## Context

Retries are desirable for transient network and `5xx`/`429` failures. But IBKR
order and instruction endpoints are **not idempotent**: resubmitting an order
after a timeout can create a **duplicate order** — a serious financial error.
IBKR does not provide an idempotency key to make order submission safe to retry.

An earlier plan proposed retrying `POST/PUT/DELETE` up to three times, which
would have been dangerous for order endpoints.

## Decision

- The retry layer retries **only safe methods** (`GET`/`HEAD`/`OPTIONS`).
- **Order and instruction mutations are never retried automatically**, regardless
  of HTTP method.
- On an ambiguous failure (e.g. timeout after sending an order), the SDK returns
  an error that instructs the caller to **reconcile** (query open orders/status)
  before resubmitting.
- Automatic retry is opt-in **per call** for non-order mutations, and is
  impossible to enable globally for orders.

## Consequences

- Callers must handle ambiguous order submission by reconciliation.
- Transient failures on order submission surface as errors rather than silently
  retrying.
- Tests assert that order mutations issue exactly one attempt.
