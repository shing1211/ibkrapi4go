// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr_test

import (
	"bytes"
	"encoding/json"
	"testing"
)

func FuzzAccountSummaryJSON(f *testing.F) {
	f.Add(`{"accountType":"INDIVIDUAL","netLiquidationValue":"1234.5600","totalCashValue":"100.25","availableFunds":"900.10","SMA":"1200.0000","cashBalances":[{"currency":"USD","balance":"50.5","settledCash":"40.0"}]}`)
	f.Add(`{}`)
	f.Add(`{"accountType":"","netLiquidationValue":"","totalCashValue":"","availableFunds":"","SMA":"","cashBalances":[]}`)
	f.Add(`{"accountType":"INDIVIDUAL","netLiquidationValue":"0","totalCashValue":"-1.5","availableFunds":"9999999.99","SMA":"0","cashBalances":[{"currency":"USD","balance":"0","settledCash":"0"}]}`)
	f.Add(`{"accountType":"INDIVIDUAL","netLiquidationValue":"0.00000001","totalCashValue":"0.00000001","availableFunds":"0.00000001","SMA":"0.00000001","cashBalances":[{"currency":"USD","balance":"0.00000001","settledCash":"0.00000001"}]}`)
	f.Add(`{"accountType":"INDIVIDUAL","netLiquidationValue":"999999999999999.9999","totalCashValue":"999999999999999.9999","availableFunds":"999999999999999.9999","SMA":"999999999999999.9999","cashBalances":[{"currency":"EUR","balance":"123.45","settledCash":"67.89"}]}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			AccountType         string `json:"accountType"`
			NetLiquidationValue string `json:"netLiquidationValue"`
			TotalCashValue      string `json:"totalCashValue"`
			AvailableFunds      string `json:"availableFunds"`
			SMA                 string `json:"SMA"`
			AccruedInterest     string `json:"accruedInterest"`
			Balance             string `json:"balance"`
			BuyingPower         string `json:"buyingPower"`
			CashBalances        []struct {
				Currency    string `json:"currency"`
				Balance     string `json:"balance"`
				SettledCash string `json:"settledCash"`
			} `json:"cashBalances"`
		}
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		_ = v.AccountType
		_ = v.NetLiquidationValue
		_ = v.TotalCashValue
		_ = v.AvailableFunds
		_ = v.SMA
	})
}

func FuzzAccountPnLJSON(f *testing.F) {
	f.Add(`{"upnl":{"U1234567.Core":{"dpl":"1.5","el":"2","mv":"3","nl":"4","rowType":"1","upl":"5"}}}`)
	f.Add(`{}`)
	f.Add(`{"upnl":{}}`)
	f.Add(`{"upnl":{"U1234567.Core":{"dpl":"-1.5","el":"-2","mv":"-3","nl":"-4","rowType":"0","upl":"-5"}}}`)
	f.Add(`{"upnl":{"U1234567.Core":{"dpl":"0","el":"0","mv":"0","nl":"0","rowType":"1","upl":"0"}}}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			Upnl map[string]struct {
				Dpl     string `json:"dpl"`
				El      string `json:"el"`
				Mv      string `json:"mv"`
				Nl      string `json:"nl"`
				RowType string `json:"rowType"`
				Upl     string `json:"upl"`
			} `json:"upnl"`
		}
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		for k, r := range v.Upnl {
			_ = k
			_ = r.Dpl
			_ = r.El
			_ = r.Mv
			_ = r.Nl
			_ = r.RowType
			_ = r.Upl
		}
	})
}

func FuzzPortfolioAccountJSON(f *testing.F) {
	f.Add(`{"accountId":"U1234567","accountTitle":"Main","currency":"USD","accountAlias":"Main"}`)
	f.Add(`{}`)
	f.Add(`{"accountId":"","accountTitle":"","currency":"","accountAlias":""}`)
	f.Add(`{"accountId":"U9999999","accountTitle":"日本語テスト","currency":"JPY","accountAlias":"Test Alias"}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			AccountID     string `json:"accountId"`
			AccountTitle  string `json:"accountTitle"`
			AccountAlias  string `json:"accountAlias"`
			Currency      string `json:"currency"`
			AccountStatus string `json:"accountStatus"`
			DisplayName   string `json:"displayName"`
		}
		err := json.Unmarshal([]byte(data), &v)
		if err != nil {
			return
		}
		_ = v.AccountID
		_ = v.AccountTitle
		_ = v.AccountAlias
		_ = v.Currency
		_ = v.AccountStatus
		_ = v.DisplayName
	})
}

func FuzzPositionJSON(f *testing.F) {
	f.Add(`{"acctId":"U1234567","conid":265598,"contractDesc":"AAPL","assetClass":"STK","currency":"USD","position":"10.5","avgCost":"145.25","mktPrice":"150.00","mktValue":"1575.00","unrealizedPnl":"49.875"}`)
	f.Add(`{}`)
	f.Add(`{"acctId":"U1234567","conid":0,"contractDesc":"","assetClass":"","currency":"","position":"0","avgCost":"0","mktPrice":"0","mktValue":"0","unrealizedPnl":"0"}`)
	f.Add(`{"acctId":"U1234567","conid":-1,"contractDesc":"AAPL","assetClass":"STK","currency":"USD","position":"-10.5","avgCost":"145.25","mktPrice":"150.00","mktValue":"1575.00","unrealizedPnl":"-49.875"}`)
	f.Add(`{"acctId":"U1234567","conid":999999999,"contractDesc":"Very Long Contract Description That Might Test Buffer Handling","assetClass":"FUT","currency":"USD","position":"0.00000001","avgCost":"0.00000001","mktPrice":"0.00000001","mktValue":"0.00000001","unrealizedPnl":"0.00000001"}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
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
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		_ = v.AcctID
		_ = v.ConID
		_ = v.ContractDesc
		_ = v.AssetClass
		_ = v.Currency
		_ = v.Position
		_ = v.AvgCost
		_ = v.AvgPrice
		_ = v.MktPrice
		_ = v.MktValue
		_ = v.RealizedPnL
		_ = v.UnrealizedPnL
		_ = v.Model
	})
}

func FuzzLedgerCurrencyJSON(f *testing.F) {
	f.Add(`{"acctcode":"U1234567","currency":"USD","cashbalance":"100.25","settledcash":"90.00","netliquidationvalue":"1575.00","stockmarketvalue":"1575.00","stockoptionmarketvalue":"0","unrealizedpnl":"49.875","realizedpnl":"0"}`)
	f.Add(`{}`)
	f.Add(`{"acctcode":"","currency":"","cashbalance":"0","settledcash":"0","netliquidationvalue":"0","stockmarketvalue":"0","stockoptionmarketvalue":"0","unrealizedpnl":"0","realizedpnl":"0"}`)
	f.Add(`{"acctcode":"U1234567","currency":"USD","cashbalance":"-100.25","settledcash":"-90.00","netliquidationvalue":"-1575.00","stockmarketvalue":"-1575.00","stockoptionmarketvalue":"0","unrealizedpnl":"-49.875","realizedpnl":"-10.00"}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
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
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		_ = v.AcctCode
		_ = v.Currency
		_ = v.CashBalance
		_ = v.SettledCash
		_ = v.NetLiquidationValue
		_ = v.StockMarketValue
		_ = v.StockOptionMarketValue
		_ = v.UnrealizedPnL
		_ = v.RealizedPnL
	})
}

func FuzzContractJSON(f *testing.F) {
	f.Add(`{"con_id":265598,"symbol":"AAPL","company_name":"Apple Inc","currency":"USD","exchange":"NASDAQ","instrument_type":"STK","local_symbol":"AAPL","multiplier":"1","expiry_full":"","cusip":"037833100","category":"Technology","industry":"Consumer Electronics"}`)
	f.Add(`{}`)
	f.Add(`{"con_id":0,"symbol":"","company_name":"","currency":"","exchange":"","instrument_type":"","local_symbol":"","multiplier":"","expiry_full":"","cusip":"","category":"","industry":""}`)
	f.Add(`{"con_id":-1,"symbol":"AAPL","company_name":"Apple Inc","currency":"USD","exchange":"NASDAQ","instrument_type":"STK","local_symbol":"AAPL","multiplier":"100","expiry_full":"20261218","cusip":"037833100","category":"Technology","industry":"Consumer Electronics"}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
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
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		_ = v.ConID
		_ = v.Symbol
		_ = v.CompanyName
		_ = v.Currency
		_ = v.Exchange
		_ = v.InstrumentType
		_ = v.LocalSymbol
		_ = v.Multiplier
		_ = v.ExpiryFull
		_ = v.Cusip
		_ = v.Category
		_ = v.Industry
		_ = v.MaturityDate
	})
}

func FuzzOrderJSON(f *testing.F) {
	f.Add(`{"orderId":"999","account":"U1234567","conid":265598,"ticker":"AAPL","orderType":"LMT","side":"BUY","status":"PreSubmitted","timeInForce":"DAY","totalSize":"10","filledQuantity":"0","remainingQuantity":"10","price":"150.00","avgPrice":"0"}`)
	f.Add(`{}`)
	f.Add(`{"orderId":"","account":"","conid":0,"ticker":"","orderType":"","side":"","status":"","timeInForce":"","totalSize":"","filledQuantity":"","remainingQuantity":"","price":"","avgPrice":""}`)
	f.Add(`{"orderId":"999","account":"U1234567","conid":265598,"ticker":"AAPL","orderType":"MKT","side":"SELL","status":"Filled","timeInForce":"GTC","totalSize":"100","filledQuantity":"100","remainingQuantity":"0","price":"0","avgPrice":"150.00"}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
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
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		_ = v.OrderID
		_ = v.AccountID
		_ = v.ConID
		_ = v.Ticker
		_ = v.OrderType
		_ = v.Side
		_ = v.Status
		_ = v.TimeInForce
		_ = v.Size
		_ = v.FilledQuantity
		_ = v.RemainingQuantity
		_ = v.Price
		_ = v.AveragePrice
	})
}

func FuzzTradeJSON(f *testing.F) {
	f.Add(`{"order_id":"999","execution_id":"exec-1","account":"U1234567","conid":265598,"symbol":"AAPL","side":"B","size":"10","price":"150.00","commission":"1.00","net_amount":"1499.00","trade_time_r":1564652478000}`)
	f.Add(`{}`)
	f.Add(`{"order_id":"","execution_id":"","account":"","conid":0,"symbol":"","side":"","size":"","price":"","commission":"","net_amount":"","trade_time_r":0}`)
	f.Add(`{"order_id":"999","execution_id":"exec-1","account":"U1234567","conid":265598,"symbol":"AAPL","side":"S","size":"-10","price":"150.00","commission":"-1.00","net_amount":"-1499.00","trade_time_r":1564652478000}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
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
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		_ = v.OrderID
		_ = v.ExecutionID
		_ = v.AccountID
		_ = v.ConID
		_ = v.Symbol
		_ = v.Side
		_ = v.Size
		_ = v.Price
		_ = v.Commission
		_ = v.NetAmount
		_ = v.TradeTime
	})
}

func FuzzSnapshotJSON(f *testing.F) {
	f.Add(`[{"conid":265598,"31":"150.25","84":"150.20","_updated":1564652478}]`)
	f.Add(`[]`)
	f.Add(`[{"conid":0,"31":"0","84":"0","_updated":0}]`)
	f.Add(`[{"conid":265598,"31":"0.00000001","84":"0.00000001","_updated":1564652478}]`)
	f.Add(`[{"conid":265598,"31":"-150.25","84":"-150.20","_updated":-1}]`)
	f.Fuzz(func(t *testing.T, data string) {
		var v []map[string]json.RawMessage
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		for _, item := range v {
			for k, val := range item {
				_ = k
				_ = val
			}
		}
	})
}

func FuzzHistoryJSON(f *testing.F) {
	f.Add(`{"symbol":"AAPL","data":[{"t":1564652478,"o":"150.1","h":"151.2","l":"149.9","c":"150.9","v":"1000"}]}`)
	f.Add(`{"symbol":"","data":[]}`)
	f.Add(`{"symbol":"AAPL","data":[{"t":0,"o":"0","h":"0","l":"0","c":"0","v":"0"}]}`)
	f.Add(`{"symbol":"AAPL","data":[{"t":-1,"o":"-0.00000001","h":"-0.00000001","l":"-0.00000001","c":"-0.00000001","v":"-1"}]}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
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
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		_ = v.Symbol
		for _, b := range v.Data {
			_ = b.T
			_ = b.O
			_ = b.H
			_ = b.L
			_ = b.C
			_ = b.V
		}
	})
}

func FuzzOrderStatusJSON(f *testing.F) {
	f.Add(`{"order_id":"999","order_status":"PreSubmitted","conid":265598,"side":"BUY","filled_quantity":"0","remaining_quantity":"10","average_price":"0"}`)
	f.Add(`{}`)
	f.Add(`{"order_id":"","order_status":"","conid":0,"side":"","filled_quantity":"","remaining_quantity":"","average_price":""}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			OrderID           string `json:"order_id"`
			Status            string `json:"order_status"`
			ConID             int64  `json:"conid"`
			Side              string `json:"side"`
			OrderType         string `json:"order_type"`
			TimeInForce       string `json:"tif"`
			FilledQuantity    string `json:"filled_quantity"`
			RemainingQuantity string `json:"remaining_quantity"`
			AveragePrice      string `json:"average_price"`
		}
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		_ = v.OrderID
		_ = v.Status
		_ = v.ConID
		_ = v.Side
		_ = v.OrderType
		_ = v.TimeInForce
		_ = v.FilledQuantity
		_ = v.RemainingQuantity
		_ = v.AveragePrice
	})
}

func FuzzSubmitResultJSON(f *testing.F) {
	f.Add(`[{"order_id":"999","order_status":"PreSubmitted"}]`)
	f.Add(`[]`)
	f.Add(`[{"order_id":"","order_status":""}]`)
	f.Add(`[{"id":"reply-1","message":["msg1"],"messageIds":["id1"]}]`)
	f.Add(`[{"order_id":"999","order_status":"PreSubmitted"},{"id":"reply-1","message":["msg1"],"messageIds":["id1"]}]`)
	f.Fuzz(func(t *testing.T, data string) {
		var v []map[string]json.RawMessage
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		for _, item := range v {
			_ = item
		}
	})
}

func FuzzAlertDetailsJSON(f *testing.F) {
	f.Add(`{"alertId":"123","accountId":"U1234567","name":"AAPL above 150","alertType":"price","enabled":true,"filled":false}`)
	f.Add(`{}`)
	f.Add(`{"alertId":"","accountId":"","name":"","alertType":"","enabled":false,"filled":false}`)
	f.Add(`{"alertId":"123","accountId":"U1234567","name":"日本語アラート","alertType":"price","enabled":true,"filled":false}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			AlertID   string `json:"alertId"`
			AccountID string `json:"accountId"`
			Name      string `json:"name"`
			AlertType string `json:"alertType"`
			Enabled   bool   `json:"enabled"`
			Filled    bool   `json:"filled"`
		}
		err := json.Unmarshal([]byte(data), &v)
		if err != nil {
			return
		}
		_ = v.AlertID
		_ = v.AccountID
		_ = v.Name
		_ = v.AlertType
		_ = v.Enabled
		_ = v.Filled
	})
}

func FuzzWatchlistJSON(f *testing.F) {
	f.Add(`{"id":"wl-1","name":"Tech","instruments":[{"conId":265598,"symbol":"AAPL"}]}`)
	f.Add(`{}`)
	f.Add(`{"id":"","name":"","instruments":[]}`)
	f.Add(`{"id":"wl-1","name":"Tech","instruments":[{"conId":0,"symbol":""}]}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Instruments []struct {
				ConID  int64  `json:"conId"`
				Symbol string `json:"symbol"`
			} `json:"instruments"`
		}
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		_ = v.ID
		_ = v.Name
		for _, inst := range v.Instruments {
			_ = inst.ConID
			_ = inst.Symbol
		}
	})
}

func FuzzScannerResultJSON(f *testing.F) {
	f.Add(`{"conid":265598,"symbol":"AAPL","companyName":"Apple Inc","exchange":"NASDAQ","secType":"STK","distance":"1.5"}`)
	f.Add(`{}`)
	f.Add(`{"conid":0,"symbol":"","companyName":"","exchange":"","secType":"","distance":""}`)
	f.Add(`{"conid":265598,"symbol":"AAPL","companyName":"Apple Inc","exchange":"NASDAQ","secType":"STK","distance":"-1.5"}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			ConID       int64  `json:"conid"`
			Symbol      string `json:"symbol"`
			CompanyName string `json:"companyName"`
			Exchange    string `json:"exchange"`
			SecType     string `json:"secType"`
			Distance    string `json:"distance"`
		}
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		_ = v.ConID
		_ = v.Symbol
		_ = v.CompanyName
		_ = v.Exchange
		_ = v.SecType
		_ = v.Distance
	})
}

func FuzzAllocationJSON(f *testing.F) {
	f.Add(`{"assetClass":{"long":{"STK":"1000.5"},"short":{}}}`)
	f.Add(`{}`)
	f.Add(`{"assetClass":{"long":{},"short":{}}}`)
	f.Add(`{"assetClass":{"long":{"STK":"-1000.5","OPT":"0"},"short":{"STK":"-500"}}}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			AssetClass struct {
				Long  map[string]string `json:"long"`
				Short map[string]string `json:"short"`
			} `json:"assetClass"`
		}
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		for k, val := range v.AssetClass.Long {
			_ = k
			_ = val
		}
		for k, val := range v.AssetClass.Short {
			_ = k
			_ = val
		}
	})
}

func FuzzModelSummaryJSON(f *testing.F) {
	f.Add(`{"name":"Balanced","accountIds":["U1234567"]}`)
	f.Add(`{}`)
	f.Add(`{"name":"","accountIds":[]}`)
	f.Add(`{"name":"Model-日本語","accountIds":["U1234567","U7654321"]}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			Name       string   `json:"name"`
			AccountIDs []string `json:"accountIds"`
		}
		err := json.Unmarshal([]byte(data), &v)
		if err != nil {
			return
		}
		_ = v.Name
		_ = v.AccountIDs
	})
}

func FuzzContractSummaryJSON(f *testing.F) {
	f.Add(`{"conid":265598,"symbol":"AAPL","companyName":"Apple Inc","secType":"STK","description":"Apple Inc","exchange":"NASDAQ"}`)
	f.Add(`{}`)
	f.Add(`{"conid":0,"symbol":"","companyName":"","secType":"","description":"","exchange":""}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			ConID       int64  `json:"conid"`
			Symbol      string `json:"symbol"`
			CompanyName string `json:"companyName"`
			SecType     string `json:"secType"`
			Description string `json:"description"`
			Exchange    string `json:"exchange"`
		}
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		_ = v.ConID
		_ = v.Symbol
		_ = v.CompanyName
		_ = v.SecType
		_ = v.Description
		_ = v.Exchange
	})
}

func FuzzWhatIfResultJSON(f *testing.F) {
	f.Add(`{"amount":{"initial":"1000.50","maintenance":"800.25"}}`)
	f.Add(`{}`)
	f.Add(`{"amount":{}}`)
	f.Add(`{"amount":{"initial":"-1000.50","maintenance":"-800.25"}}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			Amount map[string]string `json:"amount"`
		}
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		for k, val := range v.Amount {
			_ = k
			_ = val
		}
	})
}

func FuzzReplyJSON(f *testing.F) {
	f.Add(`{"id":"reply-1","message":["Confirm this order"],"messageIds":["o354"]}`)
	f.Add(`{}`)
	f.Add(`{"id":"","message":[],"messageIds":[]}`)
	f.Add(`{"id":"reply-1","message":["日本語メッセージ","emoji test 🎉"],"messageIds":["id1","id2"]}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			ID         string   `json:"id"`
			Messages   []string `json:"message"`
			MessageIDs []string `json:"messageIds"`
		}
		err := json.Unmarshal([]byte(data), &v)
		if err != nil {
			return
		}
		_ = v.ID
		_ = v.Messages
		_ = v.MessageIDs
	})
}

func FuzzContractRulesJSON(f *testing.F) {
	f.Add(`{"algoEligible":true,"allOrNoneEligible":true,"canTradeAcctIds":["U1234567"],"defaultSize":"100","limitPrice":"0.01","orderTypes":["MKT","LMT"],"tifTypes":["DAY","GTC"],"TIF":"DAY","negativeCapable":false,"preview":true}`)
	f.Add(`{}`)
	f.Add(`{"algoEligible":false,"allOrNoneEligible":false,"canTradeAcctIds":[],"defaultSize":"0","limitPrice":"0","orderTypes":[],"tifTypes":[],"TIF":"","negativeCapable":false,"preview":false}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			AlgoEligible      bool     `json:"algoEligible"`
			AllOrNoneEligible bool     `json:"allOrNoneEligible"`
			CanTradeAcctIDs   []string `json:"canTradeAcctIds"`
			DefaultSize       string   `json:"defaultSize"`
			LimitPrice        string   `json:"limitPrice"`
			OrderTypes        []string `json:"orderTypes"`
			TIFTypes          []string `json:"tifTypes"`
			TimeInForce       string   `json:"TIF"`
			NegativeCapable   bool     `json:"negativeCapable"`
			Preview           bool     `json:"preview"`
		}
		err := json.Unmarshal([]byte(data), &v)
		if err != nil {
			return
		}
		_ = v.AlgoEligible
		_ = v.AllOrNoneEligible
		_ = v.CanTradeAcctIDs
		_ = v.DefaultSize
		_ = v.LimitPrice
		_ = v.OrderTypes
		_ = v.TIFTypes
		_ = v.TimeInForce
		_ = v.NegativeCapable
		_ = v.Preview
	})
}

func FuzzStrikesJSON(f *testing.F) {
	f.Add(`{"call":["150","155"],"put":["145","140"]}`)
	f.Add(`{}`)
	f.Add(`{"call":[],"put":[]}`)
	f.Add(`{"call":["-150","-155"],"put":["-145","-140"]}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			Call []string `json:"call"`
			Put  []string `json:"put"`
		}
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		_ = v.Call
		_ = v.Put
	})
}

func FuzzMTADetailsJSON(f *testing.F) {
	f.Add(`{"accounts":["U1234567","U7654321"]}`)
	f.Add(`{}`)
	f.Add(`{"accounts":[]}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			Accounts []string `json:"accounts"`
		}
		err := json.Unmarshal([]byte(data), &v)
		if err != nil {
			return
		}
		_ = v.Accounts
	})
}

func FuzzAlertActivationResultJSON(f *testing.F) {
	f.Add(`{"alertId":"123","active":true}`)
	f.Add(`{}`)
	f.Add(`{"alertId":"","active":false}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			AlertID string `json:"alertId"`
			Active  bool   `json:"active"`
		}
		err := json.Unmarshal([]byte(data), &v)
		if err != nil {
			return
		}
		_ = v.AlertID
		_ = v.Active
	})
}

func FuzzAlertCreationResultJSON(f *testing.F) {
	f.Add(`{"alertId":"124"}`)
	f.Add(`{}`)
	f.Add(`{"alertId":""}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			AlertID string `json:"alertId"`
		}
		err := json.Unmarshal([]byte(data), &v)
		if err != nil {
			return
		}
		_ = v.AlertID
	})
}

func FuzzScannerParameterJSON(f *testing.F) {
	f.Add(`{"scanTypes":["TOP_PERC_GAIN"],"instruments":["STK"]}`)
	f.Add(`{}`)
	f.Add(`{"scanTypes":[],"instruments":[]}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			ScanTypes   []string `json:"scanTypes"`
			Instruments []string `json:"instruments"`
		}
		err := json.Unmarshal([]byte(data), &v)
		if err != nil {
			return
		}
		_ = v.ScanTypes
		_ = v.Instruments
	})
}

func FuzzAllocationGroupJSON(f *testing.F) {
	f.Add(`{"name":"Group A","isHidden":false,"method":"EQUAL","accounts":[{"accountId":"U1234567","amount":"100.00"}]}`)
	f.Add(`{}`)
	f.Add(`{"name":"","isHidden":false,"method":"","accounts":[]}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			Name     string `json:"name"`
			IsHidden bool   `json:"isHidden"`
			Method   string `json:"method"`
			Accounts []struct {
				AccountID string `json:"accountId"`
				Amount    string `json:"amount"`
			} `json:"accounts"`
		}
		err := json.Unmarshal([]byte(data), &v)
		if err != nil {
			return
		}
		_ = v.Name
		_ = v.IsHidden
		_ = v.Method
		_ = v.Accounts
	})
}

func FuzzAllocationPresetJSON(f *testing.F) {
	f.Add(`{"accountId":"U1234567","percentage":"50.0"}`)
	f.Add(`{}`)
	f.Add(`{"accountId":"","percentage":""}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			AccountID  string `json:"accountId"`
			Percentage string `json:"percentage"`
		}
		err := json.Unmarshal([]byte(data), &v)
		if err != nil {
			return
		}
		_ = v.AccountID
		_ = v.Percentage
	})
}

func FuzzModelPresetJSON(f *testing.F) {
	f.Add(`{"name":"Balanced","accounts":["U1234567"]}`)
	f.Add(`{}`)
	f.Add(`{"name":"","accounts":[]}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			Name     string   `json:"name"`
			Accounts []string `json:"accounts"`
		}
		err := json.Unmarshal([]byte(data), &v)
		if err != nil {
			return
		}
		_ = v.Name
		_ = v.Accounts
	})
}

func FuzzModelAccountJSON(f *testing.F) {
	f.Add(`{"accountId":"U1234567","accountAlias":"Main"}`)
	f.Add(`{}`)
	f.Add(`{"accountId":"","accountAlias":""}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			AccountID    string `json:"accountId"`
			AccountAlias string `json:"accountAlias"`
		}
		err := json.Unmarshal([]byte(data), &v)
		if err != nil {
			return
		}
		_ = v.AccountID
		_ = v.AccountAlias
	})
}

func FuzzModelPositionJSON(f *testing.F) {
	f.Add(`{"accountId":"U1234567","conId":265598,"symbol":"AAPL","position":"10","avgCost":"145.25"}`)
	f.Add(`{}`)
	f.Add(`{"accountId":"","conId":0,"symbol":"","position":"","avgCost":""}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			AccountID string `json:"accountId"`
			ConID     int64  `json:"conId"`
			Symbol    string `json:"symbol"`
			Position  string `json:"position"`
			AvgCost   string `json:"avgCost"`
		}
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		_ = v.AccountID
		_ = v.ConID
		_ = v.Symbol
		_ = v.Position
		_ = v.AvgCost
	})
}

func FuzzTradingScheduleJSON(f *testing.F) {
	f.Add(`{"market":"AAPL","hours":"09:30-16:00"}`)
	f.Add(`{}`)
	f.Add(`{"market":"","hours":""}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			Market string `json:"market"`
			Hours  string `json:"hours"`
		}
		err := json.Unmarshal([]byte(data), &v)
		if err != nil {
			return
		}
		_ = v.Market
		_ = v.Hours
	})
}

func FuzzCurrencyPairJSON(f *testing.F) {
	f.Add(`{"conId":1,"sourceCurrency":"EUR","targetCurrency":"USD","exchangeRate":"1.08"}`)
	f.Add(`{}`)
	f.Add(`{"conId":0,"sourceCurrency":"","targetCurrency":"","exchangeRate":""}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			ConID          int64  `json:"conId"`
			SourceCurrency string `json:"sourceCurrency"`
			TargetCurrency string `json:"targetCurrency"`
			ExchangeRate   string `json:"exchangeRate"`
		}
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		_ = v.ConID
		_ = v.SourceCurrency
		_ = v.TargetCurrency
		_ = v.ExchangeRate
	})
}

func FuzzExchangeRateJSON(f *testing.F) {
	f.Add(`{"fromCurrency":"EUR","toCurrency":"USD","rate":"1.08"}`)
	f.Add(`{}`)
	f.Add(`{"fromCurrency":"","toCurrency":"","rate":""}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			FromCurrency string `json:"fromCurrency"`
			ToCurrency   string `json:"toCurrency"`
			Rate         string `json:"rate"`
		}
		err := json.Unmarshal([]byte(data), &v)
		if err != nil {
			return
		}
		_ = v.FromCurrency
		_ = v.ToCurrency
		_ = v.Rate
	})
}

func FuzzBondFilterJSON(f *testing.F) {
	f.Add(`{"instrumentId":"US912828","coupon":"2.5","maturityDate":"2030-01-15"}`)
	f.Add(`{}`)
	f.Add(`{"instrumentId":"","coupon":"","maturityDate":""}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			InstrumentID string `json:"instrumentId"`
			Coupon       string `json:"coupon"`
			MaturityDate string `json:"maturityDate"`
		}
		err := json.Unmarshal([]byte(data), &v)
		if err != nil {
			return
		}
		_ = v.InstrumentID
		_ = v.Coupon
		_ = v.MaturityDate
	})
}

func FuzzSecDefInfoJSON(f *testing.F) {
	f.Add(`{"conId":265598,"symbol":"AAPL","securityType":"STK","exchange":"NASDAQ","currency":"USD"}`)
	f.Add(`{}`)
	f.Add(`{"conId":0,"symbol":"","securityType":"","exchange":"","currency":""}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			ConID        int64  `json:"conId"`
			Symbol       string `json:"symbol"`
			SecurityType string `json:"securityType"`
			Exchange     string `json:"exchange"`
			Currency     string `json:"currency"`
		}
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		_ = v.ConID
		_ = v.Symbol
		_ = v.SecurityType
		_ = v.Exchange
		_ = v.Currency
	})
}

func FuzzSymbolSearchResultJSON(f *testing.F) {
	f.Add(`{"conid":265598,"symbol":"AAPL","companyName":"Apple Inc","secType":"STK","exchange":"NASDAQ"}`)
	f.Add(`{}`)
	f.Add(`{"conid":0,"symbol":"","companyName":"","secType":"","exchange":""}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			ConID       int64  `json:"conid"`
			Symbol      string `json:"symbol"`
			CompanyName string `json:"companyName"`
			SecType     string `json:"secType"`
			Exchange    string `json:"exchange"`
		}
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		_ = v.ConID
		_ = v.Symbol
		_ = v.CompanyName
		_ = v.SecType
		_ = v.Exchange
	})
}

func FuzzConidByExchangeJSON(f *testing.F) {
	f.Add(`{"conid":265598,"symbol":"AAPL","exchange":"NASDAQ","securityType":"STK"}`)
	f.Add(`{}`)
	f.Add(`{"conid":0,"symbol":"","exchange":"","securityType":""}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			ConID        int64  `json:"conid"`
			Symbol       string `json:"symbol"`
			Exchange     string `json:"exchange"`
			SecurityType string `json:"securityType"`
		}
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		_ = v.ConID
		_ = v.Symbol
		_ = v.Exchange
		_ = v.SecurityType
	})
}

func FuzzFutureBySymbolJSON(f *testing.F) {
	f.Add(`{"AAPL":[{"conid":265598,"symbol":"AAPL","expiry":"2026-12","exchange":"GLOBEX"}]}`)
	f.Add(`{}`)
	f.Add(`{"":{"conid":0,"symbol":"","expiry":"","exchange":""}}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v map[string][]struct {
			ConID    int64  `json:"conid"`
			Symbol   string `json:"symbol"`
			Expiry   string `json:"expiry"`
			Exchange string `json:"exchange"`
		}
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		for _, items := range v {
			for _, item := range items {
				_ = item.ConID
				_ = item.Symbol
				_ = item.Expiry
				_ = item.Exchange
			}
		}
	})
}

func FuzzInstrumentDefinitionJSON(f *testing.F) {
	f.Add(`{"conid":265598,"symbol":"AAPL","companyName":"Apple Inc","securityType":"STK","exchange":"NASDAQ","currency":"USD"}`)
	f.Add(`[]`)
	f.Add(`[{"conid":0,"symbol":"","companyName":"","securityType":"","exchange":"","currency":""}]`)
	f.Fuzz(func(t *testing.T, data string) {
		var v []struct {
			ConID        int64  `json:"conid"`
			Symbol       string `json:"symbol"`
			CompanyName  string `json:"companyName"`
			SecurityType string `json:"securityType"`
			Exchange     string `json:"exchange"`
			Currency     string `json:"currency"`
		}
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		for _, item := range v {
			_ = item.ConID
			_ = item.Symbol
			_ = item.CompanyName
			_ = item.SecurityType
			_ = item.Exchange
			_ = item.Currency
		}
	})
}

func FuzzStockBySymbolJSON(f *testing.F) {
	f.Add(`{"AAPL":[{"conid":265598,"symbol":"AAPL","exchange":"NASDAQ","securityType":"STK"}]}`)
	f.Add(`{}`)
	f.Add(`{"":{"conid":0,"symbol":"","exchange":"","securityType":""}}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v map[string][]struct {
			ConID        int64  `json:"conid"`
			Symbol       string `json:"symbol"`
			Exchange     string `json:"exchange"`
			SecurityType string `json:"securityType"`
		}
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		for _, items := range v {
			for _, item := range items {
				_ = item.ConID
				_ = item.Symbol
				_ = item.Exchange
				_ = item.SecurityType
			}
		}
	})
}

func FuzzComboPositionJSON(f *testing.F) {
	f.Add(`{"acctId":"U1234567","conid":265598,"symbol":"AAPL","position":"10"}`)
	f.Add(`{}`)
	f.Add(`{"acctId":"","conid":0,"symbol":"","position":""}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			AcctID   string `json:"acctId"`
			ConID    int64  `json:"conid"`
			Symbol   string `json:"symbol"`
			Position string `json:"position"`
		}
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		_ = v.AcctID
		_ = v.ConID
		_ = v.Symbol
		_ = v.Position
	})
}

func FuzzAccountJSON(f *testing.F) {
	f.Add(`{"id":"U1234567","alias":"Main"}`)
	f.Add(`{}`)
	f.Add(`{"id":"","alias":""}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			ID    string `json:"id"`
			Alias string `json:"alias"`
		}
		err := json.Unmarshal([]byte(data), &v)
		if err != nil {
			return
		}
		_ = v.ID
		_ = v.Alias
	})
}

func FuzzCashBalanceJSON(f *testing.F) {
	f.Add(`{"currency":"USD","balance":"50.5","settledCash":"40.0"}`)
	f.Add(`{}`)
	f.Add(`{"currency":"","balance":"","settledCash":""}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			Currency    string `json:"currency"`
			Balance     string `json:"balance"`
			SettledCash string `json:"settledCash"`
		}
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		err := dec.Decode(&v)
		if err != nil {
			return
		}
		_ = v.Currency
		_ = v.Balance
		_ = v.SettledCash
	})
}

func FuzzSummaryValueJSON(f *testing.F) {
	f.Add(`{"amount":"1575.00","currency":"USD","value":"","isNull":false}`)
	f.Add(`{}`)
	f.Add(`{"amount":null,"currency":null,"value":"","isNull":true}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v struct {
			Amount   json.RawMessage `json:"amount"`
			Currency json.RawMessage `json:"currency"`
			Value    string          `json:"value"`
			IsNull   bool            `json:"isNull"`
		}
		err := json.Unmarshal([]byte(data), &v)
		if err != nil {
			return
		}
		_ = v.Amount
		_ = v.Currency
		_ = v.Value
		_ = v.IsNull
	})
}

func FuzzMoneyStringRoundTrip(f *testing.F) {
	f.Add(`"1234.5600"`)
	f.Add(`"0"`)
	f.Add(`"-1.5"`)
	f.Add(`"999999999999999.9999"`)
	f.Add(`"0.00000001"`)
	f.Add(`"-0.00000001"`)
	f.Add(`""`)
	f.Fuzz(func(t *testing.T, data string) {
		var s string
		err := json.Unmarshal([]byte(data), &s)
		if err != nil {
			return
		}
		encoded, err := json.Marshal(s)
		if err != nil {
			t.Errorf("re-marshal failed for %q: %v", s, err)
			return
		}
		var roundTrip string
		if err := json.Unmarshal(encoded, &roundTrip); err != nil {
			t.Errorf("round-trip unmarshal failed for %q: %v", s, err)
			return
		}
		if roundTrip != s {
			t.Errorf("precision loss: original=%q round-trip=%q", s, roundTrip)
		}
	})
}

func FuzzMoneyQuantityMapRoundTrip(f *testing.F) {
	f.Add(`{"initial":"1000.50","maintenance":"800.25"}`)
	f.Add(`{}`)
	f.Add(`{"initial":"0","maintenance":"0"}`)
	f.Add(`{"initial":"-1000.50","maintenance":"-800.25"}`)
	f.Add(`{"initial":"0.00000001","maintenance":"0.00000001"}`)
	f.Fuzz(func(t *testing.T, data string) {
		var v map[string]string
		dec := json.NewDecoder(bytes.NewReader([]byte(data)))
		dec.UseNumber()
		if err := dec.Decode(&v); err != nil {
			return
		}
		for k, val := range v {
			_ = k
			enc, err := json.Marshal(val)
			if err != nil {
				t.Errorf("re-marshal failed for key=%q val=%q: %v", k, val, err)
				continue
			}
			var roundTrip string
			if err := json.Unmarshal(enc, &roundTrip); err != nil {
				t.Errorf("round-trip failed for key=%q val=%q: %v", k, val, err)
				continue
			}
			if roundTrip != val {
				t.Errorf("precision loss: key=%q original=%q round-trip=%q", k, val, roundTrip)
			}
		}
	})
}
