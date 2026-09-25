// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Package ibkr is an idiomatic Go client for the Interactive Brokers Client
// Portal API (CPAPI). It wraps the generated OpenAPI client with a stable,
// hand-written public API: a Client composition root, session management, and
// domain managers.
//
// v1 covers both the Client Portal API (CPAPI, `/v1/api/*`, `ssoBearer`) and the IB REST API (`/gw/api/*`, `oauth2Bearer`).
//
// Monetary and quantity values are decimal strings, never float64, to preserve
// precision (ADR 0008).
//
// # Quick start
//
// The Client Portal Gateway must already be authenticated in a browser before
// the SDK can connect. The SDK does not perform login.
//
//	ctx := context.Background()
//	cli, err := ibkr.NewClient(ibkr.WithGatewayURL("https://localhost:5000"))
//	if err != nil {
//	    return err
//	}
//	defer cli.Close()
//
//	if err := cli.Session().Initialize(ctx); err != nil {
//	    return err
//	}
//
//	accounts, err := cli.Account().List(ctx)
//	if err != nil {
//	    return err
//	}
//
//	for _, acc := range accounts {
//	    fmt.Println(acc.AccountID)
//	}
//
// # Full documentation
//
// https://pkg.go.dev/github.com/shing1211/ibkrapi4go/pkg/ibkr
package ibkr

import "runtime"

// Version is the SDK version.
const Version = "v1.0.7"

// GoVersion is the Go runtime version the SDK was built with.
var GoVersion = runtime.Version()

// DebugInfo returns a human-readable version string.
func DebugInfo() string {
	return "ibkrapi4go " + Version + " (built with " + GoVersion + ")"
}
