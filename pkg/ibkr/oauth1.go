// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"net/http"

	"github.com/shing1211/ibkrapi4go/client"
)

// OAuthManager exposes OAuth token operations. It is safe for concurrent use.
type OAuthManager struct {
	client *Client
}

// AccessTokenResult holds the result of an access token request.
type AccessTokenResult struct {
	// Token is the access token.
	Token string `json:"token"`
}

// LiveSessionTokenResult holds the result of a live session token request.
type LiveSessionTokenResult struct {
	// Token is the live session token.
	Token string `json:"token"`
}

// TempTokenResult holds the result of a temporary token request.
type TempTokenResult struct {
	// Token is the temporary token.
	Token string `json:"token"`
}

// RequestAccessToken requests an OAuth access token.
func (m *OAuthManager) RequestAccessToken(ctx context.Context, authorization *string) (*AccessTokenResult, error) {
	const op = "OAuth.RequestAccessToken"
	params := &client.ReqAccessTokenParams{
		Authorization: authorization,
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.ReqAccessToken(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return &AccessTokenResult{}, nil
}

// RequestLiveSessionToken requests an OAuth live session token.
func (m *OAuthManager) RequestLiveSessionToken(ctx context.Context, authorization *string) (*LiveSessionTokenResult, error) {
	const op = "OAuth.RequestLiveSessionToken"
	params := &client.ReqLiveSessionTokenParams{
		Authorization: authorization,
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.ReqLiveSessionToken(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return &LiveSessionTokenResult{}, nil
}

// RequestTempToken requests an OAuth temporary token.
func (m *OAuthManager) RequestTempToken(ctx context.Context, authorization *string) (*TempTokenResult, error) {
	const op = "OAuth.RequestTempToken"
	params := &client.ReqTempTokenParams{
		Authorization: authorization,
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.ReqTempToken(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return &TempTokenResult{}, nil
}
