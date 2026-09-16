// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
)

type Transport struct {
	base        http.RoundTripper
	reqID       func() string
	userAgent   string
	tokenHeader string
	token       func() (string, bool)
}

func NewTransport(base http.RoundTripper, opts ...func(*Transport)) *Transport {
	if base == nil {
		base = http.DefaultTransport
	}
	t := &Transport{base: base}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func WithRequestID(f func() string) func(*Transport) { return func(t *Transport) { t.reqID = f } }
func WithUserAgent(ua string) func(*Transport)       { return func(t *Transport) { t.userAgent = ua } }
func WithToken(header string, f func() (string, bool)) func(*Transport) {
	return func(t *Transport) { t.tokenHeader, t.token = header, f }
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())

	reqID := ""
	if t.reqID != nil {
		reqID = t.reqID()
		req.Header.Set("X-request-id", reqID)
	}
	if t.userAgent != "" {
		req.Header.Set("User-Agent", t.userAgent)
	}
	if t.tokenHeader != "" && t.token != nil {
		if tok, ok := t.token(); ok {
			req.Header.Set(t.tokenHeader, tok)
		}
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	if !isSuccess(resp.StatusCode) {
		// Prefer response X-request-id (set by upstream server); fall back to our request ID.
		respReqID := resp.Header.Get("X-request-id")
		if respReqID == "" {
			respReqID = reqID
		}
		resp = t.carryError(resp, respReqID)
	}
	return resp, nil
}

func isSuccess(code int) bool { return code >= 200 && code < 300 }

var headerRedact = regexp.MustCompile(`(?i)(Authorization|Cookie|Set-Cookie)\s*:`)

func redact(s string) string { return headerRedact.ReplaceAllString(s, "$1: <redacted>") }

// carryError reads the response body, parses the IBKR error envelope, and returns
// a response with the error details stored in headers. The body is replaced with
// a fresh reader so the caller can still read it.
func (t *Transport) carryError(resp *http.Response, reqID string) *http.Response {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return errorResponse(resp, reqID, http.StatusText(resp.StatusCode), "", statusSentinel[resp.StatusCode])
	}
	resp.Body.Close()

	var env ibkrErrorEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return errorResponse(resp, reqID, http.StatusText(resp.StatusCode), "", statusSentinel[resp.StatusCode])
	}

	msg := strings.TrimSpace(env.Message)
	if msg == "" {
		msg = strings.TrimSpace(env.Error)
	}
	if msg == "" {
		msg = http.StatusText(resp.StatusCode)
	}
	code := strings.TrimSpace(env.Code)
	if code == "" {
		code = strings.TrimSpace(env.Error)
	}

	// Replace body so the caller can re-read it.
	resp.Body = io.NopCloser(bytes.NewReader(body))

	return errorResponse(resp, reqID, msg, code, statusSentinel[resp.StatusCode])
}

func errorResponse(resp *http.Response, reqID, msg, code string, serr error) *http.Response {
	resp.Header.Set("X-ibkr-error", msg)
	resp.Header.Set("X-ibkr-error-code", code)
	resp.Header.Set("X-ibkr-request-id", reqID)
	if serr != nil {
		resp.Header.Set("X-ibkr-sentinel", serr.Error())
	}
	return resp
}

// ResponseError extracts an *Error from a response that was flagged as an error.
// It returns nil if the response is not an error response (2xx).
// The caller must not have read the body yet.
func ResponseError(resp *http.Response) *Error {
	if isSuccess(resp.StatusCode) {
		return nil
	}
	errMsg := resp.Header.Get("X-ibkr-error")
	code := resp.Header.Get("X-ibkr-error-code")
	reqID := resp.Header.Get("X-ibkr-request-id")
	sentinelName := resp.Header.Get("X-ibkr-sentinel")

	var serr error
	if sentinelName != "" {
		serr = sentinelByName(sentinelName)
	}
	if serr == nil {
		serr = statusSentinel[resp.StatusCode]
	}
	if errMsg == "" {
		errMsg = http.StatusText(resp.StatusCode)
	}
	return &Error{
		Op:         "unknown",
		Code:       code,
		Message:    errMsg,
		HTTPStatus: resp.StatusCode,
		RequestID:  reqID,
		Err:        serr,
	}
}

func sentinelByName(name string) error {
	switch name {
	case ErrNotAuthenticated.Error():
		return ErrNotAuthenticated
	case ErrSessionExpired.Error():
		return ErrSessionExpired
	case ErrRateLimited.Error():
		return ErrRateLimited
	case ErrNotFound.Error():
		return ErrNotFound
	case ErrInvalidRequest.Error():
		return ErrInvalidRequest
	case ErrOrderRejected.Error():
		return ErrOrderRejected
	case ErrClosed.Error():
		return ErrClosed
	default:
		return nil
	}
}

// RoundTripFunc lets a plain function satisfy http.RoundTripper.
type RoundTripFunc func(*http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Chain builds a middleware stack on top of base. Middlewares are applied
// left-to-right: ms[0] is the outermost (called first on the way in, last on
// the way out).
func Chain(base http.RoundTripper, ms ...func(http.RoundTripper) http.RoundTripper) http.RoundTripper {
	for i := len(ms) - 1; i >= 0; i-- {
		base = ms[i](base)
	}
	return base
}

// RequestID returns a middleware that injects X-request-id from f.
func RequestID(f func() string) func(http.RoundTripper) http.RoundTripper {
	return func(base http.RoundTripper) http.RoundTripper {
		return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			req = req.Clone(req.Context())
			req.Header.Set("X-request-id", f())
			return base.RoundTrip(req)
		})
	}
}

// UserAgent returns a middleware that sets User-Agent.
func UserAgent(ua string) func(http.RoundTripper) http.RoundTripper {
	return func(base http.RoundTripper) http.RoundTripper {
		return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			req = req.Clone(req.Context())
			req.Header.Set("User-Agent", ua)
			return base.RoundTrip(req)
		})
	}
}

// Auth returns a middleware that adds token as header when token() returns ok.
func Auth(header string, token func() (string, bool)) func(http.RoundTripper) http.RoundTripper {
	return func(base http.RoundTripper) http.RoundTripper {
		return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			req = req.Clone(req.Context())
			if tok, ok := token(); ok {
				req.Header.Set(header, tok)
			}
			return base.RoundTrip(req)
		})
	}
}
