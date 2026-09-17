// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Command orders demonstrates the two-step order submission flow (Submit then
// Confirm) using the mock gateway, so no real account is needed.
//
// This example shows a WhatIf dry-run, a simulated Submit, and a Confirm.
// Because the mock gateway has no live order state, the Confirm step uses the
// replyID returned by Submit.
//
// Start the gateway in another terminal:
//
//	go run ./cmd/ibkr-mock-gateway
//
// Then run this example:
//
//	go run ./examples/orders
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

	req := ibkr.OrderRequest{
		ConID:       8314,
		Side:        ibkr.SideBuy,
		Quantity:    "100",
		OrderType:   ibkr.OrderTypeLimit,
		LimitPrice:  "1.00",
		TimeInForce: ibkr.TimeInForceDay,
	}

	fmt.Println("=== WhatIf dry-run ===")
	wi, err := cli.Trade().WhatIf(ctx, acct, req)
	if err != nil {
		log.Fatalf("Trade.WhatIf: %v", err)
	}
	fmt.Printf("  initial margin impact=%s  maintenance=%s\n",
		wi.Amount["initial"], wi.Amount["maintenance"])
	if v, ok := wi.Fields["marginIsInsufficient"]; ok {
		fmt.Printf("  marginIsInsufficient=%s\n", v)
	}

	fmt.Println("\n=== Submit order ===")
	result, err := cli.Trade().Submit(ctx, acct, req)
	if err != nil {
		log.Fatalf("Trade.Submit: %v", err)
	}
	fmt.Printf("  submitted  orderId=%s  status=%s  replies=%d\n",
		result.OrderID, result.Status, len(result.Replies))

	if len(result.Replies) == 0 {
		fmt.Println("  (no pending replies; order accepted without confirmation step)")
	} else {
		replyID := result.Replies[0].ID
		fmt.Printf("  pending replyId=%s  messages=%v\n", replyID, result.Replies[0].Messages)
		confirmed, err := cli.Trade().Confirm(ctx, replyID, true)
		if err != nil {
			log.Fatalf("Trade.Confirm: %v", err)
		}
		fmt.Printf("  confirmed  orderId=%s  status=%s\n",
			confirmed.OrderID, confirmed.Status)
	}

	fmt.Println("\n=== Confirm order ===")
	if len(result.Replies) == 0 {
		fmt.Println("  no pending replies; order already accepted")
	} else {
		replyID := result.Replies[0].ID
		fmt.Printf("  pending replyId=%s messages=%v\n", replyID, result.Replies[0].Messages)
		confirmed, err := cli.Trade().Confirm(ctx, replyID, true)
		if err != nil {
			log.Fatalf("Trade.Confirm: %v", err)
		}
		fmt.Printf("  confirmed  orderId=%s  status=%s\n",
			confirmed.OrderID, confirmed.Status)
	}

	openOrders, err := cli.Trade().OpenOrders(ctx)
	if err != nil {
		log.Fatalf("Trade.OpenOrders: %v", err)
	}
	fmt.Printf("\n=== Open orders: %d ===\n", len(openOrders))
	for _, o := range openOrders {
		fmt.Printf("  orderId=%s  conId=%d  side=%s  qty=%s  orderType=%s  status=%s\n",
			o.OrderID, o.ConID, o.Side, o.RemainingQuantity, o.OrderType, o.Status)
	}
}
