// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package fake

import "context"

// RateLimiter is a test double for internal.RateLimiter. By default it allows
// every request immediately. Set WaitFn to inject delays or errors.
type RateLimiter struct {
	// WaitFn, when set, is called by Wait. It receives the context, method,
	// and path and returns an error.
	WaitFn func(ctx context.Context, method, path string) error
	// WaitCalls records the number of times Wait was called.
	WaitCalls int
	// WaitMethods records the methods passed to Wait (in order).
	WaitMethods []string
	// WaitPaths records the paths passed to Wait (in order).
	WaitPaths []string
}

// Wait satisfies internal.RateLimiter.
func (l *RateLimiter) Wait(ctx context.Context, method, path string) error {
	l.WaitCalls++
	l.WaitMethods = append(l.WaitMethods, method)
	l.WaitPaths = append(l.WaitPaths, path)
	if l.WaitFn != nil {
		return l.WaitFn(ctx, method, path)
	}
	return nil
}
