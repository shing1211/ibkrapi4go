// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

// exampleGateway returns a minimal gateway for examples. It answers logout and
// one accounts endpoint; real usage points at the local Client Portal Gateway.
func exampleGateway() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/api/logout", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{}`)
	})
	mux.HandleFunc("/v1/api/iserver/accounts", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"accounts":["U1234567"],"aliases":{"U1234567":"Main"}}`)
	})
	return httptest.NewServer(mux)
}

func ExampleNewClient() {
	srv := exampleGateway()
	defer srv.Close()

	cli, err := ibkr.NewClient(ibkr.WithGatewayURL(srv.URL))
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	fmt.Println("gateway configured:", cli.GatewayURL() == srv.URL)
	// Output: gateway configured: true
}

func ExampleClient_Account() {
	srv := exampleGateway()
	defer srv.Close()

	cli, err := ibkr.NewClient(ibkr.WithGatewayURL(srv.URL))
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	accounts, err := cli.Account().List(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	for _, a := range accounts {
		fmt.Println(a.ID, a.Alias)
	}
	// Output: U1234567 Main
}
