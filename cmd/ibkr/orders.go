// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

func runOrders() error {
	args := os.Args[2:]
	if len(args) == 0 {
		return runOrdersList()
	}

	switch args[0] {
	case "list":
		os.Args = append(os.Args[:2], args[1:]...)
		return runOrdersList()
	case "submit":
		os.Args = append(os.Args[:2], args[1:]...)
		return runOrdersSubmit()
	case "cancel":
		os.Args = append(os.Args[:2], args[1:]...)
		return runOrdersCancel()
	case "-h", "--help", "help":
		fmt.Fprintf(os.Stderr, `Usage: ibkr orders <subcommand> [flags]

Subcommands:
  list      List open orders
  submit    Submit a new order
  cancel    Cancel an open order
`)
		return nil
	default:
		return fmt.Errorf("unknown orders subcommand %q", args[0])
	}
}

func runOrdersList() error {
	cli, err := newClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	if err := cli.Session().Initialize(ctx()); err != nil {
		fmt.Fprintf(os.Stderr, "error: session initialize: %v\n", err)
		return err
	}

	orders, err := cli.Trade().OpenOrders(ctx())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: list orders: %v\n", err)
		return err
	}

	if len(orders) == 0 {
		fmt.Println("no open orders")
		return nil
	}

	w := os.Stdout
	fmt.Fprintf(w, "%-12s %-10s %-10s %-6s %-8s %-8s %-12s %-12s %-8s %-6s\n",
		"ORDER_ID", "ACCOUNT", "CONID", "SIDE", "TYPE", "SIZE", "PRICE", "AVG_PRICE", "STATUS", "TIF")
	fmt.Fprintf(w, "%s\n", strings.Repeat("-", 112))
	for _, o := range orders {
		fmt.Fprintf(w, "%-12s %-10s %-10d %-6s %-8s %-8s %-12s %-12s %-8s %-6s\n",
			o.OrderID, o.AccountID, o.ConID, o.Side, o.OrderType,
			o.Size, o.Price, o.AveragePrice, o.Status, o.TimeInForce)
	}
	return nil
}

func runOrdersSubmit() error {
	args := os.Args[2:]

	var conid int
	var side, qty, orderType, price, stopPrice, tif, coid string
	var outsideRTH bool

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-conid", "--conid":
			if i+1 < len(args) {
				conid, _ = strconv.Atoi(args[i+1])
				i++
			}
		case "-side", "--side":
			if i+1 < len(args) {
				side = strings.ToUpper(args[i+1])
				i++
			}
		case "-qty", "--qty":
			if i+1 < len(args) {
				qty = args[i+1]
				i++
			}
		case "-type", "--type":
			if i+1 < len(args) {
				orderType = strings.ToUpper(args[i+1])
				i++
			}
		case "-price", "--price":
			if i+1 < len(args) {
				price = args[i+1]
				i++
			}
		case "-stop", "--stop":
			if i+1 < len(args) {
				stopPrice = args[i+1]
				i++
			}
		case "-tif", "--tif":
			if i+1 < len(args) {
				tif = strings.ToUpper(args[i+1])
				i++
			}
		case "-rth", "--rth":
			outsideRTH = true
		case "-coid", "--coid":
			if i+1 < len(args) {
				coid = args[i+1]
				i++
			}
		case "-h", "--help":
			fmt.Fprintf(os.Stderr, `Usage: ibkr orders submit [flags]

Flags:
  -conid    INT     Contract ID (required)
  -side     SIDE    BUY or SELL (required)
  -qty      QTY     Order quantity (required)
  -type     TYPE    Order type: MKT, LMT, STP, STOP_LIMIT, MOC, LOC (default: LMT)
  -price    PRICE   Limit price
  -stop     PRICE   Stop price
  -tif      TIF     Time-in-force: DAY, GTC, IOC, OPG (default: DAY)
  -rth              Allow outside regular trading hours
  -coid     ID      Client order ID
`)
			return nil
		}
	}

	if conid == 0 {
		return fmt.Errorf("--conid is required")
	}
	if side == "" {
		return fmt.Errorf("--side is required (BUY or SELL)")
	}
	if qty == "" {
		return fmt.Errorf("--qty is required")
	}

	acct, err := mustAccount("")
	if err != nil {
		return err
	}

	cli, err := newClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	if err := cli.Session().Initialize(ctx()); err != nil {
		fmt.Fprintf(os.Stderr, "error: session initialize: %v\n", err)
		return err
	}

	req := ibkr.OrderRequest{
		ConID:         ibkr.ConID(conid),
		Side:          ibkr.Side(side),
		Quantity:      qty,
		OrderType:     ibkr.OrderType(orderType),
		LimitPrice:    price,
		StopPrice:     stopPrice,
		TimeInForce:   ibkr.TimeInForce(tif),
		OutsideRTH:    outsideRTH,
		ClientOrderID: coid,
	}
	if req.OrderType == "" {
		req.OrderType = ibkr.OrderTypeLimit
	}
	if req.TimeInForce == "" {
		req.TimeInForce = ibkr.TimeInForceDay
	}

	result, err := cli.Trade().Submit(ctx(), acct, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: submit order: %v\n", err)
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func runOrdersCancel() error {
	args := os.Args[2:]

	var orderID string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-orderid", "--orderid", "-order-id", "--order-id":
			if i+1 < len(args) {
				orderID = args[i+1]
				i++
			}
		case "-h", "--help":
			fmt.Fprintf(os.Stderr, "Usage: ibkr orders cancel -orderid ORDER_ID\n\nCancel an open order.\n")
			return nil
		}
	}

	if orderID == "" {
		return fmt.Errorf("--orderid is required")
	}

	acct, err := mustAccount("")
	if err != nil {
		return err
	}

	cli, err := newClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	if err := cli.Session().Initialize(ctx()); err != nil {
		fmt.Fprintf(os.Stderr, "error: session initialize: %v\n", err)
		return err
	}

	if err := cli.Trade().Cancel(ctx(), acct, orderID); err != nil {
		fmt.Fprintf(os.Stderr, "error: cancel order: %v\n", err)
		return err
	}

	fmt.Printf("order %s cancelled\n", orderID)
	return nil
}
