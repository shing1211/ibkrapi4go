// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Command reconcile-open-orders reconciles the session's open-order list against
// the per-order status endpoint, flagging any drift. This mirrors the
// reconciliation workflow described in docs/design/09 and is useful after a
// reconnect or a process restart.
//
// Start the gateway in another terminal:
//
//	go run ./cmd/ibkr-mock-gateway
//
// Then run this example:
//
//	go run ./examples/mock/reconcile-open-orders
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

	open, err := cli.Trade().OpenOrders(ctx)
	if err != nil {
		log.Fatalf("Trade.OpenOrders: %v", err)
	}

	fmt.Printf("=== Reconciling %d open order(s) ===\n", len(open))

	drift := 0
	for _, o := range open {
		st, err := cli.Trade().OrderStatus(ctx, o.OrderID)
		if err != nil {
			fmt.Printf("  orderId=%s: status lookup failed: %v\n", o.OrderID, err)
			drift++
			continue
		}
		if st.Status == o.Status &&
			st.FilledQuantity == o.FilledQuantity &&
			st.RemainingQuantity == o.RemainingQuantity {
			fmt.Printf("  orderId=%s: OK (status=%s)\n", o.OrderID, o.Status)
			continue
		}
		drift++
		fmt.Printf("  orderId=%s: DRIFT list(status=%s filled=%s remaining=%s) "+
			"status(status=%s filled=%s remaining=%s)\n",
			o.OrderID, o.Status, o.FilledQuantity, o.RemainingQuantity,
			st.Status, st.FilledQuantity, st.RemainingQuantity)
	}

	fmt.Printf("\n%d order(s) reconciled, %d drift\n", len(open), drift)
}
