# 0005 — v1 targets the CPAPI surface only

- Status: Superseded
- Date: 2026-09-16

## Context

The spec covers 185 operations across two surfaces (see
[ADR 0001](./0001-two-api-surfaces.md)). Covering both, plus WebSocket streaming,
plus hardening, is a large effort. The `oauth2Bearer` surface requires an OAuth2
client with token refresh and a different host — a distinct problem from the local
`ssoBearer` gateway.

Most user-facing use cases (accounts, positions, orders, quotes, streaming) live
on CPAPI.

## Decision

- **v1 implements CPAPI (`ssoBearer`) only**, and a **subset** of its 115
  operations: account, portfolio, orders/contracts, market data, session, and
  WebSocket streaming.
- The `/gw/*` (`oauth2Bearer`) surface is deferred (Phase 5) and requires a
  separate ADR.
- Uncommon CPAPI endpoints (banking, tax vouchers, event contracts, forms, FYIs)
  are not in the v1 critical path.

## Consequences

- Smaller, achievable v1.
- Some advertised spec capabilities are unavailable until later phases; the
  README/docs must say so.
- The endpoint subset is tracked in [../ROADMAP.md](../ROADMAP.md).

## Superseded

As of commit `48bdde6`, **all 185 operations are implemented** (115 CPAPI + 70 IB REST).
Phases 5-6 implemented the `oauth2Bearer` surface; Phase 7 completed the remaining CPAPI
operations. This ADR is retained for historical context.
