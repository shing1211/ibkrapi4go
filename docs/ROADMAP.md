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
| 5 | OAuth2 surface: read ops (~40 ops) | IB REST |
| 6 | OAuth2 surface: write ops, SSO, Echo, Restrictions (~30 ops) | IB REST |
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

## Phase 5 — OAuth2 surface: read ops *(complete)*

Deliverables (~40 ops):

- [x] `internal/oauth.go` — OAuth2 token acquisition/refresh with single-flight
  refresh, refresh-token rotation, and JWT-bearer assertion (`fetchWithSecret`,
  `fetchWithJWTAssertion`).
- [x] `internal/jwt.go` — RS256 JWT signing for `private_key_jwt` grant
  (pure stdlib, PKCS8/PKCS1 PEM parsing).
- [x] `pkg/ibkr/oauth.go` — JWT key options: `WithOAuth2JWTKey`,
  `WithOAuth2JWTKeyFile`, `WithOAuth2JWTKeyPEM`, `WithOAuth2JWTExpiry`.
- [x] `pkg/ibkr/rest.go` — `RESTSurface` + `RESTAccounts` (`Details`) + `RESTTaxVouchers`.
- [x] `pkg/ibkr/rest_accounts.go` — 13 account ops: `List`, `Details`,
  `LoginMessages`, `AccountStatusBulk`, `KycURL`, `LoginMessagesForAccount`,
  `AccountStatus`, `AccountTasks`, `UpdateAccountStatus`, `UpdateAccountTasks`,
  `CreateAccount`, `CreateAccountDocument`, `CreateAccountTask`.
- [x] `pkg/ibkr/rest_requests.go` — 2 request ops: `ListRequests`,
  `UpdateRequestStatus`.
- [x] `pkg/ibkr/rest_banking.go` — 8 banking ops: `ClientInstruction`,
  `InstructionSet`, `Instruction`, `QueryTransactions`, `CancelInstruction`,
  `CancelInstructionsBulk` + sub-managers (stubs for transfers).
- [x] `pkg/ibkr/rest_utilities.go` — 6 utility ops: `Enumerations`,
  `ComplexAssetTransferBrokers`, `Forms`, `RequiredForms`,
  `ParticipatingBanks`, `ValidateUsername`.
- [x] `pkg/ibkr/restrictions.go` — 2 restriction ops: `AccountRestrictions`,
  `UserRestrictions`.
- [x] `pkg/ibkr/rest_balances.go` — 1 balance op: `Query`.
- [x] `pkg/ibkr/rest_statements.go` — statement ops: `Generate`, `ListAvailable`.
- [x] `pkg/ibkr/rest_taxdocuments.go` — tax document ops: `Generate`, `ListAvailable`.
- [x] `pkg/ibkr/rest_tradeconfirmations.go` — trade confirmation ops: `Generate`, `ListAvailable`.

## Phase 6 — OAuth2 surface: write ops, SSO, Echo, Restrictions *(in progress)*

Deliverables (~30 ops, 3 of 4 PRs complete):

**PR 1 — SSO Sessions + Echo (complete):**
- [x] `pkg/ibkr/rest_sso.go` — SSO session management: `CreateBrowserSession`, `CreateSession` (2 ops).
- [x] `pkg/ibkr/rest_echo.go` — Echo utilities: `ListEchoHttps`, `CreateEchoSignedJwt` (2 ops).

**PR 2 — Transfer write ops (deferred to future work):**
- [ ] External asset transfers: `Transfer`, `TransferBulk`, `TransferV2`, `TransferBulkV2` (4 ops)
- [ ] Internal asset transfers: `Transfer`, `TransferBulk` (2 ops)
- [ ] External cash transfers: `Transfer`, `TransferBulk`, `QueryBalances` (3 ops)
- [ ] Internal cash transfers: `Transfer`, `TransferBulk` (2 ops)
- [ ] Bank instructions: `Create`, `Query`, `CreateBulk` (3 ops)
- [ ] `BulkInstructionsCancel` (1 op)

**PR 3 — Restrictions with Signed JWT (complete):**
- [x] `MasterRestrictionIDs`, `MasterListIDs`, `ListDetails`, `RestrictionDetails`, `RestrictionScope` (5 ops)
- [x] `ApplyCSV`, `VerifyCSV` (2 ops)

**PR 4 — Docs sweep (pending):**
- [ ] Final ROADMAP update
- [ ] Verify all checks pass

Notes:

- Transfer operations are deferred due to complex polymorphic union types requiring additional review.
- Restrictions with Signed JWT use Authorization query param or header as appropriate.

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
