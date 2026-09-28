// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"sync"
	"testing"
	"time"
)

// manualTicker is a Clock whose tickles fire only when the test fires them.
//
// The session's tickle loop selects on a ticker built from this clock, so a test
// can drive an exact number of rounds and know when each has been served. The
// session tests that used to sleep did so because they had no way to advance the
// loop: they guessed an interval, slept a multiple of it, and asserted on
// whatever the state happened to be. That is a race whose only failure mode is a
// slow machine.
//
// Each NewTicker call gets its own channel, and tick delivers to all of them.
// That is what makes a double-start detectable: if two loops were running, one
// tick would be consumed by both and produce two rounds instead of one. A clock
// that handed every loop the same channel would hide exactly that bug.
type manualTicker struct {
	mu    sync.Mutex
	chans []chan time.Time
}

func newManualClock() (*Clock, *manualTicker) {
	mt := &manualTicker{}
	return &Clock{
		tickerFunc: func(time.Duration) *Ticker {
			ch := make(chan time.Time, 64)
			mt.mu.Lock()
			mt.chans = append(mt.chans, ch)
			mt.mu.Unlock()
			return &Ticker{C: ch}
		},
	}, mt
}

// tick delivers one tickle to every registered ticker. The channels are buffered
// and drained by their own goroutine, so this cannot block on a loop that is
// still starting up.
func (m *manualTicker) tick() {
	m.mu.Lock()
	chans := append([]chan time.Time(nil), m.chans...)
	m.mu.Unlock()
	for _, ch := range chans {
		ch <- time.Now()
	}
}

// count reports how many tickers have been handed out, i.e. how many loops are
// running.
func (m *manualTicker) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.chans)
}

// waitForTicker blocks until at least one ticker exists. The loop builds its
// ticker inside its goroutine, so a test that starts the loop and immediately
// ticks would otherwise race the goroutine's startup.
func (m *manualTicker) waitForTicker(t *testing.T) {
	t.Helper()
	waitFor(t, "the tickle loop to start", func() bool { return m.count() > 0 })
}

// waitFor polls cond until it holds, and fails the test naming what it waited for
// if the budget runs out.
//
// This is not the same as sleeping for a fixed interval and then looking. A fixed
// sleep asserts "whatever happened by T was true"; this asserts "the condition
// became true, and I will wait up to the budget for it". The poll interval is a
// backoff, not an expectation.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out after 2s waiting for %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

// waitForTickles blocks until the fake API has served n tickle calls.
func waitForTickles(t *testing.T, api *fakeAPI, n int64) {
	t.Helper()
	waitFor(t, "the tickle loop to run", func() bool { return api.tickleCalls.Load() >= n })
}
