// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/shing1211/ibkrapi4go/client"
)

type RESTBanking struct {
	surface *RESTSurface
}

func (s *RESTSurface) Banking() *RESTBanking { return &RESTBanking{surface: s} }

func (b *RESTBanking) ExternalTransfers() *RESTExternalAssetTransfers {
	return &RESTExternalAssetTransfers{surface: b.surface}
}
func (b *RESTBanking) InternalTransfers() *RESTInternalAssetTransfers {
	return &RESTInternalAssetTransfers{surface: b.surface}
}
func (b *RESTBanking) CashTransfers() *RESTExternalCashTransfers {
	return &RESTExternalCashTransfers{surface: b.surface}
}
func (b *RESTBanking) InternalCash() *RESTInternalCashTransfers {
	return &RESTInternalCashTransfers{surface: b.surface}
}
func (b *RESTBanking) BankInstructions() *RESTBankInstructions {
	return &RESTBankInstructions{surface: b.surface}
}
func (b *RESTBanking) Instructions() *RESTInstructions {
	return &RESTInstructions{surface: b.surface}
}

type RESTExternalAssetTransfers struct{ surface *RESTSurface }
type RESTInternalAssetTransfers struct{ surface *RESTSurface }
type RESTExternalCashTransfers struct{ surface *RESTSurface }
type RESTInternalCashTransfers struct{ surface *RESTSurface }
type RESTBankInstructions struct{ surface *RESTSurface }
type RESTInstructions struct{ surface *RESTSurface }

type RESTClientInstruction struct {
	ID           int64
	AccountID    AccountID
	Type         string
	Status       string
	CreatedAt    time.Time
	Instructions []RESTInstruction
}

type RESTInstructionSet struct {
	ID           int64
	AccountID    AccountID
	Status       string
	Instructions []RESTInstruction
}

type RESTInstruction struct {
	ID        int64
	Type      string
	Status    string
	CreatedAt time.Time
}

type RESTTransaction struct {
	ID          string
	Date        time.Time
	Type        string
	Amount      string
	Currency    string
	Status      string
	Description string
}

type CancelInstructionRequest struct {
	InstructionID int64
	Reason        string
}

type TransactionQueryRequest struct {
	AccountID AccountID
	Days      int
}

func (b *RESTBanking) ClientInstruction(ctx context.Context, id int64) (*RESTClientInstruction, error) {
	const op = "Banking.ClientInstruction"
	if err := b.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := b.surface.generated.GetClientInstructionsWithResponse(ctx, id)
	if err != nil {
		return nil, wrapOp(op, err)
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, b.surface.owner.errorFrom(resp.HTTPResponse, op)
	}
	var raw clientInstructionRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		return nil, &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
	}
	return raw.toPublic(), nil
}

func (b *RESTBanking) InstructionSet(ctx context.Context, id int64) (*RESTInstructionSet, error) {
	const op = "Banking.InstructionSet"
	if err := b.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := b.surface.generated.GetInstructionSetsWithResponse(ctx, id)
	if err != nil {
		return nil, wrapOp(op, err)
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, b.surface.owner.errorFrom(resp.HTTPResponse, op)
	}
	var raw instructionSetRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		return nil, &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
	}
	return raw.toPublic(), nil
}

func (b *RESTBanking) Instruction(ctx context.Context, id int64) (*RESTInstruction, error) {
	const op = "Banking.Instruction"
	if err := b.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := b.surface.generated.GetInstructionsWithResponse(ctx, id)
	if err != nil {
		return nil, wrapOp(op, err)
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, b.surface.owner.errorFrom(resp.HTTPResponse, op)
	}
	var raw instructionRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		return nil, &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
	}
	return raw.toPublic(), nil
}

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
		return nil, wrapOp(op, err)
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, b.surface.owner.errorFrom(resp.HTTPResponse, op)
	}
	var raw transactionsRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		return nil, &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
	}
	return raw.toPublic(), nil
}

func (b *RESTBanking) CancelInstruction(ctx context.Context, req CancelInstructionRequest) error {
	const op = "Banking.CancelInstruction"
	if err := b.surface.owner.checkOpen(); err != nil {
		return err
	}
	payload := client.CreateInstructionsCancelJSONRequestBody{
		Instruction: client.CancelInstruction{
			InstructionId: float32(req.InstructionID),
			Reason:        req.Reason,
		},
	}
	resp, err := b.surface.generated.CreateInstructionsCancelWithResponse(ctx, payload)
	if err != nil {
		return wrapOp(op, err)
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		return b.surface.owner.errorFrom(resp.HTTPResponse, op)
	}
	return nil
}

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
			InstructionId: float32(r.InstructionID),
		}
	}
	resp, err := b.surface.generated.BulkInstructionsCancelWithBodyWithResponse(ctx, "application/json", mustMarshal(payload))
	if err != nil {
		return wrapOp(op, err)
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		return b.surface.owner.errorFrom(resp.HTTPResponse, op)
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
