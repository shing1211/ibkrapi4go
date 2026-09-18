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

// ForecastManager exposes event/forecast operations. It is safe for concurrent use.
type ForecastManager struct {
	client *Client
}

// ForecastCategory represents a forecast event category.
type ForecastCategory struct {
	// ID is the category identifier.
	ID string `json:"id"`
	// Name is the category name.
	Name string `json:"name"`
}

// ForecastContractDetails holds detailed information about a forecast contract.
type ForecastContractDetails struct {
	// ConID is the contract identifier.
	ConID ConID `json:"conId"`
	// Description is the contract description.
	Description string `json:"description"`
	// Category is the forecast category.
	Category string `json:"category"`
}

// ForecastMarket holds market information for a forecast contract.
type ForecastMarket struct {
	// ConID is the contract identifier.
	ConID ConID `json:"conId"`
	// Symbol is the market symbol.
	Symbol string `json:"symbol"`
	// Description is the market description.
	Description string `json:"description"`
}

// ForecastRule holds a forecast rule.
type ForecastRule struct {
	// RuleID is the rule identifier.
	RuleID string `json:"ruleId"`
	// Description is the rule description.
	Description string `json:"description"`
}

// ForecastSchedule holds a forecast schedule entry.
type ForecastSchedule struct {
	// ConID is the contract identifier.
	ConID ConID `json:"conId"`
	// ScheduleTime is the scheduled time.
	ScheduleTime string `json:"scheduleTime"`
}

// ForecastCategories returns the category tree for forecasts.
func (m *ForecastManager) ForecastCategories(ctx context.Context) ([]ForecastCategory, error) {
	const op = "Forecast.GetForecastCategories"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetForecastCategories(ctx)
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]ForecastCategory, 0, len(raw))
	for _, item := range raw {
		out = append(out, ForecastCategory{
			ID:   rawToString(item, "id"),
			Name: rawToString(item, "name"),
		})
	}
	return out, nil
}

// ForecastContract returns details for a forecast contract.
func (m *ForecastManager) ForecastContract(ctx context.Context, conid ConID) (*ForecastContractDetails, error) {
	const op = "Forecast.GetForecastContract"
	params := &client.GetForecastContractParams{
		Conid: strconv.Itoa(int(conid)),
	}
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
	return &ForecastContractDetails{
		ConID:       ConID(jsonNumberToInt(jsonNumber(raw["conId"]))),
		Description: rawToString(raw, "description"),
		Category:    rawToString(raw, "category"),
	}, nil
}

// ForecastMarkets returns available markets for a forecast contract.
func (m *ForecastManager) ForecastMarkets(ctx context.Context, underlyingConid ConID, exchange *string) ([]ForecastMarket, error) {
	const op = "Forecast.GetForecastMarkets"
	params := &client.GetForecastMarketsParams{
		UnderlyingConid: strconv.Itoa(int(underlyingConid)),
		Exchange:        exchange,
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetForecastMarkets(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]ForecastMarket, 0, len(raw))
	for _, item := range raw {
		out = append(out, ForecastMarket{
			ConID:       ConID(jsonNumberToInt(jsonNumber(item["conId"]))),
			Symbol:      rawToString(item, "symbol"),
			Description: rawToString(item, "description"),
		})
	}
	return out, nil
}

// ForecastRules returns the rules for a forecast contract.
func (m *ForecastManager) ForecastRules(ctx context.Context, conid ConID) ([]ForecastRule, error) {
	const op = "Forecast.GetForecastRules"
	params := &client.GetForecastRulesParams{
		Conid: strconv.Itoa(int(conid)),
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetForecastRules(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]ForecastRule, 0, len(raw))
	for _, item := range raw {
		out = append(out, ForecastRule{
			RuleID:      rawToString(item, "ruleId"),
			Description: rawToString(item, "description"),
		})
	}
	return out, nil
}

// ForecastSchedule returns the forecast schedule for a contract.
func (m *ForecastManager) ForecastSchedule(ctx context.Context, conid ConID) ([]ForecastSchedule, error) {
	const op = "Forecast.GetForecastSchedule"
	params := &client.GetForecastScheduleParams{
		Conid: strconv.Itoa(int(conid)),
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetForecastSchedule(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]ForecastSchedule, 0, len(raw))
	for _, item := range raw {
		out = append(out, ForecastSchedule{
			ConID:        ConID(jsonNumberToInt(jsonNumber(item["conId"]))),
			ScheduleTime: rawToString(item, "scheduleTime"),
		})
	}
	return out, nil
}
