// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"encoding/json"

	"github.com/shing1211/ibkrapi4go/client"
	"github.com/shing1211/ibkrapi4go/internal"
)

type RESTRestrictions struct {
	surface *RESTSurface
}

func (s *RESTSurface) Restrictions() *RESTRestrictions { return &RESTRestrictions{surface: s} }

func (r *RESTRestrictions) AccountRestrictions(ctx context.Context, accountID AccountID) ([]int64, error) {
	const op = "Restrictions.Account"
	if err := r.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	acct := string(accountID)
	resp, err := r.surface.generated.GetAccountRestrictionsWithResponse(ctx, &client.GetAccountRestrictionsParams{AccountId: acct})
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := r.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	var raw restrictionsIDsRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	return raw.toPublic(), nil
}

func (r *RESTRestrictions) UserRestrictions(ctx context.Context, username string) ([]int64, error) {
	const op = "Restrictions.User"
	if err := r.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := r.surface.generated.GetUserRestrictionsWithResponse(ctx, &client.GetUserRestrictionsParams{UserName: username})
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := r.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	var raw restrictionsIDsRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	return raw.toPublic(), nil
}

// MasterRestrictionEntry represents a restriction ID with its creator type.
type MasterRestrictionEntry struct {
	RestrictionID int64
	ByOperator    bool
}

// MasterListEntry represents a list ID with its creator type.
type MasterListEntry struct {
	ListID     int64
	ByOperator bool
}

// MasterRestrictionIDs returns restriction IDs created by the caller's master account.
// Requires Signed JWT in Authorization param.
func (r *RESTRestrictions) MasterRestrictionIDs(ctx context.Context, username string, auth string, isEmpTrack bool) ([]MasterRestrictionEntry, error) {
	const op = "Restrictions.MasterRestrictionIDs"
	if err := r.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	var empTrack *client.IsEmpTrack
	if isEmpTrack {
		t := client.IsEmpTrack("T")
		empTrack = &t
	}
	resp, err := r.surface.generated.GetMasterRestrictionIdsWithResponse(ctx, &client.GetMasterRestrictionIdsParams{
		MasterUserName: client.MasterUserName(username),
		IsEmpTrack:     empTrack,
		Authorization:  auth,
	})
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := r.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.JSON200 == nil {
		e := &Error{Op: op, Message: "unexpected nil body"}
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	return parseMasterRestrictions(resp.JSON200), nil
}

// MasterListIDs returns list IDs for the caller's master account.
// Requires Signed JWT in Authorization param.
func (r *RESTRestrictions) MasterListIDs(ctx context.Context, username string, auth string, isEmpTrack bool) ([]MasterListEntry, error) {
	const op = "Restrictions.MasterListIDs"
	if err := r.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	var empTrack *client.IsEmpTrack
	if isEmpTrack {
		t := client.IsEmpTrack("T")
		empTrack = &t
	}
	resp, err := r.surface.generated.GetMasterListIdsWithResponse(ctx, &client.GetMasterListIdsParams{
		MasterUserName: client.MasterUserName(username),
		IsEmpTrack:     empTrack,
		Authorization:  auth,
	})
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := r.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.JSON200 == nil {
		e := &Error{Op: op, Message: "unexpected nil body"}
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	return parseMasterListIDs(resp.JSON200), nil
}

// ListDetails holds details about a list.
type ListDetails struct {
	ListID      int64
	Name        string
	Description string
	Type        string
	Entries     []ListEntry
}

// ListEntry represents a single instrument entry in a list.
type ListEntry struct {
	ID        int64
	CreatedAt int64
}

// ListDetails returns details for a specific list.
// Requires Signed JWT in Authorization param.
func (r *RESTRestrictions) ListDetails(ctx context.Context, username string, listID int64, auth string, isEmpTrack bool) (*ListDetails, error) {
	const op = "Restrictions.ListDetails"
	if err := r.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	var empTrack *client.IsEmpTrack
	if isEmpTrack {
		t := client.IsEmpTrack("T")
		empTrack = &t
	}
	resp, err := r.surface.generated.GetListDetailsWithResponse(ctx, &client.GetListDetailsParams{
		MasterUserName: client.MasterUserName(username),
		ListId:         listID,
		IsEmpTrack:     empTrack,
		Authorization:  auth,
	})
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := r.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.JSON200 == nil {
		e := &Error{Op: op, Message: "unexpected nil body"}
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	return parseListDetails(resp.JSON200), nil
}

// RestrictionRule represents a single rule within a restriction.
type RestrictionRule struct {
	RuleID       int64
	Type         string
	ValidityType string
	StartDate    string
	EndDate      string
}

// RestrictionDetail holds details about a restriction.
type RestrictionDetail struct {
	RestrictionID   int64
	Name            string
	Description     string
	ApplicationType string
	UsageType       string
	IsWhiteList     bool
	AllowOnly       string
	MatchAll        string
	Message         string
	Rules           []RestrictionRule
	TmFirstDate     int64
	TmFrequency     string
}

// RestrictionDetails returns details for a specific restriction.
// Requires Signed JWT in Authorization param.
func (r *RESTRestrictions) RestrictionDetails(ctx context.Context, username string, restrictionID int64, auth string, isEmpTrack bool) (*RestrictionDetail, error) {
	const op = "Restrictions.RestrictionDetails"
	if err := r.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	var empTrack *client.IsEmpTrack
	if isEmpTrack {
		t := client.IsEmpTrack("T")
		empTrack = &t
	}
	resp, err := r.surface.generated.GetRestrictionDetailsWithResponse(ctx, &client.GetRestrictionDetailsParams{
		MasterUserName: client.MasterUserName(username),
		RestrictionId:  restrictionID,
		IsEmpTrack:     empTrack,
		Authorization:  auth,
	})
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := r.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.JSON200 == nil {
		e := &Error{Op: op, Message: "unexpected nil body"}
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	return parseRestrictionDetails(resp.JSON200), nil
}

// RestrictionScope holds the scope of a restriction.
type RestrictionScope struct {
	RestrictionID int64
	Scope         string
	AccountIDs    []string
	Truncated     bool
}

// RestrictionScope returns the scope for a specific restriction.
// Requires Signed JWT in Authorization param.
func (r *RESTRestrictions) RestrictionScope(ctx context.Context, username string, restrictionID int64, auth string, isEmpTrack bool) (*RestrictionScope, error) {
	const op = "Restrictions.RestrictionScope"
	if err := r.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	var empTrack *client.IsEmpTrack
	if isEmpTrack {
		t := client.IsEmpTrack("T")
		empTrack = &t
	}
	resp, err := r.surface.generated.GetRestrictionScopeWithResponse(ctx, &client.GetRestrictionScopeParams{
		MasterUserName: client.MasterUserName(username),
		RestrictionId:  restrictionID,
		IsEmpTrack:     empTrack,
		Authorization:  auth,
	})
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := r.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.JSON200 == nil {
		e := &Error{Op: op, Message: "unexpected nil body"}
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	return parseRestrictionScope(resp.JSON200), nil
}

// CsvApplyResponse represents the response from an ApplyCSV operation.
type CsvApplyResponse struct {
	Success   bool
	RequestID int64
	Message   string
}

// ApplyCSV applies previously verified CSV changes.
// Requires Signed JWT in Authorization header and a signed JWT body.
func (r *RESTRestrictions) ApplyCSV(ctx context.Context, auth string, csvJWTContent string) (*CsvApplyResponse, error) {
	const op = "Restrictions.ApplyCSV"
	if err := r.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := r.surface.generated.ApplyCSVWithTextBodyWithResponse(
		ctx,
		&client.ApplyCSVParams{Authorization: auth},
		client.ApplyCSVTextRequestBody(csvJWTContent),
	)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := r.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	var raw csvApplyResponseRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	return raw.toPublic(), nil
}

// CsvVerifyRequest is the request body for VerifyCSV.
type CsvVerifyRequest struct {
	UserName   string
	RequestID  int64
	Payload    []byte
	IsEmpTrack bool
}

// CsvVerifyResponse represents the response from a VerifyCSV operation.
type CsvVerifyResponse struct {
	Success   bool
	RequestID int64
	Message   string
}

// VerifyCSV verifies CSV changes before applying.
// Requires Signed JWT in Authorization header and a JSON body.
func (r *RESTRestrictions) VerifyCSV(ctx context.Context, auth string, req CsvVerifyRequest) (*CsvVerifyResponse, error) {
	const op = "Restrictions.VerifyCSV"
	if err := r.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	payload := client.VerifyRequest{
		UserName:  req.UserName,
		RequestId: req.RequestID,
		Payload:   req.Payload,
	}
	if req.IsEmpTrack {
		t := "T"
		payload.IsEmpTrack = &t
	}
	resp, err := r.surface.generated.VerifyCSVWithResponse(
		ctx,
		&client.VerifyCSVParams{Authorization: auth},
		payload,
	)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := r.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	var raw csvVerifyResponseRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(r.surface.owner.cfg.logger, e)
		return nil, e
	}
	return raw.toPublic(), nil
}

// Helper types and functions

type restrictionsIDsRaw struct {
	Ids []json.RawMessage `json:"ids,omitempty"`
}

func (rr *restrictionsIDsRaw) toPublic() []int64 {
	if rr.Ids == nil {
		return nil
	}
	out := make([]int64, 0, len(rr.Ids))
	for _, raw := range rr.Ids {
		var f float64
		if err := json.Unmarshal(raw, &f); err != nil {
			continue
		}
		out = append(out, int64(f))
	}
	return out
}

func parseMasterRestrictions(r *client.MasterRestrictionsResponse) []MasterRestrictionEntry {
	if r == nil || r.Restrictions == nil {
		return nil
	}
	out := make([]MasterRestrictionEntry, 0, len(r.Restrictions))
	for _, e := range r.Restrictions {
		out = append(out, MasterRestrictionEntry{
			RestrictionID: int64PtrVal(e.RestrictionId),
			ByOperator:    e.ByOperator != nil && *e.ByOperator,
		})
	}
	return out
}

func parseMasterListIDs(r *client.MasterListIdsResponse) []MasterListEntry {
	if r == nil || r.Lists == nil {
		return nil
	}
	out := make([]MasterListEntry, 0, len(r.Lists))
	for _, e := range r.Lists {
		out = append(out, MasterListEntry{
			ListID:     int64PtrVal(e.ListId),
			ByOperator: e.ByOperator != nil && *e.ByOperator,
		})
	}
	return out
}

func parseListDetails(r *client.ListDetailsResponse) *ListDetails {
	if r == nil {
		return nil
	}
	d := &ListDetails{
		ListID:      r.ListId,
		Name:        strPtrVal(r.Name),
		Description: strPtrVal(r.Description),
		Type:        strPtrVal(r.Type),
	}
	if r.Entries != nil {
		d.Entries = make([]ListEntry, 0, len(r.Entries))
		for _, e := range r.Entries {
			d.Entries = append(d.Entries, ListEntry{
				ID:        int64PtrVal(e.Id),
				CreatedAt: int64PtrVal(e.CreatedAt),
			})
		}
	}
	return d
}

func parseRestrictionDetails(r *client.RestrictionDetailsResponse) *RestrictionDetail {
	if r == nil {
		return nil
	}
	d := &RestrictionDetail{
		RestrictionID:   r.RestrictionId,
		Name:            strPtrVal(r.Name),
		Description:     strPtrVal(r.Description),
		ApplicationType: strPtrVal(r.ApplicationType),
		UsageType:       strPtrVal(r.UsageType),
		IsWhiteList:     r.IsWhiteList != nil && *r.IsWhiteList == "T",
		AllowOnly:       strPtrVal(r.AllowOnly),
		MatchAll:        strPtrVal(r.MatchAll),
		Message:         strPtrVal(r.Message),
		TmFirstDate:     int64PtrVal(r.TmFirstDate),
		TmFrequency:     strPtrVal(r.TmFrequency),
	}
	if r.Rules != nil {
		d.Rules = make([]RestrictionRule, 0, len(r.Rules))
		for _, rule := range r.Rules {
			d.Rules = append(d.Rules, RestrictionRule{
				RuleID:       int64PtrVal(rule.RuleId),
				Type:         strPtrVal(rule.Type),
				ValidityType: strPtrVal(rule.ValidityType),
				StartDate:    strPtrVal(rule.StartDate),
				EndDate:      strPtrVal(rule.EndDate),
			})
		}
	}
	return d
}

func parseRestrictionScope(r *client.RestrictionScopeResponse) *RestrictionScope {
	if r == nil {
		return nil
	}
	s := &RestrictionScope{
		RestrictionID: r.RestrictionId,
		Scope:         r.Scope,
		Truncated:     r.Truncated != nil && *r.Truncated,
	}
	if r.AccountIds != nil {
		for _, id := range *r.AccountIds {
			if id != nil {
				s.AccountIDs = append(s.AccountIDs, *id)
			}
		}
	}
	return s
}

type csvApplyResponseRaw struct {
	Success   *bool   `json:"success,omitempty"`
	RequestId *int64  `json:"requestId,omitempty"`
	Message   *string `json:"message,omitempty"`
}

func (r *csvApplyResponseRaw) toPublic() *CsvApplyResponse {
	if r == nil {
		return nil
	}
	return &CsvApplyResponse{
		Success:   r.Success != nil && *r.Success,
		RequestID: int64PtrVal(r.RequestId),
		Message:   strPtrVal(r.Message),
	}
}

type csvVerifyResponseRaw struct {
	Success   *bool   `json:"success,omitempty"`
	RequestId *int64  `json:"requestId,omitempty"`
	Message   *string `json:"message,omitempty"`
}

func (r *csvVerifyResponseRaw) toPublic() *CsvVerifyResponse {
	if r == nil {
		return nil
	}
	return &CsvVerifyResponse{
		Success:   r.Success != nil && *r.Success,
		RequestID: int64PtrVal(r.RequestId),
		Message:   strPtrVal(r.Message),
	}
}
