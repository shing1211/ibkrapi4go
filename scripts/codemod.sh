#!/usr/bin/env bash
# Copyright 2026 shing1211
# SPDX-License-Identifier: Apache-2.0
#
# codemod.sh — automated renames for v0.x → v1.0 migration.
#
# Usage:  scripts/codemod.sh [directory]
#         Default: current directory.
#
# What it handles (74 renames total):
#   - 57 Get-prefix removals on manager methods
#   - 15 Go initialism casing fixes (types + methods)
#   - 2 deprecated error sentinel renames
#
# What it does NOT handle (manual changes required):
#   - AccountID / ConID type-consistency wrapping
#   - float32 → int64 for banking IDs
#   - RESTInstructions type removal

set -euo pipefail

DIR="${1:-.}"

if [ ! -d "$DIR" ]; then
  echo "error: directory '$DIR' does not exist" >&2
  exit 1
fi

echo "Applying codemod renames to: $DIR"
echo "This modifies .go files in-place. Review changes with 'git diff'."
echo ""

# ── 1. Get-prefix removals (57 methods) ──────────────────────────────────
# All word-boundary safe: old name appears only as an identifier.

GET_RENAMES=(
  "GetScannerParameters=ScannerParameters"
  "GetScannerResults=ScannerResults"
  "GetAccountOwners=AccountOwners"
  "GetDynamicAccounts=DynamicAccounts"
  "GetFundSummary=FundSummary"
  "GetBalanceSummary=BalanceSummary"
  "GetMarginSummary=MarginSummary"
  "GetAccountMarketSummary=AccountMarketSummary"
  "GetBrokerageAccounts=BrokerageAccounts"
  "GetModelPresets=ModelPresets"
  "GetAccountsInModel=AccountsInModel"
  "GetInvestedAccountsInModel=InvestedAccountsInModel"
  "GetAllModels=AllModels"
  "GetAllModelPositions=AllModelPositions"
  "GetModelSummarySingle=ModelSummarySingle"
  "GetSessionValidation=SessionValidation"
  "GetSessionToken=SessionToken"
  "GetSpecificWatchlist=SpecificWatchlist"
  "GetAllWatchlists=AllWatchlists"
  "GetAllocatableSubaccounts=AllocatableSubaccounts"
  "GetAllocationGroups=AllocationGroups"
  "GetSingleAllocationGroup=SingleAllocationGroup"
  "GetAllocationPresets=AllocationPresets"
  "GetFYIDelivery=FYIDelivery"
  "GetFYIDisclaimers=FYIDisclaimers"
  "GetAllFYIs=AllFYIs"
  "GetFYISettings=FYISettings"
  "GetUnreadFYIs=UnreadFYIs"
  "GetTradingSchedule=TradingSchedule"
  "GetAlgosByInstrument=AlgosByInstrument"
  "GetInfoAndRules=InfoAndRules"
  "GetCurrencyPairs=CurrencyPairs"
  "GetExchangeRates=ExchangeRates"
  "GetBondFilters=BondFilters"
  "GetContractInfo=SecDefInfos"
  "GetContractSymbolsFromBody=ContractSymbolsFromBody"
  "GetConidsByExchange=ConidsByExchange"
  "GetFutureBySymbol=FutureBySymbol"
  "GetInstrumentDefinition=InstrumentDefinition"
  "GetTradingScheduleBySymbol=TradingScheduleBySymbol"
  "GetStockBySymbol=StockBySymbol"
  "GetPerformanceAllPeriods=PerformanceAllPeriods"
  "GetSinglePerformancePeriod=SinglePerformancePeriod"
  "GetTransactions=Transactions"
  "GetAlertDetails=AlertDetail"
  "GetMtaDetails=MtaDetail"
  "GetAllAlerts=AllAlerts"
  "GetForecastCategories=ForecastCategories"
  "GetForecastContract=ForecastContract"
  "GetForecastMarkets=ForecastMarkets"
  "GetForecastRules=ForecastRules"
  "GetForecastSchedule=ForecastSchedule"
  "GetAllAccountsForConid=AllAccountsForConid"
  "GetManySubaccounts=ManySubaccounts"
  "GetComboPositions=ComboPositions"
  "GetUncachedPositions=UncachedPositions"
  "GetStatus=Status"
)

# ── 2. Go initialism casing fixes (15 symbols) ───────────────────────────

INITIALISM_RENAMES=(
  "EchoHttpsResponse=EchoHTTPSResponse"
  "ListEchoHttps=ListEchoHTTPS"
  "SignedJwtEchoRequest=SignedJWTEchoRequest"
  "SignedJwtEchoResponse=SignedJWTEchoResponse"
  "CreateEchoSignedJwt=CreateEchoSignedJWT"
  "SsoBrowserSessionRequest=SSOBrowserSessionRequest"
  "SsoSessionRequest=SSOSessionRequest"
  "CsvApplyResponse=CSVApplyResponse"
  "CsvVerifyRequest=CSVVerifyRequest"
  "CsvVerifyResponse=CSVVerifyResponse"
  "RealizedPnl=RealizedPnL"
  "UnrealizedPnl=UnrealizedPnL"
  "ReqAccessToken=RequestAccessToken"
  "ReqLiveSessionToken=RequestLiveSessionToken"
  "ReqTempToken=RequestTempToken"
)

# ── 3. Deprecated error sentinel renames (2) ─────────────────────────────

ERROR_RENAMES=(
  "ErrStreamDisconnected=ErrWSDisconnected"
  "ErrStreamReconnected=ErrWSReconnected"
)

# ── Apply all renames ────────────────────────────────────────────────────

ALL_RENAMES=("${GET_RENAMES[@]}" "${INITIALISM_RENAMES[@]}" "${ERROR_RENAMES[@]}")

count=0
for entry in "${ALL_RENAMES[@]}"; do
  old="${entry%%=*}"
  new="${entry#*=}"
  # \b ensures we only match whole identifiers, not substrings.
  # This is a GNU sed extension; works on Linux and macOS (with GNU sed installed).
  find "$DIR" -name '*.go' -not -path '*/client/*' -exec \
    sed -i "s/\b${old}\b/${new}/g" {} +
  count=$((count + 1))
done

echo "Applied $count rename rules across all .go files (excluding client/)."
echo ""
echo "Next steps:"
echo "  1. Review: git diff"
echo "  2. Fix AccountID/ConID type wrappers by hand (see docs/MIGRATION.md §4)"
echo "  3. Fix float32 → int64 for banking IDs by hand (see docs/MIGRATION.md §5)"
echo "  4. Build and test: go build ./... && go test ./..."
