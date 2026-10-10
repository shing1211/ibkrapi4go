// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Command cancel-order demonstrates submitting a resting order and then
// cancelling it against the mock gateway, so no real account is needed.
//
// Start the gateway in another terminal:
//
//	go run ./cmd/ibkr-mock-gateway
//
// Then run this example:
//
//	go run ./examples/mock/cancel-order
//
// The gateway base URL comes from IBKR_GATEWAY_URL and defaults to the mock's
// default address, http://localhost:5001.
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

	accounts, err := cli.Account().List(ctx)
	if err != nil {
		log.Fatalf("Account.List: %v", err)
	}
	if len(accounts) == 0 {
		log.Fatal("no accounts")
	}
	acct := accounts[0].ID

	// A far-from-market limit order so it rests instead of filling.
	req := ibkr.OrderRequest{
		ConID:       8314,
		Side:        ibkr.SideBuy,
		Quantity:    "100",
		OrderType:   ibkr.OrderTypeLimit,
		LimitPrice:  "1.00",
		TimeInForce: ibkr.TimeInForceDay,
	}

	fmt.Println("=== Submit order ===")
	result, err := cli.Trade().Submit(ctx, acct, req)
	if err != nil {
		log.Fatalf("Trade.Submit: %v", err)
	}
	fmt.Printf("  orderId=%s  status=%s\n", result.OrderID, result.Status)
	orderID := result.OrderID
	if orderID == "" {
		log.Fatal("no orderId returned")
	}

	fmt.Println("\n=== Cancel order ===")
	if err := cli.Trade().Cancel(ctx, acct, orderID); err != nil {
		log.Fatalf("Trade.Cancel: %v", err)
	}
	fmt.Printf("  cancel requested for orderId=%s\n", orderID)

	fmt.Println("\n=== Open orders after cancel ===")
	openOrders, err := cli.Trade().OpenOrders(ctx)
	if err != nil {
		log.Fatalf("Trade.OpenOrders: %v", err)
	}
	fmt.Printf("  %d open order(s)\n", len(openOrders))
	for _, o := range openOrders {
		fmt.Printf("  orderId=%s  status=%s\n", o.OrderID, o.Status)
	}
}
