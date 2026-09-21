// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Command multi-account demonstrates using MultiClient to aggregate portfolio data
// across multiple IBKR accounts or gateways simultaneously.
//
// This example requires paper trading credentials. Set the environment variables
// before running:
//
//	IBKR_GATEWAY=https://localhost:5000 \
//	IBKR_USERNAME=username1 \
//	IBKR_PASSWORD=password1 \
//	IBKR_GATEWAY2=https://localhost:5000 \
//	IBKR_USERNAME2=username2 \
//	IBKR_PASSWORD2=password2 \
//	  go run ./examples/multi-account
//
// For a single gateway with multiple linked accounts, MultiClient can be used
// with clients pointing to the same gateway — each client will receive different
// accounts via Session.Initialize.
//
// All operations are READ-ONLY: no orders are submitted and no positions are modified.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

func main() {
	gw1 := os.Getenv("IBKR_GATEWAY")
	user1 := os.Getenv("IBKR_USERNAME")
	pass1 := os.Getenv("IBKR_PASSWORD")

	gw2 := os.Getenv("IBKR_GATEWAY2")
	user2 := os.Getenv("IBKR_USERNAME2")
	pass2 := os.Getenv("IBKR_PASSWORD2")

	if gw1 == "" || user1 == "" || pass1 == "" {
		log.Fatal("IBKR_GATEWAY, IBKR_USERNAME, and IBKR_PASSWORD are required")
	}

	gateway2 := gw2
	username2 := user2
	password2 := pass2
	if gateway2 == "" {
		gateway2 = gw1
		username2 = user1
		password2 = pass1
	}

	makeClient := func(gw, user, pass string) *ibkr.Client {
		jar, _ := cookiejar.New(nil)
		httpCli := &http.Client{
			Jar: jar,
		}
		cli, err := ibkr.NewClient(
			ibkr.WithGatewayURL(gw),
			ibkr.WithHTTPClient(httpCli),
			ibkr.WithTickleInterval(time.Hour),
		)
		if err != nil {
			log.Fatalf("NewClient(%s): %v", gw, err)
		}
		return cli
	}

	cli1 := makeClient(gw1, user1, pass1)
	cli2 := makeClient(gateway2, username2, password2)

	mc, err := ibkr.NewMultiClient([]*ibkr.Client{cli1, cli2})
	if err != nil {
		log.Fatalf("NewMultiClient: %v", err)
	}
	defer mc.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := cli1.Session().Initialize(ctx); err != nil {
		log.Fatalf("Session.Initialize(client1): %v", err)
	}
	if err := cli2.Session().Initialize(ctx); err != nil {
		log.Fatalf("Session.Initialize(client2): %v", err)
	}

	accounts, err := mc.Accounts(ctx)
	if err != nil {
		log.Fatalf("MultiClient.Accounts: %v", err)
	}
	fmt.Printf("=== Accounts (%d total) ===\n", len(accounts))
	for _, a := range accounts {
		fmt.Printf("  %s  alias=%s\n", a.ID, a.Alias)
	}

	positions, err := mc.Positions(ctx)
	if err != nil {
		log.Fatalf("MultiClient.Positions: %v", err)
	}
	fmt.Printf("\n=== Positions (%d total) ===\n", len(positions))
	for _, p := range positions {
		fmt.Printf("  [%s] %s  qty=%s  mktVal=%s  unrealizedPnL=%s\n",
			p.AccountID, p.ContractDesc, p.Quantity, p.MktValue, p.UnrealizedPnL)
	}
}
