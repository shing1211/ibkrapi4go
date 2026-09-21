// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package fake

import (
	"context"

	"github.com/shing1211/ibkrapi4go/internal"
)

// WSClient is a test double for internal.WSClient. Subscribe records calls
// and returns handles backed by the internal.NewTestWSHandle helper so
// Close is safe to call.
type WSClient struct {
	// SubscribeFn, when set, overrides the default Subscribe behaviour.
	SubscribeFn func(ctx context.Context, sink internal.WSSink, conids []int, fields []string) (*internal.WSHandle, error)
	// ActiveSubscriptionsResult is the value returned by ActiveSubscriptions.
	ActiveSubscriptionsResult int
	// CloseErr is the error returned by Close.
	CloseErr error

	// SubscribeCalls records the number of Subscribe calls.
	SubscribeCalls int
	// CloseCalls records the number of Close calls.
	CloseCalls int
}

// Subscribe satisfies internal.WSClient.
func (c *WSClient) Subscribe(ctx context.Context, sink internal.WSSink, conids []int, fields []string) (*internal.WSHandle, error) {
	c.SubscribeCalls++
	if c.SubscribeFn != nil {
		return c.SubscribeFn(ctx, sink, conids, fields)
	}
	return internal.NewTestWSHandle(), nil
}

// ActiveSubscriptions satisfies internal.WSClient.
func (c *WSClient) ActiveSubscriptions() int {
	return c.ActiveSubscriptionsResult
}

// Close satisfies internal.WSClient.
func (c *WSClient) Close() error {
	c.CloseCalls++
	return c.CloseErr
}
