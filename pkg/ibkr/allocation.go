// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

// AllocationManager exposes FA allocation operations. It is safe for concurrent use.
type AllocationManager struct {
	client *Client
}

// Subaccount holds a sub-account in an allocation group.
type Subaccount struct {
	// AccountID is the account identifier.
	AccountID string `json:"accountId"`
	// Amount is the allocation amount.
	Amount string `json:"amount"`
}

// AllocationGroup holds an allocation group definition.
type AllocationGroup struct {
	// Name is the group name.
	Name string `json:"name"`
	// IsHidden reports whether the group is hidden.
	IsHidden bool `json:"isHidden"`
	// Method is the allocation method.
	Method string `json:"method"`
	// Accounts is the list of sub-accounts in the group.
	Accounts []Subaccount `json:"accounts"`
}

// AllocationPreset holds a default allocation preset.
type AllocationPreset struct {
	// AccountID is the account identifier.
	AccountID string `json:"accountId"`
	// Percentage is the default allocation percentage.
	Percentage string `json:"percentage"`
}

// GetAllocatableSubaccounts returns accounts that can be allocated.
func (m *AllocationManager) GetAllocatableSubaccounts(ctx context.Context) ([]string, error) {
	const op = "Allocation.GetAllocatableSubaccounts"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetAllocatableSubaccounts(ctx)
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

// GetAllocationGroups returns all allocation groups.
func (m *AllocationManager) GetAllocationGroups(ctx context.Context) ([]AllocationGroup, error) {
	const op = "Allocation.GetAllocationGroups"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetAllocationGroups(ctx)
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]AllocationGroup, 0, len(raw))
	for _, item := range raw {
		out = append(out, AllocationGroup{
			Name:     rawToString(item, "name"),
			IsHidden: rawToBool(item, "isHidden"),
			Method:   rawToString(item, "method"),
		})
	}
	return out, nil
}

// CreateAllocationGroup creates a new allocation group.
func (m *AllocationManager) CreateAllocationGroup(ctx context.Context, name string, accounts []Subaccount) error {
	const op = "Allocation.CreateAllocationGroup"
	accts := make([]struct {
		AcctID string `json:"acctId"`
		Amount string `json:"amount,omitempty"`
	}, len(accounts))
	for i, a := range accounts {
		accts[i].AcctID = a.AccountID
		accts[i].Amount = a.Amount
	}
	bodyJSON := map[string]interface{}{
		"name":     name,
		"accounts": accts,
	}
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.CreateAllocationGroupWithBody(ctx,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// ModifyAllocationGroup modifies an existing allocation group.
func (m *AllocationManager) ModifyAllocationGroup(ctx context.Context, name string, accounts []Subaccount) error {
	const op = "Allocation.ModifyAllocationGroup"
	accts := make([]struct {
		AcctID string `json:"acctId"`
		Amount string `json:"amount,omitempty"`
	}, len(accounts))
	for i, a := range accounts {
		accts[i].AcctID = a.AccountID
		accts[i].Amount = a.Amount
	}
	bodyJSON := map[string]interface{}{
		"name":     name,
		"accounts": accts,
	}
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.ModifyAllocationGroupWithBody(ctx,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// DeleteAllocationGroup deletes an allocation group.
func (m *AllocationManager) DeleteAllocationGroup(ctx context.Context, name string) error {
	const op = "Allocation.DeleteAllocationGroup"
	bodyJSON := map[string]interface{}{"name": name}
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.DeleteAllocationGroupWithBody(ctx,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// GetSingleAllocationGroup returns a single allocation group.
func (m *AllocationManager) GetSingleAllocationGroup(ctx context.Context, name string) (*AllocationGroup, error) {
	const op = "Allocation.GetSingleAllocationGroup"
	bodyJSON := map[string]interface{}{"name": name}
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return nil, &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetSingleAllocationGroupWithBody(ctx,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &AllocationGroup{
		Name:     rawToString(raw, "name"),
		IsHidden: rawToBool(raw, "isHidden"),
		Method:   rawToString(raw, "method"),
	}, nil
}

// GetAllocationPresets returns the default allocation presets.
func (m *AllocationManager) GetAllocationPresets(ctx context.Context) ([]AllocationPreset, error) {
	const op = "Allocation.GetAllocationPresets"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetAllocationPresets(ctx)
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]AllocationPreset, 0, len(raw))
	for _, item := range raw {
		out = append(out, AllocationPreset{
			AccountID:  rawToString(item, "accountId"),
			Percentage: rawToString(item, "percentage"),
		})
	}
	return out, nil
}

// SetAllocationPreset sets the default allocation preset.
func (m *AllocationManager) SetAllocationPreset(ctx context.Context, presets []AllocationPreset) error {
	const op = "Allocation.SetAllocationPreset"
	accts := make([]map[string]string, len(presets))
	for i, p := range presets {
		accts[i] = map[string]string{
			"accountId":  p.AccountID,
			"percentage": p.Percentage,
		}
	}
	body, err := json.Marshal(accts)
	if err != nil {
		return &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.SetAllocationPresetWithBody(ctx,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
