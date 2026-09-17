// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

// ScannerManager exposes scanner operations. It is safe for concurrent use.
type ScannerManager struct {
	client *Client
}

// ScannerParameter holds a scanner parameter definition.
type ScannerParameter struct {
	// Type is the scanner type.
	Type string `json:"type"`
	// Name is the scanner name.
	Name string `json:"name"`
}

// ScannerResult holds a single scanner result.
type ScannerResult struct {
	// ConID is the contract identifier.
	ConID ConID `json:"conId"`
	// Symbol is the ticker symbol.
	Symbol string `json:"symbol"`
	// CompanyName is the issuer/company name.
	CompanyName string `json:"companyName"`
	// Exchange is the listing exchange.
	Exchange string `json:"exchange"`
	// SecType is the security type.
	SecType string `json:"secType"`
	// Distance is the scanner distance metric.
	Distance string `json:"distance"`
}

// GetScannerParameters returns the available scanner parameters.
func (m *ScannerManager) GetScannerParameters(ctx context.Context) (json.RawMessage, error) {
	const op = "Scanner.GetScannerParameters"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetScannerParameters(ctx)
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

// GetScannerResults runs a scanner with the given request body.
func (m *ScannerManager) GetScannerResults(ctx context.Context, scannerJSON map[string]interface{}) ([]ScannerResult, error) {
	const op = "Scanner.GetScannerResults"
	body, err := json.Marshal(scannerJSON)
	if err != nil {
		return nil, &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetScannerResultsWithBody(ctx,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]ScannerResult, 0, len(raw))
	for _, item := range raw {
		out = append(out, ScannerResult{
			ConID:       ConID(jsonNumberToInt(jsonNumber(item["conid"]))),
			Symbol:      rawToString(item, "symbol"),
			CompanyName: rawToString(item, "companyName"),
			Exchange:    rawToString(item, "exchange"),
			SecType:     rawToString(item, "secType"),
			Distance:    rawToString(item, "distance"),
		})
	}
	return out, nil
}
