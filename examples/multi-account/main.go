// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Command multi-account demonstrates using MultiClient to aggregate portfolio data
// across multiple IBKR accounts or gateways simultaneously.
//
// It requires one or two running IBKR Client Portal Gateways that you have
// already authenticated in a browser (there is no username/password login — see
// docs/GATEWAY-SETUP.md and docs/AUTH.md). Point it at non-default gateways with
// the optional IBKR_GATEWAY and IBKR_GATEWAY2 variables:
//
//	IBKR_GATEWAY=https://localhost:5000 \
//	IBKR_GATEWAY2=https://localhost:5001 \
//	  go run ./examples/multi-account
//
// IBKR_GATEWAY2 defaults to IBKR_GATEWAY. Each client keeps its own cookie jar,
// so two clients may target the same gateway to represent separate browser
// sessions. For a single gateway with multiple linked accounts, MultiClient can
// be used with clients pointing to the same gateway — each client will receive
// different accounts via Session.Initialize.
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
	gw2 := os.Getenv("IBKR_GATEWAY2")
	if gw2 == "" {
		gw2 = gw1
	}

	makeClient := func(gw string) *ibkr.Client {
		opts := []ibkr.Option{ibkr.WithTickleInterval(time.Hour)}
		if gw != "" {
			// A dedicated cookie jar keeps each client's gateway session separate.
			jar, _ := cookiejar.New(nil)
			opts = append(opts,
				ibkr.WithGatewayURL(gw),
				ibkr.WithHTTPClient(&http.Client{Jar: jar}),
			)
		}
		cli, err := ibkr.NewClient(opts...)
		if err != nil {
			log.Fatalf("NewClient: %v", err)
		}
		return cli
	}

	cli1 := makeClient(gw1)
	cli2 := makeClient(gw2)

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
