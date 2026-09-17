// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/oapi-codegen/runtime/types"
	"github.com/shing1211/ibkrapi4go/client"
)

type ListRequestsFilter struct {
	From   time.Time
	To     time.Time
	Status string
	Limit  int64
	Offset int64
}

func (m *RESTRequests) ListRequests(ctx context.Context, f ListRequestsFilter) ([]RESTRequestSummary, error) {
	const op = "Requests.List"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	req := client.RequestDetailsRequest{}
	if !f.From.IsZero() {
		req.StartDate = types.Date{Time: f.From}
	}
	if !f.To.IsZero() {
		req.EndDate = types.Date{Time: f.To}
	}
	if f.Limit > 0 {
		req.Limit = &f.Limit
	}
	if f.Offset > 0 {
		req.Offset = &f.Offset
	}
	if f.Status != "" {
		s := client.RequestDetailsRequestStatus(f.Status)
		req.Status = &s
	}
	resp, err := m.surface.generated.ListRequestsWithResponse(ctx, &client.ListRequestsParams{RequestDetails: req})
	if err != nil {
		return nil, wrapOp(op, err)
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, m.surface.owner.errorFrom(resp.HTTPResponse, op)
	}
	var raw listRequestsRaw
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		return nil, &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
	}
	return raw.toPublic(), nil
}

func (m *RESTRequests) UpdateRequestStatus(ctx context.Context, requestID int64, status string) error {
	const op = "Requests.UpdateStatus"
	if err := m.surface.owner.checkOpen(); err != nil {
		return err
	}
	payload := client.AmRequestStatusResponse{Status: &status}
	data, _ := json.Marshal(payload)
	resp, err := m.surface.generated.UpdateRequestsStatusWithBodyWithResponse(ctx, requestID, "application/json", bytes.NewReader(data))
	if err != nil {
		return wrapOp(op, err)
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		return m.surface.owner.errorFrom(resp.HTTPResponse, op)
	}
	return nil
}

type RESTRequestSummary struct {
	ID            int64
	AccountID     AccountID
	Type          string
	Status        string
	DateSubmitted time.Time
}

type listRequestsRaw struct {
	Limit   *int64                 `json:"limit,omitempty"`
	Offset  *int64                 `json:"offset,omitempty"`
	Details []listRequestDetailRaw `json:"requestDetails,omitempty"`
	Total   *int64                 `json:"total,omitempty"`
}

type listRequestDetailRaw struct {
	AccountID     *string `json:"accountID,omitempty"`
	DateSubmitted *string `json:"dateSubmitted,omitempty"`
	RequestId     *int64  `json:"requestId,omitempty"`
	RequestType   *string `json:"requestType,omitempty"`
	Status        *string `json:"status,omitempty"`
}

func (r *listRequestsRaw) toPublic() []RESTRequestSummary {
	if r.Details == nil {
		return nil
	}
	out := make([]RESTRequestSummary, 0, len(r.Details))
	for _, d := range r.Details {
		var dt time.Time
		if d.DateSubmitted != nil && *d.DateSubmitted != "" {
			dt, _ = time.Parse("2006-01-02", *d.DateSubmitted)
		}
		var id int64
		if d.RequestId != nil {
			id = *d.RequestId
		}
		var acctID AccountID
		if d.AccountID != nil {
			acctID = AccountID(*d.AccountID)
		}
		out = append(out, RESTRequestSummary{
			ID:            id,
			AccountID:     acctID,
			Type:          strPtrVal(d.RequestType),
			Status:        strPtrVal(d.Status),
			DateSubmitted: dt,
		})
	}
	return out
}
