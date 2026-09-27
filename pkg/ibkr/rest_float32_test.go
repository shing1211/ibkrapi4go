// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import "testing"

// This file covers rest.go's float32ToStr helper.
//
// WHY IT IS TESTED DIRECTLY
// float32ToStr is unexported and reached only through
// RESTTaxVouchers.Dividends (rest.go), which calls it for every dividend's
// money and quantity fields. Its tests live here to keep the formatting rules
// in one place.
//
// The helper used to sit alongside four sibling converters — requestIDRaw,
// countriesRaw, yearsRaw and dividendsRaw — that had no production caller
// anywhere in the module and have since been deleted. Their tests went with
// them; this one stays because float32ToStr is still live.

func f32Ptr(v float32) *float32 { return &v }

func TestFloat32ToStr(t *testing.T) {
	if got := float32ToStr(nil); got != "" {
		t.Errorf("float32ToStr(nil) = %q; want empty", got)
	}
	// Formatting at float32 width yields the shortest decimal that round-trips
	// at that width, so an ordinary amount keeps its human-scale form instead of
	// the binary-float64 expansion.
	if got := float32ToStr(f32Ptr(1234.56)); got != "1234.56" {
		t.Errorf("float32ToStr(1234.56) = %q; want \"1234.56\"", got)
	}
	if got := float32ToStr(f32Ptr(0)); got != "0" {
		t.Errorf("float32ToStr(0) = %q; want \"0\"", got)
	}
	if got := float32ToStr(f32Ptr(-0.5)); got != "-0.5" {
		t.Errorf("float32ToStr(-0.5) = %q; want \"-0.5\"", got)
	}
	// No exponent form: the format verb is 'f', so a large value stays plain.
	if got := float32ToStr(f32Ptr(1e10)); got != "10000000000" {
		t.Errorf("float32ToStr(1e10) = %q; want 10000000000", got)
	}
	// The generated voucher model is *float32, so precision is already gone by
	// the time this helper sees the value: above 2^24 the nearest float32 wins.
	// Pinned so the boundary is documented rather than discovered in production.
	if got := float32ToStr(f32Ptr(16777217)); got != "16777216" {
		t.Errorf("float32ToStr(16777217) = %q; want 16777216 (nearest float32)", got)
	}
}
