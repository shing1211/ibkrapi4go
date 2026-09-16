# 0011 — OAuth2 / IB REST surface

- Status: Accepted
- Date: 2026-09-17

## Context

[ADR 0002](./0002-auth-models.md) and [ADR 0005](./0005-v1-scope.md) deferred the
`oauth2Bearer` surface (`/gw/api/v1/*`, `/gw/api/v2/*`, `/oauth2/*`). It differs
from the Client Portal API in three ways that matter to the client:

- it is hosted (e.g. `https://api.ibkr.com`), not the local gateway;
- it is authenticated with an OAuth2 bearer token obtained from a token
  endpoint, not a browser session;
- tokens expire and must be refreshed, and refresh tokens may rotate.

Phase 5 brings this surface in. It requires its own decision record before
implementation (per ADR 0002).

## Decision

- Add an **`OAuth2` token source** (`internal/oauth.go`) that supports the
  `client_credentials` and `refresh_token` grants, refreshes **before** expiry,
  serializes concurrent refreshes (single-flight), and **rotates** the refresh
  token when the server returns a new one. Tokens are held in memory only.
- Model the surface as a **distinct composed client**, `Client.REST()`, bound to
  its own base URL (`https://api.ibkr.com` by default) and injecting
  `Authorization: Bearer <token>` from the token source. Auth is per surface
  (ADR 0001/0002).
- Use **no new runtime dependency**: token acquisition is `net/http` +
  `application/x-www-form-urlencoded`; no OAuth2 or OpenTelemetry library is
  added (ADR 0004).
- `/gw` managers adapt generated responses to **stable public types**; generated
  types are not exported (design/04).

## Consequences

- `NewClient` accepts `WithRESTGateway` and `WithOAuth2*` options; credentials
  may also come from the environment (`IBKR_CLIENT_ID`, `IBKR_CLIENT_SECRET`,
  `IBKR_CLIENT_REFRESH_TOKEN`).
- A `401` from the REST surface surfaces as `ErrSessionExpired`; the token
  source refreshes on the next call rather than retrying blindly.
- Refresh and rotation are covered by tests; live-gateway verification is
  pending.
- The `/gw` manager set is built incrementally; the token layer and surface
  plumbing are the foundation.
