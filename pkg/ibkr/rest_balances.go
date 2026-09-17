// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"encoding/json"

	"github.com/shing1211/ibkrapi4go/client"
)

type RESTBalances struct {
	surface *RESTSurface
}

func (s *RESTSurface) Balances() *RESTBalances { return &RESTBalances{surface: s} }

func (m *RESTBalances) Query(ctx context.Context, accountID AccountID, currency string) ([]CashBalanceDetail, error) {
	const op = "Balances.Query"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	var instr client.CreateBalancesQueryJSONBody_Instruction
	_ = instr.FromQueryWithdrawableCashEquity(client.QueryWithdrawableCashEquity{
		AccountId:           string(accountID),
		ClientInstructionId: 0,
		Currency:            currency,
	})
	payload := client.CreateBalancesQueryJSONRequestBody{
		Instruction:     instr,
		InstructionType: client.CreateBalancesQueryJSONBodyInstructionTypeQUERYWITHDRAWABLECASHEQUITY,
	}
	resp, err := m.surface.generated.CreateBalancesQueryWithResponse(ctx, payload)
	if err != nil {
		return nil, wrapOp(op, err)
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, m.surface.owner.errorFrom(resp.HTTPResponse, op)
	}
	var raw balancesRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		return nil, &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
	}
	return raw.toPublic(), nil
}

type CashBalanceDetail struct {
	AccountID   AccountID
	Currency    string
	CashBalance string
}

type balancesRaw struct {
	Balanaces []balanceRaw `json:"balanaces,omitempty"`
}

type balanceRaw struct {
	AccountId *string `json:"accountId,omitempty"`
	Currency  *string `json:"currency,omitempty"`
	Amount    *string `json:"amount,omitempty"`
}

func (r *balancesRaw) toPublic() []CashBalanceDetail {
	if r.Balanaces == nil {
		return nil
	}
	out := make([]CashBalanceDetail, 0, len(r.Balanaces))
	for _, b := range r.Balanaces {
		out = append(out, CashBalanceDetail{
			AccountID:   AccountID(strPtrVal(b.AccountId)),
			Currency:    strPtrVal(b.Currency),
			CashBalance: strPtrVal(b.Amount),
		})
	}
	return out
}
