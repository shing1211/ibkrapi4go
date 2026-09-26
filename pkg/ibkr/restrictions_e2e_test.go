// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"testing"
	"time"
)

// newRESTClient builds a client wired to the mock gateway with OAuth2
// configured, which RESTSurface requires. It reuses the gateway fixtures rather
// than hand-written handlers so assertions run against realistic bodies.
func newRESTClient(t *testing.T) *Client {
	t.Helper()
	gw := newGateway(t)
	cli, err := NewClient(
		WithGatewayURL(gw.URL),
		WithRESTGateway(gw.URL),
		WithOAuth2ClientCredentials("cid", "sec"),
		WithOAuth2TokenURL(gw.URL+"/oauth2/api/v1/token"),
		WithTickleInterval(time.Hour),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })
	return cli
}

func restrictionsSurface(t *testing.T) *RESTRestrictions {
	t.Helper()
	surface, err := newRESTClient(t).REST()
	if err != nil {
		t.Fatalf("REST: %v", err)
	}
	return surface.Restrictions()
}

func TestRestrictions_AccountRestrictions(t *testing.T) {
	ids, err := restrictionsSurface(t).AccountRestrictions(context.Background(), AccountID("U1234567"))
	if err != nil {
		t.Fatalf("AccountRestrictions: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("ids = %v; want two entries", ids)
	}
	if ids[0] != 1001 || ids[1] != 1002 {
		t.Errorf("ids = %v; want [1001 1002]", ids)
	}
}

func TestRestrictions_UserRestrictions(t *testing.T) {
	ids, err := restrictionsSurface(t).UserRestrictions(context.Background(), "jdoe")
	if err != nil {
		t.Fatalf("UserRestrictions: %v", err)
	}
	if len(ids) != 1 || ids[0] != 1001 {
		t.Errorf("ids = %v; want [1001]", ids)
	}
}

func TestRestrictions_MasterRestrictionIDs(t *testing.T) {
	got, err := restrictionsSurface(t).MasterRestrictionIDs(context.Background(), "jdoe", "", false)
	if err != nil {
		t.Fatalf("MasterRestrictionIDs: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("entries = %+v; want two", got)
	}
	if got[0].RestrictionID != 1001 || got[0].ByOperator {
		t.Errorf("entries[0] = %+v; want {1001 false}", got[0])
	}
	if got[1].RestrictionID != 1002 || !got[1].ByOperator {
		t.Errorf("entries[1] = %+v; want {1002 true}", got[1])
	}
}

func TestRestrictions_MasterListIDs(t *testing.T) {
	got, err := restrictionsSurface(t).MasterListIDs(context.Background(), "jdoe", "", false)
	if err != nil {
		t.Fatalf("MasterListIDs: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("entries = %+v; want one", got)
	}
	if got[0].ListID != 2001 || got[0].ByOperator {
		t.Errorf("entries[0] = %+v; want {2001 false}", got[0])
	}
}

func TestRestrictions_ListDetails(t *testing.T) {
	got, err := restrictionsSurface(t).ListDetails(context.Background(), "jdoe", 2001, "", false)
	if err != nil {
		t.Fatalf("ListDetails: %v", err)
	}
	if got.ListID != 2001 {
		t.Errorf("listId = %d; want 2001", got.ListID)
	}
	if got.Name != "My Issuer List" {
		t.Errorf("name = %q; want My Issuer List", got.Name)
	}
	if got.Type != "ISSUERORCONID" {
		t.Errorf("type = %q; want ISSUERORCONID", got.Type)
	}
	if len(got.Entries) != 1 {
		t.Fatalf("entries = %+v; want one", got.Entries)
	}
	if got.Entries[0].ID != 300001 {
		t.Errorf("entries[0].id = %d; want 300001", got.Entries[0].ID)
	}
}

func TestRestrictions_RestrictionDetails(t *testing.T) {
	got, err := restrictionsSurface(t).RestrictionDetails(context.Background(), "jdoe", 1001, "", false)
	if err != nil {
		t.Fatalf("RestrictionDetails: %v", err)
	}
	if got.RestrictionID != 1001 {
		t.Errorf("restrictionId = %d; want 1001", got.RestrictionID)
	}
	if got.Name != "My Restriction" {
		t.Errorf("name = %q; want My Restriction", got.Name)
	}
	if got.IsWhiteList {
		t.Error("isWhiteList = true; the fixture encodes \"F\"")
	}
	if len(got.Rules) != 1 {
		t.Fatalf("rules = %+v; want one", got.Rules)
	}
	if got.Rules[0].Type != "CONID" || got.Rules[0].ValidityType != "GTD" {
		t.Errorf("rules[0] = %+v; want {CONID GTD}", got.Rules[0])
	}
}

func TestRestrictions_RestrictionScope(t *testing.T) {
	got, err := restrictionsSurface(t).RestrictionScope(context.Background(), "jdoe", 1001, "", false)
	if err != nil {
		t.Fatalf("RestrictionScope: %v", err)
	}
	if got.RestrictionID != 1001 {
		t.Errorf("restrictionId = %d; want 1001", got.RestrictionID)
	}
	if got.Scope != "Active For Some" {
		t.Errorf("scope = %q; want Active For Some", got.Scope)
	}
	if len(got.AccountIDs) != 1 || got.AccountIDs[0] != AccountID("U1234567") {
		t.Errorf("accountIds = %v; want [U1234567]", got.AccountIDs)
	}
	if got.Truncated {
		t.Error("truncated = true; the fixture encodes false")
	}
}

func TestRestrictions_ApplyCSV(t *testing.T) {
	got, err := restrictionsSurface(t).ApplyCSV(context.Background(), "", "csv-content")
	if err != nil {
		t.Fatalf("ApplyCSV: %v", err)
	}
	if !got.Success {
		t.Error("success = false; want true")
	}
	if got.RequestID != 6001 {
		t.Errorf("requestId = %d; want 6001", got.RequestID)
	}
	if got.Message != "OK" {
		t.Errorf("message = %q; want OK", got.Message)
	}
}

func TestRestrictions_VerifyCSV(t *testing.T) {
	got, err := restrictionsSurface(t).VerifyCSV(context.Background(), "", CSVVerifyRequest{
		UserName:  "jdoe",
		RequestID: 6002,
		Payload:   []byte("csv-content"),
	})
	if err != nil {
		t.Fatalf("VerifyCSV: %v", err)
	}
	if !got.Success {
		t.Error("success = false; want true")
	}
	if got.RequestID != 6002 {
		t.Errorf("requestId = %d; want 6002", got.RequestID)
	}
}
