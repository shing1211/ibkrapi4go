# Authentication

IBKR has **two authentication models**, corresponding to the two API surfaces
(see [GLOSSARY.md](./GLOSSARY.md), [ADR 0002](./adr/0002-auth-models.md), and
[ADR 0011](./adr/0011-oauth2-surface.md)). The SDK implements both: CPAPI
(`ssoBearer`) and the IB REST API (`oauth2Bearer`).

## Model comparison

| | `ssoBearer` (CPAPI) | `oauth2Bearer` (IB REST) |
|---|---|---|
| Surface | `/v1/api/*` | `/gw/api/v1/*`, `/gw/api/v2/*`, `/oauth2/*` |
| Where | Local Client Portal Gateway | `api.ibkr.com` |
| Login | Interactive (browser + 2FA) | OAuth2 token endpoint |
| SDK holds | Session token from gateway | Access + refresh tokens |
| Operations | 123 | 70 |

## Important: no username/password

The published OpenAPI spec defines **no password grant**. There is no supported
way to programmatically log a user in with a username and password. The README's
early draft (`Auth().SSO(ctx, "user", "pass")`) was incorrect and has been
removed.

For `ssoBearer`, the user logs in to the gateway in a browser. The SDK then:

1. Initializes the brokerage session (`POST /v1/api/iserver/auth/ssodh/init`).
2. Keeps it alive via the tickle loop (see [SESSIONS.md](./SESSIONS.md)).
3. Reacts to expiry by asking the caller to re-authenticate in the browser.

## CPAPI session interface

`Client.Session()` returns a `*SessionManager`:

```go
func (m *SessionManager) Initialize(ctx context.Context) error
func (m *SessionManager) Close(ctx context.Context) error   // best-effort logout
func (m *SessionManager) State() SessionState
func (m *SessionManager) Status(ctx context.Context) (*AuthStatus, error)
```

The SDK does **not** open a browser or drive a login flow. Interactive
authentication is out of scope; users authenticate the gateway themselves.

## OAuth2 (IB REST surface)

The hosted `/gw/*` surface uses an OAuth2 bearer token obtained from the token
endpoint (`https://api.ibkr.com/oauth2/api/v1/token`). The token source in
`internal/oauth.go` supports the `client_credentials` and `refresh_token`
grants, plus `private_key_jwt` (`client_assertion`) via `internal/jwt.go`. It
refreshes before expiry, serializes concurrent refreshes (single-flight), and
rotates the refresh token when the server returns a new one. Tokens live in
memory only. Configure it with `WithOAuth2ClientCredentials`,
`WithOAuth2RefreshToken`, and the `WithOAuth2JWTKey*` options (or the
`IBKR_CLIENT_ID` / `IBKR_CLIENT_SECRET` / `IBKR_CLIENT_REFRESH_TOKEN`
environment variables). See [ADR 0011](./adr/0011-oauth2-surface.md).

`Client.REST()` exposes the lifecycle controls:

- `Token(ctx)` returns the current access token and refreshes automatically
  when needed. Treat the returned value as a secret.
- `ForceRefresh(ctx)` discards the access-token cache and acquires a new token.
- `Invalidate()` clears only the access-token cache; the rotated refresh token
  is preserved and the next request reacquires an access token.

`Invalidate` does not cancel an already-running request. A stale in-flight
result is returned only to its original caller and is not cached.

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
