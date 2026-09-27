// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// orders_red_test.go proves each check in orders_confirmation.go is
// load-bearing, by the same mutation method red_test.go uses: copy the real
// repository into the shared scratch tree, break exactly one documented or coded
// fact, and require the message that names it.
//
// Two controls mirror the ones in red_test.go and are the reason a case here means
// anything. TestOrdersChecksPassOnUnmutatedTree stops a checker that rejects every
// tree from satisfying every red case, and ordersGreenCases() holds edits that are
// *true statements about the code* — a renamed parameter, a pointer in the
// generated model, a json tag on a type nobody submits — so a checker weakened
// into matching names or tags fails them. The last green case is the one that
// matters most: it is the disagreement this checker is documented not to resolve,
// and a check that enforced it would be making a decision this program is not
// authorised to make.
package main

import (
	"fmt"
	"strings"
	"testing"
)

const (
	odoc   = ordersDoc
	ogen   = genClientPath
	oclint = "pkg/ibkr/client.go"
)

// TestOrdersChecksAreLoadBearing mutates one documented or coded fact per case and
// requires the message that names it.
func TestOrdersChecksAreLoadBearing(t *testing.T) {
	for _, tc := range ordersRedCases() {
		t.Run(tc.name, func(t *testing.T) {
			for _, e := range tc.edits {
				mutate(t, e.rel, e.old, e.repl)
			}
			err := checkOrdersAndConfirmation(scratchRoot)
			if err == nil {
				t.Fatalf("checkOrdersAndConfirmation passed; expected a failure containing %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("checkOrdersAndConfirmation failed for the wrong reason.\nwant substring: %q\ngot: %v", tc.want, err)
			}
		})
	}
}

// TestOrdersChecksPassOnUnmutatedTree is the control for every red case above.
func TestOrdersChecksPassOnUnmutatedTree(t *testing.T) {
	if err := checkOrdersAndConfirmation(scratchRoot); err != nil {
		t.Fatalf("checkOrdersAndConfirmation on the unmutated tree: %v", err)
	}
}

// TestOrdersChecksPassOnGreenCases covers edits that are true statements about the
// code, which the checker has to accept.
func TestOrdersChecksPassOnGreenCases(t *testing.T) {
	for _, tc := range ordersGreenCases() {
		t.Run(tc.name, func(t *testing.T) {
			for _, e := range tc.edits {
				mutate(t, e.rel, e.old, e.repl)
			}
			if err := checkOrdersAndConfirmation(scratchRoot); err != nil {
				t.Fatalf("checkOrdersAndConfirmation rejected a true statement about the code: %v", err)
			}
		})
	}
}

// TestOrdersAggregatorReportsEveryCheck exists because a check that is written,
// reviewed and never wired into the aggregator is not a check. It breaks two
// independent facts in two different checks and requires both messages from one
// run, which is what proves each of those two is actually called.
func TestOrdersAggregatorReportsEveryCheck(t *testing.T) {
	mutate(t, odoc, "`POST /v1/api/iserver/account/{accountId}/orders`", "`POST /v1/api/iserver/account/{accountId}/tickets`")
	mutate(t, tradeSrcPath, "Quantity      string    `json:\"quantity\"`", "Quantity      float32   `json:\"quantity\"`")
	err := checkOrdersAndConfirmation(scratchRoot)
	if err == nil {
		t.Fatal("checkOrdersAndConfirmation passed; expected both the endpoint and the money-precision failure")
	}
	for _, want := range []string{"order flow endpoint drift", "money precision drift"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("the aggregator did not report %q.\ngot: %v", want, err)
		}
	}
}

// TestPublicAPICheckRunsBothDirections is the negative control for the two
// directions checkDocumentedStruct now runs. Each case breaks only the
// code-to-document direction — a field the declaration has and the document does
// not — and requires the real check to fail. The control is checkDocumentedSubset,
// which is the document-to-code direction alone: exactly what checkPublicOrderAPI
// used to do, and the reason ParentID and IsSingleGroup went undocumented for as
// long as they did. If the subset comparison could see these cases they would
// prove nothing about the new direction.
//
// TestPublicAPICheckSubsetControlIsNotInert is the other half: the control has to
// fail on a document-side defect too, or "the subset check passed" above would be
// satisfied by a function that never fails anything.
func TestPublicAPICheckRunsBothDirections(t *testing.T) {
	for _, tc := range []struct {
		name string
		typ  string
		edit edit
	}{
		{"orderrequest/doc-drops-parentid", "OrderRequest",
			replace(odoc, "    ParentID    string      // parent order id, for grouping\n", "")},
		{"orderrequest/doc-drops-issinglegroup", "OrderRequest",
			replace(odoc, "    IsSingleGroup bool      // quantity is per-conid rather than a total\n", "")},
		{"orderrequest/code-adds-an-undocumented-field", "OrderRequest",
			replace(tradeSrcPath, "\tIsSingleGroup bool\n}", "\tIsSingleGroup bool\n\tSettlementCycle string\n}")},
		{"reply/doc-drops-messageids", "Reply",
			replace(odoc, "    MessageIDs  []string // additional message ids\n", "")},
		{"reply/code-adds-an-undocumented-field", "Reply",
			replace(tradeSrcPath, "\tMessageIDs []string\n}", "\tMessageIDs []string\n\tSeverity int\n}")},
		{"reply/code-embeds-a-type", "Reply",
			replace(tradeSrcPath, "type Reply struct {\n", "type Reply struct {\n\tfmt.Stringer\n")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mutate(t, tc.edit.rel, tc.edit.old, tc.edit.repl)
			doc, err := parseDoc(scratchRoot, ordersDoc)
			if err != nil {
				t.Fatalf("parseDoc: %v", err)
			}
			code, err := parseGo(scratchRoot, tradeSrcPath)
			if err != nil {
				t.Fatalf("parseGo: %v", err)
			}
			if err := checkDocumentedStruct(doc, code, tc.typ, matchSet); err == nil {
				t.Fatalf("checkDocumentedStruct passed on a tree where %s has a field the document omits", tc.typ)
			}
			if err := checkDocumentedSubset(doc, code, tc.typ); err != nil {
				t.Fatalf("the weakened subset check also failed, so this case does not isolate the "+
					"code-to-document direction: %v", err)
			}
		})
	}
}

func TestPublicAPICheckSubsetControlIsNotInert(t *testing.T) {
	mutate(t, odoc, "    Quantity    string      // never float64", "    Quantity    float64     // never float64")
	doc, err := parseDoc(scratchRoot, ordersDoc)
	if err != nil {
		t.Fatalf("parseDoc: %v", err)
	}
	code, err := parseGo(scratchRoot, tradeSrcPath)
	if err != nil {
		t.Fatalf("parseGo: %v", err)
	}
	if err := checkDocumentedSubset(doc, code, "OrderRequest"); err == nil {
		t.Fatal("checkDocumentedSubset passed on a document that retypes a field; the control is inert")
	}
}

// checkDocumentedSubset is the weaker comparison the control above needs: every
// field the document lists must be declared with the documented type, and nothing
// whatsoever is said about the fields the code declares. It is the direction
// checkPublicOrderAPI used to run, kept here so a test can show the difference
// between the two; no check in this program calls it.
func checkDocumentedSubset(doc *docFile, code *goSrc, name string) error {
	docFields, err := doc.structBlockFields(name)
	if err != nil {
		return err
	}
	realFields, err := code.structFields(name)
	if err != nil {
		return err
	}
	byName := map[string]codeField{}
	for _, rf := range realFields {
		if rf.name != "" {
			byName[rf.name] = rf
		}
	}
	for _, df := range docFields {
		rf, ok := byName[df.name]
		if !ok {
			return fmt.Errorf("%s documents %s field %q, which is not declared", doc.rel, name, df.name)
		}
		if df.typ != rf.typ {
			return fmt.Errorf("%s documents %s field %s as %s but it is declared %s", doc.rel, name, df.name, df.typ, rf.typ)
		}
	}
	return nil
}

func ordersRedCases() []redCase {
	return []redCase{
		// --- claim 1: the endpoints the flow list names ----------------------
		{
			name:  "endpoint/doc-submit-path",
			edits: []edit{replace(odoc, "`POST /v1/api/iserver/account/{accountId}/orders`", "`POST /v1/api/iserver/account/{accountId}/ticket`")},
			want:  "order flow endpoint drift",
		},
		{
			name:  "endpoint/doc-submit-verb",
			edits: []edit{replace(odoc, "`POST /v1/api/iserver/account/{accountId}/orders`", "`PUT /v1/api/iserver/account/{accountId}/orders`")},
			want:  "order flow endpoint drift",
		},
		{
			name:  "endpoint/doc-confirm-path",
			edits: []edit{replace(odoc, "`POST /v1/api/iserver/reply/{replyId}` with the reply id set.", "`POST /v1/api/iserver/replies/{replyId}` with the reply id set.")},
			want:  "order flow endpoint drift",
		},
		{
			name:  "endpoint/doc-status-verb",
			edits: []edit{replace(odoc, "`GET /v1/api/iserver/account/order/status/{orderId}`", "`POST /v1/api/iserver/account/order/status/{orderId}`")},
			want:  "order flow endpoint drift",
		},
		{
			name:  "endpoint/doc-status-path",
			edits: []edit{replace(odoc, "`GET /v1/api/iserver/account/order/status/{orderId}`", "`GET /v1/api/iserver/account/order/{orderId}/status`")},
			want:  "order flow endpoint drift",
		},
		{
			name:  "endpoint/doc-modify-verb",
			edits: []edit{replace(odoc, "`POST .../order/{orderId}` / `DELETE .../order/{orderId}`", "`PATCH .../order/{orderId}` / `DELETE .../order/{orderId}`")},
			want:  "order flow endpoint drift",
		},
		{
			name:  "endpoint/doc-cancel-verb",
			edits: []edit{replace(odoc, "`POST .../order/{orderId}` / `DELETE .../order/{orderId}`", "`POST .../order/{orderId}` / `POST .../order/{orderId}`")},
			want:  "order flow endpoint drift",
		},
		{
			name:  "endpoint/doc-modify-tail",
			edits: []edit{replace(odoc, "`POST .../order/{orderId}` / `DELETE .../order/{orderId}`", "`POST .../orders/{orderId}` / `DELETE .../order/{orderId}`")},
			want:  "order flow endpoint drift",
		},
		{
			name:  "endpoint/code-confirm-path",
			edits: []edit{replace(ogen, `operationPath := fmt.Sprintf("/v1/api/iserver/reply/%s", pathParam0)`, `operationPath := fmt.Sprintf("/v1/api/iserver/replies/%s", pathParam0)`)},
			want:  "order flow endpoint drift",
		},
		{
			// The step names a method; rewiring that method to another operation's
			// endpoint is the failure a name-only check would miss.
			name: "endpoint/submit-rewired-to-modify",
			edits: []edit{replace(tradeSrcPath,
				"m.client.generated.SubmitNewOrderWithBody(ctx, string(account),",
				"m.client.generated.ModifyOpenOrderWithBody(ctx, string(account),")},
			want: "order flow endpoint drift",
		},
		{
			name:  "endpoint/doc-step-loses-its-endpoint",
			edits: []edit{replace(odoc, "`POST /v1/api/iserver/reply/{replyId}` with the reply id set.", "with the reply id set.")},
			want:  "states 0 HTTP endpoint(s)",
		},
		{
			name:  "endpoint/doc-flow-list-emptied",
			edits: []edit{replace(odoc, "1. **Submit** — `POST /v1/api/iserver/account/{accountId}/orders`.", "1. Places the order with the gateway.")},
			want:  `the flow list has no step titled "Submit"`,
		},

		// --- claim 3: the wire body is built by hand -------------------------
		{
			name:  "wire/quantity-becomes-a-number",
			edits: []edit{replace(tradeSrcPath, "Quantity      string    `json:\"quantity\"`", "Quantity      float32   `json:\"quantity\"`")},
			want:  "money precision drift",
		},
		{
			name:  "wire/price-becomes-a-number",
			edits: []edit{replace(tradeSrcPath, "Price         string    `json:\"price,omitempty\"`", "Price         float32   `json:\"price,omitempty\"`")},
			want:  "money precision drift",
		},
		{
			name:  "wire/auxprice-becomes-a-number",
			edits: []edit{replace(tradeSrcPath, "AuxPrice      string    `json:\"auxPrice,omitempty\"`", "AuxPrice      float32   `json:\"auxPrice,omitempty\"`")},
			want:  "money precision drift",
		},
		{
			name:  "wire/quantity-tag-renamed",
			edits: []edit{replace(tradeSrcPath, "Quantity      string    `json:\"quantity\"`", "Quantity      string    `json:\"qty\"`")},
			want:  "money precision drift",
		},
		{
			// The value crosses to the wire untouched; a conversion on the way is
			// the defect class that truncated an acknowledgement id, and it would
			// not change any type this checker can see.
			name:  "wire/tojson-converts-quantity",
			edits: []edit{replace(tradeSrcPath, "Quantity:      r.Quantity,", "Quantity:      strings.TrimSpace(r.Quantity),")},
			want:  "not a copy of OrderRequest.Quantity",
		},
		{
			name:  "wire/tojson-crosses-two-fields",
			edits: []edit{replace(tradeSrcPath, "Price:         r.LimitPrice,", "Price:         r.StopPrice,")},
			want:  "money precision drift",
		},
		{
			name:  "wire/tojson-drops-the-price",
			edits: []edit{replace(tradeSrcPath, "\t\tPrice:         r.LimitPrice,\n", "")},
			want:  "money precision drift",
		},
		{
			name: "wire/submit-marshals-the-generated-model",
			edits: []edit{replace(tradeSrcPath,
				"}\n\n\tbody, err := json.Marshal(ordersSubmissionJSON{Orders: []orderTicketJSON{req.toJSON()}})",
				"}\n\n\tbody, err := json.Marshal(client.OrdersSubmissionRequest{})")},
			want: "marshals a client.OrdersSubmissionRequest literal",
		},
		{
			name:  "wire/envelope-drops-the-hand-built-ticket",
			edits: []edit{replace(tradeSrcPath, "Orders []orderTicketJSON `json:\"orders\"`", "Orders []client.SingleOrderSubmissionRequest `json:\"orders\"`")},
			want:  "declares orders as \"[]client.SingleOrderSubmissionRequest\"",
		},
		{
			name: "wire/submit-uses-the-typed-body-variant",
			edits: []edit{replace(tradeSrcPath,
				"m.client.generated.SubmitNewOrderWithBody(ctx, string(account),",
				"m.client.generated.SubmitNewOrder(ctx, string(account),")},
			want: "the typed-body variant SubmitNewOrder",
		},
		{
			// The document's justification is that the generated type is float32.
			// If that stops being true the sentence is no longer a reason to build
			// the body by hand, and nothing else would notice.
			name: "wire/generated-type-stops-using-float32",
			edits: []edit{replace(ogen,
				"\t// Quantity Quantity of the order ticket in units of the instrument.\n\tQuantity float32 `json:\"quantity\"`",
				"\t// Quantity Quantity of the order ticket in units of the instrument.\n\tQuantity string `json:\"quantity\"`")},
			want: `declares "quantity" as string`,
		},
		{
			// The alias is what says which declaration the operation takes. A
			// re-pointed alias must be reported, not read past.
			name: "wire/generated-alias-repointed",
			edits: []edit{replace(ogen,
				"type SubmitNewOrderJSONRequestBody = OrdersSubmissionRequest",
				"type SubmitNewOrderJSONRequestBody = OrderSubmitSuccess")},
			want: "which is not a named type",
		},
		{
			name:  "wire/doc-justification-reworded",
			edits: []edit{replace(odoc, "built by hand (`orderTicketJSON`) with money/quantity as", "built by hand (`orderTicketJSON`) with money and quantity as")},
			want:  "is gone or reworded",
		},

		// --- claim 2: the public API the document publishes ------------------
		{
			name:  "api/doc-invents-a-field",
			edits: []edit{replace(odoc, "    AllOrNone   bool        // fill completely or not at all", "    AllOrNone   bool        // fill completely or not at all\n    Venue       string      // not declared anywhere")},
			want:  `documents OrderRequest field "Venue"`,
		},
		{
			name:  "api/doc-retypes-quantity",
			edits: []edit{replace(odoc, "    Quantity    string      // never float64", "    Quantity    float64     // never float64")},
			want:  "documents OrderRequest field Quantity as float64",
		},
		{
			name: "api/doc-reorders-submitresult",
			edits: []edit{replace(odoc,
				"    OrderID string\n    Status  string\n",
				"    Status  string\n    OrderID string\n")},
			want: "order api drift",
		},
		{
			name:  "api/doc-drops-a-submitresult-field",
			edits: []edit{replace(odoc, "    Status  string\n", "")},
			want:  "documents SubmitResult as 2 fields",
		},
		{
			name:  "api/code-adds-a-submitresult-field",
			edits: []edit{replace(tradeSrcPath, "	Replies []Reply\n}", "	Replies []Reply\n	Warning string\n}")},
			want:  "documents SubmitResult as 3 fields",
		},
		{
			name:  "api/submit-signature-drifts",
			edits: []edit{replace(odoc, "req OrderRequest) (*SubmitResult, error)", "req *OrderRequest) (*SubmitResult, error)")},
			want:  "order api drift",
		},
		{
			name:  "api/doc-confirm-loses-the-flag",
			edits: []edit{replace(odoc, "Confirm(ctx context.Context, replyID string, confirmed bool)", "Confirm(ctx context.Context, replyID string)")},
			want:  "order api drift",
		},
		{
			name:  "api/code-confirm-loses-the-flag",
			edits: []edit{replace(tradeSrcPath, "func (m *TradeManager) Confirm(ctx context.Context, replyID string, confirmed bool)", "func (m *TradeManager) Confirm(ctx context.Context, replyID string)")},
			want:  "order api drift",
		},
		{
			name:  "api/reconcile-call-loses-its-shape",
			edits: []edit{replace(tradeSrcPath, "func (m *TradeManager) OpenOrders(ctx context.Context) ([]Order, error) {", "func (m *TradeManager) OpenOrders(ctx context.Context, account AccountID) ([]Order, error) {")},
			want:  "the documented recovery path would not compile",
		},
		{
			name:  "api/trade-accessor-returns-something-else",
			edits: []edit{replace(oclint, "func (c *Client) Trade() *TradeManager { return c.tradeManager }", "func (c *Client) Trade() Order { return Order{} }")},
			want:  "Client.Trade returns Order, not *TradeManager",
		},

		// --- claim 2b: the Reply and OrderRequest field sets, both directions --
		// The document's Reply block once said `Message string` where the
		// declaration says `Messages []string`. The block is corrected and is now
		// enforced in both directions, so the historical defect has to fail.
		{
			name:  "reply/doc-reverts-to-message-string",
			edits: []edit{replace(odoc, "    Messages    []string // human-readable warning texts", "    Message     string   // human-readable warning texts")},
			want:  `documents Reply field "Message", which pkg/ibkr/trade.go does not declare`,
		},
		{
			name:  "reply/doc-retypes-messages",
			edits: []edit{replace(odoc, "    Messages    []string // human-readable warning texts", "    Messages    string   // human-readable warning texts")},
			want:  "documents Reply field Messages as string",
		},
		{
			// A rename in the code. The document's field set is unchanged and
			// every field it lists still exists — under a different name for one
			// of them. Only the type-and-name direction catches this.
			name:  "reply/code-renames-messages",
			edits: []edit{replace(tradeSrcPath, "\tMessages []string\n", "\tTexts []string\n")},
			want:  `documents Reply field "Messages", which pkg/ibkr/trade.go does not declare`,
		},
		{
			// A retype in the code, the mirror of reply/doc-retypes-messages.
			name:  "reply/code-retypes-messages",
			edits: []edit{replace(tradeSrcPath, "\tMessages []string\n", "\tMessages string\n")},
			want:  "documents Reply field Messages as []string but pkg/ibkr/trade.go:100 declares string",
		},
		{
			// A field the code declares and the document omits. Every field the
			// document lists still exists with the documented type, so a
			// document-to-code-only check is silent here.
			name:  "reply/doc-drops-messageids",
			edits: []edit{replace(odoc, "    MessageIDs  []string // additional message ids\n", "")},
			want:  "declares Reply field MessageIDs at pkg/ibkr/trade.go:102, which docs/design/09-orders-and-confirmation.md does not document",
		},
		{
			name:  "reply/code-adds-an-undocumented-field",
			edits: []edit{replace(tradeSrcPath, "\tMessageIDs []string\n}", "\tMessageIDs []string\n\t// Severity is newly added and undocumented.\n\tSeverity int\n}")},
			want:  "declares Reply field Severity at pkg/ibkr/trade.go:",
		},
		{
			// An embedded field carries no name, so it can only surface through
			// the code-to-document direction. A document that shows the struct's
			// fields and silently drops the embedded one is still wrong.
			name:  "reply/code-embeds-a-type",
			edits: []edit{replace(tradeSrcPath, "type Reply struct {\n", "type Reply struct {\n\tfmt.Stringer\n")},
			want:  "embeds fmt.Stringer at pkg/ibkr/trade.go:",
		},
		{
			// The same shape of defect that went unnoticed before: a declared
			// field the document does not mention.
			name:  "orderrequest/doc-drops-parentid",
			edits: []edit{replace(odoc, "    ParentID    string      // parent order id, for grouping\n", "")},
			want:  "declares OrderRequest field ParentID at pkg/ibkr/trade.go:",
		},
		{
			name:  "orderrequest/doc-drops-issinglegroup",
			edits: []edit{replace(odoc, "    IsSingleGroup bool      // quantity is per-conid rather than a total\n", "")},
			want:  "declares OrderRequest field IsSingleGroup at pkg/ibkr/trade.go:",
		},
		{
			name:  "orderrequest/code-adds-an-undocumented-field",
			edits: []edit{replace(tradeSrcPath, "\tIsSingleGroup bool\n}", "\tIsSingleGroup bool\n\t// SettlementCycle is newly added and undocumented.\n\tSettlementCycle string\n}")},
			want:  "declares OrderRequest field SettlementCycle at pkg/ibkr/trade.go:",
		},
		{
			// Equal counts, equal per-name agreement, and the document lists one
			// field twice. Counting alone cannot see this, so the check does not
			// rely on counting.
			name:  "orderrequest/doc-lists-a-field-twice",
			edits: []edit{replace(odoc, "    ParentID    string      // parent order id, for grouping\n", "    ParentID    string      // parent order id, for grouping\n    ParentID    string      // listed a second time\n")},
			want:  `documents OrderRequest field "ParentID" at both`,
		},

		// --- claim 4: confirmation is explicit -------------------------------
		{
			// The rule is about the whole package, not about Confirm. A helper that
			// answers a reply would satisfy every other check here.
			name: "confirm/a-second-caller-answers-replies",
			edits: []edit{replace(tradeSrcPath,
				"// --- internals ---",
				"// autoConfirmForTest answers a reply without the caller asking.\nfunc (m *TradeManager) autoConfirmForTest(ctx context.Context, replyID string) error {\n\tresp, err := m.client.generated.ConfirmOrderReplyWithBody(ctx, replyID,\n\t\t\"application/json\", bytes.NewReader(nil))\n\tif resp != nil {\n\t\tresp.Body.Close()\n\t}\n\treturn err\n}\n\n// --- internals ---")},
			want: "calls the reply-confirmation operation ConfirmOrderReplyWithBody",
		},
		{
			name:  "confirm/flag-becomes-a-literal-true",
			edits: []edit{replace(tradeSrcPath, "confirmReplyJSON{Confirmed: confirmed}", "confirmReplyJSON{Confirmed: true}")},
			want:  "not the caller's confirmed flag marshalled",
		},
		{
			name:  "confirm/reply-id-becomes-a-literal",
			edits: []edit{replace(tradeSrcPath, "ConfirmOrderReplyWithBody(ctx, replyID,", "ConfirmOrderReplyWithBody(ctx, \"\",")},
			want:  "where the reply id \"replyID\" belongs",
		},
		{
			name:  "confirm/flag-tag-gains-omitempty",
			edits: []edit{replace(tradeSrcPath, "Confirmed bool `json:\"confirmed\"`", "Confirmed bool `json:\"confirmed,omitempty\"`")},
			want:  "carries omitempty",
		},
		{
			name:  "confirm/flag-tag-renamed",
			edits: []edit{replace(tradeSrcPath, "Confirmed bool `json:\"confirmed\"`", "Confirmed bool `json:\"confirm\"`")},
			want:  `is serialised as "confirm"`,
		},
		{
			name:  "confirm/doc-rule-reworded",
			edits: []edit{replace(odoc, "Confirmation is **explicit**: the manager does not silently auto-confirm", "Confirmation is usually explicit: the manager rarely auto-confirms")},
			want:  "no longer states that confirmation is explicit",
		},

		// --- claim 5: order mutations are single-attempt ---------------------
		{
			name: "retry/submit-bypasses-mutate",
			edits: []edit{replace(tradeSrcPath,
				"resp, err := m.mutate(ctx, op, func() (*http.Response, error) {\n\t\treturn m.client.generated.SubmitNewOrderWithBody",
				"resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {\n\t\treturn m.client.generated.SubmitNewOrderWithBody")},
			want: "does not go through m.mutate",
		},
		{
			name: "retry/confirm-bypasses-mutate",
			edits: []edit{replace(tradeSrcPath,
				"resp, err := m.mutate(ctx, op, func() (*http.Response, error) {\n\t\treturn m.client.generated.ConfirmOrderReplyWithBody",
				"resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {\n\t\treturn m.client.generated.ConfirmOrderReplyWithBody")},
			want: "does not go through m.mutate",
		},
		{
			name: "retry/modify-bypasses-mutate",
			edits: []edit{replace(tradeSrcPath,
				"resp, err := m.mutate(ctx, op, func() (*http.Response, error) {\n\t\treturn m.client.generated.ModifyOpenOrderWithBody",
				"resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {\n\t\treturn m.client.generated.ModifyOpenOrderWithBody")},
			want: "does not go through m.mutate",
		},
		{
			name: "retry/cancel-bypasses-mutate",
			edits: []edit{replace(tradeSrcPath,
				"resp, err := m.mutate(ctx, op, func() (*http.Response, error) {\n\t\treturn m.client.generated.CancelOpenOrder(",
				"resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {\n\t\treturn m.client.generated.CancelOpenOrder(")},
			want: "does not go through m.mutate",
		},
		{
			// Still routed through mutate, but now also calling the retrying path
			// directly — the half of the rule the first case cannot reach.
			name: "retry/code-calls-netdo-directly",
			edits: []edit{replace(tradeSrcPath,
				"\tif sink := m.client.telemetrySink(); sink != nil {\n\t\tsink.OnOrderSubmit(ctx, internal.OrderEventInfo{",
				"\tif _, err := m.client.netDo(ctx, op, func() (*http.Response, error) { return nil, nil }); err != nil {\n\t\treturn nil, err\n\t}\n\tif sink := m.client.telemetrySink(); sink != nil {\n\t\tsink.OnOrderSubmit(ctx, internal.OrderEventInfo{")},
			want: "calls netDo directly instead of through m.mutate",
		},
		{
			name:  "retry/mutate-no-longer-reaches-the-gateway",
			edits: []edit{replace(tradeSrcPath, "resp, err := m.client.netDo(ctx, op, fn)", "resp, err := fn()")},
			want:  "TradeManager.mutate no longer reaches the gateway",
		},
		{
			// The rule names four operations; dropping one leaves a mutation that
			// is single-attempt but undocumented, which is only visible in the
			// reverse direction.
			name:  "retry/doc-drops-an-operation",
			edits: []edit{replace(odoc, "**No automatic retry** of submit/modify/cancel/confirm", "**No automatic retry** of submit/modify/confirm")},
			want:  "does not name it",
		},
		{
			name:  "retry/doc-adds-an-unknown-operation",
			edits: []edit{replace(odoc, "**No automatic retry** of submit/modify/cancel/confirm", "**No automatic retry** of submit/modify/cancel/confirm/resend")},
			want:  `names "resend", which this checker does not know how to verify`,
		},
		{
			name: "retry/doc-drops-the-rule",
			edits: []edit{replace(odoc,
				"- **No automatic retry** of submit/modify/cancel/confirm\n",
				"- Retries are avoided where possible.\n")},
			want: "does not name the operations it covers",
		},

		// --- claim 6: the error-mapping table -------------------------------
		{
			name:  "mapping/rejection-stops-wrapping-errorderrejected",
			edits: []edit{replace(tradeSrcPath, "Message: errMsg, Err: ErrOrderRejected}", "Message: errMsg, Err: nil}")},
			want:  "wraps nil, not ErrOrderRejected",
		},
		{
			name:  "mapping/rejection-drops-the-code",
			edits: []edit{replace(tradeSrcPath, "Code: rawToString(item, \"error\"), Message: errMsg,", "Message: errMsg,")},
			want:  "the rejection error sets no Code",
		},
		{
			name:  "mapping/rejection-drops-the-message",
			edits: []edit{replace(tradeSrcPath, "Code: rawToString(item, \"error\"), Message: errMsg,", "Code: rawToString(item, \"error\"),")},
			want:  "the rejection error sets no Message",
		},
		{
			name: "mapping/reply-no-longer-continues",
			edits: []edit{replace(tradeSrcPath,
				"\t\t\t\tMessageIDs: rawToStringSlice(item, \"messageIds\"),\n\t\t\t})\n\t\t\tcontinue\n",
				"\t\t\t\tMessageIDs: rawToStringSlice(item, \"messageIds\"),\n\t\t\t})\n")},
			want: "does not continue to the next item",
		},
		{
			// The Reply is still built, but nothing collects it, so a confirmation
			// the gateway asked about would be dropped rather than reported.
			name: "mapping/reply-stops-being-collected",
			edits: []edit{replace(tradeSrcPath,
				"out.Replies = append(out.Replies, Reply{\n\t\t\t\tID:         id,\n\t\t\t\tMessages:   rawToStringSlice(item, \"message\"),\n\t\t\t\tMessageIDs: rawToStringSlice(item, \"messageIds\"),\n\t\t\t})",
				"_ = Reply{\n\t\t\t\tID:         id,\n\t\t\t\tMessages:   rawToStringSlice(item, \"message\"),\n\t\t\t\tMessageIDs: rawToStringSlice(item, \"messageIds\"),\n\t\t\t}")},
			want: "the confirmation branch never appends to SubmitResult.Replies",
		},
		{
			name:  "mapping/empty-response-guard-drops-the-replies-half",
			edits: []edit{replace(tradeSrcPath, "if out.OrderID == \"\" && len(out.Replies) == 0 {", "if out.OrderID == \"\" {")},
			want:  "has no guard rejecting a response that carries neither an order id nor a reply",
		},
		{
			name:  "mapping/empty-response-guard-stops-rejecting",
			edits: []edit{replace(tradeSrcPath, "return nil, &Error{Op: op, Message: \"unexpected order response\", Err: ErrOrderRejected}", "return out, nil")},
			want:  "the empty-response guard builds no *Error",
		},
		{
			name:  "mapping/accepted-ignores-the-replies",
			edits: []edit{replace(tradeSrcPath, "return len(r.Replies) == 0 && r.OrderID != \"\"", "return r.OrderID != \"\"")},
			want:  "confirmation drift",
		},
		{
			name:  "mapping/accepted-drops-the-order-id",
			edits: []edit{replace(tradeSrcPath, "return len(r.Replies) == 0 && r.OrderID != \"\"", "return len(r.Replies) == 0")},
			want:  "confirmation drift",
		},
		{
			name:  "mapping/reconcile-instruction-names-another-api",
			edits: []edit{replace(tradeSrcPath, "reconcile via Trade().OpenOrders before retrying", "reconcile via Trade().Trades before retrying")},
			want:  "does not name OpenOrders",
		},
		{
			name:  "mapping/doc-table-row-removed",
			edits: []edit{replace(odoc, "| Requires confirmation | returned in `SubmitResult.Replies` (not an error) |", "| Requires confirmation | returned in the result |")},
			want:  `no longer has the "Requires confirmation" row`,
		},
	}
}

func ordersGreenCases() []greenCase {
	return []greenCase{
		{
			// The document lists StopPrice before TimeInForce, which is the order
			// the declaration uses. Moving them back the way round is the edit this
			// case makes, and it has to keep passing: field order in a struct is
			// not a contract a caller can observe, so it is not pinned. The
			// declaration still lists both fields, in both directions, and the
			// matchSet comparison is order-blind by construction.
			name: "api/field-order-may-differ",
			edits: []edit{replace(odoc,
				"    StopPrice   string      // for STOP/STOP_LIMIT orders\n    TimeInForce TimeInForce\n",
				"    TimeInForce TimeInForce\n    StopPrice   string      // for STOP/STOP_LIMIT orders\n")},
		},
		{
			// The document's Reply block used to say `Message string` where the
			// declaration says `Messages []string`. It is corrected now, so the
			// field set is enforced in both directions and this case asserts the
			// corrected text is accepted. The comments here are prose and are
			// expected to be free: the check reads the field and its type, not the
			// line, so a reworded comment is not drift.
			name: "api/the-corrected-reply-block-passes",
			edits: []edit{replace(odoc,
				"    Messages    []string // human-readable warning texts\n    MessageIDs  []string // additional message ids",
				"    Messages    []string // the warning texts IBKR returned\n    MessageIDs  []string // which warning categories applied")},
		},
		{
			// The two fields the code declared and the document omitted. Their
			// comments are free; the fields and their types are now pinned.
			name: "api/the-two-newly-documented-fields-pass",
			edits: []edit{replace(odoc,
				"    ParentID    string      // parent order id, for grouping\n    IsSingleGroup bool      // quantity is per-conid rather than a total",
				"    ParentID    string      // bracket parent\n    IsSingleGroup bool      // one-cancels-all group")},
		},
		{
			// client.gen.go declares `json:"quantity"` as float32 on five
			// unrelated types. Reading a tag instead of the declaration would make
			// this case fail, which is the whole point of resolving through the
			// alias and the slice element type.
			name: "wire/an-unrelated-generated-type-may-carry-the-same-tag",
			edits: []edit{replace(ogen,
				"type TradingInstrumentV2 struct {\n\t// Quantity Quantity of the instrument.\n\t//\n\t// Example: 1000\n\tQuantity float32 `json:\"quantity\"`",
				"type TradingInstrumentV2 struct {\n\t// Quantity Quantity of the instrument.\n\t//\n\t// Example: 1000\n\tQuantity string `json:\"quantity\"`")},
		},
		{
			// A pointer to a float32 is the same wire type, and the document's claim
			// is about the wire.
			name: "wire/pointer-money-fields-are-still-float32",
			edits: []edit{replace(ogen,
				"\t// Quantity Quantity of the order ticket in units of the instrument.\n\tQuantity float32 `json:\"quantity\"`",
				"\t// Quantity Quantity of the order ticket in units of the instrument.\n\tQuantity *float32 `json:\"quantity\"`")},
		},
		{
			// The path placeholder's name is the document's prose; only the shape is
			// a claim.
			name:  "endpoint/path-param-names-are-prose",
			edits: []edit{replace(odoc, "`POST /v1/api/iserver/account/{accountId}/orders`", "`POST /v1/api/iserver/account/{acct}/orders`")},
		},
		{
			// Renaming a parameter changes no contract a caller can observe, and
			// the checks that care about a parameter resolve it from the code.
			name: "confirm/a-param-rename-is-not-drift",
			edits: []edit{
				replace(tradeSrcPath, "replyID string, confirmed bool", "reply string, confirmed bool"),
				replace(tradeSrcPath, "ConfirmOrderReplyWithBody(ctx, replyID,", "ConfirmOrderReplyWithBody(ctx, reply,"),
			},
		},
	}
}
