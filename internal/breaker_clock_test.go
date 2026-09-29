// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"errors"
	"testing"
	"time"
)

// testClock is a controllable Clock for tests. internal.Clock's function fields are
// unexported and it has no exported constructor, so SetClock is only reachable from
// inside this package - which is where its only possible consumer lives, since
// internal/ is not importable from outside the module.
type testClock struct{ now time.Time }

func (c *testClock) advance(d time.Duration) { c.now = c.now.Add(d) }

func (c *testClock) install(b *Breaker) {
	b.SetClock(&Clock{nowFunc: func() time.Time { return c.now }})
}

// breakerEpoch is an arbitrary fixed start, so no assertion depends on the wall
// clock. The existing tests in this package wait on real time - time.Sleep(60ms)
// against a 40ms cooldown - which is both slower and racy under load. SetClock
// exists to remove exactly that, and was unused; these tests use it.

// TestBreaker_CooldownIsDrivenByTheInjectedClock pins the full
// closed -> open -> half-open -> closed cycle with no wall-clock wait, including
// both sides of the cooldown boundary: still open one nanosecond before it elapses,
// probe allowed exactly at it.
func TestBreaker_CooldownIsDrivenByTheInjectedClock(t *testing.T) {
	const cooldown = 30 * time.Second
	clk := &testClock{now: time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)}
	b := NewBreaker(2, cooldown)
	clk.install(b)

	if err := b.Allow(); err != nil {
		t.Fatalf("Allow (closed): %v", err)
	}

	b.Record(errors.New("boom"), 500)
	if err := b.Allow(); err != nil {
		t.Fatalf("Allow after 1 failure: %v", err)
	}
	b.Record(errors.New("boom"), 500)

	if err := b.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("Allow at threshold = %v; want ErrCircuitOpen", err)
	}

	clk.advance(cooldown - time.Nanosecond)
	if err := b.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("Allow just before cooldown elapsed = %v; want ErrCircuitOpen", err)
	}

	clk.advance(time.Nanosecond)
	if err := b.Allow(); err != nil {
		t.Fatalf("Allow at cooldown = %v; want the half-open probe allowed", err)
	}
	if err := b.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("second probe = %v; want ErrCircuitOpen", err)
	}

	b.Record(nil, 200)
	if err := b.Allow(); err != nil {
		t.Errorf("Allow after recovery = %v; want nil", err)
	}
}

// TestBreaker_FailedProbeReopensTheCircuit pins that a half-open probe which fails
// reopens the circuit immediately, rather than leaving it half-open.
func TestBreaker_FailedProbeReopensTheCircuit(t *testing.T) {
	const cooldown = 30 * time.Second
	clk := &testClock{now: time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)}
	b := NewBreaker(1, cooldown)
	clk.install(b)

	b.Record(errors.New("boom"), 500)
	if err := b.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("Allow while open = %v; want ErrCircuitOpen", err)
	}

	clk.advance(cooldown)
	if err := b.Allow(); err != nil {
		t.Fatalf("Allow at cooldown = %v; want a probe", err)
	}

	if err := b.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("Allow after a failed probe = %v; want ErrCircuitOpen", err)
	}
}

// TestBreaker_SetErrorBudgetTripsTheBreaker wires the budget in and pins that it
// can open the circuit on its own, below the consecutive threshold.
func TestBreaker_SetErrorBudgetTripsTheBreaker(t *testing.T) {
	// Consecutive threshold is high, so only the budget can trip it. Successes
	// between failures reset the consecutive counter, leaving the budget as the
	// only thing accumulating.
	clk := &testClock{now: time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)}
	b := NewBreaker(100, time.Minute)
	clk.install(b)
	b.SetErrorBudget(3, 3)

	for i := 1; i <= 2; i++ {
		b.Record(errors.New("boom"), 500)
		b.Record(nil, 200)
		if err := b.Allow(); err != nil {
			t.Fatalf("Allow after failure %d = %v; the budget tripped early", i, err)
		}
	}

	b.Record(errors.New("boom"), 500)
	if err := b.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("Allow = %v; want ErrCircuitOpen once the budget of 3 is exhausted", err)
	}
}

// TestBreaker_SetClockNilFallsBackToRealTime pins the nil path, which Record and
// Allow both take on every call. A nil clock must mean the real clock, not the
// zero time - the zero time would make every cooldown already elapsed.
//
// What this cannot do is detect that specific regression. Replacing the fallback
// with the zero time still leaves the breaker refusing requests, because
// openUntil = zeroTime + cooldown is in the future relative to zeroTime, so the
// observable behaviour is identical until an hour of real time has passed. Making
// the difference visible would mean waiting an hour, which is the thing SetClock
// exists to avoid. The assertion below is therefore weaker than it looks and is
// labelled as such: it pins that a nil clock does not panic and keeps the circuit
// open, not that it returns the wall clock.
func TestBreaker_SetClockNilFallsBackToRealTime(t *testing.T) {
	b := NewBreaker(1, time.Hour)
	b.SetClock(nil) // documented as "a nil clock uses the real clock"

	b.Record(errors.New("boom"), 500)
	if err := b.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("Allow = %v; want ErrCircuitOpen", err)
	}
	if err := b.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("Allow immediately after tripping = %v; want ErrCircuitOpen", err)
	}
}
