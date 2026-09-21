// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import "time"

// WithEndpointTimeout sets a default timeout for all requests.
// This is overridden by context deadlines set on individual requests.
func WithEndpointTimeout(d time.Duration) Option {
	return func(c *config) error {
		if d < 0 {
			return &ConfigError{Field: "EndpointTimeout", Message: "must not be negative"}
		}
		c.endpointTimeout = d
		return nil
	}
}
