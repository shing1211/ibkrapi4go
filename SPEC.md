# SPEC.md — IBKR Web API v2.39 Specification Reference

> Auto-generated from: `https://api.ibkr.com/gw/api/v3/api-docs`
> Spec version: 2.39.0 | Endpoints: 185 | Schemas: 443
> Generated: 2026-09-16

---

## Servers

| Environment | URL | Notes |
|-------------|-----|-------|
| Local | `https://localhost:5000` | Client Portal Gateway |
| Production | `https://api.ibkr.com` | Live trading |
| Sandbox | `https://qa.interactivebrokers.com` | Testing |

---

## Endpoint Index (185 total)

### Account Management — Accounts *(14 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| POST | `/gw/api/v1/accounts` | Create Account | `createAccounts` |
| PATCH | `/gw/api/v1/accounts` | Update Account | `updateAccounts` |
| GET | `/gw/api/v1/accounts` | Retrieve Processed Application | `listAccounts` |
| POST | `/gw/api/v1/accounts/documents` | Submit General Agreements And Disclosures | `createAccountsDocuments` |
| GET | `/gw/api/v1/accounts/login-messages` | Login Messages | `listAccountsLoginMessages` |
| GET | `/gw/api/v1/accounts/status` | Account Status List | `listAccountsStatus` |
| GET | `/gw/api/v1/accounts/{accountId}/details` | Account Details | `getAccountsDetails` |
| GET | `/gw/api/v1/accounts/{accountId}/kyc` | KYC Information | `getAccountsKyc` |
| GET | `/gw/api/v1/accounts/{accountId}/login-messages` | Login Messages For Account | `getAccountsLoginMessages` |
| GET | `/gw/api/v1/accounts/{accountId}/status` | Account Status | `getAccountsStatus` |
| PATCH | `/gw/api/v1/accounts/{accountId}/status` | Update Account Status | `updateAccountsStatus` |
| POST | `/gw/api/v1/accounts/{accountId}/tasks` | Assign Tasks | `createAccountsTasks` |
| GET | `/gw/api/v1/accounts/{accountId}/tasks` | Get Tasks | `getAccountsTasks` |
| PATCH | `/gw/api/v1/accounts/{accountId}/tasks` | Update Tasks Status | `updateAccountsTasks` |

### Account Management — Banking *(20 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| POST | `/gw/api/v1/bank-instructions` | Manage Bank Instructions | `createBankInstructions` |
| POST | `/gw/api/v1/bank-instructions/query` | View Bank Instructions | `createBankInstructionsQuery` |
| POST | `/gw/api/v1/bank-instructions:bulk` | Creates Multiple Banking Instructions | `bulkBankInstructions` |
| POST | `/gw/api/v1/external-asset-transfers` | Transfer Positions Externally (ACATS, ATON, FOP) | `createExternalAssetTransfers` |
| POST | `/gw/api/v1/external-asset-transfers:bulk` | Creates Multiple External Asset Transfers | `bulkExternalAssetTransfers` |
| POST | `/gw/api/v1/external-cash-transfers` | Transfer Cash Externally | `createExternalCashTransfers` |
| POST | `/gw/api/v1/external-cash-transfers/query` | Query External Cash Transfers | `createExternalCashTransfersQuery` |
| POST | `/gw/api/v1/external-cash-transfers:bulk` | Creates Multiple External Cash Transfers | `bulkExternalCashTransfers` |
| POST | `/gw/api/v1/internal-asset-transfers` | Transfer Positions Internally | `createInternalAssetTransfers` |
| POST | `/gw/api/v1/internal-asset-transfers:bulk` | Creates Multiple Internal Asset Transfers | `bulkInternalAssetTransfers` |
| POST | `/gw/api/v1/internal-cash-transfers` | Transfer Cash Internally | `createInternalCashTransfers` |
| POST | `/gw/api/v1/internal-cash-transfers:bulk` | Creates Multiple Internal Cash Transfers | `bulkInternalCashTransfers` |
| GET | `/gw/api/v1/instructions/{instructionId}` | Get Instruction | `getInstructions` |
| GET | `/gw/api/v1/instruction-sets/{instructionSetId}` | Get Instruction Set | `getInstructionSets` |
| GET | `/gw/api/v1/client-instructions/{clientInstructionId}` | Get Client Instruction | `getClientInstructions` |
| POST | `/gw/api/v1/instructions/cancel` | Cancel Instruction | `createInstructionsCancel` |
| POST | `/gw/api/v1/instructions/cancel:bulk` | Cancel Multiple Instructions | `bulkInstructionsCancel` |
| POST | `/gw/api/v1/instructions/query` | Query Instructions | `createInstructionsQuery` |
| GET | `/gw/api/v1/participating-banks` | List Participating Banks | `listParticipatingBanks` |

### Account Management — Reports *(6 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| POST | `/gw/api/v1/statements` | Generates Statements | `createStatements` |
| GET | `/gw/api/v1/statements/available` | Available Statements | `listStatementsAvailable` |
| POST | `/gw/api/v1/trade-confirmations` | Fetch Trade Confirmations | `createTradeConfirmations` |
| GET | `/gw/api/v1/trade-confirmations/available` | Available Trade Confirmations | `listTradeConfirmationsAvailable` |
| POST | `/gw/api/v1/tax-documents` | Fetch Tax Forms | `createTaxDocuments` |
| GET | `/gw/api/v1/tax-documents/available` | Available Tax Documents | `listTaxDocumentsAvailable` |

### Account Management — Utilities *(9 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| GET | `/gw/api/v1/enumerations/{enumerationType}` | Get Enumerations | `getEnumerations` |
| GET | `/gw/api/v1/enumerations/complex-asset-transfer` | Get Participating Brokers | `listComplexAssetTransfer` |
| GET | `/gw/api/v1/forms` | Get Forms | `listForms` |
| GET | `/gw/api/v1/forms/required-forms` | Get Required Forms | `listFormsRequiredForms` |
| GET | `/gw/api/v1/requests` | List Requests | `listRequests` |
| GET | `/gw/api/v1/requests/{requestId}/status` | Get Request Status | `getRequestsStatus` |
| PATCH | `/gw/api/v1/requests/{requestId}/status` | Update Request Status | `updateRequestsStatus` |
| GET | `/gw/api/v1/validations/usernames/{username}` | Validate Username | `getValidationsUsernames` |
| GET | `/gw/api/v1/echo/https` | Echo HTTPS | `listEchoHttps` |

### Authorization *(4 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| POST | `/oauth2/api/v1/token` | Create Access Token | `generateToken` |
| POST | `/gw/api/v1/sso-browser-sessions` | Create SSO Browser Session | `createSsoBrowserSessions` |
| POST | `/gw/api/v1/sso-sessions` | Create SSO Session | `createSsoSessions` |
| GET | `/gw/api/v1/echo/signed-jwt` | Echo Signed JWT | `createEchoSignedJwt` |

### PreTrade Compliance *(9 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| POST | `/gw/api/v1/restrictions` | Apply PTC CSV | `applyCSV` |
| POST | `/gw/api/v1/restrictions/verify` | Verify PTC CSV | `verifyCSV` |
| GET | `/gw/api/v1/restrictions/account` | Get Account Restrictions | `getAccountRestrictions` |
| GET | `/gw/api/v1/restrictions/ids` | Get Restriction IDs | `getMasterRestrictionIds` |
| GET | `/gw/api/v1/restrictions/list` | Get List Details | `getListDetails` |
| GET | `/gw/api/v1/restrictions/lists/ids` | Get List IDs | `getMasterListIds` |
| GET | `/gw/api/v1/restrictions/restriction` | Get Restriction Details | `getRestrictionDetails` |
| GET | `/gw/api/v1/restrictions/restriction-scope` | Get Restriction Scope | `getRestrictionScope` |
| GET | `/gw/api/v1/restrictions/user` | Get User Restrictions | `getUserRestrictions` |

### Third Parties *(7 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| POST | `/gw/api/v1/balances/query` | View Cash Balances | `createBalancesQuery` |
| GET | `/gw/api/v1/tax-vouchers/active-countries` | Get Active Countries | `getActiveCountryList` |
| GET | `/gw/api/v1/tax-vouchers/dividends` | Fetch Dividend Details | `fetchDividends_1` |
| GET | `/gw/api/v1/tax-vouchers/years` | Get Available Years | `getYears` |
| GET | `/gw/api/v1/tax-vouchers/{requestId}/download` | Download Tax Voucher | `downloadFile` |
| GET | `/gw/api/v1/tax-vouchers/{requestId}/state` | Get Request State | `getCurrentState_1` |
| POST | `/gw/api/v1/tax-vouchers` | Create Tax Voucher Request | `createTaxVoucherRequests` |

### Trading — Accounts *(11 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| GET | `/v1/api/iserver/accounts` | List All Accounts | `getBrokerageAccounts` |
| GET | `/v1/api/iserver/account/{accountId}/summary` | Account Summary | `getAccountSummary` |
| GET | `/v1/api/iserver/account/{accountId}/summary/available_funds` | Available Funds | `getFundSummary` |
| GET | `/v1/api/iserver/account/{accountId}/summary/balances` | Balance Summary | `getBalanceSummary` |
| GET | `/v1/api/iserver/account/{accountId}/summary/margins` | Margin Summary | `getMarginSummary` |
| GET | `/v1/api/iserver/account/{accountId}/summary/market_value` | Market Value Summary | `getAccountMarketSummary` |
| GET | `/v1/api/iserver/account/pnl/partitioned` | Account P&L | `getPnl` |
| POST | `/v1/api/iserver/account` | Switch Active Account | `setActiveAccount` |
| POST | `/v1/api/iserver/dynaccount` | Set Dynamic Account | `setDynamicAccount` |
| GET | `/v1/api/iserver/account/search/{searchPattern}` | Search Accounts | `getDynamicAccounts` |
| GET | `/v1/api/acesws/{accountId}/signatures-and-owners` | Account Signatures | `getAccountOwners` |

### Trading — Orders *(11 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| POST | `/v1/api/iserver/account/{accountId}/orders` | Submit New Order | `submitNewOrder` |
| POST | `/v1/api/iserver/account/{accountId}/orders/whatif` | Preview Margin Impact | `previewMarginImpact` |
| POST | `/v1/api/iserver/account/{accountId}/order/{orderId}` | Modify Open Order | `modifyOpenOrder` |
| DELETE | `/v1/api/iserver/account/{accountId}/order/{orderId}` | Cancel Open Order | `cancelOpenOrder` |
| GET | `/v1/api/iserver/account/orders` | Get Open Orders | `getOpenOrders` |
| GET | `/v1/api/iserver/account/order/status/{orderId}` | Order Status | `getOrderStatus` |
| GET | `/v1/api/iserver/account/trades` | Trade History | `getTradeHistory` |
| POST | `/v1/api/iserver/reply/{replyId}` | Confirm Order Reply | `confirmOrderReply` |
| POST | `/v1/api/iserver/notification` | Dismiss Server Prompt | `ackServerPrompt` |
| POST | `/v1/api/iserver/questions/suppress` | Suppress Order Replies | `suppressOrderReplies` |
| POST | `/v1/api/iserver/questions/suppress/reset` | Reset Order Suppression | `resetOrderSuppression` |

### Trading — Contracts *(17 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| POST | `/v1/api/iserver/secdef/search` | Search Instruments (body) | `getContractSymbolsFromBody` |
| GET | `/v1/api/iserver/secdef/search` | Search Instruments | `getContractSymbols` |
| GET | `/v1/api/iserver/contract/{conid}/info` | Instrument Info | `getInstrumentInfo` |
| GET | `/v1/api/iserver/contract/{conid}/info-and-rules` | Info + Rules | `getInfoAndRules` |
| GET | `/v1/api/iserver/contract/{conid}/algos` | Algos For Instrument | `getAlgosByInstrument` |
| POST | `/v1/api/iserver/contract/rules` | Search Contract Rules | `getContractRules` |
| GET | `/v1/api/iserver/secdef/info` | Contract Info | `getContractInfo` |
| GET | `/v1/api/iserver/secdef/strikes` | Contract Strikes | `getContractStrikes` |
| GET | `/v1/api/iserver/secdef/bond-filters` | Bond Filters | `getBondFilters` |
| GET | `/v1/api/trsrv/stocks` | Get Stock By Symbol | `getStockBySymbol` |
| GET | `/v1/api/trsrv/futures` | Get Future By Symbol | `getFutureBySymbol` |
| GET | `/v1/api/trsrv/secdef` | Instrument Definition | `getInstrumentDefinition` |
| GET | `/v1/api/trsrv/secdef/schedule` | Trading Schedule | `getTradingSchedule` |
| GET | `/v1/api/trsrv/all-conids` | Get Conids By Exchange | `getConidsByExchange` |
| GET | `/v1/api/contract/trading-schedule` | Trading Schedule | `getTradingSchedule` |
| GET | `/v1/api/iserver/currency/pairs` | Currency Pairs | `getCurrencyPairs` |
| GET | `/v1/api/iserver/exchangerate` | Exchange Rates | `getExchangeRates` |

### Trading — Portfolio *(13 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| GET | `/v1/api/portfolio/accounts` | List All Accounts | `getAllAccounts` |
| GET | `/v1/api/portfolio/subaccounts` | List Subaccounts | `getAllSubaccounts` |
| GET | `/v1/api/portfolio/subaccounts2` | Large Account Subaccounts | `getManySubaccounts` |
| GET | `/v1/api/portfolio/positions/{conid}` | Positions By Conid | `getAllAccountsForConid` |
| GET | `/v1/api/portfolio/{accountId}/positions/{pageId}` | Paginated Positions | `getPaginatedPositions` |
| GET | `/v1/api/portfolio/{accountId}/position/{conid}` | Single Position | `getPositionByConid` |
| GET | `/v1/api/portfolio/{accountId}/summary` | Portfolio Summary | `getPortfolioSummary` |
| GET | `/v1/api/portfolio/{accountId}/allocation` | Asset Allocation | `getAssetAllocation` |
| GET | `/v1/api/portfolio/{accountId}/ledger` | Portfolio Ledger | `getPortfolioLedger` |
| GET | `/v1/api/portfolio/{accountId}/meta` | Portfolio Metadata | `getPortfolioMetadata` |
| GET | `/v1/api/portfolio/{accountId}/combo/positions` | Combo Positions | `getComboPositions` |
| GET | `/v1/api/portfolio2/{accountId}/positions` | Uncached Positions | `getUncachedPositions` |
| POST | `/v1/api/portfolio/{accountId}/positions/invalidate` | Refresh Position Cache | `invalidatePositionCache` |

### Trading — Market Data *(4 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| GET | `/v1/api/iserver/marketdata/snapshot` | Live Market Data Snapshot | `getMdSnapshot` |
| GET | `/v1/api/iserver/marketdata/history` | Historical OHLC Bar Data | `getMdHistory` |
| POST | `/v1/api/iserver/marketdata/unsubscribe` | Unsubscribe | `closeMdStream` |
| GET | `/v1/api/iserver/marketdata/unsubscribeall` | Unsubscribe All | `closeAllMdStreams` |

### Trading — WebSocket *(1 endpoint)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| GET | `/v1/api/ws` | Open WebSocket | `openWebsocket` |

### Trading — Session *(5 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| POST | `/v1/api/tickle` | Brokerage Keep-Alive Ping | `getSessionToken` |
| POST | `/v1/api/logout` | Terminate Session | `logout` |
| POST | `/v1/api/iserver/auth/ssodh/init` | Initialize Brokerage Session | `initializeSession` |
| POST | `/v1/api/iserver/auth/status` | Session Status | `getBrokerageStatus` |
| GET | `/v1/api/sso/validate` | Validate SSO Session | `getSessionValidation` |

### Trading — OAuth *(3 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| POST | `/v1/api/oauth/request_token` | Obtain Request Token | `reqTempToken` |
| POST | `/v1/api/oauth/access_token` | Generate Access Token | `reqAccessToken` |
| POST | `/v1/api/oauth/live_session_token` | Generate Live Session Token | `reqLiveSessionToken` |

### Trading — Alerts *(6 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| POST | `/v1/api/iserver/account/{accountId}/alert` | Create/Modify Alert | `createAlert` |
| POST | `/v1/api/iserver/account/{accountId}/alert/activate` | Activate/Deactivate Alert | `activateAlert` |
| GET | `/v1/api/iserver/account/{accountId}/alerts` | List All Alerts | `getAllAlerts` |
| GET | `/v1/api/iserver/account/alert/{alertId}` | Alert Details | `getAlertDetails` |
| GET | `/v1/api/iserver/account/mta` | Mobile Trading Alert | `getMtaDetails` |
| DELETE | `/v1/api/iserver/account/{accountId}/alert/{alertId}` | Delete Alert | `deleteAlert` |

### Trading — Watchlists *(4 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| GET | `/v1/api/iserver/watchlists` | All Watchlists | `getAllWatchlists` |
| GET | `/v1/api/iserver/watchlist` | Single Watchlist | `getSpecificWatchlist` |
| POST | `/v1/api/iserver/watchlist` | Create Watchlist | `postNewWatchlist` |
| DELETE | `/v1/api/iserver/watchlist` | Delete Watchlist | `deleteWatchlist` |

### Trading — FA Allocation *(8 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| GET | `/v1/api/iserver/account/allocation/group` | List Allocation Groups | `getAllocationGroups` |
| POST | `/v1/api/iserver/account/allocation/group` | Add Allocation Group | `createAllocationGroup` |
| PUT | `/v1/api/iserver/account/allocation/group` | Modify Allocation Group | `modifyAllocationGroup` |
| POST | `/v1/api/iserver/account/allocation/group/delete` | Delete Allocation Group | `deleteAllocationGroup` |
| POST | `/v1/api/iserver/account/allocation/group/single` | Get Single Group | `getSingleAllocationGroup` |
| POST | `/v1/api/iserver/account/allocation/presets` | Set Allocation Preset | `setAllocationPreset` |
| GET | `/v1/api/iserver/account/allocation/presets` | Get Allocation Presets | `getAllocatableSubaccounts` |
| GET | `/v1/api/iserver/account/allocation/accounts` | Allocatable Subaccounts | `getAllocatableSubaccounts` |

### Trading — FA Model Portfolios *(10 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| POST | `/v1/api/fa/model/list` | Get All Models | `getAllmodels` |
| POST | `/v1/api/fa/model/summary` | Get Model Summary | `getModelSummarySingle` |
| POST | `/v1/api/fa/model/accounts-details` | Get Model Accounts | `getAccountsInModel` |
| POST | `/v1/api/fa/model/positions` | Get All Model Positions | `getAllModelPositions` |
| POST | `/v1/api/fa/model/save` | Set Model Target Positions | `setModelTargetPositions` |
| POST | `/v1/api/fa/model/invest-divest` | Invest/Divest Account | `setAccountinvestmentInModel` |
| POST | `/v1/api/fa/model/invest-divest-positions` | Get Invested Accounts | `getInvestedAccountsInModel` |
| POST | `/v1/api/fa/model/submit-transfers` | Submit Model Orders | `submitModelOrders` |
| POST | `/v1/api/fa/fa-preset/get` | Get Model Preset | `getModelPresets` |
| POST | `/v1/api/fa/fa-preset/save` | Set Model Preset | `setModelPresets` |

### Trading — Portfolio Analyst *(4 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| POST | `/v1/api/pa/performance` | Account Performance | `getSinglePerformancePeriod` |
| POST | `/v1/api/pa/allperiods` | Performance All Periods | `getPerformanceAllPeriods` |
| POST | `/v1/api/pa/allocation` | Portfolio Allocation | `createAllocation` |
| POST | `/v1/api/pa/transactions` | Transaction History | `getTransactions` |

### Trading — Scanner *(2 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| POST | `/v1/api/iserver/scanner/run` | Run Market Scanner | `getScannerResults` |
| GET | `/v1/api/iserver/scanner/params` | Scanner Parameters | `getScannerParameters` |

### Trading — FYIs *(11 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| GET | `/v1/api/fyi/notifications` | All Notifications | `getAllFyis` |
| GET | `/v1/api/fyi/unreadnumber` | Unread Count | `getUnreadFyis` |
| GET | `/v1/api/fyi/deliveryoptions` | Get Delivery Options | `getFyiDelivery` |
| POST | `/v1/api/fyi/deliveryoptions/device` | Toggle Device Delivery | `modifyFyiDelivery` |
| GET | `/v1/api/fyi/settings` | Get FYI Settings | `getFyiSettings` |
| POST | `/v1/api/fyi/settings/{typecode}` | Modify FYI Settings | `modifyFyiNotification` |
| GET | `/v1/api/fyi/disclaimer/{typecode}` | Get FYI Disclaimer | `getFyiDisclaimerss` |
| PUT | `/v1/api/fyi/disclaimer/{typecode}` | Mark Disclaimer Read | `readFyiDisclaimer` |
| PUT | `/v1/api/fyi/notifications/{notificationId}` | Mark Notification Read | `readFyiNotification` |
| PUT | `/v1/api/fyi/deliveryoptions/email` | Modify Email Delivery | `modifyFyiEmails` |
| DELETE | `/v1/api/fyi/deliveryoptions/{deviceId}` | Delete Device | `deleteFyiDevice` |

### Trading — Event Contracts *(5 endpoints)*

| Method | Path | Summary | Operation ID |
|--------|------|---------|--------------|
| GET | `/v1/api/forecast/category/tree` | Event Categories | `getForecastCategories` |
| GET | `/v1/api/forecast/contract/market` | Event Markets | `getForecastMarkets` |
| GET | `/v1/api/forecast/contract/details` | Event Contract Details | `getForecastContract` |
| GET | `/v1/api/forecast/contract/rules` | Event Contract Rules | `getForecastRules` |
| GET | `/v1/api/forecast/contract/schedules` | Event Schedules | `getForecastSchedule` |

---

## Codegen Status

| Status | Description |
|--------|-------------|
| ✅ | 184/185 endpoints generate cleanly |
| ⚠️ | `/gw/api/v1/balances/query` — path param mismatch in spec (POST with 0 path params, but declared as having 1). Handled manually. |

---

*Last updated: 2026-09-16*
