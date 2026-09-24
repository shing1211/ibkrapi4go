// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package fake

import (
	"sync"
	"time"

	"github.com/shing1211/ibkrapi4go/internal"
)

// Clock implements internal.Clock with programmatic time control.
type Clock struct {
	mu  sync.Mutex
	now time.Time
}

// NewClock returns a fake clock starting at t.
func NewClock(t time.Time) *Clock {
	return &Clock{now: t}
}

// Now returns the fake current time.
func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves the fake clock forward by d.
func (c *Clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// NewTicker returns a real ticker; the fake clock does not control it.
// Tests that need controlled tickers should use Advance + real time.
func (c *Clock) NewTicker(d time.Duration) *internal.Ticker {
	t := time.NewTicker(d)
	return &internal.Ticker{C: t.C, Tick: t}
}

// After returns a channel that fires after the duration relative to the fake clock.
func (c *Clock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// Sleep pauses for d using real time.
func (c *Clock) Sleep(d time.Duration) {
	time.Sleep(d)
}
