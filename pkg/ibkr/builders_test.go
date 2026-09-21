// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// OrderBuilder tests
// ---------------------------------------------------------------------------

func TestOrderBuilder_Defaults(t *testing.T) {
	b := NewOrderBuilder()
	req, err := b.
		ConID(265598).
		Quantity("10").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Side != SideBuy {
		t.Errorf("default Side = %q, want %q", req.Side, SideBuy)
	}
	if req.OrderType != OrderTypeLimit {
		t.Errorf("default OrderType = %q, want %q", req.OrderType, OrderTypeLimit)
	}
	if req.TimeInForce != TimeInForceDay {
		t.Errorf("default TimeInForce = %q, want %q", req.TimeInForce, TimeInForceDay)
	}
}

func TestOrderBuilder_Chain(t *testing.T) {
	req, err := NewOrderBuilder().
		ConID(265598).
		Side(SideSell).
		Type(OrderTypeMarket).
		Quantity("5").
		LimitPrice("150.00").
		StopPrice("145.00").
		TimeInForce(TimeInForceGTC).
		OutsideRTH(true).
		AllOrNone(true).
		ClientOrderID("my-ref").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.ConID != 265598 {
		t.Errorf("ConID = %d, want 265598", req.ConID)
	}
	if req.Side != SideSell {
		t.Errorf("Side = %q, want %q", req.Side, SideSell)
	}
	if req.OrderType != OrderTypeMarket {
		t.Errorf("OrderType = %q, want %q", req.OrderType, OrderTypeMarket)
	}
	if req.Quantity != "5" {
		t.Errorf("Quantity = %q, want %q", req.Quantity, "5")
	}
	if req.LimitPrice != "150.00" {
		t.Errorf("LimitPrice = %q, want %q", req.LimitPrice, "150.00")
	}
	if req.StopPrice != "145.00" {
		t.Errorf("StopPrice = %q, want %q", req.StopPrice, "145.00")
	}
	if req.TimeInForce != TimeInForceGTC {
		t.Errorf("TimeInForce = %q, want %q", req.TimeInForce, TimeInForceGTC)
	}
	if !req.OutsideRTH {
		t.Error("OutsideRTH = false, want true")
	}
	if !req.AllOrNone {
		t.Error("AllOrNone = false, want true")
	}
	if req.ClientOrderID != "my-ref" {
		t.Errorf("ClientOrderID = %q, want %q", req.ClientOrderID, "my-ref")
	}
}

func TestOrderBuilder_MissingConID(t *testing.T) {
	_, err := NewOrderBuilder().
		Quantity("10").
		Build()
	if err == nil {
		t.Fatal("expected error for missing ConID")
	}
}

func TestOrderBuilder_MissingQuantity(t *testing.T) {
	_, err := NewOrderBuilder().
		ConID(265598).
		Build()
	if err == nil {
		t.Fatal("expected error for missing Quantity")
	}
}

func TestOrderBuilder_ZeroConID(t *testing.T) {
	_, err := NewOrderBuilder().
		ConID(0).
		Quantity("10").
		Build()
	if err == nil {
		t.Fatal("expected error for zero ConID")
	}
}

func TestOrderBuilder_InvalidSide(t *testing.T) {
	_, err := NewOrderBuilder().
		ConID(265598).
		Quantity("10").
		Side("INVALID").
		Build()
	if err == nil {
		t.Fatal("expected error for invalid Side")
	}
}

func TestOrderBuilder_EmptyType(t *testing.T) {
	_, err := NewOrderBuilder().
		ConID(265598).
		Quantity("10").
		Type("").
		Build()
	if err == nil {
		t.Fatal("expected error for empty Type")
	}
}

func TestOrderBuilder_FirstErrorWins(t *testing.T) {
	_, err := NewOrderBuilder().
		ConID(0).
		Quantity("").
		Side("BAD").
		Build()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "ConID") {
		t.Errorf("first error should mention ConID, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ContractBuilder tests
// ---------------------------------------------------------------------------

func TestContractBuilder_Happy(t *testing.T) {
	c, err := NewContractBuilder().
		ConID(265598).
		Symbol("AAPL").
		Exchange("SMART").
		SecType("STK").
		Currency("USD").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ConID != 265598 {
		t.Errorf("ConID = %d, want 265598", c.ConID)
	}
	if c.Symbol != "AAPL" {
		t.Errorf("Symbol = %q, want %q", c.Symbol, "AAPL")
	}
	if c.Exchange != "SMART" {
		t.Errorf("Exchange = %q, want %q", c.Exchange, "SMART")
	}
	if c.InstrumentType != "STK" {
		t.Errorf("InstrumentType = %q, want %q", c.InstrumentType, "STK")
	}
	if c.Currency != "USD" {
		t.Errorf("Currency = %q, want %q", c.Currency, "USD")
	}
}

func TestContractBuilder_MissingSymbol(t *testing.T) {
	_, err := NewContractBuilder().
		ConID(265598).
		Exchange("SMART").
		SecType("STK").
		Currency("USD").
		Build()
	if err == nil {
		t.Fatal("expected error for missing Symbol")
	}
}

func TestContractBuilder_MissingExchange(t *testing.T) {
	_, err := NewContractBuilder().
		ConID(265598).
		Symbol("AAPL").
		SecType("STK").
		Currency("USD").
		Build()
	if err == nil {
		t.Fatal("expected error for missing Exchange")
	}
}

func TestContractBuilder_MissingSecType(t *testing.T) {
	_, err := NewContractBuilder().
		ConID(265598).
		Symbol("AAPL").
		Exchange("SMART").
		Currency("USD").
		Build()
	if err == nil {
		t.Fatal("expected error for missing SecType")
	}
}

func TestContractBuilder_MissingCurrency(t *testing.T) {
	_, err := NewContractBuilder().
		ConID(265598).
		Symbol("AAPL").
		Exchange("SMART").
		SecType("STK").
		Build()
	if err == nil {
		t.Fatal("expected error for missing Currency")
	}
}

func TestContractBuilder_MissingConID(t *testing.T) {
	_, err := NewContractBuilder().
		Symbol("AAPL").
		Exchange("SMART").
		SecType("STK").
		Currency("USD").
		Build()
	if err == nil {
		t.Fatal("expected error for missing ConID")
	}
}

func TestContractBuilder_EmptySymbol(t *testing.T) {
	_, err := NewContractBuilder().
		ConID(265598).
		Symbol("").
		Build()
	if err == nil {
		t.Fatal("expected error for empty Symbol")
	}
}

// ---------------------------------------------------------------------------
// TransferInstructionBuilder tests
// ---------------------------------------------------------------------------

func TestTransferInstructionBuilder_Happy(t *testing.T) {
	price := "150.00"
	tradeDate := "20260101"
	settleDate := "20260102"
	instr, err := NewTransferInstructionBuilder().
		SourceAccountID("U1234567").
		TargetAccountID("U7654321").
		ConID(265598).
		Quantity("100").
		Price(&price).
		TradeDate(&tradeDate).
		SettleDate(&settleDate).
		ClientInstructionID("my-inst").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if instr.SourceAccountID != "U1234567" {
		t.Errorf("SourceAccountID = %q, want %q", instr.SourceAccountID, "U1234567")
	}
	if instr.TargetAccountID != "U7654321" {
		t.Errorf("TargetAccountID = %q, want %q", instr.TargetAccountID, "U7654321")
	}
	if instr.ConID != 265598 {
		t.Errorf("ConID = %d, want 265598", instr.ConID)
	}
	if instr.TransferQuantity != "100" {
		t.Errorf("TransferQuantity = %q, want %q", instr.TransferQuantity, "100")
	}
	if instr.TransferPrice == nil || *instr.TransferPrice != "150.00" {
		t.Errorf("TransferPrice = %v, want 150.00", instr.TransferPrice)
	}
	if instr.TradeDate == nil || *instr.TradeDate != "20260101" {
		t.Errorf("TradeDate = %v, want 20260101", instr.TradeDate)
	}
	if instr.SettleDate == nil || *instr.SettleDate != "20260102" {
		t.Errorf("SettleDate = %v, want 20260102", instr.SettleDate)
	}
	if instr.ClientInstructionID != "my-inst" {
		t.Errorf("ClientInstructionID = %q, want %q", instr.ClientInstructionID, "my-inst")
	}
}

func TestTransferInstructionBuilder_MissingSourceAccount(t *testing.T) {
	_, err := NewTransferInstructionBuilder().
		TargetAccountID("U7654321").
		ConID(265598).
		Quantity("100").
		Build()
	if err == nil {
		t.Fatal("expected error for missing SourceAccountID")
	}
}

func TestTransferInstructionBuilder_MissingTargetAccount(t *testing.T) {
	_, err := NewTransferInstructionBuilder().
		SourceAccountID("U1234567").
		ConID(265598).
		Quantity("100").
		Build()
	if err == nil {
		t.Fatal("expected error for missing TargetAccountID")
	}
}

func TestTransferInstructionBuilder_MissingConID(t *testing.T) {
	_, err := NewTransferInstructionBuilder().
		SourceAccountID("U1234567").
		TargetAccountID("U7654321").
		Quantity("100").
		Build()
	if err == nil {
		t.Fatal("expected error for missing ConID")
	}
}

func TestTransferInstructionBuilder_MissingQuantity(t *testing.T) {
	_, err := NewTransferInstructionBuilder().
		SourceAccountID("U1234567").
		TargetAccountID("U7654321").
		ConID(265598).
		Build()
	if err == nil {
		t.Fatal("expected error for missing Quantity")
	}
}

func TestTransferInstructionBuilder_OptionalFieldsNil(t *testing.T) {
	instr, err := NewTransferInstructionBuilder().
		SourceAccountID("U1234567").
		TargetAccountID("U7654321").
		ConID(265598).
		Quantity("10").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if instr.TransferPrice != nil {
		t.Errorf("TransferPrice should be nil, got %v", instr.TransferPrice)
	}
	if instr.TradeDate != nil {
		t.Errorf("TradeDate should be nil, got %v", instr.TradeDate)
	}
	if instr.SettleDate != nil {
		t.Errorf("SettleDate should be nil, got %v", instr.SettleDate)
	}
}
