# Migrating from v0.x to v1.0

This guide covers every breaking change between v0.1.0 and v1.0.0. Run the
codemod script first — it handles the majority of mechanical renames — then
fix the remaining items by hand.

## Quick start

```bash
# 1. Run the codemod (handles Get-prefix, initialisms, error renames)
scripts/codemod.sh

# 2. Review what changed
git diff

# 3. Fix any remaining type-level changes by hand (see below)
# 4. Compile and run tests
go build ./...
go test ./...
```

## Breaking changes

### 1. `Get*` prefix removed (57 methods)

All `Get*` methods on manager types have been renamed to drop the `Get` prefix.
Go convention omits the prefix when the receiver already provides context.

| # | Manager | Old name | New name |
|---|---------|----------|----------|
| 1 | `ScannerManager` | `GetScannerParameters` | `ScannerParameters` |
| 2 | `ScannerManager` | `GetScannerResults` | `ScannerResults` |
| 3 | `TradingAccountManager` | `GetAccountOwners` | `AccountOwners` |
| 4 | `TradingAccountManager` | `GetDynamicAccounts` | `DynamicAccounts` |
| 5 | `TradingAccountManager` | `GetFundSummary` | `FundSummary` |
| 6 | `TradingAccountManager` | `GetBalanceSummary` | `BalanceSummary` |
| 7 | `TradingAccountManager` | `GetMarginSummary` | `MarginSummary` |
| 8 | `TradingAccountManager` | `GetAccountMarketSummary` | `AccountMarketSummary` |
| 9 | `TradingAccountManager` | `GetBrokerageAccounts` | `BrokerageAccounts` |
| 10 | `ModelManager` | `GetModelPresets` | `ModelPresets` |
| 11 | `ModelManager` | `GetAccountsInModel` | `AccountsInModel` |
| 12 | `ModelManager` | `GetInvestedAccountsInModel` | `InvestedAccountsInModel` |
| 13 | `ModelManager` | `GetAllModels` | `AllModels` |
| 14 | `ModelManager` | `GetAllModelPositions` | `AllModelPositions` |
| 15 | `ModelManager` | `GetModelSummarySingle` | `ModelSummarySingle` |
| 16 | `SessionManager` | `GetSessionValidation` | `SessionValidation` |
| 17 | `SessionManager` | `GetSessionToken` | `SessionToken` |
| 18 | `WatchlistManager` | `GetSpecificWatchlist` | `SpecificWatchlist` |
| 19 | `WatchlistManager` | `GetAllWatchlists` | `AllWatchlists` |
| 20 | `AllocationManager` | `GetAllocatableSubaccounts` | `AllocatableSubaccounts` |
| 21 | `AllocationManager` | `GetAllocationGroups` | `AllocationGroups` |
| 22 | `AllocationManager` | `GetSingleAllocationGroup` | `SingleAllocationGroup` |
| 23 | `AllocationManager` | `GetAllocationPresets` | `AllocationPresets` |
| 24 | `FYIManager` | `GetFYIDelivery` | `FYIDelivery` |
| 25 | `FYIManager` | `GetFYIDisclaimers` | `FYIDisclaimers` |
| 26 | `FYIManager` | `GetAllFYIs` | `AllFYIs` |
| 27 | `FYIManager` | `GetFYISettings` | `FYISettings` |
| 28 | `FYIManager` | `GetUnreadFYIs` | `UnreadFYIs` |
| 29 | `TradeManager` | `GetTradingSchedule` | `TradingSchedule` |
| 30 | `TradeManager` | `GetAlgosByInstrument` | `AlgosByInstrument` |
| 31 | `TradeManager` | `GetInfoAndRules` | `InfoAndRules` |
| 32 | `TradeManager` | `GetCurrencyPairs` | `CurrencyPairs` |
| 33 | `TradeManager` | `GetExchangeRates` | `ExchangeRates` |
| 34 | `TradeManager` | `GetBondFilters` | `BondFilters` |
| 35 | `TradeManager` | `GetContractInfo` | `SecDefInfos` |
| 36 | `TradeManager` | `GetContractSymbolsFromBody` | `ContractSymbolsFromBody` |
| 37 | `TradeManager` | `GetConidsByExchange` | `ConidsByExchange` |
| 38 | `TradeManager` | `GetFutureBySymbol` | `FutureBySymbol` |
| 39 | `TradeManager` | `GetInstrumentDefinition` | `InstrumentDefinition` |
| 40 | `TradeManager` | `GetTradingScheduleBySymbol` | `TradingScheduleBySymbol` |
| 41 | `TradeManager` | `GetStockBySymbol` | `StockBySymbol` |
| 42 | `PerformanceManager` | `GetPerformanceAllPeriods` | `PerformanceAllPeriods` |
| 43 | `PerformanceManager` | `GetSinglePerformancePeriod` | `SinglePerformancePeriod` |
| 44 | `PerformanceManager` | `GetTransactions` | `Transactions` |
| 45 | `AlertManager` | `GetAlertDetails` | `AlertDetail` |
| 46 | `AlertManager` | `GetMtaDetails` | `MtaDetail` |
| 47 | `AlertManager` | `GetAllAlerts` | `AllAlerts` |
| 48 | `ForecastManager` | `GetForecastCategories` | `ForecastCategories` |
| 49 | `ForecastManager` | `GetForecastContract` | `ForecastContract` |
| 50 | `ForecastManager` | `GetForecastMarkets` | `ForecastMarkets` |
| 51 | `ForecastManager` | `GetForecastRules` | `ForecastRules` |
| 52 | `ForecastManager` | `GetForecastSchedule` | `ForecastSchedule` |
| 53 | `PortfolioManager` | `GetAllAccountsForConid` | `AllAccountsForConid` |
| 54 | `PortfolioManager` | `GetManySubaccounts` | `ManySubaccounts` |
| 55 | `PortfolioManager` | `GetComboPositions` | `ComboPositions` |
| 56 | `PortfolioManager` | `GetUncachedPositions` | `UncachedPositions` |
| 57 | `RESTRequests` | `GetStatus` | `Status` |

**Note:** The rename is the Go method name only. The underlying generated
client method (e.g. `client.GetScannerParameters`) is not changed — it is
internal and not part of the public API.

### 2. Go initialism casing fixed (15 symbols)

Go convention requires that common initialisms (`HTTP`, `JWT`, `SSO`, `CSV`,
`PnL`) are all-caps or follow established casing. 15 symbols were renamed:

| # | Old name | New name | Kind |
|---|----------|----------|------|
| 1 | `EchoHttpsResponse` | `EchoHTTPSResponse` | type |
| 2 | `ListEchoHttps` | `ListEchoHTTPS` | method |
| 3 | `SignedJwtEchoRequest` | `SignedJWTEchoRequest` | type |
| 4 | `SignedJwtEchoResponse` | `SignedJWTEchoResponse` | type |
| 5 | `CreateEchoSignedJwt` | `CreateEchoSignedJWT` | method |
| 6 | `SsoBrowserSessionRequest` | `SSOBrowserSessionRequest` | type |
| 7 | `SsoSessionRequest` | `SSOSessionRequest` | type |
| 8 | `CsvApplyResponse` | `CSVApplyResponse` | type |
| 9 | `CsvVerifyRequest` | `CSVVerifyRequest` | type |
| 10 | `CsvVerifyResponse` | `CSVVerifyResponse` | type |
| 11 | `RealizedPnl` | `RealizedPnL` | type field |
| 12 | `UnrealizedPnl` | `UnrealizedPnL` | type field |
| 13 | `ReqAccessToken` | `RequestAccessToken` | method |
| 14 | `ReqLiveSessionToken` | `RequestLiveSessionToken` | method |
| 15 | `ReqTempToken` | `RequestTempToken` | method |

> **Note:** The CHANGELOG lists 14 symbols. This guide documents 15 — the
> discrepancy comes from `ReqTempToken` → `RequestTempToken` being counted
> separately from the initialism group. All 15 are covered by the codemod.

### 3. Deprecated error sentinels removed

Two error sentinels were deprecated in v0.2.0 and removed in v0.3.0:

| Old name | New name |
|----------|----------|
| `ErrStreamDisconnected` | `ErrWSDisconnected` |
| `ErrStreamReconnected` | `ErrWSReconnected` |

### 4. Type consistency: `AccountID` and `ConID`

`AccountID` (type `string`) is now used consistently for all account-ID fields
across 21 struct fields in response types. Callers passing raw `string` account
IDs may need to wrap:

```go
// Before (v0.x)
id := "U1234567"

// After (v1.0)
id := ibkr.AccountID("U1234567")
```

`ConID` (type `int`) is now used consistently for `ConID` fields in banking
transfer request types (`AssetTransferRequest`, `PositionV2Request`,
`InternalAssetTransferRequest`).

### 5. `float32` → `int64` for banking IDs

`BankInstructionCreateRequest.ClientInstructionID` and `TransferResult` fields
(`ClientInstructionID`, `InstructionID`, `IbReferenceID`) are now `int64`
instead of `float32`. These are integer IDs, not floating-point values.

```go
// Before (v0.x)
req.ClientInstructionID = float32(12345)

// After (v1.0)
req.ClientInstructionID = 12345
```

### 6. Internal types unexported

The following wrapper types were unexported (lowercased) in v0.2.0. Since they
were in `internal/`, this is not a public API break but is listed for
completeness:

- `LoginMessagesWrapper` → `loginMessagesWrapper`
- `AccountStatusBulkWrapper` → `accountStatusBulkWrapper`
- `Au10TixWrapper` → `au10tixWrapper`
- `RegistrationTasksWrapper` → `registrationTasksWrapper`

## What the codemod handles automatically

The `scripts/codemod.sh` script performs all mechanical text replacements:

- **57 Get-prefix removals** — safe word-boundary replacement in `.go` files
- **15 Go initialism casing fixes** — safe word-boundary replacement
- **2 error sentinel renames** — safe word-boundary replacement

**Total: 74 renames** handled automatically.

## What requires manual changes

The following changes are **type-level** and cannot be handled by simple text
replacement:

1. **`AccountID` wrapping** — if you pass raw `string` values where
   `AccountID` is now required, you must wrap them with `ibkr.AccountID(...)`.
   This affects 21 struct fields across response types.

2. **`ConID` type change** — banking transfer request types now use `int`
   instead of whatever type you were passing. Verify your code compiles.

3. **`float32` → `int64` for banking IDs** — change `float32` literals/variables
   to `int64`. The codemod cannot distinguish type usage from other text.

4. **`RESTInstructions` type removed** — if you referenced this type, remove the
   reference. It had zero methods and was dead code.

## Compatibility shims

None — all renames are backward-incompatible. Callers must update to the new
names. There are no type aliases or wrapper functions for backward compatibility.

## Verifying your migration

```bash
# Compile check
go build ./...

# Run tests
go test ./...

# Check for leftover old names
grep -rn 'GetScannerParameters\|GetScannerResults\|EchoHttps\|SignedJwtEcho\|SsoBrowser\|SsoSession\|CsvApply\|CsvVerify\|RealizedPnl\|UnrealizedPnl\|ReqAccessToken\|ReqLiveSessionToken\|ReqTempToken\|ErrStreamDisconnected\|ErrStreamReconnected' --include='*.go' .

# Verify docs
make docs-check
```
