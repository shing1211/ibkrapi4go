// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"
	"time"
)

// TestWaitForGoroutinesToSettle_ReportsRealLeak proves the poll is not vacuous.
// A goroutine that never exits must be reported, not quietly tolerated.
func TestWaitForGoroutinesToSettle_ReportsRealLeak(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		<-stop
	}()

	if err := waitForGoroutinesToSettle(200 * time.Millisecond); err == nil {
		t.Fatal("expected a permanently blocked goroutine to be reported as a leak")
	}
}

// TestWaitForGoroutinesToSettle_CleanWhenNothingLeaked is the other direction:
// with nothing outstanding the poll must return nil promptly, not burn its
// whole budget.
func TestWaitForGoroutinesToSettle_CleanWhenNothingLeaked(t *testing.T) {
	start := time.Now()
	if err := waitForGoroutinesToSettle(2 * time.Second); err != nil {
		t.Fatalf("unexpected leak reported: %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("took %s to report a clean result; should return immediately", elapsed)
	}
}

// TestWaitForGoroutinesToSettle_ToleratesSlowShutdown covers the case the helper
// exists for: a goroutine that exits, but not instantly.
func TestWaitForGoroutinesToSettle_ToleratesSlowShutdown(t *testing.T) {
	go func() {
		time.Sleep(150 * time.Millisecond)
	}()

	if err := waitForGoroutinesToSettle(2 * time.Second); err != nil {
		t.Fatalf("a goroutine that exited on its own was reported as a leak: %v", err)
	}
}
