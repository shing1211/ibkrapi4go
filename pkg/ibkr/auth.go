// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
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
