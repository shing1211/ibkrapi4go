// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestForecast exercises all five forecast wrappers against the mock gateway. The
// fixtures are the ones internal/mockgateway serves for these opIds, so the
// assertions are pinned to that payload rather than to values invented here.
func TestForecast(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)
	ctx := context.Background()

	t.Run("categories", func(t *testing.T) {
		raw, err := cli.Forecast().ForecastCategories(ctx)
		if err != nil {
			t.Fatalf("ForecastCategories: %v", err)
		}
		if !strings.Contains(string(raw), "Elections") {
			t.Errorf("categories = %s; want the Elections category", raw)
		}
	})

	t.Run("contract", func(t *testing.T) {
		got, err := cli.Forecast().ForecastContract(ctx, 123456)
		if err != nil {
			t.Fatalf("ForecastContract: %v", err)
		}
		// ConidYes/ConidNo are json.Number, so they are compared as text. That is
		// the point of the type: the digits IBKR sent are the digits we hand back,
		// with no float in between (ADR 0008).
		if got.ConidYes.String() != "123456" || got.ConidNo.String() != "123457" {
			t.Errorf("conids = yes %q no %q; want 123456 / 123457", got.ConidYes, got.ConidNo)
		}
		if got.Question != "US Presidential Election" || got.Symbol != "USPREZ" {
			t.Errorf("got question %q symbol %q; want the fixture values", got.Question, got.Symbol)
		}
		if got.Side != "Y" || got.Category != "Elections" {
			t.Errorf("got side %q category %q; want Y / Elections", got.Side, got.Category)
		}
	})

	t.Run("markets", func(t *testing.T) {
		got, err := cli.Forecast().ForecastMarkets(ctx, 265598, "")
		if err != nil {
			t.Fatalf("ForecastMarkets: %v", err)
		}
		if got.MarketName != "USPREZ" || got.Exchange != "MVO" || got.Symbol != "USPREZ" {
			t.Errorf("got market %q exchange %q symbol %q; want USPREZ / MVO / USPREZ",
				got.MarketName, got.Exchange, got.Symbol)
		}
		// contracts is a nested array: the point of this case is that it decodes
		// into the slice rather than being dropped by the key-by-key path.
		if len(got.Contracts) != 1 {
			t.Fatalf("contracts = %d; want 1", len(got.Contracts))
		}
		c := got.Contracts[0]
		if c.Conid.String() != "123456" || c.UnderlyingConid.String() != "265598" {
			t.Errorf("contract conids = %q / %q; want 123456 / 265598", c.Conid, c.UnderlyingConid)
		}
		if c.Side != "Y" || c.Expiration != "20261103" {
			t.Errorf("contract side %q expiration %q; want Y / 20261103", c.Side, c.Expiration)
		}
	})

	t.Run("rules", func(t *testing.T) {
		got, err := cli.Forecast().ForecastRules(ctx, 123456)
		if err != nil {
			t.Fatalf("ForecastRules: %v", err)
		}
		if got.AssetClass != "STK" || got.MarketName != "USPREZ" || got.ProductCode != "USPREZ" {
			t.Errorf("got assetClass %q market %q product %q; want STK / USPREZ / USPREZ",
				got.AssetClass, got.MarketName, got.ProductCode)
		}
		if got.SourceAgency != "IBKR" || got.ExchangeTimezone != "US/Eastern" {
			t.Errorf("got agency %q timezone %q; want IBKR / US/Eastern", got.SourceAgency, got.ExchangeTimezone)
		}
		if got.MeasuredPeriod != "1D" || got.PriceIncrement != "1" {
			t.Errorf("got period %q increment %q; want 1D / 1", got.MeasuredPeriod, got.PriceIncrement)
		}
	})

	t.Run("schedule", func(t *testing.T) {
		got, err := cli.Forecast().ForecastSchedule(ctx, 123456)
		if err != nil {
			t.Fatalf("ForecastSchedule: %v", err)
		}
		if got.Timezone != "US/Eastern" {
			t.Errorf("timezone = %q; want US/Eastern", got.Timezone)
		}
		// trading_schedules nests twice, so this also covers the inner slice.
		if len(got.TradingSchedules) != 1 {
			t.Fatalf("tradingSchedules = %d; want 1", len(got.TradingSchedules))
		}
		day := got.TradingSchedules[0]
		if day.DayOfWeek != "2026-11-03" {
			t.Errorf("dayOfWeek = %q; want 2026-11-03", day.DayOfWeek)
		}
		if len(day.TradingTimes) != 1 {
			t.Fatalf("tradingTimes = %d; want 1", len(day.TradingTimes))
		}
		tt := day.TradingTimes[0]
		if tt.Open != "2026-11-03T00:00:00Z" || tt.Close != "2026-11-03T23:59:59Z" {
			t.Errorf("trading time = %q..%q; want the fixture window", tt.Open, tt.Close)
		}
	})
}

// TestForecast_QueryParamsAreTransmitted covers the plumbing the mock gateway
// cannot: it routes on path alone, so a wrapper that forgot to pass conid would
// still get the fixture back and the bug would be invisible. Here the query string
// is recorded and asserted.
//
// An omitted optional must stay omitted rather than being sent empty, since
// exchange="" is a different request from exchange absent.
func TestForecast_QueryParamsAreTransmitted(t *testing.T) {
	type seen struct {
		path  string
		query string
	}
	var got []seen
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, seen{path: r.URL.Path, query: r.URL.RawQuery})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	cli, err := NewClient(WithGatewayURL(srv.URL), WithTickleInterval(time.Hour))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })
	ctx := context.Background()

	if _, err := cli.Forecast().ForecastContract(ctx, 123456); err != nil {
		t.Fatalf("ForecastContract: %v", err)
	}
	if _, err := cli.Forecast().ForecastRules(ctx, 123456); err != nil {
		t.Fatalf("ForecastRules: %v", err)
	}
	if _, err := cli.Forecast().ForecastSchedule(ctx, 123456); err != nil {
		t.Fatalf("ForecastSchedule: %v", err)
	}
	if _, err := cli.Forecast().ForecastMarkets(ctx, 265598, ""); err != nil {
		t.Fatalf("ForecastMarkets without exchange: %v", err)
	}
	if _, err := cli.Forecast().ForecastMarkets(ctx, 265598, "MVO"); err != nil {
		t.Fatalf("ForecastMarkets with exchange: %v", err)
	}

	if len(got) != 5 {
		t.Fatalf("recorded %d requests; want 5: %+v", len(got), got)
	}
	for i, want := range []struct {
		path  string
		query string
	}{
		{"/v1/api/forecast/contract/details", "conid=123456"},
		{"/v1/api/forecast/contract/rules", "conid=123456"},
		{"/v1/api/forecast/contract/schedules", "conid=123456"},
		{"/v1/api/forecast/contract/market", "underlyingConid=265598"},
		{"/v1/api/forecast/contract/market", "underlyingConid=265598&exchange=MVO"},
	} {
		if got[i].path != want.path {
			t.Errorf("request %d path = %q; want %q", i, got[i].path, want.path)
		}
		if got[i].query != want.query {
			t.Errorf("request %d query = %q; want %q", i, got[i].query, want.query)
		}
	}
}

// TestForecast_EmptyExchangeIsOmitted is the negative half of the case above,
// stated separately because it is the assertion that would catch an empty
// optional being sent as exchange=.
func TestForecast_EmptyExchangeIsOmitted(t *testing.T) {
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	cli, err := NewClient(WithGatewayURL(srv.URL), WithTickleInterval(time.Hour))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })

	if _, err := cli.Forecast().ForecastMarkets(context.Background(), 265598, ""); err != nil {
		t.Fatalf("ForecastMarkets: %v", err)
	}
	if strings.Contains(query, "exchange") {
		t.Errorf("query = %q; want no exchange key when the argument is empty", query)
	}
}

// TestForecast_NumericFieldsAreNotFloats asserts the ADR 0008 property directly:
// a number in the payload reaches the caller as the digits IBKR sent, rather than
// through a float64.
//
// The value is 2^53+1, not 0.1, and the choice is load-bearing. float64 cannot
// represent 2^53+1, so a round trip through one yields 9007199254740992 and the
// assertion below fails. A decimal like 0.1 does not: Go formats floats with the
// shortest representation that round-trips, so %v on float64(0.1) is "0.1" and a
// test built on it would pass against the exact bug it is meant to catch.
func TestForecast_NumericFieldsAreNotFloats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"strike":9007199254740993,"conid_yes":123456}`))
	}))
	defer srv.Close()

	cli, err := NewClient(WithGatewayURL(srv.URL), WithTickleInterval(time.Hour))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })

	got, err := cli.Forecast().ForecastContract(context.Background(), 123456)
	if err != nil {
		t.Fatalf("ForecastContract: %v", err)
	}
	if got.Strike.String() != "9007199254740993" {
		t.Errorf("strike = %q; want \"9007199254740993\" exactly - a float64 would read %s",
			got.Strike, "9007199254740992")
	}
	if got.ConidYes != json.Number("123456") {
		t.Errorf("conidYes = %q; want 123456", got.ConidYes)
	}
}
