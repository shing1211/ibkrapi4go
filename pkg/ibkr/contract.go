// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
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
