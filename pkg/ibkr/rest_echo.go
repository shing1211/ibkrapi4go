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

// EchoHTTPSResponse is the response from the HTTPS echo endpoint.
type EchoHTTPSResponse struct {
	RequestMethod  string
	SecurityPolicy string
}

// ListEchoHTTPS performs an echo request with HTTPS security policy validation.
// This is a simple GET request that echoes back the security policy information.
func (m *RESTEcho) ListEchoHTTPS(ctx context.Context) (*EchoHTTPSResponse, error) {
	const op = "Echo.ListEchoHTTPS"
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

// SignedJWTEchoRequest is a request to test the signed JWT security policy.
type SignedJWTEchoRequest struct {
	// Iss is the issuer to include in the JWT.
	Iss string
}

// SignedJWTEchoResponse is the response from the signed JWT echo endpoint.
type SignedJWTEchoResponse struct {
	RequestMethod  string
	SecurityPolicy string
}

// CreateEchoSignedJWT performs an echo request with signed JWT security policy validation.
// This is used to test that the JWT security policy is working correctly.
func (m *RESTEcho) CreateEchoSignedJWT(ctx context.Context, req SignedJWTEchoRequest) (*SignedJWTEchoResponse, error) {
	const op = "Echo.CreateEchoSignedJWT"
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

func (r *echoResponseRaw) toPublic() *EchoHTTPSResponse {
	if r == nil {
		return nil
	}
	return &EchoHTTPSResponse{
		RequestMethod:  strPtrVal(r.RequestMethod),
		SecurityPolicy: strPtrVal(r.SecurityPolicy),
	}
}

func (r *echoResponseRaw) toPublicSignedJwt() *SignedJWTEchoResponse {
	if r == nil {
		return nil
	}
	return &SignedJWTEchoResponse{
		RequestMethod:  strPtrVal(r.RequestMethod),
		SecurityPolicy: strPtrVal(r.SecurityPolicy),
	}
}
