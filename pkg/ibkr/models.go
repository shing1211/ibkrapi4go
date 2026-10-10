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
	Accounts []AccountID `json:"accounts"`
}

// ModelAccount holds account details for a model.
type ModelAccount struct {
	// AccountID is the account identifier.
	AccountID AccountID `json:"accountId"`
	// AccountAlias is the account alias.
	AccountAlias string `json:"accountAlias"`
}

// ModelPosition holds a position in a model portfolio.
type ModelPosition struct {
	// AccountID is the account identifier.
	AccountID AccountID `json:"accountId"`
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
	AccountIDs []AccountID `json:"accountIds"`
}

// FullMasterStatus reports whether the account is a full master.
type FullMasterStatus struct {
	// IsFullMaster is true when the account has full master permissions.
	IsFullMaster bool `json:"isFullMaster"`
	// ReqID echoes the request identifier.
	ReqID int64 `json:"reqID"`
	// SubscriptionStatus is the polling state: 0 means still computing.
	SubscriptionStatus int64 `json:"subscriptionStatus"`
}

// ModelCashTransfer is a single cash movement proposed by the cash analyzer.
type ModelCashTransfer struct {
	// Amount is the cash amount.
	Amount string `json:"amt"`
	// Currency is the ISO 4217 currency code.
	Currency string `json:"currency"`
}

// ModelCashAnalysis is the result of the model cash analyzer.
type ModelCashAnalysis struct {
	// ReqID echoes the request identifier.
	ReqID int64 `json:"reqID"`
	// SubscriptionStatus is the polling state: 0 means still computing.
	SubscriptionStatus int64 `json:"subscriptionStatus"`
	// CashTransfers are the proposed cash movements.
	CashTransfers []ModelCashTransfer `json:"cashTransfers"`
}

// RebalanceSubscription is the acknowledgement returned when a rebalance is
// accepted for asynchronous processing.
type RebalanceSubscription struct {
	// ReqID echoes the request identifier.
	ReqID string `json:"reqID"`
	// SubscriptionKey must be supplied on follow-up poll requests.
	SubscriptionKey string `json:"subscriptionKey"`
	// SubscriptionStatus is the polling state: 0 means still computing.
	SubscriptionStatus int64 `json:"subscriptionStatus"`
}

// RebalanceAllocation is a single instrument allocation in a rebalance preview.
type RebalanceAllocation struct {
	// ConID is the contract identifier.
	ConID ConID `json:"conId"`
	// Symbol is the ticker symbol.
	Symbol string `json:"symbol"`
	// Quantity is the target position size.
	Quantity string `json:"quantity"`
	// CashQty is the cash-denominated target.
	CashQty string `json:"cashQty"`
	// Price is the reference price.
	Price string `json:"price"`
}

// RebalancePreview is the allocation preview returned by a rebalance to
// specific targets.
type RebalancePreview struct {
	// ReqID echoes the request identifier.
	ReqID int64 `json:"reqID"`
	// SubscriptionStatus is the polling state: 0 means still computing.
	SubscriptionStatus int64 `json:"subscriptionStatus"`
	// Allocation lists the per-instrument target allocations.
	Allocation []RebalanceAllocation `json:"allocation"`
	// TotalBuy is the aggregate buy notional.
	TotalBuy string `json:"totalBuy"`
}

// ModelRebalanceTarget is a position target for a rebalance request.
type ModelRebalanceTarget struct {
	// ConID is the contract identifier to target.
	ConID ConID `json:"conid"`
	// Target is the allocation as a fraction of the model, e.g. "0.3".
	Target string `json:"target"`
	// Locked prevents the rebalancer from adjusting this target.
	Locked bool `json:"locked,omitempty"`
}

// ModelCashTarget is a currency cash target for a rebalance request.
type ModelCashTarget struct {
	// Currency is the ISO 4217 currency code.
	Currency string `json:"ccy"`
	// Target is the allocation as a fraction of the model, e.g. "0.1".
	Target string `json:"target"`
	// Locked prevents the rebalancer from adjusting this target.
	Locked bool `json:"locked,omitempty"`
}

// ModelInvestment is a single invest or divest instruction.
type ModelInvestment struct {
	// Model is the destination model.
	Model string `json:"model"`
	// AmountToInvest is the cash amount. Use a negative value to divest.
	AmountToInvest string `json:"amtToInvest"`
	// InvestCurrency is the ISO 4217 currency code of the amount.
	InvestCurrency string `json:"investCurrency,omitempty"`
}

// ModelDivestResult reports the outcome of an invest/divest request.
type ModelDivestResult struct {
	// ReqID echoes the request identifier.
	ReqID int64 `json:"reqID"`
	// SubscriptionStatus is the polling state: 0 means still computing.
	SubscriptionStatus int64 `json:"subscriptionStatus"`
	// Allocation lists the resulting per-instrument allocations.
	Allocation []RebalanceAllocation `json:"allocation"`
	// CashTransfers are the cash movements that were requested.
	CashTransfers []ModelCashTransfer `json:"cashTransfers"`
}

// ModelOrderInstruction is a single order to submit against a model portfolio.
// Price and quantity fields are strings to preserve decimal precision.
type ModelOrderInstruction struct {
	// ConID is the contract identifier to trade.
	ConID ConID `json:"conid"`
	// AccountID is the sub-account the order applies to.
	AccountID AccountID `json:"acctId,omitempty"`
	// Side is BUY or SELL.
	Side string `json:"side,omitempty"`
	// Quantity is the share quantity.
	Quantity string `json:"quantity,omitempty"`
	// CashQty is the cash-denominated quantity, mutually exclusive with Quantity.
	CashQty string `json:"cashQty,omitempty"`
	// OrderType is the IBKR order type, e.g. LMT or MKT.
	OrderType string `json:"orderType,omitempty"`
	// Price is the limit price.
	Price string `json:"price,omitempty"`
	// AuxPrice is the stop or trailing price.
	AuxPrice string `json:"auxPrice,omitempty"`
	// TrailingAmt is the trailing amount.
	TrailingAmt string `json:"trailingAmt,omitempty"`
	// TIF is the time-in-force, e.g. DAY or GTC.
	TIF string `json:"tif,omitempty"`
	// ClientOrderID must be unique per order and correlates downstream events.
	ClientOrderID string `json:"cOID,omitempty"`
	// IsSingleGroup submits the order set as a single OCA group.
	IsSingleGroup bool `json:"isSingleGroup,omitempty"`
}

// ModelOrderConfirmation is the per-order acknowledgement returned when
// submitting orders against a model portfolio.
type ModelOrderConfirmation struct {
	// OrderID is the broker-assigned order identifier.
	OrderID string `json:"orderId"`
	// OrderStatus is the order status reported by the broker.
	OrderStatus string `json:"orderStatus"`
	// ReplyID identifies the reply message.
	ReplyID string `json:"replyId"`
	// Messages holds the human-readable reply lines.
	Messages []string `json:"message"`
	// EncryptMessage is set when the reply is encrypted for the caller.
	EncryptMessage string `json:"encryptMessage"`
	// IsSuspended is true when the order could not be placed.
	IsSuspended bool `json:"isSuspended"`
}

// ModelPresets returns model portfolio presets.
func (m *ModelManager) ModelPresets(ctx context.Context, reqID int64) ([]ModelPreset, error) {
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
			Accounts: toStringIDSlice(rawToStringSlice(item, "accounts")),
		})
	}
	return out, nil
}

// SetModelPresets saves model portfolio presets.
func (m *ModelManager) SetModelPresets(ctx context.Context, presets []ModelPreset) error {
	const op = "Model.SetModelPresets"
	bodyJSON := make([]map[string]interface{}, len(presets))
	for i, p := range presets {
		accts := make([]string, len(p.Accounts))
		for j, a := range p.Accounts {
			accts[j] = string(a)
		}
		bodyJSON[i] = map[string]interface{}{
			"name":     p.Name,
			"accounts": accts,
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

// AccountsInModel returns accounts in a model.
func (m *ModelManager) AccountsInModel(ctx context.Context, modelName string) ([]ModelAccount, error) {
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
			AccountID:    AccountID(rawToString(item, "accountId")),
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

// InvestedAccountsInModel returns invested accounts and positions in a model.
func (m *ModelManager) InvestedAccountsInModel(ctx context.Context, modelName string) (json.RawMessage, error) {
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

// ModelsPager returns a paginated iterator over all model portfolio names.
//
//	pager := client.Models().ModelsPager(ctx, reqID)
//	for pager.Next(ctx) {
//	    name := pager.Value()
//	    // ...
//	}
//	if err := pager.Err(); err != nil { ... }
func (m *ModelManager) ModelsPager(_ context.Context, reqID int64) *Pager[string] {
	fetched := false
	return NewPager(func(ctx context.Context, page int) ([]string, error) {
		if fetched || page > 0 {
			return nil, nil
		}
		names, err := m.AllModels(ctx, reqID)
		if err != nil {
			return nil, err
		}
		fetched = true
		return names, nil
	})
}

// AllModels returns all model portfolios.
//
// Deprecated: Use ModelsPager instead for paginated iteration.
func (m *ModelManager) AllModels(ctx context.Context, reqID int64) ([]string, error) {
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

// AllModelPositions returns all positions across models.
func (m *ModelManager) AllModelPositions(ctx context.Context, modelName string) ([]ModelPosition, error) {
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
			AccountID: AccountID(rawToString(item, "accountId")),
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

// ModelSummarySingle returns a model summary.
func (m *ModelManager) ModelSummarySingle(ctx context.Context, modelName string) (*ModelSummary, error) {
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
		AccountIDs: toStringIDSlice(rawToStringSlice(raw, "accountIds")),
	}, nil
}

// IsFullMaster reports whether the account holds full master permissions.
func (m *ModelManager) IsFullMaster(ctx context.Context, reqID int64, subscriptionKey *string) (*FullMasterStatus, error) {
	const op = "Model.IsFullMaster"
	body := client.IsFullMasterJSONRequestBody(client.IsFullMasterJSONBody{
		ReqID:           reqID,
		SubscriptionKey: subscriptionKey,
	})
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.IsFullMaster(ctx, body)
	})
	if err != nil {
		return nil, err
	}
	var raw struct {
		IsFullMaster       *bool  `json:"isFullMaster,omitempty"`
		ReqID              *int64 `json:"reqID,omitempty"`
		SubscriptionStatus *int64 `json:"subscriptionStatus,omitempty"`
	}
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := &FullMasterStatus{}
	if raw.IsFullMaster != nil {
		out.IsFullMaster = *raw.IsFullMaster
	}
	if raw.ReqID != nil {
		out.ReqID = *raw.ReqID
	}
	if raw.SubscriptionStatus != nil {
		out.SubscriptionStatus = *raw.SubscriptionStatus
	}
	return out, nil
}

// ModelCashAnalyzer analyzes cash needs across a model portfolio.
func (m *ModelManager) ModelCashAnalyzer(ctx context.Context, reqID int64, subscriptionKey *string) (*ModelCashAnalysis, error) {
	const op = "Model.ModelCashAnalyzer"
	body := client.ModelCashAnalyzerJSONRequestBody(client.ModelCashAnalyzerJSONBody{
		ReqID:           reqID,
		SubscriptionKey: subscriptionKey,
	})
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.ModelCashAnalyzer(ctx, body)
	})
	if err != nil {
		return nil, err
	}
	var raw struct {
		ReqID              *int64 `json:"reqID,omitempty"`
		SubscriptionStatus *int64 `json:"subscriptionStatus,omitempty"`
		CashTransfers      []struct {
			Amt      string `json:"amt"`
			Currency string `json:"currency"`
		} `json:"cashTransfers"`
	}
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := &ModelCashAnalysis{
		CashTransfers: make([]ModelCashTransfer, 0, len(raw.CashTransfers)),
	}
	if raw.ReqID != nil {
		out.ReqID = *raw.ReqID
	}
	if raw.SubscriptionStatus != nil {
		out.SubscriptionStatus = *raw.SubscriptionStatus
	}
	for _, t := range raw.CashTransfers {
		out.CashTransfers = append(out.CashTransfers, ModelCashTransfer{
			Amount:   t.Amt,
			Currency: t.Currency,
		})
	}
	return out, nil
}

// RebalanceToExistingTargets rebalances a model toward its stored targets.
func (m *ModelManager) RebalanceToExistingTargets(ctx context.Context, modelName string, reqID int64, subscriptionKey *string) (*RebalanceSubscription, error) {
	const op = "Model.RebalanceToExistingTargets"
	body := client.RebalanceToExistingTargetsJSONRequestBody(client.RebalanceToExistingTargetsJSONBody{
		Model:           modelName,
		ReqID:           reqID,
		SubscriptionKey: subscriptionKey,
	})
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.RebalanceToExistingTargets(ctx, body)
	})
	if err != nil {
		return nil, err
	}
	return decodeRebalanceSubscription(resp, op)
}

// RebalanceToNewTargets rebalances a model toward a full replacement set of
// cash and position targets.
func (m *ModelManager) RebalanceToNewTargets(ctx context.Context, modelName string, reqID int64, cashTargets []ModelCashTarget, positionTargets []ModelRebalanceTarget, subscriptionKey *string) (*RebalanceSubscription, error) {
	const op = "Model.RebalanceToNewTargets"
	cash := make([]map[string]interface{}, 0, len(cashTargets))
	for _, t := range cashTargets {
		cash = append(cash, map[string]interface{}{
			"ccy":    t.Currency,
			"target": t.Target,
			"locked": t.Locked,
		})
	}
	positions := make([]map[string]interface{}, 0, len(positionTargets))
	for _, t := range positionTargets {
		positions = append(positions, map[string]interface{}{
			"conid":  int64(t.ConID),
			"target": t.Target,
			"locked": t.Locked,
		})
	}
	bodyJSON := map[string]interface{}{
		"model":           modelName,
		"reqID":           reqID,
		"cashTargets":     cash,
		"positionTargets": positions,
	}
	if subscriptionKey != nil {
		bodyJSON["subscriptionKey"] = *subscriptionKey
	}
	resp, err := m.postModelJSON(ctx, op, "RebalanceToNewTargets", bodyJSON)
	if err != nil {
		return nil, err
	}
	return decodeRebalanceSubscription(resp, op)
}

// RebalanceToSpecificTargets rebalances only the listed instruments toward the
// supplied targets, preserving the rest of the model.
func (m *ModelManager) RebalanceToSpecificTargets(ctx context.Context, modelName string, reqID int64, positionTargets []ModelRebalanceTarget, subscriptionKey *string) (*RebalancePreview, error) {
	const op = "Model.RebalanceToSpecificTargets"
	positions := make([]map[string]interface{}, 0, len(positionTargets))
	for _, t := range positionTargets {
		positions = append(positions, map[string]interface{}{
			"conid":  int64(t.ConID),
			"target": t.Target,
			"locked": t.Locked,
		})
	}
	bodyJSON := map[string]interface{}{
		"model":           modelName,
		"reqID":           reqID,
		"positionTargets": positions,
	}
	if subscriptionKey != nil {
		bodyJSON["subscriptionKey"] = *subscriptionKey
	}
	resp, err := m.postModelJSON(ctx, op, "RebalanceToSpecificTargets", bodyJSON)
	if err != nil {
		return nil, err
	}
	var raw struct {
		ReqID              *int64 `json:"reqID,omitempty"`
		SubscriptionStatus *int64 `json:"subscriptionStatus,omitempty"`
		Allocation         []struct {
			ConID    *int64 `json:"conId,omitempty"`
			Symbol   string `json:"symbol"`
			Quantity string `json:"quantity"`
			CashQty  string `json:"cashQty"`
			Price    string `json:"price"`
		} `json:"allocation"`
		TotalBuy string `json:"totalBuy"`
	}
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := &RebalancePreview{
		TotalBuy:   raw.TotalBuy,
		Allocation: make([]RebalanceAllocation, 0, len(raw.Allocation)),
	}
	if raw.ReqID != nil {
		out.ReqID = *raw.ReqID
	}
	if raw.SubscriptionStatus != nil {
		out.SubscriptionStatus = *raw.SubscriptionStatus
	}
	for _, a := range raw.Allocation {
		item := RebalanceAllocation{
			Symbol:   a.Symbol,
			Quantity: a.Quantity,
			CashQty:  a.CashQty,
			Price:    a.Price,
		}
		if a.ConID != nil {
			item.ConID = ConID(*a.ConID)
		}
		out.Allocation = append(out.Allocation, item)
	}
	return out, nil
}

// TwsInvestDivest invests cash into, or divests from, model portfolios.
// Exactly one of account, accountList, or group may be supplied. A negative
// amount divests.
func (m *ModelManager) TwsInvestDivest(ctx context.Context, reqID int64, investments []ModelInvestment, account *string, accountList []string, group *string, subscriptionKey *string) (*ModelDivestResult, error) {
	const op = "Model.TwsInvestDivest"
	models := make([]map[string]interface{}, 0, len(investments))
	for _, in := range investments {
		models = append(models, map[string]interface{}{
			"model":          in.Model,
			"amtToInvest":    in.AmountToInvest,
			"investCurrency": in.InvestCurrency,
		})
	}
	bodyJSON := map[string]interface{}{
		"reqID":     reqID,
		"modelList": models,
	}
	if account != nil {
		bodyJSON["account"] = *account
	}
	if len(accountList) > 0 {
		bodyJSON["accountList"] = accountList
	}
	if group != nil {
		bodyJSON["group"] = *group
	}
	if subscriptionKey != nil {
		bodyJSON["subscriptionKey"] = *subscriptionKey
	}
	resp, err := m.postModelJSON(ctx, op, "TwsInvestDivest", bodyJSON)
	if err != nil {
		return nil, err
	}
	var raw struct {
		ReqID              *int64 `json:"reqID,omitempty"`
		SubscriptionStatus *int64 `json:"subscriptionStatus,omitempty"`
		Allocation         []struct {
			ConID    *int64 `json:"conId,omitempty"`
			Symbol   string `json:"symbol"`
			Quantity string `json:"quantity"`
			CashQty  string `json:"cashQty"`
			Price    string `json:"price"`
		} `json:"allocation"`
		CashTransfers []struct {
			Amt      string `json:"amt"`
			Currency string `json:"currency"`
		} `json:"cashTransfers"`
	}
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := &ModelDivestResult{
		Allocation:    make([]RebalanceAllocation, 0, len(raw.Allocation)),
		CashTransfers: make([]ModelCashTransfer, 0, len(raw.CashTransfers)),
	}
	if raw.ReqID != nil {
		out.ReqID = *raw.ReqID
	}
	if raw.SubscriptionStatus != nil {
		out.SubscriptionStatus = *raw.SubscriptionStatus
	}
	for _, a := range raw.Allocation {
		item := RebalanceAllocation{
			Symbol:   a.Symbol,
			Quantity: a.Quantity,
			CashQty:  a.CashQty,
			Price:    a.Price,
		}
		if a.ConID != nil {
			item.ConID = ConID(*a.ConID)
		}
		out.Allocation = append(out.Allocation, item)
	}
	for _, t := range raw.CashTransfers {
		out.CashTransfers = append(out.CashTransfers, ModelCashTransfer{
			Amount:   t.Amt,
			Currency: t.Currency,
		})
	}
	return out, nil
}

// SubmitModelPortfolioOrder submits orders against a model portfolio.
func (m *ModelManager) SubmitModelPortfolioOrder(ctx context.Context, modelCode string, orders []ModelOrderInstruction) ([]ModelOrderConfirmation, error) {
	const op = "Model.SubmitModelPortfolioOrder"
	payload := make([]map[string]interface{}, 0, len(orders))
	for _, o := range orders {
		item := map[string]interface{}{
			"conid": int64(o.ConID),
		}
		putIfSet(item, "acctId", string(o.AccountID))
		putIfSet(item, "side", o.Side)
		putIfSet(item, "quantity", o.Quantity)
		putIfSet(item, "cashQty", o.CashQty)
		putIfSet(item, "orderType", o.OrderType)
		putIfSet(item, "price", o.Price)
		putIfSet(item, "auxPrice", o.AuxPrice)
		putIfSet(item, "trailingAmt", o.TrailingAmt)
		putIfSet(item, "tif", o.TIF)
		putIfSet(item, "cOID", o.ClientOrderID)
		if o.IsSingleGroup {
			item["isSingleGroup"] = true
		}
		payload = append(payload, item)
	}
	body, err := json.Marshal(map[string]interface{}{"orders": payload})
	if err != nil {
		return nil, &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.SubmitModelPortfolioOrderWithBody(ctx, modelCode,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return nil, err
	}
	var raw []struct {
		OrderID        string   `json:"order_id"`
		OrderStatus    string   `json:"order_status"`
		ID             string   `json:"id"`
		Message        []string `json:"message"`
		EncryptMessage string   `json:"encrypt_message"`
		IsSuspended    bool     `json:"isSuspended"`
	}
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]ModelOrderConfirmation, 0, len(raw))
	for _, c := range raw {
		out = append(out, ModelOrderConfirmation{
			OrderID:        c.OrderID,
			OrderStatus:    c.OrderStatus,
			ReplyID:        c.ID,
			Messages:       c.Message,
			EncryptMessage: c.EncryptMessage,
			IsSuspended:    c.IsSuspended,
		})
	}
	return out, nil
}

// postModelJSON marshals bodyJSON and posts it to a model operation that has
// no nameable generated request-body type because it embeds anonymous structs.
func (m *ModelManager) postModelJSON(ctx context.Context, op string, method string, bodyJSON map[string]interface{}) (*http.Response, error) {
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return nil, &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	var call func(context.Context) (*http.Response, error)
	switch method {
	case "RebalanceToNewTargets":
		call = func(c context.Context) (*http.Response, error) {
			return m.client.generated.RebalanceToNewTargetsWithBody(c, "application/json", bytes.NewReader(body))
		}
	case "RebalanceToSpecificTargets":
		call = func(c context.Context) (*http.Response, error) {
			return m.client.generated.RebalanceToSpecificTargetsWithBody(c, "application/json", bytes.NewReader(body))
		}
	case "TwsInvestDivest":
		call = func(c context.Context) (*http.Response, error) {
			return m.client.generated.TwsInvestDivestWithBody(c, "application/json", bytes.NewReader(body))
		}
	default:
		return nil, &Error{Op: op, Message: "unsupported model operation: " + method}
	}
	return m.client.netDo(ctx, op, func() (*http.Response, error) {
		return call(ctx)
	})
}

func decodeRebalanceSubscription(resp *http.Response, op string) (*RebalanceSubscription, error) {
	var raw struct {
		ReqID              string `json:"reqID"`
		SubscriptionKey    string `json:"subscriptionKey"`
		SubscriptionStatus *int64 `json:"subscriptionStatus,omitempty"`
	}
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := &RebalanceSubscription{
		ReqID:           raw.ReqID,
		SubscriptionKey: raw.SubscriptionKey,
	}
	if raw.SubscriptionStatus != nil {
		out.SubscriptionStatus = *raw.SubscriptionStatus
	}
	return out, nil
}

func putIfSet(m map[string]interface{}, key string, value string) {
	if value != "" {
		m[key] = value
	}
}
