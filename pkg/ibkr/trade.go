// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/shing1211/ibkrapi4go/client"
	"github.com/shing1211/ibkrapi4go/internal"
)

// OrderRequest describes an order to submit or modify. Monetary and quantity
// fields are decimal strings and are sent verbatim (ADR 0008). An empty optional
// field is omitted from the request.
type OrderRequest struct {
	// ConID is the contract identifier (required for submit).
	ConID ConID
	// Side is BUY or SELL.
	Side Side
	// Quantity is the order size as a decimal string; may be fractional.
	Quantity string
	// OrderType is the IBKR order type, e.g. OrderTypeLimit.
	OrderType OrderType
	// LimitPrice is the limit price; empty for market orders.
	LimitPrice string
	// StopPrice is the stop/auxiliary price; empty when unused.
	StopPrice string
	// TimeInForce is the order's time-in-force; empty uses the gateway default.
	TimeInForce TimeInForce
	// OutsideRTH allows execution outside regular trading hours.
	OutsideRTH bool
	// AllOrNone requires the whole order to fill at once.
	AllOrNone bool
	// ClientOrderID is an optional caller-supplied order reference.
	ClientOrderID string
}

func (r OrderRequest) toJSON() orderTicketJSON {
	t := orderTicketJSON{
		ConID:         r.ConID,
		Side:          string(r.Side),
		Quantity:      r.Quantity,
		OrderType:     string(r.OrderType),
		Price:         r.LimitPrice,
		AuxPrice:      r.StopPrice,
		TimeInForce:   string(r.TimeInForce),
		OutsideRTH:    r.OutsideRTH,
		AllOrNone:     r.AllOrNone,
		ClientOrderID: r.ClientOrderID,
	}
	return t
}

// Reply is an order reply message that must be confirmed before the order is
// accepted.
type Reply struct {
	// ID is the reply id used with Confirm.
	ID string
	// Messages are the human-readable warning texts.
	Messages []string
	// MessageIDs categorize the messages.
	MessageIDs []string
}

// SubmitResult is the outcome of a submit/confirm/modify call. When Replies is
// non-empty the order is not yet accepted and Confirm must be called.
type SubmitResult struct {
	// OrderID is set once the order is accepted.
	OrderID string
	// Status is the IBKR order status when accepted.
	Status string
	// Replies are pending confirmation messages; non-empty means not accepted.
	Replies []Reply
}

// Accepted reports whether the order was accepted (no confirmation pending).
func (r *SubmitResult) Accepted() bool { return len(r.Replies) == 0 && r.OrderID != "" }

// Order is a working or recently completed order.
type Order struct {
	OrderID           string
	AccountID         AccountID
	ConID             ConID
	Ticker            string
	OrderType         string
	Side              string
	Status            string
	TimeInForce       string
	Size              string
	FilledQuantity    string
	RemainingQuantity string
	Price             string
	AveragePrice      string
	OrderDescription  string
	LastExecutionTime string
}

// OrderStatus is the status of a single order.
type OrderStatus struct {
	OrderID           string
	Status            string
	ConID             ConID
	Side              string
	OrderType         string
	TimeInForce       string
	FilledQuantity    string
	RemainingQuantity string
	AveragePrice      string
	// Fields holds every scalar field returned by the gateway, stringified.
	Fields map[string]string
}

// Trade is an executed trade in the trade history.
type Trade struct {
	OrderID     string
	ExecutionID string
	AccountID   AccountID
	ConID       ConID
	Symbol      string
	Side        string
	Size        string
	Price       string
	Commission  string
	NetAmount   string
	TradeTime   string
}

// WhatIfResult is the margin-impact preview for an order.
type WhatIfResult struct {
	// Amount maps margin-impact keys (e.g. "initial", "maintenance") to decimal
	// strings.
	Amount map[string]string
	// Fields holds every scalar field returned by the gateway, stringified.
	Fields map[string]string
}

// Submit places an order. If the returned SubmitResult has pending Replies the
// order is not yet accepted; call Confirm. Submit is never retried: two calls
// create two orders (ADR 0009).
func (m *TradeManager) Submit(ctx context.Context, account AccountID, req OrderRequest) (*SubmitResult, error) {
	const op = "Trade.Submit"
	body, err := json.Marshal(ordersSubmissionJSON{Orders: []orderTicketJSON{req.toJSON()}})
	if err != nil {
		return nil, &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.mutate(ctx, op, func() (*http.Response, error) {
		return m.client.generated.SubmitNewOrderWithBody(ctx, string(account),
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return nil, err
	}
	result, err := m.parseSubmitResult(resp, op)
	if err != nil {
		if errors.Is(err, ErrOrderRejected) {
			m.countOrder(ctx, internal.MetricOrdersRejected, 1)
		}
		return nil, err
	}
	m.countOrder(ctx, internal.MetricOrdersSubmitted, 1)
	return result, nil
}

// Confirm answers a pending order reply. Pass confirmed=true to proceed; false
// cancels the pending order.
func (m *TradeManager) Confirm(ctx context.Context, replyID string, confirmed bool) (*SubmitResult, error) {
	const op = "Trade.Confirm"
	body, _ := json.Marshal(confirmReplyJSON{Confirmed: confirmed})
	resp, err := m.mutate(ctx, op, func() (*http.Response, error) {
		return m.client.generated.ConfirmOrderReplyWithBody(ctx, replyID,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return nil, err
	}
	result, err := m.parseSubmitResult(resp, op)
	if err != nil {
		if errors.Is(err, ErrOrderRejected) {
			m.countOrder(ctx, internal.MetricOrdersRejected, 1)
		}
		return nil, err
	}
	if result.Accepted() {
		m.countOrder(ctx, internal.MetricOrdersConfirmed, 1)
	}
	return result, nil
}

// WhatIf previews the margin impact of an order without submitting it.
func (m *TradeManager) WhatIf(ctx context.Context, account AccountID, req OrderRequest) (*WhatIfResult, error) {
	const op = "Trade.WhatIf"
	body, err := json.Marshal(ordersSubmissionJSON{Orders: []orderTicketJSON{req.toJSON()}})
	if err != nil {
		return nil, &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.PreviewMarginImpactWithBody(ctx, string(account),
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := &WhatIfResult{Fields: map[string]string{}}
	for k, v := range raw {
		if k == "amount" {
			var amounts map[string]json.Number
			if json.Unmarshal(v, &amounts) == nil {
				out.Amount = numbersToStrings(amounts)
			}
			continue
		}
		out.Fields[k] = rawScalarString(v)
	}
	return out, nil
}

// Modify changes an existing open order. It is never retried.
func (m *TradeManager) Modify(ctx context.Context, account AccountID, orderID string, req OrderRequest) (*SubmitResult, error) {
	const op = "Trade.Modify"
	body, err := json.Marshal(req.toJSON())
	if err != nil {
		return nil, &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.mutate(ctx, op, func() (*http.Response, error) {
		return m.client.generated.ModifyOpenOrderWithBody(ctx, string(account), orderID,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return nil, err
	}
	result, err := m.parseSubmitResult(resp, op)
	if err != nil {
		if errors.Is(err, ErrOrderRejected) {
			m.countOrder(ctx, internal.MetricOrdersRejected, 1)
		}
		return nil, err
	}
	m.countOrder(ctx, internal.MetricOrdersModified, 1)
	return result, nil
}

// Cancel cancels an open order. It is never retried.
func (m *TradeManager) Cancel(ctx context.Context, account AccountID, orderID string) error {
	const op = "Trade.Cancel"
	resp, err := m.mutate(ctx, op, func() (*http.Response, error) {
		return m.client.generated.CancelOpenOrder(ctx, string(account), orderID, nil)
	})
	if err != nil {
		if errors.Is(err, ErrOrderRejected) {
			m.countOrder(ctx, internal.MetricOrdersRejected, 1)
		}
		return err
	}
	resp.Body.Close()
	m.countOrder(ctx, internal.MetricOrdersCancelled, 1)
	return nil
}

// countOrder increments an order metric when metrics are configured.
func (m *TradeManager) countOrder(ctx context.Context, name string, delta int64) {
	if sink := m.client.metricsSink(); sink != nil {
		sink.Counter(ctx, name, delta)
	}
}

// OpenOrders returns the orders currently working or completed in this session.
func (m *TradeManager) OpenOrders(ctx context.Context) ([]Order, error) {
	const op = "Trade.OpenOrders"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetOpenOrders(ctx, nil)
	})
	if err != nil {
		return nil, err
	}
	var raw struct {
		Orders []map[string]json.RawMessage `json:"orders"`
	}
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]Order, 0, len(raw.Orders))
	for _, o := range raw.Orders {
		out = append(out, Order{
			OrderID:           rawToString(o, "orderId"),
			AccountID:         AccountID(firstNonEmpty(rawToString(o, "account"), rawToString(o, "acct"))),
			ConID:             ConID(jsonNumberToInt(jsonNumber(o["conid"]))),
			Ticker:            rawToString(o, "ticker"),
			OrderType:         rawToString(o, "orderType"),
			Side:              rawToString(o, "side"),
			Status:            rawToString(o, "status"),
			TimeInForce:       rawToString(o, "timeInForce"),
			Size:              rawToString(o, "totalSize"),
			FilledQuantity:    rawToString(o, "filledQuantity"),
			RemainingQuantity: rawToString(o, "remainingQuantity"),
			Price:             rawToString(o, "price"),
			AveragePrice:      rawToString(o, "avgPrice"),
			OrderDescription:  rawToString(o, "orderDesc"),
			LastExecutionTime: rawToString(o, "lastExecutionTime_r"),
		})
	}
	return out, nil
}

// OrderStatus returns the status of a single order.
func (m *TradeManager) OrderStatus(ctx context.Context, orderID string) (*OrderStatus, error) {
	const op = "Trade.OrderStatus"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetOrderStatus(ctx, orderID)
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	st := &OrderStatus{
		OrderID:           firstNonEmpty(rawToString(raw, "order_id"), rawToString(raw, "orderId")),
		Status:            firstNonEmpty(rawToString(raw, "order_status"), rawToString(raw, "status")),
		ConID:             ConID(jsonNumberToInt(jsonNumber(raw["conid"]))),
		Side:              rawToString(raw, "side"),
		OrderType:         rawToString(raw, "order_type"),
		TimeInForce:       rawToString(raw, "tif"),
		FilledQuantity:    rawToString(raw, "filled_quantity"),
		RemainingQuantity: rawToString(raw, "remaining_quantity"),
		AveragePrice:      rawToString(raw, "average_price"),
		Fields:            map[string]string{},
	}
	for k, v := range raw {
		st.Fields[k] = rawScalarString(v)
	}
	return st, nil
}

// Trades returns the trade history for the session. days <= 0 uses the gateway
// default window.
func (m *TradeManager) Trades(ctx context.Context, days int) ([]Trade, error) {
	const op = "Trade.Trades"
	var params *client.GetTradeHistoryParams
	if days > 0 {
		d := int64(days)
		params = &client.GetTradeHistoryParams{Days: &d}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetTradeHistory(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]Trade, 0, len(raw))
	for _, t := range raw {
		out = append(out, Trade{
			OrderID:     rawToString(t, "order_id"),
			ExecutionID: rawToString(t, "execution_id"),
			AccountID:   AccountID(rawToString(t, "account")),
			ConID:       ConID(jsonNumberToInt(jsonNumber(t["conid"]))),
			Symbol:      rawToString(t, "symbol"),
			Side:        rawToString(t, "side"),
			Size:        rawToString(t, "size"),
			Price:       rawToString(t, "price"),
			Commission:  rawToString(t, "commission"),
			NetAmount:   rawToString(t, "net_amount"),
			TradeTime:   rawToString(t, "trade_time_r"),
		})
	}
	return out, nil
}

// SuppressOrderReplies suppresses specific order reply messages so they are
// no longer returned as pending confirmations. Pass the message IDs from a
// Reply.MessageIDs slice to suppress them.
func (m *TradeManager) SuppressOrderReplies(ctx context.Context, messageIDs []string) error {
	const op = "Trade.SuppressOrderReplies"
	ids := make([]client.SuppressOrderRepliesJSONBodyMessageIds, len(messageIDs))
	for i, id := range messageIDs {
		ids[i] = client.SuppressOrderRepliesJSONBodyMessageIds(id)
	}
	body := client.SuppressOrderRepliesJSONRequestBody{
		MessageIds: &ids,
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.SuppressOrderReplies(ctx, body)
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// ResetOrderSuppression clears all suppressed order reply messages, restoring
// them to the default behaviour where they are returned as pending
// confirmations.
func (m *TradeManager) ResetOrderSuppression(ctx context.Context) error {
	const op = "Trade.ResetOrderSuppression"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.ResetOrderSuppression(ctx)
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// --- internals --------------------------------------------------------------

// mutate runs an order-mutating call exactly once. Transport timeouts are
// reported as an ambiguous outcome so callers reconcile rather than resubmit
// (ADR 0009).
func (m *TradeManager) mutate(ctx context.Context, op string, fn func() (*http.Response, error)) (*http.Response, error) {
	resp, err := m.client.netDo(ctx, op, fn)
	if err != nil {
		if isAmbiguous(err) {
			return nil, &Error{
				Op:      op,
				Code:    "ambiguous",
				Message: "ambiguous outcome; the order may or may not have been accepted — reconcile via Trade().OpenOrders before retrying",
				Err:     err,
			}
		}
		return nil, err
	}
	return resp, nil
}

// parseSubmitResult reads a submit/confirm/modify response, which is either an
// array of reply items (confirmation pending) or an array of accepted orders.
func (m *TradeManager) parseSubmitResult(resp *http.Response, op string) (*SubmitResult, error) {
	var items []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &items); err != nil {
		return nil, err
	}
	out := &SubmitResult{}
	for _, item := range items {
		if errMsg := rawToString(item, "error"); errMsg != "" {
			return nil, &Error{Op: op, Code: rawToString(item, "error"), Message: errMsg, Err: ErrOrderRejected}
		}
		if id := rawToString(item, "id"); id != "" {
			out.Replies = append(out.Replies, Reply{
				ID:         id,
				Messages:   rawToStringSlice(item, "message"),
				MessageIDs: rawToStringSlice(item, "messageIds"),
			})
			continue
		}
		if oid := firstNonEmpty(rawToString(item, "order_id"), rawToString(item, "orderId")); oid != "" {
			out.OrderID = oid
			out.Status = firstNonEmpty(rawToString(item, "order_status"), rawToString(item, "status"))
		}
	}
	if out.OrderID == "" && len(out.Replies) == 0 {
		return nil, &Error{Op: op, Message: "unexpected order response", Err: ErrOrderRejected}
	}
	return out, nil
}

// isAmbiguous reports whether err is a timeout or context deadline, which means
// the request may have reached the server.
func isAmbiguous(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var te interface{ Timeout() bool }
	if errors.As(err, &te) && te.Timeout() {
		return true
	}
	return false
}

// orderTicketJSON is the wire shape for a single order. Money/quantity are
// strings to preserve precision (ADR 0008).
type orderTicketJSON struct {
	ConID         ConID     `json:"conid"`
	Side          string    `json:"side"`
	Quantity      string    `json:"quantity"`
	OrderType     string    `json:"orderType"`
	Price         string    `json:"price,omitempty"`
	AuxPrice      string    `json:"auxPrice,omitempty"`
	TimeInForce   string    `json:"tif,omitempty"`
	OutsideRTH    bool      `json:"outsideRTH,omitempty"`
	AllOrNone     bool      `json:"allOrNone,omitempty"`
	ClientOrderID string    `json:"cOID,omitempty"`
	AccountID     AccountID `json:"acctId,omitempty"`
}

type ordersSubmissionJSON struct {
	Orders []orderTicketJSON `json:"orders"`
}

type confirmReplyJSON struct {
	Confirmed bool `json:"confirmed"`
}

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// jsonNumber returns n as a json.Number, or empty when absent.
func jsonNumber(raw json.RawMessage) json.Number {
	if len(raw) == 0 {
		return ""
	}
	var n json.Number
	if json.Unmarshal(raw, &n) == nil {
		return n
	}
	return ""
}
