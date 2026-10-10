// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import "context"

// Pager is a generic paginated iterator. It fetches pages lazily via the
// provided fetch function and yields items one at a time.
//
// Usage:
//
//	p := NewPager(func(ctx context.Context, page int) ([]T, error) {
//	    return fetchPage(ctx, page)
//	})
//	for p.Next(ctx) {
//	    item := p.Value()
//	    // ...
//	}
//	if err := p.Err(); err != nil { ... }
//
// A Pager is not safe for concurrent use.
type Pager[T any] struct {
	fetch func(ctx context.Context, page int) ([]T, error)
	page  int
	items []T
	idx   int
	cur   T
	err   error
	done  bool
}

// NewPager creates a Pager with the given fetch function. The fetch function
// should return an empty slice when no more pages exist.
func NewPager[T any](fetch func(ctx context.Context, page int) ([]T, error)) *Pager[T] {
	return &Pager[T]{fetch: fetch, idx: -1}
}

// Next advances to the next item, fetching further pages as needed. It returns
// false when the iteration is exhausted or has failed.
func (p *Pager[T]) Next(ctx context.Context) bool {
	if p.err != nil || p.done {
		return false
	}
	p.idx++
	if p.idx < len(p.items) {
		p.cur = p.items[p.idx]
		return true
	}
	page, err := p.fetch(ctx, p.page)
	if err != nil {
		p.err = err
		return false
	}
	if len(page) == 0 {
		p.done = true
		return false
	}
	p.page++
	p.items = page
	p.idx = 0
	p.cur = p.items[0]
	return true
}

// Value returns the current item. Call it only after Next returns true.
func (p *Pager[T]) Value() T { return p.cur }

// Err returns the first error encountered, if any.
func (p *Pager[T]) Err() error { return p.err }
