// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestIBKRPrintln_NilEnv covers the guard. Every subcommand now receives an
// explicit env rather than reading the process, so a nil env should not happen -
// but the guard exists precisely because it did, and a guard that is never
// exercised is one that can be deleted or inverted without anyone noticing.
func TestIBKRPrintln_NilEnv(t *testing.T) {
	// Must not panic, and must not write to the process stdout.
	ibkrPrintln(nil, "dropped")

	e := &env{}
	ibkrPrintln(e, "dropped: no writer")

	e.stdout = nil
	ibkrPrintln(e, "dropped: nil writer")
}

func TestIBKRPrintln_WritesToInjectedWriter(t *testing.T) {
	var out bytes.Buffer
	ibkrPrintln(&env{stdout: &out}, "hello")
	if got := out.String(); got != "hello\n" {
		t.Errorf("wrote %q; want %q", got, "hello\n")
	}
}

// TestTruncate covers the boundary, which is where an off-by-one hides. The
// condition is len(s) <= n, so a string of exactly n is returned untouched while
// n+1 is cut to n-1 runes plus the ellipsis. Both sides are asserted because a
// change to <= or < produces a length of n+1 that a length-only check would miss.
func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		in   string
		n    int
		want string
	}{
		{"shorter than the limit", "AAPL", 30, "AAPL"},
		{"exactly the limit", "12345", 5, "12345"},
		{"one over the limit", "123456", 5, "1234…"},
		{"well over the limit", "Apple Inc Class A Common Stock", 10, "Apple Inc…"},
		{"empty", "", 10, ""},
		// A zero limit would make s[:n-1] a negative index and panic. The
		// positions table calls truncate with a fixed 30, so this is not
		// reachable today, but the function is exported to the file, not the
		// package, and the boundary is worth pinning.
		{"zero limit on empty input", "", 0, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := truncate(tc.in, tc.n)
			if got != tc.want {
				t.Errorf("truncate(%q, %d) = %q; want %q", tc.in, tc.n, got, tc.want)
			}
		})
	}
}

// TestTruncate_NeverExceedsLimit states the invariant a caller depends on: the
// result must fit the column width, because the value is printed with a %-12s
// format verb into a fixed-width table. A result of n+1 runes would wrap the row.
func TestTruncate_NeverExceedsLimit(t *testing.T) {
	const limit = 12
	for _, s := range []string{"", "A", "Apple", "Apple Inc Class A Common Stock", strings.Repeat("X", 100)} {
		got := truncate(s, limit)
		if n := len([]rune(got)); n > limit {
			t.Errorf("truncate(%q, %d) = %q, which is %d runes; want at most %d", s, limit, got, n, limit)
		}
	}
}
