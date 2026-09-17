// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/shing1211/ibkrapi4go/internal/mockgateway"
)

type fixtureCase struct {
	name string
	body string
}

func allFixtureCases() []fixtureCase {
	f := mockgateway.DefaultFixtures()

	ops := []string{
		mockgateway.OpInitializeSession,
		mockgateway.OpGetBrokerageStatus,
		mockgateway.OpGetSessionToken,
		mockgateway.OpLogout,
		mockgateway.OpGetSessionValidation,
		mockgateway.OpGetBrokerageAccounts,
		mockgateway.OpGetPnl,
		mockgateway.OpGetAccountSummary,
		mockgateway.OpGetContractSymbols,
		mockgateway.OpGetContractSymbolsFromBody,
		mockgateway.OpGetInstrumentInfo,
		mockgateway.OpGetContractRules,
		mockgateway.OpGetContractStrikes,
		mockgateway.OpGetAllAccounts,
		mockgateway.OpGetAllSubaccounts,
		mockgateway.OpGetUncachedPositions,
		mockgateway.OpInvalidatePositionCache,
		mockgateway.OpGetPaginatedPositions,
		mockgateway.OpGetPositionByConid,
		mockgateway.OpGetPortfolioLedger,
		mockgateway.OpGetAssetAllocation,
		mockgateway.OpGetPortfolioSummary,
		mockgateway.OpGetPortfolioMetadata,
		mockgateway.OpGetMdSnapshot,
		mockgateway.OpGetMdHistory,
		mockgateway.OpCloseMdStream,
		mockgateway.OpCloseAllMdStreams,
		mockgateway.OpSubmitNewOrder,
		mockgateway.OpConfirmOrderReply,
		mockgateway.OpPreviewMarginImpact,
		mockgateway.OpModifyOpenOrder,
		mockgateway.OpCancelOpenOrder,
		mockgateway.OpGetOpenOrders,
		mockgateway.OpGetOrderStatus,
		mockgateway.OpGetTradeHistory,
		mockgateway.OpGetAccountOwners,
		mockgateway.OpSetActiveAccount,
		mockgateway.OpGetDynamicAccounts,
		mockgateway.OpGetFundSummary,
		mockgateway.OpGetBalanceSummary,
		mockgateway.OpGetMarginSummary,
		mockgateway.OpGetAccountMarketSummary,
		mockgateway.OpSetDynamicAccount,
		mockgateway.OpGetAlertDetails,
		mockgateway.OpGetMtaDetails,
		mockgateway.OpCreateAlert,
		mockgateway.OpActivateAlert,
		mockgateway.OpDeleteAlert,
		mockgateway.OpGetAllAlerts,
		mockgateway.OpGetTradingSchedule,
		mockgateway.OpGetAlgosByInstrument,
		mockgateway.OpGetInfoAndRules,
		mockgateway.OpGetCurrencyPairs,
		mockgateway.OpGetExchangeRates,
		mockgateway.OpGetBondFilters,
		mockgateway.OpGetContractInfo,
		mockgateway.OpGetConidsByExchange,
		mockgateway.OpGetFutureBySymbol,
		mockgateway.OpGetInstrumentDefinition,
		mockgateway.OpGetStockBySymbol,
		mockgateway.OpGetForecastCategories,
		mockgateway.OpGetForecastContract,
		mockgateway.OpGetForecastMarkets,
		mockgateway.OpGetForecastRules,
		mockgateway.OpGetForecastSchedule,
		mockgateway.OpGetAllocatableSubaccounts,
		mockgateway.OpGetAllocationGroups,
		mockgateway.OpCreateAllocationGroup,
		mockgateway.OpModifyAllocationGroup,
		mockgateway.OpDeleteAllocationGroup,
		mockgateway.OpGetSingleAllocationGroup,
		mockgateway.OpGetAllocationPresets,
		mockgateway.OpSetAllocationPreset,
		mockgateway.OpGetModelPresets,
		mockgateway.OpSetModelPresets,
		mockgateway.OpGetAccountsInModel,
		mockgateway.OpSetAccountInvestmentInModel,
		mockgateway.OpGetInvestedAccountsInModel,
		mockgateway.OpGetAllModels,
		mockgateway.OpGetAllModelPositions,
		mockgateway.OpSetModelTargetPositions,
		mockgateway.OpSubmitModelOrders,
		mockgateway.OpGetModelSummarySingle,
		mockgateway.OpGetFyiDelivery,
		mockgateway.OpModifyFyiDelivery,
		mockgateway.OpModifyFyiEmails,
		mockgateway.OpDeleteFyiDevice,
		mockgateway.OpGetFyiDisclaimers,
		mockgateway.OpReadFyiDisclaimer,
		mockgateway.OpGetAllFyis,
		mockgateway.OpReadFyiNotification,
		mockgateway.OpGetFyiSettings,
		mockgateway.OpModifyFyiNotification,
		mockgateway.OpGetUnreadFyis,
		mockgateway.OpReqAccessToken,
		mockgateway.OpReqLiveSessionToken,
		mockgateway.OpReqTempToken,
		mockgateway.OpAckServerPrompt,
		mockgateway.OpSuppressOrderReplies,
		mockgateway.OpResetOrderSuppression,
		mockgateway.OpGetAllAccountsForConid,
		mockgateway.OpGetManySubaccounts,
		mockgateway.OpGetComboPositions,
		mockgateway.OpCreateAllocation,
		mockgateway.OpGetPerformanceAllPeriods,
		mockgateway.OpGetSinglePerformancePeriod,
		mockgateway.OpGetTransactions,
		mockgateway.OpGetScannerParameters,
		mockgateway.OpGetScannerResults,
		mockgateway.OpDeleteWatchlist,
		mockgateway.OpGetSpecificWatchlist,
		mockgateway.OpPostNewWatchlist,
		mockgateway.OpGetAllWatchlists,
		mockgateway.OpOpenWebsocket,
		mockgateway.OpListAccounts,
		mockgateway.OpUpdateAccounts,
		mockgateway.OpCreateAccounts,
		mockgateway.OpCreateAccountsDocuments,
		mockgateway.OpListAccountsLoginMessages,
		mockgateway.OpListAccountsStatus,
		mockgateway.OpGetAccountsDetails,
		mockgateway.OpGetAccountsKyc,
		mockgateway.OpGetAccountsLoginMessages,
		mockgateway.OpGetAccountsStatus,
		mockgateway.OpUpdateAccountsStatus,
		mockgateway.OpGetAccountsTasks,
		mockgateway.OpUpdateAccountsTasks,
		mockgateway.OpCreateAccountsTasks,
		mockgateway.OpCreateBankInstructions,
		mockgateway.OpCreateBankInstructionsQuery,
		mockgateway.OpBulkBankInstructions,
		mockgateway.OpGetClientInstructions,
		mockgateway.OpCreateExternalAssetTransfers,
		mockgateway.OpBulkExternalAssetTransfers,
		mockgateway.OpCreateExternalCashTransfers,
		mockgateway.OpCreateExternalCashTransfersQuery,
		mockgateway.OpBulkExternalCashTransfers,
		mockgateway.OpGetInstructionSets,
		mockgateway.OpCreateInstructionsCancel,
		mockgateway.OpBulkInstructionsCancel,
		mockgateway.OpCreateInstructionsQuery,
		mockgateway.OpGetInstructions,
		mockgateway.OpCreateInternalAssetTransfers,
		mockgateway.OpBulkInternalAssetTransfers,
		mockgateway.OpCreateInternalCashTransfers,
		mockgateway.OpBulkInternalCashTransfers,
		mockgateway.OpCreateExternalAssetTransfers2,
		mockgateway.OpBulkExternalAssetTransfers2,
		mockgateway.OpCreateStatements,
		mockgateway.OpListStatementsAvailable,
		mockgateway.OpCreateTaxDocuments,
		mockgateway.OpListTaxDocumentsAvailable,
		mockgateway.OpCreateTradeConfirmations,
		mockgateway.OpListTradeConfirmationsAvailable,
		mockgateway.OpListEnumerationsComplexAssetTransfer,
		mockgateway.OpGetEnumerations,
		mockgateway.OpListForms,
		mockgateway.OpListFormsRequiredForms,
		mockgateway.OpListParticipatingBanks,
		mockgateway.OpListRequests,
		mockgateway.OpGetRequestsStatus,
		mockgateway.OpUpdateRequestsStatus,
		mockgateway.OpGetValidationsUsernames,
		mockgateway.OpCreateSsoBrowserSessions,
		mockgateway.OpCreateSsoSessions,
		mockgateway.OpGenerateToken,
		mockgateway.OpApplyCSV,
		mockgateway.OpGetAccountRestrictions,
		mockgateway.OpGetMasterRestrictionIds,
		mockgateway.OpGetListDetails,
		mockgateway.OpGetMasterListIds,
		mockgateway.OpGetRestrictionDetails,
		mockgateway.OpGetRestrictionScope,
		mockgateway.OpGetUserRestrictions,
		mockgateway.OpVerifyCSV,
		mockgateway.OpCreateBalancesQuery,
		mockgateway.OpCreateTaxVoucherRequests,
		mockgateway.OpGetActiveCountryList,
		mockgateway.OpFetchDividends1,
		mockgateway.OpGetYears,
		mockgateway.OpDownloadFile,
		mockgateway.OpGetCurrentState1,
		mockgateway.OpListEchoHttps,
		mockgateway.OpCreateEchoSignedJwt,
	}

	var cases []fixtureCase
	seen := make(map[string]bool)

	for _, op := range ops {
		if seen[op] {
			continue
		}
		seen[op] = true

		if fx, ok := f.Get(op); ok {
			cases = append(cases, fixtureCase{
				name: op,
				body: fx.Body,
			})
		}
	}

	return cases
}

func FuzzAllFixturesDecode(f *testing.F) {
	cases := allFixtureCases()

	for _, c := range cases {
		f.Add(c.name, c.body)
	}

	f.Fuzz(func(t *testing.T, opName, body string) {
		if body == "" {
			return
		}

		var v any
		dec := json.NewDecoder(bytes.NewReader([]byte(body)))
		dec.UseNumber()
		if err := dec.Decode(&v); err != nil {
			return
		}
		_ = v
	})
}

func FuzzFixtureMutation(f *testing.F) {
	cases := allFixtureCases()

	for _, c := range cases {
		f.Add(c.name, c.body)
	}

	f.Fuzz(func(t *testing.T, opName, body string) {
		if body == "" {
			return
		}

		data := []byte(body)

		for i := 0; i < len(data); i++ {
			d := make([]byte, len(data))
			copy(d, data)
			d[i] = data[i] ^ 0xFF
			var v any
			dec := json.NewDecoder(bytes.NewReader(d))
			dec.UseNumber()
			_ = dec.Decode(&v)
		}

		for n := 0; n <= len(body); n++ {
			for i := 0; i < len(body)-n; i++ {
				d := append([]byte(body[:i]), body[i+n:]...)
				var v any
				dec := json.NewDecoder(bytes.NewReader(d))
				dec.UseNumber()
				_ = dec.Decode(&v)
			}
		}

		truncated := make([]byte, len(body)/2)
		copy(truncated, body[:len(truncated)])
		var v1 any
		dec1 := json.NewDecoder(bytes.NewReader(truncated))
		dec1.UseNumber()
		_ = dec1.Decode(&v1)

		malformed := append(append([]byte{}, body...), 0x00, 0x00, 0x00)
		var v2 any
		dec2 := json.NewDecoder(bytes.NewReader(malformed))
		dec2.UseNumber()
		_ = dec2.Decode(&v2)

		var v3 any
		dec3 := json.NewDecoder(bytes.NewReader([]byte(body)))
		dec3.UseNumber()
		_ = dec3.Decode(&v3)
	})
}

func FuzzGenericJSONDecode(f *testing.F) {
	f.Add(`{"authenticated":true,"established":true}`)
	f.Add(`{"accounts":["U1234567","U7654321"],"aliases":{"U1234567":"Main"}}`)
	f.Add(`{"upnl":{"U1234567.Core":{"dpl":"1.5","el":"2","mv":"3","nl":"4","rowType":"1","upl":"5"}}}`)
	f.Add(`{"accountType":"INDIVIDUAL","netLiquidationValue":"1234.5600","totalCashValue":"100.25","availableFunds":"900.10","SMA":"1200.0000","cashBalances":[{"currency":"USD","balance":"50.5","settledCash":"40.0"}]}`)
	f.Add(`[{"conid":265598,"symbol":"AAPL","companyName":"Apple Inc","secType":"STK","description":"Apple Inc","exchange":"NASDAQ"}]`)
	f.Add(`{"con_id":265598,"symbol":"AAPL","company_name":"Apple Inc","currency":"USD","exchange":"NASDAQ","instrument_type":"STK","local_symbol":"AAPL","multiplier":"1","expiry_full":"","cusip":"037833100","category":"Technology","industry":"Consumer Electronics"}`)
	f.Add(`{"algoEligible":true,"allOrNoneEligible":true,"canTradeAcctIds":["U1234567"],"defaultSize":"100","limitPrice":"0.01","orderTypes":["MKT","LMT"],"tifTypes":["DAY","GTC"],"TIF":"DAY","negativeCapable":false,"preview":true}`)
	f.Add(`{"call":["150","155"],"put":["145","140"]}`)
	f.Add(`[{"accountId":"U1234567","accountTitle":"Main","currency":"USD","accountAlias":"Main"},{"accountId":"U7654321","accountTitle":"Second","currency":"USD"}]`)
	f.Add(`[{"acctId":"U1234567","conid":265598,"contractDesc":"AAPL","assetClass":"STK","currency":"USD","position":"10.5","avgCost":"145.25","mktPrice":"150.00","mktValue":"1575.00","unrealizedPnl":"49.875"}]`)
	f.Add(`{"status":"success"}`)
	f.Add(`{"USD":{"acctcode":"U1234567","currency":"USD","cashbalance":"100.25","netliquidationvalue":"1575.00","stockmarketvalue":"1575.00","unrealizedpnl":"49.875","realizedpnl":"0"}}`)
	f.Add(`{"assetClass":{"long":{"STK":"1000.5"},"short":{}},"sector":{"long":{"Technology":"1000.5"},"short":{}},"group":{"long":{},"short":{}}}`)
	f.Add(`{"netliquidation":{"amount":"1575.00","currency":"USD"},"totalcashvalue":{"amount":"100.25","currency":"USD"}}`)
	f.Add(`{"accountId":"U1234567","accountTitle":"Main","currency":"USD","accountAlias":"Main"}`)
	f.Add(`[{"conid":265598,"31":"150.25","84":"150.20","_updated":1564652478,"server_id":"q0"}]`)
	f.Add(`{"symbol":"AAPL","data":[{"t":1564652478,"o":"150.1","h":"151.2","l":"149.9","c":"150.9","v":"1000"}]}`)
	f.Add(`[{"order_id":"999","order_status":"PreSubmitted"}]`)
	f.Add(`{"amount":{"initial":"1000.50","maintenance":"800.25"}}`)
	f.Add(`{"order_id":"999","msg":"Request was submitted"}`)
	f.Add(`{"orders":[{"orderId":"999","account":"U1234567","conid":265598,"ticker":"AAPL","orderType":"LMT","side":"BUY","status":"PreSubmitted","timeInForce":"DAY","totalSize":"10","filledQuantity":"0","remainingQuantity":"10","price":"150.00","avgPrice":"0"}]}`)
	f.Add(`{"order_id":"999","order_status":"PreSubmitted","conid":265598,"side":"BUY","filled_quantity":"0","remaining_quantity":"10","average_price":"0"}`)
	f.Add(`[{"order_id":"999","execution_id":"exec-1","account":"U1234567","conid":265598,"symbol":"AAPL","side":"B","size":"10","price":"150.00","commission":"1.00","net_amount":"1499.00","trade_time_r":1564652478000}]`)
	f.Add(`{"session":"tok-123"}`)
	f.Add(`{}`)

	f.Fuzz(func(t *testing.T, data string) {
		var v any
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		_ = v
	})
}

func FuzzFixtureByteCorruption(f *testing.F) {
	cases := allFixtureCases()
	for _, c := range cases {
		f.Add(c.name, c.body)
	}

	f.Fuzz(func(t *testing.T, opName, body string) {
		if body == "" {
			return
		}

		data := []byte(body)
		if len(data) == 0 {
			return
		}

		swapPos := len(data) / 2
		if swapPos > 0 && swapPos < len(data) {
			d := make([]byte, len(data))
			copy(d, data)
			d[swapPos], d[swapPos-1] = d[swapPos-1], d[swapPos]
			var v any
			dec := json.NewDecoder(bytes.NewReader(d))
			dec.UseNumber()
			_ = dec.Decode(&v)
		}

		dup := bytes.Repeat(data, 2)
		var v2 any
		dec2 := json.NewDecoder(bytes.NewReader(dup))
		dec2.UseNumber()
		_ = dec2.Decode(&v2)

		if len(data) > 1 {
			half := len(data) / 2
			var v3 any
			dec3 := json.NewDecoder(bytes.NewReader(data[:half]))
			dec3.UseNumber()
			_ = dec3.Decode(&v3)
		}

		var v4 any
		dec4 := json.NewDecoder(bytes.NewReader(data))
		dec4.UseNumber()
		_ = dec4.Decode(&v4)
	})
}

func TestAllFixturesDecode(t *testing.T) {
	cases := allFixtureCases()
	t.Logf("testing %d fixture ops", len(cases))

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.body == "" {
				t.Skip("empty body")
			}
			var v any
			dec := json.NewDecoder(bytes.NewReader([]byte(c.body)))
			dec.UseNumber()
			err := dec.Decode(&v)
			if err != nil {
				t.Errorf("failed to decode fixture %q: %v", c.name, err)
			}
		})
	}
}

func TestFixtureCount(t *testing.T) {
	cases := allFixtureCases()
	t.Logf("total fixture cases: %d", len(cases))

	if len(cases) == 0 {
		t.Error("no fixture cases found")
	}
}
