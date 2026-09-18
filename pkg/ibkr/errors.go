// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"github.com/shing1211/ibkrapi4go/internal"
)

// Error is the SDK error type. All API failures surface as *Error.
type Error = internal.Error

// ConfigError reports invalid client configuration. It is returned from
// NewClient, never as a panic.
type ConfigError struct {
	// Field names the offending configuration field.
	Field string
	// Message describes the problem.
	Message string
}

// Error implements the error interface.
func (e *ConfigError) Error() string {
	return "ibkr: config: " + e.Field + ": " + e.Message
}

// Sentinel errors.
var (
	ErrNotAuthenticated = internal.ErrNotAuthenticated
	ErrSessionExpired   = internal.ErrSessionExpired
	ErrRateLimited      = internal.ErrRateLimited
	ErrNotFound         = internal.ErrNotFound
	ErrInvalidRequest   = internal.ErrInvalidRequest
	ErrOrderRejected    = internal.ErrOrderRejected
	ErrClosed           = internal.ErrClosed
	ErrStreamingLimit   = internal.ErrStreamingLimit
	ErrWSDisconnected   = internal.ErrWSDisconnected
	ErrWSReconnected    = internal.ErrWSReconnected

	// Deprecated: Use ErrWSDisconnected.
	ErrStreamDisconnected = ErrWSDisconnected
	// Deprecated: Use ErrWSReconnected.
	ErrStreamReconnected = ErrWSReconnected
	ErrCircuitOpen       = internal.ErrCircuitOpen
)

// SessionState is the lifecycle state of the gateway session.
type SessionState = internal.SessionState

// Session states. See docs/SESSIONS.md.
const (
	StateDisconnected  = internal.StateDisconnected
	StateInitializing  = internal.StateInitializing
	StateAuthenticated = internal.StateAuthenticated
	StateExpired       = internal.StateExpired
	StateClosed        = internal.StateClosed
)
