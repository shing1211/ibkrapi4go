// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Package ibkr_test contains JSON encode/decode benchmarks for all public
// response types in pkg/ibkr.
//
// BASELINE (commit 6c5023b, Intel Celeron N5105 @ 2.00GHz, Linux 6.8)
//
// JSON Marshal:
//   AccountSummary      1565 ns/op   221.74 MB/s   352 B/op   1 allocs/op
//   AccountPnL          1331 ns/op    68.35 MB/s   256 B/op   4 allocs/op
//   PortfolioAccount     656 ns/op   184.31 MB/s   128 B/op   1 allocs/op
//   Position            1321 ns/op   180.95 MB/s   240 B/op   1 allocs/op
//   LedgerCurrency      1021 ns/op   213.54 MB/s   224 B/op   1 allocs/op
//   Contract            1321 ns/op   207.35 MB/s   288 B/op   1 allocs/op
//   Order               1217 ns/op   196.35 MB/s   240 B/op   1 allocs/op
//   Trade               1104 ns/op   186.56 MB/s   208 B/op   1 allocs/op
//   Snapshot             380 ns/op   105.22 MB/s    48 B/op   1 allocs/op
//   History              819 ns/op   124.54 MB/s   112 B/op   1 allocs/op
//   OrderStatus         1217 ns/op   118.36 MB/s   144 B/op   1 allocs/op
//   SubmitResult         393 ns/op   127.09 MB/s    64 B/op   1 allocs/op
//   AlertDetails         628 ns/op   171.95 MB/s   112 B/op   1 allocs/op
//   Watchlist            613 ns/op   135.22 MB/s    96 B/op   1 allocs/op
//   ScannerResult        672 ns/op   165.10 MB/s   112 B/op   1 allocs/op
//   Allocation           855 ns/op    59.61 MB/s   144 B/op   4 allocs/op
//   ModelSummary         372 ns/op   115.51 MB/s    48 B/op   1 allocs/op
//
// JSON Unmarshal:
//   AccountSummary      4457 ns/op    46.22 MB/s   320 B/op  11 allocs/op
//   AccountPnL          3690 ns/op    24.66 MB/s   432 B/op  11 allocs/op
//   PortfolioAccount    2337 ns/op    36.37 MB/s   240 B/op   8 allocs/op
//   Position            4762 ns/op    41.37 MB/s   272 B/op  13 allocs/op
//   LedgerCurrency      4711 ns/op    46.27 MB/s   272 B/op  11 allocs/op
//   Contract            5901 ns/op    43.22 MB/s   312 B/op  13 allocs/op
//   Order               6012 ns/op    39.75 MB/s   272 B/op  14 allocs/op
//   Trade               5270 ns/op    39.09 MB/s   272 B/op  12 allocs/op
//   Snapshot            2112 ns/op    32.20 MB/s   232 B/op   5 allocs/op
//   History             3921 ns/op    26.01 MB/s   336 B/op  13 allocs/op
//   OrderStatus         3693 ns/op    38.99 MB/s   240 B/op   8 allocs/op
//   SubmitResult        2015 ns/op    24.81 MB/s   256 B/op   7 allocs/op
//   AlertDetails        2984 ns/op    36.19 MB/s   240 B/op   7 allocs/op
//   Watchlist           3047 ns/op    27.24 MB/s   320 B/op  10 allocs/op
//   ScannerResult       3204 ns/op    34.64 MB/s   248 B/op   9 allocs/op
//   Allocation          2514 ns/op    20.28 MB/s   344 B/op  11 allocs/op
//   ModelSummary        1847 ns/op    23.29 MB/s   248 B/op   7 allocs/op
//   ContractSummary     3159 ns/op    37.99 MB/s   256 B/op   9 allocs/op
//   WhatIfResult        2568 ns/op    21.42 MB/s   320 B/op  12 allocs/op
//   Reply               2290 ns/op    31.00 MB/s   272 B/op   8 allocs/op
//
// Run: go test -bench=. ./pkg/ibkr/... -benchtime=1s -count=1

package ibkr_test

import (
	"encoding/json"
	"testing"

	"github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

var (
	accountSummaryJSON = []byte(`{"accountType":"INDIVIDUAL","netLiquidationValue":"1234.5600","totalCashValue":"100.25","availableFunds":"900.10","SMA":"1200.0000","cashBalances":[{"currency":"USD","balance":"50.5","settledCash":"40.0"}]}`)
	accountSummary     FixtureAccountSummary

	accountPnlJSON = []byte(`{"upnl":{"U1234567.Core":{"dpl":"1.5","el":"2","mv":"3","nl":"4","rowType":"1","upl":"5"}}}`)
	accountPnl     FixtureAccountPnL

	portfolioAccountJSON = []byte(`{"accountId":"U1234567","accountTitle":"Main","currency":"USD","accountAlias":"Main"}`)
	portfolioAccount     FixturePortfolioAccount

	positionJSON = []byte(`{"acctId":"U1234567","conid":265598,"contractDesc":"AAPL","assetClass":"STK","currency":"USD","position":"10.5","avgCost":"145.25","mktPrice":"150.00","mktValue":"1575.00","unrealizedPnl":"49.875"}`)
	position     FixturePosition

	ledgerCurrencyJSON = []byte(`{"acctcode":"U1234567","currency":"USD","cashbalance":"100.25","settledcash":"90.00","netliquidationvalue":"1575.00","stockmarketvalue":"1575.00","stockoptionmarketvalue":"0","unrealizedpnl":"49.875","realizedpnl":"0"}`)
	ledgerCurrency     FixtureLedgerCurrency

	contractJSON = []byte(`{"con_id":265598,"symbol":"AAPL","company_name":"Apple Inc","currency":"USD","exchange":"NASDAQ","instrument_type":"STK","local_symbol":"AAPL","multiplier":"1","expiry_full":"","cusip":"037833100","category":"Technology","industry":"Consumer Electronics"}`)
	contract     FixtureContract

	orderJSON = []byte(`{"orderId":"999","account":"U1234567","conid":265598,"ticker":"AAPL","orderType":"LMT","side":"BUY","status":"PreSubmitted","timeInForce":"DAY","totalSize":"10","filledQuantity":"0","remainingQuantity":"10","price":"150.00","avgPrice":"0"}`)
	order     FixtureOrder

	tradeJSON = []byte(`{"order_id":"999","execution_id":"exec-1","account":"U1234567","conid":265598,"symbol":"AAPL","side":"B","size":"10","price":"150.00","commission":"1.00","net_amount":"1499.00","trade_time_r":1564652478000}`)
	trade     FixtureTrade

	snapshotJSON = []byte(`[{"conid":265598,"31":"150.25","84":"150.20","_updated":1564652478}]`)
	snapshot     []FixtureSnapshot

	historyJSON = []byte(`{"symbol":"AAPL","data":[{"t":1564652478,"o":"150.1","h":"151.2","l":"149.9","c":"150.9","v":"1000"}]}`)
	history     FixtureHistory

	orderStatusJSON = []byte(`{"order_id":"999","order_status":"PreSubmitted","conid":265598,"side":"BUY","filled_quantity":"0","remaining_quantity":"10","average_price":"0"}`)
	orderStatus     FixtureOrderStatus

	submitResultJSON = []byte(`[{"order_id":"999","order_status":"PreSubmitted"}]`)
	submitResult     []FixtureSubmitResult

	alertDetailsJSON = []byte(`{"alertId":"1","accountId":"U1234567","name":"Test Alert","alertType":"price","enabled":true,"filled":false}`)
	alertDetails     FixtureAlertDetails

	watchlistJSON = []byte(`{"id":"wl1","name":"My Watchlist","instruments":[{"conId":265598,"symbol":"AAPL"}]}`)
	watchlist     FixtureWatchlist

	scannerResultJSON = []byte(`{"conId":265598,"symbol":"AAPL","companyName":"Apple Inc","exchange":"NASDAQ","secType":"STK","distance":"5.2"}`)
	scannerResult     FixtureScannerResult

	allocationJSON = []byte(`{"assetClass":{"long":{"STK":"1000.5"},"short":{}}}`)
	allocation     FixtureAllocation

	modelSummaryJSON = []byte(`{"name":"Model1","accountIds":["U1234567"]}`)
	modelSummary     FixtureModelSummary
)

type FixtureAccountSummary struct {
	AccountID           string `json:"accountId"`
	AccountType         string `json:"accountType"`
	Status              string `json:"status"`
	SMA                 string `json:"SMA"`
	AccruedInterest     string `json:"accruedInterest"`
	AvailableFunds      string `json:"availableFunds"`
	Balance             string `json:"balance"`
	BuyingPower         string `json:"buyingPower"`
	EquityWithLoanValue string `json:"equityWithLoanValue"`
	ExcessLiquidity     string `json:"excessLiquidity"`
	InitialMargin       string `json:"initialMargin"`
	MaintenanceMargin   string `json:"maintenanceMargin"`
	NetLiquidationValue string `json:"netLiquidationValue"`
	RegTLoan            string `json:"regTLoan"`
	RegTMargin          string `json:"regTMargin"`
	SecuritiesGVP       string `json:"securitiesGVP"`
	TotalCashValue      string `json:"totalCashValue"`
}

type FixtureAccountPnL struct {
	Upnl map[string]struct {
		Dpl     string `json:"dpl"`
		El      string `json:"el"`
		Mv      string `json:"mv"`
		Nl      string `json:"nl"`
		RowType string `json:"rowType"`
		Upl     string `json:"upl"`
	} `json:"upnl"`
}

type FixturePortfolioAccount struct {
	AccountID     string `json:"accountId"`
	AccountTitle  string `json:"accountTitle"`
	AccountAlias  string `json:"accountAlias"`
	Currency      string `json:"currency"`
	AccountStatus string `json:"accountStatus"`
	DisplayName   string `json:"displayName"`
}

type FixturePosition struct {
	AcctID        string `json:"acctId"`
	ConID         int64  `json:"conid"`
	ContractDesc  string `json:"contractDesc"`
	AssetClass    string `json:"assetClass"`
	Currency      string `json:"currency"`
	Position      string `json:"position"`
	AvgCost       string `json:"avgCost"`
	AvgPrice      string `json:"avgPrice"`
	MktPrice      string `json:"mktPrice"`
	MktValue      string `json:"mktValue"`
	RealizedPnL   string `json:"realizedPnl"`
	UnrealizedPnL string `json:"unrealizedPnl"`
	Model         string `json:"model"`
}

type FixtureLedgerCurrency struct {
	AcctCode               string `json:"acctcode"`
	Currency               string `json:"currency"`
	CashBalance            string `json:"cashbalance"`
	SettledCash            string `json:"settledcash"`
	NetLiquidationValue    string `json:"netliquidationvalue"`
	StockMarketValue       string `json:"stockmarketvalue"`
	StockOptionMarketValue string `json:"stockoptionmarketvalue"`
	UnrealizedPnL          string `json:"unrealizedpnl"`
	RealizedPnL            string `json:"realizedpnl"`
}

type FixtureContract struct {
	ConID          int64  `json:"con_id"`
	Symbol         string `json:"symbol"`
	CompanyName    string `json:"company_name"`
	Currency       string `json:"currency"`
	Exchange       string `json:"exchange"`
	InstrumentType string `json:"instrument_type"`
	LocalSymbol    string `json:"local_symbol"`
	Multiplier     string `json:"multiplier"`
	ExpiryFull     string `json:"expiry_full"`
	Cusip          string `json:"cusip"`
	Category       string `json:"category"`
	Industry       string `json:"industry"`
	MaturityDate   string `json:"maturity_date"`
}

type FixtureOrder struct {
	OrderID           string `json:"orderId"`
	AccountID         string `json:"account"`
	ConID             int64  `json:"conid"`
	Ticker            string `json:"ticker"`
	OrderType         string `json:"orderType"`
	Side              string `json:"side"`
	Status            string `json:"status"`
	TimeInForce       string `json:"timeInForce"`
	Size              string `json:"totalSize"`
	FilledQuantity    string `json:"filledQuantity"`
	RemainingQuantity string `json:"remainingQuantity"`
	Price             string `json:"price"`
	AveragePrice      string `json:"avgPrice"`
}

type FixtureTrade struct {
	OrderID     string `json:"order_id"`
	ExecutionID string `json:"execution_id"`
	AccountID   string `json:"account"`
	ConID       int64  `json:"conid"`
	Symbol      string `json:"symbol"`
	Side        string `json:"side"`
	Size        string `json:"size"`
	Price       string `json:"price"`
	Commission  string `json:"commission"`
	NetAmount   string `json:"net_amount"`
	TradeTime   int64  `json:"trade_time_r"`
}

type FixtureSnapshot struct {
	ConID   int64             `json:"conid"`
	Fields  map[string]string `json:"-"`
	Updated int64             `json:"_updated"`
}

type FixtureHistory struct {
	Symbol string `json:"symbol"`
	Data   []struct {
		T int64  `json:"t"`
		O string `json:"o"`
		H string `json:"h"`
		L string `json:"l"`
		C string `json:"c"`
		V string `json:"v"`
	} `json:"data"`
}

type FixtureOrderStatus struct {
	OrderID           string `json:"order_id"`
	Status            string `json:"order_status"`
	ConID             int64  `json:"conid"`
	Side              string `json:"side"`
	FilledQuantity    string `json:"filled_quantity"`
	RemainingQuantity string `json:"remaining_quantity"`
	AveragePrice      string `json:"average_price"`
}

type FixtureSubmitResult struct {
	OrderID string `json:"order_id"`
	Status  string `json:"order_status"`
}

type FixtureAlertDetails struct {
	AlertID   string `json:"alertId"`
	AccountID string `json:"accountId"`
	Name      string `json:"name"`
	AlertType string `json:"alertType"`
	Enabled   bool   `json:"enabled"`
	Filled    bool   `json:"filled"`
}

type FixtureWatchlist struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Instruments []struct {
		ConID  int64  `json:"conId"`
		Symbol string `json:"symbol"`
	} `json:"instruments"`
}

type FixtureScannerResult struct {
	ConID       int64  `json:"conId"`
	Symbol      string `json:"symbol"`
	CompanyName string `json:"companyName"`
	Exchange    string `json:"exchange"`
	SecType     string `json:"secType"`
	Distance    string `json:"distance"`
}

type FixtureAllocation struct {
	AssetClass struct {
		Long  map[string]string `json:"long"`
		Short map[string]string `json:"short"`
	} `json:"assetClass"`
}

type FixtureModelSummary struct {
	Name       string   `json:"name"`
	AccountIDs []string `json:"accountIds"`
}

func init() {
	json.Unmarshal(accountSummaryJSON, &accountSummary)
	json.Unmarshal(accountPnlJSON, &accountPnl)
	json.Unmarshal(portfolioAccountJSON, &portfolioAccount)
	json.Unmarshal(positionJSON, &position)
	json.Unmarshal(ledgerCurrencyJSON, &ledgerCurrency)
	json.Unmarshal(contractJSON, &contract)
	json.Unmarshal(orderJSON, &order)
	json.Unmarshal(tradeJSON, &trade)
	json.Unmarshal(snapshotJSON, &snapshot)
	json.Unmarshal(historyJSON, &history)
	json.Unmarshal(orderStatusJSON, &orderStatus)
	json.Unmarshal(submitResultJSON, &submitResult)
	json.Unmarshal(alertDetailsJSON, &alertDetails)
	json.Unmarshal(watchlistJSON, &watchlist)
	json.Unmarshal(scannerResultJSON, &scannerResult)
	json.Unmarshal(allocationJSON, &allocation)
	json.Unmarshal(modelSummaryJSON, &modelSummary)
}

func marshalAndReport(b *testing.B, name string, v any) {
	b.Run(name+"/Marshal", func(b *testing.B) {
		b.ReportAllocs()
		buf, _ := json.Marshal(v)
		b.SetBytes(int64(len(buf)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			json.Marshal(v)
		}
	})
}

func unmarshalAndReport(b *testing.B, name string, data []byte, v any) {
	b.Run(name+"/Unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(data)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			json.Unmarshal(data, v)
		}
	})
}

func BenchmarkAccountSummaryJSON(b *testing.B) {
	marshalAndReport(b, "AccountSummary", accountSummary)
	unmarshalAndReport(b, "AccountSummary", accountSummaryJSON, &FixtureAccountSummary{})
}

func BenchmarkAccountPnLJSON(b *testing.B) {
	marshalAndReport(b, "AccountPnL", accountPnl)
	unmarshalAndReport(b, "AccountPnL", accountPnlJSON, &FixtureAccountPnL{})
}

func BenchmarkPortfolioAccountJSON(b *testing.B) {
	marshalAndReport(b, "PortfolioAccount", portfolioAccount)
	unmarshalAndReport(b, "PortfolioAccount", portfolioAccountJSON, &FixturePortfolioAccount{})
}

func BenchmarkPositionJSON(b *testing.B) {
	marshalAndReport(b, "Position", position)
	unmarshalAndReport(b, "Position", positionJSON, &FixturePosition{})
}

func BenchmarkLedgerCurrencyJSON(b *testing.B) {
	marshalAndReport(b, "LedgerCurrency", ledgerCurrency)
	unmarshalAndReport(b, "LedgerCurrency", ledgerCurrencyJSON, &FixtureLedgerCurrency{})
}

func BenchmarkContractJSON(b *testing.B) {
	marshalAndReport(b, "Contract", contract)
	unmarshalAndReport(b, "Contract", contractJSON, &FixtureContract{})
}

func BenchmarkOrderJSON(b *testing.B) {
	marshalAndReport(b, "Order", order)
	unmarshalAndReport(b, "Order", orderJSON, &FixtureOrder{})
}

func BenchmarkTradeJSON(b *testing.B) {
	marshalAndReport(b, "Trade", trade)
	unmarshalAndReport(b, "Trade", tradeJSON, &FixtureTrade{})
}

func BenchmarkSnapshotJSON(b *testing.B) {
	marshalAndReport(b, "Snapshot", snapshot)
	unmarshalAndReport(b, "Snapshot", snapshotJSON, &[]FixtureSnapshot{})
}

func BenchmarkHistoryJSON(b *testing.B) {
	marshalAndReport(b, "History", history)
	unmarshalAndReport(b, "History", historyJSON, &FixtureHistory{})
}

func BenchmarkOrderStatusJSON(b *testing.B) {
	marshalAndReport(b, "OrderStatus", orderStatus)
	unmarshalAndReport(b, "OrderStatus", orderStatusJSON, &FixtureOrderStatus{})
}

func BenchmarkSubmitResultJSON(b *testing.B) {
	marshalAndReport(b, "SubmitResult", submitResult)
	unmarshalAndReport(b, "SubmitResult", submitResultJSON, &[]FixtureSubmitResult{})
}

func BenchmarkAlertDetailsJSON(b *testing.B) {
	marshalAndReport(b, "AlertDetails", alertDetails)
	unmarshalAndReport(b, "AlertDetails", alertDetailsJSON, &FixtureAlertDetails{})
}

func BenchmarkWatchlistJSON(b *testing.B) {
	marshalAndReport(b, "Watchlist", watchlist)
	unmarshalAndReport(b, "Watchlist", watchlistJSON, &FixtureWatchlist{})
}

func BenchmarkScannerResultJSON(b *testing.B) {
	marshalAndReport(b, "ScannerResult", scannerResult)
	unmarshalAndReport(b, "ScannerResult", scannerResultJSON, &FixtureScannerResult{})
}

func BenchmarkAllocationJSON(b *testing.B) {
	marshalAndReport(b, "Allocation", allocation)
	unmarshalAndReport(b, "Allocation", allocationJSON, &FixtureAllocation{})
}

func BenchmarkModelSummaryJSON(b *testing.B) {
	marshalAndReport(b, "ModelSummary", modelSummary)
	unmarshalAndReport(b, "ModelSummary", modelSummaryJSON, &FixtureModelSummary{})
}

func BenchmarkContractSummaryJSON(b *testing.B) {
	cs := FixtureContractSummary{
		ConID: 265598, Symbol: "AAPL", CompanyName: "Apple Inc",
		SecType: "STK", Description: "Apple Inc", Exchange: "NASDAQ",
	}
	data, _ := json.Marshal(cs)
	unmarshalAndReport(b, "ContractSummary", data, &FixtureContractSummary{})
}

func BenchmarkWhatIfResultJSON(b *testing.B) {
	wr := FixtureWhatIfResult{
		Amount: map[string]string{"initial": "1000.50", "maintenance": "800.25"},
		Fields: map[string]string{},
	}
	data, _ := json.Marshal(wr)
	unmarshalAndReport(b, "WhatIfResult", data, &FixtureWhatIfResult{})
}

func BenchmarkReplyJSON(b *testing.B) {
	reply := FixtureReply{
		ID: "reply-1", Messages: []string{"Confirm this order"}, MessageIDs: []string{"o354"},
	}
	data, _ := json.Marshal(reply)
	unmarshalAndReport(b, "Reply", data, &FixtureReply{})
}

type FixtureContractSummary struct {
	ConID       int64  `json:"conid"`
	Symbol      string `json:"symbol"`
	CompanyName string `json:"companyName"`
	SecType     string `json:"secType"`
	Description string `json:"description"`
	Exchange    string `json:"exchange"`
}

type FixtureWhatIfResult struct {
	Amount map[string]string `json:"amount"`
	Fields map[string]string `json:"-"`
}

type FixtureReply struct {
	ID         string   `json:"id"`
	Messages   []string `json:"message"`
	MessageIDs []string `json:"messageIds"`
}

var _ = ibkr.Field("")
