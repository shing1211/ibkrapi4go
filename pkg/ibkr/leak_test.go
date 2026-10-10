// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"testing"
	"time"

	"go.uber.org/goleak"
)

// waitForGoroutinesToSettle polls until goleak reports no leaked goroutines, or
// until the deadline passes, and returns the final error.
//
// Client.Close, like internal.WSConn.Close, documents a *bounded* wait: it may
// return while background goroutines are still exiting. A goroutine cannot
// signal "I have exited" from inside its own exit path, so there is always a
// brief window in which one is still live when Close returns. A bare
// goleak.VerifyNone races that window and loses under load even though nothing
// leaked.
//
// Polling until the goroutines clear separates the two cases: a goroutine that
// is merely winding down exits on its own, while a real leak never does. The
// deadline bounds the wait, so a genuine leak still fails rather than hangs.
func waitForGoroutinesToSettle(within time.Duration) error {
	deadline := time.Now().Add(within)
	var err error
	for {
		if err = goleak.Find(); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// assertNoLeaks fails t if goroutines are still running once the test's own
// cleanup has had a chance to wind them down.
//
// Register it with t.Cleanup as the *first* cleanup a test performs, so that it
// runs last and observes the state after everything else has been torn down.
func assertNoLeaks(t *testing.T) {
	t.Helper()
	if waitForGoroutinesToSettle(2*time.Second) == nil {
		return
	}
	// Re-run the vetted assertion so the failure carries goleak's own report.
	goleak.VerifyNone(t)
}

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
// with nothing outstanding the poll must return promptly, not burn its budget.
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
