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

// FYIManager exposes FYI notification and delivery operations. It is safe for
// concurrent use.
type FYIManager struct {
	client *Client
}

// FYIDelivery holds delivery option information.
type FYIDelivery struct {
	// DeviceID is the device identifier.
	DeviceID string `json:"deviceId"`
	// Type is the delivery type (e.g. "email", "sms").
	Type string `json:"type"`
	// Value is the delivery address/value.
	Value string `json:"value"`
}

// FYIDisclaimer holds a disclaimer for a FYI type.
type FYIDisclaimer struct {
	// TypeCode is the FYI type code.
	TypeCode string `json:"typeCode"`
	// Content is the disclaimer content.
	Content string `json:"content"`
}

// FYINotification holds a single FYI notification.
type FYINotification struct {
	// ID is the notification identifier.
	ID string `json:"id"`
	// TypeCode is the FYI type code.
	TypeCode string `json:"typeCode"`
	// Subject is the notification subject.
	Subject string `json:"subject"`
	// Read reports whether the notification has been read.
	Read bool `json:"read"`
}

// FYISettings holds FYI notification settings.
type FYISettings struct {
	// TypeCode is the FYI type code.
	TypeCode string `json:"typeCode"`
	// Enabled reports whether this FYI type is enabled.
	Enabled bool `json:"enabled"`
}

// UnreadFYIs holds the count of unread FYIs.
type UnreadFYIs struct {
	// Count is the number of unread FYIs.
	Count int `json:"count"`
}

// GetFYIDelivery returns the delivery options for FYIs.
func (m *FYIManager) GetFYIDelivery(ctx context.Context) ([]FYIDelivery, error) {
	const op = "FYI.GetFYIDelivery"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetFyiDelivery(ctx)
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]FYIDelivery, 0, len(raw))
	for _, item := range raw {
		out = append(out, FYIDelivery{
			DeviceID: rawToString(item, "deviceId"),
			Type:     rawToString(item, "type"),
			Value:    rawToString(item, "value"),
		})
	}
	return out, nil
}

// ModifyFYIDelivery adds or updates a FYI delivery option.
func (m *FYIManager) ModifyFYIDelivery(ctx context.Context, deliveryType, value string) error {
	const op = "FYI.ModifyFYIDelivery"
	bodyJSON := map[string]interface{}{
		"type":  deliveryType,
		"value": value,
	}
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.ModifyFyiDeliveryWithBody(ctx,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// ModifyFYIEmails enables or disables email delivery for FYIs.
func (m *FYIManager) ModifyFYIEmails(ctx context.Context, enabled bool) error {
	const op = "FYI.ModifyFYIEmails"
	params := &client.ModifyFyiEmailsParams{
		Enabled: enabled,
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.ModifyFyiEmails(ctx, params)
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// DeleteFYIDevice removes a FYI delivery device.
func (m *FYIManager) DeleteFYIDevice(ctx context.Context, deviceID string) error {
	const op = "FYI.DeleteFYIDevice"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.DeleteFyiDevice(ctx, deviceID)
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// GetFYIDisclaimers returns the disclaimer for a given FYI type code.
func (m *FYIManager) GetFYIDisclaimers(ctx context.Context, typeCode string) (*FYIDisclaimer, error) {
	const op = "FYI.GetFYIDisclaimers"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetFyiDisclaimerss(ctx, typeCode)
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &FYIDisclaimer{
		TypeCode: rawToString(raw, "typeCode"),
		Content:  rawToString(raw, "content"),
	}, nil
}

// ReadFYIDisclaimer marks a FYI disclaimer as read.
func (m *FYIManager) ReadFYIDisclaimer(ctx context.Context, typeCode string) error {
	const op = "FYI.ReadFYIDisclaimer"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.ReadFyiDisclaimer(ctx, typeCode)
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// GetAllFYIs returns all FYI notifications.
func (m *FYIManager) GetAllFYIs(ctx context.Context, max int64) ([]FYINotification, error) {
	const op = "FYI.GetAllFYIs"
	params := &client.GetAllFyisParams{Max: max}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetAllFyis(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]FYINotification, 0, len(raw))
	for _, item := range raw {
		out = append(out, FYINotification{
			ID:       rawToString(item, "id"),
			TypeCode: rawToString(item, "typeCode"),
			Subject:  rawToString(item, "subject"),
			Read:     rawToBool(item, "read"),
		})
	}
	return out, nil
}

// ReadFYINotification marks a FYI notification as read.
func (m *FYIManager) ReadFYINotification(ctx context.Context, notificationID string) error {
	const op = "FYI.ReadFYINotification"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.ReadFyiNotification(ctx, notificationID)
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// GetFYISettings returns FYI notification settings.
func (m *FYIManager) GetFYISettings(ctx context.Context) ([]FYISettings, error) {
	const op = "FYI.GetFYISettings"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetFyiSettings(ctx)
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]FYISettings, 0, len(raw))
	for _, item := range raw {
		out = append(out, FYISettings{
			TypeCode: rawToString(item, "typeCode"),
			Enabled:  rawToBool(item, "enabled"),
		})
	}
	return out, nil
}

// ModifyFYINotification enables or disables a FYI notification type.
func (m *FYIManager) ModifyFYINotification(ctx context.Context, typeCode string, enabled bool) error {
	const op = "FYI.ModifyFYINotification"
	body := client.ModifyFyiNotificationJSONRequestBody(client.ModifyFyiNotificationJSONBody{
		Enabled: &enabled,
	})
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.ModifyFyiNotification(ctx, typeCode, body)
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// GetUnreadFYIs returns the count of unread FYIs.
func (m *FYIManager) GetUnreadFYIs(ctx context.Context) (*UnreadFYIs, error) {
	const op = "FYI.GetUnreadFYIs"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetUnreadFyis(ctx)
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &UnreadFYIs{
		Count: jsonNumberToInt(jsonNumber(raw["count"])),
	}, nil
}
