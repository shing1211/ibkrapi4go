// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// AccountID identifies a brokerage account.
type AccountID string

// Account is a tradable account and its alias.
type Account struct {
	// ID is the account identifier, e.g. "U1234567".
	ID AccountID
	// Alias is the human-readable account alias, if one is configured.
	Alias string
}

// CashBalance is the cash position for a single currency.
type CashBalance struct {
	// Currency is the currency the values represent.
	Currency string
	// Balance is the total available currency held in the account.
	Balance string
	// SettledCash is the available settled cash that can be withdrawn.
	SettledCash string
}

// AccountSummary is an at-a-glance view of account values. All monetary values
// are decimal strings exactly as returned by IBKR (see ADR 0008).
type AccountSummary struct {
	// AccountID is the account the summary belongs to.
	AccountID AccountID
	// AccountType describes the account type; empty for standard individual.
	AccountType string
	// Status is the non-tradeable status message, if any.
	Status string
	// SMA is the simple moving average of the account.
	SMA string
	// AccruedInterest is interest accruing since the previous coupon date.
	AccruedInterest string
	// AvailableFunds is equity available for trading.
	AvailableFunds string
	// Balance is the total account balance.
	Balance string
	// BuyingPower is the total buying power available.
	BuyingPower string
	// CashBalances holds per-currency balance information.
	CashBalances []CashBalance
	// EquityWithLoanValue is the equity with loan value.
	EquityWithLoanValue string
	// ExcessLiquidity is cash in excess of the usual requirement.
	ExcessLiquidity string
	// InitialMargin is the available initial margin.
	InitialMargin string
	// MaintenanceMargin is the available maintenance margin.
	MaintenanceMargin string
	// NetLiquidationValue is the net liquidation value.
	NetLiquidationValue string
	// RegTLoan is the US Regulation T loan value.
	RegTLoan string
	// RegTMargin is the US Regulation T initial margin requirement.
	RegTMargin string
	// SecuritiesGVP is the gross position value across securities.
	SecuritiesGVP string
	// TotalCashValue is cash recognized at trade time plus futures P&L.
	TotalCashValue string
}

// AccountPnLRow is the profit-and-loss row for one account.
type AccountPnLRow struct {
	// AccountKey is the raw partition key, e.g. "U1234567.Core".
	AccountKey string
	// AccountID is the account id parsed from AccountKey.
	AccountID AccountID
	// Dpl is the daily P&L.
	Dpl string
	// El is the excess liquidity.
	El string
	// Mv is the margin value.
	Mv string
	// Nl is the net liquidity.
	Nl string
	// RowType is the positional row type; always 1 for individual accounts.
	RowType string
	// Upl is the unrealized P&L.
	Upl string
}

// AccountManager exposes account listing, summaries, and P&L. It is safe for
// concurrent use.
type AccountManager struct {
	client *Client
}

// List returns all accessible account ids with their aliases.
func (m *AccountManager) List(ctx context.Context) ([]Account, error) {
	const op = "Account.List"
	resp, err := m.do(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetBrokerageAccounts(ctx)
	})
	if err != nil {
		return nil, err
	}
	var raw userAccountsRaw
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]Account, 0, len(raw.Accounts))
	for _, id := range raw.Accounts {
		out = append(out, Account{ID: AccountID(id), Alias: raw.Aliases[id]})
	}
	return out, nil
}

// Summary returns the account summary for the given account.
func (m *AccountManager) Summary(ctx context.Context, id AccountID) (*AccountSummary, error) {
	const op = "Account.Summary"
	resp, err := m.do(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetAccountSummary(ctx, string(id))
	})
	if err != nil {
		return nil, err
	}
	var raw accountSummaryRaw
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return raw.toPublic(id), nil
}

// PnL returns the partitioned profit-and-loss for the session's accounts.
func (m *AccountManager) PnL(ctx context.Context) ([]AccountPnLRow, error) {
	const op = "Account.PnL"
	resp, err := m.do(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetPnl(ctx)
	})
	if err != nil {
		return nil, err
	}
	var raw accountPnLRaw
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	rows := make([]AccountPnLRow, 0, len(raw.Upnl))
	for key, r := range raw.Upnl {
		rows = append(rows, AccountPnLRow{
			AccountKey: key,
			AccountID:  AccountID(strings.SplitN(key, ".", 2)[0]),
			Dpl:        r.Dpl.String(),
			El:         r.El.String(),
			Mv:         r.Mv.String(),
			Nl:         r.Nl.String(),
			RowType:    r.RowType.String(),
			Upl:        r.Upl.String(),
		})
	}
	return rows, nil
}

// do runs a raw generated call, guarding against a closed client and mapping
// non-2xx responses and transport errors to *Error. The returned response body
// is open and owned by the caller.
func (m *AccountManager) do(ctx context.Context, op string, fn func() (*http.Response, error)) (*http.Response, error) {
	if err := m.client.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := fn()
	if err != nil {
		return nil, wrapOp(op, err)
	}
	if e := m.client.errorFrom(resp, op); e != nil {
		resp.Body.Close()
		return nil, e
	}
	return resp, nil
}

// decodeJSON reads and closes resp.Body, decoding JSON with json.Number so
// decimal values are preserved as strings rather than binary floats (ADR 0008).
func decodeJSON(resp *http.Response, op string, v any) error {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &Error{Op: op, Message: "read response: " + err.Error(), Err: err}
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(v); err != nil {
		return &Error{Op: op, Message: "decode response: " + err.Error(), Err: err}
	}
	return nil
}

// derefString returns *s or def when s is nil.
func derefString(s *string, def string) string {
	if s == nil {
		return def
	}
	return *s
}

// userAccountsRaw mirrors the dynamic parts of the gateway accounts response.
type userAccountsRaw struct {
	Accounts []string          `json:"accounts"`
	Aliases  map[string]string `json:"aliases"`
}

// accountSummaryRaw decodes monetary fields as json.Number to preserve decimal
// precision.
type accountSummaryRaw struct {
	SMA             json.Number `json:"SMA"`
	AccountType     string      `json:"accountType"`
	AccruedInterest json.Number `json:"accruedInterest"`
	AvailableFunds  json.Number `json:"availableFunds"`
	Balance         json.Number `json:"balance"`
	BuyingPower     json.Number `json:"buyingPower"`
	CashBalances    []struct {
		Balance     json.Number `json:"balance"`
		Currency    string      `json:"currency"`
		SettledCash json.Number `json:"settledCash"`
	} `json:"cashBalances"`
	EquityWithLoanValue json.Number `json:"equityWithLoanValue"`
	ExcessLiquidity     json.Number `json:"excessLiquidity"`
	InitialMargin       json.Number `json:"initialMargin"`
	MaintenanceMargin   json.Number `json:"maintenanceMargin"`
	NetLiquidationValue json.Number `json:"netLiquidationValue"`
	RegTLoan            json.Number `json:"regTLoan"`
	RegTMargin          json.Number `json:"regTMargin"`
	SecuritiesGVP       json.Number `json:"securitiesGVP"`
	Status              string      `json:"status"`
	TotalCashValue      json.Number `json:"totalCashValue"`
}

func (r accountSummaryRaw) toPublic(id AccountID) *AccountSummary {
	s := &AccountSummary{
		AccountID:           id,
		AccountType:         r.AccountType,
		Status:              r.Status,
		SMA:                 r.SMA.String(),
		AccruedInterest:     r.AccruedInterest.String(),
		AvailableFunds:      r.AvailableFunds.String(),
		Balance:             r.Balance.String(),
		BuyingPower:         r.BuyingPower.String(),
		EquityWithLoanValue: r.EquityWithLoanValue.String(),
		ExcessLiquidity:     r.ExcessLiquidity.String(),
		InitialMargin:       r.InitialMargin.String(),
		MaintenanceMargin:   r.MaintenanceMargin.String(),
		NetLiquidationValue: r.NetLiquidationValue.String(),
		RegTLoan:            r.RegTLoan.String(),
		RegTMargin:          r.RegTMargin.String(),
		SecuritiesGVP:       r.SecuritiesGVP.String(),
		TotalCashValue:      r.TotalCashValue.String(),
	}
	for _, cb := range r.CashBalances {
		s.CashBalances = append(s.CashBalances, CashBalance{
			Currency:    cb.Currency,
			Balance:     cb.Balance.String(),
			SettledCash: cb.SettledCash.String(),
		})
	}
	return s
}

// accountPnLRaw mirrors the partitioned PnL response. The partition key is
// dynamic (e.g. "U1234567.Core"), so it is decoded as a map.
type accountPnLRaw struct {
	Upnl map[string]accountPnLRowRaw `json:"upnl"`
}

type accountPnLRowRaw struct {
	Dpl     json.Number `json:"dpl"`
	El      json.Number `json:"el"`
	Mv      json.Number `json:"mv"`
	Nl      json.Number `json:"nl"`
	RowType json.Number `json:"rowType"`
	Upl     json.Number `json:"upl"`
}
