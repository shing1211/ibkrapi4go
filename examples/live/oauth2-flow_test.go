// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"unrelated", errors.New("dial tcp: connection refused"), false},
		{"status 401", errors.New("request failed with status 401"), true},
		{"status 403", errors.New("request failed with status 403"), true},
		{"word authentication", errors.New("authentication required"), true},
		{"word credentials", errors.New("bad credentials supplied"), true},
		{"word unauthorized", errors.New("unauthorized"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAuthError(tt.err); got != tt.want {
				t.Errorf("isAuthError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestHandleAuthError_PassesThroughNonAuthError(t *testing.T) {
	sentinel := errors.New("dial tcp: connection refused")

	// A non-auth error must be returned unchanged without touching the surface.
	if err := handleAuthError(context.Background(), sentinel, nil); !errors.Is(err, sentinel) {
		t.Errorf("handleAuthError = %v, want the original error", err)
	}
}

func TestHandleAuthError_NilError(t *testing.T) {
	if err := handleAuthError(context.Background(), nil, nil); err != nil {
		t.Errorf("handleAuthError(nil) = %v, want nil", err)
	}
}

func TestHandleAuthError_AuthErrorOnClosedClient(t *testing.T) {
	cli, err := ibkr.NewClient(
		ibkr.WithOAuth2ClientCredentials("cid", "secret"),
		ibkr.WithOAuth2RefreshToken("refresh"),
	)
	if err != nil {
		t.Skipf("OAuth2 client not configurable in this environment: %v", err)
	}
	rest, err := cli.REST()
	if err != nil {
		t.Fatalf("REST: %v", err)
	}
	if err := cli.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// On a closed client the refresh cannot run, so the failure must be
	// reported rather than silently swallowed.
	err = handleAuthError(context.Background(), errors.New("request failed with status 401"), rest)
	if err == nil {
		t.Fatal("handleAuthError returned nil for an auth error on a closed client")
	}
	if !errors.Is(err, ibkr.ErrClosed) {
		t.Errorf("handleAuthError error = %v, want it to wrap ErrClosed", err)
	}
}

func TestHTTPStatusCheck(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   bool
	}{
		{"ok", http.StatusOK, false},
		{"created", http.StatusCreated, false},
		{"bad request", http.StatusBadRequest, true},
		{"unauthorized", http.StatusUnauthorized, true},
		{"server error", http.StatusInternalServerError, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tt.status}
			if got := httpStatusCheck(resp); got != tt.want {
				t.Errorf("httpStatusCheck(%d) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}

	if httpStatusCheck(nil) {
		t.Error("httpStatusCheck(nil) = true, want false")
	}
}
