// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/shing1211/ibkrapi4go/internal/mockgateway"
)

// TestRESTExternalCashTransfers_AmountPrecision pins why the bank-instruction
// amount is a json.Number rather than a float32.
//
// Every other cash-transfer test uses an amount like 250.75 or 1000.50, all of
// which a float32 represents exactly - so those assertions pass whether or not
// the amount was rounded, and prove nothing. A float32 mantissa is 24 bits, so it
// rounds anything above 2^24 (16777216). This is a money-moving request: the
// value the caller supplies is the value the gateway receives, and a caller asking
// to move 12345678.91 was previously putting 12345679 on the wire.
func TestRESTExternalCashTransfers_AmountPrecision(t *testing.T) {
	const amount = "12345678.91"

	for _, tc := range []struct {
		name  string
		depot bool
	}{
		{"deposit", true},
		{"withdraw", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			banking, gw := bankingWithGateway(t)

			_, err := banking.CashTransfers().TransferBulk(context.Background(), []CashTransferRequest{
				{
					AccountID:             AccountID("U1234567"),
					ClientInstructionID:   "1",
					Amount:                amount,
					Currency:              "USD",
					BankInstructionMethod: "ACH",
				},
			}, tc.depot)
			if err != nil {
				t.Fatalf("TransferBulk(%s): %v", tc.name, err)
			}

			req := lastRESTRequest(t, gw)
			var body struct {
				Instructions []struct {
					Amount json.RawMessage `json:"amount"`
				} `json:"instructions"`
			}
			decodeRequestBody(t, req, &body)
			if len(body.Instructions) != 1 {
				t.Fatalf("instructions = %+v; want one", body.Instructions)
			}

			// Exactly the caller's digits, and a bare number rather than a quoted
			// string, because the spec declares this field as `type: number`.
			if got := string(body.Instructions[0].Amount); got != amount {
				t.Errorf("amount on the wire = %s; want %s - the caller's digits must survive "+
					"unchanged, and it must stay a JSON number", got, amount)
			}

			// Show what the float32 path produced, so the expectation above is not
			// a literal somebody picked.
			var asFloat32 float32 = 12345678.91
			if rounded := float32ToStrForTest(asFloat32); rounded == amount {
				t.Errorf("float32 held %s exactly (%q); this test no longer proves the fix",
					amount, rounded)
			}
		})
	}
}

// float32ToStrForTest renders a float32 the way the old code did, so the test can
// show the value the float path would have produced.
func float32ToStrForTest(v float32) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// TestRESTExternalCashTransfers_AmountEmptyIsZero keeps the previous behaviour for
// an amount the caller did not set. A zero-value json.Number is the empty string,
// which encoding/json refuses to marshal, so moneyToNumber substitutes 0.
func TestRESTExternalCashTransfers_AmountEmptyIsZero(t *testing.T) {
	banking, gw := bankingWithGateway(t)

	_, err := banking.CashTransfers().TransferBulk(context.Background(), []CashTransferRequest{
		{
			AccountID:             AccountID("U1234567"),
			ClientInstructionID:   "1",
			Currency:              "USD",
			BankInstructionMethod: "ACH",
		},
	}, true)
	if err != nil {
		t.Fatalf("TransferBulk: %v", err)
	}

	req := lastRESTRequest(t, gw)
	var body struct {
		Instructions []struct {
			Amount json.RawMessage `json:"amount"`
		} `json:"instructions"`
	}
	decodeRequestBody(t, req, &body)
	if len(body.Instructions) != 1 {
		t.Fatalf("instructions = %+v; want one", body.Instructions)
	}
	if got, want := string(body.Instructions[0].Amount), "0"; got != want {
		t.Errorf("empty amount went out as %s; want %s - an unset amount must not be an "+
			"empty json.Number, which encoding/json rejects", got, want)
	}
}

// unused reference so the mockgateway import stays meaningful if the fixtures move
var _ = mockgateway.OpCreateExternalCashTransfers
