// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import "time"

// WSOrderStatus is the IBKR order status string values from WebSocket frames.
type WSOrderStatus string

const (
	WSOrderStatusPreSubmitted  WSOrderStatus = "PreSubmitted"
	WSOrderStatusSubmitted     WSOrderStatus = "Submitted"
	WSOrderStatusFilled        WSOrderStatus = "Filled"
	WSOrderStatusPartialFill   WSOrderStatus = "PartialFill"
	WSOrderStatusCancelled     WSOrderStatus = "Cancelled"
	WSOrderStatusPendingCancel WSOrderStatus = "PendingCancel"
	WSOrderStatusPendingSubmit WSOrderStatus = "PendingSubmit"
	WSOrderStatusInactive      WSOrderStatus = "Inactive"
	WSOrderStatusWarnState     WSOrderStatus = "WarnState"
)

// OrderEvent is a typed order status update from the "sor" WebSocket frame.
// Zero-value fields indicate the payload did not contain that field.
type OrderEvent struct {
	Conid         int64
	OrderID       int64
	ClientOrderID string
	Account       string
	Status        WSOrderStatus
	Side          string
	OrderType     string
	TIF           string
	Size          string
	CumFill       string
	AveragePrice  string
	TotalSize     string
	OrderTime     string
	Received      time.Time
}

// NotificationEvent is a typed notification from the "ntf" WebSocket frame.
type NotificationEvent struct {
	Topic    string
	Message  string
	Received time.Time
}

// UserMessageEvent is a typed user message from the "usr" WebSocket frame.
type UserMessageEvent struct {
	Message  string
	Received time.Time
}
