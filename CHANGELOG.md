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
- Initialized the Go module (`go.mod`) and committed the generated OpenAPI
  client (`client/client.gen.go`, package `client`), with a deterministic SPDX
  header and a reproducible `make codegen-verify` check.
- Project documentation set: `SPEC.md`, `ARCHITECTURE.md`, `AUTH.md`,
  `SESSIONS.md`, `ERRORS.md`, `RATE-LIMITING.md`, `STREAMING.md`, `LOGGING.md`,
  `CODEGEN.md`, `TESTING.md`, `RELEASING.md`, `CONFIG.md`, `GLOSSARY.md`,
  `ROADMAP.md`.
- Architecture Decision Records (`docs/adr/0001`–`0012`).
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

[Unreleased]: https://github.com/shing1211/ibkrapi4go/commits/main
