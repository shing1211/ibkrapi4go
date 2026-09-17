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

// RESTEcho exposes echo utility operations for testing security policies.
type RESTEcho struct {
	surface *RESTSurface
}

// Echo returns the Echo utilities manager.
func (s *RESTSurface) Echo() *RESTEcho { return &RESTEcho{surface: s} }

// EchoHttpsResponse is the response from the HTTPS echo endpoint.
type EchoHttpsResponse struct {
	RequestMethod  string
	SecurityPolicy string
}

// ListEchoHttps performs an echo request with HTTPS security policy validation.
// This is a simple GET request that echoes back the security policy information.
func (m *RESTEcho) ListEchoHttps(ctx context.Context) (*EchoHttpsResponse, error) {
	const op = "Echo.ListEchoHttps"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.ListEchoHttpsWithResponse(ctx)
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
	var raw echoResponseRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	return raw.toPublic(), nil
}

// SignedJwtEchoRequest is a request to test the signed JWT security policy.
type SignedJwtEchoRequest struct {
	// Iss is the issuer to include in the JWT.
	Iss string
}

// SignedJwtEchoResponse is the response from the signed JWT echo endpoint.
type SignedJwtEchoResponse struct {
	RequestMethod  string
	SecurityPolicy string
}

// CreateEchoSignedJwt performs an echo request with signed JWT security policy validation.
// This is used to test that the JWT security policy is working correctly.
func (m *RESTEcho) CreateEchoSignedJwt(ctx context.Context, req SignedJwtEchoRequest) (*SignedJwtEchoResponse, error) {
	const op = "Echo.CreateEchoSignedJwt"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	payload := client.SignedJwtEchoRequest{}
	if req.Iss != "" {
		payload.Iss = &req.Iss
	}
	resp, err := m.surface.generated.CreateEchoSignedJwtWithBodyWithResponse(
		ctx,
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
	var raw echoResponseRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	return raw.toPublicSignedJwt(), nil
}

type echoResponseRaw struct {
	RequestMethod  *string `json:"requestMethod,omitempty"`
	SecurityPolicy *string `json:"securityPolicy,omitempty"`
}

func (r *echoResponseRaw) toPublic() *EchoHttpsResponse {
	if r == nil {
		return nil
	}
	return &EchoHttpsResponse{
		RequestMethod:  strPtrVal(r.RequestMethod),
		SecurityPolicy: strPtrVal(r.SecurityPolicy),
	}
}

func (r *echoResponseRaw) toPublicSignedJwt() *SignedJwtEchoResponse {
	if r == nil {
		return nil
	}
	return &SignedJwtEchoResponse{
		RequestMethod:  strPtrVal(r.RequestMethod),
		SecurityPolicy: strPtrVal(r.SecurityPolicy),
	}
}
