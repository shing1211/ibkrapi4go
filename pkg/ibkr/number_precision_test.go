// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"reflect"
	"testing"

	"github.com/shing1211/ibkrapi4go/internal/mockgateway"
)

// This closes the last of the json.Number class. money_precision_test.go covers
// the monetary fields on the account summary, positions and ledger; these are the
// remaining ones - historical bar prices, option strike prices, and a contract
// multiplier.
//
// Same argument applies. Each of these decodes through json.Number and is handed
// to the caller as a string, so a float32 regression would round silently rather
// than fail to compile. The values below change under a float32, and
// float32Loses - shared with the money tests - enforces that so this file cannot
// quietly stop proving anything.

// TestMarketDataHistory_BarPricePrecision covers the OHLCV decode, where a single
// anonymous struct carries five json.Number fields into a Bar. Open, high, low
// and close are prices; volume is a count, and is included because it travels the
// same path and would round the same way.
func TestMarketDataHistory_BarPricePrecision(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	const (
		open   = "16777217.89"
		high   = "12345678.91"
		low    = "33554432.55"
		close  = "67108865.13"
		volume = "12345.6789"
	)
	for _, lit := range []string{open, high, low, close, volume} {
		float32Loses(t, lit)
	}

	gw.srv.Fixtures().Set(mockgateway.OpGetMdHistory, mockgateway.Fixture{
		Body: `{"symbol":"AAPL","data":[{"t":1700000000,` +
			`"o":` + open + `,` +
			`"h":` + high + `,` +
			`"l":` + low + `,` +
			`"c":` + close + `,` +
			`"v":` + volume + `}]}`,
	})

	h, err := cli.MarketData().History(context.Background(), HistoryOptions{
		ConID:  ConID(265598),
		Period: "1d",
		Bar:    "1d",
	})
	if err != nil {
		t.Fatalf("MarketData.History: %v", err)
	}
	if len(h.Bars) != 1 {
		t.Fatalf("bars = %+v; want one", h.Bars)
	}

	bar := h.Bars[0]
	for _, tc := range []struct{ name, got, want string }{
		{"open", bar.Open, open},
		{"high", bar.High, high},
		{"low", bar.Low, low},
		{"close", bar.Close, close},
		{"volume", bar.Volume, volume},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q; want %q - the gateway's digits must survive verbatim",
				tc.name, tc.got, tc.want)
		}
	}
}

// TestTradeStrikes_Precision covers the []json.Number slice. A slice is a distinct
// decode target from a scalar: an element index that regressed would leave the
// surrounding structure intact, so the assertions check both the values and the
// length.
func TestTradeStrikes_Precision(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	const (
		call1 = "16777217.89"
		call2 = "12345678.91"
		put1  = "33554432.55"
		put2  = "12345.6789"
	)
	for _, lit := range []string{call1, call2, put1, put2} {
		float32Loses(t, lit)
	}

	gw.srv.Fixtures().Set(mockgateway.OpGetContractStrikes, mockgateway.Fixture{
		Body: `{"call":[` + call1 + `,` + call2 + `],"put":[` + put1 + `,` + put2 + `]}`,
	})

	strikes, err := cli.Trade().Strikes(context.Background(), ConID(265598), "OPT", "20260918")
	if err != nil {
		t.Fatalf("Trade.Strikes: %v", err)
	}

	if want := []string{call1, call2}; !reflect.DeepEqual(strikes.Call, want) {
		t.Errorf("call strikes = %#v; want %#v", strikes.Call, want)
	}
	if want := []string{put1, put2}; !reflect.DeepEqual(strikes.Put, want) {
		t.Errorf("put strikes = %#v; want %#v", strikes.Put, want)
	}
}

// TestContractInfo_MultiplierPrecision covers the contract multiplier, the last
// json.Number that reaches a caller as a string. Strikes and bar prices are prices
// a user acts on; a multiplier is a scale factor, so a rounding error there would
// quietly misstate the size of a contract rather than its price.
func TestContractInfo_MultiplierPrecision(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	const mult = "12345.6789"
	float32Loses(t, mult)

	// Trade.ContractInfo calls the getInstrumentInfo endpoint, not getContractInfo;
	// the SDK's op label is "Trade.ContractInfo" while the route is a different
	// operation, so the fixture has to be set on the route the generated client
	// actually calls.
	gw.srv.Fixtures().Set(mockgateway.OpGetInstrumentInfo, mockgateway.Fixture{
		Body: `{"con_id":265598,"symbol":"AAPL","instrument_type":"STK",` +
			`"multiplier":` + mult + `}`,
	})

	info, err := cli.Trade().ContractInfo(context.Background(), ConID(265598))
	if err != nil {
		t.Fatalf("Trade.ContractInfo: %v", err)
	}
	if info.Multiplier != mult {
		t.Errorf("multiplier = %q; want %q - the gateway's digits must survive verbatim",
			info.Multiplier, mult)
	}
}
