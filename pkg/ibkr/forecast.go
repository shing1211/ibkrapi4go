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

// ForecastManager exposes forecast (event contract) operations. It is safe for
// concurrent use.
type ForecastManager struct {
	client *Client
}

// ForecastDetails describes one event contract.
//
// Numeric fields are json.Number rather than int64 or float64. IBKR sends them as
// JSON numbers, and ADR 0008 requires that a decimal arriving over the wire is
// never routed through a binary float, so the digits are carried through as they
// were sent. A field IBKR omits is the empty json.Number.
type ForecastDetails struct {
	// Category is the event category, e.g. Elections.
	Category string `json:"category"`
	// ConidNo is the contract id of the No side.
	ConidNo json.Number `json:"conid_no"`
	// ConidYes is the contract id of the Yes side.
	ConidYes json.Number `json:"conid_yes"`
	// Exchange is the listing exchange.
	Exchange string `json:"exchange"`
	// Expiration is the expiry date in YYYYMMDD form.
	Expiration string `json:"expiration"`
	// LogoCategory is the logo-service category used to retrieve an image.
	LogoCategory string `json:"logo_category"`
	// MarketName is the market identifier.
	MarketName string `json:"market_name"`
	// MeasuredPeriod is the settlement measurement window, e.g. 1D.
	MeasuredPeriod string `json:"measured_period"`
	// Payout is the payout ratio or amount, as IBKR sends it.
	Payout string `json:"payout"`
	// Question is the event question.
	Question string `json:"question"`
	// Side is Y or N, denoting the Yes or No contract.
	Side string `json:"side"`
	// Strike is the strike price.
	Strike json.Number `json:"strike"`
	// StrikeLabel is the strike label to display.
	StrikeLabel string `json:"strike_label"`
	// Symbol is the contract symbol.
	Symbol string `json:"symbol"`
	// UnderlyingConid is the immediate underlier to the contract.
	UnderlyingConid json.Number `json:"underlying_conid"`
}

// ForecastMarketContract is one contract within a forecast market.
type ForecastMarketContract struct {
	// Conid is the contract identifier.
	Conid json.Number `json:"conid"`
	// Expiration is the expiry date in YYYYMMDD form.
	Expiration string `json:"expiration"`
	// ExpiryLabel is the label to show on the expiration tab.
	ExpiryLabel string `json:"expiry_label"`
	// Side is Y or N, denoting the Yes or No contract.
	Side string `json:"side"`
	// Strike is the strike price.
	Strike json.Number `json:"strike"`
	// StrikeLabel is the strike label to display.
	StrikeLabel string `json:"strike_label"`
	// TimeSpecifier is the specified date of expiration.
	TimeSpecifier string `json:"time_specifier"`
	// UnderlyingConid is the immediate underlier to the contract.
	UnderlyingConid json.Number `json:"underlying_conid"`
}

// ForecastMarket lists the contracts available for one forecast market.
type ForecastMarket struct {
	// Contracts are the contracts matching the requested market.
	Contracts []ForecastMarketContract `json:"contracts"`
	// Exchange is the exchange passed in the request, or determined internally.
	Exchange string `json:"exchange"`
	// ExcludeHistoricalData reports whether the UI should omit the underlying chart.
	ExcludeHistoricalData bool `json:"exclude_historical_data"`
	// LogoCategory is the logo-service category used to retrieve an image.
	LogoCategory string `json:"logo_category"`
	// MarketName is the market identifier.
	MarketName string `json:"market_name"`
	// Payout is the payout ratio or amount, as IBKR sends it.
	Payout string `json:"payout"`
	// Symbol is the market symbol.
	Symbol string `json:"symbol"`
}

// ForecastRules describes the trading rules for an event contract. As with
// ForecastDetails, the numeric fields are json.Number.
type ForecastRules struct {
	// AssetClass is the asset class the contract settles against.
	AssetClass string `json:"asset_class"`
	// DataAndResolutionLink links to the data and resolution source.
	DataAndResolutionLink string `json:"data_and_resolution_link"`
	// Description describes the contract.
	Description string `json:"description"`
	// ExchangeTimezone is the exchange timezone.
	ExchangeTimezone string `json:"exchange_timezone"`
	// LastTradeTime is the last trade time, as sent by IBKR.
	LastTradeTime json.Number `json:"last_trade_time"`
	// MarketName is the market identifier.
	MarketName string `json:"market_name"`
	// MarketRulesLink links to the published market rules.
	MarketRulesLink string `json:"market_rules_link"`
	// MeasuredPeriod is the settlement measurement window, e.g. 1D.
	MeasuredPeriod string `json:"measured_period"`
	// Payout is the payout ratio or amount, as IBKR sends it.
	Payout string `json:"payout"`
	// PayoutTime is the payout time, as sent by IBKR.
	PayoutTime json.Number `json:"payout_time"`
	// PriceIncrement is the minimum price increment.
	PriceIncrement string `json:"price_increment"`
	// ProductCode is the product code.
	ProductCode string `json:"product_code"`
	// ReleaseTime is the release time, as sent by IBKR.
	ReleaseTime json.Number `json:"release_time"`
	// SourceAgency is the agency reporting the outcome.
	SourceAgency string `json:"source_agency"`
	// Threshold is the outcome threshold, if the contract has one.
	Threshold string `json:"threshold"`
}

// ForecastTradingTime is one trading block within a schedule.
type ForecastTradingTime struct {
	// Close is the closing time of the session.
	Close string `json:"close"`
	// Open is the opening time of the session.
	Open string `json:"open"`
}

// ForecastTradingSchedule is the trading schedule for a single date.
type ForecastTradingSchedule struct {
	// DayOfWeek is the date the schedule applies to.
	DayOfWeek string `json:"day_of_week"`
	// TradingTimes are the trading blocks for the date. A contract with an
	// intraday closure returns more than one.
	TradingTimes []ForecastTradingTime `json:"trading_times"`
}

// ForecastSchedule describes when an event contract trades.
type ForecastSchedule struct {
	// Timezone is the exchange timezone of the event contract.
	Timezone string `json:"timezone"`
	// TradingSchedules is one entry per date the contract trades.
	TradingSchedules []ForecastTradingSchedule `json:"trading_schedules"`
}

// ForecastCategories returns the event-contract category tree.
//
// The payload is returned as raw JSON rather than a typed model. The tree is an
// object keyed by category id, and the generated response type flattens that map
// into a single struct with name, parent_id and markets fields, so a model built
// from it would describe a shape IBKR does not send. ScannerParameters is
// passthrough for the same reason.
func (m *ForecastManager) ForecastCategories(ctx context.Context) (json.RawMessage, error) {
	const op = "Forecast.GetForecastCategories"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetForecastCategories(ctx)
	})
	if err != nil {
		return nil, err
	}
	var out json.RawMessage
	if err := decodeJSON(resp, op, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ForecastContract returns the details of one event contract.
func (m *ForecastManager) ForecastContract(ctx context.Context, conid ConID) (*ForecastDetails, error) {
	const op = "Forecast.GetForecastContract"
	params := &client.GetForecastContractParams{Conid: strconv.Itoa(int(conid))}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetForecastContract(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &ForecastDetails{
		Category:        rawToString(raw, "category"),
		ConidNo:         json.Number(rawToString(raw, "conid_no")),
		ConidYes:        json.Number(rawToString(raw, "conid_yes")),
		Exchange:        rawToString(raw, "exchange"),
		Expiration:      rawToString(raw, "expiration"),
		LogoCategory:    rawToString(raw, "logo_category"),
		MarketName:      rawToString(raw, "market_name"),
		MeasuredPeriod:  rawToString(raw, "measured_period"),
		Payout:          rawToString(raw, "payout"),
		Question:        rawToString(raw, "question"),
		Side:            rawToString(raw, "side"),
		Strike:          json.Number(rawToString(raw, "strike")),
		StrikeLabel:     rawToString(raw, "strike_label"),
		Symbol:          rawToString(raw, "symbol"),
		UnderlyingConid: json.Number(rawToString(raw, "underlying_conid")),
	}, nil
}

// ForecastMarkets returns the contracts available for one forecast market.
// An empty exchange omits the parameter, letting IBKR determine it.
func (m *ForecastManager) ForecastMarkets(ctx context.Context, underlyingConid ConID, exchange string) (*ForecastMarket, error) {
	const op = "Forecast.GetForecastMarkets"
	params := &client.GetForecastMarketsParams{UnderlyingConid: strconv.Itoa(int(underlyingConid))}
	if exchange != "" {
		params.Exchange = &exchange
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetForecastMarkets(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := &ForecastMarket{
		Exchange:              rawToString(raw, "exchange"),
		ExcludeHistoricalData: rawToBool(raw, "exclude_historical_data"),
		LogoCategory:          rawToString(raw, "logo_category"),
		MarketName:            rawToString(raw, "market_name"),
		Payout:                rawToString(raw, "payout"),
		Symbol:                rawToString(raw, "symbol"),
	}
	// contracts is a nested array, so it is decoded whole rather than key by key.
	if b, ok := raw["contracts"]; ok {
		if err := decodeJSONBytes(b, op, &out.Contracts); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// ForecastRules returns the trading rules for one event contract.
func (m *ForecastManager) ForecastRules(ctx context.Context, conid ConID) (*ForecastRules, error) {
	const op = "Forecast.GetForecastRules"
	params := &client.GetForecastRulesParams{Conid: strconv.Itoa(int(conid))}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetForecastRules(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &ForecastRules{
		AssetClass:            rawToString(raw, "asset_class"),
		DataAndResolutionLink: rawToString(raw, "data_and_resolution_link"),
		Description:           rawToString(raw, "description"),
		ExchangeTimezone:      rawToString(raw, "exchange_timezone"),
		LastTradeTime:         json.Number(rawToString(raw, "last_trade_time")),
		MarketName:            rawToString(raw, "market_name"),
		MarketRulesLink:       rawToString(raw, "market_rules_link"),
		MeasuredPeriod:        rawToString(raw, "measured_period"),
		Payout:                rawToString(raw, "payout"),
		PayoutTime:            json.Number(rawToString(raw, "payout_time")),
		PriceIncrement:        rawToString(raw, "price_increment"),
		ProductCode:           rawToString(raw, "product_code"),
		ReleaseTime:           json.Number(rawToString(raw, "release_time")),
		SourceAgency:          rawToString(raw, "source_agency"),
		Threshold:             rawToString(raw, "threshold"),
	}, nil
}

// ForecastSchedule returns the trading schedule for one event contract.
func (m *ForecastManager) ForecastSchedule(ctx context.Context, conid ConID) (*ForecastSchedule, error) {
	const op = "Forecast.GetForecastSchedule"
	params := &client.GetForecastScheduleParams{Conid: strconv.Itoa(int(conid))}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetForecastSchedule(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := &ForecastSchedule{Timezone: rawToString(raw, "timezone")}
	// trading_schedules is a nested array, so it is decoded whole rather than key by key.
	if b, ok := raw["trading_schedules"]; ok {
		if err := decodeJSONBytes(b, op, &out.TradingSchedules); err != nil {
			return nil, err
		}
	}
	return out, nil
}
