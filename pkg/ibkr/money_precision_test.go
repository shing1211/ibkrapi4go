// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"strconv"
	"testing"

	"github.com/shing1211/ibkrapi4go/internal/mockgateway"
)

// Account summaries, positions and the ledger declare their monetary fields as
// json.Number specifically so decimal digits survive the decode, and then hand
// them to callers as strings. 39 such fields exist across these three responses.
//
// Every other assertion on them uses a value a float32 represents exactly -
// 1234.5600, 100.25, 49.875, 40.0 - so those tests pass whether the field is a
// float or a json.Number, and prove nothing about precision. That is the same
// shape of blind spot that hid real money bugs twice in this repository
// (v1.1.9 on the voucher response, v1.1.13 on the banking request), and it is why
// the tax-voucher path has its own precision test and these three do not.
//
// A float32 mantissa is 24 bits, so it rounds anything above 2^24 (16777216) to a
// whole number of units. The values below are chosen to be destroyed by that
// path, and each test asserts what the float32 path would have produced so the
// expectation cannot quietly stop proving anything.

// float32Loses reports the failure this file exists to prevent, expressed once.
//
// The check is deliberately about detectability rather than about exact
// representability. Some values are lossy in binary but still render as the
// original literal under strconv's shortest-round-trip formatting - 0.007 is one:
// the nearest float32 formats back as "0.007", so no string comparison could ever
// notice it had been through a float. A test asserting such a value looks
// rigorous and proves nothing, so each value used below is required to change
// under the float32 path.
func float32Loses(t *testing.T, literal string) {
	t.Helper()
	v, err := strconv.ParseFloat(literal, 64)
	if err != nil {
		t.Fatalf("parse %q: %v", literal, err)
	}
	got := strconv.FormatFloat(float64(float32(v)), 'f', -1, 32)
	if got == literal {
		t.Errorf("float32 renders %s back as %q, so no string assertion could detect a "+
			"float32 regression; this value does not prove the precision fix", literal, got)
	}
}

func TestAccountSummary_MoneyPrecision(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	// Every monetary field below exceeds 2^24, so a float32 decode would return
	// the rounded whole number rather than these digits.
	const (
		nlv = "16777217.89" // just above 2^24
		sma = "12345678.91"
		eq  = "33554432.55" // 2^25 + 32.55
		fee = "12345.6789"  // fractional part below float32 resolution here
	)
	for _, lit := range []string{nlv, sma, eq, fee} {
		float32Loses(t, lit)
	}

	gw.srv.Fixtures().Set(mockgateway.OpGetAccountSummary, mockgateway.Fixture{
		Body: `{"accountType":"INDIVIDUAL",` +
			`"netLiquidationValue":` + nlv + `,` +
			`"totalCashValue":` + sma + `,` +
			`"availableFunds":` + eq + `,` +
			`"SMA":` + sma + `,` +
			`"buyingPower":` + eq + `,` +
			`"balance":` + nlv + `,` +
			`"equityWithLoanValue":` + nlv + `,` +
			`"excessLiquidity":` + sma + `,` +
			`"initialMargin":` + eq + `,` +
			`"maintenanceMargin":` + sma + `,` +
			`"regTLoan":` + eq + `,` +
			`"regTMargin":` + sma + `,` +
			`"securitiesGVP":` + eq + `,` +
			`"accruedInterest":` + fee + `,` +
			`"cashBalances":[{"currency":"USD","balance":` + eq + `,"settledCash":` + nlv + `}]}`,
	})

	sum, err := cli.Account().Summary(context.Background(), AccountID("U1234567"))
	if err != nil {
		t.Fatalf("Account.Summary: %v", err)
	}

	// Every monetary json.Number field on accountSummaryRaw appears here. The set
	// is deliberately exhaustive: a raw money field added without a lossy-value
	// assertion is the exact shape of the gap this file exists to close, and
	// leaving one out would reopen it silently.
	for _, tc := range []struct{ name, got, want string }{
		{"netLiquidationValue", sum.NetLiquidationValue, nlv},
		{"totalCashValue", sum.TotalCashValue, sma},
		{"availableFunds", sum.AvailableFunds, eq},
		{"SMA", sum.SMA, sma},
		{"buyingPower", sum.BuyingPower, eq},
		{"balance", sum.Balance, nlv},
		{"equityWithLoanValue", sum.EquityWithLoanValue, nlv},
		{"excessLiquidity", sum.ExcessLiquidity, sma},
		{"initialMargin", sum.InitialMargin, eq},
		{"maintenanceMargin", sum.MaintenanceMargin, sma},
		{"regTLoan", sum.RegTLoan, eq},
		{"regTMargin", sum.RegTMargin, sma},
		{"securitiesGVP", sum.SecuritiesGVP, eq},
		{"accruedInterest", sum.AccruedInterest, fee},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q; want %q - the gateway's digits must survive verbatim",
				tc.name, tc.got, tc.want)
		}
	}

	// The nested cash balance is a separate decode target: a slice element that
	// regressed would leave the top-level fields correct.
	if len(sum.CashBalances) != 1 {
		t.Fatalf("cashBalances = %+v; want one", sum.CashBalances)
	}
	if got := sum.CashBalances[0].Balance; got != eq {
		t.Errorf("cashBalances[0].balance = %q; want %q", got, eq)
	}
	if got := sum.CashBalances[0].SettledCash; got != nlv {
		t.Errorf("cashBalances[0].settledCash = %q; want %q", got, nlv)
	}
}

func TestPortfolioPositions_MoneyPrecision(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	const (
		avgCost = "16777217.89"
		mktPrc  = "12345678.91"
		mktVal  = "33554432.55"
		realPnL = "12345.6789"
		unreal  = "67108865.13" // 2^26 + 65.13
	)
	for _, lit := range []string{avgCost, mktPrc, mktVal, realPnL, unreal} {
		float32Loses(t, lit)
	}

	gw.srv.Fixtures().Set(mockgateway.OpGetUncachedPositions, mockgateway.Fixture{
		Body: `[{"assetCategory":"STK","currency":"USD","fxRisk":0,` +
			`"position":10,"avgCost":` + avgCost + `,` +
			`"avgPrice":` + mktPrc + `,` +
			`"mktPrice":` + mktPrc + `,` +
			`"mktValue":` + mktVal + `,` +
			`"realizedPnl":` + realPnL + `,` +
			`"unrealizedPnl":` + unreal + `,` +
			`"openPrice":` + mktPrc + `,` +
			`"conid":265598,"contractDesc":"AAPL"}]`,
	})

	positions, err := cli.Portfolio().Positions(context.Background(), AccountID("U1234567"))
	if err != nil {
		t.Fatalf("Portfolio.Positions: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("positions = %+v; want one", positions)
	}

	for _, tc := range []struct{ name, got, want string }{
		{"avgCost", positions[0].AvgCost, avgCost},
		{"avgPrice", positions[0].AvgPrice, mktPrc},
		{"mktPrice", positions[0].MktPrice, mktPrc},
		{"mktValue", positions[0].MktValue, mktVal},
		{"realizedPnl", positions[0].RealizedPnL, realPnL},
		{"unrealizedPnl", positions[0].UnrealizedPnL, unreal},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q; want %q - the gateway's digits must survive verbatim",
				tc.name, tc.got, tc.want)
		}
	}
}

func TestPortfolioLedger_MoneyPrecision(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	const (
		cash    = "16777217.89"
		settl   = "12345678.91"
		nlv     = "33554432.55"
		smv     = "67108865.13"
		optSVal = "12345.6789"
		upl     = "12345.6789"
	)
	for _, lit := range []string{cash, settl, nlv, smv, optSVal, upl} {
		float32Loses(t, lit)
	}

	// The ledger keys currencies, so the map itself is a decode target as well as
	// each value.
	gw.srv.Fixtures().Set(mockgateway.OpGetPortfolioLedger, mockgateway.Fixture{
		Body: `{"USD":{"acctcode":"U1234567","currency":"USD",` +
			`"cashbalance":` + cash + `,` +
			`"settledcash":` + settl + `,` +
			`"netliquidationvalue":` + nlv + `,` +
			`"stockmarketvalue":` + smv + `,` +
			`"stockoptionmarketvalue":` + optSVal + `,` +
			`"unrealizedpnl":` + upl + `,` +
			`"realizedpnl":` + cash + `}}`,
	})

	ledger, err := cli.Portfolio().Ledger(context.Background(), AccountID("U1234567"))
	if err != nil {
		t.Fatalf("Portfolio.Ledger: %v", err)
	}

	usd, ok := ledger["USD"]
	if !ok {
		t.Fatalf("ledger = %+v; want a USD entry", ledger)
	}
	// Every monetary json.Number field on ledgerRaw appears here.
	for _, tc := range []struct{ name, got, want string }{
		{"cashbalance", usd.CashBalance, cash},
		{"settledcash", usd.SettledCash, settl},
		{"netliquidationvalue", usd.NetLiquidationValue, nlv},
		{"stockmarketvalue", usd.StockMarketValue, smv},
		{"stockoptionmarketvalue", usd.StockOptionMarketValue, optSVal},
		{"unrealizedpnl", usd.UnrealizedPnL, upl},
		{"realizedpnl", usd.RealizedPnL, cash},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q; want %q - the gateway's digits must survive verbatim",
				tc.name, tc.got, tc.want)
		}
	}
}
