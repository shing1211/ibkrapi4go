// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"errors"
	"testing"
)

// TestWrapOp_PlainError pins the ordinary case: a non-*Error is wrapped in a
// fresh *Error that carries Op, the underlying message, and the cause.
func TestWrapOp_PlainError(t *testing.T) {
	cause := errors.New("dial tcp 127.0.0.1:1: connect: connection refused")

	e := wrapOp("Accounts.List", cause)

	if e.Op != "Accounts.List" {
		t.Errorf("Op = %q; want Accounts.List", e.Op)
	}
	if e.Message != cause.Error() {
		t.Errorf("Message = %q; want %q", e.Message, cause.Error())
	}
	if !errors.Is(e, cause) {
		t.Error("errors.Is(wrapped, cause) = false; want true")
	}
	if e.Code != "" || e.HTTPStatus != 0 {
		t.Errorf("Code = %q, HTTPStatus = %d; want the zero values a plain error has", e.Code, e.HTTPStatus)
	}
}

// TestWrapOp_AdoptsErrorWithOp pins that an already-tagged *Error is adopted
// unchanged. Overwriting Op here would lose the origin the inner layer knew
// (docs/ERRORS.md: token failures surface with Op == "OAuth.Token").
func TestWrapOp_AdoptsErrorWithOp(t *testing.T) {
	inner := &Error{Op: "OAuth.Token", Message: "token endpoint returned 401", HTTPStatus: 401, Err: ErrSessionExpired}

	e := wrapOp("SSO.CreateSession", inner)

	if e != inner {
		t.Errorf("wrapOp returned %p; want the same *Error it was given (%p)", e, inner)
	}
	if e.Op != "OAuth.Token" {
		t.Errorf("Op = %q; want the inner OAuth.Token, not overwritten with SSO.CreateSession", e.Op)
	}
	if e.HTTPStatus != 401 {
		t.Errorf("HTTPStatus = %d; want 401", e.HTTPStatus)
	}
	if !errors.Is(e, ErrSessionExpired) {
		t.Error("errors.Is(wrapped, ErrSessionExpired) = false; want true")
	}
}

// TestWrapOp_FillsMissingOp pins the one field wrapOp still owns: an *Error that
// has not been tagged yet (the shape every >= 400 guard in rest.go builds)
// must come back tagged, and tagged exactly once.
func TestWrapOp_FillsMissingOp(t *testing.T) {
	inner := &Error{Code: "http_error", Message: "GetRequestsStatus: 500", HTTPStatus: 500}

	e := wrapOp("Requests.GetStatus", inner)

	if e != inner {
		t.Errorf("wrapOp returned %p; want the same *Error it was given (%p)", e, inner)
	}
	if e.Op != "Requests.GetStatus" {
		t.Errorf("Op = %q; want Requests.GetStatus filled in", e.Op)
	}
}

// TestWrapOp_PreservesHTTPDetail is the regression test for the double-wrap
// defect: the >= 400 guards build a typed *Error carrying Code and HTTPStatus,
// and the returned value — not something one level down inside .Err — has to
// report them, or the documented errors.As idiom reads 0.
func TestWrapOp_PreservesHTTPDetail(t *testing.T) {
	inner := &Error{Code: "http_error", Message: "GetRequestsStatus: 500", HTTPStatus: 500}

	e := wrapOp("Requests.GetStatus", inner)

	if e.Code != "http_error" {
		t.Errorf("Code = %q; want http_error on the returned error", e.Code)
	}
	if e.HTTPStatus != 500 {
		t.Errorf("HTTPStatus = %d; want 500 on the returned error", e.HTTPStatus)
	}
	if e.Err != nil {
		t.Errorf("Err = %v; want nil, so there is no second hop to dig through", e.Err)
	}

	// The same two fields, reached the way a caller reaches them.
	var got *Error
	if !errors.As(error(e), &got) {
		t.Fatalf("errors.As = false; want the returned value itself to be the *Error")
	}
	if got.HTTPStatus != 500 || got.Code != "http_error" {
		t.Errorf("errors.As yielded HTTPStatus=%d Code=%q; want 500/http_error", got.HTTPStatus, got.Code)
	}
}
