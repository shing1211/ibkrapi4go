// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/shing1211/ibkrapi4go/client"
	"github.com/shing1211/ibkrapi4go/internal"
)

func strToInt(s string) int {
	v, _ := strconv.ParseInt(s, 10, 64)
	return int(v)
}

// strToDecimal converts a decimal string to float64 for the generated client
// payload. The SDK public API uses strings per ADR 0008; the internal conversion to
// float is required to produce spec-compliant JSON number wire shapes. Precision loss
// is accepted for now; the long-term fix is patching the spec to use string types.
func strToDecimal(s string) float32 {
	v, _ := strconv.ParseFloat(s, 32)
	return float32(v)
}

// strToDecimalPtr is the optional variant.
func strToDecimalPtr(s *string) *float32 {
	if s == nil {
		return nil
	}
	v := strToDecimal(*s)
	return &v
}

// fopInstructionJSON is the wire shape for external asset transfers.
// Money/quantity fields are strings to preserve precision (ADR 0008).
type fopInstructionJSON struct {
	AccountId             string `json:"accountId"`
	ClientInstructionId   int    `json:"clientInstructionId"`
	ContraBrokerAccountId string `json:"contraBrokerAccountId"`
	ContraBrokerDtcCode   string `json:"contraBrokerDtcCode"`
	Direction             string `json:"direction"`
	Quantity              string `json:"quantity"`
	TradingInstrument     any    `json:"tradingInstrument"`
}

type externalAssetTransferJSON struct {
	Instruction     fopInstructionJSON `json:"instruction"`
	InstructionType string             `json:"instructionType"`
}

// depositInstructionJSON is the wire shape for deposit fund instructions.
// Amount is a string to preserve precision (ADR 0008).
type depositInstructionJSON struct {
	AccountId             string  `json:"accountId"`
	Amount                string  `json:"amount"`
	BankInstructionMethod string  `json:"bankInstructionMethod"`
	BankInstructionName   *string `json:"bankInstructionName,omitempty"`
	ClientInstructionId   int     `json:"clientInstructionId"`
	Currency              string  `json:"currency"`
}

// withdrawInstructionJSON is the wire shape for withdrawal fund instructions.
// Amount is a string to preserve precision (ADR 0008).
type withdrawInstructionJSON struct {
	AccountId             string `json:"accountId"`
	Amount                string `json:"amount"`
	BankInstructionMethod string `json:"bankInstructionMethod"`
	BankInstructionName   string `json:"bankInstructionName"`
	ClientInstructionId   int    `json:"clientInstructionId"`
	Currency              string `json:"currency"`
}

type cashTransferJSON struct {
	Instruction     any    `json:"instruction"`
	InstructionType string `json:"instructionType"`
}

// internalTransferJSON is the wire shape for internal asset transfers.
// TransferQuantity and TransferPrice are strings (ADR 0008).
type internalTransferJSON struct {
	ClientInstructionId int     `json:"clientInstructionId"`
	SourceAccountId     string  `json:"sourceAccountId"`
	TargetAccountId     string  `json:"targetAccountId"`
	TradingInstrument   any     `json:"tradingInstrument"`
	TransferQuantity    string  `json:"transferQuantity"`
	TransferPrice       *string `json:"transferPrice,omitempty"`
	TradeDate           *string `json:"tradeDate,omitempty"`
	SettleDate          *string `json:"settleDate,omitempty"`
}

type internalAssetTransferJSON struct {
	Instruction     internalTransferJSON `json:"instruction"`
	InstructionType string               `json:"instructionType"`
}

// withdrawableFundsResultJSON decodes the withdrawable funds response.
// CashBalance is kept as json.Number to avoid float32 precision loss (ADR 0008).
type withdrawableFundsResultJSON struct {
	CashBalance json.Number `json:"cashBalance"`
	Currency    string      `json:"currency"`
}

// RESTBanking is the sub-manager for all IBKR banking REST endpoints,
// including transfers, cash operations, and instruction management.
type RESTBanking struct {
	surface *RESTSurface
}

// Banking returns the banking sub-manager for accessing banking endpoints.
func (s *RESTSurface) Banking() *RESTBanking { return &RESTBanking{surface: s} }

// ExternalTransfers returns the external asset transfers sub-manager.
func (b *RESTBanking) ExternalTransfers() *RESTExternalAssetTransfers {
	return &RESTExternalAssetTransfers{surface: b.surface}
}

// InternalTransfers returns the internal asset transfers sub-manager.
func (b *RESTBanking) InternalTransfers() *RESTInternalAssetTransfers {
	return &RESTInternalAssetTransfers{surface: b.surface}
}

// CashTransfers returns the external cash transfers sub-manager.
func (b *RESTBanking) CashTransfers() *RESTExternalCashTransfers {
	return &RESTExternalCashTransfers{surface: b.surface}
}

// InternalCash returns the internal cash transfers sub-manager.
func (b *RESTBanking) InternalCash() *RESTInternalCashTransfers {
	return &RESTInternalCashTransfers{surface: b.surface}
}

// BankInstructions returns the bank instructions sub-manager for ACH and Open Banking operations.
func (b *RESTBanking) BankInstructions() *RESTBankInstructions {
	return &RESTBankInstructions{surface: b.surface}
}

// RESTExternalAssetTransfers provides external asset transfer operations (FOP, DWAC, etc.).
type RESTExternalAssetTransfers struct{ surface *RESTSurface }

// RESTInternalAssetTransfers provides internal asset transfer operations between IBKR accounts.
type RESTInternalAssetTransfers struct{ surface *RESTSurface }

// RESTExternalCashTransfers provides external cash transfer operations (deposits and withdrawals).
type RESTExternalCashTransfers struct{ surface *RESTSurface }

// RESTInternalCashTransfers provides internal cash transfer operations between IBKR accounts.
type RESTInternalCashTransfers struct{ surface *RESTSurface }

// RESTBankInstructions provides bank instruction management (ACH, Open Banking).
type RESTBankInstructions struct{ surface *RESTSurface }

// RESTClientInstruction represents a client instruction with its associated sub-instructions.
//
// Example:
//
//	ci, err := surface.Banking().ClientInstruction(ctx, 12345)
//	if err != nil { ... }
//	fmt.Println(ci.Status)
//	for _, ins := range ci.Instructions {
//	    fmt.Println(ins.ID, ins.Status)
//	}
type RESTClientInstruction struct {
	ID           int64
	AccountID    AccountID
	Type         string
	Status       string
	CreatedAt    time.Time
	Instructions []RESTInstruction
}

// RESTInstructionSet represents a set of instructions grouped under one account.
type RESTInstructionSet struct {
	ID           int64
	AccountID    AccountID
	Status       string
	Instructions []RESTInstruction
}

// RESTInstruction represents a single instruction with ID, type, status, and timestamp.
type RESTInstruction struct {
	ID        int64
	Type      string
	Status    string
	CreatedAt time.Time
}

// RESTTransaction represents a banking transaction record.
//
// Example:
//
//	txs, err := surface.Banking().QueryTransactions(ctx, TransactionQueryRequest{
//	    AccountID: "U1234567",
//	    Days:      30,
//	})
//	for _, tx := range txs {
//	    fmt.Println(tx.Date, tx.Type, tx.Amount, tx.Currency)
//	}
type RESTTransaction struct {
	ID          string
	Date        time.Time
	Type        string
	Amount      string
	Currency    string
	Status      string
	Description string
}

// CancelInstructionRequest represents a request to cancel an instruction.
//
// Example:
//
//	err := surface.Banking().CancelInstruction(ctx, CancelInstructionRequest{
//	    InstructionID: 67890,
//	    Reason:        "no longer needed",
//	})
type CancelInstructionRequest struct {
	InstructionID int64
	Reason        string
}

// TransactionQueryRequest represents a request to query transactions for an account.
type TransactionQueryRequest struct {
	AccountID AccountID
	Days      int
}

// ClientInstruction retrieves a client instruction by its ID, including all sub-instructions.
func (b *RESTBanking) ClientInstruction(ctx context.Context, id int64) (*RESTClientInstruction, error) {
	const op = "Banking.ClientInstruction"
	if err := b.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := b.surface.generated.GetClientInstructionsWithResponse(ctx, id)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(b.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := b.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(b.surface.owner.cfg.logger, e)
		return nil, e
	}
	var raw clientInstructionRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(b.surface.owner.cfg.logger, e)
		return nil, e
	}
	return raw.toPublic(), nil
}

// InstructionSet retrieves an instruction set by its ID.
func (b *RESTBanking) InstructionSet(ctx context.Context, id int64) (*RESTInstructionSet, error) {
	const op = "Banking.InstructionSet"
	if err := b.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := b.surface.generated.GetInstructionSetsWithResponse(ctx, id)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(b.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := b.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(b.surface.owner.cfg.logger, e)
		return nil, e
	}
	var raw instructionSetRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(b.surface.owner.cfg.logger, e)
		return nil, e
	}
	return raw.toPublic(), nil
}

// Instruction retrieves a single instruction by its ID.
func (b *RESTBanking) Instruction(ctx context.Context, id int64) (*RESTInstruction, error) {
	const op = "Banking.Instruction"
	if err := b.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := b.surface.generated.GetInstructionsWithResponse(ctx, id)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(b.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := b.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(b.surface.owner.cfg.logger, e)
		return nil, e
	}
	var raw instructionRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(b.surface.owner.cfg.logger, e)
		return nil, e
	}
	return raw.toPublic(), nil
}

// QueryTransactions queries transactions for an account over a specified number of days.
func (b *RESTBanking) QueryTransactions(ctx context.Context, req TransactionQueryRequest) ([]RESTTransaction, error) {
	const op = "Banking.QueryTransactions"
	if err := b.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	days := float32(req.Days)
	if req.Days <= 0 {
		days = 5
	}
	payload := client.CreateInstructionsQueryJSONRequestBody{
		Instruction: client.QueryRecentInstructions{
			AccountId:           string(req.AccountID),
			ClientInstructionId: 0,
			TransactionHistory: struct {
				DaysToGoBack    float32                                                          `json:"daysToGoBack"`
				TransactionType *client.QueryRecentInstructionsTransactionHistoryTransactionType `json:"transactionType,omitempty"`
			}{
				DaysToGoBack: days,
			},
		},
	}
	resp, err := b.surface.generated.CreateInstructionsQueryWithResponse(ctx, payload)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(b.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := b.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(b.surface.owner.cfg.logger, e)
		return nil, e
	}
	var raw transactionsRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(b.surface.owner.cfg.logger, e)
		return nil, e
	}
	return raw.toPublic(), nil
}

// CancelInstruction cancels a pending instruction.
func (b *RESTBanking) CancelInstruction(ctx context.Context, req CancelInstructionRequest) error {
	const op = "Banking.CancelInstruction"
	if err := b.surface.owner.checkOpen(); err != nil {
		return err
	}
	payload := client.CreateInstructionsCancelJSONRequestBody{
		Instruction: client.CancelInstruction{
			InstructionId: int(req.InstructionID),
			Reason:        req.Reason,
		},
	}
	resp, err := b.surface.generated.CreateInstructionsCancelWithResponse(ctx, payload)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(b.surface.owner.cfg.logger, e)
		return e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := b.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(b.surface.owner.cfg.logger, e)
		return e
	}
	return nil
}

// CancelInstructionsBulk cancels multiple pending instructions in a single request.
func (b *RESTBanking) CancelInstructionsBulk(ctx context.Context, reqs []CancelInstructionRequest) error {
	const op = "Banking.CancelInstructionsBulk"
	if err := b.surface.owner.checkOpen(); err != nil {
		return err
	}
	payload := client.BulkInstructionsCancelJSONBody{
		Instructions: make([]interface{}, len(reqs)),
	}
	for i, r := range reqs {
		payload.Instructions[i] = client.CancelInstruction{
			InstructionId: int(r.InstructionID),
		}
	}
	resp, err := b.surface.generated.BulkInstructionsCancelWithBodyWithResponse(ctx, "application/json", mustMarshal(payload))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(b.surface.owner.cfg.logger, e)
		return e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := b.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(b.surface.owner.cfg.logger, e)
		return e
	}
	return nil
}

type clientInstructionRaw struct {
	InstructionId   *int64                 `json:"instructionId,omitempty"`
	AccountId       *string                `json:"accountId,omitempty"`
	InstructionType *string                `json:"instructionType,omitempty"`
	Status          *string                `json:"status,omitempty"`
	CreatedAt       *string                `json:"createdAt,omitempty"`
	InstructionsRaw []instructionDetailRaw `json:"instructions,omitempty"`
}

type instructionDetailRaw struct {
	InstructionId *int64  `json:"instructionId,omitempty"`
	Type          *string `json:"type,omitempty"`
	Status        *string `json:"status,omitempty"`
	CreatedAt     *string `json:"createdAt,omitempty"`
}

func (r *clientInstructionRaw) toPublic() *RESTClientInstruction {
	if r == nil {
		return nil
	}
	ci := &RESTClientInstruction{
		ID:        int64PtrVal(r.InstructionId),
		AccountID: AccountID(strPtrVal(r.AccountId)),
		Type:      strPtrVal(r.InstructionType),
		Status:    strPtrVal(r.Status),
	}
	if r.CreatedAt != nil {
		ci.CreatedAt, _ = time.Parse("2006-01-02T15:04:05Z", *r.CreatedAt)
	}
	for _, ir := range r.InstructionsRaw {
		var created time.Time
		if ir.CreatedAt != nil {
			created, _ = time.Parse("2006-01-02T15:04:05Z", *ir.CreatedAt)
		}
		ci.Instructions = append(ci.Instructions, RESTInstruction{
			ID:        int64PtrVal(ir.InstructionId),
			Type:      strPtrVal(ir.Type),
			Status:    strPtrVal(ir.Status),
			CreatedAt: created,
		})
	}
	return ci
}

type instructionSetRaw struct {
	InstructionSetId *int64                 `json:"instructionSetId,omitempty"`
	AccountId        *string                `json:"accountId,omitempty"`
	Status           *string                `json:"status,omitempty"`
	InstructionsRaw  []instructionDetailRaw `json:"instructions,omitempty"`
}

func (r *instructionSetRaw) toPublic() *RESTInstructionSet {
	if r == nil {
		return nil
	}
	is := &RESTInstructionSet{
		ID:        int64PtrVal(r.InstructionSetId),
		AccountID: AccountID(strPtrVal(r.AccountId)),
		Status:    strPtrVal(r.Status),
	}
	for _, ir := range r.InstructionsRaw {
		var created time.Time
		if ir.CreatedAt != nil {
			created, _ = time.Parse("2006-01-02T15:04:05Z", *ir.CreatedAt)
		}
		is.Instructions = append(is.Instructions, RESTInstruction{
			ID:        int64PtrVal(ir.InstructionId),
			Type:      strPtrVal(ir.Type),
			Status:    strPtrVal(ir.Status),
			CreatedAt: created,
		})
	}
	return is
}

type instructionRaw struct {
	InstructionId *int64  `json:"instructionId,omitempty"`
	Type          *string `json:"type,omitempty"`
	Status        *string `json:"status,omitempty"`
	CreatedAt     *string `json:"createdAt,omitempty"`
}

func (r *instructionRaw) toPublic() *RESTInstruction {
	if r == nil {
		return nil
	}
	ins := &RESTInstruction{ID: int64PtrVal(r.InstructionId), Type: strPtrVal(r.Type), Status: strPtrVal(r.Status)}
	if r.CreatedAt != nil {
		ins.CreatedAt, _ = time.Parse("2006-01-02T15:04:05Z", *r.CreatedAt)
	}
	return ins
}

type transactionsRaw struct {
	Transactions []transactionRaw `json:"transactions,omitempty"`
}

type transactionRaw struct {
	ID          *string `json:"id,omitempty"`
	Date        *string `json:"date,omitempty"`
	Type        *string `json:"type,omitempty"`
	Amount      *string `json:"amount,omitempty"`
	Currency    *string `json:"currency,omitempty"`
	Status      *string `json:"status,omitempty"`
	Description *string `json:"description,omitempty"`
}

func (r *transactionsRaw) toPublic() []RESTTransaction {
	if r.Transactions == nil {
		return nil
	}
	out := make([]RESTTransaction, 0, len(r.Transactions))
	for _, t := range r.Transactions {
		var date time.Time
		if t.Date != nil {
			date, _ = time.Parse("2006-01-02", *t.Date)
		}
		out = append(out, RESTTransaction{
			ID:          strPtrVal(t.ID),
			Date:        date,
			Type:        strPtrVal(t.Type),
			Amount:      strPtrVal(t.Amount),
			Currency:    strPtrVal(t.Currency),
			Status:      strPtrVal(t.Status),
			Description: strPtrVal(t.Description),
		})
	}
	return out
}

func int64PtrVal(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func strPtrVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func mustMarshal(v interface{}) io.Reader {
	data, _ := json.Marshal(v)
	return bytes.NewReader(data)
}

// ============================================================================
// Request types for transfer operations
// ============================================================================

// AssetTransferRequest represents a request to transfer assets externally (FOP, DWAC, etc.)
type AssetTransferRequest struct {
	AccountID             AccountID
	ClientInstructionID   string
	ContraBrokerAccountID AccountID
	ContraBrokerDtcCode   string
	Direction             string // "IN" or "OUT"
	Quantity              string
	ConID                 ConID
	// For V2 only
	Positions []PositionV2Request
}

// PositionV2Request represents a position for V2 transfers
type PositionV2Request struct {
	ConID    ConID
	Quantity string
}

// InternalAssetTransferRequest represents a request to transfer assets between IBKR accounts
type InternalAssetTransferRequest struct {
	ClientInstructionID string
	SourceAccountID     AccountID
	TargetAccountID     AccountID
	ConID               ConID
	TransferQuantity    string
	TransferPrice       *string
	TradeDate           *string
	SettleDate          *string
}

// CashTransferRequest represents a request to transfer cash externally (deposit/withdrawal)
type CashTransferRequest struct {
	AccountID             AccountID
	ClientInstructionID   string
	Amount                string
	Currency              string
	BankInstructionMethod string // "ACH", "WIRE", "eDDA", "OPEN_BANKING"
	BankInstructionName   *string
}

// InternalCashTransferRequest represents a request to transfer cash between IBKR accounts
type InternalCashTransferRequest struct {
	ClientInstructionID string
	SourceAccountID     AccountID
	TargetAccountID     AccountID
	Amount              string
	Currency            string
	ClientNote          *string
}

// BankInstructionCreateRequest represents a request to create a bank instruction
type BankInstructionCreateRequest struct {
	AccountID           AccountID
	ClientInstructionID float32
	BankInstructionCode string // "ACH_INSTRUCTION", "OPEN_BANKING_INSTRUCTION", etc.
	BankInstructionName string
	BankAccountNumber   string
	BankRoutingNumber   string
	BankAccountTypeCode int // 1 = Checking, 2 = Savings
	BankName            string
	Currency            string
	AchType             string // "DEBIT", "CREDIT", "DEBIT_CREDIT"
}

// BankInstructionQueryRequest represents a request to query bank instructions
type BankInstructionQueryRequest struct {
	AccountID             AccountID
	BankInstructionMethod string
}

// TransferResult represents the result of a transfer instruction
type TransferResult struct {
	ClientInstructionID int64
	InstructionID       int64
	InstructionStatus   string
	IbReferenceID       *int64
	Description         *string
}

// ============================================================================
// RESTExternalAssetTransfers - External asset transfer operations
// ============================================================================

// Transfer initiates a single external asset transfer (FOP, DWAC, etc.)
func (m *RESTExternalAssetTransfers) Transfer(ctx context.Context, req AssetTransferRequest) (string, error) {
	const op = "ExternalAssetTransfers.Transfer"
	if err := m.surface.owner.checkOpen(); err != nil {
		return "", err
	}

	payload := externalAssetTransferJSON{
		Instruction: fopInstructionJSON{
			AccountId:             string(req.AccountID),
			ClientInstructionId:   strToInt(req.ClientInstructionID),
			ContraBrokerAccountId: string(req.ContraBrokerAccountID),
			ContraBrokerDtcCode:   req.ContraBrokerDtcCode,
			Direction:             req.Direction,
			Quantity:              req.Quantity,
			TradingInstrument:     map[string]any{"conid": int(req.ConID)},
		},
		InstructionType: "FOP",
	}

	resp, err := m.surface.generated.CreateExternalAssetTransfersWithBodyWithResponse(ctx, "application/json", mustMarshal(payload))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return "", e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return "", e
	}
	if resp.JSON202 == nil {
		e := &Error{Op: op, Message: "unexpected nil 202 response"}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return "", e
	}
	return strconv.FormatFloat(float64(resp.JSON202.InstructionSetId), 'f', -1, 32), nil
}

// TransferBulk initiates multiple external asset transfers in a single request
func (m *RESTExternalAssetTransfers) TransferBulk(ctx context.Context, reqs []AssetTransferRequest) ([]TransferResult, error) {
	const op = "ExternalAssetTransfers.TransferBulk"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}

	type fopInstrJSON struct {
		AccountId             string `json:"accountId"`
		ClientInstructionId   int    `json:"clientInstructionId"`
		ContraBrokerAccountId string `json:"contraBrokerAccountId"`
		ContraBrokerDtcCode   string `json:"contraBrokerDtcCode"`
		Direction             string `json:"direction"`
		Quantity              string `json:"quantity"`
		TradingInstrument     any    `json:"tradingInstrument"`
	}
	type bulkPayload struct {
		InstructionType string         `json:"instructionType"`
		Instructions    []fopInstrJSON `json:"instructions"`
	}

	instrs := make([]fopInstrJSON, len(reqs))
	for i, req := range reqs {
		instrs[i] = fopInstrJSON{
			AccountId:             string(req.AccountID),
			ClientInstructionId:   strToInt(req.ClientInstructionID),
			ContraBrokerAccountId: string(req.ContraBrokerAccountID),
			ContraBrokerDtcCode:   req.ContraBrokerDtcCode,
			Direction:             req.Direction,
			Quantity:              req.Quantity,
			TradingInstrument:     map[string]any{"conid": int(req.ConID)},
		}
	}

	payload := bulkPayload{InstructionType: "FOP", Instructions: instrs}

	resp, err := m.surface.generated.BulkExternalAssetTransfersWithBodyWithResponse(ctx, "application/json", mustMarshal(payload))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.JSON202 == nil {
		e := &Error{Op: op, Message: "unexpected nil 202 response"}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	return extractBulkResults(resp.JSON202.InstructionResults), nil
}

// TransferV2 initiates a single external asset transfer using V2 API
func (m *RESTExternalAssetTransfers) TransferV2(ctx context.Context, req AssetTransferRequest) (string, error) {
	const op = "ExternalAssetTransfers.TransferV2"
	if err := m.surface.owner.checkOpen(); err != nil {
		return "", err
	}

	instr := client.CreateExternalAssetTransfers2JSONBody_Instruction{}
	positions := make([]client.TradingInstrumentV2, len(req.Positions))
	for i, pos := range req.Positions {
		positions[i] = client.TradingInstrumentV2{Quantity: strToDecimal(pos.Quantity)}
		_ = positions[i].FromTradingInstrumentV20(client.TradingInstrumentV20{Conid: int(pos.ConID)})
	}

	_ = instr.FromFopInstructionV2(client.FopInstructionV2{
		AccountId:             string(req.AccountID),
		ClientInstructionId:   strToInt(req.ClientInstructionID),
		ContraBrokerAccountId: string(req.ContraBrokerAccountID),
		ContraBrokerDtcCode:   req.ContraBrokerDtcCode,
		Direction:             client.FopInstructionV2Direction(req.Direction),
		Positions:             positions,
	})

	payload := client.CreateExternalAssetTransfers2JSONRequestBody{
		Instruction:     instr,
		InstructionType: client.CreateExternalAssetTransfers2JSONBodyInstructionTypeFOP,
	}

	resp, err := m.surface.generated.CreateExternalAssetTransfers2WithBodyWithResponse(ctx, "application/json", mustMarshal(payload))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return "", e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return "", e
	}
	if resp.JSON202 == nil {
		e := &Error{Op: op, Message: "unexpected nil 202 response"}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return "", e
	}
	return strconv.Itoa(resp.JSON202.InstructionSetId), nil
}

// TransferBulkV2 initiates multiple external asset transfers using V2 API
func (m *RESTExternalAssetTransfers) TransferBulkV2(ctx context.Context, reqs []AssetTransferRequest) ([]TransferResult, error) {
	const op = "ExternalAssetTransfers.TransferBulkV2"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}

	type instrV2JSON struct {
		AccountId             string `json:"accountId"`
		ClientInstructionId   int    `json:"clientInstructionId"`
		ContraBrokerAccountId string `json:"contraBrokerAccountId"`
		ContraBrokerDtcCode   string `json:"contraBrokerDtcCode"`
		Direction             string `json:"direction"`
		Positions             []struct {
			Conid    int    `json:"conid"`
			Quantity string `json:"quantity"`
		} `json:"positions"`
	}
	type bulkPayloadV2 struct {
		InstructionType string        `json:"instructionType"`
		Instructions    []instrV2JSON `json:"instructions"`
	}

	instrs := make([]instrV2JSON, len(reqs))
	for i, req := range reqs {
		positions := make([]struct {
			Conid    int    `json:"conid"`
			Quantity string `json:"quantity"`
		}, len(req.Positions))
		for j, pos := range req.Positions {
			positions[j] = struct {
				Conid    int    `json:"conid"`
				Quantity string `json:"quantity"`
			}{Conid: int(pos.ConID), Quantity: pos.Quantity}
		}
		instrs[i] = instrV2JSON{
			AccountId:             string(req.AccountID),
			ClientInstructionId:   strToInt(req.ClientInstructionID),
			ContraBrokerAccountId: string(req.ContraBrokerAccountID),
			ContraBrokerDtcCode:   req.ContraBrokerDtcCode,
			Direction:             req.Direction,
			Positions:             positions,
		}
	}

	payload := bulkPayloadV2{InstructionType: "FOP", Instructions: instrs}

	resp, err := m.surface.generated.BulkExternalAssetTransfers2WithBodyWithResponse(ctx, "application/json", mustMarshal(payload))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.JSON202 == nil {
		e := &Error{Op: op, Message: "unexpected nil 202 response"}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	return extractBulkResults(resp.JSON202.InstructionResults), nil
}

// ============================================================================
// RESTInternalAssetTransfers - Internal asset transfer operations
// ============================================================================

// Transfer initiates a single internal asset transfer between IBKR accounts
func (m *RESTInternalAssetTransfers) Transfer(ctx context.Context, req InternalAssetTransferRequest) (string, error) {
	const op = "InternalAssetTransfers.Transfer"
	if err := m.surface.owner.checkOpen(); err != nil {
		return "", err
	}

	payload := internalTransferJSON{
		ClientInstructionId: strToInt(req.ClientInstructionID),
		SourceAccountId:     string(req.SourceAccountID),
		TargetAccountId:     string(req.TargetAccountID),
		TradingInstrument:   map[string]any{"conid": int(req.ConID)},
		TransferQuantity:    req.TransferQuantity,
		TransferPrice:       req.TransferPrice,
		TradeDate:           req.TradeDate,
		SettleDate:          req.SettleDate,
	}

	body := internalAssetTransferJSON{
		Instruction:     payload,
		InstructionType: "INTERNAL_POSITION_TRANSFER",
	}

	resp, err := m.surface.generated.CreateInternalAssetTransfersWithBodyWithResponse(ctx, "application/json", mustMarshal(body))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return "", e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return "", e
	}
	if resp.JSON202 == nil {
		e := &Error{Op: op, Message: "unexpected nil 202 response"}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return "", e
	}
	return strconv.FormatFloat(float64(resp.JSON202.InstructionSetId), 'f', -1, 32), nil
}

// TransferBulk initiates multiple internal asset transfers in a single request
func (m *RESTInternalAssetTransfers) TransferBulk(ctx context.Context, reqs []InternalAssetTransferRequest) ([]TransferResult, error) {
	const op = "InternalAssetTransfers.TransferBulk"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}

	type bulkInternalPayload struct {
		InstructionType string                 `json:"instructionType"`
		Instructions    []internalTransferJSON `json:"instructions"`
	}

	instrs := make([]internalTransferJSON, len(reqs))
	for i, req := range reqs {
		instrs[i] = internalTransferJSON{
			ClientInstructionId: strToInt(req.ClientInstructionID),
			SourceAccountId:     string(req.SourceAccountID),
			TargetAccountId:     string(req.TargetAccountID),
			TradingInstrument:   map[string]any{"conid": int(req.ConID)},
			TransferQuantity:    req.TransferQuantity,
			TransferPrice:       req.TransferPrice,
			TradeDate:           req.TradeDate,
			SettleDate:          req.SettleDate,
		}
	}

	payload := bulkInternalPayload{InstructionType: "INTERNAL_POSITION_TRANSFER", Instructions: instrs}

	resp, err := m.surface.generated.BulkInternalAssetTransfersWithBodyWithResponse(ctx, "application/json", mustMarshal(payload))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.JSON202 == nil {
		e := &Error{Op: op, Message: "unexpected nil 202 response"}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	return extractBulkResults(resp.JSON202.InstructionResults), nil
}

// ============================================================================
// RESTExternalCashTransfers - External cash transfer operations
// ============================================================================

// Transfer initiates a single external cash transfer (deposit or withdrawal)
func (m *RESTExternalCashTransfers) Transfer(ctx context.Context, req CashTransferRequest, isDeposit bool) (string, error) {
	const op = "ExternalCashTransfers.Transfer"
	if err := m.surface.owner.checkOpen(); err != nil {
		return "", err
	}

	type instrPayload struct {
		Amount                string  `json:"amount"`
		AccountId             string  `json:"accountId"`
		BankInstructionMethod string  `json:"bankInstructionMethod"`
		BankInstructionName   *string `json:"bankInstructionName,omitempty"`
		ClientInstructionId   int     `json:"clientInstructionId"`
		Currency              string  `json:"currency"`
	}
	type cashPayload struct {
		Instruction     instrPayload `json:"instruction"`
		InstructionType string       `json:"instructionType"`
	}

	var bn *string
	if req.BankInstructionName != nil {
		bn = req.BankInstructionName
	}
	instrType := "WITHDRAWAL"
	if isDeposit {
		instrType = "DEPOSIT"
	}
	payload := cashPayload{
		Instruction: instrPayload{
			Amount:                req.Amount,
			AccountId:             string(req.AccountID),
			BankInstructionMethod: req.BankInstructionMethod,
			BankInstructionName:   bn,
			ClientInstructionId:   strToInt(req.ClientInstructionID),
			Currency:              req.Currency,
		},
		InstructionType: instrType,
	}

	resp, err := m.surface.generated.CreateExternalCashTransfersWithBodyWithResponse(ctx, "application/json", mustMarshal(payload))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return "", e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return "", e
	}
	if resp.JSON202 == nil {
		e := &Error{Op: op, Message: "unexpected nil 202 response"}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return "", e
	}
	return strconv.FormatFloat(float64(resp.JSON202.InstructionSetId), 'f', -1, 32), nil
}

// TransferBulk initiates multiple external cash transfers in a single request
func (m *RESTExternalCashTransfers) TransferBulk(ctx context.Context, reqs []CashTransferRequest, isDeposit bool) ([]TransferResult, error) {
	const op = "ExternalCashTransfers.TransferBulk"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}

	var instrType client.BulkExternalCashTransfersJSONBodyInstructionType
	if isDeposit {
		instrType = client.BulkExternalCashTransfersJSONBodyInstructionTypeDEPOSIT
	} else {
		instrType = client.BulkExternalCashTransfersJSONBodyInstructionTypeWITHDRAWAL
	}

	payload := client.BulkExternalCashTransfersJSONBody{
		InstructionType: instrType,
		Instructions:    make([]interface{}, len(reqs)),
	}

	for i, req := range reqs {
		if isDeposit {
			depositInstr := client.DepositFundsInstruction{
				AccountId:             string(req.AccountID),
				Amount:                strToDecimal(req.Amount),
				BankInstructionMethod: client.DepositFundsInstructionBankInstructionMethod(req.BankInstructionMethod),
				ClientInstructionId:   strToInt(req.ClientInstructionID),
				Currency:              req.Currency,
			}
			if req.BankInstructionName != nil {
				depositInstr.BankInstructionName = req.BankInstructionName
			}
			payload.Instructions[i] = depositInstr
		} else {
			withdrawInstr := client.WithdrawFundsInstruction{
				AccountId:             string(req.AccountID),
				Amount:                strToDecimal(req.Amount),
				BankInstructionMethod: client.WithdrawFundsInstructionBankInstructionMethod(req.BankInstructionMethod),
				BankInstructionName:   derefStr(req.BankInstructionName),
				ClientInstructionId:   strToInt(req.ClientInstructionID),
				Currency:              req.Currency,
			}
			payload.Instructions[i] = withdrawInstr
		}
	}

	resp, err := m.surface.generated.BulkExternalCashTransfersWithBodyWithResponse(ctx, "application/json", mustMarshal(payload))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.JSON202 == nil {
		e := &Error{Op: op, Message: "unexpected nil 202 response"}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	return extractBulkResults(resp.JSON202.InstructionResults), nil
}

// QueryBalances queries cash balances (withdrawable funds)
func (m *RESTExternalCashTransfers) QueryBalances(ctx context.Context, req CashTransferRequest) (*WithdrawableFundsResult, error) {
	const op = "ExternalCashTransfers.QueryBalances"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}

	instr := client.CreateExternalCashTransfersQueryJSONBody_Instruction{}
	_ = instr.FromQueryWithdrawableFunds(client.QueryWithdrawableFunds{
		AccountId: string(req.AccountID),
		Currency:  req.Currency,
	})

	payload := client.CreateExternalCashTransfersQueryJSONRequestBody{
		Instruction:     instr,
		InstructionType: client.CreateExternalCashTransfersQueryJSONBodyInstructionTypeQUERYWITHDRAWABLEFUNDS,
	}

	resp, err := m.surface.generated.CreateExternalCashTransfersQueryWithBodyWithResponse(ctx, "application/json", mustMarshal(payload))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}

	// Parse the response
	var result struct {
		CashBalance float32 `json:"cashBalance"`
		Currency    string  `json:"currency"`
	}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}

	return &WithdrawableFundsResult{
		CashBalance: fmt.Sprintf("%.2f", result.CashBalance),
		Currency:    result.Currency,
	}, nil
}

// WithdrawableFundsResult represents withdrawable funds query result
type WithdrawableFundsResult struct {
	CashBalance string
	Currency    string
}

// ============================================================================
// RESTInternalCashTransfers - Internal cash transfer operations
// ============================================================================

// Transfer initiates a single internal cash transfer between IBKR accounts
func (m *RESTInternalCashTransfers) Transfer(ctx context.Context, req InternalCashTransferRequest) (string, error) {
	const op = "InternalCashTransfers.Transfer"
	if err := m.surface.owner.checkOpen(); err != nil {
		return "", err
	}

	instr := client.InternalCashTransferInstruction{
		Amount:              strToDecimal(req.Amount),
		ClientInstructionId: strToInt(req.ClientInstructionID),
		Currency:            req.Currency,
		SourceAccountId:     string(req.SourceAccountID),
		TargetAccountId:     string(req.TargetAccountID),
	}
	if req.ClientNote != nil {
		instr.ClientNote = req.ClientNote
	}

	payload := client.CreateInternalCashTransfersJSONRequestBody{
		Instruction:     instr,
		InstructionType: client.CreateInternalCashTransfersJSONBodyInstructionTypeINTERNALCASHTRANSFER,
	}

	resp, err := m.surface.generated.CreateInternalCashTransfersWithBodyWithResponse(ctx, "application/json", mustMarshal(payload))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return "", e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return "", e
	}
	if resp.JSON202 == nil {
		e := &Error{Op: op, Message: "unexpected nil 202 response"}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return "", e
	}
	return strconv.Itoa(resp.JSON202.InstructionSetId), nil
}

// TransferBulk initiates multiple internal cash transfers in a single request
func (m *RESTInternalCashTransfers) TransferBulk(ctx context.Context, reqs []InternalCashTransferRequest) ([]TransferResult, error) {
	const op = "InternalCashTransfers.TransferBulk"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}

	payload := client.BulkInternalCashTransfersJSONBody{
		InstructionType: client.BulkInternalCashTransfersJSONBodyInstructionTypeINTERNALCASHTRANSFER,
		Instructions:    make([]interface{}, len(reqs)),
	}

	for i, req := range reqs {
		instr := client.InternalCashTransferInstruction{
			Amount:              strToDecimal(req.Amount),
			ClientInstructionId: strToInt(req.ClientInstructionID),
			Currency:            req.Currency,
			SourceAccountId:     string(req.SourceAccountID),
			TargetAccountId:     string(req.TargetAccountID),
		}
		if req.ClientNote != nil {
			instr.ClientNote = req.ClientNote
		}
		payload.Instructions[i] = instr
	}

	resp, err := m.surface.generated.BulkInternalCashTransfersWithBodyWithResponse(ctx, "application/json", mustMarshal(payload))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.JSON202 == nil {
		e := &Error{Op: op, Message: "unexpected nil 202 response"}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	return extractBulkResults(resp.JSON202.InstructionResults), nil
}

// ============================================================================
// RESTBankInstructions - Bank instruction operations
// ============================================================================

// Create creates a single bank instruction (ACH, Open Banking, etc.)
func (m *RESTBankInstructions) Create(ctx context.Context, req BankInstructionCreateRequest) (string, error) {
	const op = "BankInstructions.Create"
	if err := m.surface.owner.checkOpen(); err != nil {
		return "", err
	}

	instr := client.CreateBankInstructionsJSONBody_Instruction{}
	_ = instr.FromAchInstruction(client.AchInstruction{
		AccountId:           string(req.AccountID),
		AchType:             client.AchInstructionAchType(req.AchType),
		BankInstructionCode: client.AchInstructionBankInstructionCode(req.BankInstructionCode),
		BankInstructionName: req.BankInstructionName,
		ClientInstructionId: int(req.ClientInstructionID),
		Currency:            req.Currency,
		ClientAccountInfo: struct {
			BankAccountNumber   string                                                    `json:"bankAccountNumber"`
			BankAccountTypeCode client.AchInstructionClientAccountInfoBankAccountTypeCode `json:"bankAccountTypeCode"`
			BankName            string                                                    `json:"bankName"`
			BankRoutingNumber   string                                                    `json:"bankRoutingNumber"`
		}{
			BankAccountNumber:   req.BankAccountNumber,
			BankAccountTypeCode: client.AchInstructionClientAccountInfoBankAccountTypeCode(req.BankAccountTypeCode),
			BankName:            req.BankName,
			BankRoutingNumber:   req.BankRoutingNumber,
		},
	})

	payload := client.CreateBankInstructionsJSONRequestBody{
		Instruction:     instr,
		InstructionType: client.CreateBankInstructionsJSONBodyInstructionTypeACHINSTRUCTION,
	}

	resp, err := m.surface.generated.CreateBankInstructionsWithBodyWithResponse(ctx, "application/json", mustMarshal(payload))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return "", e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return "", e
	}
	if resp.JSON202 == nil {
		e := &Error{Op: op, Message: "unexpected nil 202 response"}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return "", e
	}
	return strconv.Itoa(resp.JSON202.InstructionSetId), nil
}

// Query queries bank instructions
func (m *RESTBankInstructions) Query(ctx context.Context, req BankInstructionQueryRequest) (*BankInstructionResult, error) {
	const op = "BankInstructions.Query"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}

	instr := client.CreateBankInstructionsQueryJSONBody_Instruction{}
	_ = instr.FromQueryBankInstruction(client.QueryBankInstruction{
		AccountId:             string(req.AccountID),
		BankInstructionMethod: client.QueryBankInstructionBankInstructionMethod(req.BankInstructionMethod),
	})

	payload := client.CreateBankInstructionsQueryJSONRequestBody{
		Instruction:     instr,
		InstructionType: client.CreateBankInstructionsQueryJSONBodyInstructionTypeQUERYBANKINSTRUCTION,
	}

	resp, err := m.surface.generated.CreateBankInstructionsQueryWithBodyWithResponse(ctx, "application/json", mustMarshal(payload))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}

	var result BankInstructionResult
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	return &result, nil
}

// CreateBulk creates multiple bank instructions in a single request
func (m *RESTBankInstructions) CreateBulk(ctx context.Context, reqs []BankInstructionCreateRequest) ([]TransferResult, error) {
	const op = "BankInstructions.CreateBulk"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}

	payload := client.BulkBankInstructionsJSONBody{
		InstructionType: client.BulkBankInstructionsJSONBodyInstructionTypeACHINSTRUCTION,
		Instructions:    make([]interface{}, len(reqs)),
	}

	for i, req := range reqs {
		payload.Instructions[i] = client.AchInstruction{
			AccountId:           string(req.AccountID),
			AchType:             client.AchInstructionAchType(req.AchType),
			BankInstructionCode: client.AchInstructionBankInstructionCode(req.BankInstructionCode),
			BankInstructionName: req.BankInstructionName,
			ClientInstructionId: int(req.ClientInstructionID),
			Currency:            req.Currency,
			ClientAccountInfo: struct {
				BankAccountNumber   string                                                    `json:"bankAccountNumber"`
				BankAccountTypeCode client.AchInstructionClientAccountInfoBankAccountTypeCode `json:"bankAccountTypeCode"`
				BankName            string                                                    `json:"bankName"`
				BankRoutingNumber   string                                                    `json:"bankRoutingNumber"`
			}{
				BankAccountNumber:   req.BankAccountNumber,
				BankAccountTypeCode: client.AchInstructionClientAccountInfoBankAccountTypeCode(req.BankAccountTypeCode),
				BankName:            req.BankName,
				BankRoutingNumber:   req.BankRoutingNumber,
			},
		}
	}

	resp, err := m.surface.generated.BulkBankInstructionsWithBodyWithResponse(ctx, "application/json", mustMarshal(payload))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.JSON202 == nil {
		e := &Error{Op: op, Message: "unexpected nil 202 response"}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	return extractBulkResults(resp.JSON202.InstructionResults), nil
}

// BankInstructionResult represents the result of querying bank instructions
type BankInstructionResult struct {
	AccountID             AccountID
	BankInstructionName   string
	BankInstructionMethod string
	Status                string
}

// ============================================================================
// Helper functions
// ============================================================================

// makeTradingInstrumentRef creates a TradingInstrumentRef from conid
func makeTradingInstrumentRef(conid ConID) client.TradingInstrumentRef {
	ref := client.TradingInstrumentRef{}
	_ = ref.FromTradingInstrumentRef0(client.TradingInstrumentRef0{Conid: int(conid)})
	return ref
}

// extractBulkResults extracts TransferResult slice from bulk response
func extractBulkResults(results *[]struct {
	InstructionResult client.InstructionResult `json:"instructionResult"`
	InstructionSetId  float32                  `json:"instructionSetId"`
	Status            int64                    `json:"status"`
}) []TransferResult {
	if results == nil {
		return nil
	}
	out := make([]TransferResult, 0, len(*results))
	for _, r := range *results {
		out = append(out, TransferResult{
			ClientInstructionID: int64(r.InstructionResult.ClientInstructionId),
			InstructionID:       int64(r.InstructionResult.InstructionId),
			InstructionStatus:   string(r.InstructionResult.InstructionStatus),
			IbReferenceID:       intPtrToInt64Ptr(r.InstructionResult.IbReferenceId),
			Description:         r.InstructionResult.Description,
		})
	}
	return out
}

// derefStr returns empty string if ptr is nil, otherwise returns *ptr
func derefStr(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// f32PtrToInt64Ptr converts a *float32 to *int64, returning nil if the input is nil.
func f32PtrToInt64Ptr(p *float32) *int64 {
	if p == nil {
		return nil
	}
	v := int64(*p)
	return &v
}

func intPtrToInt64Ptr(p *int) *int64 {
	if p == nil {
		return nil
	}
	v := int64(*p)
	return &v
}
