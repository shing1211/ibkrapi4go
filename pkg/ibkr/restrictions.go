// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"encoding/json"

	"github.com/shing1211/ibkrapi4go/client"
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
		return nil, wrapOp(op, err)
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, r.surface.owner.errorFrom(resp.HTTPResponse, op)
	}
	var raw restrictionsIDsRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		return nil, &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
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
		return nil, wrapOp(op, err)
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, r.surface.owner.errorFrom(resp.HTTPResponse, op)
	}
	var raw restrictionsIDsRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		return nil, &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
	}
	return raw.toPublic(), nil
}

type RESTListDetail struct {
	ListID    int64
	AccountID AccountID
	Name      string
	Type      string
}

type RESTRestrictionDetail struct {
	RestrictionID int64
	AccountID     AccountID
	Scope         string
	Description   string
}

type restrictionsIDsRaw struct {
	Ids []json.RawMessage `json:"ids,omitempty"`
}

func (r *restrictionsIDsRaw) toPublic() []int64 {
	if r.Ids == nil {
		return nil
	}
	out := make([]int64, 0, len(r.Ids))
	for _, raw := range r.Ids {
		var f float64
		if err := json.Unmarshal(raw, &f); err != nil {
			continue
		}
		out = append(out, int64(f))
	}
	return out
}
