// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"github.com/shing1211/ibkrapi4go/internal"
)

// Error is the SDK error type. All API failures surface as *Error.
type Error = internal.Error

// Sentinel errors.
var (
	ErrNotAuthenticated = internal.ErrNotAuthenticated
	ErrSessionExpired   = internal.ErrSessionExpired
	ErrRateLimited      = internal.ErrRateLimited
	ErrNotFound         = internal.ErrNotFound
	ErrInvalidRequest   = internal.ErrInvalidRequest
	ErrOrderRejected    = internal.ErrOrderRejected
	ErrClosed           = internal.ErrClosed
)
