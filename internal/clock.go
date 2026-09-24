// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import "time"

// Clock abstracts time operations so tests can use a deterministic fake.
// The zero value is a real clock.
type Clock struct {
	nowFunc    func() time.Time
	tickerFunc func(time.Duration) *Ticker
	afterFunc  func(time.Duration) <-chan time.Time
	sleepFunc  func(time.Duration)
}

// Ticker holds a channel that fires on each tick.
type Ticker struct {
	C    <-chan time.Time
	Tick *time.Ticker
}

// Stop stops the ticker.
func (t *Ticker) Stop() {
	if t.Tick != nil {
		t.Tick.Stop()
	}
}

// Now returns the current time.
func (c *Clock) Now() time.Time {
	if c.nowFunc != nil {
		return c.nowFunc()
	}
	return time.Now()
}

// NewTicker returns a ticker that fires every duration.
func (c *Clock) NewTicker(d time.Duration) *Ticker {
	if c.tickerFunc != nil {
		return c.tickerFunc(d)
	}
	t := time.NewTicker(d)
	return &Ticker{C: t.C, Tick: t}
}

// After returns a channel that fires once after duration d.
func (c *Clock) After(d time.Duration) <-chan time.Time {
	if c.afterFunc != nil {
		return c.afterFunc(d)
	}
	return time.After(d)
}

// Sleep pauses the current goroutine for d.
func (c *Clock) Sleep(d time.Duration) {
	if c.sleepFunc != nil {
		c.sleepFunc(d)
		return
	}
	time.Sleep(d)
}
