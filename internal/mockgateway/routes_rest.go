// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package mockgateway

import "net/http"

// IB REST operation IDs (the canonical opIds from docs/SPEC.md). These are the
// 70 IB REST rows, including the OAuth2 token endpoint (generateToken). The two
// SPEC v2 rows keep their "_2" suffix.
const (
	// Account Management Accounts.
	OpListAccounts              = "listAccounts"
	OpUpdateAccounts            = "updateAccounts"
	OpCreateAccounts            = "createAccounts"
	OpCreateAccountsDocuments   = "createAccountsDocuments"
	OpListAccountsLoginMessages = "listAccountsLoginMessages"
	OpListAccountsStatus        = "listAccountsStatus"
	OpGetAccountsDetails        = "getAccountsDetails"
	OpGetAccountsKyc            = "getAccountsKyc"
	OpGetAccountsLoginMessages  = "getAccountsLoginMessages"
	OpGetAccountsStatus         = "getAccountsStatus"
	OpUpdateAccountsStatus      = "updateAccountsStatus"
	OpGetAccountsTasks          = "getAccountsTasks"
	OpUpdateAccountsTasks       = "updateAccountsTasks"
	OpCreateAccountsTasks       = "createAccountsTasks"

	// Account Management Banking.
	OpCreateBankInstructions           = "createBankInstructions"
	OpCreateBankInstructionsQuery      = "createBankInstructionsQuery"
	OpBulkBankInstructions             = "bulkBankInstructions"
	OpGetClientInstructions            = "getClientInstructions"
	OpCreateExternalAssetTransfers     = "createExternalAssetTransfers"
	OpBulkExternalAssetTransfers       = "bulkExternalAssetTransfers"
	OpCreateExternalCashTransfers      = "createExternalCashTransfers"
	OpCreateExternalCashTransfersQuery = "createExternalCashTransfersQuery"
	OpBulkExternalCashTransfers        = "bulkExternalCashTransfers"
	OpGetInstructionSets               = "getInstructionSets"
	OpCreateInstructionsCancel         = "createInstructionsCancel"
	OpBulkInstructionsCancel           = "bulkInstructionsCancel"
	OpCreateInstructionsQuery          = "createInstructionsQuery"
	OpGetInstructions                  = "getInstructions"
	OpCreateInternalAssetTransfers     = "createInternalAssetTransfers"
	OpBulkInternalAssetTransfers       = "bulkInternalAssetTransfers"
	OpCreateInternalCashTransfers      = "createInternalCashTransfers"
	OpBulkInternalCashTransfers        = "bulkInternalCashTransfers"
	OpCreateExternalAssetTransfers2    = "createExternalAssetTransfers_2"
	OpBulkExternalAssetTransfers2      = "bulkExternalAssetTransfers_2"

	// Account Management Reports.
	OpCreateStatements                = "createStatements"
	OpListStatementsAvailable         = "listStatementsAvailable"
	OpCreateTaxDocuments              = "createTaxDocuments"
	OpListTaxDocumentsAvailable       = "listTaxDocumentsAvailable"
	OpCreateTradeConfirmations        = "createTradeConfirmations"
	OpListTradeConfirmationsAvailable = "listTradeConfirmationsAvailable"

	// Account Management Utilities.
	OpListEnumerationsComplexAssetTransfer = "listEnumerationsComplexAssetTransfer"
	OpGetEnumerations                      = "getEnumerations"
	OpListForms                            = "listForms"
	OpListFormsRequiredForms               = "listFormsRequiredForms"
	OpListParticipatingBanks               = "listParticipatingBanks"
	OpListRequests                         = "listRequests"
	OpGetRequestsStatus                    = "getRequestsStatus"
	OpUpdateRequestsStatus                 = "updateRequestsStatus"
	OpGetValidationsUsernames              = "getValidationsUsernames"

	// Authorization SSO Browser Sessions / SSO Sessions / Token.
	OpCreateSsoBrowserSessions = "createSsoBrowserSessions"
	OpCreateSsoSessions        = "createSsoSessions"
	OpGenerateToken            = "generateToken"

	// PreTrade Compliance Restrictions.
	OpApplyCSV                = "applyCSV"
	OpGetAccountRestrictions  = "getAccountRestrictions"
	OpGetMasterRestrictionIds = "getMasterRestrictionIds"
	OpGetListDetails          = "getListDetails"
	OpGetMasterListIds        = "getMasterListIds"
	OpGetRestrictionDetails   = "getRestrictionDetails"
	OpGetRestrictionScope     = "getRestrictionScope"
	OpGetUserRestrictions     = "getUserRestrictions"
	OpVerifyCSV               = "verifyCSV"

	// Third Parties Cash Balances.
	OpCreateBalancesQuery = "createBalancesQuery"

	// Third Parties Tax Vouchers.
	OpCreateTaxVoucherRequests = "createTaxVoucherRequests"
	OpGetActiveCountryList     = "getActiveCountryList"
	OpFetchDividends1          = "fetchDividends_1"
	OpGetYears                 = "getYears"
	OpDownloadFile             = "downloadFile"
	OpGetCurrentState1         = "getCurrentState_1"

	// Utilities Echo.
	OpListEchoHttps       = "listEchoHttps"
	OpCreateEchoSignedJwt = "createEchoSignedJwt"
)

// restRoutes returns the 70 IB REST routes from docs/SPEC.md. Every route
// except the OAuth2 token endpoint is bearer-protected; the token endpoint is
// the credential entry point and is therefore unauthenticated.
func restRoutes() []route {
	get := []string{http.MethodGet}
	post := []string{http.MethodPost}
	patch := []string{http.MethodPatch}

	return []route{
		// --- account management accounts ---
		mkBearerRoute(OpListAccounts, get, "/gw/api/v1/accounts"),
		mkBearerRoute(OpUpdateAccounts, patch, "/gw/api/v1/accounts"),
		mkBearerRoute(OpCreateAccounts, post, "/gw/api/v1/accounts"),
		mkBearerRoute(OpCreateAccountsDocuments, post, "/gw/api/v1/accounts/documents"),
		mkBearerRoute(OpListAccountsLoginMessages, get, "/gw/api/v1/accounts/login-messages"),
		mkBearerRoute(OpListAccountsStatus, get, "/gw/api/v1/accounts/status"),
		mkBearerRoute(OpGetAccountsDetails, get, "/gw/api/v1/accounts/{accountId}/details"),
		mkBearerRoute(OpGetAccountsKyc, get, "/gw/api/v1/accounts/{accountId}/kyc"),
		mkBearerRoute(OpGetAccountsLoginMessages, get, "/gw/api/v1/accounts/{accountId}/login-messages"),
		mkBearerRoute(OpGetAccountsStatus, get, "/gw/api/v1/accounts/{accountId}/status"),
		mkBearerRoute(OpUpdateAccountsStatus, patch, "/gw/api/v1/accounts/{accountId}/status"),
		mkBearerRoute(OpGetAccountsTasks, get, "/gw/api/v1/accounts/{accountId}/tasks"),
		mkBearerRoute(OpUpdateAccountsTasks, patch, "/gw/api/v1/accounts/{accountId}/tasks"),
		mkBearerRoute(OpCreateAccountsTasks, post, "/gw/api/v1/accounts/{accountId}/tasks"),

		// --- account management banking ---
		mkBearerRoute(OpCreateBankInstructions, post, "/gw/api/v1/bank-instructions"),
		mkBearerRoute(OpCreateBankInstructionsQuery, post, "/gw/api/v1/bank-instructions/query"),
		mkBearerRoute(OpBulkBankInstructions, post, "/gw/api/v1/bank-instructions:bulk"),
		mkBearerRoute(OpGetClientInstructions, get, "/gw/api/v1/client-instructions/{clientInstructionId}"),
		mkBearerRoute(OpCreateExternalAssetTransfers, post, "/gw/api/v1/external-asset-transfers"),
		mkBearerRoute(OpBulkExternalAssetTransfers, post, "/gw/api/v1/external-asset-transfers:bulk"),
		mkBearerRoute(OpCreateExternalCashTransfers, post, "/gw/api/v1/external-cash-transfers"),
		mkBearerRoute(OpCreateExternalCashTransfersQuery, post, "/gw/api/v1/external-cash-transfers/query"),
		mkBearerRoute(OpBulkExternalCashTransfers, post, "/gw/api/v1/external-cash-transfers:bulk"),
		mkBearerRoute(OpGetInstructionSets, get, "/gw/api/v1/instruction-sets/{instructionSetId}"),
		mkBearerRoute(OpCreateInstructionsCancel, post, "/gw/api/v1/instructions/cancel"),
		mkBearerRoute(OpBulkInstructionsCancel, post, "/gw/api/v1/instructions/cancel:bulk"),
		mkBearerRoute(OpCreateInstructionsQuery, post, "/gw/api/v1/instructions/query"),
		mkBearerRoute(OpGetInstructions, get, "/gw/api/v1/instructions/{instructionId}"),
		mkBearerRoute(OpCreateInternalAssetTransfers, post, "/gw/api/v1/internal-asset-transfers"),
		mkBearerRoute(OpBulkInternalAssetTransfers, post, "/gw/api/v1/internal-asset-transfers:bulk"),
		mkBearerRoute(OpCreateInternalCashTransfers, post, "/gw/api/v1/internal-cash-transfers"),
		mkBearerRoute(OpBulkInternalCashTransfers, post, "/gw/api/v1/internal-cash-transfers:bulk"),
		mkBearerRoute(OpCreateExternalAssetTransfers2, post, "/gw/api/v2/external-asset-transfers"),
		mkBearerRoute(OpBulkExternalAssetTransfers2, post, "/gw/api/v2/external-asset-transfers:bulk"),

		// --- account management reports ---
		mkBearerRoute(OpCreateStatements, post, "/gw/api/v1/statements"),
		mkBearerRoute(OpListStatementsAvailable, get, "/gw/api/v1/statements/available"),
		mkBearerRoute(OpCreateTaxDocuments, post, "/gw/api/v1/tax-documents"),
		mkBearerRoute(OpListTaxDocumentsAvailable, get, "/gw/api/v1/tax-documents/available"),
		mkBearerRoute(OpCreateTradeConfirmations, post, "/gw/api/v1/trade-confirmations"),
		mkBearerRoute(OpListTradeConfirmationsAvailable, get, "/gw/api/v1/trade-confirmations/available"),

		// --- account management utilities ---
		mkBearerRoute(OpListEnumerationsComplexAssetTransfer, get, "/gw/api/v1/enumerations/complex-asset-transfer"),
		mkBearerRoute(OpGetEnumerations, get, "/gw/api/v1/enumerations/{enumerationType}"),
		mkBearerRoute(OpListForms, get, "/gw/api/v1/forms"),
		mkBearerRoute(OpListFormsRequiredForms, get, "/gw/api/v1/forms/required-forms"),
		mkBearerRoute(OpListParticipatingBanks, get, "/gw/api/v1/participating-banks"),
		mkBearerRoute(OpListRequests, get, "/gw/api/v1/requests"),
		mkBearerRoute(OpGetRequestsStatus, get, "/gw/api/v1/requests/{requestId}/status"),
		mkBearerRoute(OpUpdateRequestsStatus, patch, "/gw/api/v1/requests/{requestId}/status"),
		mkBearerRoute(OpGetValidationsUsernames, get, "/gw/api/v1/validations/usernames/{username}"),

		// --- authorization ---
		mkBearerRoute(OpCreateSsoBrowserSessions, post, "/gw/api/v1/sso-browser-sessions"),
		mkBearerRoute(OpCreateSsoSessions, post, "/gw/api/v1/sso-sessions"),
		mkRoute(OpGenerateToken, post, "/oauth2/api/v1/token", false),

		// --- pre-trade compliance restrictions ---
		mkBearerRoute(OpApplyCSV, post, "/gw/api/v1/restrictions"),
		mkBearerRoute(OpGetAccountRestrictions, get, "/gw/api/v1/restrictions/account"),
		mkBearerRoute(OpGetMasterRestrictionIds, get, "/gw/api/v1/restrictions/ids"),
		mkBearerRoute(OpGetListDetails, get, "/gw/api/v1/restrictions/list"),
		mkBearerRoute(OpGetMasterListIds, get, "/gw/api/v1/restrictions/lists/ids"),
		mkBearerRoute(OpGetRestrictionDetails, get, "/gw/api/v1/restrictions/restriction"),
		mkBearerRoute(OpGetRestrictionScope, get, "/gw/api/v1/restrictions/restriction-scope"),
		mkBearerRoute(OpGetUserRestrictions, get, "/gw/api/v1/restrictions/user"),
		mkBearerRoute(OpVerifyCSV, post, "/gw/api/v1/restrictions/verify"),

		// --- third parties cash balances ---
		mkBearerRoute(OpCreateBalancesQuery, post, "/gw/api/v1/balances/query"),

		// --- third parties tax vouchers ---
		mkBearerRoute(OpCreateTaxVoucherRequests, post, "/gw/api/v1/tax-vouchers"),
		mkBearerRoute(OpGetActiveCountryList, get, "/gw/api/v1/tax-vouchers/active-countries"),
		mkBearerRoute(OpFetchDividends1, get, "/gw/api/v1/tax-vouchers/dividends"),
		mkBearerRoute(OpGetYears, get, "/gw/api/v1/tax-vouchers/years"),
		mkBearerRoute(OpDownloadFile, get, "/gw/api/v1/tax-vouchers/{requestId}/download"),
		mkBearerRoute(OpGetCurrentState1, get, "/gw/api/v1/tax-vouchers/{requestId}/state"),

		// --- utilities echo ---
		mkBearerRoute(OpListEchoHttps, get, "/gw/api/v1/echo/https"),
		mkBearerRoute(OpCreateEchoSignedJwt, post, "/gw/api/v1/echo/signed-jwt"),
	}
}

// registerRESTFixtures adds the T3 IB REST fixtures to f. Amounts and
// quantities are JSON strings wherever the wrapper decodes them as strings
// (ADR 0008); the generated REST models that require numbers use numbers.
func registerRESTFixtures(f *Fixtures) {
	// Single-instruction acknowledgements (HTTP 202).
	single := Fixture{
		Status: http.StatusAccepted,
		Body:   `{"instructionSetId":1001,"status":202,"instructionResult":{"clientInstructionId":1,"instructionId":2001,"instructionStatus":"PENDING","instructionType":"FOP","ibReferenceId":9001,"description":"Please poll for status after 10 minutes"}}`,
	}
	bulk := Fixture{
		Status: http.StatusAccepted,
		Body:   `{"instructionSetId":1001,"status":207,"instructionResults":[{"instructionResult":{"clientInstructionId":1,"instructionId":2001,"instructionStatus":"PENDING","instructionType":"FOP","ibReferenceId":9001,"description":"Please poll for status after 10 minutes"},"instructionSetId":1001,"status":202}]}`,
	}

	// --- account management accounts ---
	f.Set(OpListAccounts, Fixture{Body: `[{"id":"U1234567","accountAlias":"Main","baseCurrency":"USD","accountTitle":"Main Account","accountType":"INDIVIDUAL","masterAccount":"U1234567"}]`})
	f.Set(OpUpdateAccounts, Fixture{Body: `{}`})
	f.Set(OpCreateAccounts, Fixture{Body: `{}`})
	f.Set(OpCreateAccountsDocuments, Fixture{Body: `{}`})
	f.Set(OpListAccountsLoginMessages, Fixture{Body: `{"accountId":"U1234567","loginMessagePresent":true,"loginMessages":[{"id":1,"description":"Welcome","messageType":"INFO","state":"UNREAD","recordDate":"2026-01-02T15:04:05Z","username":"jdoe","contentId":1,"tasks":[1]}]}`})
	f.Set(OpListAccountsStatus, Fixture{Body: `{"accounts":[{"accountId":"U1234567","adminAccountId":"U1234567","dateOpened":"2020-01-02T15:04:05Z","dateStarted":"2020-01-02T15:04:05Z","description":"Individual account","masterAccountId":"U1234567","message":"ok","state":"ACTIVE","status":"ACTIVE"}],"total":1,"offset":0,"limit":10}`})
	f.Set(OpGetAccountsDetails, Fixture{Body: `{"accountId":"U1234567","accountAlias":"Main","accountTitle":"Main Account","baseCurrency":"USD","household":"HH-1","applicantType":"INDIVIDUAL","orgType":"INDIVIDUAL"}`})
	f.Set(OpGetAccountsKyc, Fixture{Body: `{"entityId":1,"externalId":"https://example.test/kyc/U1234567","hasError":false,"state":"PENDING","startDate":"2026-01-02T15:04:05Z"}`})
	f.Set(OpGetAccountsLoginMessages, Fixture{Body: `{"accountId":"U1234567","loginMessagePresent":true,"loginMessages":[{"id":1,"description":"Welcome","messageType":"INFO","state":"UNREAD","recordDate":"2026-01-02T15:04:05Z","username":"jdoe","contentId":1,"tasks":[1]}]}`})
	f.Set(OpGetAccountsStatus, Fixture{Body: `{"accountId":"U1234567","adminAccountId":"U1234567","dateOpened":"2020-01-02T15:04:05Z","dateStarted":"2020-01-02T15:04:05Z","description":"Individual account","masterAccountId":"U1234567","message":"ok","state":"ACTIVE","status":"ACTIVE"}`})
	f.Set(OpUpdateAccountsStatus, Fixture{Body: `{}`})
	f.Set(OpGetAccountsTasks, Fixture{Body: `{"accountId":"U1234567","registrationTasks":[{"taskId":"t1","action":"REVIEW","dateCompleted":"2026-01-02T15:04:05Z","externalId":"ext-1","formName":"Form 1","formNumber":1,"isCompleted":true,"isDeclined":false,"isRequiredForApproval":true,"questionIds":[1,2],"state":"ACTIVE","warning":"none"}],"state":"ACTIVE","status":"ok"}`})
	f.Set(OpUpdateAccountsTasks, Fixture{Body: `[{"accountId":"U1234567","taskAction":[{"formNumber":1,"message":"ok","status":"COMPLETED"}]}]`})
	f.Set(OpCreateAccountsTasks, Fixture{Body: `[{"accountId":"U1234567","taskAction":[{"formNumber":1,"message":"assigned","status":"PENDING"}]}]`})

	// --- account management banking ---
	f.Set(OpCreateBankInstructions, single)
	f.Set(OpCreateBankInstructionsQuery, Fixture{Status: http.StatusCreated, Body: `{"accountID":"U1234567","bankInstructionName":"Main Bank","bankInstructionMethod":"ACH","status":"ACTIVE"}`})
	f.Set(OpBulkBankInstructions, bulk)
	f.Set(OpGetClientInstructions, Fixture{Body: `{"instructionId":2001,"accountId":"U1234567","instructionType":"FOP","status":"PENDING","createdAt":"2026-01-02T15:04:05Z","instructions":[{"instructionId":2001,"type":"FOP","status":"PENDING","createdAt":"2026-01-02T15:04:05Z"}]}`})
	f.Set(OpCreateExternalAssetTransfers, single)
	f.Set(OpBulkExternalAssetTransfers, bulk)
	f.Set(OpCreateExternalCashTransfers, single)
	f.Set(OpCreateExternalCashTransfersQuery, Fixture{Body: `{"cashBalance":12345.67,"currency":"USD"}`})
	f.Set(OpBulkExternalCashTransfers, bulk)
	f.Set(OpGetInstructionSets, Fixture{Body: `{"instructionSetId":3001,"accountId":"U1234567","instructions":[{"instructionId":2001,"type":"FOP","status":"PENDING","createdAt":"2026-01-02T15:04:05Z"}]}`})
	f.Set(OpCreateInstructionsCancel, Fixture{Status: http.StatusCreated, Body: `{"instructionSetId":4001,"status":201}`})
	f.Set(OpBulkInstructionsCancel, Fixture{Body: `{"instructionSetId":4001,"status":202,"instructionResults":[]}`})
	f.Set(OpCreateInstructionsQuery, Fixture{Status: http.StatusAccepted, Body: `{"transactions":[{"id":"tx-1","date":"2026-01-02","type":"DEPOSIT","amount":"1000.00","currency":"USD","status":"COMPLETED","description":"Synthetic deposit"}]}`})
	f.Set(OpGetInstructions, Fixture{Body: `{"instructionId":2001,"type":"FOP","status":"PENDING","createdAt":"2026-01-02T15:04:05Z"}`})
	f.Set(OpCreateInternalAssetTransfers, single)
	f.Set(OpBulkInternalAssetTransfers, bulk)
	f.Set(OpCreateInternalCashTransfers, single)
	f.Set(OpBulkInternalCashTransfers, bulk)
	f.Set(OpCreateExternalAssetTransfers2, single)
	f.Set(OpBulkExternalAssetTransfers2, bulk)

	// --- account management reports ---
	document := `{"data":{"value":"JVBERi0xLjQK","mimeType":"application/pdf","encoding":"BASE64","gzip":false}}`
	f.Set(OpCreateStatements, Fixture{Body: document})
	f.Set(OpListStatementsAvailable, Fixture{Body: `{"data":{"value":{"annual":["2025"],"monthly":["2026-01"],"daily":{"startDate":"2026-01-01","endDate":"2026-01-31"}}}}`})
	f.Set(OpCreateTaxDocuments, Fixture{Body: document})
	f.Set(OpListTaxDocumentsAvailable, Fixture{Body: `{"data":{"value":{"forms":[{"taxFormName":"1099"},{"taxFormName":"1042S"}]}}}`})
	f.Set(OpCreateTradeConfirmations, Fixture{Body: document})
	f.Set(OpListTradeConfirmationsAvailable, Fixture{Body: `{"data":{"value":["2026-01-02","2026-01-03"]}}`})

	// --- account management utilities ---
	f.Set(OpListEnumerationsComplexAssetTransfer, Fixture{Body: `{"brokers":["BROKER_A","BROKER_B"],"instructionType":"COMPLEX_ASSET_TRANSFER"}`})
	f.Set(OpGetEnumerations, Fixture{Body: `{"enumerationsType":"assetClass","jsonData":null}`})
	f.Set(OpListForms, Fixture{Body: `{"forms":[{"formNo":1,"name":"Form 1","content":"synthetic"}]}`})
	f.Set(OpListFormsRequiredForms, Fixture{Body: `{"forms":["Form 1"],"hasError":false}`})
	f.Set(OpListParticipatingBanks, Fixture{Body: `{"banks":[{"id":"bank-1","name":"Example Bank"}]}`})
	f.Set(OpListRequests, Fixture{Body: `{"limit":50,"offset":0,"requestDetails":[{"accountID":"U1234567","dateSubmitted":"2026-01-02","requestId":5001,"requestType":"ACCOUNT_UPDATE","status":"COMPLETED"}],"total":1}`})
	f.Set(OpGetRequestsStatus, Fixture{Body: `{"requestId":5001,"status":"COMPLETED","executedAt":"2026-01-02T15:04:05Z"}`})
	f.Set(OpUpdateRequestsStatus, Fixture{Body: `{"requestId":"5001","status":"COMPLETED"}`})
	f.Set(OpGetValidationsUsernames, Fixture{Body: `{"available":true}`})

	// --- authorization ---
	f.Set(OpCreateSsoBrowserSessions, Fixture{Body: `{"active":true,"url":"https://example.test/sso/browser"}`})
	f.Set(OpCreateSsoSessions, Fixture{Body: `{"accessToken":"sso-access-token","active":true,"tokenType":"Bearer"}`})
	f.Set(OpGenerateToken, Fixture{Body: `{"access_token":"mock-access","token_type":"Bearer","expires_in":3600,"refresh_token":"mock-refresh"}`})

	// --- pre-trade compliance restrictions ---
	f.Set(OpApplyCSV, Fixture{Body: `{"success":true,"requestId":6001,"message":"OK"}`})
	f.Set(OpGetAccountRestrictions, Fixture{Body: `{"ids":[1001,1002]}`})
	f.Set(OpGetMasterRestrictionIds, Fixture{Body: `{"masterUserName":"jdoe","restrictions":[{"restrictionId":1001,"byOperator":false},{"restrictionId":1002,"byOperator":true}],"status":"ok"}`})
	f.Set(OpGetListDetails, Fixture{Body: `{"listId":2001,"name":"My Issuer List","description":"Synthetic list","type":"ISSUERORCONID","entries":[{"id":300001,"createdAt":1700000000000,"type":"ISSUER"}],"status":"ok"}`})
	f.Set(OpGetMasterListIds, Fixture{Body: `{"masterUserName":"jdoe","lists":[{"listId":2001,"byOperator":false}],"status":"ok"}`})
	f.Set(OpGetRestrictionDetails, Fixture{Body: `{"restrictionId":1001,"name":"My Restriction","description":"Synthetic restriction","applicationType":"Restrict","usageType":"Trading","isWhiteList":"F","allowOnly":"F","matchAll":"F","message":"restricted","rules":[{"ruleId":1,"type":"CONID","validityType":"GTD","startDate":"01/01/2026","endDate":"12/31/2026"}],"tmFirstDate":1700000000000,"tmFrequency":"Daily","status":"ok"}`})
	f.Set(OpGetRestrictionScope, Fixture{Body: `{"restrictionId":1001,"scope":"Active For Some","accountIds":["U1234567"],"truncated":false,"status":"ok"}`})
	f.Set(OpGetUserRestrictions, Fixture{Body: `{"ids":[1001]}`})
	f.Set(OpVerifyCSV, Fixture{Body: `{"success":true,"requestId":6002,"message":"OK"}`})

	// --- third parties cash balances ---
	f.Set(OpCreateBalancesQuery, Fixture{Status: http.StatusCreated, Body: `{"balanaces":[{"accountId":"U1234567","currency":"USD","amount":"12345.67"}]}`})

	// --- third parties tax vouchers ---
	f.Set(OpCreateTaxVoucherRequests, Fixture{Body: `[{"corpactionId":"ca-1","countryCode":"US","custAcctId":"U1234567","requestId":"tv-1","requestState":"COMPLETED","year":2025}]`})
	f.Set(OpGetActiveCountryList, Fixture{Body: `[{"country":"United States","countryCode":"US","currency":"USD"},{"country":"United Kingdom","countryCode":"GB","currency":"GBP"}]`})
	f.Set(OpFetchDividends1, Fixture{Body: `[{"corpactionId":"ca-1","country":"US","currency":"USD","exDate":"2026-01-02","isin":"US0378331005","payDate":"2026-01-15","securityDesc":"Apple Inc","symbol":"AAPL"}]`})
	f.Set(OpGetYears, Fixture{Body: `[2025,2024]`})
	f.Set(OpDownloadFile, Fixture{Body: `"c3ludGhldGljLXRheC12b3VjaGVy"`})
	f.Set(OpGetCurrentState1, Fixture{Body: `{"requestId":"tv-1","requestState":"COMPLETED"}`})

	// --- utilities echo ---
	f.Set(OpListEchoHttps, Fixture{Body: `{"requestMethod":"GET","securityPolicy":"HTTPS"}`})
	f.Set(OpCreateEchoSignedJwt, Fixture{Body: `{"requestMethod":"POST","securityPolicy":"SIGNED_JWT"}`})
}
