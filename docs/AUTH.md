# Authentication

IBKR has **two authentication models**, corresponding to the two API surfaces
(see [GLOSSARY.md](./GLOSSARY.md) and [ADR 0002](./adr/0002-auth-models.md)).
SDK v1 implements only the `ssoBearer` model.

## Model comparison

| | `ssoBearer` (v1) | `oauth2Bearer` (deferred) |
|---|---|---|
| Surface | `/v1/api/*` | `/gw/api/v1/*`, `/gw/api/v2/*` |
| Where | Local Client Portal Gateway | `api.ibkr.com` |
| Login | Interactive (browser + 2FA) | OAuth2 token endpoint |
| SDK holds | Session token from gateway | Access + refresh tokens |
| Operations | 115 | 70 |

## Important: no username/password

The published OpenAPI spec defines **no password grant**. There is no supported
way to programmatically log a user in with a username and password. The README's
early draft (`Auth().SSO(ctx, "user", "pass")`) was incorrect and has been
removed.

For `ssoBearer`, the user logs in to the gateway in a browser. The SDK then:

1. Initializes the brokerage session (`POST /v1/api/iserver/auth/ssodh/init`).
2. Keeps it alive via the tickle loop (see [SESSIONS.md](./SESSIONS.md)).
3. Reacts to expiry by asking the caller to re-authenticate in the browser.

## Interface (planned)

```go
type SessionManager interface {
    // Initialize establishes a brokerage session on an already-authenticated
    // gateway. Returns ErrNotAuthenticated if the gateway has no browser session.
    Initialize(ctx context.Context) error

    // Status reports the current brokerage session state.
    Status(ctx context.Context) (SessionStatus, error)

    // Logout terminates the gateway session and stops the tickle loop.
    Logout(ctx context.Context) error
}
```

The SDK does **not** open a browser or drive a login flow. Interactive
authentication is out of scope; users authenticate the gateway themselves.

## OAuth2 (deferred)

The `oauth2Bearer` flow (token request, refresh, rotation) will be specified in
a later ADR when the `/gw/*` surface is implemented. It is **not** part of v1.

## Secrets handling

- Tokens live in memory only; never written to disk.
- Tokens, cookies, and `Authorization` headers are redacted from logs and errors.
- Credentials are supplied by the environment or the gateway session — never
  hardcoded. See [../SECURITY.md](../SECURITY.md) and
  [design/08-concurrency.md](./design/08-concurrency.md).

## Failure modes

| Symptom | Cause | Handling |
|---------|-------|----------|
| `401` on any call | Session expired | Trigger session status check; surface `ErrSessionExpired`; do not auto-login |
| Tickle fails twice | Gateway restarted | Mark session `EXPIRED`; require re-initialization |
| Gateway unreachable | Not running | Surface connection error; no retry storm |
