// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/shing1211/ibkrapi4go/internal/mockgateway"
)

// accountsWithGateway returns the REST accounts sub-manager together with the
// mock gateway behind it. The mutation wrappers return nothing but an error, so
// every test here also reads back the request the gateway recorded: the path
// proves which operation was called, and the recorded body is the only
// observable product of a mutation.
func accountsWithGateway(t *testing.T) (*RESTAccounts, *gateway) {
	t.Helper()
	cli, gw := newRESTClientWithGateway(t)
	surface, err := cli.REST()
	if err != nil {
		t.Fatalf("REST: %v", err)
	}
	return surface.Accounts(), gw
}

// lastRESTRequest returns the most recent gateway request that is not the
// OAuth2 token exchange, so a test can assert the method, path, headers, and
// body of the call it just made.
func lastRESTRequest(t *testing.T, gw *gateway) *mockgateway.Request {
	t.Helper()
	reqs := gw.srv.Recorder().Requests()
	for i := len(reqs) - 1; i >= 0; i-- {
		if reqs[i].Path != "/oauth2/api/v1/token" {
			return reqs[i]
		}
	}
	t.Fatal("gateway recorded no REST request")
	return nil
}

// decodeRequestBody decodes the body the gateway recorded for req into v.
func decodeRequestBody(t *testing.T, req *mockgateway.Request, v any) {
	t.Helper()
	if err := json.Unmarshal(req.Body, v); err != nil {
		t.Fatalf("decode recorded body %q: %v", req.Body, err)
	}
}

// assertLoginMessage checks the fields of the single login message both login
// message fixtures carry, so both wrappers are held to the same decoded values.
func assertLoginMessage(t *testing.T, label string, got LoginMessage) {
	t.Helper()
	if got.ID != 1 {
		t.Errorf("%s id = %d; want 1", label, got.ID)
	}
	if got.Description != "Welcome" {
		t.Errorf("%s description = %q; want Welcome", label, got.Description)
	}
	if got.MessageType != "INFO" {
		t.Errorf("%s messageType = %q; want INFO", label, got.MessageType)
	}
	if got.State != "UNREAD" {
		t.Errorf("%s state = %q; want UNREAD", label, got.State)
	}
	if got.Username != "jdoe" {
		t.Errorf("%s username = %q; want jdoe", label, got.Username)
	}
	if got.ContentID != 1 {
		t.Errorf("%s contentId = %d; want 1", label, got.ContentID)
	}
	if len(got.Tasks) != 1 || got.Tasks[0] != 1 {
		t.Errorf("%s tasks = %v; want [1]", label, got.Tasks)
	}
	want := time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC)
	if !got.RecordDate.Equal(want) {
		t.Errorf("%s recordDate = %s; want %s", label, got.RecordDate, want)
	}
}

func TestRESTAccounts_LoginMessages(t *testing.T) {
	accounts, gw := accountsWithGateway(t)

	messages, err := accounts.LoginMessages(context.Background())
	if err != nil {
		t.Fatalf("LoginMessages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("messages = %+v; want exactly one", messages)
	}
	assertLoginMessage(t, "messages[0]", messages[0])

	req := lastRESTRequest(t, gw)
	if req.Method != http.MethodGet {
		t.Errorf("method = %s; want GET", req.Method)
	}
	if req.Path != "/gw/api/v1/accounts/login-messages" {
		t.Errorf("path = %q; want /gw/api/v1/accounts/login-messages", req.Path)
	}
}

func TestRESTAccounts_LoginMessagesForAccount(t *testing.T) {
	accounts, gw := accountsWithGateway(t)

	messages, err := accounts.LoginMessagesForAccount(context.Background(), AccountID("U1234567"))
	if err != nil {
		t.Fatalf("LoginMessagesForAccount: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("messages = %+v; want exactly one", messages)
	}
	assertLoginMessage(t, "messages[0]", messages[0])

	req := lastRESTRequest(t, gw)
	if req.Method != http.MethodGet {
		t.Errorf("method = %s; want GET", req.Method)
	}
	if req.Path != "/gw/api/v1/accounts/U1234567/login-messages" {
		t.Errorf("path = %q; want /gw/api/v1/accounts/U1234567/login-messages", req.Path)
	}
}

func TestRESTAccounts_Status(t *testing.T) {
	accounts, gw := accountsWithGateway(t)

	status, err := accounts.Status(context.Background(), AccountID("U1234567"))
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status == nil {
		t.Fatal("Status = nil; want a decoded status")
	}
	if status.AccountID != AccountID("U1234567") {
		t.Errorf("accountId = %q; want U1234567", status.AccountID)
	}
	if status.AdminAccountID != AccountID("U1234567") {
		t.Errorf("adminAccountId = %q; want U1234567", status.AdminAccountID)
	}
	if status.MasterAccountID != AccountID("U1234567") {
		t.Errorf("masterAccountId = %q; want U1234567", status.MasterAccountID)
	}
	if status.Status != "ACTIVE" || status.State != "ACTIVE" {
		t.Errorf("status/state = %q/%q; want ACTIVE/ACTIVE", status.Status, status.State)
	}
	if status.Description != "Individual account" {
		t.Errorf("description = %q; want Individual account", status.Description)
	}
	if status.Message != "ok" {
		t.Errorf("message = %q; want ok", status.Message)
	}
	wantDate := time.Date(2020, time.January, 2, 15, 4, 5, 0, time.UTC)
	if !status.DateOpened.Equal(wantDate) {
		t.Errorf("dateOpened = %s; want %s", status.DateOpened, wantDate)
	}
	if !status.DateStarted.Equal(wantDate) {
		t.Errorf("dateStarted = %s; want %s", status.DateStarted, wantDate)
	}
	// The fixture omits dateClosed, so it must stay the zero time.
	if !status.DateClosed.IsZero() {
		t.Errorf("dateClosed = %s; want the zero time", status.DateClosed)
	}

	req := lastRESTRequest(t, gw)
	if req.Method != http.MethodGet {
		t.Errorf("method = %s; want GET", req.Method)
	}
	if req.Path != "/gw/api/v1/accounts/U1234567/status" {
		t.Errorf("path = %q; want /gw/api/v1/accounts/U1234567/status", req.Path)
	}
}

func TestRESTAccounts_Update(t *testing.T) {
	accounts, gw := accountsWithGateway(t)
	typ, value := "MGMT_WEB", true

	if err := accounts.Update(context.Background(), AccountConfiguration{
		AccountID: AccountID("U1234567"),
		Type:      &typ,
		Value:     &value,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	req := lastRESTRequest(t, gw)
	if req.Method != http.MethodPatch {
		t.Errorf("method = %s; want PATCH", req.Method)
	}
	if req.Path != "/gw/api/v1/accounts" {
		t.Errorf("path = %q; want /gw/api/v1/accounts", req.Path)
	}
	if got := req.Headers.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q; want application/json", got)
	}
	var got struct {
		AccountID string `json:"accountId"`
		Type      string `json:"type"`
		Value     bool   `json:"value"`
	}
	decodeRequestBody(t, req, &got)
	if got.AccountID != "U1234567" {
		t.Errorf("accountId = %q; want U1234567", got.AccountID)
	}
	if got.Type != "MGMT_WEB" {
		t.Errorf("type = %q; want MGMT_WEB", got.Type)
	}
	if !got.Value {
		t.Error("value = false; want true")
	}

	// An empty AccountID must be omitted rather than sent as "".
	if err := accounts.Update(context.Background(), AccountConfiguration{Type: &typ, Value: &value}); err != nil {
		t.Fatalf("Update without account: %v", err)
	}
	req = lastRESTRequest(t, gw)
	var keys map[string]any
	decodeRequestBody(t, req, &keys)
	if _, ok := keys["accountId"]; ok {
		t.Errorf("body = %s; want no accountId key", req.Body)
	}
	if _, ok := keys["type"]; !ok {
		t.Errorf("body = %s; want a type key", req.Body)
	}
}

func TestRESTAccounts_Create(t *testing.T) {
	accounts, gw := accountsWithGateway(t)
	payload := `{"firstName":"Jane","lastName":"Doe","baseCurrency":"USD","applicantType":"INDIVIDUAL"}`

	// An empty mimeType must fall back to application/json.
	if err := accounts.Create(context.Background(), strings.NewReader(payload), ""); err != nil {
		t.Fatalf("Create: %v", err)
	}

	req := lastRESTRequest(t, gw)
	if req.Method != http.MethodPost {
		t.Errorf("method = %s; want POST", req.Method)
	}
	if req.Path != "/gw/api/v1/accounts" {
		t.Errorf("path = %q; want /gw/api/v1/accounts", req.Path)
	}
	if got := req.Headers.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q; want application/json", got)
	}
	// The payload is opaque to Create and must reach the gateway verbatim.
	if string(req.Body) != payload {
		t.Errorf("body = %s; want %s", req.Body, payload)
	}

	// An explicit mimeType must be forwarded rather than defaulted.
	csv := "firstName,lastName\nJane,Doe\n"
	if err := accounts.Create(context.Background(), strings.NewReader(csv), "text/csv"); err != nil {
		t.Fatalf("Create with mimeType: %v", err)
	}
	req = lastRESTRequest(t, gw)
	if got := req.Headers.Get("Content-Type"); got != "text/csv" {
		t.Errorf("Content-Type = %q; want text/csv", got)
	}
	if string(req.Body) != csv {
		t.Errorf("body = %q; want %q", req.Body, csv)
	}
}

func TestRESTAccounts_UpdateTasks(t *testing.T) {
	accounts, gw := accountsWithGateway(t)

	updates := []TaskUpdate{
		{TaskID: "t1", IsCompleted: true},
		{TaskID: "t2", IsCompleted: true, Action: "DECLINE"},
		{TaskID: "t3"},
	}
	if err := accounts.UpdateTasks(context.Background(), AccountID("U1234567"), "registration", updates); err != nil {
		t.Fatalf("UpdateTasks: %v", err)
	}

	req := lastRESTRequest(t, gw)
	if req.Method != http.MethodPatch {
		t.Errorf("method = %s; want PATCH", req.Method)
	}
	if req.Path != "/gw/api/v1/accounts/U1234567/tasks" {
		t.Errorf("path = %q; want /gw/api/v1/accounts/U1234567/tasks", req.Path)
	}
	if got := req.Headers.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q; want application/json", got)
	}
	var got struct {
		Type  string           `json:"type"`
		Tasks []map[string]any `json:"tasks"`
	}
	decodeRequestBody(t, req, &got)
	if got.Type != "registration" {
		t.Errorf("type = %q; want registration", got.Type)
	}
	if len(got.Tasks) != 3 {
		t.Fatalf("body = %s; want three task entries", req.Body)
	}
	if got.Tasks[0]["taskId"] != "t1" || got.Tasks[0]["isCompleted"] != true {
		t.Errorf("tasks[0] = %v; want t1 completed", got.Tasks[0])
	}
	if got.Tasks[1]["taskId"] != "t2" || got.Tasks[1]["isCompleted"] != true {
		t.Errorf("tasks[1] = %v; want t2 completed", got.Tasks[1])
	}
	if got.Tasks[1]["action"] != "DECLINE" {
		t.Errorf("tasks[1].action = %v; want DECLINE", got.Tasks[1]["action"])
	}
	// isCompleted has no omitempty, so a false update is sent as an explicit
	// false rather than dropped from the PATCH body.
	if got.Tasks[2]["isCompleted"] != false {
		t.Errorf("tasks[2].isCompleted = %v; want an explicit false", got.Tasks[2]["isCompleted"])
	}
	if got.Tasks[2]["taskId"] != "t3" {
		t.Errorf("tasks[2].taskId = %v; want t3", got.Tasks[2]["taskId"])
	}
}

// TestRESTAccounts_UpdateTasks_EmitsFalseIsCompleted is the regression test for
// silent data loss on the task mutation path. IsCompleted once carried
// omitempty on a plain bool, so encoding/json dropped false and
// TaskUpdate{TaskID: "t2", IsCompleted: false} serialised to {"taskId":"t2"}.
// Because the endpoint is a PATCH, a missing key means "leave unchanged", so
// asking to mark a task not-completed silently mutated nothing. The assertion is
// on the raw outbound body, since the omission is invisible to a decoded value.
func TestRESTAccounts_UpdateTasks_EmitsFalseIsCompleted(t *testing.T) {
	accounts, gw := accountsWithGateway(t)

	err := accounts.UpdateTasks(context.Background(), AccountID("U1234567"), "registration", []TaskUpdate{
		{TaskID: "t1", IsCompleted: true},
		{TaskID: "t2", IsCompleted: false},
	})
	if err != nil {
		t.Fatalf("UpdateTasks: %v", err)
	}

	body := string(lastRESTRequest(t, gw).Body)
	if !strings.Contains(body, `"isCompleted":false`) {
		t.Errorf("body = %s; want it to contain %q so the false update is not silently dropped", body, `"isCompleted":false`)
	}
	// Guard against a fix that only ever emits false: a true update must still be
	// distinguishable, so both entries carry their own explicit value.
	if !strings.Contains(body, `"isCompleted":true`) {
		t.Errorf("body = %s; want it to contain %q", body, `"isCompleted":true`)
	}

	// A single false-only update is the minimal reproduction: the key must be
	// present on its own, with taskId.
	err = accounts.UpdateTasks(context.Background(), AccountID("U1234567"), "", []TaskUpdate{
		{TaskID: "t2", IsCompleted: false},
	})
	if err != nil {
		t.Fatalf("UpdateTasks with a single false update: %v", err)
	}
	if got, want := string(lastRESTRequest(t, gw).Body), `{"tasks":[{"taskId":"t2","isCompleted":false}]}`; got != want {
		t.Errorf("body = %s; want %s", got, want)
	}
}

func TestRESTAccounts_AssignTask(t *testing.T) {
	accounts, gw := accountsWithGateway(t)
	externalID, formName, state := "ext-1", "Form 1", "ACTIVE"
	formNumber, completed, declined, required := int64(1), true, false, true
	questionIDs := []int64{1, 2}

	if err := accounts.AssignTask(context.Background(), AccountID("U1234567"), TaskAssignment{
		ExternalID:            &externalID,
		FormName:              &formName,
		FormNumber:            &formNumber,
		IsCompleted:           &completed,
		IsDeclined:            &declined,
		IsRequiredForApproval: &required,
		QuestionIDs:           &questionIDs,
		State:                 &state,
	}); err != nil {
		t.Fatalf("AssignTask: %v", err)
	}

	req := lastRESTRequest(t, gw)
	if req.Method != http.MethodPost {
		t.Errorf("method = %s; want POST", req.Method)
	}
	if req.Path != "/gw/api/v1/accounts/U1234567/tasks" {
		t.Errorf("path = %q; want /gw/api/v1/accounts/U1234567/tasks", req.Path)
	}
	if got := req.Headers.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q; want application/json", got)
	}
	var got struct {
		ExternalID            string  `json:"externalId"`
		FormName              string  `json:"formName"`
		FormNumber            int64   `json:"formNumber"`
		IsCompleted           bool    `json:"isCompleted"`
		IsDeclined            bool    `json:"isDeclined"`
		IsRequiredForApproval bool    `json:"isRequiredForApproval"`
		QuestionIDs           []int64 `json:"questionIds"`
		State                 string  `json:"state"`
	}
	decodeRequestBody(t, req, &got)
	if got.ExternalID != "ext-1" {
		t.Errorf("externalId = %q; want ext-1", got.ExternalID)
	}
	if got.FormName != "Form 1" {
		t.Errorf("formName = %q; want Form 1", got.FormName)
	}
	if got.FormNumber != 1 {
		t.Errorf("formNumber = %d; want 1", got.FormNumber)
	}
	if !got.IsCompleted {
		t.Error("isCompleted = false; want true")
	}
	// The generated model uses pointer bools, so an explicit false survives the
	// encode and stays distinguishable from an unset flag.
	if got.IsDeclined {
		t.Error("isDeclined = true; want the explicitly sent false")
	}
	if !got.IsRequiredForApproval {
		t.Error("isRequiredForApproval = false; want true")
	}
	if len(got.QuestionIDs) != 2 || got.QuestionIDs[0] != 1 || got.QuestionIDs[1] != 2 {
		t.Errorf("questionIds = %v; want [1 2]", got.QuestionIDs)
	}
	if got.State != "ACTIVE" {
		t.Errorf("state = %q; want ACTIVE", got.State)
	}
}
