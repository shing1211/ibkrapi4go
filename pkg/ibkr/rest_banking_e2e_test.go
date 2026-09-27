// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/shing1211/ibkrapi4go/client"
	"github.com/shing1211/ibkrapi4go/internal"
	"github.com/shing1211/ibkrapi4go/internal/mockgateway"
)

// bankingWithGateway returns the banking sub-manager together with the mock
// gateway behind it. Every wrapper here POSTs or GETs a banking endpoint whose
// observable product is either a decoded value or the instruction set ID echoed
// back as a string, so each test reads the value the caller gets and the
// request the gateway recorded.
func bankingWithGateway(t *testing.T) (*RESTBanking, *gateway) {
	t.Helper()
	cli, gw := newRESTClientWithGateway(t)
	surface, err := cli.REST()
	if err != nil {
		t.Fatalf("REST: %v", err)
	}
	return surface.Banking(), gw
}

// bankingRequestPosts asserts the shared contract of every banking POST: the
// call is a JSON POST and the recorded path is want. The generated
// *WithBodyWithResponse methods set no Content-Type of their own, so the
// "application/json" argument the wrappers pass is what puts the header on the
// wire.
func bankingRequestPosts(t *testing.T, req *mockgateway.Request, want string) {
	t.Helper()
	if req.Method != http.MethodPost {
		t.Errorf("method = %s; want POST", req.Method)
	}
	if req.Path != want {
		t.Errorf("path = %q; want %q", req.Path, want)
	}
	if got := req.Headers.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q; want application/json", got)
	}
}

// bankingRequestGets asserts the method and path of a banking GET. The three
// instruction lookups are the only GETs in this file.
func bankingRequestGets(t *testing.T, req *mockgateway.Request, want string) {
	t.Helper()
	if req.Method != http.MethodGet {
		t.Errorf("method = %s; want GET", req.Method)
	}
	if req.Path != want {
		t.Errorf("path = %q; want %q", req.Path, want)
	}
}

// jsonKeys returns the sorted top-level keys of the recorded request body, so a
// test can pin the exact key set rather than only the fields it cares about.
// Sorted because Go map iteration order is not deterministic.
func jsonKeys(t *testing.T, req *mockgateway.Request) []string {
	t.Helper()
	var m map[string]json.RawMessage
	decodeRequestBody(t, req, &m)
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ---------------------------------------------------------------------------
// Instruction lookups
// ---------------------------------------------------------------------------

func TestRESTBanking_ClientInstruction(t *testing.T) {
	banking, gw := bankingWithGateway(t)

	ci, err := banking.ClientInstruction(context.Background(), 2001)
	if err != nil {
		t.Fatalf("ClientInstruction: %v", err)
	}
	if ci == nil {
		t.Fatal("ClientInstruction = nil; want a decoded instruction")
	}
	if ci.ID != 2001 {
		t.Errorf("id = %d; want 2001", ci.ID)
	}
	if ci.AccountID != AccountID("U1234567") {
		t.Errorf("accountId = %q; want U1234567", ci.AccountID)
	}
	// Type comes from the body's `instructionType`, not the caller's wording.
	if ci.Type != "FOP" {
		t.Errorf("type = %q; want FOP", ci.Type)
	}
	if ci.Status != "PENDING" {
		t.Errorf("status = %q; want PENDING", ci.Status)
	}
	want := time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC)
	if !ci.CreatedAt.Equal(want) {
		t.Errorf("createdAt = %s; want %s", ci.CreatedAt, want)
	}
	if len(ci.Instructions) != 1 {
		t.Fatalf("instructions = %+v; want one sub-instruction", ci.Instructions)
	}
	sub := ci.Instructions[0]
	if sub.ID != 2001 {
		t.Errorf("instructions[0].id = %d; want 2001", sub.ID)
	}
	if sub.Type != "FOP" {
		t.Errorf("instructions[0].type = %q; want FOP", sub.Type)
	}
	if sub.Status != "PENDING" {
		t.Errorf("instructions[0].status = %q; want PENDING", sub.Status)
	}
	if !sub.CreatedAt.Equal(want) {
		t.Errorf("instructions[0].createdAt = %s; want %s", sub.CreatedAt, want)
	}

	bankingRequestGets(t, lastRESTRequest(t, gw), "/gw/api/v1/client-instructions/2001")
}

// TestRESTBanking_ClientInstruction_EmptyBody pins the absent-field arm of
// clientInstructionRaw.toPublic: every field is a pointer with omitempty, so a
// 200 carrying none of them must decode to a zero-valued struct and an empty
// sub-instruction slice rather than an error or a nil pointer.
func TestRESTBanking_ClientInstruction_EmptyBody(t *testing.T) {
	banking, gw := bankingWithGateway(t)
	gw.srv.Fixtures().Set(mockgateway.OpGetClientInstructions, mockgateway.Fixture{Body: `{}`})

	ci, err := banking.ClientInstruction(context.Background(), 2001)
	if err != nil {
		t.Fatalf("ClientInstruction: %v", err)
	}
	if ci == nil {
		t.Fatal("ClientInstruction = nil; want a zero-valued struct")
	}
	if ci.ID != 0 || ci.AccountID != "" || ci.Type != "" || ci.Status != "" {
		t.Errorf("ci = %+v; want all scalar fields zero for an empty body", ci)
	}
	if !ci.CreatedAt.IsZero() {
		t.Errorf("createdAt = %s; want the zero time when createdAt is absent", ci.CreatedAt)
	}
	if len(ci.Instructions) != 0 {
		t.Errorf("instructions = %+v; want empty for a body with no instructions key", ci.Instructions)
	}
}

func TestRESTBanking_InstructionSet(t *testing.T) {
	banking, gw := bankingWithGateway(t)

	set, err := banking.InstructionSet(context.Background(), 3001)
	if err != nil {
		t.Fatalf("InstructionSet: %v", err)
	}
	if set.ID != 3001 {
		t.Errorf("id = %d; want 3001", set.ID)
	}
	if set.AccountID != AccountID("U1234567") {
		t.Errorf("accountId = %q; want U1234567", set.AccountID)
	}
	// instructionSetRaw has a Status field, but the shared instruction-set
	// fixture carries no `status` key, so there is nothing to decode. Asserted
	// explicitly so a fixture that later adds one fails here rather than making
	// this silently stop covering the empty case.
	if set.Status != "" {
		t.Errorf("status = %q; want empty — the fixture has no status key", set.Status)
	}
	if len(set.Instructions) != 1 {
		t.Fatalf("instructions = %+v; want one", set.Instructions)
	}
	sub := set.Instructions[0]
	if sub.ID != 2001 {
		t.Errorf("instructions[0].id = %d; want 2001", sub.ID)
	}
	if sub.Type != "FOP" {
		t.Errorf("instructions[0].type = %q; want FOP", sub.Type)
	}
	if sub.Status != "PENDING" {
		t.Errorf("instructions[0].status = %q; want PENDING", sub.Status)
	}
	want := time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC)
	if !sub.CreatedAt.Equal(want) {
		t.Errorf("instructions[0].createdAt = %s; want %s", sub.CreatedAt, want)
	}

	bankingRequestGets(t, lastRESTRequest(t, gw), "/gw/api/v1/instruction-sets/3001")
}

func TestRESTBanking_Instruction(t *testing.T) {
	banking, gw := bankingWithGateway(t)

	ins, err := banking.Instruction(context.Background(), 2001)
	if err != nil {
		t.Fatalf("Instruction: %v", err)
	}
	if ins.ID != 2001 {
		t.Errorf("id = %d; want 2001", ins.ID)
	}
	if ins.Type != "FOP" {
		t.Errorf("type = %q; want FOP", ins.Type)
	}
	if ins.Status != "PENDING" {
		t.Errorf("status = %q; want PENDING", ins.Status)
	}
	want := time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC)
	if !ins.CreatedAt.Equal(want) {
		t.Errorf("createdAt = %s; want %s", ins.CreatedAt, want)
	}

	bankingRequestGets(t, lastRESTRequest(t, gw), "/gw/api/v1/instructions/2001")
}

// ---------------------------------------------------------------------------
// Transaction history
// ---------------------------------------------------------------------------

func TestRESTBanking_QueryTransactions(t *testing.T) {
	banking, gw := bankingWithGateway(t)

	txs, err := banking.QueryTransactions(context.Background(), TransactionQueryRequest{
		AccountID: AccountID("U1234567"),
		Days:      30,
	})
	if err != nil {
		t.Fatalf("QueryTransactions: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("transactions = %+v; want one", txs)
	}
	tx := txs[0]
	if tx.ID != "tx-1" {
		t.Errorf("id = %q; want tx-1", tx.ID)
	}
	// Date is a date-only wire value, parsed with a date-only layout, so the
	// result is midnight UTC on the wire's day.
	want := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	if !tx.Date.Equal(want) {
		t.Errorf("date = %s; want %s", tx.Date, want)
	}
	if tx.Type != "DEPOSIT" {
		t.Errorf("type = %q; want DEPOSIT", tx.Type)
	}
	// Money stays a string (ADR 0008): "1000.00" must not become 1000.
	if tx.Amount != "1000.00" {
		t.Errorf("amount = %q; want \"1000.00\"", tx.Amount)
	}
	if tx.Currency != "USD" {
		t.Errorf("currency = %q; want USD", tx.Currency)
	}
	if tx.Status != "COMPLETED" {
		t.Errorf("status = %q; want COMPLETED", tx.Status)
	}
	if tx.Description != "Synthetic deposit" {
		t.Errorf("description = %q; want Synthetic deposit", tx.Description)
	}

	req := lastRESTRequest(t, gw)
	bankingRequestPosts(t, req, "/gw/api/v1/instructions/query")
	var body struct {
		Instruction struct {
			AccountID           string `json:"accountId"`
			ClientInstructionID int    `json:"clientInstructionId"`
			TransactionHistory  struct {
				DaysToGoBack    float32 `json:"daysToGoBack"`
				TransactionType string  `json:"transactionType"`
			} `json:"transactionHistory"`
		} `json:"instruction"`
	}
	decodeRequestBody(t, req, &body)
	if body.Instruction.AccountID != "U1234567" {
		t.Errorf("accountId = %q; want U1234567", body.Instruction.AccountID)
	}
	// TransactionQueryRequest has no client instruction ID field, so the wrapper
	// always sends 0.
	if body.Instruction.ClientInstructionID != 0 {
		t.Errorf("clientInstructionId = %d; want 0 — the request type has no such field", body.Instruction.ClientInstructionID)
	}
	if body.Instruction.TransactionHistory.DaysToGoBack != 30 {
		t.Errorf("daysToGoBack = %v; want the caller's 30", body.Instruction.TransactionHistory.DaysToGoBack)
	}
	// The wrapper never sets transactionType and the generated model marks it
	// omitempty, so the key must be absent from the history object rather than
	// present-and-empty. Asserted on the raw key set, not on a decoded struct,
	// because a decoded string cannot tell the two apart.
	var hist struct {
		Instruction struct {
			TransactionHistory map[string]json.RawMessage `json:"transactionHistory"`
		} `json:"instruction"`
	}
	decodeRequestBody(t, req, &hist)
	if _, ok := hist.Instruction.TransactionHistory["daysToGoBack"]; !ok {
		t.Errorf("body = %s; want a daysToGoBack key", req.Body)
	}
	if _, ok := hist.Instruction.TransactionHistory["transactionType"]; ok {
		t.Errorf("body = %s; want no transactionType key — the wrapper never sets it", req.Body)
	}
}

// TestRESTBanking_QueryTransactions_DefaultDays pins the two arms of the
// days<=0 default together: a zero and a negative Days must both send 5, the
// value the wrapper documents as the fallback, and never the caller's value.
// Each case is a subtest so it gets its own gateway: the leak check that
// newGateway registers runs at the end of the enclosing test, so two gateways
// alive at once in one test body would report the first as a leak.
func TestRESTBanking_QueryTransactions_DefaultDays(t *testing.T) {
	for _, days := range []int{0, -7} {
		t.Run(fmt.Sprintf("days=%d", days), func(t *testing.T) {
			banking, gw := bankingWithGateway(t)
			if _, err := banking.QueryTransactions(context.Background(), TransactionQueryRequest{
				AccountID: AccountID("U1234567"),
				Days:      days,
			}); err != nil {
				t.Fatalf("QueryTransactions(Days=%d): %v", days, err)
			}
			req := lastRESTRequest(t, gw)
			var body struct {
				Instruction struct {
					TransactionHistory struct {
						DaysToGoBack float32 `json:"daysToGoBack"`
					} `json:"transactionHistory"`
				} `json:"instruction"`
			}
			decodeRequestBody(t, req, &body)
			if body.Instruction.TransactionHistory.DaysToGoBack != 5 {
				t.Errorf("daysToGoBack = %v; want the 5-day default for Days=%d",
					body.Instruction.TransactionHistory.DaysToGoBack, days)
			}
		})
	}
}

// TestRESTBanking_QueryTransactions_EmptyResult pins the nil-Transactions arm
// of transactionsRaw.toPublic: a 202 whose body has no transactions key must
// yield a nil slice and no error.
func TestRESTBanking_QueryTransactions_EmptyResult(t *testing.T) {
	banking, gw := bankingWithGateway(t)
	gw.srv.Fixtures().Set(mockgateway.OpCreateInstructionsQuery, mockgateway.Fixture{
		Status: http.StatusAccepted,
		Body:   `{}`,
	})

	txs, err := banking.QueryTransactions(context.Background(), TransactionQueryRequest{
		AccountID: AccountID("U1234567"),
	})
	if err != nil {
		t.Fatalf("QueryTransactions: %v", err)
	}
	if txs != nil {
		t.Errorf("transactions = %+v; want nil for a body with no transactions key", txs)
	}
}

// ---------------------------------------------------------------------------
// Instruction cancellation
// ---------------------------------------------------------------------------

func TestRESTBanking_CancelInstruction(t *testing.T) {
	banking, gw := bankingWithGateway(t)

	if err := banking.CancelInstruction(context.Background(), CancelInstructionRequest{
		InstructionID: 67890,
		Reason:        "no longer needed",
	}); err != nil {
		t.Fatalf("CancelInstruction: %v", err)
	}

	req := lastRESTRequest(t, gw)
	bankingRequestPosts(t, req, "/gw/api/v1/instructions/cancel")
	// The whole request product is the recorded body: this wrapper returns only
	// an error, so the body is the sole observable proof of what was cancelled.
	var body struct {
		Instruction struct {
			InstructionID int    `json:"instructionId"`
			Reason        string `json:"reason"`
		} `json:"instruction"`
	}
	decodeRequestBody(t, req, &body)
	if body.Instruction.InstructionID != 67890 {
		t.Errorf("instructionId = %d; want 67890", body.Instruction.InstructionID)
	}
	if body.Instruction.Reason != "no longer needed" {
		t.Errorf("reason = %q; want the caller's reason forwarded", body.Instruction.Reason)
	}
}

func TestRESTBanking_CancelInstructionsBulk(t *testing.T) {
	banking, gw := bankingWithGateway(t)

	if err := banking.CancelInstructionsBulk(context.Background(), []CancelInstructionRequest{
		{InstructionID: 1, Reason: "first"},
		{InstructionID: 2, Reason: "second"},
	}); err != nil {
		t.Fatalf("CancelInstructionsBulk: %v", err)
	}

	req := lastRESTRequest(t, gw)
	bankingRequestPosts(t, req, "/gw/api/v1/instructions/cancel:bulk")
	// The bulk body is an array of bare instruction objects, not the
	// {"instruction": ...} envelope the single-cancel path wraps. Pinned here
	// because the shape difference is invisible in the returned error.
	var body struct {
		Instructions []struct {
			InstructionID int    `json:"instructionId"`
			Reason        string `json:"reason"`
		} `json:"instructions"`
	}
	decodeRequestBody(t, req, &body)
	if len(body.Instructions) != 2 {
		t.Fatalf("instructions = %+v; want two", body.Instructions)
	}
	for i, want := range []int{1, 2} {
		if body.Instructions[i].InstructionID != want {
			t.Errorf("instructions[%d].instructionId = %d; want %d", i, body.Instructions[i].InstructionID, want)
		}
		// Each CancelInstructionRequest carries a Reason, but the bulk payload
		// only populates InstructionId, so the caller's reason never reaches the
		// wire. Asserted as the current contract so a future fix is a visible
		// change here rather than a silent one.
		if body.Instructions[i].Reason != "" {
			t.Errorf("instructions[%d].reason = %q; want empty — the bulk payload never populates it",
				i, body.Instructions[i].Reason)
		}
	}
}

// ---------------------------------------------------------------------------
// External asset transfers (V2)
// ---------------------------------------------------------------------------

func TestRESTExternalAssetTransfers_TransferV2(t *testing.T) {
	banking, gw := bankingWithGateway(t)

	id, err := banking.ExternalTransfers().TransferV2(context.Background(), AssetTransferRequest{
		AccountID:             AccountID("U1234567"),
		ClientInstructionID:   "1012983",
		ContraBrokerAccountID: AccountID("12345678A"),
		ContraBrokerDtcCode:   "534",
		Direction:             "IN",
		Quantity:              "100",
		ConID:                 ConID(459200101),
		Positions: []PositionV2Request{
			{ConID: ConID(459200101), Quantity: "10"},
			{ConID: ConID(765432100), Quantity: "2.5"},
		},
	})
	if err != nil {
		t.Fatalf("TransferV2: %v", err)
	}
	// The v2 acknowledgement's instructionSetId is an int, rendered with Itoa.
	if id != "1001" {
		t.Errorf("id = %q; want 1001 from the acknowledgement", id)
	}

	req := lastRESTRequest(t, gw)
	bankingRequestPosts(t, req, "/gw/api/v2/external-asset-transfers")
	if keys := jsonKeys(t, req); len(keys) != 2 || keys[0] != "instruction" || keys[1] != "instructionType" {
		t.Errorf("body keys = %v; want exactly [instruction instructionType]", keys)
	}
	var body struct {
		Instruction struct {
			AccountID             string `json:"accountId"`
			ClientInstructionID   int    `json:"clientInstructionId"`
			ContraBrokerAccountID string `json:"contraBrokerAccountId"`
			ContraBrokerDtcCode   string `json:"contraBrokerDtcCode"`
			Currency              string `json:"currency"`
			Direction             string `json:"direction"`
			Positions             []struct {
				ConID    int     `json:"conid"`
				Quantity float32 `json:"quantity"`
			} `json:"positions"`
		} `json:"instruction"`
		InstructionType string `json:"instructionType"`
	}
	decodeRequestBody(t, req, &body)
	if body.InstructionType != "FOP" {
		t.Errorf("instructionType = %q; want FOP", body.InstructionType)
	}
	if body.Instruction.AccountID != "U1234567" {
		t.Errorf("accountId = %q; want U1234567", body.Instruction.AccountID)
	}
	// The string client instruction ID is converted to the int the model wants.
	if body.Instruction.ClientInstructionID != 1012983 {
		t.Errorf("clientInstructionId = %d; want 1012983", body.Instruction.ClientInstructionID)
	}
	if body.Instruction.ContraBrokerAccountID != "12345678A" {
		t.Errorf("contraBrokerAccountId = %q; want 12345678A", body.Instruction.ContraBrokerAccountID)
	}
	if body.Instruction.ContraBrokerDtcCode != "534" {
		t.Errorf("contraBrokerDtcCode = %q; want 534", body.Instruction.ContraBrokerDtcCode)
	}
	if body.Instruction.Direction != "IN" {
		t.Errorf("direction = %q; want IN", body.Instruction.Direction)
	}
	// AssetTransferRequest has no Currency field, and the generated model marks
	// currency non-omitempty, so an empty string is sent rather than the key
	// being dropped.
	if body.Instruction.Currency != "" {
		t.Errorf("currency = %q; want empty — AssetTransferRequest exposes no currency", body.Instruction.Currency)
	}
	if len(body.Instruction.Positions) != 2 {
		t.Fatalf("positions = %+v; want two", body.Instruction.Positions)
	}
	// The single-transfer V2 path takes quantities from Positions, not from the
	// top-level Quantity field: V2 splits a transfer into per-position amounts.
	if body.Instruction.Positions[0].ConID != 459200101 {
		t.Errorf("positions[0].conid = %d; want 459200101", body.Instruction.Positions[0].ConID)
	}
	if body.Instruction.Positions[0].Quantity != 10 {
		t.Errorf("positions[0].quantity = %v; want 10 as a JSON number", body.Instruction.Positions[0].Quantity)
	}
	if body.Instruction.Positions[1].ConID != 765432100 {
		t.Errorf("positions[1].conid = %d; want 765432100", body.Instruction.Positions[1].ConID)
	}
	if body.Instruction.Positions[1].Quantity != 2.5 {
		t.Errorf("positions[1].quantity = %v; want 2.5 as a JSON number", body.Instruction.Positions[1].Quantity)
	}
	// The top-level Quantity is therefore not represented anywhere on the wire
	// for this call. Asserted via the key set of the instruction object so the
	// claim is checkable rather than inferred.
	var instr map[string]json.RawMessage
	decodeRequestBody(t, req, &struct {
		Instruction map[string]json.RawMessage `json:"instruction"`
	}{Instruction: instr})
	if _, ok := instr["quantity"]; ok {
		t.Errorf("body = %s; want no top-level quantity key in a v2 instruction", req.Body)
	}
}

// TestRESTExternalAssetTransfers_TransferV2_NoPositions pins the empty-positions
// arm: a request with no positions must send an empty JSON array rather than
// null, because the wrapper allocates the slice with make(len(req.Positions)).
func TestRESTExternalAssetTransfers_TransferV2_NoPositions(t *testing.T) {
	banking, gw := bankingWithGateway(t)

	if _, err := banking.ExternalTransfers().TransferV2(context.Background(), AssetTransferRequest{
		AccountID:           AccountID("U1234567"),
		ClientInstructionID: "1",
		Direction:           "OUT",
	}); err != nil {
		t.Fatalf("TransferV2: %v", err)
	}
	req := lastRESTRequest(t, gw)
	var body struct {
		Instruction struct {
			Positions []struct {
				ConID int `json:"conid"`
			} `json:"positions"`
		} `json:"instruction"`
	}
	decodeRequestBody(t, req, &body)
	if body.Instruction.Positions == nil {
		t.Errorf("body = %s; want an empty positions array, not null", req.Body)
	}
	if len(body.Instruction.Positions) != 0 {
		t.Errorf("positions = %+v; want empty", body.Instruction.Positions)
	}
}

func TestRESTExternalAssetTransfers_TransferBulkV2(t *testing.T) {
	banking, gw := bankingWithGateway(t)

	results, err := banking.ExternalTransfers().TransferBulkV2(context.Background(), []AssetTransferRequest{
		{
			AccountID:             AccountID("U1234567"),
			ClientInstructionID:   "1",
			ContraBrokerAccountID: AccountID("12345678A"),
			ContraBrokerDtcCode:   "534",
			Direction:             "IN",
			Positions:             []PositionV2Request{{ConID: ConID(459200101), Quantity: "10"}},
		},
		{
			AccountID:           AccountID("U7654321"),
			ClientInstructionID: "2",
			Direction:           "OUT",
		},
	})
	if err != nil {
		t.Fatalf("TransferBulkV2: %v", err)
	}
	// The shared bulk acknowledgement fixture carries exactly one result.
	if len(results) != 1 {
		t.Fatalf("results = %+v; want one", results)
	}
	got := results[0]
	if got.ClientInstructionID != 1 {
		t.Errorf("clientInstructionId = %d; want 1", got.ClientInstructionID)
	}
	if got.InstructionID != 2001 {
		t.Errorf("instructionId = %d; want 2001", got.InstructionID)
	}
	if got.InstructionStatus != "PENDING" {
		t.Errorf("instructionStatus = %q; want PENDING", got.InstructionStatus)
	}
	// The fixture carries ibReferenceId, so the non-nil arm of
	// intPtrToInt64Ptr runs and the value is readable through the pointer.
	if got.IbReferenceID == nil {
		t.Fatal("ibReferenceId = nil; want 9001 from the acknowledgement")
	}
	if *got.IbReferenceID != 9001 {
		t.Errorf("ibReferenceId = %d; want 9001", *got.IbReferenceID)
	}
	if got.Description == nil {
		t.Fatal("description = nil; want the fixture's polling guidance")
	}
	if *got.Description != "Please poll for status after 10 minutes" {
		t.Errorf("description = %q; want the fixture's polling guidance", *got.Description)
	}

	req := lastRESTRequest(t, gw)
	bankingRequestPosts(t, req, "/gw/api/v2/external-asset-transfers:bulk")
	var body struct {
		InstructionType string `json:"instructionType"`
		Instructions    []struct {
			AccountID             string `json:"accountId"`
			ClientInstructionID   int    `json:"clientInstructionId"`
			ContraBrokerAccountID string `json:"contraBrokerAccountId"`
			ContraBrokerDtcCode   string `json:"contraBrokerDtcCode"`
			Direction             string `json:"direction"`
			Positions             []struct {
				ConID    int    `json:"conid"`
				Quantity string `json:"quantity"`
			} `json:"positions"`
		} `json:"instructions"`
	}
	decodeRequestBody(t, req, &body)
	if body.InstructionType != "FOP" {
		t.Errorf("instructionType = %q; want FOP", body.InstructionType)
	}
	if len(body.Instructions) != 2 {
		t.Fatalf("instructions = %+v; want two", body.Instructions)
	}
	first := body.Instructions[0]
	if first.AccountID != "U1234567" || first.ClientInstructionID != 1 {
		t.Errorf("instructions[0] account/client = %q/%d; want U1234567/1", first.AccountID, first.ClientInstructionID)
	}
	if first.ContraBrokerAccountID != "12345678A" || first.ContraBrokerDtcCode != "534" {
		t.Errorf("instructions[0] contra = %q/%q; want 12345678A/534", first.ContraBrokerAccountID, first.ContraBrokerDtcCode)
	}
	if first.Direction != "IN" {
		t.Errorf("instructions[0].direction = %q; want IN", first.Direction)
	}
	if len(first.Positions) != 1 {
		t.Fatalf("instructions[0].positions = %+v; want one", first.Positions)
	}
	if first.Positions[0].ConID != 459200101 {
		t.Errorf("instructions[0].positions[0].conid = %d; want 459200101", first.Positions[0].ConID)
	}
	// Unlike the single v2 path, the bulk v2 path declares its own local
	// position struct with a string quantity, so the same input reaches the wire
	// as "10" here and as 10 above. Asserted so the divergence is on the record.
	if first.Positions[0].Quantity != "10" {
		t.Errorf("instructions[0].positions[0].quantity = %q; want the string \"10\"", first.Positions[0].Quantity)
	}
	second := body.Instructions[1]
	if second.AccountID != "U7654321" || second.ClientInstructionID != 2 || second.Direction != "OUT" {
		t.Errorf("instructions[1] = %+v; want U7654321/2/OUT", second)
	}
	if second.Positions == nil {
		t.Errorf("body = %s; want an empty positions array, not null", req.Body)
	}
}

// ---------------------------------------------------------------------------
// Internal asset transfers (bulk)
// ---------------------------------------------------------------------------

func TestRESTInternalAssetTransfers_TransferBulk(t *testing.T) {
	banking, gw := bankingWithGateway(t)

	price := "150.25"
	tradeDate := "20260102"
	settleDate := "20260105"
	results, err := banking.InternalTransfers().TransferBulk(context.Background(), []InternalAssetTransferRequest{
		{
			ClientInstructionID: "1",
			SourceAccountID:     AccountID("U1234567"),
			TargetAccountID:     AccountID("U7654321"),
			ConID:               ConID(459200101),
			TransferQuantity:    "10",
			TransferPrice:       &price,
			TradeDate:           &tradeDate,
			SettleDate:          &settleDate,
		},
		{
			// All three optional pointers nil: the three omitempty keys must be
			// dropped rather than sent as null.
			ClientInstructionID: "2",
			SourceAccountID:     AccountID("U1234567"),
			TargetAccountID:     AccountID("U7654321"),
			ConID:               ConID(765432100),
			TransferQuantity:    "1.5",
		},
	})
	if err != nil {
		t.Fatalf("TransferBulk: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %+v; want one", results)
	}
	if results[0].ClientInstructionID != 1 || results[0].InstructionID != 2001 {
		t.Errorf("results[0] ids = %d/%d; want 1/2001", results[0].ClientInstructionID, results[0].InstructionID)
	}
	if results[0].InstructionStatus != "PENDING" {
		t.Errorf("results[0].instructionStatus = %q; want PENDING", results[0].InstructionStatus)
	}
	if results[0].IbReferenceID == nil || *results[0].IbReferenceID != 9001 {
		t.Errorf("results[0].ibReferenceId = %v; want 9001", results[0].IbReferenceID)
	}

	req := lastRESTRequest(t, gw)
	bankingRequestPosts(t, req, "/gw/api/v1/internal-asset-transfers:bulk")
	var body struct {
		InstructionType string                 `json:"instructionType"`
		Instructions    []internalTransferJSON `json:"instructions"`
	}
	decodeRequestBody(t, req, &body)
	if body.InstructionType != "INTERNAL_POSITION_TRANSFER" {
		t.Errorf("instructionType = %q; want INTERNAL_POSITION_TRANSFER", body.InstructionType)
	}
	if len(body.Instructions) != 2 {
		t.Fatalf("instructions = %+v; want two", body.Instructions)
	}
	first := body.Instructions[0]
	if first.ClientInstructionId != 1 {
		t.Errorf("instructions[0].clientInstructionId = %d; want 1", first.ClientInstructionId)
	}
	if first.SourceAccountId != "U1234567" || first.TargetAccountId != "U7654321" {
		t.Errorf("instructions[0] accounts = %q->%q; want U1234567->U7654321", first.SourceAccountId, first.TargetAccountId)
	}
	// Quantity and price are strings, not binary floats (ADR 0008): "10" must
	// not become 10 and "150.25" must not become 150.25-as-float.
	if first.TransferQuantity != "10" {
		t.Errorf("instructions[0].transferQuantity = %q; want the string \"10\"", first.TransferQuantity)
	}
	if first.TransferPrice == nil || *first.TransferPrice != "150.25" {
		t.Errorf("instructions[0].transferPrice = %v; want the string \"150.25\"", first.TransferPrice)
	}
	if first.TradeDate == nil || *first.TradeDate != "20260102" {
		t.Errorf("instructions[0].tradeDate = %v; want the string \"20260102\"", first.TradeDate)
	}
	if first.SettleDate == nil || *first.SettleDate != "20260105" {
		t.Errorf("instructions[0].settleDate = %v; want the string \"20260105\"", first.SettleDate)
	}
	// The trading instrument is the nested {"conid": N} object the v1 internal
	// transfer API expects, carried inside each instruction rather than beside
	// it. Decoded into its own struct so the assertion is on the wire shape.
	var withInstrument struct {
		Instructions []struct {
			TradingInstrument struct {
				ConID int `json:"conid"`
			} `json:"tradingInstrument"`
		} `json:"instructions"`
	}
	decodeRequestBody(t, req, &withInstrument)
	if len(withInstrument.Instructions) != 2 {
		t.Fatalf("raw instructions = %+v; want two", withInstrument.Instructions)
	}
	if withInstrument.Instructions[0].TradingInstrument.ConID != 459200101 {
		t.Errorf("instructions[0].tradingInstrument = %+v; want {\"conid\":459200101}",
			withInstrument.Instructions[0].TradingInstrument)
	}
	if withInstrument.Instructions[1].TradingInstrument.ConID != 765432100 {
		t.Errorf("instructions[1].tradingInstrument = %+v; want {\"conid\":765432100}",
			withInstrument.Instructions[1].TradingInstrument)
	}

	// The three optional keys are omitempty, so the all-nil second instruction
	// must not carry them at all.
	var raw []map[string]json.RawMessage
	decodeRequestBody(t, req, &struct {
		Instructions *[]map[string]json.RawMessage `json:"instructions"`
	}{&raw})
	if len(raw) != 2 {
		t.Fatalf("raw instructions = %+v; want two", raw)
	}
	for _, key := range []string{"transferPrice", "tradeDate", "settleDate"} {
		if _, ok := raw[1][key]; ok {
			t.Errorf("body = %s; want no %s key when the pointer is nil", req.Body, key)
		}
		if _, ok := raw[0][key]; !ok {
			t.Errorf("body = %s; want the %s key when the pointer is set", req.Body, key)
		}
	}
}

// ---------------------------------------------------------------------------
// External cash transfers (bulk)
// ---------------------------------------------------------------------------

func TestRESTExternalCashTransfers_TransferBulk_Deposit(t *testing.T) {
	banking, gw := bankingWithGateway(t)
	name := "Main Bank"

	results, err := banking.CashTransfers().TransferBulk(context.Background(), []CashTransferRequest{
		{
			AccountID:             AccountID("U1234567"),
			ClientInstructionID:   "1",
			Amount:                "1000.50",
			Currency:              "USD",
			BankInstructionMethod: "ACH",
			BankInstructionName:   &name,
		},
	}, true)
	if err != nil {
		t.Fatalf("TransferBulk(deposit): %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %+v; want one", results)
	}
	if results[0].ClientInstructionID != 1 || results[0].InstructionID != 2001 {
		t.Errorf("results[0] ids = %d/%d; want 1/2001", results[0].ClientInstructionID, results[0].InstructionID)
	}
	if results[0].InstructionStatus != "PENDING" {
		t.Errorf("results[0].instructionStatus = %q; want PENDING", results[0].InstructionStatus)
	}

	req := lastRESTRequest(t, gw)
	bankingRequestPosts(t, req, "/gw/api/v1/external-cash-transfers:bulk")
	var body struct {
		InstructionType string `json:"instructionType"`
		Instructions    []struct {
			AccountID             string  `json:"accountId"`
			Amount                float32 `json:"amount"`
			BankInstructionMethod string  `json:"bankInstructionMethod"`
			BankInstructionName   *string `json:"bankInstructionName"`
			ClientInstructionID   int     `json:"clientInstructionId"`
			Currency              string  `json:"currency"`
		} `json:"instructions"`
	}
	decodeRequestBody(t, req, &body)
	if body.InstructionType != "DEPOSIT" {
		t.Errorf("instructionType = %q; want DEPOSIT — isDeposit selects it", body.InstructionType)
	}
	if len(body.Instructions) != 1 {
		t.Fatalf("instructions = %+v; want one", body.Instructions)
	}
	got := body.Instructions[0]
	if got.AccountID != "U1234567" {
		t.Errorf("accountId = %q; want U1234567", got.AccountID)
	}
	if got.ClientInstructionID != 1 {
		t.Errorf("clientInstructionId = %d; want 1", got.ClientInstructionID)
	}
	// The generated deposit model declares amount as a JSON number, so the
	// wrapper's string input is converted to float32 and lands as 1000.5.
	if got.Amount != 1000.5 {
		t.Errorf("amount = %v; want 1000.5 as a JSON number", got.Amount)
	}
	if got.BankInstructionMethod != "ACH" {
		t.Errorf("bankInstructionMethod = %q; want ACH", got.BankInstructionMethod)
	}
	if got.BankInstructionName == nil || *got.BankInstructionName != "Main Bank" {
		t.Errorf("bankInstructionName = %v; want Main Bank", got.BankInstructionName)
	}
	if got.Currency != "USD" {
		t.Errorf("currency = %q; want USD", got.Currency)
	}
}

// TestRESTExternalCashTransfers_TransferBulk_DepositNoName pins the nil
// BankInstructionName arm of the deposit branch: the generated model marks the
// field omitempty, so the key must be dropped.
func TestRESTExternalCashTransfers_TransferBulk_DepositNoName(t *testing.T) {
	banking, gw := bankingWithGateway(t)

	if _, err := banking.CashTransfers().TransferBulk(context.Background(), []CashTransferRequest{{
		AccountID:             AccountID("U1234567"),
		ClientInstructionID:   "1",
		Amount:                "100",
		Currency:              "USD",
		BankInstructionMethod: "WIRE",
	}}, true); err != nil {
		t.Fatalf("TransferBulk(deposit): %v", err)
	}
	req := lastRESTRequest(t, gw)
	var raw []map[string]json.RawMessage
	decodeRequestBody(t, req, &struct {
		Instructions *[]map[string]json.RawMessage `json:"instructions"`
	}{&raw})
	if len(raw) != 1 {
		t.Fatalf("raw instructions = %+v; want one", raw)
	}
	if _, ok := raw[0]["bankInstructionName"]; ok {
		t.Errorf("body = %s; want no bankInstructionName key when the pointer is nil", req.Body)
	}
	if _, ok := raw[0]["bankInstructionMethod"]; !ok {
		t.Errorf("body = %s; want the bankInstructionMethod key", req.Body)
	}
}

func TestRESTExternalCashTransfers_TransferBulk_Withdrawal(t *testing.T) {
	banking, gw := bankingWithGateway(t)
	name := "Savings Bank"

	results, err := banking.CashTransfers().TransferBulk(context.Background(), []CashTransferRequest{{
		AccountID:             AccountID("U1234567"),
		ClientInstructionID:   "7",
		Amount:                "250.75",
		Currency:              "USD",
		BankInstructionMethod: "WIRE",
		BankInstructionName:   &name,
	}}, false)
	if err != nil {
		t.Fatalf("TransferBulk(withdrawal): %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %+v; want one", results)
	}
	if results[0].ClientInstructionID != 1 {
		t.Errorf("results[0].clientInstructionId = %d; want 1 from the acknowledgement", results[0].ClientInstructionID)
	}

	req := lastRESTRequest(t, gw)
	bankingRequestPosts(t, req, "/gw/api/v1/external-cash-transfers:bulk")
	var body struct {
		InstructionType string `json:"instructionType"`
		Instructions    []struct {
			AccountID             string  `json:"accountId"`
			Amount                float32 `json:"amount"`
			BankInstructionMethod string  `json:"bankInstructionMethod"`
			BankInstructionName   string  `json:"bankInstructionName"`
			ClientInstructionID   int     `json:"clientInstructionId"`
			Currency              string  `json:"currency"`
		} `json:"instructions"`
	}
	decodeRequestBody(t, req, &body)
	if body.InstructionType != "WITHDRAWAL" {
		t.Errorf("instructionType = %q; want WITHDRAWAL — isDeposit=false selects it", body.InstructionType)
	}
	if len(body.Instructions) != 1 {
		t.Fatalf("instructions = %+v; want one", body.Instructions)
	}
	got := body.Instructions[0]
	if got.Amount != 250.75 {
		t.Errorf("amount = %v; want 250.75 as a JSON number", got.Amount)
	}
	// The withdrawal model declares bankInstructionName as a required string, so
	// derefStr has to flatten the caller's pointer; a non-nil pointer is
	// forwarded verbatim.
	if got.BankInstructionName != "Savings Bank" {
		t.Errorf("bankInstructionName = %q; want Savings Bank", got.BankInstructionName)
	}
	if got.BankInstructionMethod != "WIRE" {
		t.Errorf("bankInstructionMethod = %q; want WIRE", got.BankInstructionMethod)
	}
	if got.ClientInstructionID != 7 {
		t.Errorf("clientInstructionId = %d; want 7", got.ClientInstructionID)
	}
}

// TestRESTExternalCashTransfers_TransferBulk_WithdrawalNoName pins derefStr's
// nil arm: the withdrawal model requires a string name, so a nil pointer becomes
// an empty string rather than being dropped. That is why the key is present and
// empty here while the deposit branch omits it entirely.
func TestRESTExternalCashTransfers_TransferBulk_WithdrawalNoName(t *testing.T) {
	banking, gw := bankingWithGateway(t)

	if _, err := banking.CashTransfers().TransferBulk(context.Background(), []CashTransferRequest{{
		AccountID:             AccountID("U1234567"),
		ClientInstructionID:   "1",
		Amount:                "100",
		Currency:              "USD",
		BankInstructionMethod: "ACH",
	}}, false); err != nil {
		t.Fatalf("TransferBulk(withdrawal): %v", err)
	}
	req := lastRESTRequest(t, gw)
	var body struct {
		Instructions []struct {
			BankInstructionName *string `json:"bankInstructionName"`
		} `json:"instructions"`
	}
	decodeRequestBody(t, req, &body)
	if len(body.Instructions) != 1 {
		t.Fatalf("instructions = %+v; want one", body.Instructions)
	}
	if body.Instructions[0].BankInstructionName == nil {
		t.Fatalf("body = %s; want an empty bankInstructionName, not a missing key — derefStr returns \"\" for nil", req.Body)
	}
	if *body.Instructions[0].BankInstructionName != "" {
		t.Errorf("bankInstructionName = %q; want empty", *body.Instructions[0].BankInstructionName)
	}
}

// ---------------------------------------------------------------------------
// Internal cash transfers (bulk)
// ---------------------------------------------------------------------------

func TestRESTInternalCashTransfers_TransferBulk(t *testing.T) {
	banking, gw := bankingWithGateway(t)
	note := "monthly sweep"

	results, err := banking.InternalCash().TransferBulk(context.Background(), []InternalCashTransferRequest{
		{
			ClientInstructionID: "1",
			SourceAccountID:     AccountID("U1234567"),
			TargetAccountID:     AccountID("U7654321"),
			Amount:              "500.25",
			Currency:            "USD",
			ClientNote:          &note,
		},
		{
			// ClientNote nil: the generated model marks it omitempty, so the key
			// must be dropped for the second instruction.
			ClientInstructionID: "2",
			SourceAccountID:     AccountID("U1234567"),
			TargetAccountID:     AccountID("U7654321"),
			Amount:              "1",
			Currency:            "GBP",
		},
	})
	if err != nil {
		t.Fatalf("TransferBulk: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %+v; want one", results)
	}
	if results[0].ClientInstructionID != 1 || results[0].InstructionID != 2001 {
		t.Errorf("results[0] ids = %d/%d; want 1/2001", results[0].ClientInstructionID, results[0].InstructionID)
	}
	if results[0].InstructionStatus != "PENDING" {
		t.Errorf("results[0].instructionStatus = %q; want PENDING", results[0].InstructionStatus)
	}

	req := lastRESTRequest(t, gw)
	bankingRequestPosts(t, req, "/gw/api/v1/internal-cash-transfers:bulk")
	var body struct {
		InstructionType string `json:"instructionType"`
		Instructions    []struct {
			Amount              float32 `json:"amount"`
			ClientInstructionID int     `json:"clientInstructionId"`
			ClientNote          *string `json:"clientNote"`
			Currency            string  `json:"currency"`
			SourceAccountID     string  `json:"sourceAccountId"`
			TargetAccountID     string  `json:"targetAccountId"`
		} `json:"instructions"`
	}
	decodeRequestBody(t, req, &body)
	if body.InstructionType != "INTERNAL_CASH_TRANSFER" {
		t.Errorf("instructionType = %q; want INTERNAL_CASH_TRANSFER", body.InstructionType)
	}
	if len(body.Instructions) != 2 {
		t.Fatalf("instructions = %+v; want two", body.Instructions)
	}
	first := body.Instructions[0]
	if first.Amount != 500.25 {
		t.Errorf("instructions[0].amount = %v; want 500.25 as a JSON number", first.Amount)
	}
	if first.ClientInstructionID != 1 {
		t.Errorf("instructions[0].clientInstructionId = %d; want 1", first.ClientInstructionID)
	}
	if first.SourceAccountID != "U1234567" || first.TargetAccountID != "U7654321" {
		t.Errorf("instructions[0] accounts = %q->%q; want U1234567->U7654321", first.SourceAccountID, first.TargetAccountID)
	}
	if first.Currency != "USD" {
		t.Errorf("instructions[0].currency = %q; want USD", first.Currency)
	}
	if first.ClientNote == nil || *first.ClientNote != "monthly sweep" {
		t.Errorf("instructions[0].clientNote = %v; want the caller's note", first.ClientNote)
	}
	if body.Instructions[1].ClientNote != nil {
		t.Errorf("instructions[1].clientNote = %v; want the key dropped for a nil note", *body.Instructions[1].ClientNote)
	}
	if body.Instructions[1].Currency != "GBP" {
		t.Errorf("instructions[1].currency = %q; want GBP", body.Instructions[1].Currency)
	}
}

// ---------------------------------------------------------------------------
// Bank instructions
// ---------------------------------------------------------------------------

func TestRESTBankInstructions_Query(t *testing.T) {
	banking, gw := bankingWithGateway(t)

	res, err := banking.BankInstructions().Query(context.Background(), BankInstructionQueryRequest{
		AccountID:             AccountID("U1234567"),
		BankInstructionMethod: "ACH",
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if res.AccountID != AccountID("U1234567") {
		t.Errorf("accountId = %q; want U1234567", res.AccountID)
	}
	if res.BankInstructionName != "Main Bank" {
		t.Errorf("bankInstructionName = %q; want Main Bank", res.BankInstructionName)
	}
	if res.BankInstructionMethod != "ACH" {
		t.Errorf("bankInstructionMethod = %q; want ACH", res.BankInstructionMethod)
	}
	if res.Status != "ACTIVE" {
		t.Errorf("status = %q; want ACTIVE", res.Status)
	}

	req := lastRESTRequest(t, gw)
	bankingRequestPosts(t, req, "/gw/api/v1/bank-instructions/query")
	var body struct {
		Instruction struct {
			AccountID             string `json:"accountId"`
			BankInstructionMethod string `json:"bankInstructionMethod"`
			ClientInstructionID   int    `json:"clientInstructionId"`
		} `json:"instruction"`
		InstructionType string `json:"instructionType"`
	}
	decodeRequestBody(t, req, &body)
	if body.InstructionType != "QUERY_BANK_INSTRUCTION" {
		t.Errorf("instructionType = %q; want QUERY_BANK_INSTRUCTION", body.InstructionType)
	}
	if body.Instruction.AccountID != "U1234567" {
		t.Errorf("accountId = %q; want U1234567", body.Instruction.AccountID)
	}
	if body.Instruction.BankInstructionMethod != "ACH" {
		t.Errorf("bankInstructionMethod = %q; want ACH", body.Instruction.BankInstructionMethod)
	}
	// BankInstructionQueryRequest has no client instruction ID, so 0 is sent
	// rather than the key being omitted.
	if body.Instruction.ClientInstructionID != 0 {
		t.Errorf("clientInstructionId = %d; want 0 — the request type has no such field", body.Instruction.ClientInstructionID)
	}
}

// TestRESTBankInstructions_Query_MalformedBody pins the decode guard: the
// wrapper unmarshals resp.Body into a plain struct, so a body that is not a JSON
// object must surface a "decode:" error rather than a zero-valued result.
func TestRESTBankInstructions_Query_MalformedBody(t *testing.T) {
	banking, gw := bankingWithGateway(t)
	gw.srv.Fixtures().Set(mockgateway.OpCreateBankInstructionsQuery, mockgateway.Fixture{
		Status: http.StatusCreated,
		Body:   `[]`,
	})

	res, err := banking.BankInstructions().Query(context.Background(), BankInstructionQueryRequest{
		AccountID: AccountID("U1234567"),
	})
	if err == nil {
		t.Fatalf("Query on a JSON array body = %+v, nil error; want a decode error", res)
	}
	if res != nil {
		t.Errorf("result = %+v; want nil alongside the error", res)
	}
	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("err = %v (%T); want *Error", err, err)
	}
	if e.Op != "BankInstructions.Query" {
		t.Errorf("op = %q; want BankInstructions.Query", e.Op)
	}
}

func TestRESTBankInstructions_CreateBulk(t *testing.T) {
	banking, gw := bankingWithGateway(t)

	results, err := banking.BankInstructions().CreateBulk(context.Background(), []BankInstructionCreateRequest{
		{
			AccountID:           AccountID("U1234567"),
			ClientInstructionID: 1012983,
			BankInstructionCode: "ACH_INSTRUCTION",
			BankInstructionName: "Main Bank",
			BankAccountNumber:   "101267576983",
			BankRoutingNumber:   "202012983",
			BankAccountTypeCode: 1,
			BankName:            "JPM Chase",
			Currency:            "USD",
			AchType:             "DEBIT",
		},
		{
			AccountID:           AccountID("U7654321"),
			ClientInstructionID: 2,
			BankInstructionCode: "ACH_INSTRUCTION",
			BankInstructionName: "Savings Bank",
			BankAccountNumber:   "9999",
			BankRoutingNumber:   "1111",
			BankAccountTypeCode: 2,
			BankName:            "Bank Two",
			Currency:            "GBP",
			AchType:             "CREDIT",
		},
	})
	if err != nil {
		t.Fatalf("CreateBulk: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %+v; want one", results)
	}
	if results[0].ClientInstructionID != 1 || results[0].InstructionID != 2001 {
		t.Errorf("results[0] ids = %d/%d; want 1/2001", results[0].ClientInstructionID, results[0].InstructionID)
	}
	if results[0].InstructionStatus != "PENDING" {
		t.Errorf("results[0].instructionStatus = %q; want PENDING", results[0].InstructionStatus)
	}
	if results[0].IbReferenceID == nil || *results[0].IbReferenceID != 9001 {
		t.Errorf("results[0].ibReferenceId = %v; want 9001", results[0].IbReferenceID)
	}
	if results[0].Description == nil || *results[0].Description != "Please poll for status after 10 minutes" {
		t.Errorf("results[0].description = %v; want the fixture's polling guidance", results[0].Description)
	}

	req := lastRESTRequest(t, gw)
	bankingRequestPosts(t, req, "/gw/api/v1/bank-instructions:bulk")
	var body struct {
		InstructionType string                  `json:"instructionType"`
		Instructions    []client.AchInstruction `json:"instructions"`
	}
	decodeRequestBody(t, req, &body)
	if body.InstructionType != "ACH_INSTRUCTION" {
		t.Errorf("instructionType = %q; want ACH_INSTRUCTION", body.InstructionType)
	}
	if len(body.Instructions) != 2 {
		t.Fatalf("instructions = %+v; want two", body.Instructions)
	}
	first := body.Instructions[0]
	if first.AccountId != "U1234567" {
		t.Errorf("instructions[0].accountId = %q; want U1234567", first.AccountId)
	}
	if first.AchType != client.AchInstructionAchType("DEBIT") {
		t.Errorf("instructions[0].achType = %q; want DEBIT", first.AchType)
	}
	if first.BankInstructionCode != client.AchInstructionBankInstructionCode("ACH_INSTRUCTION") {
		t.Errorf("instructions[0].bankInstructionCode = %q; want ACH_INSTRUCTION", first.BankInstructionCode)
	}
	if first.BankInstructionName != "Main Bank" {
		t.Errorf("instructions[0].bankInstructionName = %q; want Main Bank", first.BankInstructionName)
	}
	// ClientInstructionID is a float32 on the public request, truncated to int.
	if first.ClientInstructionId != 1012983 {
		t.Errorf("instructions[0].clientInstructionId = %d; want 1012983", first.ClientInstructionId)
	}
	if first.Currency != "USD" {
		t.Errorf("instructions[0].currency = %q; want USD", first.Currency)
	}
	// The bank account details are nested under clientAccountInfo, so the
	// whole ACH block is asserted rather than just its outer keys.
	if first.ClientAccountInfo.BankAccountNumber != "101267576983" {
		t.Errorf("clientAccountInfo.bankAccountNumber = %q; want 101267576983", first.ClientAccountInfo.BankAccountNumber)
	}
	if first.ClientAccountInfo.BankRoutingNumber != "202012983" {
		t.Errorf("clientAccountInfo.bankRoutingNumber = %q; want 202012983", first.ClientAccountInfo.BankRoutingNumber)
	}
	if first.ClientAccountInfo.BankName != "JPM Chase" {
		t.Errorf("clientAccountInfo.bankName = %q; want JPM Chase", first.ClientAccountInfo.BankName)
	}
	// 1 = Checking, 2 = Savings.
	if first.ClientAccountInfo.BankAccountTypeCode != 1 {
		t.Errorf("clientAccountInfo.bankAccountTypeCode = %d; want 1", int(first.ClientAccountInfo.BankAccountTypeCode))
	}
	second := body.Instructions[1]
	if second.AccountId != "U7654321" || second.Currency != "GBP" {
		t.Errorf("instructions[1] = %q/%q; want U7654321/GBP", second.AccountId, second.Currency)
	}
	if second.AchType != client.AchInstructionAchType("CREDIT") {
		t.Errorf("instructions[1].achType = %q; want CREDIT", second.AchType)
	}
	if second.ClientAccountInfo.BankAccountTypeCode != 2 {
		t.Errorf("instructions[1].clientAccountInfo.bankAccountTypeCode = %d; want 2", int(second.ClientAccountInfo.BankAccountTypeCode))
	}
}

// ---------------------------------------------------------------------------
// Helpers with no production call site
// ---------------------------------------------------------------------------

// TestStrToDecimalPtr covers strToDecimalPtr directly. It has no caller in
// pkg/ibkr, so no e2e path can reach it; both arms are asserted here instead:
// the nil short-circuit and the parse-through.
func TestStrToDecimalPtr(t *testing.T) {
	if got := strToDecimalPtr(nil); got != nil {
		t.Errorf("strToDecimalPtr(nil) = %v; want nil", *got)
	}
	for _, tc := range []struct {
		in   string
		want float32
	}{
		{"12.5", 12.5},
		{"100", 100},
		{"0", 0},
		{"-1.25", -1.25},
		// A non-numeric string parses to zero rather than erroring, matching
		// strToDecimal's discarded error.
		{"not-a-number", 0},
	} {
		got := strToDecimalPtr(&tc.in)
		if got == nil {
			t.Errorf("strToDecimalPtr(%q) = nil; want %v", tc.in, tc.want)
			continue
		}
		if *got != tc.want {
			t.Errorf("strToDecimalPtr(%q) = %v; want %v", tc.in, *got, tc.want)
		}
	}
}

// TestF32PtrToInt64Ptr covers f32PtrToInt64Ptr directly. It has no caller in
// pkg/ibkr, so no e2e path can reach it; both arms are asserted here instead.
func TestF32PtrToInt64Ptr(t *testing.T) {
	if got := f32PtrToInt64Ptr(nil); got != nil {
		t.Errorf("f32PtrToInt64Ptr(nil) = %v; want nil", *got)
	}
	for _, tc := range []struct {
		in   float32
		want int64
	}{
		{42, 42},
		{0, 0},
		{9001, 9001},
		{-3, -3},
		// The conversion truncates toward zero rather than rounding.
		{12.9, 12},
		{-12.9, -12},
	} {
		in := tc.in
		got := f32PtrToInt64Ptr(&in)
		if got == nil {
			t.Errorf("f32PtrToInt64Ptr(%v) = nil; want %d", tc.in, tc.want)
			continue
		}
		if *got != tc.want {
			t.Errorf("f32PtrToInt64Ptr(%v) = %d; want %d", tc.in, *got, tc.want)
		}
	}
}

// TestMakeTradingInstrumentRef covers makeTradingInstrumentRef directly. It has
// no caller in pkg/ibkr, so no e2e path can reach it; the union it builds is
// asserted by its marshalled wire shape, which is the only way to observe it.
//
// The generated TradingInstrumentRef also carries a plain non-omitempty
// Currency string outside the union, so the marshalled object is
// {"conid":N,"currency":""} for any conid: the wrapper cannot set currency, and
// there is no omitempty to drop the key.
func TestMakeTradingInstrumentRef(t *testing.T) {
	ref := makeTradingInstrumentRef(ConID(459200101))
	raw, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got struct {
		ConID    int    `json:"conid"`
		Currency string `json:"currency"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	if got.ConID != 459200101 {
		t.Errorf("conid = %d; want 459200101", got.ConID)
	}
	if got.Currency != "" {
		t.Errorf("currency = %q; want empty — the wrapper sets no currency", got.Currency)
	}
	if keys := sortedKeys(t, raw); len(keys) != 2 || keys[0] != "conid" || keys[1] != "currency" {
		t.Errorf("ref keys = %v; want exactly [conid currency]", keys)
	}

	// A zero conid is still emitted: the union member has no omitempty, so the
	// key is present rather than dropped.
	raw, err = json.Marshal(makeTradingInstrumentRef(ConID(0)))
	if err != nil {
		t.Fatalf("marshal zero: %v", err)
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	if got.ConID != 0 {
		t.Errorf("conid = %d; want 0 for a zero conid", got.ConID)
	}
	if keys := sortedKeys(t, raw); len(keys) != 2 {
		t.Errorf("ref(0) keys = %v; want the conid key present even for a zero conid", keys)
	}
}

// sortedKeys returns the sorted top-level JSON object keys in raw.
func sortedKeys(t *testing.T, raw []byte) []string {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ---------------------------------------------------------------------------
// The >= 400 guard, across every banking wrapper that has one
// ---------------------------------------------------------------------------

// bankingCall names one banking wrapper: the gateway op it reaches, the path it
// must hit, and how to invoke it. hasAck marks the wrappers that read a 202
// acknowledgement and therefore guard on resp.JSON202 being non-nil; the rest
// either return only an error or decode resp.Body directly.
type bankingCall struct {
	fixture string
	path    string
	hasAck  bool
	call    func(*testing.T, *RESTBanking) error
}

// bankingCalls returns one case per banking wrapper, in path order.
func bankingCalls() []bankingCall {
	ctx := context.Background()
	return []bankingCall{
		{
			mockgateway.OpCreateBankInstructions, "/gw/api/v1/bank-instructions", true,
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.BankInstructions().Create(ctx, BankInstructionCreateRequest{
					AccountID: AccountID("U1234567"), ClientInstructionID: 1, AchType: "DEBIT",
				})
				return err
			},
		},
		{
			mockgateway.OpBulkBankInstructions, "/gw/api/v1/bank-instructions:bulk", true,
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.BankInstructions().CreateBulk(ctx, []BankInstructionCreateRequest{{
					AccountID: AccountID("U1234567"), ClientInstructionID: 1, AchType: "DEBIT",
				}})
				return err
			},
		},
		{
			// No acknowledgement: Query decodes resp.Body directly.
			mockgateway.OpCreateBankInstructionsQuery, "/gw/api/v1/bank-instructions/query", false,
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.BankInstructions().Query(ctx, BankInstructionQueryRequest{
					AccountID: AccountID("U1234567"), BankInstructionMethod: "ACH",
				})
				return err
			},
		},
		{
			// No acknowledgement: decodes resp.Body into clientInstructionRaw.
			mockgateway.OpGetClientInstructions, "/gw/api/v1/client-instructions/1", false,
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.ClientInstruction(ctx, 1)
				return err
			},
		},
		{
			mockgateway.OpCreateExternalAssetTransfers, "/gw/api/v1/external-asset-transfers", true,
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.ExternalTransfers().Transfer(ctx, AssetTransferRequest{
					AccountID: AccountID("U1234567"), ClientInstructionID: "1", Direction: "IN",
				})
				return err
			},
		},
		{
			mockgateway.OpBulkExternalAssetTransfers, "/gw/api/v1/external-asset-transfers:bulk", true,
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.ExternalTransfers().TransferBulk(ctx, []AssetTransferRequest{{
					AccountID: AccountID("U1234567"), ClientInstructionID: "1", Direction: "IN",
				}})
				return err
			},
		},
		{
			mockgateway.OpCreateExternalAssetTransfers2, "/gw/api/v2/external-asset-transfers", true,
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.ExternalTransfers().TransferV2(ctx, AssetTransferRequest{
					AccountID: AccountID("U1234567"), ClientInstructionID: "1", Direction: "IN",
				})
				return err
			},
		},
		{
			mockgateway.OpBulkExternalAssetTransfers2, "/gw/api/v2/external-asset-transfers:bulk", true,
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.ExternalTransfers().TransferBulkV2(ctx, []AssetTransferRequest{{
					AccountID: AccountID("U1234567"), ClientInstructionID: "1", Direction: "IN",
				}})
				return err
			},
		},
		{
			mockgateway.OpCreateExternalCashTransfers, "/gw/api/v1/external-cash-transfers", true,
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.CashTransfers().Transfer(ctx, CashTransferRequest{
					AccountID: AccountID("U1234567"), ClientInstructionID: "1", Currency: "USD",
				}, true)
				return err
			},
		},
		{
			mockgateway.OpBulkExternalCashTransfers, "/gw/api/v1/external-cash-transfers:bulk", true,
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.CashTransfers().TransferBulk(ctx, []CashTransferRequest{{
					AccountID: AccountID("U1234567"), ClientInstructionID: "1", Currency: "USD",
				}}, true)
				return err
			},
		},
		{
			// No acknowledgement: QueryBalances decodes resp.Body directly.
			mockgateway.OpCreateExternalCashTransfersQuery, "/gw/api/v1/external-cash-transfers/query", false,
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.CashTransfers().QueryBalances(ctx, CashTransferRequest{
					AccountID: AccountID("U1234567"), Currency: "USD",
				})
				return err
			},
		},
		{
			// No acknowledgement: decodes resp.Body into instructionSetRaw.
			mockgateway.OpGetInstructionSets, "/gw/api/v1/instruction-sets/1", false,
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.InstructionSet(ctx, 1)
				return err
			},
		},
		{
			// No acknowledgement: the cancel wrappers return only an error.
			mockgateway.OpCreateInstructionsCancel, "/gw/api/v1/instructions/cancel", false,
			func(t *testing.T, b *RESTBanking) error {
				return b.CancelInstruction(ctx, CancelInstructionRequest{InstructionID: 1, Reason: "r"})
			},
		},
		{
			mockgateway.OpBulkInstructionsCancel, "/gw/api/v1/instructions/cancel:bulk", false,
			func(t *testing.T, b *RESTBanking) error {
				return b.CancelInstructionsBulk(ctx, []CancelInstructionRequest{{InstructionID: 1}})
			},
		},
		{
			// No acknowledgement: decodes resp.Body into transactionsRaw.
			mockgateway.OpCreateInstructionsQuery, "/gw/api/v1/instructions/query", false,
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.QueryTransactions(ctx, TransactionQueryRequest{
					AccountID: AccountID("U1234567"), Days: 5,
				})
				return err
			},
		},
		{
			// No acknowledgement: decodes resp.Body into instructionRaw.
			mockgateway.OpGetInstructions, "/gw/api/v1/instructions/1", false,
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.Instruction(ctx, 1)
				return err
			},
		},
		{
			mockgateway.OpCreateInternalAssetTransfers, "/gw/api/v1/internal-asset-transfers", true,
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.InternalTransfers().Transfer(ctx, InternalAssetTransferRequest{
					ClientInstructionID: "1", SourceAccountID: AccountID("U1234567"),
					TargetAccountID: AccountID("U7654321"), TransferQuantity: "1",
				})
				return err
			},
		},
		{
			mockgateway.OpBulkInternalAssetTransfers, "/gw/api/v1/internal-asset-transfers:bulk", true,
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.InternalTransfers().TransferBulk(ctx, []InternalAssetTransferRequest{{
					ClientInstructionID: "1", SourceAccountID: AccountID("U1234567"),
					TargetAccountID: AccountID("U7654321"), TransferQuantity: "1",
				}})
				return err
			},
		},
		{
			mockgateway.OpCreateInternalCashTransfers, "/gw/api/v1/internal-cash-transfers", true,
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.InternalCash().Transfer(ctx, InternalCashTransferRequest{
					ClientInstructionID: "1", SourceAccountID: AccountID("U1234567"),
					TargetAccountID: AccountID("U7654321"), Amount: "1", Currency: "USD",
				})
				return err
			},
		},
		{
			mockgateway.OpBulkInternalCashTransfers, "/gw/api/v1/internal-cash-transfers:bulk", true,
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.InternalCash().TransferBulk(ctx, []InternalCashTransferRequest{{
					ClientInstructionID: "1", SourceAccountID: AccountID("U1234567"),
					TargetAccountID: AccountID("U7654321"), Amount: "1", Currency: "USD",
				}})
				return err
			},
		},
	}
}

// TestRESTBanking_ServerError covers the >= 400 guard on all twenty banking
// wrappers at once. internal.ResponseError returns nil only for a status below
// 400 and isErrorStatus is `>= 400`, so the guard can never hand a caller
// (nil, nil); every case must come back as a typed *Error carrying the status,
// which is what the errors.As(err, &e); e.HTTPStatus idiom in docs/ERRORS.md
// depends on. The recorded path is asserted too, so a case cannot pass by
// reaching some other endpoint.
func TestRESTBanking_ServerError(t *testing.T) {
	for _, tc := range bankingCalls() {
		t.Run(tc.path, func(t *testing.T) {
			banking, gw := bankingWithGateway(t)
			gw.srv.Fixtures().Set(tc.fixture, mockgateway.Fixture{
				Status: http.StatusInternalServerError,
				Body:   `{"error":"boom","message":"synthetic failure"}`,
			})

			err := tc.call(t, banking)
			if err == nil {
				t.Fatal("call on 500 = nil error; want an error")
			}
			var e *Error
			if !errors.As(err, &e) {
				t.Fatalf("err = %v (%T); want *Error", err, err)
			}
			// Op must be the wrapper's own, not the transport's "unknown"
			// placeholder that ResponseError starts from.
			if e.Op == "" || e.Op == "unknown" {
				t.Errorf("op = %q; want the wrapper's op", e.Op)
			}
			if e.HTTPStatus != http.StatusInternalServerError {
				t.Errorf("httpStatus = %d; want 500", e.HTTPStatus)
			}
			if req := lastRESTRequest(t, gw); req.Path != tc.path {
				t.Errorf("path = %q; want %q", req.Path, tc.path)
			}
		})
	}
}

// TestRESTBanking_NilAcknowledgement covers the `resp.JSON202 == nil` branch on
// every wrapper that reads a 202 acknowledgement (the twelve marked hasAck). A
// 200 with a well-formed JSON body parses without error but leaves the declared
// 202 variant nil, so the guard is the only thing standing between a caller and
// a fabricated instruction-set ID. Every wrapper must report the same explicit
// message rather than a zero-valued success.
func TestRESTBanking_NilAcknowledgement(t *testing.T) {
	cases := 0
	for _, tc := range bankingCalls() {
		if !tc.hasAck {
			continue
		}
		cases++
		t.Run(tc.path, func(t *testing.T) {
			banking, gw := bankingWithGateway(t)
			gw.srv.Fixtures().Set(tc.fixture, mockgateway.Fixture{
				Status: http.StatusOK,
				Body:   `{"instructionSetId":1001,"status":202}`,
			})

			err := tc.call(t, banking)
			if err == nil {
				t.Fatal("call on 200 with no 202 variant = nil error; want an explicit error")
			}
			var e *Error
			if !errors.As(err, &e) {
				t.Fatalf("err = %v (%T); want *Error", err, err)
			}
			if e.Message != "unexpected nil 202 response" {
				t.Errorf("message = %q; want \"unexpected nil 202 response\"", e.Message)
			}
			// The guard builds the error itself, so there is no status to read:
			// a caller must not be able to mistake this for an HTTP failure.
			if e.HTTPStatus != 0 {
				t.Errorf("httpStatus = %d; want 0 — the guard does not set one", e.HTTPStatus)
			}
			if req := lastRESTRequest(t, gw); req.Path != tc.path {
				t.Errorf("path = %q; want %q", req.Path, tc.path)
			}
		})
	}
	// Guards the table itself: if a case is silently dropped from bankingCalls,
	// the count stops matching and the test says so instead of quietly covering
	// less than it claims.
	if cases != 12 {
		t.Errorf("ran %d nil-acknowledgement cases; want 12", cases)
	}
}

// TestRESTBanking_InstructionSetIDAboveFloat32Mantissa pins the exact rendering
// of a 202 acknowledgement's instructionSetId on the three wrappers that hand it
// back as a string. The generated model types the field as an int, and an int
// routed through a 32-bit float keeps only 24 bits of mantissa, so anything
// above 2^24 (16,777,216) comes back truncated and a caller that polls or
// reconciles by that string addresses the wrong instruction — silent data
// corruption on a banking path. 1988905739 is the magnitude IBKR's own spec
// documents as this field's example, so it is a value the gateway can
// plausibly return rather than a contrived one.
func TestRESTBanking_InstructionSetIDAboveFloat32Mantissa(t *testing.T) {
	const want = "1988905739"
	cases := []struct {
		name    string
		fixture string
		path    string
		call    func(*testing.T, *RESTBanking) (string, error)
	}{
		{
			"ExternalAssetTransfers.Transfer", mockgateway.OpCreateExternalAssetTransfers,
			"/gw/api/v1/external-asset-transfers",
			func(t *testing.T, b *RESTBanking) (string, error) {
				return b.ExternalTransfers().Transfer(context.Background(), AssetTransferRequest{
					AccountID: AccountID("U1234567"), ClientInstructionID: "1", Direction: "IN",
				})
			},
		},
		{
			"InternalAssetTransfers.Transfer", mockgateway.OpCreateInternalAssetTransfers,
			"/gw/api/v1/internal-asset-transfers",
			func(t *testing.T, b *RESTBanking) (string, error) {
				return b.InternalTransfers().Transfer(context.Background(), InternalAssetTransferRequest{
					ClientInstructionID: "1", SourceAccountID: AccountID("U1234567"),
					TargetAccountID: AccountID("U7654321"), TransferQuantity: "1",
				})
			},
		},
		{
			"ExternalCashTransfers.Transfer", mockgateway.OpCreateExternalCashTransfers,
			"/gw/api/v1/external-cash-transfers",
			func(t *testing.T, b *RESTBanking) (string, error) {
				return b.CashTransfers().Transfer(context.Background(), CashTransferRequest{
					AccountID: AccountID("U1234567"), ClientInstructionID: "1", Currency: "USD",
				}, true)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			banking, gw := bankingWithGateway(t)
			gw.srv.Fixtures().Set(tc.fixture, mockgateway.Fixture{
				Status: http.StatusAccepted,
				Body:   `{"instructionSetId":1988905739,"status":202}`,
			})

			id, err := tc.call(t, banking)
			if err != nil {
				t.Fatalf("call: %v", err)
			}
			// Compared as a whole string, not parsed back to a number: the caller
			// receives the rendered form, so the rendered form is the contract.
			if id != want {
				t.Errorf("instructionSetId = %q; want %q exactly — the field is an int and must not be rendered through a 32-bit float, which keeps only 24 mantissa bits and truncates above 2^24", id, want)
			}
			if req := lastRESTRequest(t, gw); req.Path != tc.path {
				t.Errorf("path = %q; want %q", req.Path, tc.path)
			}
		})
	}
}

// TestRESTBanking_DecodeError covers the wrapper's own `json.Unmarshal` failure
// branch on the three banking wrappers whose success status decodes into a
// oneOf union (so the generated parser accepts any JSON and the wrapper is the
// first thing to reject a bad body). A 200 whose body is a JSON array where an
// object is expected must surface a "decode:" error rather than a zero-valued
// result that looks like a real answer.
func TestRESTBanking_DecodeError(t *testing.T) {
	cases := []struct {
		fixture string
		path    string
		op      string
		call    func(*testing.T, *RESTBanking) error
	}{
		{
			mockgateway.OpGetClientInstructions, "/gw/api/v1/client-instructions/1",
			"Banking.ClientInstruction",
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.ClientInstruction(context.Background(), 1)
				return err
			},
		},
		{
			mockgateway.OpGetInstructions, "/gw/api/v1/instructions/1",
			"Banking.Instruction",
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.Instruction(context.Background(), 1)
				return err
			},
		},
		{
			mockgateway.OpCreateExternalCashTransfersQuery, "/gw/api/v1/external-cash-transfers/query",
			"ExternalCashTransfers.QueryBalances",
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.CashTransfers().QueryBalances(context.Background(), CashTransferRequest{
					AccountID: AccountID("U1234567"), Currency: "USD",
				})
				return err
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			banking, gw := bankingWithGateway(t)
			// A 200 so the >= 400 guard passes, and the body is what the wrapper
			// goes on to decode itself.
			gw.srv.Fixtures().Set(tc.fixture, mockgateway.Fixture{
				Status: http.StatusOK,
				Body:   `[]`,
			})

			err := tc.call(t, banking)
			if err == nil {
				t.Fatal("call on a JSON array body = nil error; want a decode error")
			}
			var e *Error
			if !errors.As(err, &e) {
				t.Fatalf("err = %v (%T); want *Error", err, err)
			}
			if e.Op != tc.op {
				t.Errorf("op = %q; want %s", e.Op, tc.op)
			}
			// The "decode: " prefix is the wrapper's own marker, so it proves the
			// failure came from the wrapper's unmarshal and not from the transport.
			if !strings.HasPrefix(e.Message, "decode: ") {
				t.Errorf("message = %q; want the \"decode: \" prefix", e.Message)
			}
			// The decoding error is retained, so errors.Unwrap reaches it.
			if e.Err == nil {
				t.Error("Err = nil; want the json error retained")
			}
			// A decode failure is not an HTTP failure.
			if e.HTTPStatus != 0 {
				t.Errorf("httpStatus = %d; want 0 — a 200 body failed to decode", e.HTTPStatus)
			}
			if req := lastRESTRequest(t, gw); req.Path != tc.path {
				t.Errorf("path = %q; want %q", req.Path, tc.path)
			}
		})
	}
}

// TestRESTBanking_ResponseParseError covers the `wrapOp(op, err)` branch on the
// banking wrappers whose success status decodes into a plain generated struct
// rather than a oneOf union. There the generated Parse*Response rejects the body
// first and hands the error back through *WithResponse, so the wrapper never
// runs its own unmarshal. The contract is the same typed error either way, but
// the message comes from the generated parser and carries no "decode: " prefix —
// asserted so the two failure modes stay distinguishable to a caller.
func TestRESTBanking_ResponseParseError(t *testing.T) {
	cases := []struct {
		fixture string
		path    string
		op      string
		status  int
		call    func(*testing.T, *RESTBanking) error
	}{
		{
			// GetInstructionSets' 200 decodes into BulkMultiStatusResponse.
			mockgateway.OpGetInstructionSets, "/gw/api/v1/instruction-sets/1",
			"Banking.InstructionSet", http.StatusOK,
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.InstructionSet(context.Background(), 1)
				return err
			},
		},
		{
			// CreateInstructionsQuery's 202 decodes into InstructionResponse.
			mockgateway.OpCreateInstructionsQuery, "/gw/api/v1/instructions/query",
			"Banking.QueryTransactions", http.StatusAccepted,
			func(t *testing.T, b *RESTBanking) error {
				_, err := b.QueryTransactions(context.Background(), TransactionQueryRequest{
					AccountID: AccountID("U1234567"), Days: 5,
				})
				return err
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			banking, gw := bankingWithGateway(t)
			gw.srv.Fixtures().Set(tc.fixture, mockgateway.Fixture{Status: tc.status, Body: `[]`})

			err := tc.call(t, banking)
			if err == nil {
				t.Fatal("call on a JSON array body = nil error; want a parse error")
			}
			var e *Error
			if !errors.As(err, &e) {
				t.Fatalf("err = %v (%T); want *Error", err, err)
			}
			// wrapOp tags the error with the wrapper's op, which is what makes
			// errors.As(err, &e); e.Op usable to attribute the failure.
			if e.Op != tc.op {
				t.Errorf("op = %q; want %s", e.Op, tc.op)
			}
			if strings.HasPrefix(e.Message, "decode: ") {
				t.Errorf("message = %q; want the raw generated-parser message, not the wrapper's decode marker", e.Message)
			}
			if e.Err == nil {
				t.Error("Err = nil; want the json error retained for errors.Unwrap")
			}
			if req := lastRESTRequest(t, gw); req.Path != tc.path {
				t.Errorf("path = %q; want %q", req.Path, tc.path)
			}
		})
	}
}

// TestRESTBanking_ClientClosed pins the checkOpen guard for the sub-managers
// this file exercises. It is the only branch that fires before any request is
// built, so a wrapper that skipped it would try to use a closed client instead
// of returning internal.ErrClosed.
func TestRESTBanking_ClientClosed(t *testing.T) {
	cli, _ := newRESTClientWithGateway(t)
	if err := cli.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	surface, err := cli.REST()
	if err != nil {
		t.Fatalf("REST on a closed client: %v", err)
	}
	banking := surface.Banking()
	ctx := context.Background()

	cases := []struct {
		name string
		op   string
		call func() error
	}{
		{"ClientInstruction", "Banking.ClientInstruction", func() error {
			_, err := banking.ClientInstruction(ctx, 1)
			return err
		}},
		{"InstructionSet", "Banking.InstructionSet", func() error {
			_, err := banking.InstructionSet(ctx, 1)
			return err
		}},
		{"Instruction", "Banking.Instruction", func() error {
			_, err := banking.Instruction(ctx, 1)
			return err
		}},
		{"QueryTransactions", "Banking.QueryTransactions", func() error {
			_, err := banking.QueryTransactions(ctx, TransactionQueryRequest{Days: 5})
			return err
		}},
		{"CancelInstruction", "Banking.CancelInstruction", func() error {
			return banking.CancelInstruction(ctx, CancelInstructionRequest{InstructionID: 1})
		}},
		{"CancelInstructionsBulk", "Banking.CancelInstructionsBulk", func() error {
			return banking.CancelInstructionsBulk(ctx, []CancelInstructionRequest{{InstructionID: 1}})
		}},
		{"TransferV2", "ExternalAssetTransfers.TransferV2", func() error {
			_, err := banking.ExternalTransfers().TransferV2(ctx, AssetTransferRequest{})
			return err
		}},
		{"TransferBulkV2", "ExternalAssetTransfers.TransferBulkV2", func() error {
			_, err := banking.ExternalTransfers().TransferBulkV2(ctx, []AssetTransferRequest{{}})
			return err
		}},
		{"InternalTransferBulk", "InternalAssetTransfers.TransferBulk", func() error {
			_, err := banking.InternalTransfers().TransferBulk(ctx, []InternalAssetTransferRequest{{}})
			return err
		}},
		{"CashTransferBulk", "ExternalCashTransfers.TransferBulk", func() error {
			_, err := banking.CashTransfers().TransferBulk(ctx, []CashTransferRequest{{}}, true)
			return err
		}},
		{"InternalCashTransferBulk", "InternalCashTransfers.TransferBulk", func() error {
			_, err := banking.InternalCash().TransferBulk(ctx, []InternalCashTransferRequest{{}})
			return err
		}},
		{"BankInstructionsQuery", "BankInstructions.Query", func() error {
			_, err := banking.BankInstructions().Query(ctx, BankInstructionQueryRequest{})
			return err
		}},
		{"BankInstructionsCreateBulk", "BankInstructions.CreateBulk", func() error {
			_, err := banking.BankInstructions().CreateBulk(ctx, []BankInstructionCreateRequest{{}})
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if err == nil {
				t.Fatal("call on a closed client = nil error; want internal.ErrClosed")
			}
			// The guard returns internal.ErrClosed unwrapped, so errors.Is is the
			// right assertion here rather than errors.As on *Error.
			if !errors.Is(err, internal.ErrClosed) {
				t.Errorf("err = %v; want internal.ErrClosed", err)
			}
		})
	}
}
