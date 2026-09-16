// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

// ConID is an IBKR contract identifier.
type ConID int

// Field selects a market-data field in a snapshot request. See the IBKR
// market-data field reference for the full list.
type Field string

// Common market-data fields.
const (
	FieldLastPrice     Field = "31"
	FieldBidPrice      Field = "84"
	FieldAskPrice      Field = "86"
	FieldBidSize       Field = "88"
	FieldAskSize       Field = "85"
	FieldVolume        Field = "87"
	FieldOpen          Field = "7295"
	FieldHigh          Field = "70"
	FieldLow           Field = "71"
	FieldClose         Field = "7296"
	FieldChange        Field = "82"
	FieldChangePercent Field = "83"
	FieldSymbol        Field = "55"
)

// Side is the buy/sell direction of an order.
type Side string

// Order sides.
const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

// OrderType is an IBKR order type.
type OrderType string

// Common order types.
const (
	OrderTypeMarket        OrderType = "MKT"
	OrderTypeLimit         OrderType = "LMT"
	OrderTypeStop          OrderType = "STP"
	OrderTypeStopLimit     OrderType = "STOP_LIMIT"
	OrderTypeMarketOnClose OrderType = "MOC"
	OrderTypeLimitOnClose  OrderType = "LOC"
)

// TimeInForce is an order's time-in-force policy.
type TimeInForce string

// Time-in-force values.
const (
	TimeInForceDay TimeInForce = "DAY"
	TimeInForceGTC TimeInForce = "GTC"
	TimeInForceIOC TimeInForce = "IOC"
	TimeInForceOPG TimeInForce = "OPG"
)
