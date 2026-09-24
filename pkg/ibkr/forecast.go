// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

// ForecastManager exposes forecast (event contract) operations. It is safe for
// concurrent use.
type ForecastManager struct {
	client *Client
}
