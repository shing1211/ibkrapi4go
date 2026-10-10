// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"encoding/json"
	"testing"
)

func TestTradingAccount_AccountOwners(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)
	ctx := context.Background()

	raw, err := cli.TradingAccount().AccountOwners(ctx, AccountID("U1234567"))
	if err != nil {
		t.Fatalf("AccountOwners: %v", err)
	}
	var got struct {
		AccountID  string         `json:"accountId"`
		Owners     []AccountOwner `json:"owners"`
		Signatures []string       `json:"signatures"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode owners: %v", err)
	}
	if got.AccountID != "U1234567" {
		t.Errorf("accountId = %q; want U1234567", got.AccountID)
	}
	if len(got.Owners) != 1 {
		t.Fatalf("owners = %+v; want exactly one", got.Owners)
	}
	if got.Owners[0].Name != "Jane Doe" || got.Owners[0].OwnerType != "INDIVIDUAL" {
		t.Errorf("owners[0] = %+v; want {Jane Doe INDIVIDUAL}", got.Owners[0])
	}
	if len(got.Signatures) != 1 || got.Signatures[0] != "Jane Doe" {
		t.Errorf("signatures = %+v; want [Jane Doe]", got.Signatures)
	}
}

func TestTradingAccount_SetActiveAccount(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)
	ctx := context.Background()

	if err := cli.TradingAccount().SetActiveAccount(ctx, AccountID("U1234567")); err != nil {
		t.Fatalf("SetActiveAccount: %v", err)
	}
}

func TestTradingAccount_DynamicAccounts(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)
	ctx := context.Background()

	accounts, err := cli.TradingAccount().DynamicAccounts(ctx, "Main")
	if err != nil {
		t.Fatalf("DynamicAccounts: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("accounts = %+v; want exactly one", accounts)
	}
	if accounts[0].AccountID != AccountID("U1234567") {
		t.Errorf("accountId = %q; want U1234567", accounts[0].AccountID)
	}
	if accounts[0].AccountTitle != "Main Account" {
		t.Errorf("accountTitle = %q; want Main Account", accounts[0].AccountTitle)
	}
}

func TestTradingAccount_FundSummary(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)
	ctx := context.Background()

	got, err := cli.TradingAccount().FundSummary(ctx, AccountID("U1234567"))
	if err != nil {
		t.Fatalf("FundSummary: %v", err)
	}
	// Money is carried as a string to preserve precision (ADR 0008).
	if got.BuyingPower != "4000.00" {
		t.Errorf("buyingPower = %q; want 4000.00", got.BuyingPower)
	}
	if got.AvailableFunds != "900.10" {
		t.Errorf("availableFunds = %q; want 900.10", got.AvailableFunds)
	}
	if got.NetLiquidationValue != "1234.56" {
		t.Errorf("netLiquidationValue = %q; want 1234.56", got.NetLiquidationValue)
	}
	if got.AccountID != AccountID("U1234567") {
		t.Errorf("accountId = %q; want U1234567", got.AccountID)
	}
}

func TestTradingAccount_BalanceSummary(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)
	ctx := context.Background()

	got, err := cli.TradingAccount().BalanceSummary(ctx, AccountID("U1234567"))
	if err != nil {
		t.Fatalf("BalanceSummary: %v", err)
	}
	if got.Balance != "1000.00" {
		t.Errorf("balance = %q; want 1000.00", got.Balance)
	}
	if got.SettledCash != "500.00" {
		t.Errorf("settledCash = %q; want 500.00", got.SettledCash)
	}
	if got.AccruedInterest != "1.25" {
		t.Errorf("accruedInterest = %q; want 1.25", got.AccruedInterest)
	}
}

func TestTradingAccount_MarginSummary(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)
	ctx := context.Background()

	got, err := cli.TradingAccount().MarginSummary(ctx, AccountID("U1234567"))
	if err != nil {
		t.Fatalf("MarginSummary: %v", err)
	}
	if got.InitialMargin != "100.00" {
		t.Errorf("initialMargin = %q; want 100.00", got.InitialMargin)
	}
	if got.MaintenanceMargin != "80.00" {
		t.Errorf("maintenanceMargin = %q; want 80.00", got.MaintenanceMargin)
	}
	if got.ExcessLiquidity != "900.00" {
		t.Errorf("excessLiquidity = %q; want 900.00", got.ExcessLiquidity)
	}
}

func TestTradingAccount_AccountMarketSummary(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)
	ctx := context.Background()

	got, err := cli.TradingAccount().AccountMarketSummary(ctx, AccountID("U1234567"))
	if err != nil {
		t.Fatalf("AccountMarketSummary: %v", err)
	}
	if got.Stock != "1000.50" {
		t.Errorf("stock = %q; want 1000.50", got.Stock)
	}
	if got.Futures != "0" {
		t.Errorf("futures = %q; want 0", got.Futures)
	}
	if got.StocksAndFutures != "1000.50" {
		t.Errorf("stocksAndFutures = %q; want 1000.50", got.StocksAndFutures)
	}
}

func TestTradingAccount_SetDynamicAccount(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)
	ctx := context.Background()

	if err := cli.TradingAccount().SetDynamicAccount(ctx, AccountID("U1234567")); err != nil {
		t.Fatalf("SetDynamicAccount: %v", err)
	}
}

func TestTradingAccount_BrokerageAccounts(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)
	ctx := context.Background()

	raw, err := cli.TradingAccount().BrokerageAccounts(ctx)
	if err != nil {
		t.Fatalf("BrokerageAccounts: %v", err)
	}
	var got struct {
		Accounts []string          `json:"accounts"`
		Aliases  map[string]string `json:"aliases"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode brokerage accounts: %v", err)
	}
	if len(got.Accounts) != 2 {
		t.Fatalf("accounts = %+v; want two", got.Accounts)
	}
	if got.Accounts[0] != "U1234567" || got.Accounts[1] != "U7654321" {
		t.Errorf("accounts = %+v; want [U1234567 U7654321]", got.Accounts)
	}
	if got.Aliases["U1234567"] != "Main" {
		t.Errorf("aliases[U1234567] = %q; want Main", got.Aliases["U1234567"])
	}
}
