// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRESTAccounts_List(t *testing.T) {
	var accountPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/api/v1/token":
			fmt.Fprint(w, `{"access_token":"tok-1","expires_in":3600,"token_type":"Bearer"}`)
		case "/gw/api/v1/accounts":
			accountPath = r.URL.Path
			fmt.Fprint(w, `[{"id":"U123","accountAlias":"Main","baseCurrency":"USD"}]`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithRESTGateway(srv.URL),
		WithOAuth2ClientCredentials("cid", "sec"),
		WithOAuth2TokenURL(srv.URL+"/oauth2/api/v1/token"),
		WithTickleInterval(time.Hour),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	rest, err := cli.REST()
	if err != nil {
		t.Fatalf("REST: %v", err)
	}
	accounts, err := rest.Accounts().List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(accounts) != 1 || string(accounts[0].ID) != "U123" {
		t.Errorf("accounts = %+v; want [U123]", accounts)
	}
	if accountPath != "/gw/api/v1/accounts" {
		t.Errorf("account path = %q; want /gw/api/v1/accounts", accountPath)
	}
}

func TestRESTAccounts_BulkStatus(t *testing.T) {
	var accountPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/api/v1/token":
			fmt.Fprint(w, `{"access_token":"tok-1","expires_in":3600,"token_type":"Bearer"}`)
		case "/gw/api/v1/accounts/status":
			accountPath = r.URL.Path
			fmt.Fprint(w, `{"accounts":[{"accountId":"U123","status":"ACTIVE"}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithRESTGateway(srv.URL),
		WithOAuth2ClientCredentials("cid", "sec"),
		WithOAuth2TokenURL(srv.URL+"/oauth2/api/v1/token"),
		WithTickleInterval(time.Hour),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	rest, err := cli.REST()
	if err != nil {
		t.Fatalf("REST: %v", err)
	}
	statuses, err := rest.Accounts().BulkStatus(context.Background())
	if err != nil {
		t.Fatalf("BulkStatus: %v", err)
	}
	if len(statuses) != 1 || string(statuses[0].AccountID) != "U123" {
		t.Errorf("statuses = %+v; want [U123/ACTIVE]", statuses)
	}
	if statuses[0].Status != "ACTIVE" {
		t.Errorf("status = %q; want ACTIVE", statuses[0].Status)
	}
	if accountPath != "/gw/api/v1/accounts/status" {
		t.Errorf("account path = %q; want /gw/api/v1/accounts/status", accountPath)
	}
}

func TestRESTAccounts_KycURL(t *testing.T) {
	var accountPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/api/v1/token":
			fmt.Fprint(w, `{"access_token":"tok-1","expires_in":3600,"token_type":"Bearer"}`)
		case "/gw/api/v1/accounts/U123/kyc":
			accountPath = r.URL.Path
			fmt.Fprint(w, `{"externalId":"https://au10tix.com/verify/abc123"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithRESTGateway(srv.URL),
		WithOAuth2ClientCredentials("cid", "sec"),
		WithOAuth2TokenURL(srv.URL+"/oauth2/api/v1/token"),
		WithTickleInterval(time.Hour),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	rest, err := cli.REST()
	if err != nil {
		t.Fatalf("REST: %v", err)
	}
	url, err := rest.Accounts().KycURL(context.Background(), "U123")
	if err != nil {
		t.Fatalf("KycURL: %v", err)
	}
	if url != "https://au10tix.com/verify/abc123" {
		t.Errorf("url = %q; want https://au10tix.com/verify/abc123", url)
	}
	if accountPath != "/gw/api/v1/accounts/U123/kyc" {
		t.Errorf("account path = %q; want /gw/api/v1/accounts/U123/kyc", accountPath)
	}
}

func TestRESTAccounts_Tasks(t *testing.T) {
	var accountPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/api/v1/token":
			fmt.Fprint(w, `{"access_token":"tok-1","expires_in":3600,"token_type":"Bearer"}`)
		default:
			if strings.HasPrefix(r.URL.Path, "/gw/api/v1/accounts/U123/tasks") {
				accountPath = r.URL.Path
				fmt.Fprint(w, `{"registrationTasks":[{"taskId":"T1","action":"SIGN","isCompleted":false}]}`)
			}
		}
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithRESTGateway(srv.URL),
		WithOAuth2ClientCredentials("cid", "sec"),
		WithOAuth2TokenURL(srv.URL+"/oauth2/api/v1/token"),
		WithTickleInterval(time.Hour),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	rest, err := cli.REST()
	if err != nil {
		t.Fatalf("REST: %v", err)
	}
	tasks, err := rest.Accounts().Tasks(context.Background(), "U123", "registration")
	if err != nil {
		t.Fatalf("Tasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].TaskID != "T1" {
		t.Errorf("tasks = %+v; want [T1/SIGN]", tasks)
	}
	if accountPath == "" || !strings.HasPrefix(accountPath, "/gw/api/v1/accounts/U123/tasks") {
		t.Errorf("account path = %q; want /gw/api/v1/accounts/U123/tasks", accountPath)
	}
}

func TestRESTAccounts_UpdateStatus(t *testing.T) {
	var reqBody, accountPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/api/v1/token":
			fmt.Fprint(w, `{"access_token":"tok-1","expires_in":3600,"token_type":"Bearer"}`)
		case "/gw/api/v1/accounts/U123/status":
			accountPath = r.URL.Path
			body := make([]byte, 1024)
			n, _ := r.Body.Read(body)
			reqBody = string(body[:n])
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithRESTGateway(srv.URL),
		WithOAuth2ClientCredentials("cid", "sec"),
		WithOAuth2TokenURL(srv.URL+"/oauth2/api/v1/token"),
		WithTickleInterval(time.Hour),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	rest, err := cli.REST()
	if err != nil {
		t.Fatalf("REST: %v", err)
	}
	err = rest.Accounts().UpdateStatus(context.Background(), "U123", "ACTIVE")
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	if !strings.Contains(reqBody, "ACTIVE") {
		t.Errorf("request body = %q; want to contain ACTIVE", reqBody)
	}
	if accountPath != "/gw/api/v1/accounts/U123/status" {
		t.Errorf("account path = %q; want /gw/api/v1/accounts/U123/status", accountPath)
	}
}

func TestRESTAccounts_MutationsAreSingleAttempt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/api/v1/token" {
			fmt.Fprint(w, `{"access_token":"tok-1","expires_in":3600,"token_type":"Bearer"}`)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithRESTGateway(srv.URL),
		WithOAuth2ClientCredentials("cid", "sec"),
		WithOAuth2TokenURL(srv.URL+"/oauth2/api/v1/token"),
		WithTickleInterval(time.Hour),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	rest, err := cli.REST()
	if err != nil {
		t.Fatalf("REST: %v", err)
	}

	err = rest.Accounts().UpdateStatus(context.Background(), "U123", "ACTIVE")
	if err == nil {
		t.Fatal("UpdateStatus on 503 = nil; want error")
	}
}

func TestRESTAccounts_SubmitDocument(t *testing.T) {
	var gotContentType, gotAccountID, accountPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/api/v1/token":
			fmt.Fprint(w, `{"access_token":"tok-1","expires_in":3600,"token_type":"Bearer"}`)
		case "/gw/api/v1/accounts/documents":
			accountPath = r.URL.Path
			gotContentType = r.Header.Get("Content-Type")
			r.ParseMultipartForm(32 << 20)
			gotAccountID = r.FormValue("accountId")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cli, err := NewClient(
		WithGatewayURL(srv.URL),
		WithRESTGateway(srv.URL),
		WithOAuth2ClientCredentials("cid", "sec"),
		WithOAuth2TokenURL(srv.URL+"/oauth2/api/v1/token"),
		WithTickleInterval(time.Hour),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	rest, err := cli.REST()
	if err != nil {
		t.Fatalf("REST: %v", err)
	}
	err = rest.Accounts().SubmitDocument(context.Background(), "U123", strings.NewReader("pdf content"), "doc.pdf", "application/pdf")
	if err != nil {
		t.Fatalf("SubmitDocument: %v", err)
	}
	if !strings.HasPrefix(gotContentType, "multipart/form-data") {
		t.Errorf("Content-Type = %q; want multipart/form-data", gotContentType)
	}
	if gotAccountID != "U123" {
		t.Errorf("accountId = %q; want U123", gotAccountID)
	}
	if accountPath != "/gw/api/v1/accounts/documents" {
		t.Errorf("account path = %q; want /gw/api/v1/accounts/documents", accountPath)
	}
}
