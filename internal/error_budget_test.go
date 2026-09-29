// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import "testing"

// errorBudget is unexported, so it is covered here rather than in the external
// breaker tests. It needs no clock: record takes the outcome, not a timestamp, so
// the window is a pure count over the last N outcomes.

var budgetEpoch = 0 // retained only so the file's history is easy to follow

// TestErrorBudget_TripsOnFailuresWithinTheWindow pins the trip condition: budget
// failures among the last size outcomes.
func TestErrorBudget_TripsOnFailuresWithinTheWindow(t *testing.T) {
	eb := newErrorBudget(3, 5)
	if eb == nil {
		t.Fatal("newErrorBudget(3, 5) = nil; want a budget")
	}

	// Three failures inside a window of five exhausts the budget.
	for i := 1; i <= 3; i++ {
		if got := eb.record(true); got != (i == 3) {
			t.Errorf("record(failure) #%d = %v; want %v", i, got, i == 3)
		}
	}
}

// TestErrorBudget_SuccessesRecoverIt is the unit-level version of the defect the
// 2026-09-27 spike found. A window of failures alone never shrinks, so an exhausted
// budget stayed exhausted and every later failure re-tripped. Outcomes are recorded
// instead, so successes evict old failures.
func TestErrorBudget_SuccessesRecoverIt(t *testing.T) {
	eb := newErrorBudget(2, 3)

	eb.record(true)
	if !eb.record(true) {
		t.Fatal("budget of 2 was not exhausted by 2 failures")
	}

	// Three successes fill a window of three, so both failures are evicted.
	eb.record(false)
	eb.record(false)
	eb.record(false)

	if tripped := eb.record(true); tripped {
		t.Error("budget tripped on the first failure after recovery; successes did " +
			"not push the earlier failures out of the window")
	}
}

// TestErrorBudget_DisabledWhenUnset pins the nil cases, since a nil budget is
// consulted on every outcome and must never be silently substituted.
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

	// record on a nil budget must be a safe no-op: Record consults the budget on
	// every outcome, so a nil one is the default state.
	var nilBudget *errorBudget
	if nilBudget.record(true) {
		t.Error("record(true) on a nil budget returned true; want false")
	}
	if nilBudget.record(false) {
		t.Error("record(false) on a nil budget returned true; want false")
	}
}
