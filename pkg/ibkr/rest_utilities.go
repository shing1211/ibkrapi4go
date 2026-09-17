// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"encoding/json"

	"github.com/shing1211/ibkrapi4go/client"
)

type RESTUtilities struct {
	surface *RESTSurface
}

func (s *RESTSurface) Utilities() *RESTUtilities { return &RESTUtilities{surface: s} }

func (m *RESTUtilities) Enumerations(ctx context.Context, enumType string) ([]string, error) {
	const op = "Utilities.Enumerations"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.GetEnumerationsWithResponse(ctx, client.EnumerationType(enumType), nil)
	if err != nil {
		return nil, wrapOp(op, err)
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, m.surface.owner.errorFrom(resp.HTTPResponse, op)
	}
	var raw []string
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		return nil, &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
	}
	return raw, nil
}

func (m *RESTUtilities) ComplexAssetTransferBrokers(ctx context.Context) ([]string, error) {
	const op = "Utilities.ComplexAssetTransferBrokers"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.ListEnumerationsComplexAssetTransferWithResponse(ctx, nil)
	if err != nil {
		return nil, wrapOp(op, err)
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, m.surface.owner.errorFrom(resp.HTTPResponse, op)
	}
	var raw []string
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		return nil, &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
	}
	return raw, nil
}

func (m *RESTUtilities) Forms(ctx context.Context, formNos []int64) ([]Form, error) {
	const op = "Utilities.Forms"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.ListFormsWithResponse(ctx, &client.ListFormsParams{FormNo: &formNos})
	if err != nil {
		return nil, wrapOp(op, err)
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, m.surface.owner.errorFrom(resp.HTTPResponse, op)
	}
	var raw formsRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		return nil, &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
	}
	return raw.toPublic(), nil
}

func (m *RESTUtilities) RequiredForms(ctx context.Context) ([]Form, error) {
	const op = "Utilities.RequiredForms"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.ListFormsRequiredFormsWithResponse(ctx, nil)
	if err != nil {
		return nil, wrapOp(op, err)
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, m.surface.owner.errorFrom(resp.HTTPResponse, op)
	}
	var raw formsRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		return nil, &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
	}
	return raw.toPublic(), nil
}

func (m *RESTUtilities) ParticipatingBanks(ctx context.Context) ([]Bank, error) {
	const op = "Utilities.ParticipatingBanks"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.ListParticipatingBanksWithResponse(ctx, nil)
	if err != nil {
		return nil, wrapOp(op, err)
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, m.surface.owner.errorFrom(resp.HTTPResponse, op)
	}
	var raw banksRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		return nil, &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
	}
	return raw.toPublic(), nil
}

func (m *RESTUtilities) ValidateUsername(ctx context.Context, username string) (bool, error) {
	const op = "Utilities.ValidateUsername"
	if err := m.surface.owner.checkOpen(); err != nil {
		return false, err
	}
	resp, err := m.surface.generated.GetValidationsUsernamesWithResponse(ctx, username)
	if err != nil {
		return false, wrapOp(op, err)
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		return false, m.surface.owner.errorFrom(resp.HTTPResponse, op)
	}
	var raw validationRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		return false, &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
	}
	return raw.Available, nil
}

type Form struct {
	FormNo  int64
	Name    string
	Content string
}

type Bank struct {
	ID   string
	Name string
}

type validationRaw struct {
	Available bool `json:"available"`
}

type formsRaw struct {
	Forms []formRaw `json:"forms,omitempty"`
}

type formRaw struct {
	FormNo  *int64  `json:"formNo,omitempty"`
	Name    *string `json:"name,omitempty"`
	Content *string `json:"content,omitempty"`
}

func (r *formsRaw) toPublic() []Form {
	if r.Forms == nil {
		return nil
	}
	out := make([]Form, 0, len(r.Forms))
	for _, f := range r.Forms {
		out = append(out, Form{
			FormNo:  int64PtrVal(f.FormNo),
			Name:    strPtrVal(f.Name),
			Content: strPtrVal(f.Content),
		})
	}
	return out
}

type banksRaw struct {
	Banks []bankRaw `json:"banks,omitempty"`
}

type bankRaw struct {
	ID   *string `json:"id,omitempty"`
	Name *string `json:"name,omitempty"`
}

func (r *banksRaw) toPublic() []Bank {
	if r.Banks == nil {
		return nil
	}
	out := make([]Bank, 0, len(r.Banks))
	for _, b := range r.Banks {
		out = append(out, Bank{
			ID:   strPtrVal(b.ID),
			Name: strPtrVal(b.Name),
		})
	}
	return out
}
