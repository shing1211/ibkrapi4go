// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"encoding/json"
	"net/http"
)

// PortfolioAccount describes a brokerage or sub-account.
type PortfolioAccount struct {
	// AccountID is the account identifier.
	AccountID AccountID
	// AccountTitle is the account's legal title.
	AccountTitle string
	// AccountAlias is the user-defined alias.
	AccountAlias string
	// Currency is the account base currency.
	Currency string
	// AccountStatus is the account's current status.
	AccountStatus string
	// DisplayName is the account's display name.
	DisplayName string
}

// Position is a holding in a single instrument. Monetary values are decimal
// strings exactly as returned by IBKR (ADR 0008).
type Position struct {
	// AccountID is the owning account.
	AccountID AccountID
	// ConID is the contract identifier.
	ConID ConID
	// ContractDesc is the contract description.
	ContractDesc string
	// AssetClass is the instrument asset class (e.g. STK, OPT, FUT).
	AssetClass string
	// Currency is the instrument's currency.
	Currency string
	// Quantity is the position size; may be fractional.
	Quantity string
	// AvgCost is the average cost.
	AvgCost string
	// AvgPrice is the average price.
	AvgPrice string
	// MktPrice is the current market price.
	MktPrice string
	// MktValue is the current market value.
	MktValue string
	// RealizedPnl is the instrument realized P&L.
	RealizedPnl string
	// UnrealizedPnl is the instrument unrealized P&L.
	UnrealizedPnl string
	// Model is the model portfolio name, if any.
	Model string
}

// LedgerCurrency is the ledger balance for one currency.
type LedgerCurrency struct {
	// AccountCode is the account id the ledger belongs to.
	AccountCode string
	// Currency is the balance currency.
	Currency string
	// CashBalance is the cash balance.
	CashBalance string
	// SettledCash is the settled cash balance.
	SettledCash string
	// NetLiquidationValue is the net liquidation value.
	NetLiquidationValue string
	// StockMarketValue is the stock market value.
	StockMarketValue string
	// StockOptionMarketValue is the stock-option market value.
	StockOptionMarketValue string
	// UnrealizedPnl is the unrealized P&L.
	UnrealizedPnl string
	// RealizedPnl is the realized P&L.
	RealizedPnl string
}

// SummaryValue is one key of a portfolio summary.
type SummaryValue struct {
	// Amount is the numerical value as a decimal string.
	Amount string
	// Currency is the currency the amount is denominated in.
	Currency string
	// Value is the string form of the value, when the field is non-numerical.
	Value string
	// IsNull reports whether the value does not exist (vs. being zero).
	IsNull bool
}

// PortfolioSummary maps each summary key (e.g. "netliquidation") to its value.
type PortfolioSummary map[string]SummaryValue

// AllocationBreakdown holds long and short values for one allocation dimension.
type AllocationBreakdown struct {
	// Long maps a category to its long value as a decimal string.
	Long map[string]string
	// Short maps a category to its short value as a decimal string.
	Short map[string]string
}

// Allocation maps an allocation dimension (e.g. "assetClass", "sector", "group")
// to its long/short breakdown.
type Allocation map[string]AllocationBreakdown

// PortfolioManager exposes positions, ledger, allocation, and summaries. It is
// safe for concurrent use.
type PortfolioManager struct {
	client *Client
}

// Accounts returns the tradable portfolio accounts.
func (m *PortfolioManager) Accounts(ctx context.Context) ([]PortfolioAccount, error) {
	const op = "Portfolio.Accounts"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetAllAccounts(ctx)
	})
	if err != nil {
		return nil, err
	}
	var raw []accountAttributesRaw
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return accountsFromRaw(raw), nil
}

// Subaccounts returns the account's subaccounts.
func (m *PortfolioManager) Subaccounts(ctx context.Context) ([]PortfolioAccount, error) {
	const op = "Portfolio.Subaccounts"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetAllSubaccounts(ctx)
	})
	if err != nil {
		return nil, err
	}
	var raw []accountAttributesRaw
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return accountsFromRaw(raw), nil
}

// Positions returns all positions for the account in a single (uncached) call.
func (m *PortfolioManager) Positions(ctx context.Context, account AccountID) ([]Position, error) {
	const op = "Portfolio.Positions"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetUncachedPositions(ctx, string(account), nil)
	})
	if err != nil {
		return nil, err
	}
	var raw []positionRaw
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return positionsFromRaw(account, raw), nil
}

// PositionsPaginated returns an iterator over the account's positions, fetching
// pages until an empty page is returned.
func (m *PortfolioManager) PositionsPaginated(ctx context.Context, account AccountID) *PositionIterator {
	fetch := func(fctx context.Context, pageID int) ([]Position, error) {
		const op = "Portfolio.PositionsPaginated"
		resp, err := m.client.netDo(fctx, op, func() (*http.Response, error) {
			return m.client.generated.GetPaginatedPositions(fctx, string(account), int64(pageID), nil)
		})
		if err != nil {
			return nil, err
		}
		var raw []positionRaw
		if err := decodeJSON(resp, op, &raw); err != nil {
			return nil, err
		}
		return positionsFromRaw(account, raw), nil
	}
	return newPositionIterator(fetch)
}

// Position returns the account's position in a single instrument.
func (m *PortfolioManager) Position(ctx context.Context, account AccountID, conid ConID) (*Position, error) {
	const op = "Portfolio.Position"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetPositionByConid(ctx, string(account), int64(conid))
	})
	if err != nil {
		return nil, err
	}
	var raw []positionRaw
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	positions := positionsFromRaw(account, raw)
	if len(positions) == 0 {
		return nil, &Error{Op: op, Message: "position not found", Err: ErrNotFound}
	}
	return &positions[0], nil
}

// Ledger returns the account ledger keyed by currency.
func (m *PortfolioManager) Ledger(ctx context.Context, account AccountID) (map[string]LedgerCurrency, error) {
	const op = "Portfolio.Ledger"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetPortfolioLedger(ctx, string(account))
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]ledgerRaw
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make(map[string]LedgerCurrency, len(raw))
	for cur, r := range raw {
		out[cur] = LedgerCurrency{
			AccountCode:            r.AcctCode,
			Currency:               r.Currency,
			CashBalance:            r.CashBalance.String(),
			SettledCash:            r.SettledCash.String(),
			NetLiquidationValue:    r.NetLiquidationValue.String(),
			StockMarketValue:       r.StockMarketValue.String(),
			StockOptionMarketValue: r.StockOptionMarketValue.String(),
			UnrealizedPnl:          r.UnrealizedPnl.String(),
			RealizedPnl:            r.RealizedPnl.String(),
		}
	}
	return out, nil
}

// Allocation returns the account's asset allocation.
func (m *PortfolioManager) Allocation(ctx context.Context, account AccountID) (Allocation, error) {
	const op = "Portfolio.Allocation"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetAssetAllocation(ctx, string(account), nil)
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]allocationBreakdownRaw
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make(Allocation, len(raw))
	for dim, b := range raw {
		out[dim] = AllocationBreakdown{
			Long:  numbersToStrings(b.Long),
			Short: numbersToStrings(b.Short),
		}
	}
	return out, nil
}

// Summary returns the account portfolio summary.
func (m *PortfolioManager) Summary(ctx context.Context, account AccountID) (PortfolioSummary, error) {
	const op = "Portfolio.Summary"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetPortfolioSummary(ctx, string(account))
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]summaryValueRaw
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make(PortfolioSummary, len(raw))
	for key, v := range raw {
		out[key] = SummaryValue{
			Amount:   rawScalarString(v.Amount),
			Currency: rawScalarString(v.Currency),
			Value:    v.Value,
			IsNull:   v.IsNull,
		}
	}
	return out, nil
}

// Meta returns the account attributes.
func (m *PortfolioManager) Meta(ctx context.Context, account AccountID) (*PortfolioAccount, error) {
	const op = "Portfolio.Meta"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetPortfolioMetadata(ctx, string(account))
	})
	if err != nil {
		return nil, err
	}
	var raw accountAttributesRaw
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	a := accountFromRaw(raw)
	return &a, nil
}

// Invalidate asks the gateway to refresh its position cache for the account.
func (m *PortfolioManager) Invalidate(ctx context.Context, account AccountID) error {
	const op = "Portfolio.Invalidate"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.InvalidatePositionCache(ctx, string(account))
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// --- raw adapters -----------------------------------------------------------

type accountAttributesRaw struct {
	AccountID     string `json:"accountId"`
	AccountTitle  string `json:"accountTitle"`
	AccountAlias  string `json:"accountAlias"`
	Currency      string `json:"currency"`
	AccountStatus string `json:"accountStatus"`
	DisplayName   string `json:"displayName"`
}

func accountFromRaw(r accountAttributesRaw) PortfolioAccount {
	return PortfolioAccount{
		AccountID:     AccountID(r.AccountID),
		AccountTitle:  r.AccountTitle,
		AccountAlias:  r.AccountAlias,
		Currency:      r.Currency,
		AccountStatus: r.AccountStatus,
		DisplayName:   r.DisplayName,
	}
}

func accountsFromRaw(raw []accountAttributesRaw) []PortfolioAccount {
	out := make([]PortfolioAccount, 0, len(raw))
	for _, r := range raw {
		out = append(out, accountFromRaw(r))
	}
	return out
}

type positionRaw struct {
	AcctID        string      `json:"acctId"`
	ConID         json.Number `json:"conid"`
	ContractDesc  string      `json:"contractDesc"`
	AssetClass    string      `json:"assetClass"`
	Currency      string      `json:"currency"`
	Position      json.Number `json:"position"`
	AvgCost       json.Number `json:"avgCost"`
	AvgPrice      json.Number `json:"avgPrice"`
	MktPrice      json.Number `json:"mktPrice"`
	MktValue      json.Number `json:"mktValue"`
	RealizedPnl   json.Number `json:"realizedPnl"`
	UnrealizedPnl json.Number `json:"unrealizedPnl"`
	Model         string      `json:"model"`
}

func positionsFromRaw(account AccountID, raw []positionRaw) []Position {
	out := make([]Position, 0, len(raw))
	for _, r := range raw {
		id := account
		if r.AcctID != "" {
			id = AccountID(r.AcctID)
		}
		out = append(out, Position{
			AccountID:     id,
			ConID:         ConID(jsonNumberToInt(r.ConID)),
			ContractDesc:  r.ContractDesc,
			AssetClass:    r.AssetClass,
			Currency:      r.Currency,
			Quantity:      r.Position.String(),
			AvgCost:       r.AvgCost.String(),
			AvgPrice:      r.AvgPrice.String(),
			MktPrice:      r.MktPrice.String(),
			MktValue:      r.MktValue.String(),
			RealizedPnl:   r.RealizedPnl.String(),
			UnrealizedPnl: r.UnrealizedPnl.String(),
			Model:         r.Model,
		})
	}
	return out
}

type ledgerRaw struct {
	AcctCode               string      `json:"acctcode"`
	Currency               string      `json:"currency"`
	CashBalance            json.Number `json:"cashbalance"`
	SettledCash            json.Number `json:"settledcash"`
	NetLiquidationValue    json.Number `json:"netliquidationvalue"`
	StockMarketValue       json.Number `json:"stockmarketvalue"`
	StockOptionMarketValue json.Number `json:"stockoptionmarketvalue"`
	UnrealizedPnl          json.Number `json:"unrealizedpnl"`
	RealizedPnl            json.Number `json:"realizedpnl"`
}

type allocationBreakdownRaw struct {
	Long  map[string]json.Number `json:"long"`
	Short map[string]json.Number `json:"short"`
}

type summaryValueRaw struct {
	Amount   json.RawMessage `json:"amount"`
	Currency json.RawMessage `json:"currency"`
	Value    string          `json:"value"`
	IsNull   bool            `json:"isNull"`
}

// numbersToStrings converts a JSON-number map to a decimal-string map.
func numbersToStrings(in map[string]json.Number) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v.String()
	}
	return out
}
