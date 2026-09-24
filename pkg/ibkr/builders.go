// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import "fmt"

// ---------------------------------------------------------------------------
// OrderBuilder
// ---------------------------------------------------------------------------

// OrderBuilder constructs an OrderRequest with chainable setters and eager
// validation. The zero value is not usable; create one with NewOrderBuilder.
type OrderBuilder struct {
	req      OrderRequest
	err      error
	errField string
}

// NewOrderBuilder returns a new OrderBuilder with sensible defaults:
//   - Side: SideBuy
//   - OrderType: OrderTypeLimit
//   - TimeInForce: TimeInForceDay
func NewOrderBuilder() *OrderBuilder {
	return &OrderBuilder{
		req: OrderRequest{
			Side:        SideBuy,
			OrderType:   OrderTypeLimit,
			TimeInForce: TimeInForceDay,
			OutsideRTH:  false,
			AllOrNone:   false,
		},
	}
}

func (b *OrderBuilder) setErr(field, msg string) *OrderBuilder {
	if b.err == nil {
		b.err = fmt.Errorf("OrderBuilder: %s: %s", field, msg)
		b.errField = field
	}
	return b
}

// ConID sets the contract identifier. Required for order submission.
func (b *OrderBuilder) ConID(id ConID) *OrderBuilder {
	if id == 0 {
		return b.setErr("ConID", "must be non-zero")
	}
	b.req.ConID = id
	return b
}

// Side sets the buy/sell direction. Default is SideBuy.
func (b *OrderBuilder) Side(side Side) *OrderBuilder {
	switch side {
	case SideBuy, SideSell:
		b.req.Side = side
	default:
		return b.setErr("Side", fmt.Sprintf("invalid side %q, must be BUY or SELL", side))
	}
	return b
}

// Type sets the IBKR order type. Default is OrderTypeLimit.
func (b *OrderBuilder) Type(orderType OrderType) *OrderBuilder {
	if orderType == "" {
		return b.setErr("Type", "must not be empty")
	}
	b.req.OrderType = orderType
	return b
}

// Quantity sets the order size as a decimal string. Required.
func (b *OrderBuilder) Quantity(qty string) *OrderBuilder {
	if qty == "" {
		return b.setErr("Quantity", "must not be empty")
	}
	b.req.Quantity = qty
	return b
}

// LimitPrice sets the limit price for limit orders. Optional.
func (b *OrderBuilder) LimitPrice(price string) *OrderBuilder {
	b.req.LimitPrice = price
	return b
}

// StopPrice sets the stop/auxiliary price. Optional.
func (b *OrderBuilder) StopPrice(price string) *OrderBuilder {
	b.req.StopPrice = price
	return b
}

// TimeInForce sets the time-in-force policy. Default is TimeInForceDay.
func (b *OrderBuilder) TimeInForce(tif TimeInForce) *OrderBuilder {
	if tif == "" {
		return b.setErr("TimeInForce", "must not be empty")
	}
	b.req.TimeInForce = tif
	return b
}

// OutsideRTH allows execution outside regular trading hours.
func (b *OrderBuilder) OutsideRTH(v bool) *OrderBuilder {
	b.req.OutsideRTH = v
	return b
}

// AllOrNone requires the whole order to fill at once.
func (b *OrderBuilder) AllOrNone(v bool) *OrderBuilder {
	b.req.AllOrNone = v
	return b
}

// ClientOrderID sets an optional caller-supplied order reference.
func (b *OrderBuilder) ClientOrderID(id string) *OrderBuilder {
	b.req.ClientOrderID = id
	return b
}

// ParentID sets the parent OrderID for bracket orders.
func (b *OrderBuilder) ParentID(id string) *OrderBuilder {
	b.req.ParentID = id
	return b
}

// IsSingleGroup marks the order as part of an OCA group.
func (b *OrderBuilder) IsSingleGroup() *OrderBuilder {
	b.req.IsSingleGroup = true
	return b
}

// Build validates and returns the OrderRequest. Returns an error if any
// required field is missing or any value is invalid.
func (b *OrderBuilder) Build() (*OrderRequest, error) {
	if b.err != nil {
		return nil, b.err
	}
	if b.req.ConID == 0 {
		return nil, fmt.Errorf("OrderBuilder: ConID: required field not set")
	}
	if b.req.Quantity == "" {
		return nil, fmt.Errorf("OrderBuilder: Quantity: required field not set")
	}
	out := b.req
	return &out, nil
}

// ---------------------------------------------------------------------------
// ContractBuilder
// ---------------------------------------------------------------------------

// ContractBuilder constructs a Contract with chainable setters and eager
// validation. The zero value is not usable; create one with NewContractBuilder.
type ContractBuilder struct {
	contract Contract
	err      error
}

// NewContractBuilder returns a new ContractBuilder with empty defaults.
func NewContractBuilder() *ContractBuilder {
	return &ContractBuilder{}
}

func (b *ContractBuilder) setErr(field, msg string) *ContractBuilder {
	if b.err == nil {
		b.err = fmt.Errorf("ContractBuilder: %s: %s", field, msg)
	}
	return b
}

// ConID sets the contract identifier. Required.
func (b *ContractBuilder) ConID(id ConID) *ContractBuilder {
	if id == 0 {
		return b.setErr("ConID", "must be non-zero")
	}
	b.contract.ConID = id
	return b
}

// Symbol sets the ticker symbol. Required.
func (b *ContractBuilder) Symbol(sym string) *ContractBuilder {
	if sym == "" {
		return b.setErr("Symbol", "must not be empty")
	}
	b.contract.Symbol = sym
	return b
}

// Exchange sets the listing exchange. Required.
func (b *ContractBuilder) Exchange(ex string) *ContractBuilder {
	if ex == "" {
		return b.setErr("Exchange", "must not be empty")
	}
	b.contract.Exchange = ex
	return b
}

// SecType sets the security type (e.g. STK, OPT, FUT). Required.
func (b *ContractBuilder) SecType(st string) *ContractBuilder {
	if st == "" {
		return b.setErr("SecType", "must not be empty")
	}
	b.contract.InstrumentType = st
	return b
}

// Currency sets the instrument currency. Required.
func (b *ContractBuilder) Currency(cur string) *ContractBuilder {
	if cur == "" {
		return b.setErr("Currency", "must not be empty")
	}
	b.contract.Currency = cur
	return b
}

// Build validates and returns the Contract. Returns an error if any required
// field is missing.
func (b *ContractBuilder) Build() (*Contract, error) {
	if b.err != nil {
		return nil, b.err
	}
	if b.contract.ConID == 0 {
		return nil, fmt.Errorf("ContractBuilder: ConID: required field not set")
	}
	if b.contract.Symbol == "" {
		return nil, fmt.Errorf("ContractBuilder: Symbol: required field not set")
	}
	if b.contract.Exchange == "" {
		return nil, fmt.Errorf("ContractBuilder: Exchange: required field not set")
	}
	if b.contract.InstrumentType == "" {
		return nil, fmt.Errorf("ContractBuilder: SecType: required field not set")
	}
	if b.contract.Currency == "" {
		return nil, fmt.Errorf("ContractBuilder: Currency: required field not set")
	}
	out := b.contract
	return &out, nil
}

// ---------------------------------------------------------------------------
// TransferInstructionBuilder
// ---------------------------------------------------------------------------

// TransferInstructionBuilder constructs an InternalAssetTransferRequest with
// chainable setters and eager validation. The zero value is not usable; create
// one with NewTransferInstructionBuilder.
type TransferInstructionBuilder struct {
	req InternalAssetTransferRequest
	err error
}

// NewTransferInstructionBuilder returns a new TransferInstructionBuilder.
func NewTransferInstructionBuilder() *TransferInstructionBuilder {
	return &TransferInstructionBuilder{}
}

func (b *TransferInstructionBuilder) setErr(field, msg string) *TransferInstructionBuilder {
	if b.err == nil {
		b.err = fmt.Errorf("TransferInstructionBuilder: %s: %s", field, msg)
	}
	return b
}

// SourceAccountID sets the source account for the transfer. Required.
func (b *TransferInstructionBuilder) SourceAccountID(id AccountID) *TransferInstructionBuilder {
	if id == "" {
		return b.setErr("SourceAccountID", "must not be empty")
	}
	b.req.SourceAccountID = id
	return b
}

// TargetAccountID sets the target account for the transfer. Required.
func (b *TransferInstructionBuilder) TargetAccountID(id AccountID) *TransferInstructionBuilder {
	if id == "" {
		return b.setErr("TargetAccountID", "must not be empty")
	}
	b.req.TargetAccountID = id
	return b
}

// ConID sets the contract to transfer. Required.
func (b *TransferInstructionBuilder) ConID(id ConID) *TransferInstructionBuilder {
	if id == 0 {
		return b.setErr("ConID", "must be non-zero")
	}
	b.req.ConID = id
	return b
}

// Quantity sets the transfer quantity as a decimal string. Required.
func (b *TransferInstructionBuilder) Quantity(qty string) *TransferInstructionBuilder {
	if qty == "" {
		return b.setErr("Quantity", "must not be empty")
	}
	b.req.TransferQuantity = qty
	return b
}

// Price sets the optional transfer price.
func (b *TransferInstructionBuilder) Price(price *string) *TransferInstructionBuilder {
	b.req.TransferPrice = price
	return b
}

// TradeDate sets the optional trade date.
func (b *TransferInstructionBuilder) TradeDate(date *string) *TransferInstructionBuilder {
	b.req.TradeDate = date
	return b
}

// SettleDate sets the optional settle date.
func (b *TransferInstructionBuilder) SettleDate(date *string) *TransferInstructionBuilder {
	b.req.SettleDate = date
	return b
}

// ClientInstructionID sets the optional caller-supplied instruction reference.
func (b *TransferInstructionBuilder) ClientInstructionID(id string) *TransferInstructionBuilder {
	b.req.ClientInstructionID = id
	return b
}

// Build validates and returns the InternalAssetTransferRequest. Returns an
// error if any required field is missing.
func (b *TransferInstructionBuilder) Build() (*InternalAssetTransferRequest, error) {
	if b.err != nil {
		return nil, b.err
	}
	if b.req.SourceAccountID == "" {
		return nil, fmt.Errorf("TransferInstructionBuilder: SourceAccountID: required field not set")
	}
	if b.req.TargetAccountID == "" {
		return nil, fmt.Errorf("TransferInstructionBuilder: TargetAccountID: required field not set")
	}
	if b.req.ConID == 0 {
		return nil, fmt.Errorf("TransferInstructionBuilder: ConID: required field not set")
	}
	if b.req.TransferQuantity == "" {
		return nil, fmt.Errorf("TransferInstructionBuilder: Quantity: required field not set")
	}
	out := b.req
	return &out, nil
}
