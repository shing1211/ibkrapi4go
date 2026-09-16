// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
)

type AuthStatus struct {
	Authenticated bool
	Connected     bool
	Established   bool
	Fail          string
}

func (c *Client) AuthStatus(ctx context.Context) (*AuthStatus, error) {
	resp, err := c.generated.GetBrokerageStatusWithResponse(ctx)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, &Error{Code: "auth_status", Message: "unexpected response"}
	}
	bs := resp.JSON200
	return &AuthStatus{
		Authenticated: bs.Authenticated != nil && *bs.Authenticated,
		Connected:     bs.Connected != nil && *bs.Connected,
		Established:   bs.Established != nil && *bs.Established,
		Fail:          derefString(bs.Fail, ""),
	}, nil
}

func derefString(s *string, def string) string {
	if s == nil {
		return def
	}
	return *s
}
