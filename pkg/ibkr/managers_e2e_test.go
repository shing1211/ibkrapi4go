// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"errors"
	"testing"
	"time"
)

func newTestClient(t *testing.T, gw *gateway) *Client {
	t.Helper()
	cli, err := NewClient(WithGatewayURL(gw.URL), WithTickleInterval(time.Hour))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })
	return cli
}

func TestContracts(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)
	ctx := context.Background()

	results, err := cli.Trade().SearchContracts(ctx, "AAPL")
	if err != nil {
		t.Fatalf("SearchContracts: %v", err)
	}
	if len(results) != 1 || results[0].ConID != 265598 || results[0].Symbol != "AAPL" {
		t.Errorf("results = %+v; want conid 265598 AAPL", results)
	}

	info, err := cli.Trade().ContractInfo(ctx, 265598)
	if err != nil {
		t.Fatalf("ContractInfo: %v", err)
	}
	if info.Symbol != "AAPL" || info.Currency != "USD" || info.Multiplier != "1" {
		t.Errorf("info = %+v; want AAPL/USD multiplier 1", info)
	}

	rules, err := cli.Trade().ContractRules(ctx, 265598)
	if err != nil {
		t.Fatalf("ContractRules: %v", err)
	}
	if !rules.AlgoEligible || len(rules.OrderTypes) != 2 || rules.TimeInForce != "DAY" {
		t.Errorf("rules = %+v; want algoEligible with 2 order types TIF=DAY", rules)
	}

	strikes, err := cli.Trade().Strikes(ctx, 265598, "OPT", "JAN24")
	if err != nil {
		t.Fatalf("Strikes: %v", err)
	}
	if len(strikes.Call) != 2 || strikes.Call[0] != "150" || len(strikes.Put) != 2 || strikes.Put[1] != "140" {
		t.Errorf("strikes = %+v; want call[150,155] put[145,140]", strikes)
	}
}

func TestPortfolio(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)
	ctx := context.Background()

	accounts, err := cli.Portfolio().Accounts(ctx)
	if err != nil {
		t.Fatalf("Accounts: %v", err)
	}
	if len(accounts) != 2 || accounts[0].AccountID != "U1234567" || accounts[0].AccountAlias != "Main" {
		t.Errorf("accounts = %+v; want 2 with U1234567 alias Main", accounts)
	}

	positions, err := cli.Portfolio().Positions(ctx, "U1234567")
	if err != nil {
		t.Fatalf("Positions: %v", err)
	}
	if len(positions) != 1 || positions[0].Quantity != "10.5" || positions[0].UnrealizedPnL != "49.875" {
		t.Errorf("positions = %+v; want qty 10.5 upl 49.875", positions)
	}
	if positions[0].ConID != 265598 {
		t.Errorf("conid = %v; want 265598", positions[0].ConID)
	}

	// Paginated: page 0 has 2, page 1 has 1, page 2 is empty.
	var got []Position
	it := cli.Portfolio().PositionsPaginated(ctx, "U1234567")
	for it.Next(ctx) {
		got = append(got, it.Value())
	}
	if err := it.Err(); err != nil {
		t.Fatalf("PositionsPaginated: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("paginated positions = %d; want 3", len(got))
	}

	pos, err := cli.Portfolio().Position(ctx, "U1234567", 265598)
	if err != nil {
		t.Fatalf("Position: %v", err)
	}
	if pos.MktValue != "1575.00" {
		t.Errorf("mktValue = %q; want 1575.00", pos.MktValue)
	}

	ledger, err := cli.Portfolio().Ledger(ctx, "U1234567")
	if err != nil {
		t.Fatalf("Ledger: %v", err)
	}
	if ledger["USD"].CashBalance != "100.25" || ledger["USD"].NetLiquidationValue != "1575.00" {
		t.Errorf("ledger = %+v; want cash 100.25 nav 1575.00", ledger["USD"])
	}

	alloc, err := cli.Portfolio().Allocation(ctx, "U1234567")
	if err != nil {
		t.Fatalf("Allocation: %v", err)
	}
	if alloc["assetClass"].Long["STK"] != "1000.5" {
		t.Errorf("allocation = %+v; want STK long 1000.5", alloc["assetClass"].Long)
	}

	summary, err := cli.Portfolio().Summary(ctx, "U1234567")
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if summary["netliquidation"].Amount != "1575.00" {
		t.Errorf("summary netliquidation = %q; want 1575", summary["netliquidation"].Amount)
	}

	meta, err := cli.Portfolio().Meta(ctx, "U1234567")
	if err != nil {
		t.Fatalf("Meta: %v", err)
	}
	if meta.AccountTitle != "Main" {
		t.Errorf("meta = %+v; want title Main", meta)
	}

	if err := cli.Portfolio().Invalidate(ctx, "U1234567"); err != nil {
		t.Fatalf("Invalidate: %v", err)
	}
}

func TestMarketData(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)
	ctx := context.Background()

	snaps, err := cli.MarketData().Snapshot(ctx, []ConID{265598}, []Field{FieldLastPrice, FieldBidPrice})
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if len(snaps) != 1 || snaps[0].ConID != 265598 {
		t.Fatalf("snapshots = %+v; want one 265598", snaps)
	}
	if snaps[0].Fields[FieldLastPrice] != "150.25" {
		t.Errorf("last = %q; want 150.25", snaps[0].Fields[FieldLastPrice])
	}

	hist, err := cli.MarketData().History(ctx, HistoryOptions{ConID: 265598, Period: "1d", Bar: "1min"})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(hist.Bars) != 1 || hist.Bars[0].Close != "150.9" || hist.Bars[0].Volume != "1000" {
		t.Errorf("history = %+v; want 1 bar close 150.9", hist)
	}

	if err := cli.MarketData().Unsubscribe(ctx, 265598); err != nil {
		t.Fatalf("Unsubscribe: %v", err)
	}
	if err := cli.MarketData().UnsubscribeAll(ctx); err != nil {
		t.Fatalf("UnsubscribeAll: %v", err)
	}
}

func TestOrders_QueryAndPreview(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)
	ctx := context.Background()

	orders, err := cli.Trade().OpenOrders(ctx)
	if err != nil {
		t.Fatalf("OpenOrders: %v", err)
	}
	if len(orders) != 1 || orders[0].OrderID != "999" || orders[0].Price != "150.00" {
		t.Errorf("orders = %+v; want order 999 price 150.00", orders)
	}

	st, err := cli.Trade().OrderStatus(ctx, "999")
	if err != nil {
		t.Fatalf("OrderStatus: %v", err)
	}
	if st.Status != "PreSubmitted" || st.RemainingQuantity != "10" {
		t.Errorf("status = %+v; want PreSubmitted remaining 10", st)
	}

	trades, err := cli.Trade().Trades(ctx, 7)
	if err != nil {
		t.Fatalf("Trades: %v", err)
	}
	if len(trades) != 1 || trades[0].Symbol != "AAPL" || trades[0].NetAmount != "1499.00" {
		t.Errorf("trades = %+v; want AAPL net 1499.00", trades)
	}

	preview, err := cli.Trade().WhatIf(ctx, "U1234567", OrderRequest{
		ConID: 265598, Side: SideBuy, Quantity: "10", OrderType: OrderTypeLimit, LimitPrice: "150.00",
	})
	if err != nil {
		t.Fatalf("WhatIf: %v", err)
	}
	if preview.Amount["initial"] != "1000.50" {
		t.Errorf("whatif initial = %q; want 1000.50", preview.Amount["initial"])
	}
}

func TestOrders_SubmitReplyThenConfirm(t *testing.T) {
	gw := newGateway(t)
	gw.submitReply.Store(true)
	cli := newTestClient(t, gw)
	ctx := context.Background()

	res, err := cli.Trade().Submit(ctx, "U1234567", OrderRequest{
		ConID: 265598, Side: SideBuy, Quantity: "10", OrderType: OrderTypeLimit, LimitPrice: "150.00",
	})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if res.Accepted() || len(res.Replies) != 1 || res.Replies[0].ID != "reply-1" {
		t.Fatalf("result = %+v; want one pending reply", res)
	}

	confirmed, err := cli.Trade().Confirm(ctx, res.Replies[0].ID, true)
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if !confirmed.Accepted() || confirmed.OrderID != "999" {
		t.Errorf("confirmed = %+v; want accepted order 999", confirmed)
	}
}

func TestOrders_SubmitSuccess(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)
	ctx := context.Background()

	res, err := cli.Trade().Submit(ctx, "U1234567", OrderRequest{
		ConID: 265598, Side: SideBuy, Quantity: "10", OrderType: OrderTypeMarket,
	})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if !res.Accepted() || res.OrderID != "999" || res.Status != "PreSubmitted" {
		t.Errorf("result = %+v; want accepted 999 PreSubmitted", res)
	}
}

func TestOrders_MutationsAreSingleAttempt(t *testing.T) {
	gw := newGateway(t)
	gw.failSubmit.Store(true)
	cli := newTestClient(t, gw)
	ctx := context.Background()

	_, err := cli.Trade().Submit(ctx, "U1234567", OrderRequest{
		ConID: 265598, Side: SideBuy, Quantity: "10", OrderType: OrderTypeMarket,
	})
	if err == nil {
		t.Fatal("Submit: want error on 500")
	}
	if gw.submitCalls.Load() != 1 {
		t.Errorf("submit attempts = %d; want exactly 1 (no auto-retry)", gw.submitCalls.Load())
	}

	gw.failOrder.Store(true)
	if _, err := cli.Trade().Modify(ctx, "U1234567", "999", OrderRequest{
		ConID: 265598, Side: SideBuy, Quantity: "11", OrderType: OrderTypeLimit, LimitPrice: "151.00",
	}); err == nil {
		t.Fatal("Modify: want error on 500")
	}
	if err := cli.Trade().Cancel(ctx, "U1234567", "999"); err == nil {
		t.Fatal("Cancel: want error on 500")
	}
	if gw.orderCalls.Load() != 2 {
		t.Errorf("order mutation attempts = %d; want exactly 2 (modify+cancel, no retry)", gw.orderCalls.Load())
	}
}

func TestManagersReturnErrClosed(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)
	ctx := context.Background()
	if err := cli.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := cli.Portfolio().Accounts(ctx); !errors.Is(err, ErrClosed) {
		t.Errorf("Portfolio.Accounts = %v; want ErrClosed", err)
	}
	if _, err := cli.Trade().OpenOrders(ctx); !errors.Is(err, ErrClosed) {
		t.Errorf("Trade.OpenOrders = %v; want ErrClosed", err)
	}
	if _, err := cli.MarketData().Snapshot(ctx, []ConID{1}, nil); !errors.Is(err, ErrClosed) {
		t.Errorf("MarketData.Snapshot = %v; want ErrClosed", err)
	}
}
