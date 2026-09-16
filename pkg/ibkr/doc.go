// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Package ibkr is an idiomatic Go client for the Interactive Brokers Client
// Portal API (CPAPI). It wraps the generated OpenAPI client with a stable,
// hand-written public API: a Client composition root, session management, and
// domain managers.
//
// v1 targets the CPAPI surface (`/v1/api/*`, `ssoBearer`) only. See
// docs/ARCHITECTURE.md and docs/adr/0005-v1-scope.md.
//
// Monetary and quantity values are decimal strings, never float64, to preserve
// precision (ADR 0008).
package ibkr

import "runtime"

// Version is the SDK version.
const Version = "v0.1.0"

// GoVersion is the Go runtime version the SDK was built with.
var GoVersion = runtime.Version()

// DebugInfo returns a human-readable version string.
func DebugInfo() string {
	return "ibkrapi4go " + Version + " (built with " + GoVersion + ")"
}
