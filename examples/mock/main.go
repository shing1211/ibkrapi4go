// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Command mock connects the SDK to a local ibkr-mock-gateway, initializes a
// brokerage session, and lists the mock accounts.
//
// Start the gateway in another terminal:
//
//	go run ./cmd/ibkr-mock-gateway
//
// Then run this example:
//
//	go run ./examples/mock
//
// The gateway base URL comes from IBKR_GATEWAY_URL and defaults to the mock's
// default address, http://localhost:5001. No real IBKR account is contacted.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

// defaultGatewayURL matches cmd/ibkr-mock-gateway's default -addr.
const defaultGatewayURL = "http://localhost:5001"

func main() {
	gateway := os.Getenv("IBKR_GATEWAY_URL")
	if gateway == "" {
		gateway = defaultGatewayURL
	}

	cli, err := ibkr.NewClient(
		ibkr.WithGatewayURL(gateway),
		ibkr.WithTickleInterval(time.Hour),
	)
	if err != nil {
		log.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	ctx := context.Background()
	if err := cli.Session().Initialize(ctx); err != nil {
		log.Fatalf("Session.Initialize: %v", err)
	}
	fmt.Printf("session: %s (gateway %s)\n", cli.Session().State(), cli.GatewayURL())

	accounts, err := cli.Account().List(ctx)
	if err != nil {
		log.Fatalf("Account.List: %v", err)
	}
	fmt.Printf("accounts: %d\n", len(accounts))
	for _, a := range accounts {
		fmt.Printf("  %s (%s)\n", a.ID, a.Alias)
	}
}
