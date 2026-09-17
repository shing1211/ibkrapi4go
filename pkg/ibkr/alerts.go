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

// AlertManager exposes alert operations. It is safe for concurrent use.
type AlertManager struct {
	client *Client
}

// AlertDetails describes an alert's configuration and state.
type AlertDetails struct {
	// AlertID is the alert identifier.
	AlertID string `json:"alertId"`
	// AccountID is the account the alert belongs to.
	AccountID string `json:"accountId"`
	// Name is the human-readable alert name.
	Name string `json:"name"`
	// AlertType is the alert type.
	AlertType string `json:"alertType"`
	// Enabled reports whether the alert is active.
	Enabled bool `json:"enabled"`
	// Filled reports whether the alert has been triggered.
	Filled bool `json:"filled"`
}

// MTADetails holds multi-trade account details.
type MTADetails struct {
	// Accounts is the list of accounts in the MTA group.
	Accounts []string `json:"accounts"`
}

// AlertActivationResult is the result of activating or deactivating an alert.
type AlertActivationResult struct {
	// AlertID is the alert identifier.
	AlertID string `json:"alertId"`
	// Active reports whether the alert is now active.
	Active bool `json:"active"`
}

// AlertCreationResult is the result of creating an alert.
type AlertCreationResult struct {
	// AlertID is the newly created alert identifier.
	AlertID string `json:"alertId"`
}

// AlertDeletionResult is the result of deleting an alert.
type AlertDeletionResult struct {
	// AlertID is the deleted alert identifier.
	AlertID string `json:"alertId"`
}

// GetAlertDetails returns the details of a specific alert.
func (m *AlertManager) GetAlertDetails(ctx context.Context, alertID int64, alertType string) (*AlertDetails, error) {
	const op = "Alert.GetAlertDetails"
	params := &client.GetAlertDetailsParams{
		Type: client.GetAlertDetailsParamsType(alertType),
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetAlertDetails(ctx, alertID, params)
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &AlertDetails{
		AlertID:   rawToString(raw, "alertId"),
		AccountID: rawToString(raw, "accountId"),
		Name:      rawToString(raw, "name"),
		AlertType: rawToString(raw, "alertType"),
		Enabled:   rawToBool(raw, "enabled"),
		Filled:    rawToBool(raw, "filled"),
	}, nil
}

// GetMtaDetails returns multi-trade account details.
func (m *AlertManager) GetMtaDetails(ctx context.Context) (*MTADetails, error) {
	const op = "Alert.GetMtaDetails"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetMtaDetails(ctx)
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &MTADetails{
		Accounts: rawToStringSlice(raw, "accounts"),
	}, nil
}

// CreateAlert creates a new alert for the given account.
func (m *AlertManager) CreateAlert(ctx context.Context, accountID AccountID, name, alertType string) (*AlertCreationResult, error) {
	const op = "Alert.CreateAlert"
	bodyJSON := map[string]interface{}{
		"name":      name,
		"alertType": alertType,
	}
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return nil, &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.CreateAlertWithBody(ctx, string(accountID),
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &AlertCreationResult{
		AlertID: rawToString(raw, "alertId"),
	}, nil
}

// ActivateAlert enables or disables an alert.
func (m *AlertManager) ActivateAlert(ctx context.Context, accountID AccountID, alertID int64, active bool) (*AlertActivationResult, error) {
	const op = "Alert.ActivateAlert"
	bodyJSON := map[string]interface{}{
		"alertId": alertID,
		"active":  active,
	}
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return nil, &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.ActivateAlertWithBody(ctx, string(accountID),
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &AlertActivationResult{
		AlertID: rawToString(raw, "alertId"),
		Active:  rawToBool(raw, "active"),
	}, nil
}

// DeleteAlert deletes an alert.
func (m *AlertManager) DeleteAlert(ctx context.Context, accountID AccountID, alertID string) error {
	const op = "Alert.DeleteAlert"
	body := client.DeleteAlertJSONBody{}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.DeleteAlert(ctx, string(accountID), alertID, body)
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// GetAllAlerts returns all alerts for the given account.
func (m *AlertManager) GetAllAlerts(ctx context.Context, accountID AccountID) ([]AlertDetails, error) {
	const op = "Alert.GetAllAlerts"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetAllAlerts(ctx, string(accountID))
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]AlertDetails, 0, len(raw))
	for _, item := range raw {
		out = append(out, AlertDetails{
			AlertID:   rawToString(item, "alertId"),
			AccountID: rawToString(item, "accountId"),
			Name:      rawToString(item, "name"),
			AlertType: rawToString(item, "alertType"),
			Enabled:   rawToBool(item, "enabled"),
			Filled:    rawToBool(item, "filled"),
		})
	}
	return out, nil
}

// alertIDToString converts an alert ID to string.
func alertIDToString(id int64) string {
	return strconv.FormatInt(id, 10)
}
