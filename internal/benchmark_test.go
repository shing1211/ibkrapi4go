// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Package internal_test contains HTTP RT, WebSocket, and session-init
// benchmarks for the internal package.
//
// BASELINE (commit 6c5023b, Intel Celeron N5105 @ 2.00GHz, Linux 6.8)
//
// HTTP round-trip latency (mock gateway, in-memory):
//   Account.List          p50:  99,529 us  p95: 102,646 us  p99: 582,181 us
//   Account.Summary       p50:  99,668 us  p95: 102,418 us  p99: 103,250 us
//   Trade.Submit          p50:  99,645 us  p95: 102,298 us  p99: 104,560 us
//   MarketData.Snapshot   p50:  99,463 us  p95: 102,240 us  p99: 103,406 us
//   Portfolio.Positions   p50:  99,559 us  p95: 101,932 us  p99: 105,670 us
//   Portfolio.Ledger      p50:  99,400 us  p95: 102,687 us  p99: 104,134 us
//
// Session init latency (cold NewClient + Initialize):
//   Session.Initialize    p50: 2,005,931 us  p95: 2,005,931 us  p99: 2,005,931 us
//
// WebSocket subscribe/unsubscribe (10 conids):
//   SubscribeUnsubscribe  1,474,498 ns/op  138,569 B/op  703 allocs/op
//   SubscribeReceiveTick  1,731,482 ns/op  138,225 B/op  699 allocs/op
//
// WS message decode:
//   Dispatch              3,113 ns/op  16.71 MB/s  688 B/op  15 allocs/op
//
// Note: HTTP p50/p95 latencies are ~100ms due to the mock's auth-status poll
// loop (1s sleep + 1s poll interval in Session.Initialize). The actual mock
// response is immediate. This is the SDK's session-init overhead, not the
// network path.
//
// Run: go test -bench=. ./internal/... -benchtime=1s -count=1

package internal_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/shing1211/ibkrapi4go/internal"
	"github.com/shing1211/ibkrapi4go/internal/mockgateway"
	"github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

func startMockServer() *httptest.Server {
	return httptest.NewServer(mockgateway.New().Handler())
}

func newTestClient(b *testing.B, serverURL string) *ibkr.Client {
	b.Helper()
	c, err := ibkr.NewClient(
		ibkr.WithGatewayURL(serverURL),
		ibkr.WithInsecureSkipVerify(true),
		ibkr.WithRequestTimeout(10*time.Second),
	)
	if err != nil {
		b.Fatalf("NewClient: %v", err)
	}
	return c
}

func measureLatency(b *testing.B, name string, fn func() error) {
	b.Run(name, func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		n := b.N
		latencies := make([]time.Duration, 0, n)
		for i := 0; i < n; i++ {
			start := time.Now()
			if err := fn(); err != nil {
				b.Fatalf("%s error: %v", name, err)
			}
			latencies = append(latencies, time.Since(start))
		}
		sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
		p50 := latencies[n*50/100]
		p95 := latencies[n*95/100]
		p99 := latencies[n*99/100]
		b.ReportMetric(float64(p50.Microseconds()), "p50_latency_us")
		b.ReportMetric(float64(p95.Microseconds()), "p95_latency_us")
		b.ReportMetric(float64(p99.Microseconds()), "p99_latency_us")
	})
}

func BenchmarkHTTPAccountList(b *testing.B) {
	srv := startMockServer()
	defer srv.Close()
	c := newTestClient(b, srv.URL)

	ctx := context.Background()
	accounts, err := c.Account().List(ctx)
	if err != nil {
		b.Fatalf("Account.List warmup: %v", err)
	}
	if len(accounts) == 0 {
		b.Fatalf("no accounts returned")
	}

	measureLatency(b, "Account.List", func() error {
		_, err := c.Account().List(ctx)
		return err
	})
}

func BenchmarkHTTPAccountSummary(b *testing.B) {
	srv := startMockServer()
	defer srv.Close()
	c := newTestClient(b, srv.URL)

	ctx := context.Background()
	accounts, err := c.Account().List(ctx)
	if err != nil {
		b.Fatalf("Account.List warmup: %v", err)
	}
	if len(accounts) == 0 {
		b.Fatalf("no accounts returned")
	}

	accountID := accounts[0].ID
	summary, err := c.Account().Summary(ctx, accountID)
	if err != nil {
		b.Fatalf("Account.Summary warmup: %v", err)
	}
	if summary == nil {
		b.Fatalf("no summary returned")
	}

	measureLatency(b, "Account.Summary", func() error {
		_, err := c.Account().Summary(ctx, accountID)
		return err
	})
}

func BenchmarkHTTPTradeSubmit(b *testing.B) {
	srv := startMockServer()
	defer srv.Close()
	c := newTestClient(b, srv.URL)

	ctx := context.Background()
	accounts, err := c.Account().List(ctx)
	if err != nil {
		b.Fatalf("Account.List warmup: %v", err)
	}
	if len(accounts) == 0 {
		b.Fatalf("no accounts returned")
	}
	accountID := accounts[0].ID

	req := ibkr.OrderRequest{
		ConID:       265598,
		Side:        ibkr.SideBuy,
		Quantity:    "1",
		OrderType:   ibkr.OrderTypeLimit,
		LimitPrice:  "150.00",
		TimeInForce: ibkr.TimeInForceDay,
	}
	result, err := c.Trade().Submit(ctx, accountID, req)
	if err != nil {
		b.Fatalf("Trade.Submit warmup: %v", err)
	}
	if result == nil {
		b.Fatalf("no submit result returned")
	}

	measureLatency(b, "Trade.Submit", func() error {
		_, err := c.Trade().Submit(ctx, accountID, req)
		return err
	})
}

func BenchmarkHTTPMarketDataSnapshot(b *testing.B) {
	srv := startMockServer()
	defer srv.Close()
	c := newTestClient(b, srv.URL)

	ctx := context.Background()
	snapshots, err := c.MarketData().Snapshot(ctx, []ibkr.ConID{265598}, []ibkr.Field{ibkr.FieldLastPrice})
	if err != nil {
		b.Fatalf("MarketData.Snapshot warmup: %v", err)
	}
	if len(snapshots) == 0 {
		b.Fatalf("no snapshots returned")
	}

	measureLatency(b, "MarketData.Snapshot", func() error {
		_, err := c.MarketData().Snapshot(ctx, []ibkr.ConID{265598}, []ibkr.Field{ibkr.FieldLastPrice})
		return err
	})
}

func BenchmarkHTTPPortfolioPositions(b *testing.B) {
	srv := startMockServer()
	defer srv.Close()
	c := newTestClient(b, srv.URL)

	ctx := context.Background()
	accounts, err := c.Account().List(ctx)
	if err != nil {
		b.Fatalf("Account.List warmup: %v", err)
	}
	if len(accounts) == 0 {
		b.Fatalf("no accounts returned")
	}
	accountID := accounts[0].ID

	positions, err := c.Portfolio().Positions(ctx, accountID)
	if err != nil {
		b.Fatalf("Portfolio.Positions warmup: %v", err)
	}
	if len(positions) == 0 {
		b.Fatalf("no positions returned")
	}

	measureLatency(b, "Portfolio.Positions", func() error {
		_, err := c.Portfolio().Positions(ctx, accountID)
		return err
	})
}

func BenchmarkHTTPPortfolioLedger(b *testing.B) {
	srv := startMockServer()
	defer srv.Close()
	c := newTestClient(b, srv.URL)

	ctx := context.Background()
	accounts, err := c.Account().List(ctx)
	if err != nil {
		b.Fatalf("Account.List warmup: %v", err)
	}
	if len(accounts) == 0 {
		b.Fatalf("no accounts returned")
	}
	accountID := accounts[0].ID

	ledger, err := c.Portfolio().Ledger(ctx, accountID)
	if err != nil {
		b.Fatalf("Portfolio.Ledger warmup: %v", err)
	}
	if len(ledger) == 0 {
		b.Fatalf("no ledger returned")
	}

	measureLatency(b, "Portfolio.Ledger", func() error {
		_, err := c.Portfolio().Ledger(ctx, accountID)
		return err
	})
}

func BenchmarkSessionInit(b *testing.B) {
	ctx := context.Background()
	measureLatency(b, "Session.Initialize", func() error {
		srv := startMockServer()
		defer srv.Close()
		c := newTestClient(b, srv.URL)
		return c.Session().Initialize(ctx)
	})
}

type wsSink struct {
	msgs int
}

func (s *wsSink) Wants(conid int) bool                     { return true }
func (s *wsSink) Deliver(u internal.WSUpdate)             { s.msgs++ }
func (s *wsSink) Fail(err error)                          {}
func (s *wsSink) WantsSystem() bool                       { return false }
func (s *wsSink) DeliverSystem(frame internal.WSSystemFrame) {}

func BenchmarkWSSubscribeUnsubscribe(b *testing.B) {
	b.Run("SubscribeUnsubscribe", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			script := mockgateway.StreamScript{
				OnSubscribe: func(conids []int, fields []string) []mockgateway.Tick {
					ticks := make([]mockgateway.Tick, 0, len(conids))
					for _, c := range conids {
						ticks = append(ticks, mockgateway.Tick{ConID: c, Field: "31", Value: "150.25"})
					}
					return ticks
				},
			}
			srv := mockgateway.New(mockgateway.WithStreamScript(&script))
			hs := httptest.NewServer(srv.Handler())
			gatewayURL := hs.URL

			conn, err := internal.DialWS(context.Background(), gatewayURL, internal.WSOptions{
				HTTPClient: hs.Client(),
				Reconnect:  false,
			})
			if err != nil {
				b.Fatalf("DialWS: %v", err)
			}

			sink := &wsSink{}
			conids := []int{265598, 8314, 756733, 12345, 67890, 11111, 22222, 33333, 44444, 55555}
			handle, err := conn.Subscribe(context.Background(), sink, sink, conids, []string{"31"})
			if err != nil {
				b.Fatalf("Subscribe: %v", err)
			}
			handle.Close()
			conn.Close()
			hs.Close()
		}
	})

	b.Run("SubscribeReceiveTick", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			script := mockgateway.StreamScript{
				OnSubscribe: func(conids []int, fields []string) []mockgateway.Tick {
					ticks := make([]mockgateway.Tick, 0, len(conids))
					for _, c := range conids {
						ticks = append(ticks, mockgateway.Tick{ConID: c, Field: "31", Value: "150.25"})
					}
					return ticks
				},
			}
			srv := mockgateway.New(mockgateway.WithStreamScript(&script))
			hs := httptest.NewServer(srv.Handler())
			gatewayURL := hs.URL

			conn, err := internal.DialWS(context.Background(), gatewayURL, internal.WSOptions{
				HTTPClient: hs.Client(),
				Reconnect:  false,
			})
			if err != nil {
				b.Fatalf("DialWS: %v", err)
			}

			sink := &wsSink{}
			conids := []int{265598}
			handle, err := conn.Subscribe(context.Background(), sink, sink, conids, []string{"31"})
			if err != nil {
				b.Fatalf("Subscribe: %v", err)
			}

			srv.PushTick(mockgateway.Tick{ConID: 265598, Field: "31", Value: "150.25"})

			handle.Close()
			conn.Close()
			hs.Close()
		}
	})
}

func BenchmarkWSMessageDecode(b *testing.B) {
	frame := map[string]any{"conid": 265598, "31": "150.25", "_updated": 1564652478}
	data, _ := json.Marshal(frame)

	b.Run("Dispatch", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(data)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var m map[string]json.RawMessage
			json.Unmarshal(data, &m)
		}
	})
}
