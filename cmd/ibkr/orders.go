// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	ibkr "github.com/shing1211/ibkrapi4go/pkg/ibkr"
)

func runOrders(e *env, args []string) error {
	if len(args) == 0 {
		return runOrdersList(e)
	}

	switch args[0] {
	case "list":
		return runOrdersList(e)
	case "submit":
		return runOrdersSubmit(e, args[1:])
	case "cancel":
		return runOrdersCancel(e, args[1:])
	case "-h", "--help", "help":
		_, _ = fmt.Fprintf(e.stderr, `Usage: ibkr orders <subcommand> [flags]

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

func runOrdersList(e *env) error {
	cli, err := e.newClient()
	if err != nil {
		return err
	}
	defer func() { _ = cli.Close() }()

	if err := cli.Session().Initialize(ctx()); err != nil {
		_, _ = fmt.Fprintf(e.stderr, "error: session initialize: %v\n", err)
		return err
	}

	orders, err := cli.Trade().OpenOrders(ctx())
	if err != nil {
		_, _ = fmt.Fprintf(e.stderr, "error: list orders: %v\n", err)
		return err
	}

	if len(orders) == 0 {
		ibkrPrintln(e, "no open orders")
		return nil
	}

	w := e.stdout
	_, _ = fmt.Fprintf(w, "%-12s %-10s %-10s %-6s %-8s %-8s %-12s %-12s %-8s %-6s\n",
		"ORDER_ID", "ACCOUNT", "CONID", "SIDE", "TYPE", "SIZE", "PRICE", "AVG_PRICE", "STATUS", "TIF")
	_, _ = fmt.Fprintf(w, "%s\n", strings.Repeat("-", 112))
	for _, o := range orders {
		_, _ = fmt.Fprintf(w, "%-12s %-10s %-10d %-6s %-8s %-8s %-12s %-12s %-8s %-6s\n",
			o.OrderID, o.AccountID, o.ConID, o.Side, o.OrderType,
			o.Size, o.Price, o.AveragePrice, o.Status, o.TimeInForce)
	}
	return nil
}

func runOrdersSubmit(e *env, args []string) error {

	var conid int
	var side, qty, orderType, price, stopPrice, tif, coid string
	var outsideRTH bool

	for i := 0; i < len(args); i++ {
		switch argAt(args, i) {
		case "-conid", "--conid":
			if v, ok := argValue(args, i); ok {
				conid, _ = strconv.Atoi(v)
				i++
			}
		case "-side", "--side":
			if v, ok := argValue(args, i); ok {
				side = strings.ToUpper(v)
				i++
			}
		case "-qty", "--qty":
			if v, ok := argValue(args, i); ok {
				qty = v
				i++
			}
		case "-type", "--type":
			if v, ok := argValue(args, i); ok {
				orderType = strings.ToUpper(v)
				i++
			}
		case "-price", "--price":
			if v, ok := argValue(args, i); ok {
				price = v
				i++
			}
		case "-stop", "--stop":
			if v, ok := argValue(args, i); ok {
				stopPrice = v
				i++
			}
		case "-tif", "--tif":
			if v, ok := argValue(args, i); ok {
				tif = strings.ToUpper(v)
				i++
			}
		case "-rth", "--rth":
			outsideRTH = true
		case "-coid", "--coid":
			if v, ok := argValue(args, i); ok {
				coid = v
				i++
			}
		case "-h", "--help":
			_, _ = fmt.Fprintf(e.stderr, `Usage: ibkr orders submit [flags]

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

	cli, err := e.newClient()
	if err != nil {
		return err
	}
	defer func() { _ = cli.Close() }()

	if err := cli.Session().Initialize(ctx()); err != nil {
		_, _ = fmt.Fprintf(e.stderr, "error: session initialize: %v\n", err)
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
		_, _ = fmt.Fprintf(e.stderr, "error: submit order: %v\n", err)
		return err
	}

	enc := json.NewEncoder(e.stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func runOrdersCancel(e *env, args []string) error {

	var orderID string

	for i := 0; i < len(args); i++ {
		switch argAt(args, i) {
		case "-orderid", "--orderid", "-order-id", "--order-id":
			if v, ok := argValue(args, i); ok {
				orderID = v
				i++
			}
		case "-h", "--help":
			_, _ = fmt.Fprintf(e.stderr, "Usage: ibkr orders cancel -orderid ORDER_ID\n\nCancel an open order.\n")
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

	cli, err := e.newClient()
	if err != nil {
		return err
	}
	defer func() { _ = cli.Close() }()

	if err := cli.Session().Initialize(ctx()); err != nil {
		_, _ = fmt.Fprintf(e.stderr, "error: session initialize: %v\n", err)
		return err
	}

	if err := cli.Trade().Cancel(ctx(), acct, orderID); err != nil {
		_, _ = fmt.Fprintf(e.stderr, "error: cancel order: %v\n", err)
		return err
	}

	fmt.Printf("order %s cancelled\n", orderID)
	return nil
}
