# Gateway Setup

The SDK talks to the **Client Portal Gateway** — a small Java process that IBKR
ships and that you run locally. The SDK does **not** drive the interactive login;
you authenticate the gateway in a browser, and the SDK uses the resulting session.

See also: [AUTH.md](./AUTH.md) · [SESSIONS.md](./SESSIONS.md) ·
[CONFIG.md](./CONFIG.md) · [PERMISSIONS.md](./PERMISSIONS.md) ·
[MOCK-GATEWAY.md](./MOCK-GATEWAY.md).

## 1. Download and run the gateway

1. Download the Client Portal Gateway from IBKR (the "Client Portal API" /
   Web API download). Pick the build for your OS.
2. Unzip it. It contains a `bin/` directory and a `root/` directory holding the
   self-signed certificate and `conf.yaml`.
3. Start it:

   ```bash
   # Unix
   bin/run.sh root/conf.yaml

   # Windows
   bin\run.bat root\conf.yaml
   ```

   The gateway listens on `https://localhost:5000` by default.

Point the SDK at a non-default host/port with `WithGatewayURL`:

```go
cli, err := ibkr.NewClient(
    ibkr.WithGatewayURL("https://localhost:5000"),
    ibkr.WithInsecureSkipVerify(true), // local self-signed cert
)
```

## 2. Authenticate in a browser

Open `https://localhost:5000` in a browser. This is where you complete IBKR
login and 2FA. The gateway issues a session cookie that the SDK reuses.

There is **no username/password grant**. The published OpenAPI spec defines no
password flow, and the SDK deliberately has no `SSO(user, pass)` call. Do not
put credentials in environment variables expecting the SDK to log in — it
cannot. See [AUTH.md](./AUTH.md).

## 3. Initialize the session

Authenticating in the browser is not enough; the brokerage session must be
initialized before trading endpoints work:

```go
if err := cli.Session().Initialize(ctx); err != nil {
    return err
}
defer cli.Close()
```

`Initialize` calls `POST /v1/api/iserver/auth/ssodh/init`. `Close` performs a
best-effort logout that clears the **gateway** session; it does not log the user
out of the IBKR website. Details and the tickle heartbeat are in
[SESSIONS.md](./SESSIONS.md).

## Session lifecycle

| Step | Who | Notes |
|------|-----|-------|
| Login + 2FA | User (browser) | Gateway holds the cookie |
| `Session().Initialize` | SDK | Required before trading calls |
| Tickle / keep-alive | SDK | Automatic; see [SESSIONS.md](./SESSIONS.md) |
| Session expiry | Gateway | SDK surfaces `ErrSessionExpired`; user re-authenticates in the browser |
| `Client.Close` | Caller | Best-effort gateway logout |

The SDK never opens a browser or automates login. If the session expires
mid-run, re-authenticate in the browser and re-run `Initialize`.

## TLS notes

The gateway serves a self-signed certificate, so a browser warns on first visit
and the SDK needs `WithInsecureSkipVerify(true)` (or
`IBKR_INSECURE_SKIP_VERIFY=true`) against `localhost`.

- Insecure verification **must only** be used against a loopback host; the SDK
  logs a warning if it is enabled for a non-loopback address.
- Prefer trusting the gateway's `root/` certificate in your OS trust store over
  disabling verification, especially outside local development.
- The hosted IB REST (`oauth2Bearer`) surface at `https://api.ibkr.com` uses a
  publicly trusted certificate and does **not** need insecure verification. See
  [CONFIG.md](./CONFIG.md#tls) and [AUTH.md](./AUTH.md).

## Common problems

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `connection refused` | Gateway not running | Start `run.sh` / `run.bat` |
| `x509: certificate signed by unknown authority` | Self-signed cert | `WithInsecureSkipVerify(true)` for localhost |
| `401` / session errors | Not authenticated or expired | Log in via the browser, re-run `Initialize` |
| Trading calls fail but quotes work | Brokerage session not initialized | Call `Session().Initialize` |
| Entitlement / delayed-data symptoms | Missing market-data permission | See [PERMISSIONS.md](./PERMISSIONS.md) |

For local tests and examples, use the in-repo mock gateway instead of the real
one — see [MOCK-GATEWAY.md](./MOCK-GATEWAY.md).
