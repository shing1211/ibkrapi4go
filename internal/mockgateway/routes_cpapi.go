// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package mockgateway

import "net/http"

// CPAPI operation IDs (the canonical opIds from docs/SPEC.md) that are not
// already declared alongside the Phase-1 routes. Two SPEC rows share the
// "getTradingSchedule" opId; both routes register under OpGetTradingSchedule.
const (
	// Trading accounts.
	OpGetAccountOwners        = "getAccountOwners"
	OpSetActiveAccount        = "setActiveAccount"
	OpGetDynamicAccounts      = "getDynamicAccounts"
	OpGetFundSummary          = "getFundSummary"
	OpGetBalanceSummary       = "getBalanceSummary"
	OpGetMarginSummary        = "getMarginSummary"
	OpGetAccountMarketSummary = "getAccountMarketSummary"
	OpSetDynamicAccount       = "setDynamicAccount"

	// Trading alerts.
	OpGetAlertDetails = "getAlertDetails"
	OpGetMtaDetails   = "getMtaDetails"
	OpCreateAlert     = "createAlert"
	OpActivateAlert   = "activateAlert"
	OpDeleteAlert     = "deleteAlert"
	OpGetAllAlerts    = "getAllAlerts"

	// Trading contracts.
	OpGetTradingSchedule      = "getTradingSchedule"
	OpGetAlgosByInstrument    = "getAlgosByInstrument"
	OpGetInfoAndRules         = "getInfoAndRules"
	OpGetCurrencyPairs        = "getCurrencyPairs"
	OpGetExchangeRates        = "getExchangeRates"
	OpGetBondFilters          = "getBondFilters"
	OpGetContractInfo         = "getContractInfo"
	OpGetConidsByExchange     = "getConidsByExchange"
	OpGetFutureBySymbol       = "getFutureBySymbol"
	OpGetInstrumentDefinition = "getInstrumentDefinition"
	OpGetStockBySymbol        = "getStockBySymbol"

	// Trading event contracts.
	OpGetForecastCategories = "getForecastCategories"
	OpGetForecastContract   = "getForecastContract"
	OpGetForecastMarkets    = "getForecastMarkets"
	OpGetForecastRules      = "getForecastRules"
	OpGetForecastSchedule   = "getForecastSchedule"

	// Trading FA allocation management.
	OpGetAllocatableSubaccounts = "getAllocatableSubaccounts"
	OpGetAllocationGroups       = "getAllocationGroups"
	OpCreateAllocationGroup     = "createAllocationGroup"
	OpModifyAllocationGroup     = "modifyAllocationGroup"
	OpDeleteAllocationGroup     = "deleteAllocationGroup"
	OpGetSingleAllocationGroup  = "getSingleAllocationGroup"
	OpGetAllocationPresets      = "getAllocationPresets"
	OpSetAllocationPreset       = "setAllocationPreset"

	// Trading FA model portfolios.
	OpGetModelPresets             = "getModelPresets"
	OpSetModelPresets             = "setModelPresets"
	OpGetAccountsInModel          = "getAccountsInModel"
	OpSetAccountInvestmentInModel = "setAccountinvestmentInModel"
	OpGetInvestedAccountsInModel  = "getInvestedAccountsInModel"
	OpGetAllModels                = "getAllmodels"
	OpGetAllModelPositions        = "getAllModelPositions"
	OpSetModelTargetPositions     = "setModelTargetPositions"
	OpSubmitModelOrders           = "submitModelOrders"
	OpGetModelSummarySingle       = "getModelSummarySingle"

	// Trading FYIs and notifications.
	OpGetFyiDelivery        = "getFyiDelivery"
	OpModifyFyiDelivery     = "modifyFyiDelivery"
	OpModifyFyiEmails       = "modifyFyiEmails"
	OpDeleteFyiDevice       = "deleteFyiDevice"
	OpGetFyiDisclaimers     = "getFyiDisclaimerss"
	OpReadFyiDisclaimer     = "readFyiDisclaimer"
	OpGetAllFyis            = "getAllFyis"
	OpReadFyiNotification   = "readFyiNotification"
	OpGetFyiSettings        = "getFyiSettings"
	OpModifyFyiNotification = "modifyFyiNotification"
	OpGetUnreadFyis         = "getUnreadFyis"

	// Trading OAuth 1.0a.
	OpReqAccessToken      = "reqAccessToken"
	OpReqLiveSessionToken = "reqLiveSessionToken"
	OpReqTempToken        = "reqTempToken"

	// Trading orders.
	OpAckServerPrompt       = "ackServerPrompt"
	OpSuppressOrderReplies  = "suppressOrderReplies"
	OpResetOrderSuppression = "resetOrderSuppression"

	// Trading portfolio.
	OpGetAllAccountsForConid = "getAllAccountsForConid"
	OpGetManySubaccounts     = "getManySubaccounts"
	OpGetComboPositions      = "getComboPositions"

	// Trading portfolio analyst.
	OpCreateAllocation           = "createAllocation"
	OpGetPerformanceAllPeriods   = "getPerformanceAllPeriods"
	OpGetSinglePerformancePeriod = "getSinglePerformancePeriod"
	OpGetTransactions            = "getTransactions"

	// Trading scanner.
	OpGetScannerParameters = "getScannerParameters"
	OpGetScannerResults    = "getScannerResults"

	// Trading watchlists.
	OpDeleteWatchlist      = "deleteWatchlist"
	OpGetSpecificWatchlist = "getSpecificWatchlist"
	OpPostNewWatchlist     = "postNewWatchlist"
	OpGetAllWatchlists     = "getAllWatchlists"

	// Trading websocket.
	OpOpenWebsocket = "openWebsocket"
)

// cpapiRoutes returns the CPAPI routes that defaultRoutes does not already
// register. Every route is protected (ssoBearer); the session routes are the
// only unauthenticated CPAPI operations and they live in defaultRoutes.
func cpapiRoutes() []route {
	post := []string{http.MethodPost}
	get := []string{http.MethodGet}
	put := []string{http.MethodPut}
	del := []string{http.MethodDelete}

	return []route{
		// --- trading accounts ---
		mkRoute(OpGetAccountOwners, get, "/v1/api/acesws/{accountId}/signatures-and-owners", true),
		mkRoute(OpSetActiveAccount, post, "/v1/api/iserver/account", true),
		mkRoute(OpGetDynamicAccounts, get, "/v1/api/iserver/account/search/{searchPattern}", true),
		mkRoute(OpGetFundSummary, get, "/v1/api/iserver/account/{accountId}/summary/available_funds", true),
		mkRoute(OpGetBalanceSummary, get, "/v1/api/iserver/account/{accountId}/summary/balances", true),
		mkRoute(OpGetMarginSummary, get, "/v1/api/iserver/account/{accountId}/summary/margins", true),
		mkRoute(OpGetAccountMarketSummary, get, "/v1/api/iserver/account/{accountId}/summary/market_value", true),
		mkRoute(OpSetDynamicAccount, post, "/v1/api/iserver/dynaccount", true),

		// --- trading alerts ---
		mkRoute(OpGetAlertDetails, get, "/v1/api/iserver/account/alert/{alertId}", true),
		mkRoute(OpGetMtaDetails, get, "/v1/api/iserver/account/mta", true),
		mkRoute(OpCreateAlert, post, "/v1/api/iserver/account/{accountId}/alert", true),
		mkRoute(OpActivateAlert, post, "/v1/api/iserver/account/{accountId}/alert/activate", true),
		mkRoute(OpDeleteAlert, del, "/v1/api/iserver/account/{accountId}/alert/{alertId}", true),
		mkRoute(OpGetAllAlerts, get, "/v1/api/iserver/account/{accountId}/alerts", true),

		// --- trading contracts ---
		mkRoute(OpGetTradingSchedule, get, "/v1/api/contract/trading-schedule", true),
		mkRoute(OpGetAlgosByInstrument, get, "/v1/api/iserver/contract/{conid}/algos", true),
		mkRoute(OpGetInfoAndRules, get, "/v1/api/iserver/contract/{conid}/info-and-rules", true),
		mkRoute(OpGetCurrencyPairs, get, "/v1/api/iserver/currency/pairs", true),
		mkRoute(OpGetExchangeRates, get, "/v1/api/iserver/exchangerate", true),
		mkRoute(OpGetBondFilters, get, "/v1/api/iserver/secdef/bond-filters", true),
		mkRoute(OpGetContractInfo, get, "/v1/api/iserver/secdef/info", true),
		mkRoute(OpGetConidsByExchange, get, "/v1/api/trsrv/all-conids", true),
		mkRoute(OpGetFutureBySymbol, get, "/v1/api/trsrv/futures", true),
		mkRoute(OpGetInstrumentDefinition, get, "/v1/api/trsrv/secdef", true),
		mkRoute(OpGetTradingSchedule, get, "/v1/api/trsrv/secdef/schedule", true),
		mkRoute(OpGetStockBySymbol, get, "/v1/api/trsrv/stocks", true),

		// --- trading event contracts ---
		mkRoute(OpGetForecastCategories, get, "/v1/api/forecast/category/tree", true),
		mkRoute(OpGetForecastContract, get, "/v1/api/forecast/contract/details", true),
		mkRoute(OpGetForecastMarkets, get, "/v1/api/forecast/contract/market", true),
		mkRoute(OpGetForecastRules, get, "/v1/api/forecast/contract/rules", true),
		mkRoute(OpGetForecastSchedule, get, "/v1/api/forecast/contract/schedules", true),

		// --- trading FA allocation management ---
		mkRoute(OpGetAllocatableSubaccounts, get, "/v1/api/iserver/account/allocation/accounts", true),
		mkRoute(OpGetAllocationGroups, get, "/v1/api/iserver/account/allocation/group", true),
		mkRoute(OpCreateAllocationGroup, post, "/v1/api/iserver/account/allocation/group", true),
		mkRoute(OpModifyAllocationGroup, put, "/v1/api/iserver/account/allocation/group", true),
		mkRoute(OpDeleteAllocationGroup, post, "/v1/api/iserver/account/allocation/group/delete", true),
		mkRoute(OpGetSingleAllocationGroup, post, "/v1/api/iserver/account/allocation/group/single", true),
		mkRoute(OpGetAllocationPresets, get, "/v1/api/iserver/account/allocation/presets", true),
		mkRoute(OpSetAllocationPreset, post, "/v1/api/iserver/account/allocation/presets", true),

		// --- trading FA model portfolios ---
		mkRoute(OpGetModelPresets, post, "/v1/api/fa/fa-preset/get", true),
		mkRoute(OpSetModelPresets, post, "/v1/api/fa/fa-preset/save", true),
		mkRoute(OpGetAccountsInModel, post, "/v1/api/fa/model/accounts-details", true),
		mkRoute(OpSetAccountInvestmentInModel, post, "/v1/api/fa/model/invest-divest", true),
		mkRoute(OpGetInvestedAccountsInModel, post, "/v1/api/fa/model/invest-divest-positions", true),
		mkRoute(OpGetAllModels, post, "/v1/api/fa/model/list", true),
		mkRoute(OpGetAllModelPositions, post, "/v1/api/fa/model/positions", true),
		mkRoute(OpSetModelTargetPositions, post, "/v1/api/fa/model/save", true),
		mkRoute(OpSubmitModelOrders, post, "/v1/api/fa/model/submit-transfers", true),
		mkRoute(OpGetModelSummarySingle, post, "/v1/api/fa/model/summary", true),

		// --- trading FYIs and notifications ---
		mkRoute(OpGetFyiDelivery, get, "/v1/api/fyi/deliveryoptions", true),
		mkRoute(OpModifyFyiDelivery, post, "/v1/api/fyi/deliveryoptions/device", true),
		mkRoute(OpModifyFyiEmails, put, "/v1/api/fyi/deliveryoptions/email", true),
		mkRoute(OpDeleteFyiDevice, del, "/v1/api/fyi/deliveryoptions/{deviceId}", true),
		mkRoute(OpGetFyiDisclaimers, get, "/v1/api/fyi/disclaimer/{typecode}", true),
		mkRoute(OpReadFyiDisclaimer, put, "/v1/api/fyi/disclaimer/{typecode}", true),
		mkRoute(OpGetAllFyis, get, "/v1/api/fyi/notifications", true),
		mkRoute(OpReadFyiNotification, put, "/v1/api/fyi/notifications/{notificationId}", true),
		mkRoute(OpGetFyiSettings, get, "/v1/api/fyi/settings", true),
		mkRoute(OpModifyFyiNotification, post, "/v1/api/fyi/settings/{typecode}", true),
		mkRoute(OpGetUnreadFyis, get, "/v1/api/fyi/unreadnumber", true),

		// --- trading OAuth 1.0a ---
		mkRoute(OpReqAccessToken, post, "/v1/api/oauth/access_token", true),
		mkRoute(OpReqLiveSessionToken, post, "/v1/api/oauth/live_session_token", true),
		mkRoute(OpReqTempToken, post, "/v1/api/oauth/request_token", true),

		// --- trading orders ---
		mkRoute(OpAckServerPrompt, post, "/v1/api/iserver/notification", true),
		mkRoute(OpSuppressOrderReplies, post, "/v1/api/iserver/questions/suppress", true),
		mkRoute(OpResetOrderSuppression, post, "/v1/api/iserver/questions/suppress/reset", true),

		// --- trading portfolio ---
		mkRoute(OpGetAllAccountsForConid, get, "/v1/api/portfolio/positions/{conid}", true),
		mkRoute(OpGetManySubaccounts, get, "/v1/api/portfolio/subaccounts2", true),
		mkRoute(OpGetComboPositions, get, "/v1/api/portfolio/{accountId}/combo/positions", true),

		// --- trading portfolio analyst ---
		mkRoute(OpCreateAllocation, post, "/v1/api/pa/allocation", true),
		mkRoute(OpGetPerformanceAllPeriods, post, "/v1/api/pa/allperiods", true),
		mkRoute(OpGetSinglePerformancePeriod, post, "/v1/api/pa/performance", true),
		mkRoute(OpGetTransactions, post, "/v1/api/pa/transactions", true),

		// --- trading scanner ---
		mkRoute(OpGetScannerParameters, get, "/v1/api/iserver/scanner/params", true),
		mkRoute(OpGetScannerResults, post, "/v1/api/iserver/scanner/run", true),

		// --- trading watchlists ---
		mkRoute(OpDeleteWatchlist, del, "/v1/api/iserver/watchlist", true),
		mkRoute(OpGetSpecificWatchlist, get, "/v1/api/iserver/watchlist", true),
		mkRoute(OpPostNewWatchlist, post, "/v1/api/iserver/watchlist", true),
		mkRoute(OpGetAllWatchlists, get, "/v1/api/iserver/watchlists", true),

		// --- trading websocket (route only; streaming lands in T4) ---
		mkRoute(OpOpenWebsocket, get, "/v1/api/ws", true),
	}
}

// registerCPAPIFixtures adds the T2 CPAPI fixtures to f. Every fixture is valid
// JSON for the corresponding pkg/ibkr wrapper: monetary and quantity values are
// JSON strings (ADR 0008). Mutations return small, repeatable acknowledgements.
func registerCPAPIFixtures(f *Fixtures) {
	// --- trading accounts ---
	f.Set(OpGetAccountOwners, Fixture{Body: `{"accountId":"U1234567","owners":[{"name":"Jane Doe","ownerType":"INDIVIDUAL"}],"signatures":["Jane Doe"]}`})
	f.Set(OpSetActiveAccount, Fixture{Body: `{}`})
	f.Set(OpGetDynamicAccounts, Fixture{Body: `[{"accountId":"U1234567","accountTitle":"Main Account"}]`})
	f.Set(OpGetFundSummary, Fixture{Body: `{"accountId":"U1234567","buyingPower":"4000.00","availableFunds":"900.10","netLiquidationValue":"1234.56"}`})
	f.Set(OpGetBalanceSummary, Fixture{Body: `{"accountId":"U1234567","balance":"1000.00","settledCash":"500.00","accruedInterest":"1.25"}`})
	f.Set(OpGetMarginSummary, Fixture{Body: `{"accountId":"U1234567","initialMargin":"100.00","maintenanceMargin":"80.00","excessLiquidity":"900.00"}`})
	f.Set(OpGetAccountMarketSummary, Fixture{Body: `{"accountId":"U1234567","stock":"1000.50","futures":"0","stocksAndFutures":"1000.50"}`})
	f.Set(OpSetDynamicAccount, Fixture{Body: `{}`})

	// --- trading alerts ---
	f.Set(OpGetAlertDetails, Fixture{Body: `{"alertId":"123","accountId":"U1234567","name":"AAPL above 150","alertType":"price","enabled":true,"filled":false}`})
	f.Set(OpGetMtaDetails, Fixture{Body: `{"accounts":["U1234567","U7654321"]}`})
	f.Set(OpCreateAlert, Fixture{Body: `{"alertId":"124"}`})
	f.Set(OpActivateAlert, Fixture{Body: `{"alertId":"123","active":true}`})
	f.Set(OpDeleteAlert, Fixture{Body: `{}`})
	f.Set(OpGetAllAlerts, Fixture{Body: `[{"alertId":"123","accountId":"U1234567","name":"AAPL above 150","alertType":"price","enabled":true,"filled":false}]`})

	// --- trading contracts ---
	f.Set(OpGetTradingSchedule, Fixture{Body: `{"market":"AAPL","hours":"09:30-16:00"}`})
	f.Set(OpGetAlgosByInstrument, Fixture{Body: `{"algos":[{"algoId":"1","name":"VWAP"}]}`})
	f.Set(OpGetInfoAndRules, Fixture{Body: `{"conId":265598,"symbol":"AAPL"}`})
	f.Set(OpGetCurrencyPairs, Fixture{Body: `[{"conId":1,"sourceCurrency":"EUR","targetCurrency":"USD","exchangeRate":"1.08"}]`})
	f.Set(OpGetExchangeRates, Fixture{Body: `[{"fromCurrency":"EUR","toCurrency":"USD","rate":"1.08"}]`})
	f.Set(OpGetBondFilters, Fixture{Body: `[{"instrumentId":"US912828","coupon":"2.5","maturityDate":"2030-01-15"}]`})
	f.Set(OpGetContractInfo, Fixture{Body: `[{"conId":265598,"symbol":"AAPL","securityType":"STK","exchange":"NASDAQ","currency":"USD"}]`})
	f.Set(OpGetConidsByExchange, Fixture{Body: `[{"conid":265598,"symbol":"AAPL","exchange":"NASDAQ","securityType":"STK"}]`})
	f.Set(OpGetFutureBySymbol, Fixture{Body: `{"AAPL":[{"conid":265598,"symbol":"AAPL","expiry":"2026-12","exchange":"GLOBEX"}]}`})
	f.Set(OpGetInstrumentDefinition, Fixture{Body: `[{"conid":265598,"symbol":"AAPL","companyName":"Apple Inc","securityType":"STK","exchange":"NASDAQ","currency":"USD"}]`})
	f.Set(OpGetStockBySymbol, Fixture{Body: `{"AAPL":[{"conid":265598,"symbol":"AAPL","exchange":"NASDAQ","securityType":"STK"}]}`})

	// --- trading event contracts ---
	f.Set(OpGetForecastCategories, Fixture{Body: `[{"id":"1","name":"Elections"}]`})
	f.Set(OpGetForecastContract, Fixture{Body: `{"conId":123456,"description":"US Presidential Election","category":"Elections"}`})
	f.Set(OpGetForecastMarkets, Fixture{Body: `[{"conId":123456,"symbol":"USPREZ","description":"US Presidential Election"}]`})
	f.Set(OpGetForecastRules, Fixture{Body: `[{"ruleId":"1","description":"Contract settles on the certified result"}]`})
	f.Set(OpGetForecastSchedule, Fixture{Body: `[{"conId":123456,"scheduleTime":"2026-11-03T00:00:00Z"}]`})

	// --- trading FA allocation management ---
	f.Set(OpGetAllocatableSubaccounts, Fixture{Body: `["U1234567","U7654321"]`})
	f.Set(OpGetAllocationGroups, Fixture{Body: `[{"name":"Group A","isHidden":false,"method":"EQUAL"}]`})
	f.Set(OpCreateAllocationGroup, Fixture{Body: `{}`})
	f.Set(OpModifyAllocationGroup, Fixture{Body: `{}`})
	f.Set(OpDeleteAllocationGroup, Fixture{Body: `{}`})
	f.Set(OpGetSingleAllocationGroup, Fixture{Body: `{"name":"Group A","isHidden":false,"method":"EQUAL"}`})
	f.Set(OpGetAllocationPresets, Fixture{Body: `[{"accountId":"U1234567","percentage":"50.0"}]`})
	f.Set(OpSetAllocationPreset, Fixture{Body: `{}`})

	// --- trading FA model portfolios ---
	f.Set(OpGetModelPresets, Fixture{Body: `[{"name":"Balanced","accounts":["U1234567"]}]`})
	f.Set(OpSetModelPresets, Fixture{Body: `{}`})
	f.Set(OpGetAccountsInModel, Fixture{Body: `[{"accountId":"U1234567","accountAlias":"Main"}]`})
	f.Set(OpSetAccountInvestmentInModel, Fixture{Body: `{}`})
	f.Set(OpGetInvestedAccountsInModel, Fixture{Body: `{"model":"Balanced","accounts":[]}`})
	f.Set(OpGetAllModels, Fixture{Body: `["Balanced","Growth"]`})
	f.Set(OpGetAllModelPositions, Fixture{Body: `[{"accountId":"U1234567","conId":265598,"symbol":"AAPL","position":"10","avgCost":"145.25"}]`})
	f.Set(OpSetModelTargetPositions, Fixture{Body: `{}`})
	f.Set(OpSubmitModelOrders, Fixture{Body: `{}`})
	f.Set(OpGetModelSummarySingle, Fixture{Body: `{"name":"Balanced","accountIds":["U1234567"]}`})

	// --- trading FYIs and notifications ---
	f.Set(OpGetFyiDelivery, Fixture{Body: `[{"deviceId":"dev-1","type":"email","value":"user@example.com"}]`})
	f.Set(OpModifyFyiDelivery, Fixture{Body: `{}`})
	f.Set(OpModifyFyiEmails, Fixture{Body: `{}`})
	f.Set(OpDeleteFyiDevice, Fixture{Body: `{}`})
	f.Set(OpGetFyiDisclaimers, Fixture{Body: `{"typeCode":"1","content":"Synthetic disclaimer."}`})
	f.Set(OpReadFyiDisclaimer, Fixture{Body: `{}`})
	f.Set(OpGetAllFyis, Fixture{Body: `[{"id":"1","typeCode":"1","subject":"Welcome","read":false}]`})
	f.Set(OpReadFyiNotification, Fixture{Body: `{}`})
	f.Set(OpGetFyiSettings, Fixture{Body: `[{"typeCode":"1","enabled":true}]`})
	f.Set(OpModifyFyiNotification, Fixture{Body: `{}`})
	f.Set(OpGetUnreadFyis, Fixture{Body: `{"count":3}`})

	// --- trading OAuth 1.0a ---
	f.Set(OpReqAccessToken, Fixture{Body: `{"token":"access-token","oauth_token":"access-token","oauth_token_secret":"secret"}`})
	f.Set(OpReqLiveSessionToken, Fixture{Body: `{"token":"live-session-token"}`})
	f.Set(OpReqTempToken, Fixture{Body: `{"token":"temp-token","oauth_token":"temp-token","oauth_token_secret":"secret"}`})

	// --- trading orders ---
	f.Set(OpAckServerPrompt, Fixture{Body: `{}`})
	f.Set(OpSuppressOrderReplies, Fixture{Body: `{}`})
	f.Set(OpResetOrderSuppression, Fixture{Body: `{}`})

	// --- trading portfolio ---
	f.Set(OpGetAllAccountsForConid, Fixture{Body: `[{"accountId":"U1234567","accountTitle":"Main","accountAlias":"Main","currency":"USD"}]`})
	f.Set(OpGetManySubaccounts, Fixture{Body: `[{"accountId":"U1234567","accountTitle":"Main","accountAlias":"Main","currency":"USD"}]`})
	f.Set(OpGetComboPositions, Fixture{Body: `[{"acctId":"U1234567","conid":265598,"symbol":"AAPL","position":"10"}]`})

	// --- trading portfolio analyst ---
	f.Set(OpCreateAllocation, Fixture{Body: `{}`})
	f.Set(OpGetPerformanceAllPeriods, Fixture{Body: `{"data":[]}`})
	f.Set(OpGetSinglePerformancePeriod, Fixture{Body: `{"accountId":"U1234567","data":{}}`})
	f.Set(OpGetTransactions, Fixture{Body: `[{"accountId":"U1234567","conId":265598,"symbol":"AAPL","side":"BUY","quantity":"10","price":"150.00","amount":"1500.00"}]`})

	// --- trading scanner ---
	f.Set(OpGetScannerParameters, Fixture{Body: `{"scanTypes":["TOP_PERC_GAIN"],"instruments":["STK"]}`})
	f.Set(OpGetScannerResults, Fixture{Body: `[{"conid":265598,"symbol":"AAPL","companyName":"Apple Inc","exchange":"NASDAQ","secType":"STK","distance":"1.5"}]`})

	// --- trading watchlists ---
	f.Set(OpDeleteWatchlist, Fixture{Body: `{}`})
	f.Set(OpGetSpecificWatchlist, Fixture{Body: `{"id":"wl-1","name":"Tech","instruments":[{"conId":265598,"symbol":"AAPL"}]}`})
	f.Set(OpPostNewWatchlist, Fixture{Body: `{}`})
	f.Set(OpGetAllWatchlists, Fixture{Body: `[{"id":"wl-1","name":"Tech","instruments":[{"conId":265598,"symbol":"AAPL"}]}]`})

	// --- trading websocket ---
	f.Set(OpOpenWebsocket, Fixture{Body: `{"status":"ok"}`})
}
