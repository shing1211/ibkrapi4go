// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/shing1211/ibkrapi4go/client"
	"github.com/shing1211/ibkrapi4go/internal"
)

// RESTSSOSessions exposes SSO session management operations.
type RESTSSOSessions struct {
	surface *RESTSurface
}

// SSO returns the SSO sessions manager.
func (s *RESTSurface) SSO() *RESTSSOSessions { return &RESTSSOSessions{surface: s} }

// SSOBrowserSessionRequest is a request to create an SSO browser session.
type SSOBrowserSessionRequest struct {
	// Credential is the IBKR username.
	Credential string
	// IP address for the session.
	IP string
}

// SSOSessionRequest is a request to create an SSO session on behalf of an end-user.
type SSOSessionRequest struct {
	// Credential is the IBKR username.
	Credential string
	// IP address for the session.
	IP string
	// AlternativeIPs are optional alternative IP addresses.
	AlternativeIPs []string
	// Service is the service name.
	Service string
}

// BrowserSessionResponse is the response from creating an SSO browser session.
type BrowserSessionResponse struct {
	Active bool
	URL    string
}

// SessionResponse is the response from creating an SSO session.
type SessionResponse struct {
	AccessToken string
	Active      bool
	TokenType   string
}

// CreateBrowserSession creates an SSO browser session.
func (m *RESTSSOSessions) CreateBrowserSession(ctx context.Context, req SSOBrowserSessionRequest) (*BrowserSessionResponse, error) {
	const op = "SSO.CreateBrowserSession"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	auth, err := m.surface.Token(ctx)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	payload := client.CreateBrowserSessionRequest{
		Credential: req.Credential,
		Ip:         req.IP,
	}
	resp, err := m.surface.generated.CreateSsoBrowserSessionsWithBodyWithResponse(
		ctx,
		&client.CreateSsoBrowserSessionsParams{Authorization: client.AuthorizationHeaderParam(auth)},
		"application/json",
		bytes.NewReader(jsonMarshal(payload)),
	)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.JSON200 == nil {
		e := &Error{Op: op, Message: "unexpected nil body"}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	out := &BrowserSessionResponse{}
	if resp.JSON200.Active != nil {
		out.Active = *resp.JSON200.Active
	}
	if resp.JSON200.Url != nil {
		out.URL = *resp.JSON200.Url
	}
	return out, nil
}

// CreateSession creates an SSO session on behalf of an end-user.
func (m *RESTSSOSessions) CreateSession(ctx context.Context, req SSOSessionRequest) (*SessionResponse, error) {
	const op = "SSO.CreateSession"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	auth, err := m.surface.Token(ctx)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	payload := client.CreateSessionRequest{
		Credential: req.Credential,
		Ip:         req.IP,
	}
	if len(req.AlternativeIPs) > 0 {
		payload.AlternativeIps = &req.AlternativeIPs
	}
	if req.Service != "" {
		payload.Service = &req.Service
	}
	resp, err := m.surface.generated.CreateSsoSessionsWithBodyWithResponse(
		ctx,
		&client.CreateSsoSessionsParams{Authorization: client.AuthorizationHeaderParam(auth)},
		"application/json",
		bytes.NewReader(jsonMarshal(payload)),
	)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.JSON200 == nil {
		e := &Error{Op: op, Message: "unexpected nil body"}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	out := &SessionResponse{
		AccessToken: resp.JSON200.AccessToken,
	}
	if resp.JSON200.Active != nil {
		out.Active = *resp.JSON200.Active
	}
	if resp.JSON200.TokenType != nil {
		out.TokenType = *resp.JSON200.TokenType
	}
	return out, nil
}

// CreateSessionRaw creates an SSO session and returns the raw HTTP response.
// This is useful when you need to handle non-standard responses.
func (m *RESTSSOSessions) CreateSessionRaw(ctx context.Context, req SSOSessionRequest) (*client.CreateSsoSessionsResponse, error) {
	const op = "SSO.CreateSession"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	auth, err := m.surface.Token(ctx)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	payload := client.CreateSessionRequest{
		Credential: req.Credential,
		Ip:         req.IP,
	}
	if len(req.AlternativeIPs) > 0 {
		payload.AlternativeIps = &req.AlternativeIPs
	}
	if req.Service != "" {
		payload.Service = &req.Service
	}
	return m.surface.generated.CreateSsoSessionsWithBodyWithResponse(
		ctx,
		&client.CreateSsoSessionsParams{Authorization: client.AuthorizationHeaderParam(auth)},
		"application/json",
		bytes.NewReader(jsonMarshal(payload)),
	)
}

func jsonMarshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}
