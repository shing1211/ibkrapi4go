# Roadmap

Phased plan for ibkrapi4go. Phases are gated by **exit criteria**, not calendar
time. Scope for v1 is the CPAPI (`ssoBearer`) surface only — see
[ADR 0005](./adr/0005-v1-scope.md).

## Scope

| Phase | Focus | Surface |
|-------|-------|---------|
| 0 | Truth, licensing, codegen validation | — |
| 1 | Core: client, session, transport, account | CPAPI |
| 2 | Portfolio, orders, contracts, market data | CPAPI |
| 3 | WebSocket streaming | CPAPI |
| 4 | Hardening: rate limits, retries, observability, docs | CPAPI |
| 5 | OAuth2 surface (`/gw/*`) | IB REST |
| — | Non-goals | — |

The spec contains 185 operations: 115 on CPAPI (`ssoBearer`) and 70 on IB REST
(`oauth2Bearer`). v1 covers a **subset** of CPAPI — the commonly used
account/portfolio/trading/market-data endpoints — not all 115.

## Phase 0 — Truth, licensing, codegen *(complete)*

Deliverables:

- [x] Full Apache-2.0 `LICENSE`, `NOTICE`, `THIRD_PARTY_NOTICES.md`, `DISCLAIMER.md`.
- [x] SPDX headers and `make license` / `make license-check`.
- [x] Measured codegen validation; `scripts/patch_spec.py` generalized.
- [x] Documentation set, ADRs, CI scaffolding.
- [x] Module initialized (`go.mod`) and generated client committed
  (`client/client.gen.go`), with a deterministic SPDX header.

Exit criteria:

- [x] Codegen succeeds and generated client compiles.
- [x] Apache-2.0 detected by GitHub.
- [x] Every count in docs derives from [SPEC.md](./SPEC.md).
- [x] `make codegen-verify` passes against the committed client.

## Phase 1 — Core *(complete)*

Deliverables:

- [x] `pkg/ibkr/client.go` — `NewClient`, functional options, `Close`.
- [x] `internal/transport.go` — RoundTripper chain (request ID, auth, errors),
  wired into the client's `*http.Client`.
- [x] `internal/session.go` — state machine + tickle ([SESSIONS.md](./SESSIONS.md)).
- [x] `pkg/ibkr/auth.go` — `SessionManager` (`Initialize`, `Close`, `State`, `Status`).
- [x] `pkg/ibkr/account.go` — `AccountManager` (`List`, `Summary`, `PnL`).

Exit criteria:

- [x] Unit tests cover the session state machine, including `goleak` after `Close`.
- [x] A request can be made against an `httptest` gateway end-to-end
  (`pkg/ibkr/endtoend_test.go`).
- [x] No exported symbol lacks a doc comment.

Notes:

- Account **details** (`/gw/api/v1/accounts/{id}/details`) belongs to the IB REST
  surface and is deferred to Phase 5; CPAPI account data is served by `List`,
  `Summary`, and `PnL`.
- Money/quantity fields are decoded as `json.Number` and exposed as `string`
  (ADR 0008) using raw generated calls plus hand-written adapters.
- `BalanceSummary`/`MarginSummary`/`FundSummary` carry into Phase 2.

## Phase 2 — Portfolio, trading, market data *(complete)*

Deliverables:

- [x] `pkg/ibkr/portfolio.go` — `PortfolioManager` (accounts, positions incl.
  pagination, ledger, allocation, summary, meta, invalidate).
- [x] `pkg/ibkr/trade.go` — `TradeManager` orders (submit/confirm/modify/cancel,
  what-if, status, open orders, trades) and the reply/confirmation flow
  ([design/09](./design/09-orders-and-confirmation.md)).
- [x] `pkg/ibkr/contract.go` — secdef search, contract info/rules, strikes.
- [x] `pkg/ibkr/marketdata.go` — `MarketDataManager` (snapshot, history,
  unsubscribe, unsubscribe-all).

Exit criteria:

- [x] Order mutations are provably **not** auto-retried
  (`TestOrders_MutationsAreSingleAttempt` asserts a single outbound request).
- [x] Money/quantity fields use `string`/`json.Number`
  (`scripts/check_money.py`, run by `make check`).
- [x] Pagination is exercised by tests (`TestPortfolio` walks pages until empty).

Notes:

- Order submissions are sent as hand-built JSON with string money/quantity so
  precision is preserved on the wire (the generated request type uses `float32`).
- Ambiguous mutation timeouts surface as an `*Error` with `Code: "ambiguous"`,
  directing callers to reconcile via `Trade().OpenOrders`.

## Phase 3 — Streaming *(complete)*

Deliverables:

- [x] `internal/ws.go`, `pkg/ibkr/ws.go` — connect, subscribe, dispatch
  (`coder/websocket`).
- [x] Channel-based `Subscription` with `Updates()`/`Errors()` and
  cancellation ([STREAMING.md](./STREAMING.md)).
- [x] Auto-reconnect with backoff and re-subscription.

Exit criteria:

- [x] Local WebSocket server test: subscribe, receive, cancel, reconnect
  (`pkg/ibkr/ws_test.go`).
- [x] No goroutine leaks after unsubscribe/close (`goleak`).

Notes:

- The transport passes `101 Switching Protocols` through untouched so the
  upgrade connection reaches `coder/websocket`.
- Updates are emitted one value per field code; a conid's updates are delivered
  to every subscription that requested it.
- Backpressure drops the oldest buffered update and increments `Dropped()`.
- Streaming is marked **pending live verification** against a real gateway.

## Phase 4 — Hardening *(complete)*

Deliverables:

- [x] Rate limiting per endpoint + global ([RATE-LIMITING.md](./RATE-LIMITING.md)).
- [x] Retry budgets for idempotent methods; circuit breaker.
- [x] `slog` logging with redaction; dependency-free telemetry hooks
  (OTel-bridgeable).
- [x] Testable `Example*` functions; godoc on exported symbols.

Exit criteria:

- [x] Rate limiter unit-tested against burst/steady limits.
- [x] A `429` with `Retry-After` is honored.
- [x] CI green across the support matrix.

Notes:

- The transport is now a composable middleware chain
  ([design/01](./design/01-transport.md)); the retry middleware retries only safe
  methods, so order mutations remain single-attempt (ADR 0009).
- The circuit breaker is disabled by default; telemetry carries no OpenTelemetry
  dependency (ADR 0004).

## Phase 5 — OAuth2 surface (post-v1) *(foundation complete; managers incremental)*

Deliverables:

- [x] `internal/oauth.go` — OAuth2 token acquisition/refresh with single-flight
  refresh and refresh-token rotation; `Client.REST()` surface bound to the IB
  REST host with `Authorization: Bearer` injection.
- [x] `pkg/ibkr/rest.go` — `RESTSurface` + `RESTAccounts` (`Details`).
- [ ] Remaining `/gw/api/v1` and `/gw/api/v2` managers (incremental).

Exit criteria:

- [x] Separate ADR accepted ([ADR 0011](./adr/0011-oauth2-surface.md)).
- [x] Token refresh and rotation tested.

Notes:

- No new runtime dependency: token acquisition uses `net/http` form encoding
  (ADR 0004); no OAuth2 library or OpenTelemetry SDK is added.
- Credentials may come from `WithOAuth2*` options or `IBKR_CLIENT_ID` /
  `IBKR_CLIENT_SECRET` / `IBKR_CLIENT_REFRESH_TOKEN`.
- Pending live-gateway verification.

## Non-goals

- TWS API (proprietary socket protocol), FIX.
- Account opening / KYC workflows.
- Redistribution of IBKR's OpenAPI spec (fetched at build time; gitignored).
- A hosted/doc-site service.

## Backlog (not scheduled)

Benchmarks, fuzz tests, coverage badge, pre-commit hooks, GitHub Discussions,
`FUNDING.yml`, docs website.

---

*Scope and counts are governed by [SPEC.md](./SPEC.md).*
