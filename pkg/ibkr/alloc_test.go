// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Package ibkr_test contains allocation counting tests for hot paths.
//
// Run: go test -bench=BenchmarkAlloc -benchmem ./pkg/ibkr/... -benchtime=1s

package ibkr_test

import (
	"encoding/json"
	"testing"
)

func BenchmarkAllocJSONDecodeAccountSummary(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v FixtureAccountSummary
		json.Unmarshal(accountSummaryJSON, &v)
	}
}

func BenchmarkAllocJSONDecodePosition(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v FixturePosition
		json.Unmarshal(positionJSON, &v)
	}
}

func BenchmarkAllocJSONDecodeOrder(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v FixtureOrder
		json.Unmarshal(orderJSON, &v)
	}
}

func BenchmarkAllocJSONDecodeTrade(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v FixtureTrade
		json.Unmarshal(tradeJSON, &v)
	}
}

func BenchmarkAllocJSONDecodeContract(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v FixtureContract
		json.Unmarshal(contractJSON, &v)
	}
}

func BenchmarkAllocJSONDecodeSnapshot(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v []FixtureSnapshot
		json.Unmarshal(snapshotJSON, &v)
	}
}

func BenchmarkAllocJSONDecodeHistory(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v FixtureHistory
		json.Unmarshal(historyJSON, &v)
	}
}

func BenchmarkAllocJSONDecodeAccountPnL(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v FixtureAccountPnL
		json.Unmarshal(accountPnlJSON, &v)
	}
}

func BenchmarkAllocJSONEncodeAccountSummary(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(accountSummary)
	}
}

func BenchmarkAllocJSONEncodePosition(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(position)
	}
}

func BenchmarkAllocJSONEncodeOrder(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(order)
	}
}

func BenchmarkAllocJSONEncodeTrade(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(trade)
	}
}
