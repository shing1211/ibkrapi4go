// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import "runtime"

const Version = "v0.1.0"

var (
	GoVersion = runtime.Version()
)

func DebugInfo() string {
	return "ibkrapi4go " + Version + " (built with " + GoVersion + ")"
}
