// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import "context"

// PositionIterator iterates a paginated positions result. Usage:
//
//	it := cli.Portfolio().PositionsPaginated(ctx, accountID)
//	for it.Next(ctx) {
//	    p := it.Value()
//	    // ...
//	}
//	if err := it.Err(); err != nil { ... }
//
// A PositionIterator is not safe for concurrent use.
type PositionIterator struct {
	fetch func(context.Context, int) ([]Position, error)
	page  int
	buf   []Position
	idx   int
	cur   Position
	err   error
	done  bool
}

func newPositionIterator(fetch func(context.Context, int) ([]Position, error)) *PositionIterator {
	return &PositionIterator{fetch: fetch, idx: -1}
}

// Next advances to the next position, fetching further pages as needed. It
// returns false when the iteration is exhausted or has failed.
func (it *PositionIterator) Next(ctx context.Context) bool {
	if it.err != nil || it.done {
		return false
	}
	it.idx++
	if it.idx < len(it.buf) {
		it.cur = it.buf[it.idx]
		return true
	}
	page, err := it.fetch(ctx, it.page)
	if err != nil {
		it.err = err
		return false
	}
	if len(page) == 0 {
		it.done = true
		return false
	}
	it.page++
	it.buf = page
	it.idx = 0
	it.cur = it.buf[0]
	return true
}

// Value returns the current position. Call it only after Next returns true.
func (it *PositionIterator) Value() Position { return it.cur }

// Err returns the first error encountered, if any.
func (it *PositionIterator) Err() error { return it.err }
