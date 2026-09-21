// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"net/http"
)

type (
	// TokenProvider abstracts OAuth2 token acquisition and refresh.
	TokenProvider interface {
		Token(ctx context.Context) (string, error)
	}

	// RoundTripper abstracts the HTTP transport layer. It is satisfied by
	// http.RoundTripper and exists as a named interface to allow fakes in tests.
	RoundTripper = http.RoundTripper

	// Middleware wraps an http.RoundTripper. It receives the next transport and
	// returns a new transport that adds behaviour (logging, tracing, auth, etc.).
	// Implementations must be safe for concurrent use.
	Middleware = func(http.RoundTripper) http.RoundTripper

	// SessionMachine abstracts the session state machine. It drives authentication
	// lifecycle (init, tickle, close) and exposes the current state and token.
	SessionMachine interface {
		State() SessionState
		Token() (token string, ok bool)
		Initialize(ctx context.Context) error
		Close(ctx context.Context) error
	}

	// WSClient abstracts a single multiplexed WebSocket connection to the gateway.
	WSClient interface {
		Subscribe(ctx context.Context, sink WSSink, conids []int, fields []string) (*WSHandle, error)
		ActiveSubscriptions() int
		Close() error
	}

	// RateLimiter abstracts the rate limiter's Wait method.
	RateLimiter interface {
		Wait(ctx context.Context, method, path string) error
	}
)

// Compile-time interface satisfaction checks.
var (
	_ TokenProvider  = (*TokenSource)(nil)
	_ SessionMachine = (*Session)(nil)
	_ WSClient       = (*WSConn)(nil)
	_ RateLimiter    = (*Limiter)(nil)
)
