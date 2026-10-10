// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"net/http"

	"github.com/shing1211/ibkrapi4go/internal"
)

// Middleware is a function that wraps an http.RoundTripper, adding behaviour
// such as logging, tracing, custom authentication, or retry logic. It is
// equivalent to internal.Middleware and re-exported here for convenience.
type Middleware = internal.Middleware

// WithTransportMiddleware adds a middleware to the HTTP transport chain.
// Middleware wraps the transport in the order provided. The first middleware
// in the list wraps the outermost layer; the last wraps the innermost.
//
// User middleware sits between the SDK's ErrorDecode middleware and the base
// HTTP transport — so all SDK layers (request-id, user-agent, auth, logging,
// metrics, circuit breaker, retry, rate-limit, timeout, error decode) execute
// before any user middleware.
//
// Example:
//
//	cli, _ := ibkr.NewClient(
//	    ibkr.WithGatewayURL("https://localhost:5000"),
//	    ibkr.WithTransportMiddleware(func(next http.RoundTripper) http.RoundTripper {
//	        return &loggingTransport{next: next}
//	    }),
//	)
func WithTransportMiddleware(mw func(http.RoundTripper) http.RoundTripper) Option {
	return func(c *config) error {
		c.userMiddleware = append(c.userMiddleware, mw)
		return nil
	}
}
