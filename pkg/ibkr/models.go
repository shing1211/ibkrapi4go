// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/shing1211/ibkrapi4go/client"
)

// ModelManager exposes model portfolio operations. It is safe for concurrent use.
type ModelManager struct {
	client *Client
}

// ModelPreset holds a model portfolio preset.
type ModelPreset struct {
	// Name is the model name.
	Name string `json:"name"`
	// Accounts is the list of accounts in the model.
	Accounts []string `json:"accounts"`
}

// ModelAccount holds account details for a model.
type ModelAccount struct {
	// AccountID is the account identifier.
	AccountID string `json:"accountId"`
	// AccountAlias is the account alias.
	AccountAlias string `json:"accountAlias"`
}

// ModelPosition holds a position in a model portfolio.
type ModelPosition struct {
	// AccountID is the account identifier.
	AccountID string `json:"accountId"`
	// ConID is the contract identifier.
	ConID ConID `json:"conId"`
	// Symbol is the ticker symbol.
	Symbol string `json:"symbol"`
	// Position is the position size.
	Position string `json:"position"`
	// AvgCost is the average cost.
	AvgCost string `json:"avgCost"`
}

// ModelSummary holds a model portfolio summary.
type ModelSummary struct {
	// Name is the model name.
	Name string `json:"name"`
	// AccountIDs is the list of account IDs in the model.
	AccountIDs []string `json:"accountIds"`
}

// GetModelPresets returns model portfolio presets.
func (m *ModelManager) GetModelPresets(ctx context.Context, reqID int64) ([]ModelPreset, error) {
	const op = "Model.GetModelPresets"
	body := client.GetModelPresetsJSONRequestBody(client.GetModelPresetsJSONBody{
		ReqID: reqID,
	})
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetModelPresets(ctx, body)
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]ModelPreset, 0, len(raw))
	for _, item := range raw {
		out = append(out, ModelPreset{
			Name:     rawToString(item, "name"),
			Accounts: rawToStringSlice(item, "accounts"),
		})
	}
	return out, nil
}

// SetModelPresets saves model portfolio presets.
func (m *ModelManager) SetModelPresets(ctx context.Context, presets []ModelPreset) error {
	const op = "Model.SetModelPresets"
	bodyJSON := make([]map[string]interface{}, len(presets))
	for i, p := range presets {
		bodyJSON[i] = map[string]interface{}{
			"name":     p.Name,
			"accounts": p.Accounts,
		}
	}
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.SetModelPresetsWithBody(ctx,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// GetAccountsInModel returns accounts in a model.
func (m *ModelManager) GetAccountsInModel(ctx context.Context, modelName string) ([]ModelAccount, error) {
	const op = "Model.GetAccountsInModel"
	bodyJSON := map[string]interface{}{"model": modelName}
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return nil, &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetAccountsInModelWithBody(ctx,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]ModelAccount, 0, len(raw))
	for _, item := range raw {
		out = append(out, ModelAccount{
			AccountID:    rawToString(item, "accountId"),
			AccountAlias: rawToString(item, "accountAlias"),
		})
	}
	return out, nil
}

// SetAccountInvestmentInModel invests or divests in a model.
func (m *ModelManager) SetAccountInvestmentInModel(ctx context.Context, modelName string, amount string) error {
	const op = "Model.SetAccountInvestmentInModel"
	bodyJSON := map[string]interface{}{
		"model":  modelName,
		"amount": amount,
	}
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.SetAccountinvestmentInModelWithBody(ctx,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// GetInvestedAccountsInModel returns invested accounts and positions in a model.
func (m *ModelManager) GetInvestedAccountsInModel(ctx context.Context, modelName string) (json.RawMessage, error) {
	const op = "Model.GetInvestedAccountsInModel"
	bodyJSON := map[string]interface{}{"model": modelName}
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return nil, &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetInvestedAccountsInModelWithBody(ctx,
			"application/json", bytes.NewReader(body))
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

// GetAllModels returns all model portfolios.
func (m *ModelManager) GetAllModels(ctx context.Context, reqID int64) ([]string, error) {
	const op = "Model.GetAllModels"
	bodyJSON := map[string]interface{}{"reqID": reqID}
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return nil, &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetAllmodelsWithBody(ctx,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return nil, err
	}
	var raw []string
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// GetAllModelPositions returns all positions across models.
func (m *ModelManager) GetAllModelPositions(ctx context.Context, modelName string) ([]ModelPosition, error) {
	const op = "Model.GetAllModelPositions"
	bodyJSON := map[string]interface{}{"model": modelName}
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return nil, &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetAllModelPositionsWithBody(ctx,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]ModelPosition, 0, len(raw))
	for _, item := range raw {
		out = append(out, ModelPosition{
			AccountID: rawToString(item, "accountId"),
			ConID:     ConID(jsonNumberToInt(jsonNumber(item["conId"]))),
			Symbol:    rawToString(item, "symbol"),
			Position:  rawToString(item, "position"),
			AvgCost:   rawToString(item, "avgCost"),
		})
	}
	return out, nil
}

// SetModelTargetPositions sets target positions for a model.
func (m *ModelManager) SetModelTargetPositions(ctx context.Context, modelName string, positions []map[string]interface{}) error {
	const op = "Model.SetModelTargetPositions"
	bodyJSON := map[string]interface{}{
		"model":     modelName,
		"positions": positions,
	}
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.SetModelTargetPositionsWithBody(ctx,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// SubmitModelOrders submits orders for a model.
func (m *ModelManager) SubmitModelOrders(ctx context.Context, modelName string) error {
	const op = "Model.SubmitModelOrders"
	bodyJSON := map[string]interface{}{"model": modelName}
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.SubmitModelOrdersWithBody(ctx,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// GetModelSummarySingle returns a model summary.
func (m *ModelManager) GetModelSummarySingle(ctx context.Context, modelName string) (*ModelSummary, error) {
	const op = "Model.GetModelSummarySingle"
	bodyJSON := map[string]interface{}{"model": modelName}
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return nil, &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetModelSummarySingleWithBody(ctx,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &ModelSummary{
		Name:       rawToString(raw, "name"),
		AccountIDs: rawToStringSlice(raw, "accountIds"),
	}, nil
}
