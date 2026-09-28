// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"encoding/json"
	"testing"
)

// The ModelManager read and write methods below had no test at all: the four
// model tests that existed covered rebalance, invest/divest, the cash analyzer
// and the portfolio-order collision, and none of them touched the preset and
// account queries or their three mutations. That left a public surface of the SDK
// - the whole model-portfolio configuration API - never exercised.

// modelObjectBody decodes the recorded body as a JSON object. These endpoints
// send either an object or an array depending on the call, so the caller decodes
// into the shape it is asserting about rather than a shared helper guessing.
func modelObjectBody(t *testing.T, gw *gateway) map[string]any {
	t.Helper()
	var body map[string]any
	decodeRequestBody(t, lastRESTRequest(t, gw), &body)
	return body
}

func TestModels_ModelPresets(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	got, err := cli.Model().ModelPresets(context.Background(), 42)
	if err != nil {
		t.Fatalf("ModelPresets: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("no presets returned")
	}
	// Presets carry a name and the account IDs grouped under it; both must come
	// through, since the grouping is the whole point of a preset.
	found := false
	for _, p := range got {
		if p.Name != "" && len(p.Accounts) > 0 {
			found = true
			if p.Accounts[0] == "" {
				t.Errorf("preset %q has an empty account ID at index 0", p.Name)
			}
		}
	}
	if !found {
		t.Errorf("no preset had both a name and accounts: %+v", got)
	}
}

func TestModels_SetModelPresets(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	err := cli.Model().SetModelPresets(context.Background(), []ModelPreset{
		{Name: "Balanced", Accounts: []AccountID{"U111", "U222"}},
		{Name: "Growth", Accounts: []AccountID{"U333"}},
	})
	if err != nil {
		t.Fatalf("SetModelPresets: %v", err)
	}

	// The body is a JSON array of presets. The accounts must go out as strings,
	// not numbers: they are identifiers.
	var body []struct {
		Name     string   `json:"name"`
		Accounts []string `json:"accounts"`
	}
	decodeRequestBody(t, lastRESTRequest(t, gw), &body)
	if len(body) != 2 {
		t.Fatalf("request body has %d presets; want two", len(body))
	}
	if body[0].Name != "Balanced" {
		t.Errorf("presets[0].name = %q; want Balanced", body[0].Name)
	}
	if len(body[0].Accounts) != 2 || body[0].Accounts[0] != "U111" {
		t.Errorf("presets[0].accounts = %q; want [U111 U222]", body[0].Accounts)
	}
}

func TestModels_AccountsInModel(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	got, err := cli.Model().AccountsInModel(context.Background(), "Balanced")
	if err != nil {
		t.Fatalf("AccountsInModel: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("no accounts returned")
	}
	for _, a := range got {
		if a.AccountID == "" {
			t.Errorf("account with an empty ID: %+v", a)
		}
	}

	if body := modelObjectBody(t, gw); body["model"] != "Balanced" {
		t.Errorf("request body model = %v; want Balanced", body["model"])
	}
}

func TestModels_SetAccountInvestmentInModel(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	if err := cli.Model().SetAccountInvestmentInModel(context.Background(), "Balanced", "25000.75"); err != nil {
		t.Fatalf("SetAccountInvestmentInModel: %v", err)
	}

	// The amount is money: it is handed in as a string and must go out as a
	// string, carrying the caller's digits. It is a string in the request body
	// here, so this also pins that the wrapper does not route it through a float.
	body := modelObjectBody(t, gw)
	if got, want := body["amount"], "25000.75"; got != want {
		t.Errorf("request body amount = %#v; want %v", got, want)
	}
	if body["model"] != "Balanced" {
		t.Errorf("request body model = %v; want Balanced", body["model"])
	}
}

func TestModels_InvestedAccountsInModel(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	raw, err := cli.Model().InvestedAccountsInModel(context.Background(), "Balanced")
	if err != nil {
		t.Fatalf("InvestedAccountsInModel: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("empty response body")
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode invested accounts: %v", err)
	}
}

func TestModels_AllModels(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	names, err := cli.Model().AllModels(context.Background(), 7)
	if err != nil {
		t.Fatalf("AllModels: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("no model portfolios returned")
	}
	for _, n := range names {
		if n == "" {
			t.Error("empty model name in the result list")
		}
	}
}

func TestModels_ModelsPager(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)
	ctx := context.Background()

	pager := cli.Model().ModelsPager(ctx, 7)
	var names []string
	for pager.Next(ctx) {
		names = append(names, pager.Value())
	}
	if err := pager.Err(); err != nil {
		t.Fatalf("pager: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("pager produced no model names")
	}

	// The pager wraps AllModels, so the two must agree. This is the check that
	// would catch the pager silently diverging from the call it replaced - the
	// models example now uses the pager, so a divergence is user-visible.
	direct, err := cli.Model().AllModels(ctx, 7)
	if err != nil {
		t.Fatalf("AllModels: %v", err)
	}
	if len(names) != len(direct) {
		t.Errorf("pager returned %d names, AllModels returned %d", len(names), len(direct))
		return
	}
	for i := range names {
		if names[i] != direct[i] {
			t.Errorf("name %d: pager = %q, AllModels = %q", i, names[i], direct[i])
		}
	}
}

func TestModels_AllModelPositions(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	positions, err := cli.Model().AllModelPositions(context.Background(), "Balanced")
	if err != nil {
		t.Fatalf("AllModelPositions: %v", err)
	}
	// An empty list is a legitimate answer, so this asserts the call completed
	// and decoded rather than demanding rows the fixture may not carry.
	if positions == nil {
		t.Error("AllModelPositions returned a nil slice; want an empty one at worst")
	}
}

func TestModels_ModelSummarySingle(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	summary, err := cli.Model().ModelSummarySingle(context.Background(), "Balanced")
	if err != nil {
		t.Fatalf("ModelSummarySingle: %v", err)
	}
	if summary == nil {
		t.Fatal("ModelSummarySingle returned nil with no error")
	}
}

func TestModels_SetModelTargetPositions(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	err := cli.Model().SetModelTargetPositions(context.Background(), "Balanced", nil)
	if err != nil {
		t.Fatalf("SetModelTargetPositions: %v", err)
	}
	if body := modelObjectBody(t, gw); body["model"] != "Balanced" {
		t.Errorf("request body model = %v; want Balanced", body["model"])
	}
}

func TestModels_SubmitModelOrders(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	err := cli.Model().SubmitModelOrders(context.Background(), "Balanced")
	if err != nil {
		t.Fatalf("SubmitModelOrders: %v", err)
	}
	if body := modelObjectBody(t, gw); body["model"] != "Balanced" {
		t.Errorf("request body model = %v; want Balanced", body["model"])
	}
}
