// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

type wsServerConfig struct {
	dropFirstConn bool
	onSubscribe   func(c *websocket.Conn, conids []int, fields []string)
}

func newWSServer(t *testing.T, cfg wsServerConfig) *httptest.Server {
	t.Helper()
	var conns atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		n := conns.Add(1)
		if cfg.dropFirstConn && n == 1 {
			// Read one subscribe frame, then drop the connection.
			_, data, err := c.Read(context.Background())
			if err == nil && cfg.onSubscribe != nil {
				method, conids, fields := parseWSFrame(data)
				if method == "subscribe" {
					cfg.onSubscribe(c, conids, fields)
				}
			}
			_ = c.Close(websocket.StatusInternalError, "drop")
			return
		}
		for {
			_, data, err := c.Read(context.Background())
			if err != nil {
				return
			}
			method, conids, fields := parseWSFrame(data)
			if method != "subscribe" || len(conids) == 0 {
				continue
			}
			if cfg.onSubscribe != nil {
				cfg.onSubscribe(c, conids, fields)
				continue
			}
			sendWSUpdate(c, conids[0], "31", "150.25")
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func parseWSFrame(data []byte) (method string, conids []int, fields []string) {
	var f struct {
		Method string `json:"method"`
		Params struct {
			Conids []int    `json:"conids"`
			Fields []string `json:"fields"`
		} `json:"params"`
	}
	_ = json.Unmarshal(data, &f)
	return f.Method, f.Params.Conids, f.Params.Fields
}

func sendWSUpdate(c *websocket.Conn, conid int, field, value string) {
	frame := map[string]any{"conid": conid, field: value, "_updated": time.Now().Unix()}
	b, _ := json.Marshal(frame)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = c.Write(ctx, websocket.MessageText, b)
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
	srv := newWSServer(t, wsServerConfig{})
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
	srv := newWSServer(t, wsServerConfig{dropFirstConn: true})
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
			if errors.Is(err, ErrStreamReconnected) {
				gotReconnect = true
			}
		case <-deadline:
			t.Fatalf("timeout: gotUpdate=%v gotReconnect=%v", gotUpdate, gotReconnect)
		}
	}
}

func TestWS_MetricsConnectsAndReconnects(t *testing.T) {
	m := NewInMemoryMetrics()
	srv := newWSServer(t, wsServerConfig{dropFirstConn: true})
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
			if errors.Is(err, ErrStreamReconnected) {
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
	srv := newWSServer(t, wsServerConfig{onSubscribe: func(c *websocket.Conn, conids []int, fields []string) {
		for i := 0; i < burst; i++ {
			sendWSUpdate(c, conids[0], "31", "150.25")
		}
	}})
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
	srv := newWSServer(t, wsServerConfig{})
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
