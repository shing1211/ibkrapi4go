// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func orderMetricsGateway(t *testing.T) (*httptest.Server, *atomic.Bool) {
	t.Helper()
	var reject atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if reject.Load() {
			fmt.Fprint(w, `[{"error":"order rejected"}]`)
			return
		}
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/orders"):
			fmt.Fprint(w, `[{"order_id":"999","order_status":"PreSubmitted"}]`)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/api/iserver/reply/"):
			fmt.Fprint(w, `[{"order_id":"999","order_status":"PreSubmitted"}]`)
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/order/999"):
			fmt.Fprint(w, `[{"order_id":"999","order_status":"PreSubmitted"}]`)
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/order/999"):
			fmt.Fprint(w, `{"order_id":"999","msg":"Request was submitted"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{}`)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &reject
}

func TestTrade_MetricsRecorded(t *testing.T) {
	srv, reject := orderMetricsGateway(t)
	m := NewInMemoryMetrics()
	cli, err := NewClient(WithGatewayURL(srv.URL), WithMetrics(m))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()
	ctx := context.Background()

	if _, err := cli.Trade().Submit(ctx, "U1234567", OrderRequest{
		ConID: 265598, Side: SideBuy, Quantity: "10", OrderType: OrderTypeMarket,
	}); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if _, err := cli.Trade().Confirm(ctx, "reply-1", true); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if _, err := cli.Trade().Modify(ctx, "U1234567", "999", OrderRequest{
		ConID: 265598, Side: SideBuy, Quantity: "11", OrderType: OrderTypeLimit, LimitPrice: "151.00",
	}); err != nil {
		t.Fatalf("Modify: %v", err)
	}
	if err := cli.Trade().Cancel(ctx, "U1234567", "999"); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	snap := m.Snapshot()
	for _, name := range []string{
		MetricOrdersSubmitted,
		MetricOrdersConfirmed,
		MetricOrdersModified,
		MetricOrdersCancelled,
	} {
		if got := snap.Counters[SeriesKey(name)]; got != 1 {
			t.Errorf("%s = %d; want 1", name, got)
		}
	}

	reject.Store(true)
	if _, err := cli.Trade().Submit(ctx, "U1234567", OrderRequest{
		ConID: 265598, Side: SideBuy, Quantity: "10", OrderType: OrderTypeMarket,
	}); !errors.Is(err, ErrOrderRejected) {
		t.Fatalf("Submit rejected: err = %v; want ErrOrderRejected", err)
	}
	if got := m.Snapshot().Counters[SeriesKey(MetricOrdersRejected)]; got != 1 {
		t.Errorf("rejected counter = %d; want 1", got)
	}
}

func TestTrade_NoMetricsDoesNotPanic(t *testing.T) {
	srv, _ := orderMetricsGateway(t)
	cli, err := NewClient(WithGatewayURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	if _, err := cli.Trade().Submit(context.Background(), "U1234567", OrderRequest{
		ConID: 265598, Side: SideBuy, Quantity: "10", OrderType: OrderTypeMarket,
	}); err != nil {
		t.Fatalf("Submit: %v", err)
	}
}
