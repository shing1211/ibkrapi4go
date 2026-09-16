// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestRESTSurface_Details(t *testing.T) {
	var (
		mu          sync.Mutex
		detailsAuth string
		detailsPath string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/api/logout":
			fmt.Fprint(w, `{}`)
		case "/oauth2/api/v1/token":
			fmt.Fprint(w, `{"access_token":"tok-1","expires_in":3600,"token_type":"Bearer"}`)
		case "/gw/api/v1/accounts/U1234567/details":
			mu.Lock()
			detailsAuth = r.Header.Get("Authorization")
			detailsPath = r.URL.Path
			mu.Unlock()
			fmt.Fprint(w, `{"accountId":"U1234567","accountAlias":"Main","accountTitle":"Main Account","baseCurrency":"USD","household":"HH1"}`)
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
	details, err := rest.Accounts().Details(context.Background(), "U1234567")
	if err != nil {
		t.Fatalf("Details: %v", err)
	}
	if details.ID != "U1234567" || details.Alias != "Main" || details.Title != "Main Account" {
		t.Errorf("details = %+v; want U1234567/Main/Main Account", details)
	}
	if details.BaseCurrency != "USD" || details.Household != "HH1" {
		t.Errorf("details = %+v; want USD/HH1", details)
	}

	mu.Lock()
	defer mu.Unlock()
	if detailsAuth != "Bearer tok-1" {
		t.Errorf("Authorization = %q; want Bearer tok-1", detailsAuth)
	}
	if detailsPath != "/gw/api/v1/accounts/U1234567/details" {
		t.Errorf("path = %q; want REST account details", detailsPath)
	}
}

func TestRESTSurface_NotConfigured(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	cli, err := NewClient(WithGatewayURL(srv.URL), WithTickleInterval(time.Hour))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	if _, err := cli.REST(); err == nil {
		t.Fatal("REST without oauth = nil error; want configuration error")
	}
}
