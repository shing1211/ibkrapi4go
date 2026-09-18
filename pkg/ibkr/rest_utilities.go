// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"encoding/json"

	"github.com/shing1211/ibkrapi4go/client"
	"github.com/shing1211/ibkrapi4go/internal"
)

// RESTUtilities is the sub-manager for utility endpoints (enumerations,
// forms, banks, and username validation).
type RESTUtilities struct {
	surface *RESTSurface
}

// Utilities returns the utilities sub-manager for accessing enumeration,
// form, bank, and validation endpoints.
func (s *RESTSurface) Utilities() *RESTUtilities { return &RESTUtilities{surface: s} }

// Enumerations retrieves the list of valid values for a given enumeration type.
func (m *RESTUtilities) Enumerations(ctx context.Context, enumType string) ([]string, error) {
	const op = "Utilities.Enumerations"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.GetEnumerationsWithResponse(ctx, client.EnumerationType(enumType), nil)
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
	if resp.JSON200 == nil || resp.JSON200.JsonData == nil {
		return nil, nil
	}
	out := make([]string, 0)
	for _, v := range *resp.JSON200.JsonData {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out, nil
}

// ComplexAssetTransferBrokers lists the brokers that support complex asset
// transfers (e.g., DWAC, FOP).
func (m *RESTUtilities) ComplexAssetTransferBrokers(ctx context.Context) ([]string, error) {
	const op = "Utilities.ComplexAssetTransferBrokers"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.ListEnumerationsComplexAssetTransferWithResponse(ctx, nil)
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
	if resp.JSON200 == nil {
		return nil, nil
	}
	return resp.JSON200.Brokers, nil
}

// Forms retrieves forms by their form numbers, returning their names and content.
func (m *RESTUtilities) Forms(ctx context.Context, formNos []int64) ([]Form, error) {
	const op = "Utilities.Forms"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.ListFormsWithResponse(ctx, &client.ListFormsParams{FormNo: &formNos})
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
	var raw formsRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	return raw.toPublic(), nil
}

// RequiredForms lists the forms required for opening an account.
func (m *RESTUtilities) RequiredForms(ctx context.Context) ([]Form, error) {
	const op = "Utilities.RequiredForms"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.ListFormsRequiredFormsWithResponse(ctx, nil)
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
	if resp.JSON200 == nil || resp.JSON200.Forms == nil {
		return nil, nil
	}
	out := make([]Form, len(*resp.JSON200.Forms))
	for i, name := range *resp.JSON200.Forms {
		out[i] = Form{Name: name}
	}
	return out, nil
}

// ParticipatingBanks lists the banks participating in the deposit and
// withdrawal program.
func (m *RESTUtilities) ParticipatingBanks(ctx context.Context) ([]Bank, error) {
	const op = "Utilities.ParticipatingBanks"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.ListParticipatingBanksWithResponse(ctx, nil)
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
	var raw banksRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	return raw.toPublic(), nil
}

// ValidateUsername checks whether the given username is available for use.
func (m *RESTUtilities) ValidateUsername(ctx context.Context, username string) (bool, error) {
	const op = "Utilities.ValidateUsername"
	if err := m.surface.owner.checkOpen(); err != nil {
		return false, err
	}
	resp, err := m.surface.generated.GetValidationsUsernamesWithResponse(ctx, username)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return false, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return false, e
	}
	var raw validationRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return false, e
	}
	return raw.Available, nil
}

// Form is an IBKR form with its number, human-readable name, and content.
//
// Example:
//
//	forms, err := surf.Utilities().Forms(ctx, []int64{1234})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, f := range forms {
//	    fmt.Printf("Form %d: %s\n", f.FormNo, f.Name)
//	}
type Form struct {
	FormNo  int64
	Name    string
	Content string
}

// Bank is a participating bank with its identifier and display name.
//
// Example:
//
//	banks, err := surf.Utilities().ParticipatingBanks(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, b := range banks {
//	    fmt.Printf("%s (%s)\n", b.Name, b.ID)
//	}
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
