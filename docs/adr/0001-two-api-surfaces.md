# 0001 — Treat the CPAPI and IB REST surfaces as distinct

- Status: Accepted
- Date: 2026-09-16

## Context

The single OpenAPI document at `https://api.ibkr.com/gw/api/v3/api-docs`
describes 185 operations that actually belong to **two different API surfaces**:

- `/v1/api/*` — the Client Portal API (CPAPI), served by the locally-run Client
  Portal Gateway, authenticated with `ssoBearer` (115 operations).
- `/gw/api/v1/*`, `/gw/api/v2/*`, `/oauth2/*` — the hosted IB REST API,
  authenticated with `oauth2Bearer` (70 operations).

Earlier planning documents treated all 185 operations as one homogeneous API with
a single base URL and one auth flow. That is incorrect: the two surfaces differ
in host, in credential type, and in error behavior.

## Decision

Model the two surfaces as **distinct**. The SDK exposes surface-specific clients.
v1 implements only CPAPI (see [ADR 0005](./0005-v1-scope.md)); the IB REST surface
is a separate, later addition.

## Consequences

- The `Client` composes surface clients rather than assuming one base URL and one
  auth header.
- Auth is modeled per surface (see [ADR 0002](./0002-auth-models.md)).
- `docs/SPEC.md` records the `Surface` and `Auth` for every operation.
- Documentation and examples must state which surface they target.
- Two base URLs may be configured independently.
