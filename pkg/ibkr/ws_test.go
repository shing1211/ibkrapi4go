// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shing1211/ibkrapi4go/internal/mockgateway"
)

// newWSServer starts the mockgateway WebSocket endpoint with the given script
// and returns its test server. A nil script uses the default behavior: one tick
// per subscribed conid.
func newWSServer(t *testing.T, script *mockgateway.StreamScript) *httptest.Server {
	t.Helper()
	srv := mockgateway.New(mockgateway.WithStreamScript(script))
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func recvUpdate(sub *Subscription, d time.Duration) (Update, bool) {
	select {
	case u, ok := <-sub.Updates():
		return u, ok
	case <-time.After(d):
		return Update{}, false
	}
}

func TestWS_SubscribeReceiveClose(t *testing.T) {
	srv := newWSServer(t, nil)
	cli, err := NewClient(WithGatewayURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	ctx := context.Background()
	sub, err := cli.MarketData().Subscribe(ctx, []ConID{265598}, []Field{FieldLastPrice})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	u, ok := recvUpdate(sub, 2*time.Second)
	if !ok {
		t.Fatal("no update received")
	}
	if u.ConID != 265598 || u.Field != FieldLastPrice || u.Value != "150.25" {
		t.Errorf("update = %+v; want 265598/31/150.25", u)
	}

	if err := sub.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, ok := <-sub.Updates(); ok {
		t.Error("Updates channel not closed after Close")
	}
}

func TestWS_ReconnectResubscribes(t *testing.T) {
	srv := newWSServer(t, &mockgateway.StreamScript{DropFirstConnection: true})
	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithStreamingLimits(StreamingLimits{ReconnectBase: 10 * time.Millisecond, ReconnectMax: 50 * time.Millisecond}),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	sub, err := cli.MarketData().Subscribe(context.Background(), []ConID{265598}, nil)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	// After the drop and reconnect, the second connection re-sends subscribe and
	// receives an update; a reconnect notice arrives on Errors.
	gotUpdate := false
	gotReconnect := false
	deadline := time.After(5 * time.Second)
	for !gotUpdate || !gotReconnect {
		select {
		case u := <-sub.Updates():
			if u.Value == "150.25" {
				gotUpdate = true
			}
		case err := <-sub.Errors():
			if errors.Is(err, ErrWSReconnected) {
				gotReconnect = true
			}
		case <-deadline:
			t.Fatalf("timeout: gotUpdate=%v gotReconnect=%v", gotUpdate, gotReconnect)
		}
	}
}

func TestWS_MetricsConnectsAndReconnects(t *testing.T) {
	m := NewInMemoryMetrics()
	srv := newWSServer(t, &mockgateway.StreamScript{DropFirstConnection: true})
	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithMetrics(m),
		WithStreamingLimits(StreamingLimits{ReconnectBase: 10 * time.Millisecond, ReconnectMax: 50 * time.Millisecond}),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	sub, err := cli.MarketData().Subscribe(context.Background(), []ConID{265598}, nil)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	// Wait until the drop has been observed and the reconnect fully completed
	// (an update arrives on the new connection), then check the counters.
	gotUpdate := false
	gotReconnect := false
	deadline := time.After(5 * time.Second)
	for !gotUpdate || !gotReconnect {
		select {
		case u := <-sub.Updates():
			if u.Value == "150.25" {
				gotUpdate = true
			}
		case err := <-sub.Errors():
			if errors.Is(err, ErrWSReconnected) {
				gotReconnect = true
			}
		case <-deadline:
			t.Fatalf("timeout: gotUpdate=%v gotReconnect=%v", gotUpdate, gotReconnect)
		}
	}

	snap := m.Snapshot()
	if got := snap.Counters[SeriesKey(MetricWSConnects)]; got != 1 {
		t.Errorf("ws connects = %d; want 1", got)
	}
	if got := snap.Counters[SeriesKey(MetricWSReconnects)]; got < 1 {
		t.Errorf("ws reconnects = %d; want >= 1", got)
	}
}

func TestWS_BufferOverflowDropsOldest(t *testing.T) {
	const burst = 64
	srv := newWSServer(t, &mockgateway.StreamScript{
		OnSubscribe: func(conids []int, _ []string) []mockgateway.Tick {
			ticks := make([]mockgateway.Tick, 0, burst)
			for i := 0; i < burst; i++ {
				ticks = append(ticks, mockgateway.Tick{ConID: conids[0], Field: "31", Value: "150.25"})
			}
			return ticks
		},
	})
	cli, err := NewClient(WithGatewayURL(srv.URL), WithStreamingLimits(StreamingLimits{BufferSize: 4}))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	sub, err := cli.MarketData().Subscribe(context.Background(), []ConID{265598}, nil)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	// Wait until the server has flooded a small buffer faster than we read.
	deadline := time.After(5 * time.Second)
	for sub.Dropped() == 0 {
		select {
		case <-deadline:
			t.Fatalf("expected drops with a %d-buffer and a %d burst", 4, burst)
		case <-time.After(10 * time.Millisecond):
		}
	}
	if sub.Dropped() == 0 {
		t.Error("Dropped() = 0; want > 0")
	}
}

func TestWS_Limits(t *testing.T) {
	srv := newWSServer(t, nil)
	cli, err := NewClient(WithGatewayURL(srv.URL), WithStreamingLimits(StreamingLimits{
		MaxConIDsPerRequest: 2,
		MaxFieldsPerRequest: 2,
		MaxSubscriptions:    1,
	}))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()
	ctx := context.Background()

	if _, err := cli.MarketData().Subscribe(ctx, []ConID{1, 2, 3}, nil); !errors.Is(err, ErrStreamingLimit) {
		t.Errorf("conids over limit err = %v; want ErrStreamingLimit", err)
	}
	if _, err := cli.MarketData().Subscribe(ctx, []ConID{1}, []Field{"31", "84", "86"}); !errors.Is(err, ErrStreamingLimit) {
		t.Errorf("fields over limit err = %v; want ErrStreamingLimit", err)
	}
	sub, err := cli.MarketData().Subscribe(ctx, []ConID{1}, nil)
	if err != nil {
		t.Fatalf("first Subscribe: %v", err)
	}
	defer sub.Close()
	if _, err := cli.MarketData().Subscribe(ctx, []ConID{2}, nil); !errors.Is(err, ErrStreamingLimit) {
		t.Errorf("subscriptions over limit err = %v; want ErrStreamingLimit", err)
	}
}

func TestWS_SystemUpdates(t *testing.T) {
	srv := newWSServer(t, &mockgateway.StreamScript{Hello: true})
	cli, err := NewClient(WithGatewayURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	sub, err := cli.MarketData().Subscribe(context.Background(), []ConID{265598}, []Field{FieldLastPrice})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	gotStatus := false
	gotNotification := false
	deadline := time.After(5 * time.Second)
	for !gotStatus || !gotNotification {
		select {
		case u, ok := <-sub.Updates():
			if !ok {
				t.Fatal("Updates channel closed unexpectedly")
			}
			if u.ConID == 265598 && u.Field == FieldLastPrice {
			}
		case su, ok := <-sub.SystemUpdates():
			if !ok {
				t.Fatal("SystemUpdates channel closed unexpectedly")
			}
			switch su.Type {
			case SystemUpdateStatus:
				if su.Status != "connected" {
					t.Errorf("Status = %q; want \"connected\"", su.Status)
				}
				gotStatus = true
			case SystemUpdateNotification:
				gotNotification = true
			}
		case <-deadline:
			t.Fatalf("timeout: gotStatus=%v gotNotification=%v", gotStatus, gotNotification)
		}
	}

	if err := sub.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, ok := <-sub.SystemUpdates(); ok {
		t.Error("SystemUpdates channel not closed after Close")
	}
}
