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
| 5 | OAuth2 surface: read ops (~48 ops) | IB REST |
| 6 | OAuth2 surface: write ops, SSO, Echo, Restrictions (~57 ops) | IB REST |
| 7 | Remaining CPAPI surface (~81 ops) | CPAPI |
| — | Non-goals | — |

The spec contains 185 operations: 115 on CPAPI (`ssoBearer`) and 70 on IB REST
(`oauth2Bearer`). All 185 are now implemented.

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
- [x] `pkg/ibkr/contract.go` — secdef search, contract info/rules, strikes
  (extended in Phase 7 with 13 additional ops).
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
  (OTel-bridgeable). Unified subsystem prefixes (`ibkr.http`, `ibkr.session`,
  `ibkr.ws`, `ibkr.oauth`, `ibkr.ratelimit`, `ibkr.breaker`, `ibkr.error`,
  `ibkr.config`) with a never-nil logger contract ([LOGGING.md](./LOGGING.md)).
- [x] Testable `Example*` functions; godoc on exported symbols.

Exit criteria:

- [x] Rate limiter unit-tested against burst/steady limits.
- [x] A `429` with `Retry-After` is honored.
- [x] CI green across the support matrix.

## Phase 5 — OAuth2 surface: read ops *(complete)*

Deliverables (~48 of 70 oauth2Bearer ops):

- [x] `internal/oauth.go` — OAuth2 token acquisition/refresh with single-flight
  refresh, refresh-token rotation, and JWT-bearer assertion (`fetchWithSecret`,
  `fetchWithJWTAssertion`).
- [x] `internal/jwt.go` — RS256 JWT signing for `private_key_jwt` grant
  (pure stdlib, PKCS8/PKCS1 PEM parsing).
- [x] `pkg/ibkr/oauth.go` — JWT key options: `WithOAuth2JWTKey`,
  `WithOAuth2JWTKeyFile`, `WithOAuth2JWTKeyPEM`, `WithOAuth2JWTExpiry`.
- [x] `pkg/ibkr/rest.go` — `RESTSurface` + `RESTAccounts` (`Details`) + `RESTTaxVouchers`.
- [x] `pkg/ibkr/rest_accounts.go` — 14 account ops: `List`, `Details`,
  `LoginMessages`, `BulkStatus`, `KycURL`, `LoginMessagesForAccount`,
  `Status`, `Tasks`, `Update`, `Create`, `SubmitDocument`, `UpdateStatus`,
  `UpdateTasks`, `AssignTask`.
- [x] `pkg/ibkr/rest_requests.go` — 3 request ops: `ListRequests`,
  `UpdateRequestStatus`, `GetStatus`.
- [x] `pkg/ibkr/rest_banking.go` — 6 read ops: `ClientInstruction`,
  `InstructionSet`, `Instruction`, `QueryTransactions`, `CancelInstruction`,
  `CancelInstructionsBulk` + transfer sub-managers (stubs; write ops deferred).
- [x] `pkg/ibkr/rest_utilities.go` — 6 utility ops: `Enumerations`,
  `ComplexAssetTransferBrokers`, `Forms`, `RequiredForms`,
  `ParticipatingBanks`, `ValidateUsername`.
- [x] `pkg/ibkr/restrictions.go` — 2 restriction ops: `AccountRestrictions`,
  `UserRestrictions`.
- [x] `pkg/ibkr/rest_balances.go` — 1 balance op: `Query`.
- [x] `pkg/ibkr/rest.go` — statement ops: `Generate`, `ListAvailable`.
- [x] `pkg/ibkr/rest.go` — tax document ops: `Generate`, `ListAvailable`.
- [x] `pkg/ibkr/rest.go` — trade confirmation ops: `Generate`, `ListAvailable`.

## Phase 6 — OAuth2 surface: write ops, SSO, Echo, Restrictions *(complete)*

Deliverables (~70 of 70 oauth2Bearer ops; all complete):

**PR 1 — SSO Sessions + Echo (complete):**
- [x] `pkg/ibkr/rest_sso.go` — SSO session management: `CreateBrowserSession`, `CreateSession` (2 ops).
- [x] `pkg/ibkr/rest_echo.go` — Echo utilities: `ListEchoHttps`, `CreateEchoSignedJwt` (2 ops).

**PR 2 — Transfer write ops (complete):**
- [x] External asset transfers: `Transfer`, `TransferBulk`, `TransferV2`, `TransferBulkV2` (4 ops)
- [x] Internal asset transfers: `Transfer`, `TransferBulk` (2 ops)
- [x] External cash transfers: `Transfer`, `TransferBulk`, `QueryBalances` (3 ops)
- [x] Internal cash transfers: `Transfer`, `TransferBulk` (2 ops)
- [x] Bank instructions: `Create`, `Query`, `CreateBulk` (3 ops)
- [x] `BulkInstructionsCancel` (1 op)

**PR 3 — Restrictions with Signed JWT (complete):**
- [x] `MasterRestrictionIDs`, `MasterListIDs`, `ListDetails`, `RestrictionDetails`, `RestrictionScope` (5 ops)
- [x] `ApplyCSV`, `VerifyCSV` (2 ops)

**PR 4 — Docs sweep (complete):**
- [x] Final ROADMAP update
- [x] Verify all checks pass

Notes:

- Transfer operations implemented using polymorphic union types for instruction bodies.
- All 70 OAuth2 bearer ops now implemented (70/70).

## Phase 7 — Remaining CPAPI surface *(complete)*

Deliverables (~81 CPAPI ops to reach 185/185 total):

**PR 1 — Trading Accounts (complete):**
- [x] `pkg/ibkr/trading_accounts.go` — `TradingAccountManager` (9 ops): `GetAccountOwners`, `SetActiveAccount`, `GetDynamicAccounts`, `GetFundSummary`, `GetBalanceSummary`, `GetMarginSummary`, `GetAccountMarketSummary`, `GetBrokerageAccounts`, `SetDynamicAccount`.

**PR 2 — Trading Alerts (complete):**
- [x] `pkg/ibkr/alerts.go` — `AlertManager` (6 ops): `GetAlertDetails`, `GetMtaDetails`, `CreateAlert`, `ActivateAlert`, `DeleteAlert`, `GetAllAlerts`.

**PR 3 — Trading Contracts (complete):**
- [x] `pkg/ibkr/contract.go` — Extended `TradeManager` with 13 ops: `GetTradingSchedule`, `GetContractRules` (existing), `GetAlgosByInstrument`, `GetInfoAndRules`, `GetCurrencyPairs`, `GetExchangeRates`, `GetBondFilters`, `GetContractInfo`, `GetContractSymbolsFromBody`, `GetConidsByExchange`, `GetFutureBySymbol`, `GetInstrumentDefinition`, `GetTradingScheduleBySymbol`, `GetStockBySymbol`.

**PR 4 — Event Contracts + Scanner (complete):**
- [x] `pkg/ibkr/events.go` — `ForecastManager` (5 ops): `GetForecastCategories`, `GetForecastContract`, `GetForecastMarkets`, `GetForecastRules`, `GetForecastSchedule`.
- [x] `pkg/ibkr/scanner.go` — `ScannerManager` (2 ops): `GetScannerParameters`, `GetScannerResults`.

**PR 5 — FA Allocation + Model Portfolios (complete):**
- [x] `pkg/ibkr/allocation.go` — `AllocationManager` (8 ops): `GetAllocatableSubaccounts`, `GetAllocationGroups`, `CreateAllocationGroup`, `ModifyAllocationGroup`, `DeleteAllocationGroup`, `GetSingleAllocationGroup`, `GetAllocationPresets`, `SetAllocationPreset`.
- [x] `pkg/ibkr/models.go` — `ModelManager` (10 ops): `GetModelPresets`, `SetModelPresets`, `GetAccountsInModel`, `SetAccountInvestmentInModel`, `GetInvestedAccountsInModel`, `GetAllModels`, `GetAllModelPositions`, `SetModelTargetPositions`, `SubmitModelOrders`, `GetModelSummarySingle`.

**PR 6 — FYIs/Notifications + OAuth (complete):**
- [x] `pkg/ibkr/notifications.go` — `FYIManager` (11 ops): `GetFYIDelivery`, `ModifyFYIDelivery`, `ModifyFYIEmails`, `DeleteFYIDevice`, `GetFYIDisclaimers`, `ReadFYIDisclaimer`, `GetAllFYIs`, `ReadFYINotification`, `GetFYISettings`, `ModifyFYINotification`, `GetUnreadFYIs`.
- [x] `pkg/ibkr/oauth1.go` — `OAuthManager` (3 ops): `ReqAccessToken`, `ReqLiveSessionToken`, `ReqTempToken`.

**PR 7 — Portfolio + Analyst + Watchlists (complete):**
- [x] `pkg/ibkr/portfolio.go` — Extended `PortfolioManager` with 4 ops: `GetAllAccountsForConid`, `GetManySubaccounts`, `GetComboPositions`, `GetUncachedPositions`.
- [x] `pkg/ibkr/performance.go` — `PerformanceManager` (4 ops): `CreateAllocationPA`, `GetPerformanceAllPeriods`, `GetSinglePerformancePeriod`, `GetTransactions`.
- [x] `pkg/ibkr/watchlists.go` — `WatchlistManager` (4 ops): `DeleteWatchlist`, `GetSpecificWatchlist`, `PostNewWatchlist`, `GetAllWatchlists`.

**PR 8 — Session Validation (complete):**
- [x] `pkg/ibkr/auth.go` — Extended `SessionManager` with 2 ops: `GetSessionValidation`, `GetSessionToken`.

Notes:

- All 115 CPAPI (ssoBearer) ops now implemented.
- Total coverage: 185/185 operations (115 CPAPI + 70 IB REST).
- All managers registered in `Client` with accessor methods.
- All checks pass: `make check`, `make test-race`, `make docs-check`, `make license-check`.

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
