// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/shing1211/ibkrapi4go/client"
)

// ContractSummary is a contract search result.
type ContractSummary struct {
	// ConID is the contract identifier.
	ConID ConID
	// Symbol is the ticker symbol.
	Symbol string
	// CompanyName is the issuer/company name.
	CompanyName string
	// SecType is the security type (e.g. STK, OPT, FUT).
	SecType string
	// Description is a human-readable description.
	Description string
	// Exchange is the listing exchange.
	Exchange string
}

// Contract is detailed instrument information.
type Contract struct {
	// ConID is the contract identifier.
	ConID ConID
	// Symbol is the ticker symbol.
	Symbol string
	// CompanyName is the issuer/company name.
	CompanyName string
	// Currency is the instrument currency.
	Currency string
	// Exchange is the listing exchange.
	Exchange string
	// InstrumentType is the IBKR instrument type.
	InstrumentType string
	// LocalSymbol is the exchange-local symbol.
	LocalSymbol string
	// Multiplier is the contract multiplier.
	Multiplier string
	// ExpiryFull is the full expiry for derivatives.
	ExpiryFull string
	// Cusip is the CUSIP, when available.
	Cusip string
	// Category is the instrument category.
	Category string
	// Industry is the industry classification.
	Industry string
	// MaturityDate is the maturity date for fixed income.
	MaturityDate string
}

// ContractRules are the trading rules for a contract.
type ContractRules struct {
	// ConID is the contract identifier the rules apply to.
	ConID ConID
	// Error is a server-provided error, if any.
	Error string
	// AlgoEligible reports whether algos may be used.
	AlgoEligible bool
	// AllOrNoneEligible reports whether all-or-none orders are allowed.
	AllOrNoneEligible bool
	// CanTradeAcctIDs lists accounts that may trade the contract.
	CanTradeAcctIDs []string
	// CashCurrency is the cash quantity currency.
	CashCurrency string
	// CashQtyIncrement is the cash quantity increment.
	CashQtyIncrement string
	// CashSize is the cash size.
	CashSize string
	// CostReport reports whether a cost report is required.
	CostReport bool
	// DefaultSize is the default order size.
	DefaultSize string
	// DisplaySize is the default display size.
	DisplaySize string
	// ForceOrderPreview reports whether an order preview is mandatory.
	ForceOrderPreview bool
	// HasSecondary reports whether a secondary exchange is available.
	HasSecondary bool
	// LimitPrice is the price increment for limit orders.
	LimitPrice string
	// StopPrice is the price increment for stop orders.
	StopPrice string
	// OrderTypes lists supported order types.
	OrderTypes []string
	// OrderTypesOutside lists order types allowed outside RTH.
	OrderTypesOutside []string
	// TIFTypes lists supported time-in-force values.
	TIFTypes []string
	// TimeInForce is the contract's default time-in-force.
	TimeInForce string
	// SizeIncrement is the order size increment.
	SizeIncrement string
	// PriceMagnifier is the price magnifier.
	PriceMagnifier string
	// Increment is the price increment.
	Increment string
	// NegativeCapable reports whether negative prices are allowed.
	NegativeCapable bool
	// Preview reports whether an order preview is available.
	Preview bool
}

// Strikes is the set of valid option strikes for an underlying.
type Strikes struct {
	// Call holds the call strikes as decimal strings.
	Call []string
	// Put holds the put strikes as decimal strings.
	Put []string
}

// TradeManager exposes order and contract operations. It is safe for concurrent
// use. Order mutations are never retried (ADR 0009).
type TradeManager struct {
	client *Client
}

// SearchContracts searches instruments by symbol.
func (m *TradeManager) SearchContracts(ctx context.Context, symbol string) ([]ContractSummary, error) {
	const op = "Trade.SearchContracts"
	params := &client.GetContractSymbolsParams{Symbol: &symbol}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetContractSymbols(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw []contractSummaryRaw
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]ContractSummary, 0, len(raw))
	for _, r := range raw {
		out = append(out, ContractSummary{
			ConID:       ConID(jsonNumberToInt(r.ConID)),
			Symbol:      r.Symbol,
			CompanyName: r.CompanyName,
			SecType:     r.SecType,
			Description: r.Description,
			Exchange:    r.Exchange,
		})
	}
	return out, nil
}

// ContractInfo returns detailed instrument information for a contract.
func (m *TradeManager) ContractInfo(ctx context.Context, conid ConID) (*Contract, error) {
	const op = "Trade.ContractInfo"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetInstrumentInfo(ctx, strconv.Itoa(int(conid)))
	})
	if err != nil {
		return nil, err
	}
	var raw contractInfoRaw
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &Contract{
		ConID:          ConID(jsonNumberToInt(raw.ConID)),
		Symbol:         raw.Symbol,
		CompanyName:    raw.CompanyName,
		Currency:       raw.Currency,
		Exchange:       raw.Exchange,
		InstrumentType: raw.InstrumentType,
		LocalSymbol:    raw.LocalSymbol,
		Multiplier:     raw.Multiplier.String(),
		ExpiryFull:     raw.ExpiryFull,
		Cusip:          raw.Cusip,
		Category:       raw.Category,
		Industry:       raw.Industry,
		MaturityDate:   raw.MaturityDate,
	}, nil
}

// ContractRules returns the trading rules for a contract.
func (m *TradeManager) ContractRules(ctx context.Context, conid ConID) (*ContractRules, error) {
	const op = "Trade.ContractRules"
	body := client.GetContractRulesJSONRequestBody(client.GetContractRulesJSONBody{Conid: int64(conid)})
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetContractRules(ctx, body)
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &ContractRules{
		ConID:             conid,
		Error:             rawToString(raw, "error"),
		AlgoEligible:      rawToBool(raw, "algoEligible"),
		AllOrNoneEligible: rawToBool(raw, "allOrNoneEligible"),
		CanTradeAcctIDs:   rawToStringSlice(raw, "canTradeAcctIds"),
		CashCurrency:      rawToString(raw, "cashCcy"),
		CashQtyIncrement:  rawToString(raw, "cashQtyIncr"),
		CashSize:          rawToString(raw, "cashSize"),
		CostReport:        rawToBool(raw, "costReport"),
		DefaultSize:       rawToString(raw, "defaultSize"),
		DisplaySize:       rawToString(raw, "displaySize"),
		ForceOrderPreview: rawToBool(raw, "forceOrderPreview"),
		HasSecondary:      rawToBool(raw, "hasSecondary"),
		LimitPrice:        rawToString(raw, "limitPrice"),
		StopPrice:         rawToString(raw, "stopPrice"),
		OrderTypes:        rawToStringSlice(raw, "orderTypes"),
		OrderTypesOutside: rawToStringSlice(raw, "orderTypesOutside"),
		TIFTypes:          rawToStringSlice(raw, "tifTypes"),
		TimeInForce:       rawToString(raw, "TIF"),
		SizeIncrement:     rawToString(raw, "sizeIncrement"),
		PriceMagnifier:    rawToString(raw, "priceMagnifier"),
		Increment:         rawToString(raw, "increment"),
		NegativeCapable:   rawToBool(raw, "negativeCapable"),
		Preview:           rawToBool(raw, "preview"),
	}, nil
}

// Strikes returns the valid option strikes for an underlying.
func (m *TradeManager) Strikes(ctx context.Context, conid ConID, secType, month string) (*Strikes, error) {
	const op = "Trade.Strikes"
	params := &client.GetContractStrikesParams{
		Conid:   strconv.Itoa(int(conid)),
		Sectype: client.GetContractStrikesParamsSectype(secType),
		Month:   month,
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetContractStrikes(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw struct {
		Call []json.Number `json:"call"`
		Put  []json.Number `json:"put"`
	}
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	nums := func(in []json.Number) []string {
		out := make([]string, len(in))
		for i, n := range in {
			out[i] = n.String()
		}
		return out
	}
	return &Strikes{Call: nums(raw.Call), Put: nums(raw.Put)}, nil
}

// TradingSchedule holds a trading schedule entry.
type TradingSchedule struct {
	// Market is the market name.
	Market string `json:"market"`
	// Hours are the trading hours.
	Hours string `json:"hours"`
}

// InfoAndRules holds combined contract information and rules.
type InfoAndRules struct {
	// ConID is the contract identifier.
	ConID ConID `json:"conId"`
	// Symbol is the ticker symbol.
	Symbol string `json:"symbol"`
	// Rules holds the trading rules.
	Rules *ContractRules `json:"rules,omitempty"`
}

// CurrencyPair holds a currency pair exchange rate.
type CurrencyPair struct {
	// ConID is the contract identifier.
	ConID ConID `json:"conId"`
	// SourceCurrency is the source currency.
	SourceCurrency string `json:"sourceCurrency"`
	// TargetCurrency is the target currency.
	TargetCurrency string `json:"targetCurrency"`
	// ExchangeRate is the exchange rate.
	ExchangeRate string `json:"exchangeRate"`
}

// ExchangeRate holds an exchange rate between two currencies.
type ExchangeRate struct {
	// FromCurrency is the source currency.
	FromCurrency string `json:"fromCurrency"`
	// ToCurrency is the target currency.
	ToCurrency string `json:"toCurrency"`
	// Rate is the exchange rate.
	Rate string `json:"rate"`
}

// BondFilter holds bond filter criteria.
type BondFilter struct {
	// InstrumentID is the instrument identifier.
	InstrumentID string `json:"instrumentId"`
	// Coupon is the coupon rate.
	Coupon string `json:"coupon"`
	// MaturityDate is the maturity date.
	MaturityDate string `json:"maturityDate"`
}

// SecDefInfo holds security definition information.
type SecDefInfo struct {
	// ConID is the contract identifier.
	ConID ConID `json:"conId"`
	// Symbol is the ticker symbol.
	Symbol string `json:"symbol"`
	// SecurityType is the security type.
	SecurityType string `json:"securityType"`
	// Exchange is the listing exchange.
	Exchange string `json:"exchange"`
	// Currency is the instrument currency.
	Currency string `json:"currency"`
}

// SymbolSearchResult holds a contract symbol search result.
type SymbolSearchResult struct {
	// ConID is the contract identifier.
	ConID ConID `json:"conId"`
	// Symbol is the ticker symbol.
	Symbol string `json:"symbol"`
	// CompanyName is the issuer/company name.
	CompanyName string `json:"companyName"`
	// SecType is the security type.
	SecType string `json:"secType"`
	// Exchange is the listing exchange.
	Exchange string `json:"exchange"`
}

// ConidByExchange holds a contract ID resolved by exchange.
type ConidByExchange struct {
	// ConID is the contract identifier.
	ConID ConID `json:"conId"`
	// Symbol is the ticker symbol.
	Symbol string `json:"symbol"`
	// Exchange is the listing exchange.
	Exchange string `json:"exchange"`
	// SecurityType is the security type.
	SecurityType string `json:"securityType"`
}

// FutureBySymbol holds a future contract resolved by symbol.
type FutureBySymbol struct {
	// ConID is the contract identifier.
	ConID ConID `json:"conId"`
	// Symbol is the ticker symbol.
	Symbol string `json:"symbol"`
	// Expiry is the expiry date.
	Expiry string `json:"expiry"`
	// Exchange is the listing exchange.
	Exchange string `json:"exchange"`
}

// InstrumentDefinition holds detailed instrument definition data.
type InstrumentDefinition struct {
	// ConID is the contract identifier.
	ConID ConID `json:"conId"`
	// Symbol is the ticker symbol.
	Symbol string `json:"symbol"`
	// CompanyName is the issuer/company name.
	CompanyName string `json:"companyName"`
	// SecurityType is the security type.
	SecurityType string `json:"securityType"`
	// Exchange is the listing exchange.
	Exchange string `json:"exchange"`
	// Currency is the instrument currency.
	Currency string `json:"currency"`
}

// StockBySymbol holds a stock contract resolved by symbol.
type StockBySymbol struct {
	// ConID is the contract identifier.
	ConID ConID `json:"conId"`
	// Symbol is the ticker symbol.
	Symbol string `json:"symbol"`
	// Exchange is the listing exchange.
	Exchange string `json:"exchange"`
	// SecurityType is the security type.
	SecurityType string `json:"securityType"`
}

// TradingSchedule returns the trading schedule for a given contract.
func (m *TradeManager) TradingSchedule(ctx context.Context, conid ConID, exchange *string) (*TradingSchedule, error) {
	const op = "Trade.GetTradingSchedule"
	params := &client.GetTradingScheduleParams{
		Conid:    strconv.Itoa(int(conid)),
		Exchange: exchange,
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetTradingSchedule(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &TradingSchedule{
		Market: rawToString(raw, "market"),
		Hours:  rawToString(raw, "hours"),
	}, nil
}

// AlgosByInstrument returns available algo types for a contract.
func (m *TradeManager) AlgosByInstrument(ctx context.Context, conid ConID) (json.RawMessage, error) {
	const op = "Trade.GetAlgosByInstrument"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetAlgosByInstrument(ctx, strconv.Itoa(int(conid)), nil)
	})
	if err != nil {
		return nil, err
	}
	var result json.RawMessage
	if err := decodeJSON(resp, op, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// InfoAndRules returns combined information and rules for a contract.
func (m *TradeManager) InfoAndRules(ctx context.Context, conid ConID) (*InfoAndRules, error) {
	const op = "Trade.GetInfoAndRules"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetInfoAndRules(ctx, strconv.Itoa(int(conid)))
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &InfoAndRules{
		ConID:  ConID(jsonNumberToInt(jsonNumber(raw["conId"]))),
		Symbol: rawToString(raw, "symbol"),
	}, nil
}

// CurrencyPairs returns currency pair exchange rates for a given currency.
func (m *TradeManager) CurrencyPairs(ctx context.Context, currency string) ([]CurrencyPair, error) {
	const op = "Trade.GetCurrencyPairs"
	params := &client.GetCurrencyPairsParams{Currency: currency}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetCurrencyPairs(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]CurrencyPair, 0, len(raw))
	for _, item := range raw {
		out = append(out, CurrencyPair{
			ConID:          ConID(jsonNumberToInt(jsonNumber(item["conId"]))),
			SourceCurrency: rawToString(item, "sourceCurrency"),
			TargetCurrency: rawToString(item, "targetCurrency"),
			ExchangeRate:   rawToString(item, "exchangeRate"),
		})
	}
	return out, nil
}

// ExchangeRates returns exchange rates for a currency pair.
func (m *TradeManager) ExchangeRates(ctx context.Context, source, target string) ([]ExchangeRate, error) {
	const op = "Trade.GetExchangeRates"
	params := &client.GetExchangeRatesParams{Source: source, Target: target}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetExchangeRates(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]ExchangeRate, 0, len(raw))
	for _, item := range raw {
		out = append(out, ExchangeRate{
			FromCurrency: rawToString(item, "fromCurrency"),
			ToCurrency:   rawToString(item, "toCurrency"),
			Rate:         rawToString(item, "rate"),
		})
	}
	return out, nil
}

// BondFilters returns bond filter results for a given symbol and issuer.
func (m *TradeManager) BondFilters(ctx context.Context, symbol, issuerID string) ([]BondFilter, error) {
	const op = "Trade.GetBondFilters"
	params := &client.GetBondFiltersParams{Symbol: symbol, IssuerId: issuerID}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetBondFilters(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]BondFilter, 0, len(raw))
	for _, item := range raw {
		out = append(out, BondFilter{
			InstrumentID: rawToString(item, "instrumentId"),
			Coupon:       rawToString(item, "coupon"),
			MaturityDate: rawToString(item, "maturityDate"),
		})
	}
	return out, nil
}

// SecDefInfos searches for contracts by criteria.
func (m *TradeManager) SecDefInfos(ctx context.Context, conid ConID) ([]SecDefInfo, error) {
	const op = "Trade.GetContractInfo"
	cid := strconv.Itoa(int(conid))
	params := &client.GetContractInfoParams{Conid: &cid}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetContractInfo(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]SecDefInfo, 0, len(raw))
	for _, item := range raw {
		out = append(out, SecDefInfo{
			ConID:        ConID(jsonNumberToInt(jsonNumber(item["conId"]))),
			Symbol:       rawToString(item, "symbol"),
			SecurityType: rawToString(item, "securityType"),
			Exchange:     rawToString(item, "exchange"),
			Currency:     rawToString(item, "currency"),
		})
	}
	return out, nil
}

// ContractSymbolsFromBody searches for contracts using POST body parameters.
func (m *TradeManager) ContractSymbolsFromBody(ctx context.Context, symbol string) ([]SymbolSearchResult, error) {
	const op = "Trade.GetContractSymbolsFromBody"
	bodyJSON := map[string]interface{}{"symbol": symbol}
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return nil, &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetContractSymbolsFromBodyWithBody(ctx,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]SymbolSearchResult, 0, len(raw))
	for _, item := range raw {
		out = append(out, SymbolSearchResult{
			ConID:       ConID(jsonNumberToInt(jsonNumber(item["conid"]))),
			Symbol:      rawToString(item, "symbol"),
			CompanyName: rawToString(item, "companyName"),
			SecType:     rawToString(item, "secType"),
			Exchange:    rawToString(item, "exchange"),
		})
	}
	return out, nil
}

// ConidsByExchange returns contract IDs for instruments on a given exchange.
func (m *TradeManager) ConidsByExchange(ctx context.Context, exchange string) ([]ConidByExchange, error) {
	const op = "Trade.GetConidsByExchange"
	params := &client.GetConidsByExchangeParams{Exchange: exchange}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetConidsByExchange(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]ConidByExchange, 0, len(raw))
	for _, item := range raw {
		out = append(out, ConidByExchange{
			ConID:        ConID(jsonNumberToInt(jsonNumber(item["conid"]))),
			Symbol:       rawToString(item, "symbol"),
			Exchange:     rawToString(item, "exchange"),
			SecurityType: rawToString(item, "securityType"),
		})
	}
	return out, nil
}

// FutureBySymbol returns future contracts for a given symbol.
func (m *TradeManager) FutureBySymbol(ctx context.Context, symbols string, exchange *string) ([]FutureBySymbol, error) {
	const op = "Trade.GetFutureBySymbol"
	params := &client.GetFutureBySymbolParams{Symbols: symbols, Exchange: exchange}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetFutureBySymbol(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	// The response is typically a map of symbol -> array of futures
	var out []FutureBySymbol
	for _, v := range raw {
		var items []map[string]json.RawMessage
		if json.Unmarshal(v, &items) == nil {
			for _, item := range items {
				out = append(out, FutureBySymbol{
					ConID:    ConID(jsonNumberToInt(jsonNumber(item["conid"]))),
					Symbol:   rawToString(item, "symbol"),
					Expiry:   rawToString(item, "expiry"),
					Exchange: rawToString(item, "exchange"),
				})
			}
		}
	}
	return out, nil
}

// InstrumentDefinition returns instrument definitions for given contract IDs.
func (m *TradeManager) InstrumentDefinition(ctx context.Context, conids string) ([]InstrumentDefinition, error) {
	const op = "Trade.GetInstrumentDefinition"
	params := &client.GetInstrumentDefinitionParams{Conids: conids}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetInstrumentDefinition(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]InstrumentDefinition, 0, len(raw))
	for _, item := range raw {
		out = append(out, InstrumentDefinition{
			ConID:        ConID(jsonNumberToInt(jsonNumber(item["conid"]))),
			Symbol:       rawToString(item, "symbol"),
			CompanyName:  rawToString(item, "companyName"),
			SecurityType: rawToString(item, "securityType"),
			Exchange:     rawToString(item, "exchange"),
			Currency:     rawToString(item, "currency"),
		})
	}
	return out, nil
}

// TradingScheduleBySymbol returns the trading schedule for a given symbol.
func (m *TradeManager) TradingScheduleBySymbol(ctx context.Context, assetClass, symbol string) (*TradingSchedule, error) {
	const op = "Trade.GetTradingScheduleBySymbol"
	params := &client.GetTradingSchedule2Params{
		AssetClass: client.GetTradingSchedule2ParamsAssetClass(assetClass),
		Symbol:     symbol,
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetTradingSchedule2(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &TradingSchedule{
		Market: rawToString(raw, "market"),
		Hours:  rawToString(raw, "hours"),
	}, nil
}

// StockBySymbol returns stock contracts for a given symbol.
func (m *TradeManager) StockBySymbol(ctx context.Context, symbols string) ([]StockBySymbol, error) {
	const op = "Trade.GetStockBySymbol"
	params := &client.GetStockBySymbolParams{Symbols: symbols}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetStockBySymbol(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	var out []StockBySymbol
	for _, v := range raw {
		var items []map[string]json.RawMessage
		if json.Unmarshal(v, &items) == nil {
			for _, item := range items {
				out = append(out, StockBySymbol{
					ConID:        ConID(jsonNumberToInt(jsonNumber(item["conid"]))),
					Symbol:       rawToString(item, "symbol"),
					Exchange:     rawToString(item, "exchange"),
					SecurityType: rawToString(item, "securityType"),
				})
			}
		}
	}
	return out, nil
}

// --- raw adapters -----------------------------------------------------------

type contractSummaryRaw struct {
	ConID       json.Number `json:"conid"`
	Symbol      string      `json:"symbol"`
	CompanyName string      `json:"companyName"`
	SecType     string      `json:"secType"`
	Description string      `json:"description"`
	Exchange    string      `json:"exchange"`
}

type contractInfoRaw struct {
	ConID          json.Number `json:"con_id"`
	Symbol         string      `json:"symbol"`
	CompanyName    string      `json:"company_name"`
	Currency       string      `json:"currency"`
	Exchange       string      `json:"exchange"`
	InstrumentType string      `json:"instrument_type"`
	LocalSymbol    string      `json:"local_symbol"`
	Multiplier     json.Number `json:"multiplier"`
	ExpiryFull     string      `json:"expiry_full"`
	Cusip          string      `json:"cusip"`
	Category       string      `json:"category"`
	Industry       string      `json:"industry"`
	MaturityDate   string      `json:"maturity_date"`
}
