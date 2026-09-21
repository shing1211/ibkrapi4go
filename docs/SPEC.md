# SPEC.md — IBKR OpenAPI Specification Reference

> Source: `https://api.ibkr.com/gw/api/v3/api-docs`
> Title: IB REST API | Version: 2.39.0 | OpenAPI: 3.0.0
> Endpoints: 185 | Schemas: 443 | Tags: 27 | Generated: 2026-09-21

> **Canonical counts.** All endpoint/schema numbers in this repository
> derive from this file. Regenerate with `scripts/gen_spec_index.py` — do
> not edit by hand.

---

## Servers

| Environment | URL |
|-------------|-----|
| Client Portal Gateway | `https://localhost:5000` |
| Production | `https://api.ibkr.com` |
| Sandbox | `https://qa.interactivebrokers.com` |

## Surfaces and auth

| Surface | Base path | Operations | Auth |
|---------|-----------|-----------:|------|
| Client Portal API (CPAPI) | `/v1/api/*` | 115 | `ssoBearer` |
| IB REST API | `/gw/api/v1/*`, `/gw/api/v2/*`, `/oauth2/*` | 70 | `oauth2Bearer` |
| **Total** | | **185** | |

Both surfaces are implemented: 115 CPAPI (`ssoBearer`) + 70 IB REST (`oauth2Bearer`). See [ADR 0001](./adr/0001-two-api-surfaces.md) and [ADR 0011](./adr/0011-oauth2-surface.md).

---

## Endpoint index (185 operations)

### Account Management Accounts *(14)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| GET | `/gw/api/v1/accounts` | IB REST | oauth2Bearer | Retrieve Processed Application | `listAccounts` |
| PATCH | `/gw/api/v1/accounts` | IB REST | oauth2Bearer | Update Account | `updateAccounts` |
| POST | `/gw/api/v1/accounts` | IB REST | oauth2Bearer | Create Account | `createAccounts` |
| POST | `/gw/api/v1/accounts/documents` | IB REST | oauth2Bearer | Submit General Agreements And Disclosures | `createAccountsDocuments` |
| GET | `/gw/api/v1/accounts/login-messages` | IB REST | oauth2Bearer | Get Login Messages | `listAccountsLoginMessages` |
| GET | `/gw/api/v1/accounts/status` | IB REST | oauth2Bearer | Get Account Status: Bulk Request | `listAccountsStatus` |
| GET | `/gw/api/v1/accounts/{accountId}/details` | IB REST | oauth2Bearer | Get Account Information | `getAccountsDetails` |
| GET | `/gw/api/v1/accounts/{accountId}/kyc` | IB REST | oauth2Bearer | Retrieve Au10Tix URL | `getAccountsKyc` |
| GET | `/gw/api/v1/accounts/{accountId}/login-messages` | IB REST | oauth2Bearer | Get Login Message By Account | `getAccountsLoginMessages` |
| GET | `/gw/api/v1/accounts/{accountId}/status` | IB REST | oauth2Bearer | Get Status By Account | `getAccountsStatus` |
| PATCH | `/gw/api/v1/accounts/{accountId}/status` | IB REST | oauth2Bearer | Update Status By Account | `updateAccountsStatus` |
| GET | `/gw/api/v1/accounts/{accountId}/tasks` | IB REST | oauth2Bearer | Get Registration Tasks | `getAccountsTasks` |
| PATCH | `/gw/api/v1/accounts/{accountId}/tasks` | IB REST | oauth2Bearer | Update Tasks Status | `updateAccountsTasks` |
| POST | `/gw/api/v1/accounts/{accountId}/tasks` | IB REST | oauth2Bearer | Assign Tasks | `createAccountsTasks` |

### Account Management Banking *(20)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| POST | `/gw/api/v1/bank-instructions` | IB REST | oauth2Bearer | Manage Bank Instructions | `createBankInstructions` |
| POST | `/gw/api/v1/bank-instructions/query` | IB REST | oauth2Bearer | View Bank Instructions | `createBankInstructionsQuery` |
| POST | `/gw/api/v1/bank-instructions:bulk` | IB REST | oauth2Bearer | Creates Multiple Banking Instructions(ach, Open-banking, Withdrawal, Delete, Traditional-bank-instruction-verification, Predefined-destination-instruction, Edda) | `bulkBankInstructions` |
| GET | `/gw/api/v1/client-instructions/{clientInstructionId}` | IB REST | oauth2Bearer | Get Status For ClientInstructionId | `getClientInstructions` |
| POST | `/gw/api/v1/external-asset-transfers` | IB REST | oauth2Bearer | Transfer Positions Externally (ACATS, ATON, FOP, DWAC, Complex Asset Transfer) | `createExternalAssetTransfers` |
| POST | `/gw/api/v1/external-asset-transfers:bulk` | IB REST | oauth2Bearer | Creates Multiple External Asset Transfers (Fop, DWAC And Complex Asset Transfer) | `bulkExternalAssetTransfers` |
| POST | `/gw/api/v1/external-cash-transfers` | IB REST | oauth2Bearer | Transfer Cash Externally | `createExternalCashTransfers` |
| POST | `/gw/api/v1/external-cash-transfers/query` | IB REST | oauth2Bearer | View Cash Balances | `createExternalCashTransfersQuery` |
| POST | `/gw/api/v1/external-cash-transfers:bulk` | IB REST | oauth2Bearer | Creates Multiple External Cash Transfers (Deposit And Withdraw Fund) | `bulkExternalCashTransfers` |
| GET | `/gw/api/v1/instruction-sets/{instructionSetId}` | IB REST | oauth2Bearer | Get Status For InstructionSetId | `getInstructionSets` |
| POST | `/gw/api/v1/instructions/cancel` | IB REST | oauth2Bearer | Cancel Request | `createInstructionsCancel` |
| POST | `/gw/api/v1/instructions/cancel:bulk` | IB REST | oauth2Bearer | Creates Multiple Cancel Instructions | `bulkInstructionsCancel` |
| POST | `/gw/api/v1/instructions/query` | IB REST | oauth2Bearer | Get Transaction History | `createInstructionsQuery` |
| GET | `/gw/api/v1/instructions/{instructionId}` | IB REST | oauth2Bearer | Get Status For InstructionId | `getInstructions` |
| POST | `/gw/api/v1/internal-asset-transfers` | IB REST | oauth2Bearer | Transfer Positions Internally | `createInternalAssetTransfers` |
| POST | `/gw/api/v1/internal-asset-transfers:bulk` | IB REST | oauth2Bearer | Creates Multiple Internal Asset Transfers Between The Provided Account Id Pairs | `bulkInternalAssetTransfers` |
| POST | `/gw/api/v1/internal-cash-transfers` | IB REST | oauth2Bearer | Transfer Cash Internally | `createInternalCashTransfers` |
| POST | `/gw/api/v1/internal-cash-transfers:bulk` | IB REST | oauth2Bearer | Creates Multiple Internal Cash Transfers Between The Provided Account Id Pairs | `bulkInternalCashTransfers` |
| POST | `/gw/api/v2/external-asset-transfers` | IB REST | oauth2Bearer | Transfer Positions Externally (FOP, DWAC, Complex Asset Transfer) | `createExternalAssetTransfers_2` |
| POST | `/gw/api/v2/external-asset-transfers:bulk` | IB REST | oauth2Bearer | Creates Multiple External Asset Transfers (Fop, DWAC And Complex Asset Transfer) | `bulkExternalAssetTransfers_2` |

### Account Management Reports *(6)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| POST | `/gw/api/v1/statements` | IB REST | oauth2Bearer | Generates Statements In Supported Formats Based On Request Parameters. | `createStatements` |
| GET | `/gw/api/v1/statements/available` | IB REST | oauth2Bearer | Fetch Available Daily, Monthly, And Annual Report Dates For An Account Id | `listStatementsAvailable` |
| POST | `/gw/api/v1/tax-documents` | IB REST | oauth2Bearer | Fetch Tax Forms In Supported Formats Based On Request Parameters. | `createTaxDocuments` |
| GET | `/gw/api/v1/tax-documents/available` | IB REST | oauth2Bearer | Fetch List Of Available Tax Reports/forms/documents For A Specified Account And Tax Year | `listTaxDocumentsAvailable` |
| POST | `/gw/api/v1/trade-confirmations` | IB REST | oauth2Bearer | Fetch Trade Confirmations In Supported Formats Based On Request Parameters. | `createTradeConfirmations` |
| GET | `/gw/api/v1/trade-confirmations/available` | IB REST | oauth2Bearer | Fetch List Of Available Trade Confirmation Dates, For A Specific Account Id | `listTradeConfirmationsAvailable` |

### Account Management Utilities *(9)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| GET | `/gw/api/v1/enumerations/complex-asset-transfer` | IB REST | oauth2Bearer | Get A List Of Participating Brokers For The Given Asset Type | `listEnumerationsComplexAssetTransfer` |
| GET | `/gw/api/v1/enumerations/{enumerationType}` | IB REST | oauth2Bearer | Get Enumerations | `getEnumerations` |
| GET | `/gw/api/v1/forms` | IB REST | oauth2Bearer | Get Forms | `listForms` |
| GET | `/gw/api/v1/forms/required-forms` | IB REST | oauth2Bearer | Get Required Forms | `listFormsRequiredForms` |
| GET | `/gw/api/v1/participating-banks` | IB REST | oauth2Bearer | Get Participating Banks | `listParticipatingBanks` |
| GET | `/gw/api/v1/requests` | IB REST | oauth2Bearer | Get Requests' Details By Timeframe | `listRequests` |
| GET | `/gw/api/v1/requests/{requestId}/status` | IB REST | oauth2Bearer | Get Status Of A Request | `getRequestsStatus` |
| PATCH | `/gw/api/v1/requests/{requestId}/status` | IB REST | oauth2Bearer | Update Status Of An Am Request | `updateRequestsStatus` |
| GET | `/gw/api/v1/validations/usernames/{username}` | IB REST | oauth2Bearer | Verify User Availability | `getValidationsUsernames` |

### Authorization SSO Browser Sessions *(1)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| POST | `/gw/api/v1/sso-browser-sessions` | IB REST | oauth2Bearer | Create SSO Browser Session. | `createSsoBrowserSessions` |

### Authorization SSO Sessions *(1)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| POST | `/gw/api/v1/sso-sessions` | IB REST | oauth2Bearer | Create A New SSO Session On Behalf Of An End-user. | `createSsoSessions` |

### Authorization Token *(1)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| POST | `/oauth2/api/v1/token` | IB REST | ssoBearer | Create Access Token | `generateToken` |

### PreTrade Compliance Restrictions *(9)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| POST | `/gw/api/v1/restrictions` | IB REST | oauth2Bearer | Apply PTC CSV | `applyCSV` |
| GET | `/gw/api/v1/restrictions/account` | IB REST | oauth2Bearer | Get Restriction IDs For An Account | `getAccountRestrictions` |
| GET | `/gw/api/v1/restrictions/ids` | IB REST | oauth2Bearer | Get Restriction IDs Created By The Caller's Master Account | `getMasterRestrictionIds` |
| GET | `/gw/api/v1/restrictions/list` | IB REST | oauth2Bearer | Get List Details | `getListDetails` |
| GET | `/gw/api/v1/restrictions/lists/ids` | IB REST | oauth2Bearer | Get List IDs Created By The Caller's Master Account | `getMasterListIds` |
| GET | `/gw/api/v1/restrictions/restriction` | IB REST | oauth2Bearer | Get Restriction Details | `getRestrictionDetails` |
| GET | `/gw/api/v1/restrictions/restriction-scope` | IB REST | oauth2Bearer | Get Restriction Scope | `getRestrictionScope` |
| GET | `/gw/api/v1/restrictions/user` | IB REST | oauth2Bearer | Get Restriction IDs For A User | `getUserRestrictions` |
| POST | `/gw/api/v1/restrictions/verify` | IB REST | oauth2Bearer | Verify PTC CSV | `verifyCSV` |

### Third Parties Cash Balances *(1)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| POST | `/gw/api/v1/balances/query` | IB REST | oauth2Bearer | View Cash Balances | `createBalancesQuery` |

### Third Parties Tax Vouchers *(6)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| POST | `/gw/api/v1/tax-vouchers` | IB REST | oauth2Bearer | Create Tax Voucher Requests | `createTaxVoucherRequests` |
| GET | `/gw/api/v1/tax-vouchers/active-countries` | IB REST | oauth2Bearer | Get Active Country List | `getActiveCountryList` |
| GET | `/gw/api/v1/tax-vouchers/dividends` | IB REST | oauth2Bearer | Fetch Dividend Details | `fetchDividends_1` |
| GET | `/gw/api/v1/tax-vouchers/years` | IB REST | oauth2Bearer | Get Available Years | `getYears` |
| GET | `/gw/api/v1/tax-vouchers/{requestId}/download` | IB REST | oauth2Bearer | Download Tax Voucher File | `downloadFile` |
| GET | `/gw/api/v1/tax-vouchers/{requestId}/state` | IB REST | oauth2Bearer | Get Tax Voucher Request State | `getCurrentState_1` |

### Trading Accounts *(11)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| GET | `/v1/api/acesws/{accountId}/signatures-and-owners` | CPAPI | ssoBearer | List Account Signatures And Owners | `getAccountOwners` |
| POST | `/v1/api/iserver/account` | CPAPI | ssoBearer | Switch Selected Account | `setActiveAccount` |
| GET | `/v1/api/iserver/account/pnl/partitioned` | CPAPI | ssoBearer | Account Profit And Loss | `getPnl` |
| GET | `/v1/api/iserver/account/search/{searchPattern}` | CPAPI | ssoBearer | Search Dynamic Accounts | `getDynamicAccounts` |
| GET | `/v1/api/iserver/account/{accountId}/summary` | CPAPI | ssoBearer | Summary Of Account Values | `getAccountSummary` |
| GET | `/v1/api/iserver/account/{accountId}/summary/available_funds` | CPAPI | ssoBearer | Summary Of Available Funds | `getFundSummary` |
| GET | `/v1/api/iserver/account/{accountId}/summary/balances` | CPAPI | ssoBearer | Summary Of Account Balances | `getBalanceSummary` |
| GET | `/v1/api/iserver/account/{accountId}/summary/margins` | CPAPI | ssoBearer | Summary Of Account Margin Usage | `getMarginSummary` |
| GET | `/v1/api/iserver/account/{accountId}/summary/market_value` | CPAPI | ssoBearer | Summary Of Account Market Value | `getAccountMarketSummary` |
| GET | `/v1/api/iserver/accounts` | CPAPI | ssoBearer | List All Tradable Accounts | `getBrokerageAccounts` |
| POST | `/v1/api/iserver/dynaccount` | CPAPI | ssoBearer | Set Active Dynamic Account | `setDynamicAccount` |

### Trading Alerts *(6)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| GET | `/v1/api/iserver/account/alert/{alertId}` | CPAPI | ssoBearer | Details Of A Specific Alert | `getAlertDetails` |
| GET | `/v1/api/iserver/account/mta` | CPAPI | ssoBearer | Details Of A Mobile Trading Alert | `getMtaDetails` |
| POST | `/v1/api/iserver/account/{accountId}/alert` | CPAPI | ssoBearer | Create Or Modify Alert | `createAlert` |
| POST | `/v1/api/iserver/account/{accountId}/alert/activate` | CPAPI | ssoBearer | Activate Or Deactivate An Alert | `activateAlert` |
| DELETE | `/v1/api/iserver/account/{accountId}/alert/{alertId}` | CPAPI | ssoBearer | Delete An Alert | `deleteAlert` |
| GET | `/v1/api/iserver/account/{accountId}/alerts` | CPAPI | ssoBearer | List All Alerts | `getAllAlerts` |

### Trading Contracts *(17)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| GET | `/v1/api/contract/trading-schedule` | CPAPI | ssoBearer | Trading Schedule (NEW) | `getTradingSchedule` |
| POST | `/v1/api/iserver/contract/rules` | CPAPI | ssoBearer | Search Contract Rules | `getContractRules` |
| GET | `/v1/api/iserver/contract/{conid}/algos` | CPAPI | ssoBearer | Search Algos For An Instrument | `getAlgosByInstrument` |
| GET | `/v1/api/iserver/contract/{conid}/info` | CPAPI | ssoBearer | General Instrument Information | `getInstrumentInfo` |
| GET | `/v1/api/iserver/contract/{conid}/info-and-rules` | CPAPI | ssoBearer | Instrument Info And Market Rules | `getInfoAndRules` |
| GET | `/v1/api/iserver/currency/pairs` | CPAPI | ssoBearer | Available Currency Pairs | `getCurrencyPairs` |
| GET | `/v1/api/iserver/exchangerate` | CPAPI | ssoBearer | Currency Exchange Rate | `getExchangeRates` |
| GET | `/v1/api/iserver/secdef/bond-filters` | CPAPI | ssoBearer | Search Bond Filter Information | `getBondFilters` |
| GET | `/v1/api/iserver/secdef/info` | CPAPI | ssoBearer | Instrument Attributes Detail | `getContractInfo` |
| GET | `/v1/api/iserver/secdef/search` | CPAPI | ssoBearer | Search Instruments By Symbol | `getContractSymbols` |
| POST | `/v1/api/iserver/secdef/search` | CPAPI | ssoBearer | Search Instruments By Symbol | `getContractSymbolsFromBody` |
| GET | `/v1/api/iserver/secdef/strikes` | CPAPI | ssoBearer | Search Strikes For An Underlier | `getContractStrikes` |
| GET | `/v1/api/trsrv/all-conids` | CPAPI | ssoBearer | List All Stock Conids By Exchange | `getConidsByExchange` |
| GET | `/v1/api/trsrv/futures` | CPAPI | ssoBearer | Search Futures By Symbol | `getFutureBySymbol` |
| GET | `/v1/api/trsrv/secdef` | CPAPI | ssoBearer | Instrument Definition Detail | `getInstrumentDefinition` |
| GET | `/v1/api/trsrv/secdef/schedule` | CPAPI | ssoBearer | Trading Schedule By Symbol | `getTradingSchedule` |
| GET | `/v1/api/trsrv/stocks` | CPAPI | ssoBearer | Search Stocks By Symbol | `getStockBySymbol` |

### Trading Event Contracts *(5)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| GET | `/v1/api/forecast/category/tree` | CPAPI | ssoBearer | Event Contract Categories | `getForecastCategories` |
| GET | `/v1/api/forecast/contract/details` | CPAPI | ssoBearer | Event Contract Details | `getForecastContract` |
| GET | `/v1/api/forecast/contract/market` | CPAPI | ssoBearer | Provides All Contracts For Given Underlying Market. | `getForecastMarkets` |
| GET | `/v1/api/forecast/contract/rules` | CPAPI | ssoBearer | Event Contract Rules | `getForecastRules` |
| GET | `/v1/api/forecast/contract/schedules` | CPAPI | ssoBearer | Event Contract Schedules | `getForecastSchedule` |

### Trading FA Allocation Management *(8)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| GET | `/v1/api/iserver/account/allocation/accounts` | CPAPI | ssoBearer | List Allocatable Subaccounts | `getAllocatableSubaccounts` |
| GET | `/v1/api/iserver/account/allocation/group` | CPAPI | ssoBearer | List All Allocation Groups | `getAllocationGroups` |
| POST | `/v1/api/iserver/account/allocation/group` | CPAPI | ssoBearer | Add Allocation Group | `createAllocationGroup` |
| PUT | `/v1/api/iserver/account/allocation/group` | CPAPI | ssoBearer | Modify Allocation Group | `modifyAllocationGroup` |
| POST | `/v1/api/iserver/account/allocation/group/delete` | CPAPI | ssoBearer | Delete An Allocation Group | `deleteAllocationGroup` |
| POST | `/v1/api/iserver/account/allocation/group/single` | CPAPI | ssoBearer | Retrieve Single Allocation Group | `getSingleAllocationGroup` |
| GET | `/v1/api/iserver/account/allocation/presets` | CPAPI | ssoBearer | Retrieve Allocation Presets | `getAllocationPresets` |
| POST | `/v1/api/iserver/account/allocation/presets` | CPAPI | ssoBearer | Set Allocation Preset | `setAllocationPreset` |

### Trading FA Model Portfolios *(10)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| POST | `/v1/api/fa/fa-preset/get` | CPAPI | ssoBearer | Get Model Preset | `getModelPresets` |
| POST | `/v1/api/fa/fa-preset/save` | CPAPI | ssoBearer | Set Model Preset | `setModelPresets` |
| POST | `/v1/api/fa/model/accounts-details` | CPAPI | ssoBearer | Get Models Accounts | `getAccountsInModel` |
| POST | `/v1/api/fa/model/invest-divest` | CPAPI | ssoBearer | Invest Account Into Model | `setAccountinvestmentInModel` |
| POST | `/v1/api/fa/model/invest-divest-positions` | CPAPI | ssoBearer | Summary Of Accounts Invested In The Model | `getInvestedAccountsInModel` |
| POST | `/v1/api/fa/model/list` | CPAPI | ssoBearer | Request All Models | `getAllmodels` |
| POST | `/v1/api/fa/model/positions` | CPAPI | ssoBearer | Request Model Positions | `getAllModelPositions` |
| POST | `/v1/api/fa/model/save` | CPAPI | ssoBearer | Set Model Allocations | `setModelTargetPositions` |
| POST | `/v1/api/fa/model/submit-transfers` | CPAPI | ssoBearer | Submit Transfers | `submitModelOrders` |
| POST | `/v1/api/fa/model/summary` | CPAPI | ssoBearer | Request Model Summary | `getModelSummarySingle` |

### Trading FYIs and Notifications *(11)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| GET | `/v1/api/fyi/deliveryoptions` | CPAPI | ssoBearer | Get Delivery Options | `getFyiDelivery` |
| POST | `/v1/api/fyi/deliveryoptions/device` | CPAPI | ssoBearer | Toggle Delivery To A Device | `modifyFyiDelivery` |
| PUT | `/v1/api/fyi/deliveryoptions/email` | CPAPI | ssoBearer | Toggle Email Delivery | `modifyFyiEmails` |
| DELETE | `/v1/api/fyi/deliveryoptions/{deviceId}` | CPAPI | ssoBearer | Delete A Device | `deleteFyiDevice` |
| GET | `/v1/api/fyi/disclaimer/{typecode}` | CPAPI | ssoBearer | Get Disclaimers By FYI Type | `getFyiDisclaimerss` |
| PUT | `/v1/api/fyi/disclaimer/{typecode}` | CPAPI | ssoBearer | Mark FYI Disclaimer Read | `readFyiDisclaimer` |
| GET | `/v1/api/fyi/notifications` | CPAPI | ssoBearer | List All Notifications | `getAllFyis` |
| PUT | `/v1/api/fyi/notifications/{notificationId}` | CPAPI | ssoBearer | Mark Notification Read | `readFyiNotification` |
| GET | `/v1/api/fyi/settings` | CPAPI | ssoBearer | Get Notification Settings | `getFyiSettings` |
| POST | `/v1/api/fyi/settings/{typecode}` | CPAPI | ssoBearer | Modify FYI Notifications | `modifyFyiNotification` |
| GET | `/v1/api/fyi/unreadnumber` | CPAPI | ssoBearer | Get Number Of Unread Notifications | `getUnreadFyis` |

### Trading Market Data *(4)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| GET | `/v1/api/iserver/marketdata/history` | CPAPI | ssoBearer | Historical OHLC Bar Data | `getMdHistory` |
| GET | `/v1/api/iserver/marketdata/snapshot` | CPAPI | ssoBearer | Live Market Data Snapshot | `getMdSnapshot` |
| POST | `/v1/api/iserver/marketdata/unsubscribe` | CPAPI | ssoBearer | Close A Backend Data Stream | `closeMdStream` |
| GET | `/v1/api/iserver/marketdata/unsubscribeall` | CPAPI | ssoBearer | Close All Backend Data Streams | `closeAllMdStreams` |

### Trading OAuth 1.0a *(3)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| POST | `/v1/api/oauth/access_token` | CPAPI | ssoBearer | Generate An Access Token | `reqAccessToken` |
| POST | `/v1/api/oauth/live_session_token` | CPAPI | ssoBearer | Generate A Live Session Token | `reqLiveSessionToken` |
| POST | `/v1/api/oauth/request_token` | CPAPI | ssoBearer | Obtain A Request Token | `reqTempToken` |

### Trading Orders *(11)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| GET | `/v1/api/iserver/account/order/status/{orderId}` | CPAPI | ssoBearer | Status Of A Single Order | `getOrderStatus` |
| GET | `/v1/api/iserver/account/orders` | CPAPI | ssoBearer | List Open Orders | `getOpenOrders` |
| GET | `/v1/api/iserver/account/trades` | CPAPI | ssoBearer | Trade History | `getTradeHistory` |
| DELETE | `/v1/api/iserver/account/{accountId}/order/{orderId}` | CPAPI | ssoBearer | Cancel An Open Order | `cancelOpenOrder` |
| POST | `/v1/api/iserver/account/{accountId}/order/{orderId}` | CPAPI | ssoBearer | Modify Open Order | `modifyOpenOrder` |
| POST | `/v1/api/iserver/account/{accountId}/orders` | CPAPI | ssoBearer | Submit New Order | `submitNewOrder` |
| POST | `/v1/api/iserver/account/{accountId}/orders/whatif` | CPAPI | ssoBearer | New Order Preview | `previewMarginImpact` |
| POST | `/v1/api/iserver/notification` | CPAPI | ssoBearer | Dismiss Server Prompt | `ackServerPrompt` |
| POST | `/v1/api/iserver/questions/suppress` | CPAPI | ssoBearer | Suppress Order Reply Messages | `suppressOrderReplies` |
| POST | `/v1/api/iserver/questions/suppress/reset` | CPAPI | ssoBearer | Reset Order Reply Message Suppression | `resetOrderSuppression` |
| POST | `/v1/api/iserver/reply/{replyId}` | CPAPI | ssoBearer | Confirm Order Reply Message | `confirmOrderReply` |

### Trading Portfolio *(13)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| GET | `/v1/api/portfolio/accounts` | CPAPI | ssoBearer | List All Accounts | `getAllAccounts` |
| GET | `/v1/api/portfolio/positions/{conid}` | CPAPI | ssoBearer | All Account Positions In An Instrument | `getAllAccountsForConid` |
| GET | `/v1/api/portfolio/subaccounts` | CPAPI | ssoBearer | List All Subaccounts | `getAllSubaccounts` |
| GET | `/v1/api/portfolio/subaccounts2` | CPAPI | ssoBearer | Portfolio Subaccounts (Large Account Structures) | `getManySubaccounts` |
| GET | `/v1/api/portfolio/{accountId}/allocation` | CPAPI | ssoBearer | Account Allocations | `getAssetAllocation` |
| GET | `/v1/api/portfolio/{accountId}/combo/positions` | CPAPI | ssoBearer | Combination Positions | `getComboPositions` |
| GET | `/v1/api/portfolio/{accountId}/ledger` | CPAPI | ssoBearer | Account Ledger | `getPortfolioLedger` |
| GET | `/v1/api/portfolio/{accountId}/meta` | CPAPI | ssoBearer | Account Attributes | `getPortfolioMetadata` |
| GET | `/v1/api/portfolio/{accountId}/position/{conid}` | CPAPI | ssoBearer | Account Position In An Instrument | `getPositionByConid` |
| POST | `/v1/api/portfolio/{accountId}/positions/invalidate` | CPAPI | ssoBearer | Refresh Position Cache | `invalidatePositionCache` |
| GET | `/v1/api/portfolio/{accountId}/positions/{pageId}` | CPAPI | ssoBearer | Account Positions | `getPaginatedPositions` |
| GET | `/v1/api/portfolio/{accountId}/summary` | CPAPI | ssoBearer | Account Portfolio Summary | `getPortfolioSummary` |
| GET | `/v1/api/portfolio2/{accountId}/positions` | CPAPI | ssoBearer | Account Positions (NEW) | `getUncachedPositions` |

### Trading Portfolio Analyst *(4)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| POST | `/v1/api/pa/allocation` | CPAPI | ssoBearer | Portfolio Allocation | `createAllocation` |
| POST | `/v1/api/pa/allperiods` | CPAPI | ssoBearer | Account Performance (All Time Periods) | `getPerformanceAllPeriods` |
| POST | `/v1/api/pa/performance` | CPAPI | ssoBearer | Account Performance | `getSinglePerformancePeriod` |
| POST | `/v1/api/pa/transactions` | CPAPI | ssoBearer | Transaction History | `getTransactions` |

### Trading Scanner *(2)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| GET | `/v1/api/iserver/scanner/params` | CPAPI | ssoBearer | Get Valid IServer Scanner Parameters | `getScannerParameters` |
| POST | `/v1/api/iserver/scanner/run` | CPAPI | ssoBearer | Run An IServer Market Scanner | `getScannerResults` |

### Trading Session *(5)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| POST | `/v1/api/iserver/auth/ssodh/init` | CPAPI | ssoBearer | Initialize Brokerage Session | `initializeSession` |
| POST | `/v1/api/iserver/auth/status` | CPAPI | ssoBearer | Brokerage Session Status | `getBrokerageStatus` |
| POST | `/v1/api/logout` | CPAPI | ssoBearer | Terminate Web API Session | `logout` |
| GET | `/v1/api/sso/validate` | CPAPI | ssoBearer | Validate SSO Web API Session | `getSessionValidation` |
| POST | `/v1/api/tickle` | CPAPI | ssoBearer | Brokerage Keep-Alive Ping | `getSessionToken` |

### Trading Watchlists *(4)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| DELETE | `/v1/api/iserver/watchlist` | CPAPI | ssoBearer | Delete A Saved Watchlist | `deleteWatchlist` |
| GET | `/v1/api/iserver/watchlist` | CPAPI | ssoBearer | Return A Single Saved Watchlist | `getSpecificWatchlist` |
| POST | `/v1/api/iserver/watchlist` | CPAPI | ssoBearer | Create A Watchlist | `postNewWatchlist` |
| GET | `/v1/api/iserver/watchlists` | CPAPI | ssoBearer | Return All Saved Watchlists | `getAllWatchlists` |

### Trading Websocket *(1)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| GET | `/v1/api/ws` | CPAPI | ssoBearer | Open Websocket | `openWebsocket` |

### Utilities Echo *(2)*

| Method | Path | Surface | Auth | Summary | Operation ID |
|--------|------|---------|------|---------|--------------|
| GET | `/gw/api/v1/echo/https` | IB REST | oauth2Bearer | Echo A Request With HTTPS Security Policy Back After Validation. | `listEchoHttps` |
| POST | `/gw/api/v1/echo/signed-jwt` | IB REST | oauth2Bearer | Echo A Request With Signed JWT Security Policy Back After Validation. | `createEchoSignedJwt` |

---

## Known spec defects

The published spec does not generate cleanly. See [CODEGEN.md](./CODEGEN.md)
for the patches applied by `scripts/patch_spec.py`: path-parameter
mismatches, a duplicate `operationId`, and Go type-name collisions.

