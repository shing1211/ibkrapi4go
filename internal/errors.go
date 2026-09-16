// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type Error struct {
	Op         string
	Code       string
	Message    string
	HTTPStatus int
	RequestID  string
	Err        error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %d %s/%s: %v", e.Op, e.HTTPStatus, e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %d %s/%s", e.Op, e.HTTPStatus, e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

// Sentinel errors.
var (
	ErrNotAuthenticated = errors.New("ibkr: gateway not authenticated")
	ErrSessionExpired   = errors.New("ibkr: session expired")
	ErrRateLimited      = errors.New("ibkr: rate limited")
	ErrNotFound         = errors.New("ibkr: not found")
	ErrInvalidRequest   = errors.New("ibkr: invalid request")
	ErrOrderRejected    = errors.New("ibkr: order rejected")
	ErrClosed           = errors.New("ibkr: client closed")
	ErrStreamingLimit   = errors.New("ibkr: streaming limit exceeded")
)

// statusSentinel maps HTTP status codes to sentinel errors.
var statusSentinel = map[int]error{
	http.StatusUnauthorized:    ErrSessionExpired,
	http.StatusTooManyRequests: ErrRateLimited,
	http.StatusNotFound:        ErrNotFound,
	http.StatusBadRequest:      ErrInvalidRequest,
	http.StatusForbidden:       ErrNotAuthenticated,
}

// ibkrErrorEnvelope is the shape of an IBKR error response.
type ibkrErrorEnvelope struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
	Details any    `json:"details,omitempty"`
}

// buildError constructs *Error from an HTTP response and a request ID.
func buildError(op string, status int, requestID string, body []byte, underlying error) *Error {
	code := ""
	msg := strings.TrimSpace(string(body))
	if len(msg) > 200 {
		msg = msg[:200] + "..."
	}
	if se := statusSentinel[status]; se != nil {
		code = se.Error()
	}
	return &Error{
		Op:         op,
		Code:       code,
		Message:    msg,
		HTTPStatus: status,
		RequestID:  requestID,
		Err:        underlying,
	}
}
