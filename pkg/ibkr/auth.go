// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// AuthStatus is a snapshot of the gateway brokerage session.
type AuthStatus struct {
	// Authenticated reports whether the brokerage session is authenticated.
	Authenticated bool
	// Connected reports whether the gateway is connected.
	Connected bool
	// Established reports whether the session is fully established and ready
	// to handle requests.
	Established bool
	// Fail carries the server-provided reason when authentication failed.
	Fail string
	// Token is the current session token, if one has been issued.
	Token string
	// Updated is when the status was fetched.
	Updated time.Time
}

// SessionManager owns the session lifecycle: initialization, the tickle
// heartbeat, and shutdown. It is safe for concurrent use.
type SessionManager struct {
	client *Client
}

// Initialize starts the gateway session (if not already authenticated) and,
// on success, starts the tickle heartbeat. It is idempotent when the session is
// already AUTHENTICATED.
func (m *SessionManager) Initialize(ctx context.Context) error {
	if err := m.client.checkOpen(); err != nil {
		return err
	}
	return m.client.session.Initialize(ctx)
}

// Close stops the tickle heartbeat and performs a best-effort logout. It is
// idempotent.
func (m *SessionManager) Close(ctx context.Context) error {
	return m.client.session.Close(ctx)
}

// State returns the current session state.
func (m *SessionManager) State() SessionState {
	return m.client.session.State()
}

// SessionValidation holds the result of a session validation check.
type SessionValidation struct {
	// Valid reports whether the session is valid.
	Valid bool
	// Message is the validation message.
	Message string
}

// SessionToken holds the current session token.
type SessionToken struct {
	// Token is the session token.
	Token string
}

// Status fetches the current brokerage session status from the gateway.
func (m *SessionManager) Status(ctx context.Context) (*AuthStatus, error) {
	if err := m.client.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.client.generated.GetBrokerageStatusWithResponse(ctx)
	if err != nil {
		return nil, err
	}
	if e := m.client.errorFrom(resp.HTTPResponse, "Session.Status"); e != nil {
		return nil, e
	}
	if resp.JSON200 == nil {
		return nil, &Error{Op: "Session.Status", Message: "empty response"}
	}
	bs := resp.JSON200
	st := &AuthStatus{
		Authenticated: bs.Authenticated != nil && *bs.Authenticated,
		Connected:     bs.Connected != nil && *bs.Connected,
		Established:   bs.Established != nil && *bs.Established,
		Fail:          derefString(bs.Fail, ""),
		Updated:       time.Now(),
	}
	if tok, ok := m.client.session.Token(); ok {
		st.Token = tok
	}
	return st, nil
}

// SessionValidation validates the current session.
func (m *SessionManager) SessionValidation(ctx context.Context) (*SessionValidation, error) {
	const op = "Session.GetSessionValidation"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetSessionValidation(ctx)
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &SessionValidation{
		Valid:   rawToBool(raw, "valid"),
		Message: rawToString(raw, "message"),
	}, nil
}

// SessionToken returns the current session token.
func (m *SessionManager) SessionToken(ctx context.Context) (*SessionToken, error) {
	const op = "Session.GetSessionToken"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetSessionToken(ctx)
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return &SessionToken{
		Token: rawToString(raw, "token"),
	}, nil
}
