// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Package ibkr_test contains CPU and memory profiling tests for hot paths.
//
// Run: go test -bench=BenchmarkProfile -benchmem ./pkg/ibkr/... -benchtime=3s

package ibkr_test

import (
	"encoding/json"
	"testing"
)

func BenchmarkProfileJSONDecode(b *testing.B) {
	types := []struct {
		name string
		data []byte
		typ  any
	}{
		{"AccountSummary", accountSummaryJSON, &FixtureAccountSummary{}},
		{"Position", positionJSON, &FixturePosition{}},
		{"Order", orderJSON, &FixtureOrder{}},
		{"Trade", tradeJSON, &FixtureTrade{}},
		{"Contract", contractJSON, &FixtureContract{}},
		{"Snapshot", snapshotJSON, &[]FixtureSnapshot{}},
		{"History", historyJSON, &FixtureHistory{}},
		{"AccountPnL", accountPnlJSON, &FixtureAccountPnL{}},
	}

	for _, tc := range types {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(tc.data)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				v := cloneForProfile(tc.typ)
				json.Unmarshal(tc.data, v)
			}
		})
	}
}

func BenchmarkProfileJSONEncode(b *testing.B) {
	benchmarks := []struct {
		name string
		val  any
	}{
		{"AccountSummary", accountSummary},
		{"Position", position},
		{"Order", order},
		{"Trade", trade},
		{"Contract", contract},
		{"Snapshot", snapshot},
		{"History", history},
		{"AccountPnL", accountPnl},
	}

	for _, tc := range benchmarks {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				json.Marshal(tc.val)
			}
		})
	}
}

func cloneForProfile(v any) any {
	switch v := v.(type) {
	case *FixtureAccountSummary:
		return &FixtureAccountSummary{}
	case *FixturePosition:
		return &FixturePosition{}
	case *FixtureOrder:
		return &FixtureOrder{}
	case *FixtureTrade:
		return &FixtureTrade{}
	case *FixtureContract:
		return &FixtureContract{}
	case *[]FixtureSnapshot:
		return &[]FixtureSnapshot{}
	case *FixtureHistory:
		return &FixtureHistory{}
	case *FixtureAccountPnL:
		return &FixtureAccountPnL{}
	default:
		return v
	}
}
