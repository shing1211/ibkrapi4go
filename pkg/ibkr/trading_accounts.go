// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/shing1211/ibkrapi4go/client"
)

// TradingAccountManager exposes trading-account operations. It is safe for
// concurrent use.
type TradingAccountManager struct {
	client *Client
}

// AccountOwner represents the owner of an account.
type AccountOwner struct {
	// Name is the owner's legal name.
	Name string `json:"name"`
	// OwnerType is the type of ownership.
	OwnerType string `json:"ownerType"`
}

// DynamicAccount is a dynamically-resolved account.
type DynamicAccount struct {
	// AccountID is the account identifier.
	AccountID string `json:"accountId"`
	// AccountTitle is the account's legal title.
	AccountTitle string `json:"accountTitle"`
}

// FundSummary holds available fund information.
type FundSummary struct {
	// AccountID is the account identifier.
	AccountID string `json:"accountId"`
	// BuyingPower is the buying power available.
	BuyingPower string `json:"buyingPower"`
	// AvailableFunds is the available funds.
	AvailableFunds string `json:"availableFunds"`
	// NetLiquidationValue is the net liquidation value.
	NetLiquidationValue string `json:"netLiquidationValue"`
}

// BalanceSummary holds balance information for an account.
type BalanceSummary struct {
	// AccountID is the account identifier.
	AccountID string `json:"accountId"`
	// Balance is the total account balance.
	Balance string `json:"balance"`
	// SettledCash is the settled cash.
	SettledCash string `json:"settledCash"`
	// AccruedInterest is the accrued interest.
	AccruedInterest string `json:"accruedInterest"`
}

// MarginSummary holds margin information for an account.
type MarginSummary struct {
	// AccountID is the account identifier.
	AccountID string `json:"accountId"`
	// InitialMargin is the initial margin requirement.
	InitialMargin string `json:"initialMargin"`
	// MaintenanceMargin is the maintenance margin requirement.
	MaintenanceMargin string `json:"maintenanceMargin"`
	// ExcessLiquidity is the excess liquidity.
	ExcessLiquidity string `json:"excessLiquidity"`
}

// AccountMarketSummary holds market value information for an account.
type AccountMarketSummary struct {
	// AccountID is the account identifier.
	AccountID string `json:"accountId"`
	// Stock is the total stock market value.
	Stock string `json:"stock"`
	// Futures is the total futures market value.
	Futures string `json:"futures"`
	// StocksAndFutures is the combined market value.
	StocksAndFutures string `json:"stocksAndFutures"`
}

// GetAccountOwners returns the signatures and owners for an account.
func (m *TradingAccountManager) GetAccountOwners(ctx context.Context, accountID AccountID) (json.RawMessage, error) {
	const op = "TradingAccount.GetAccountOwners"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetAccountOwners(ctx, string(accountID))
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

// SetActiveAccount sets the active account for the session.
func (m *TradingAccountManager) SetActiveAccount(ctx context.Context, accountID AccountID) error {
	const op = "TradingAccount.SetActiveAccount"
	body := client.SetActiveAccountJSONRequestBody(client.SetActiveAccountJSONBody{
		AcctId: strPtr(string(accountID)),
	})
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.SetActiveAccount(ctx, body)
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// GetDynamicAccounts searches for accounts by pattern.
func (m *TradingAccountManager) GetDynamicAccounts(ctx context.Context, searchPattern string) ([]DynamicAccount, error) {
	const op = "TradingAccount.GetDynamicAccounts"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetDynamicAccounts(ctx, searchPattern)
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]DynamicAccount, 0, len(raw))
	for _, item := range raw {
		out = append(out, DynamicAccount{
			AccountID:    rawToString(item, "accountId"),
			AccountTitle: rawToString(item, "accountTitle"),
		})
	}
	return out, nil
}

// GetFundSummary returns available fund information for an account.
func (m *TradingAccountManager) GetFundSummary(ctx context.Context, accountID AccountID) (*FundSummary, error) {
	const op = "TradingAccount.GetFundSummary"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetFundSummary(ctx, string(accountID))
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &FundSummary{
		AccountID:           string(accountID),
		BuyingPower:         rawToString(raw, "buyingPower"),
		AvailableFunds:      rawToString(raw, "availableFunds"),
		NetLiquidationValue: rawToString(raw, "netLiquidationValue"),
	}, nil
}

// GetBalanceSummary returns balance information for an account.
func (m *TradingAccountManager) GetBalanceSummary(ctx context.Context, accountID AccountID) (*BalanceSummary, error) {
	const op = "TradingAccount.GetBalanceSummary"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetBalanceSummary(ctx, string(accountID))
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &BalanceSummary{
		AccountID:       string(accountID),
		Balance:         rawToString(raw, "balance"),
		SettledCash:     rawToString(raw, "settledCash"),
		AccruedInterest: rawToString(raw, "accruedInterest"),
	}, nil
}

// GetMarginSummary returns margin information for an account.
func (m *TradingAccountManager) GetMarginSummary(ctx context.Context, accountID AccountID) (*MarginSummary, error) {
	const op = "TradingAccount.GetMarginSummary"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetMarginSummary(ctx, string(accountID))
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &MarginSummary{
		AccountID:         string(accountID),
		InitialMargin:     rawToString(raw, "initialMargin"),
		MaintenanceMargin: rawToString(raw, "maintenanceMargin"),
		ExcessLiquidity:   rawToString(raw, "excessLiquidity"),
	}, nil
}

// GetAccountMarketSummary returns market value information for an account.
func (m *TradingAccountManager) GetAccountMarketSummary(ctx context.Context, accountID AccountID) (*AccountMarketSummary, error) {
	const op = "TradingAccount.GetAccountMarketSummary"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetAccountMarketSummary(ctx, string(accountID))
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &AccountMarketSummary{
		AccountID:        string(accountID),
		Stock:            rawToString(raw, "stock"),
		Futures:          rawToString(raw, "futures"),
		StocksAndFutures: rawToString(raw, "stocksAndFutures"),
	}, nil
}

// GetBrokerageAccounts returns the list of brokerage accounts for the session.
func (m *TradingAccountManager) GetBrokerageAccounts(ctx context.Context) (json.RawMessage, error) {
	const op = "TradingAccount.GetBrokerageAccounts"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetBrokerageAccounts(ctx)
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

// SetDynamicAccount sets the dynamic account for future requests.
func (m *TradingAccountManager) SetDynamicAccount(ctx context.Context, accountID AccountID) error {
	const op = "TradingAccount.SetDynamicAccount"
	body := client.SetDynamicAccountJSONRequestBody(client.SetDynamicAccountJSONBody{
		AcctId: string(accountID),
	})
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.SetDynamicAccount(ctx, body)
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// strPtr returns a pointer to s.
func strPtr(s string) *string { return &s }
