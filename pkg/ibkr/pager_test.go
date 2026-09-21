// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"errors"
	"testing"
)

func TestPagerSinglePage(t *testing.T) {
	data := []string{"a", "b", "c"}
	p := NewPager(func(_ context.Context, page int) ([]string, error) {
		if page > 0 {
			return nil, nil
		}
		return data, nil
	})

	var got []string
	for p.Next(context.Background()) {
		got = append(got, p.Value())
	}
	if err := p.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 items, got %d", len(got))
	}
	for i, want := range data {
		if got[i] != want {
			t.Errorf("item %d: got %q, want %q", i, got[i], want)
		}
	}
}

func TestPagerMultiplePages(t *testing.T) {
	pages := [][]string{
		{"a", "b"},
		{"c"},
		{},
	}
	p := NewPager(func(_ context.Context, page int) ([]string, error) {
		if page < len(pages) {
			return pages[page], nil
		}
		return nil, nil
	})

	var got []string
	for p.Next(context.Background()) {
		got = append(got, p.Value())
	}
	if err := p.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 items, got %d: %v", len(got), got)
	}
	for i, want := range []string{"a", "b", "c"} {
		if got[i] != want {
			t.Errorf("item %d: got %q, want %q", i, got[i], want)
		}
	}
}

func TestPagerEmptyFirstPage(t *testing.T) {
	p := NewPager(func(_ context.Context, _ int) ([]string, error) {
		return nil, nil
	})

	if p.Next(context.Background()) {
		t.Fatal("expected false for empty pager")
	}
	if err := p.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPagerError(t *testing.T) {
	wantErr := errors.New("fetch failed")
	p := NewPager(func(_ context.Context, page int) ([]string, error) {
		if page == 0 {
			return []string{"ok"}, nil
		}
		return nil, wantErr
	})

	// First page succeeds.
	if !p.Next(context.Background()) {
		t.Fatal("expected first item")
	}
	if got := p.Value(); got != "ok" {
		t.Fatalf("got %q, want %q", got, "ok")
	}
	// Second page fails.
	if p.Next(context.Background()) {
		t.Fatal("expected false after error")
	}
	if !errors.Is(p.Err(), wantErr) {
		t.Fatalf("got error %v, want %v", p.Err(), wantErr)
	}
}

func TestPagerErrorImmediate(t *testing.T) {
	wantErr := errors.New("fetch failed")
	p := NewPager(func(_ context.Context, _ int) ([]string, error) {
		return nil, wantErr
	})

	if p.Next(context.Background()) {
		t.Fatal("expected false on error")
	}
	if !errors.Is(p.Err(), wantErr) {
		t.Fatalf("got error %v, want %v", p.Err(), wantErr)
	}
}

func TestPagerIntegerType(t *testing.T) {
	pages := [][]int{
		{1, 2, 3},
		{4, 5},
	}
	p := NewPager(func(_ context.Context, page int) ([]int, error) {
		if page < len(pages) {
			return pages[page], nil
		}
		return nil, nil
	})

	var got []int
	for p.Next(context.Background()) {
		got = append(got, p.Value())
	}
	if err := p.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("expected 5 items, got %d", len(got))
	}
	for i, want := range []int{1, 2, 3, 4, 5} {
		if got[i] != want {
			t.Errorf("item %d: got %d, want %d", i, got[i], want)
		}
	}
}

func TestPagerStructType(t *testing.T) {
	type item struct {
		Name string
		Val  int
	}
	data := []item{{Name: "x", Val: 1}, {Name: "y", Val: 2}}
	p := NewPager(func(_ context.Context, page int) ([]item, error) {
		if page > 0 {
			return nil, nil
		}
		return data, nil
	})

	var got []item
	for p.Next(context.Background()) {
		got = append(got, p.Value())
	}
	if err := p.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	if got[0].Name != "x" || got[0].Val != 1 {
		t.Errorf("item 0: got %+v", got[0])
	}
	if got[1].Name != "y" || got[1].Val != 2 {
		t.Errorf("item 1: got %+v", got[1])
	}
}
