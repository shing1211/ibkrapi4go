# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-09-21

This release marks the first stable API surface. All public symbols in `pkg/ibkr`
are now covered by a stability contract (ADR 0015). The API is production-ready
for Go 1.26+.

### Breaking changes from v0.x

This release is not fully API-compatible with v0.x. A migration guide and
automated codemod are provided.

- **57 methods** renamed: the `Get` prefix is removed per Go naming conventions.
  Run `scripts/codemod.sh` to automate the mechanical renames. See
  `docs/MIGRATION.md` for the full table and manual steps.
- **`float32` → `int64`** for banking IDs (`ClientInstructionID`,
  `InstructionID`, `IbReferenceID`).
- **14 initialism casing fixes** (e.g. `EchoHTTPSResponse` → `EchoHTTPSResponse`,
  `RealizedPnL` → `RealizedPnL`). Automated via `scripts/codemod.sh`.
- **`ErrStreamDisconnected` / `ErrStreamReconnected`** removed; use
  `ErrWSDisconnected` / `ErrWSReconnected`.

### Stability Hardening

- **Goroutine safety (S1):** All 6 async goroutine paths now recover from panics.
  `TestGoleak` integration covers every goroutine-creating function. ` goleak`
  is a test-only dependency; no runtime cost.
- **Backpressure (S2):** Context deadline propagation through all WS/tickle/token
  paths. New `WithRequestTimeout` option sets a hard timeout per operation.
- **Production resilience (S3):** `WithEndpointTimeout` and
  `WithCircuitBreakerBudget` options. Error budget tracking in `internal/breaker.go`
  with `BreakerMetrics` export.
- **Fault injection (S4):** 36 fault-injection tests across `internal/` and
  `pkg/ibkr/` covering transport errors, timeout, circuit breaker, and order
  submission paths.
- **Performance baseline (S5):** HTTP/2 connection pooling in `internal/transport.go`.
  Allocation benchmarks in `pkg/ibkr/alloc_test.go`. Baseline stored in
  `benchmark.baseline`; CI compares every run.

### Ecosystem

- **CLI tool (E1):** `cmd/ibkr/` binary with `accounts`, `positions`, `orders`,
  `stream`, `portfolio`, `config`, and `completion` subcommands.
- **Migration guide (E2):** `docs/MIGRATION.md` (173 lines) and `scripts/codemod.sh`
  (74 rename rules) covering all v0.x → v1.0 breaking changes.
- **API reference site (E3):** `docs/api.html`, `docs/architecture.html`,
  `docs/decisions.html` with CSS/JS assets.
- **Release automation (E4):** `.github/workflows/release-automation.yml` with
  semver enforcement, `.github/workflows/supply-chain.yml` with govulncheck and
  SBOM generation. Gitee auto-push on release.
- **Supply-chain security (E5):** `go mod verify`, `govulncheck`, secret scanning,
  SLSA-level provenance generation.

### Architectural

- **Interface segregation (A1):** Five interfaces in `internal/interfaces.go`:
  `TokenProvider`, `RoundTripper`, `SessionMachine`, `WSClient`, `RateLimiter`.
  `internal/fake/` package with five fake implementations for testing.
- **Typed builders (A2):** `OrderBuilder`, `ContractBuilder`,
  `TransferInstructionBuilder` in `pkg/ibkr/builders.go`.
- **Middleware plugin (A3):** `WithTransportMiddleware` option in
  `pkg/ibkr/middleware.go`. Example in `examples/middleware/`.
- **Unified pagers (A4):** Generic `*Pager[T]` in `pkg/ibkr/pager.go`. Four
  concrete pagers: `ModelsPager`, `FYIsPager`, `TransactionsPager`,
  `SubaccountsPager`. Four old slice-returning methods deprecated.
- **Multi-client transport (A5):** `TransportPool` in `pkg/ibkr/transport_pool.go`
  with shared `*http.Client`, session, and rate limiter across clients.

### OpenTelemetry Support

- **OTel metrics bridge (P2):** `contrib/otel/` module ships a first-class
  `OTelMetrics` implementation of `ibkr.Metrics` bridging to OpenTelemetry
  counters, histograms, and gauges. Instrument creation is lazy and cached.
  Import `github.com/shing1211/ibkrapi4go/contrib/otel` only when needed;
  the core SDK carries no OTel dependency (ADR 0004).

### Examples

- **3 new live examples** in `examples/` require paper trading credentials
  (`IBKR_GATEWAY`, `IBKR_USERNAME`, `IBKR_PASSWORD`). All operations are
  read-only.
  - `live-portfolio/`: Session.Initialize, Account.List, Portfolio
    Positions/Ledger/Summary
  - `options-chain/`: symbol search + options strike lookup for a given month
  - `screener/`: ScannerParameters discovery + live market scanner
- **Mock examples** (`mock/`, `portfolio/`, `orders/`, `marketdata-streaming/`,
  `models/`, `middleware/`) target the in-repo mock gateway and need no
  credentials.

### Deprecated

- `ScannerManager.ScannerResults` — use `Scanner().ScannerResults` via the
  `ScannerManager` returned by `Client.Scanner()`
- `PortfolioManager.Subaccounts` — use `SubaccountsPager` instead
- `FYIManager.FYIs` — use `FYIsPager` instead
- `PerformanceManager.Transactions` — use `TransactionsPager` instead
- `ModelManager.Models` — use `ModelsPager` instead
- `DefaultServerURL` — use `DefaultGatewayURL` instead

## [0.3.0] - 2026-09-18

### Breaking

This release contains breaking API changes. All symbols are frozen for the v0.x
series; breaking changes will not occur after v1.0.0.

#### `Get` prefix removed from 57 methods

Go convention is to omit the `Get` prefix when the receiver already provides
context. All affected methods are on manager types.

| Manager | Old name | New name |
|---------|----------|----------|
| `ScannerManager` | `GetScannerParameters` | `ScannerParameters` |
| `ScannerManager` | `GetScannerResults` | `ScannerResults` |
| `TradingAccountManager` | `GetAccountOwners` | `AccountOwners` |
| `TradingAccountManager` | `GetDynamicAccounts` | `DynamicAccounts` |
| `TradingAccountManager` | `GetFundSummary` | `FundSummary` |
| `TradingAccountManager` | `GetBalanceSummary` | `BalanceSummary` |
| `TradingAccountManager` | `GetMarginSummary` | `MarginSummary` |
| `TradingAccountManager` | `GetAccountMarketSummary` | `AccountMarketSummary` |
| `TradingAccountManager` | `GetBrokerageAccounts` | `BrokerageAccounts` |
| `ModelManager` | `GetModelPresets` | `ModelPresets` |
| `ModelManager` | `GetAccountsInModel` | `AccountsInModel` |
| `ModelManager` | `GetInvestedAccountsInModel` | `InvestedAccountsInModel` |
| `ModelManager` | `GetAllModels` | `AllModels` |
| `ModelManager` | `GetAllModelPositions` | `AllModelPositions` |
| `ModelManager` | `GetModelSummarySingle` | `ModelSummarySingle` |
| `SessionManager` | `GetSessionValidation` | `SessionValidation` |
| `SessionManager` | `GetSessionToken` | `SessionToken` |
| `WatchlistManager` | `GetSpecificWatchlist` | `SpecificWatchlist` |
| `WatchlistManager` | `GetAllWatchlists` | `AllWatchlists` |
| `AllocationManager` | `GetAllocatableSubaccounts` | `AllocatableSubaccounts` |
| `AllocationManager` | `GetAllocationGroups` | `AllocationGroups` |
| `AllocationManager` | `GetSingleAllocationGroup` | `SingleAllocationGroup` |
| `AllocationManager` | `GetAllocationPresets` | `AllocationPresets` |
| `FYIManager` | `GetFYIDelivery` | `FYIDelivery` |
| `FYIManager` | `GetFYIDisclaimers` | `FYIDisclaimers` |
| `FYIManager` | `GetAllFYIs` | `AllFYIs` |
| `FYIManager` | `GetFYISettings` | `FYISettings` |
| `FYIManager` | `GetUnreadFYIs` | `UnreadFYIs` |
| `TradeManager` | `GetTradingSchedule` | `TradingSchedule` |
| `TradeManager` | `GetAlgosByInstrument` | `AlgosByInstrument` |
| `TradeManager` | `GetInfoAndRules` | `InfoAndRules` |
| `TradeManager` | `GetCurrencyPairs` | `CurrencyPairs` |
| `TradeManager` | `GetExchangeRates` | `ExchangeRates` |
| `TradeManager` | `GetBondFilters` | `BondFilters` |
| `TradeManager` | `GetContractInfo` | `SecDefInfos` |
| `TradeManager` | `GetContractSymbolsFromBody` | `ContractSymbolsFromBody` |
| `TradeManager` | `GetConidsByExchange` | `ConidsByExchange` |
| `TradeManager` | `GetFutureBySymbol` | `FutureBySymbol` |
| `TradeManager` | `GetInstrumentDefinition` | `InstrumentDefinition` |
| `TradeManager` | `GetTradingScheduleBySymbol` | `TradingScheduleBySymbol` |
| `TradeManager` | `GetStockBySymbol` | `StockBySymbol` |
| `PerformanceManager` | `GetPerformanceAllPeriods` | `PerformanceAllPeriods` |
| `PerformanceManager` | `GetSinglePerformancePeriod` | `SinglePerformancePeriod` |
| `PerformanceManager` | `GetTransactions` | `Transactions` |
| `AlertManager` | `GetAlertDetails` | `AlertDetail` |
| `AlertManager` | `GetMtaDetails` | `MtaDetail` |
| `AlertManager` | `GetAllAlerts` | `AllAlerts` |
| `ForecastManager` | `GetForecastCategories` | `ForecastCategories` |
| `ForecastManager` | `GetForecastContract` | `ForecastContract` |
| `ForecastManager` | `GetForecastMarkets` | `ForecastMarkets` |
| `ForecastManager` | `GetForecastRules` | `ForecastRules` |
| `ForecastManager` | `GetForecastSchedule` | `ForecastSchedule` |
| `PortfolioManager` | `GetAllAccountsForConid` | `AllAccountsForConid` |
| `PortfolioManager` | `GetManySubaccounts` | `ManySubaccounts` |
| `PortfolioManager` | `GetComboPositions` | `ComboPositions` |
| `PortfolioManager` | `GetUncachedPositions` | `UncachedPositions` |
| `RESTRequests` | `GetStatus` | `Status` |

#### Type consistency: `AccountID` and `ConID`

- `AccountID` (type `string`) is now used consistently for all account-ID fields
  across 21 struct fields in response types. Callers passing raw `string`
  account IDs may need to wrap with `AccountID(...)`.
- `ConID` (type `int`) is now used consistently for `ConID` fields in banking
  transfer request types (`AssetTransferRequest`, `PositionV2Request`,
  `InternalAssetTransferRequest`).

#### `float32` → `int64` for banking IDs

`BankInstructionCreateRequest.ClientInstructionID` and `TransferResult`
fields (`ClientInstructionID`, `InstructionID`, `IbReferenceID`) are now
`int64` instead of `float32`. These are integer IDs, not floating-point values.

#### Go initialism casing fixed

14 symbols renamed to follow Go convention for initialisms:
`EchoHTTPSResponse`, `ListEchoHTTPS`, `SignedJWTEchoRequest`,
`SignedJWTEchoResponse`, `CreateEchoSignedJWT`, `SSOBrowserSessionRequest`,
`SSOSessionRequest`, `CSVApplyResponse`, `CSVVerifyRequest`,
`CSVVerifyResponse`, `RealizedPnL`, `UnrealizedPnL`,
`RequestAccessToken`, `RequestLiveSessionToken`, `RequestTempToken`.

### Deprecated

- `ErrStreamDisconnected` and `ErrStreamReconnected` are removed. Use
  `ErrWSDisconnected` and `ErrWSReconnected` instead.

### Internal

- `RESTInstructions` type removed (zero methods, dead code).

## [0.2.0] - 2026-09-18

### Added

- **Three new ADRs:** ADR 0015 (public API surface and stability contract),
  ADR 0016 (error handling philosophy), ADR 0017 (logging interface).
- **Godoc sprint:** 65 previously undocumented exported symbols across 7 files
  now have godoc comments, including multi-line docs with usage examples for
  complex types (`RESTClientInstruction`, `RESTTransaction`, `Form`, `Bank`,
  `CashBalanceDetail`, `ListRequestsFilter`, `RESTRequestSummary`,
  `RESTAccountSummary`, `RESTAccountStatus`, `RegistrationTaskItem`,
  `TaxVoucherDividend`, `TaxVoucherState`).
- **4 new examples:** `portfolio`, `marketdata-streaming`, `orders`, `models`
  with updated `examples/README.md`.
- **Integration test scaffold:** `test/integration_test.go` with
  `//go:build integration` gate and env-gated skip; `make test-integration`
  target.
- **Shape conformance guard:** `TestFixtureShapeConformance` (180 passing
  tests) in `pkg/ibkr/fixture_shape_conformance_test.go` using
  `fixtures.go` `All()` method.
- **Spec drift detection:** `.github/workflows/spec-drift.yml` (daily cron +
  workflow_dispatch) with `scripts/check_spec_version.py`.
- **`docs/STABILITY.md`:** user-facing stability contract derived from ADR 0015.

### Changed

- **ADR 0008 compliance (banking types):** `AssetTransferRequest`,
  `CashTransferRequest`, `InternalAssetTransferRequest`,
  `InternalCashTransferRequest`, and `PositionV2Request` field types changed
  from `float32` to `string` for `ClientInstructionID`, `Quantity`, `Amount`,
  and `TransferQuantity`. Call sites use `strToF32`/`strPtrToF32Ptr` helpers
  at the generated-client boundary.

### Deprecated

- `ErrStreamDisconnected` — use `ErrWSDisconnected` instead (removed in v0.3.0).
- `ErrStreamReconnected` — use `ErrWSReconnected` instead (removed in v0.3.0).

### Fixed

- Remaining ADR 0008 violations in `rest_banking.go` call sites (pos.Quantity,
  req.TransferPrice, InternalCashTransferInstruction.ClientInstructionID).

### Internal

- 4 wrapper types unexported: `LoginMessagesWrapper` →
  `loginMessagesWrapper`, `AccountStatusBulkWrapper` →
  `accountStatusBulkWrapper`, `Au10TixWrapper` → `au10tixWrapper`,
  `RegistrationTasksWrapper` → `registrationTasksWrapper`.
- 8 stale `FIX:` comments removed from REST wrappers.

## [0.1.1] - 2026-09-17

### Fixed

- **Codegen root cause for nil-`interface{}` panics.** `scripts/patch_spec.py`
  now applies a fourth spec patch (defect 4): inline query parameters that omit
  `type` (`type: null`) are retyped to `type: string`. `oapi-codegen` therefore
  emits a concrete `string` (or named string enum) instead of a bare
  `interface{}` for the affected 22 parameters, eliminating the nil-panic class
  at the source. The generated `client/client.gen.go` no longer contains the
  post-generation nil guards, and `scripts/patch_gen.py` is now a documented
  no-op kept for backward compatibility.
- Updated SDK callers for the retyped parameters: `TradeManager`
  `GetTradingScheduleBySymbol` casts `assetClass` to the generated enum type,
  and `FYIManager.ModifyFYIEmails` serializes the `enabled` flag with
  `strconv.FormatBool`.

## [0.1.0] - 2026-09-17

### Added

- **Complete public SDK (`pkg/ibkr`)** covering both API surfaces — **185/185
  operations**: 115 on the Client Portal API (`ssoBearer`) and 70 on the IB REST
  API (`oauth2Bearer`). Every manager is registered on `Client` with an accessor.
  - **CPAPI managers:** `SessionManager` (`Session`), `AccountManager`
    (`Account`), `PortfolioManager` (`Portfolio`), `TradeManager` (`Trade`,
    including contracts), `MarketDataManager` (`MarketData`: snapshot, history,
    and WebSocket `Subscribe`), `TradingAccountManager` (`TradingAccount`),
    `AlertManager` (`Alert`), `ForecastManager` (`Forecast`, event contracts),
    `ScannerManager` (`Scanner`), `AllocationManager` (`Allocation`),
    `ModelManager` (`Model`), `FYIManager` (`FYI`, notifications),
    `OAuthManager` (`OAuth`, OAuth1), `WatchlistManager` (`Watchlist`), and
    `PerformanceManager` (`Performance`, PortfolioAnalyst).
  - **IB REST surface (`Client.REST()`):** `RESTSurface` with `RESTAccounts`,
    `RESTBanking` (banking and transfers), `RESTRequests`, `RESTUtilities`,
    `RESTRestrictions`, `RESTBalances`, `RESTTaxVouchers`, `RESTStatements`,
    `RESTTaxDocuments`, `RESTTradeConfirmations`, `RESTSSOSessions`, and
    `RESTEcho`.
- **Internal subsystems (`internal/`):** transport middleware chain, session
  state machine with tickle/heartbeat, WebSocket connection management and
  reconnect, OAuth2 token source (`fetchWithSecret`, `fetchWithJWTAssertion`),
  per-endpoint and global rate limiting, circuit breaker, retry with
  `Retry-After` handling, and observability (structured logging, redaction,
  telemetry hooks).
- **JWT / OAuth2 infrastructure:** `internal/jwt.go` (RS256 `private_key_jwt`
  assertions with PKCS8/PKCS1 PEM parsing) and `internal/oauth.go` (access and
  refresh tokens, single-flight refresh, refresh-token rotation).
- **Unified logging framework:** structured `log/slog` output with
  `ibkr.<subsystem>` message prefixes, per-component nil-safety (a logger is
  never nil after construction), and redaction of tokens, cookies, and
  `Authorization` headers. See [docs/LOGGING.md](./docs/LOGGING.md).
- **Dependency-free metrics layer:** an OTel-shaped `Metrics` interface with
  `InMemoryMetrics`/`NopMetrics`, `WithMetrics`, and documented `ibkr.*` metric
  names for HTTP, orders, rate limiting, circuit breaking, WebSocket, and OAuth.
  See [docs/OBSERVABILITY.md](./docs/OBSERVABILITY.md) and
  [ADR 0013](./docs/adr/0013-metrics.md).
- **In-repo mock IBKR gateway:** `internal/mockgateway` serves both API surfaces
  and the WebSocket stream (185/185 operations) with scriptable faults, request
  recording, deterministic fixtures, and shape-level auth, with no new
  dependency; `cmd/ibkr-mock-gateway`, `examples/mock`, and `make mock-gateway`
  drive it. See [docs/MOCK-GATEWAY.md](./docs/MOCK-GATEWAY.md) and
  [ADR 0014](./docs/adr/0014-mock-gateway.md). Manager and WebSocket tests now
  run against it. The mock is a development/testing aid, not a conformance
  suite.
- **Benchmarks (`pkg/ibkr/benchmark_test.go`, `internal/benchmark_test.go`):**
  `testing.B` benchmarks for JSON encode/decode across 20 public response types,
  HTTP round-trip latency (mock gateway), WebSocket subscribe/unsubscribe, and
  session init. Baseline stored in `benchmark.baseline`. See
  [docs/OBSERVABILITY.md](./docs/OBSERVABILITY.md).
- **Fuzz tests (`pkg/ibkr/fuzz_test.go`, `internal/fuzz_test.go`):** 47 `testing.F`
  fuzz functions covering all major public JSON decode types, plus response decode
  fuzzing across all 185 op response shapes using mock gateway fixtures. No
  panics found; corpus generated in `testdata/fuzz/`.
- **Benchmark CI gate (`.github/workflows/ci.yml`):** `benchmarks` job compares
  current results against `benchmark.baseline`; fails on >10% regression in
  ns/op. `scripts/bench_compare.go` is pure stdlib.
- **Coverage badge (`.github/workflows/ci.yml`):** `go test -coverprofile=coverage.out`
  uploaded to codecov.io via `codecov/codecov-action@v4`; badge rendered in all
  6 README translations.
- **Pre-commit CI gate (`.github/workflows/pre-commit.yml`):** `gofmt -s -l .`,
  `go vet ./...`, and `make check` run on every push and PR; parallel to the main
  CI workflow (~30s).
- **FUNDING.yml (`.github/FUNDING.yml`):** GitHub Sponsors link for `shing1211`.
- **Docs website (`docs/index.html`):** branded GitHub Pages landing page served from
  the `docs/` folder at `https://shing1211.github.io/ibkrapi4go/`; pure HTML/CSS,
  zero new dependencies, zero build step. Enable at `Settings → Pages → Source:
  main branch, /docs folder`.
- Initialized the Go module (`go.mod`) and committed the generated OpenAPI
  client (`client/client.gen.go`, package `client`), with a deterministic SPDX
  header and a reproducible `make codegen-verify` check.
- Project documentation set: `SPEC.md`, `ARCHITECTURE.md`, `AUTH.md`,
  `SESSIONS.md`, `ERRORS.md`, `RATE-LIMITING.md`, `STREAMING.md`, `LOGGING.md`,
  `OBSERVABILITY.md`, `CODEGEN.md`, `TESTING.md`, `MOCK-GATEWAY.md`,
  `RELEASING.md`, `CONFIG.md`, `GLOSSARY.md`, `ROADMAP.md`.
- Architecture Decision Records (`docs/adr/0001`–`0014`).
- Per-module design contracts (`docs/design/01`–`09`).
- `scripts/patch_spec.py`: generalized OpenAPI spec patcher (path-parameter
  reconciliation, operation-ID de-duplication, Go type-name collision fixes).
- `scripts/validate_codegen.sh`: reproducible codegen validation. OpenAPI
  codegen against IBKR Web API v2.39.0 succeeds after three classes of spec
  patches; the generated client is ~72k LOC and compiles cleanly. See
  [docs/CODEGEN.md](./docs/CODEGEN.md).
- `scripts/patch_gen.py`: deterministic post-generation fixups. It wraps
  unguarded `runtime.StyleParamWithOptions` calls for bare-`interface{}` query
  parameters in a nil guard, and is invoked by `scripts/codegen.sh` and
  `scripts/validate_codegen.sh` so a fresh generation matches the committed
  output byte-for-byte.
- Full Apache License 2.0 text, `NOTICE`, `THIRD_PARTY_NOTICES.md`,
  `DISCLAIMER.md`.
- Code of Conduct, Security Policy, Support and Governance documents.
- README translations: 简体中文 (`zh-Hans`), 繁體中文 (`zh-Hant`), 日本語 (`ja`),
  한국어 (`ko`), Español (`es`), plus `TRANSLATING.md` and a CI consistency
  check (`scripts/check_i18n.py`). `README.zh-CN.md` is now a redirect stub.

### Changed

- README and `docs/` set updated to describe 185/185 coverage across both API
  surfaces, replacing pre-implementation "planned" language.

### Fixed

- Removed the stale claim that the public SDK (`pkg/ibkr`) and `internal/`
  packages are "not yet implemented"; both are complete.
- Corrected the documented dependency set: `testify` is not used;
  `go.uber.org/goleak` is the test-only dependency.
- Fixed nil-interface{} panics in the generated client (`client/client.gen.go`):
  16 nil guards added to request builders for `GetContractInfo` (6 fields),
  `GetAllFyis` (3 fields), `GetAssetAllocation` (1 field),
  `GetPaginatedPositions` (3 fields), `GetConidsByExchange` (1 field),
  `GetTradingSchedule2` (1 field), and `ModifyFyiEmails` (1 field).
  Root cause: spec-patch produces `interface{}` with `omitempty` for optional
  non-pointer params; the codegen template did not guard against nil. The guards
  now live in `scripts/patch_gen.py`, so `make codegen` reproduces them and
  `make codegen-verify` passes.
- Fixed 8 REST wrapper/decode mismatches in `pkg/ibkr/rest.go` and
  `pkg/ibkr/rest_utilities.go`: `TaxDocuments.Generate`,
  `TaxVouchers.CreateRequests`, `ActiveCountries`, `AvailableYears`,
  `Dividends`, `Utilities.Enumerations`, `ComplexAssetTransferBrokers`,
  and `RequiredForms`.

[Unreleased]: https://github.com/shing1211/ibkrapi4go/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/shing1211/ibkrapi4go/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/shing1211/ibkrapi4go/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/shing1211/ibkrapi4go/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/shing1211/ibkrapi4go/releases/tag/v0.1.0
