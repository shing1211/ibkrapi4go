// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"net/http"
	"testing"
)

// TestRESTSSOSessions_CreateSession_DecodesEveryFixtureField pins the mock
// gateway's SSO session fixture to the JSON tags the generated response type
// actually declares.
//
// client.CreateSessionResponse tags its fields snake_case — access_token and
// token_type — while the fixture used to spell them camelCase. Decoding is
// non-strict (json.Unmarshal in ParseCreateSsoSessionsResponse, no
// DisallowUnknownFields), so the mismatched keys were dropped without error:
// the gateway returned a 200 whose access token the wrapper could never read.
// Asserting the decoded values, not merely a nil error, is what catches that —
// a success status alone is exactly what the broken fixture produced too.
func TestRESTSSOSessions_CreateSession_DecodesEveryFixtureField(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)

	got, err := surface.SSO().CreateSession(context.Background(), SSOSessionRequest{
		Credential:     "jdoe",
		IP:             "10.0.0.1",
		AlternativeIPs: []string{"10.0.0.2"},
		Service:        "ibkrapi",
	})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if got.AccessToken != "sso-access-token" {
		t.Errorf("AccessToken = %q; want %q — the fixture key must match the generated access_token tag or the value is silently dropped",
			got.AccessToken, "sso-access-token")
	}
	if got.TokenType != "Bearer" {
		t.Errorf("TokenType = %q; want %q — the fixture key must match the generated token_type tag or the pointer stays nil",
			got.TokenType, "Bearer")
	}
	if !got.Active {
		t.Error("Active = false; want true from the fixture's active key")
	}

	req := lastRESTRequest(t, gw)
	if req.Method != http.MethodPost {
		t.Errorf("method = %s; want POST", req.Method)
	}
	if req.Path != "/gw/api/v1/sso-sessions" {
		t.Errorf("path = %q; want /gw/api/v1/sso-sessions", req.Path)
	}
}
