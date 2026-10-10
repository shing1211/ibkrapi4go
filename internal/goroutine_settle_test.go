// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"
	"time"

	"go.uber.org/goleak"
)

// waitForGoroutinesToSettle polls until goleak reports no leaked goroutines, or
// until the deadline passes, and returns the final error.
//
// WSConn.Close documents a *bounded* wait: it may return while background
// goroutines are still exiting. The goroutine that observes completion and
// signals doneCh cannot signal "I have exited" from inside its own exit path,
// so there is always a brief window in which a goroutine this package started
// is still live at the moment Close returns. A bare goleak.VerifyNone races
// that window, and loses under load even though nothing leaked.
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

// settleGoroutines fails t if goroutines are still running once Close's bounded
// wait has had a chance to wind them down.
func settleGoroutines(t *testing.T) {
	t.Helper()
	if waitForGoroutinesToSettle(2*time.Second) == nil {
		return
	}
	// Re-run the vetted assertion so the failure carries goleak's own
	// goroutine report, including the stacks.
	goleak.VerifyNone(t)
}
