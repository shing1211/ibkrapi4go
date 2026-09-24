# Glossary

Terminology used throughout this repository. Getting these right is important:
the original planning docs conflated several of them.

## API surfaces

**Client Portal API (CPAPI)**
The legacy `/v1/api/*` surface served by the locally-run Client Portal Gateway.
Authenticated with `ssoBearer`. 115 operations in spec v2.40.0. Implemented in
full.

**IB REST API**
The newer `/gw/api/v1/*` and `/gw/api/v2/*` surface served by `api.ibkr.com`.
Authenticated with `oauth2Bearer`. 70 operations (including `/oauth2/*`).
Implemented in full (Phases 5–6); see
[ADR 0011](./adr/0011-oauth2-surface.md).

**Client Portal Gateway (CPGW)**
A Java application distributed by Interactive Brokers that runs locally (default
`https://localhost:5000`), holds an authenticated brokerage session, and exposes
CPAPI. Users authenticate to it interactively via a browser.

## Authentication

**`ssoBearer`**
OpenAPI security scheme (`type: http`, `scheme: bearer`) required by every
`/v1/api` operation. The token comes from an authenticated gateway session.

**`oauth2Bearer`**
OpenAPI security scheme (`type: http`, `scheme: bearer`) required by every
`/gw/api/*` operation. Obtained via the OAuth2 token endpoint.

**tickle**
`POST /v1/api/tickle` — the session keep-alive/heartbeat. IBKR sessions expire
without periodic tickling (~60s). See [SESSIONS.md](./SESSIONS.md).

**ssodh / `ssodh/init`**
`POST /v1/api/iserver/auth/ssodh/init` — initializes a brokerage session on an
already-authenticated gateway.

## Identifiers

**`accountId`**
A brokerage account identifier, e.g. `U1234567` or `DU1234567` (paper).

**`clientId`**
A Client Portal client identifier used by the `/gw/*` surface. Distinct from
`accountId`.

**`conid`**
IBKR's numeric contract identifier, e.g. `265598` (AAPL). The canonical way to
identify an instrument; symbols are ambiguous across exchanges.

## Market data

**snapshot vs stream**
`GET /v1/api/iserver/marketdata/snapshot` returns a point-in-time quote.
Streaming quotes arrive over `/v1/api/ws` after subscribing.

**field codes**
IBKR quotes fields by numeric code, e.g. `31` = last, `84` = bid, `86` = ask,
`83`/`88` = bid/ask size. See [STREAMING.md](./STREAMING.md).

## Codegen

**spec**
The OpenAPI 3.0.0 document at `https://api.ibkr.com/gw/api/v3/api-docs`
(title "IB REST API", version 2.40.0). Cached under `specs/` (gitignored).

**generated code**
`client/*.gen.go`, produced by `oapi-codegen`. Never edited by hand.

**patch**
A programmatic fix in `scripts/patch_spec.py` applied to the spec before codegen,
because the published spec contains defects. See [CODEGEN.md](./CODEGEN.md).
