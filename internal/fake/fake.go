// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Package fake provides test doubles for internal interfaces. Every type
// implements exactly one interface (TokenProvider, SessionMachine, WSClient,
// or RateLimiter) and is safe for concurrent use. Fields may be set after
// construction and before the first call.
//
// Usage:
//
//	func TestSomething(t *testing.T) {
//	    tp := &fake.TokenProvider{Token: "test-token"}
//	    // inject tp into the component under test
//	}
package fake
