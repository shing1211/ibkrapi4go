// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/shing1211/ibkrapi4go/client"
)

// PerformanceManager exposes performance analyst operations. It is safe for
// concurrent use.
type PerformanceManager struct {
	client *Client
}

// PerformanceData holds performance analysis data.
type PerformanceData struct {
	// AccountID is the account identifier.
	AccountID AccountID `json:"accountId"`
	// Data holds the performance data as raw JSON.
	Data json.RawMessage `json:"data"`
}

// Transaction holds a transaction record.
type Transaction struct {
	// AccountID is the account identifier.
	AccountID AccountID `json:"accountId"`
	// ConID is the contract identifier.
	ConID ConID `json:"conId"`
	// Symbol is the ticker symbol.
	Symbol string `json:"symbol"`
	// Side is the buy/sell direction.
	Side string `json:"side"`
	// Quantity is the transaction quantity.
	Quantity string `json:"quantity"`
	// Price is the transaction price.
	Price string `json:"price"`
	// Amount is the transaction amount.
	Amount string `json:"amount"`
}

// CreateAllocationPA creates a performance allocation.
func (m *PerformanceManager) CreateAllocationPA(ctx context.Context, accountIDs []string, allocType string) error {
	const op = "Performance.CreateAllocation"
	bodyJSON := map[string]interface{}{
		"acctIds": accountIDs,
		"type":    allocType,
	}
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.CreateAllocationWithBody(ctx,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// PerformanceAllPeriods returns performance data for all periods.
func (m *PerformanceManager) PerformanceAllPeriods(ctx context.Context, accountIDs []string, param string) (json.RawMessage, error) {
	const op = "Performance.GetPerformanceAllPeriods"
	params := &client.GetPerformanceAllPeriodsParams{}
	if param != "" {
		params.Param = &param
	}
	acctIDs := accountIDs
	body := client.GetPerformanceAllPeriodsJSONRequestBody(client.GetPerformanceAllPeriodsJSONBody{
		AcctIds: &acctIDs,
	})
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetPerformanceAllPeriods(ctx, params, body)
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

// SinglePerformancePeriod returns performance data for a single period.
func (m *PerformanceManager) SinglePerformancePeriod(ctx context.Context, accountIDs []string, period string) (*PerformanceData, error) {
	const op = "Performance.GetSinglePerformancePeriod"
	bodyJSON := map[string]interface{}{
		"acctIds": accountIDs,
		"period":  period,
	}
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return nil, &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetSinglePerformancePeriodWithBody(ctx,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &PerformanceData{
		Data: json.RawMessage(body),
	}, nil
}

// TransactionsPager returns a paginated iterator over transactions for the
// given accounts.
//
//	pager := client.Performance().TransactionsPager(ctx, accountIDs, startDate, endDate)
//	for pager.Next(ctx) {
//	    tx := pager.Value()
//	    // ...
//	}
//	if err := pager.Err(); err != nil { ... }
func (m *PerformanceManager) TransactionsPager(_ context.Context, accountIDs []string, startDate, endDate string) *Pager[Transaction] {
	fetched := false
	return NewPager(func(ctx context.Context, page int) ([]Transaction, error) {
		if fetched || page > 0 {
			return nil, nil
		}
		txs, err := m.Transactions(ctx, accountIDs, startDate, endDate)
		if err != nil {
			return nil, err
		}
		fetched = true
		return txs, nil
	})
}

// Transactions returns transactions for the given accounts.
//
// Deprecated: Use TransactionsPager instead for paginated iteration.
func (m *PerformanceManager) Transactions(ctx context.Context, accountIDs []string, startDate, endDate string) ([]Transaction, error) {
	const op = "Performance.GetTransactions"
	bodyJSON := map[string]interface{}{
		"acctIds":   accountIDs,
		"startDate": startDate,
		"endDate":   endDate,
	}
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return nil, &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetTransactionsWithBody(ctx,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]Transaction, 0, len(raw))
	for _, item := range raw {
		out = append(out, Transaction{
			AccountID: AccountID(rawToString(item, "accountId")),
			ConID:     ConID(jsonNumberToInt(jsonNumber(item["conId"]))),
			Symbol:    rawToString(item, "symbol"),
			Side:      rawToString(item, "side"),
			Quantity:  rawToString(item, "quantity"),
			Price:     rawToString(item, "price"),
			Amount:    rawToString(item, "amount"),
		})
	}
	return out, nil
}

// conIDToString converts a ConID to string.
func conIDToString(c ConID) string {
	return strconv.Itoa(int(c))
}
