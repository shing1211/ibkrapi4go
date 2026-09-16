// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
)

type Account struct {
	ID           string `json:"id"`
	AccountAlias string `json:"alias"`
	BaseCurrency string `json:"baseCurrency"`
	Household    string `json:"household"`
}

type AccountWithDetails struct {
	Account
}

func (c *Client) Account(ctx context.Context, accountID string) (*AccountWithDetails, error) {
	resp, err := c.generated.GetAccountsDetailsWithResponse(ctx, accountID, nil)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil || resp.JSON200.Account == nil {
		return nil, &Error{Code: "account", Message: "unexpected response"}
	}
	a := resp.JSON200.Account
	return &AccountWithDetails{
		Account: Account{
			ID:           derefString(a.AccountId, ""),
			AccountAlias: derefString(a.AccountAlias, ""),
			BaseCurrency: derefString(a.BaseCurrency, ""),
			Household:    derefString(a.Household, ""),
		},
	}, nil
}
