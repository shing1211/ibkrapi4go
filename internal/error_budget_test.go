// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"
	"time"
)

// errorBudget is unexported, so it is covered here rather than in the external
// breaker tests. It needs no fake clock: record takes the time as a parameter,
// which is what makes the discrepancy below straightforward to pin.

var budgetEpoch = time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)

// TestErrorBudget_CountsFailuresNotElapsedTime pins the budget's actual behaviour,
// which is narrower than its documentation claims.
//
// errorBudget is described as a "sliding window", and evict as removing "entries
// that have slid out of the window" - but evict is passed now and never reads it.
// It bounds the slice by count only. The budget is therefore a count of recent
// failures with no time component at all.
//
// This matters to an operator: configuring "5 failures in 60s" does not yield 5
// failures in 60s, it yields 5 failures out of the last N recorded, however far
// apart they were. Correcting it is a behaviour change and needs an ADR; this test
// exists so the discrepancy is visible in the suite and not only in the source.
func TestErrorBudget_CountsFailuresNotElapsedTime(t *testing.T) {
	eb := newErrorBudget(3, 3)
	if eb == nil {
		t.Fatal("newErrorBudget(3, 3) = nil; want a budget")
	}

	// Three failures an hour apart still exhaust the budget, because nothing in
	// the implementation consults elapsed time.
	now := budgetEpoch
	for i := 1; i <= 3; i++ {
		now = now.Add(time.Hour)
		if got := eb.record(now); got != (i == 3) {
			t.Errorf("record #%d an hour apart = %v; want %v", i, got, i == 3)
		}
	}
}

// TestErrorBudget_EvictBoundsByCount pins the one thing evict does do: bound the
// slice by size. Without it a long-lived breaker would grow without limit.
func TestErrorBudget_EvictBoundsByCount(t *testing.T) {
	const size = 4
	eb := newErrorBudget(size+1, size) // budget above size, so only the bound trips

	now := budgetEpoch
	for i := 0; i < 20; i++ {
		now = now.Add(time.Nanosecond)
		eb.record(now)
	}

	eb.mu.Lock()
	got := len(eb.window)
	eb.mu.Unlock()
	if got != size {
		t.Errorf("window holds %d entries after 20 records; want %d - evict bounds "+
			"by count", got, size)
	}
}

// TestErrorBudget_DisabledWhenUnset pins the nil cases, since a nil budget is
// consulted on every failure and must never be silently substituted.
func TestErrorBudget_DisabledWhenUnset(t *testing.T) {
	for _, tc := range []struct {
		name         string
		budget, size int
	}{
		{"budget zero", 0, 5},
		{"budget negative", -1, 5},
		{"size zero", 3, 0},
		{"size negative", 3, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := newErrorBudget(tc.budget, tc.size); got != nil {
				t.Errorf("newErrorBudget(%d, %d) = %+v; want nil", tc.budget, tc.size, got)
			}
		})
	}

	// record on a nil budget must be a safe no-op: Record calls it on every
	// failure when no budget is installed.
	var nilBudget *errorBudget
	if nilBudget.record(budgetEpoch) {
		t.Error("record on a nil budget returned true; want false")
	}
}
