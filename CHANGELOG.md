# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
  12 nil guards added to request builders for `GetContractInfo` (6 fields),
  `GetConidsByExchange` (1 field), `GetAllFyis` (3 fields),
  `GetTradingSchedule2` (1 field), and `ModifyFyiEmails` (1 field).
  Root cause: spec-patch produces `interface{}` with `omitempty` for optional
  non-pointer params; codegen template did not guard against nil.
- Fixed 8 REST wrapper/decode mismatches in `pkg/ibkr/rest.go` and
  `pkg/ibkr/rest_utilities.go`: `TaxDocuments.Generate`,
  `TaxVouchers.CreateRequests`, `ActiveCountries`, `AvailableYears`,
  `Dividends`, `Utilities.Enumerations`, `ComplexAssetTransferBrokers`,
  and `RequiredForms`.

[Unreleased]: https://github.com/shing1211/ibkrapi4go/commits/main
