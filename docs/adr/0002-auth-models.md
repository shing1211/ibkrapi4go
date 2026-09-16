# 0002 — Two auth models: `ssoBearer` and `oauth2Bearer`

- Status: Accepted
- Date: 2026-09-16

## Context

The OpenAPI document defines exactly two security schemes:

- `ssoBearer` (`type: http`, `scheme: bearer`) — required by all `/v1/api`
  operations.
- `oauth2Bearer` (`type: http`, `scheme: bearer`) — required by all `/gw/api/*`
  operations.

There is **no** `password` grant or any username/password flow in the spec. The
Client Portal Gateway (CPAPI) is authenticated interactively in a browser; the
SDK cannot and should not attempt to log a user in with credentials.

## Decision

- Model authentication per surface with two implementations of a common
  `CredentialProvider` interface.
- **v1 implements only `ssoBearer`.** The SDK assumes a gateway that a human has
  already authenticated in a browser; the SDK initializes the brokerage session
  and manages the token/tickle lifecycle.
- The SDK never accepts or stores a username/password.
- `oauth2Bearer` is deferred and requires its own ADR before implementation.

## Consequences

- The README/quick-start must not show a password-based login.
- On `401`, the SDK surfaces `ErrSessionExpired` and requires re-initialization;
  it does **not** attempt a silent credential login.
- Tokens are memory-only and redacted from logs.
