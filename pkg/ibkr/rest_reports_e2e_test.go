// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/shing1211/ibkrapi4go/internal/mockgateway"
)

// restSurfaceWithGateway returns the REST surface together with the mock
// gateway behind it. The reports managers return decoded values *and* have
// request-shaping defaults (format, language, gzip) that are invisible in the
// decoded value, so every test here also reads back the recorded request.
func restSurfaceWithGateway(t *testing.T) (*RESTSurface, *gateway) {
	t.Helper()
	cli, gw := newRESTClientWithGateway(t)
	surface, err := cli.REST()
	if err != nil {
		t.Fatalf("REST: %v", err)
	}
	return surface, gw
}

func TestRESTSurface_GatewayURL(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)
	if got := surface.GatewayURL(); got != gw.URL {
		t.Errorf("GatewayURL = %q; want the configured REST base %q", got, gw.URL)
	}
}

// TestRESTSurface_SubManagerAccessors pins the binding contract of the
// sub-manager accessors: each returns a usable manager over *this* surface, so
// a manager obtained from one surface never talks to a different client's
// credentials or base URL.
func TestRESTSurface_SubManagerAccessors(t *testing.T) {
	surface, _ := restSurfaceWithGateway(t)

	managers := []struct {
		name    string
		surface *RESTSurface
		got     *RESTSurface
	}{
		{"Statements", surface, surface.Statements().surface},
		{"Requests", surface, surface.Requests().surface},
		{"TaxDocuments", surface, surface.TaxDocuments().surface},
		{"TradeConfirmations", surface, surface.TradeConfirmations().surface},
		{"TaxVouchers", surface, surface.TaxVouchers().surface},
	}
	for _, m := range managers {
		if m.got == nil {
			t.Errorf("%s() = nil manager", m.name)
			continue
		}
		if m.got != m.surface {
			t.Errorf("%s().surface = %p; want the originating surface %p", m.name, m.got, m.surface)
		}
	}
}

func TestRESTRequests_Status(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)

	info, err := surface.Requests().Status(context.Background(), 5001)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	// ID echoes the caller's request ID, and is never read from the body.
	if info.ID != 5001 {
		t.Errorf("id = %d; want the requested 5001", info.ID)
	}
	// The shared default body is a spec shape: `dateSubmitted` is the one
	// timestamp the StatusResponse variant of the oneOf carries, so the default
	// path decodes it without a per-test override. The gateway sent the instant
	// with a non-UTC offset and the wrapper normalises it. The offset is
	// -05:00, which carries the instant forward past midnight, so the expected
	// value lands on a different day than the wire string: it is only reachable
	// if the offset was genuinely applied, not copied or dropped.
	if info.ExecutedAt == nil {
		t.Fatal("executedAt = nil; want the dateSubmitted timestamp from the shared default fixture")
	}
	if *info.ExecutedAt != "2026-01-03T01:30:00Z" {
		t.Errorf("executedAt = %q; want 2026-01-03T01:30:00Z (the default fixture's instant, normalised to UTC)", *info.ExecutedAt)
	}

	req := lastRESTRequest(t, gw)
	if req.Method != http.MethodGet {
		t.Errorf("method = %s; want GET", req.Method)
	}
	if req.Path != "/gw/api/v1/requests/5001/status" {
		t.Errorf("path = %q; want /gw/api/v1/requests/5001/status", req.Path)
	}
}

// TestRESTRequests_Status_Timestamp pins the success branch's payload decode.
// The upstream 200 body is a oneOf whose StatusResponse variant carries
// `dateSubmitted`; the wrapper unwraps it and reports the value, so the exported
// ExecutedAt field is no longer permanently nil. The gateway sent the instant
// with a non-UTC offset and the wrapper normalises it, which is the documented
// contract on the field.
func TestRESTRequests_Status_Timestamp(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)
	gw.srv.Fixtures().Set(mockgateway.OpGetRequestsStatus, mockgateway.Fixture{
		Body: `{"requestId":5001,"dateSubmitted":"2026-01-02T16:04:05+01:00"}`,
	})

	info, err := surface.Requests().Status(context.Background(), 5001)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if info.ExecutedAt == nil {
		t.Fatal("executedAt = nil; want the dateSubmitted timestamp from the payload")
	}
	if *info.ExecutedAt != "2026-01-02T15:04:05Z" {
		t.Errorf("executedAt = %q; want 2026-01-02T15:04:05Z (the fixture's instant, normalised to UTC)", *info.ExecutedAt)
	}
	// The value is a real instant, not just a copy of the wire string: it has to
	// parse back as RFC 3339.
	if _, err := time.Parse(time.RFC3339, *info.ExecutedAt); err != nil {
		t.Errorf("executedAt = %q; does not parse as RFC 3339: %v", *info.ExecutedAt, err)
	}
	if info.ID != 5001 {
		t.Errorf("id = %d; want the requested 5001", info.ID)
	}
}

// TestRESTRequests_Status_NoTimestampVariant pins the other arm of the oneOf:
// the AmRequestStatusResponse variant has no time field, and it decodes as a
// StatusResponse only if the payload does not use the variant's string requestId.
// Neither may turn into an error or a fabricated timestamp — a 200 with a valid
// body is a success, and the caller's request ID is still echoed.
func TestRESTRequests_Status_NoTimestampVariant(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)
	gw.srv.Fixtures().Set(mockgateway.OpGetRequestsStatus, mockgateway.Fixture{
		Body: `{"requestId":"5001","requestType":"ACCOUNT_UPDATE","status":"COMPLETED",` +
			`"acctId":"U1234567","message":"done"}`,
	})

	info, err := surface.Requests().Status(context.Background(), 5001)
	if err != nil {
		t.Fatalf("Status on the AmRequestStatusResponse variant: %v", err)
	}
	if info.ExecutedAt != nil {
		t.Errorf("executedAt = %q; want nil for a variant that carries no timestamp", *info.ExecutedAt)
	}
	if info.ID != 5001 {
		t.Errorf("id = %d; want the requested 5001, not the body's string requestId", info.ID)
	}
}

// TestRESTRequests_Status_ServerError covers the >= 400 guard on its own: the
// success path now decodes the payload, so the 200 tests above cannot show that
// the guard runs. The mock's 500 body is the standard IBKR error envelope, so
// errorFrom can classify it.
func TestRESTRequests_Status_ServerError(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)
	gw.srv.Fixtures().Set(mockgateway.OpGetRequestsStatus, mockgateway.Fixture{
		Status: http.StatusInternalServerError,
		Body:   `{"error":"boom","message":"synthetic failure","code":"internal"}`,
	})

	info, err := surface.Requests().Status(context.Background(), 5001)
	if err == nil {
		t.Fatalf("Status on 500 = %+v, nil error; want an error", info)
	}
	if info != nil {
		t.Errorf("info = %+v; want nil alongside the error", info)
	}
	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("err = %v (%T); want *Error", err, err)
	}
	if e.Op != "Requests.GetStatus" {
		t.Errorf("op = %q; want Requests.GetStatus", e.Op)
	}
	// The guard builds a typed http_error carrying the status and hands it to
	// wrapOp, which adopts it rather than burying it, so the status and code are
	// readable on the error the caller gets — the errors.As(e.HTTPStatus) idiom
	// in docs/ERRORS.md works without reaching into .Err.
	if e.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("httpStatus = %d; want 500", e.HTTPStatus)
	}
	if e.Code != "http_error" {
		t.Errorf("code = %q; want http_error", e.Code)
	}
}

// TestRESTReports_TypedHTTPErrorExposesStatus covers the three remaining >= 400
// guards that hand wrapOp a *Error they built themselves (tax documents and
// trade confirmations; Requests.GetStatus is pinned by
// TestRESTRequests_Status_ServerError). All four are the same shape, so they are
// table-driven: the typed http_error must reach the caller intact, which is what
// the errors.As(err, &e); e.HTTPStatus idiom in docs/ERRORS.md depends on. A
// regression that re-wraps the error leaves HTTPStatus 0 and Code empty here.
func TestRESTReports_TypedHTTPErrorExposesStatus(t *testing.T) {
	cases := []struct {
		name    string
		op      string
		fixture string
		call    func(*RESTSurface) error
	}{
		{
			name:    "TaxDocuments.ListAvailable",
			op:      "TaxDocuments.ListAvailable",
			fixture: mockgateway.OpListTaxDocumentsAvailable,
			call: func(s *RESTSurface) error {
				_, err := s.TaxDocuments().ListAvailable(context.Background(), AccountID("U1234567"))
				return err
			},
		},
		{
			name:    "TaxDocuments.Generate",
			op:      "TaxDocuments.Generate",
			fixture: mockgateway.OpCreateTaxDocuments,
			call: func(s *RESTSurface) error {
				_, err := s.TaxDocuments().Generate(context.Background(), TaxDocumentRequest{
					AccountID: AccountID("U1234567"),
					Year:      "2025",
					Type:      "ALL",
				})
				return err
			},
		},
		{
			name:    "TradeConfirmations.ListAvailable",
			op:      "TradeConfirmations.ListAvailable",
			fixture: mockgateway.OpListTradeConfirmationsAvailable,
			call: func(s *RESTSurface) error {
				_, err := s.TradeConfirmations().ListAvailable(context.Background(), AccountID("U1234567"))
				return err
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			surface, gw := restSurfaceWithGateway(t)
			gw.srv.Fixtures().Set(tc.fixture, mockgateway.Fixture{
				Status: http.StatusInternalServerError,
				Body:   `{"error":"boom","message":"synthetic failure"}`,
			})

			err := tc.call(surface)
			if err == nil {
				t.Fatalf("call on 500 = nil error; want an error")
			}
			// Reached the way a caller reaches it, with no second hop through
			// .Err: the returned value itself has to be the typed error.
			var e *Error
			if !errors.As(err, &e) {
				t.Fatalf("err = %v (%T); want *Error", err, err)
			}
			if e.Op != tc.op {
				t.Errorf("op = %q; want %s", e.Op, tc.op)
			}
			if e.HTTPStatus != http.StatusInternalServerError {
				t.Errorf("httpStatus = %d; want 500", e.HTTPStatus)
			}
			if e.Code != "http_error" {
				t.Errorf("code = %q; want http_error", e.Code)
			}
		})
	}
}

func TestRESTTaxDocuments_ListAvailable(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)

	types, err := surface.TaxDocuments().ListAvailable(context.Background(), AccountID("U1234567"))
	if err != nil {
		t.Fatalf("ListAvailable: %v", err)
	}
	if len(types.Forms) != 2 {
		t.Fatalf("forms = %v; want two", types.Forms)
	}
	if types.Forms[0] != "1099" || types.Forms[1] != "1042S" {
		t.Errorf("forms = %v; want [1099 1042S]", types.Forms)
	}

	req := lastRESTRequest(t, gw)
	if req.Method != http.MethodGet {
		t.Errorf("method = %s; want GET", req.Method)
	}
	if req.Path != "/gw/api/v1/tax-documents/available" {
		t.Errorf("path = %q; want /gw/api/v1/tax-documents/available", req.Path)
	}
	if got := req.Query.Get("accountId"); got != "U1234567" {
		t.Errorf("accountId = %q; want U1234567", got)
	}
	// The generated params carry a non-pointer `year`, so the wrapper cannot
	// leave it out: it is sent present-and-empty. Pinned so a future change to
	// omit it (or to start forwarding a year) has to be deliberate.
	if got, ok := req.Query["year"]; !ok || len(got) != 1 || got[0] != "" {
		t.Errorf("year = %v (present=%t); want exactly one empty value", got, ok)
	}
}

func TestRESTTaxDocuments_Generate(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)

	// An empty Format must be defaulted to PDF on the wire.
	doc, err := surface.TaxDocuments().Generate(context.Background(), TaxDocumentRequest{
		AccountID: AccountID("U1234567"),
		Year:      "2025",
		Type:      "1099",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if doc.ContentType != "application/pdf" {
		t.Errorf("contentType = %q; want application/pdf from the fixture", doc.ContentType)
	}
	if string(doc.Data) != "JVBERi0xLjQK" {
		t.Errorf("data = %q; want the fixture's base64 payload", doc.Data)
	}
	if doc.Gzip {
		t.Error("gzip = true; the fixture encodes false")
	}

	req := lastRESTRequest(t, gw)
	if req.Method != http.MethodPost {
		t.Errorf("method = %s; want POST", req.Method)
	}
	if req.Path != "/gw/api/v1/tax-documents" {
		t.Errorf("path = %q; want /gw/api/v1/tax-documents", req.Path)
	}
	if got := req.Headers.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q; want application/json", got)
	}
	var got struct {
		AccountID string `json:"accountId"`
		Year      string `json:"year"`
		Type      string `json:"type"`
		Format    string `json:"format"`
	}
	decodeRequestBody(t, req, &got)
	if got.AccountID != "U1234567" {
		t.Errorf("accountId = %q; want U1234567", got.AccountID)
	}
	if got.Year != "2025" {
		t.Errorf("year = %q; want 2025", got.Year)
	}
	if got.Type != "1099" {
		t.Errorf("type = %q; want 1099", got.Type)
	}
	if got.Format != "PDF" {
		t.Errorf("format = %q; want the defaulted PDF", got.Format)
	}

	// An explicit Format must be forwarded rather than overwritten by the
	// default.
	if _, err := surface.TaxDocuments().Generate(context.Background(), TaxDocumentRequest{
		AccountID: AccountID("U1234567"),
		Year:      "2025",
		Type:      "1042S",
		Format:    "HTML",
	}); err != nil {
		t.Fatalf("Generate with Format: %v", err)
	}
	req = lastRESTRequest(t, gw)
	decodeRequestBody(t, req, &got)
	if got.Format != "HTML" {
		t.Errorf("format = %q; want the explicit HTML", got.Format)
	}
	if got.Type != "1042S" {
		t.Errorf("type = %q; want 1042S", got.Type)
	}
}

// TestRESTTaxDocuments_Generate_MissingDocument pins the empty-envelope branch:
// a 200 whose body carries no data.value must yield a zero-valued response
// rather than a nil pointer or an error.
func TestRESTTaxDocuments_Generate_MissingDocument(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)
	gw.srv.Fixtures().Set(mockgateway.OpCreateTaxDocuments, mockgateway.Fixture{
		Body: `{"data":{}}`,
	})

	doc, err := surface.TaxDocuments().Generate(context.Background(), TaxDocumentRequest{
		AccountID: AccountID("U1234567"),
		Year:      "2025",
		Type:      "ALL",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if doc == nil {
		t.Fatal("Generate = nil response; want a zero-valued struct")
	}
	if len(doc.Data) != 0 {
		t.Errorf("data = %q; want empty for a body with no data.value", doc.Data)
	}
	if doc.ContentType != "" || doc.Gzip {
		t.Errorf("doc = %+v; want zero-valued", doc)
	}
}

func TestRESTTradeConfirmations_ListAvailable(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)

	dates, err := surface.TradeConfirmations().ListAvailable(context.Background(), AccountID("U1234567"))
	if err != nil {
		t.Fatalf("ListAvailable: %v", err)
	}
	if len(dates.Dates) != 2 {
		t.Fatalf("dates = %v; want two", dates.Dates)
	}
	if dates.Dates[0] != "2026-01-02" || dates.Dates[1] != "2026-01-03" {
		t.Errorf("dates = %v; want [2026-01-02 2026-01-03]", dates.Dates)
	}

	req := lastRESTRequest(t, gw)
	if req.Method != http.MethodGet {
		t.Errorf("method = %s; want GET", req.Method)
	}
	if req.Path != "/gw/api/v1/trade-confirmations/available" {
		t.Errorf("path = %q; want /gw/api/v1/trade-confirmations/available", req.Path)
	}
	if got := req.Query.Get("accountId"); got != "U1234567" {
		t.Errorf("accountId = %q; want U1234567", got)
	}
	// This method no longer reads a token itself: the transport's Auth
	// middleware is the only thing that writes the Authorization header, so
	// this assertion is what proves removing that read did not silently drop
	// authentication. It pins the exact value — not merely a "Bearer " prefix —
	// so a header carrying an empty token cannot satisfy it.
	tok, err := surface.Token(context.Background())
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if tok == "" {
		t.Fatal("Token = \"\"; want a non-empty token to compare against")
	}
	if got, want := req.Headers.Get("Authorization"), "Bearer "+tok; got != want {
		t.Errorf("Authorization = %q; want %q from the transport's Auth middleware", got, want)
	}
}

func TestRESTTradeConfirmations_Generate(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)

	// An empty Format must be defaulted to application/pdf on the wire.
	conf, err := surface.TradeConfirmations().Generate(context.Background(), TradeConfirmationRequest{
		AccountID: AccountID("U1234567"),
		StartDate: "20260102",
		EndDate:   "20260103",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if conf.ContentType != "application/pdf" {
		t.Errorf("contentType = %q; want application/pdf from the fixture", conf.ContentType)
	}
	if string(conf.Data) != "JVBERi0xLjQK" {
		t.Errorf("data = %q; want the fixture's base64 payload", conf.Data)
	}
	if conf.Gzip {
		t.Error("gzip = true; the fixture encodes false")
	}

	req := lastRESTRequest(t, gw)
	if req.Method != http.MethodPost {
		t.Errorf("method = %s; want POST", req.Method)
	}
	if req.Path != "/gw/api/v1/trade-confirmations" {
		t.Errorf("path = %q; want /gw/api/v1/trade-confirmations", req.Path)
	}
	if got := req.Headers.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q; want application/json", got)
	}
	var got struct {
		AccountID string `json:"accountId"`
		StartDate string `json:"startDate"`
		EndDate   string `json:"endDate"`
		MimeType  string `json:"mimeType"`
	}
	decodeRequestBody(t, req, &got)
	if got.AccountID != "U1234567" {
		t.Errorf("accountId = %q; want U1234567", got.AccountID)
	}
	if got.StartDate != "20260102" {
		t.Errorf("startDate = %q; want 20260102", got.StartDate)
	}
	if got.EndDate != "20260103" {
		t.Errorf("endDate = %q; want 20260103", got.EndDate)
	}
	if got.MimeType != "application/pdf" {
		t.Errorf("mimeType = %q; want the defaulted application/pdf", got.MimeType)
	}
	// The public TradeConfirmationRequest exposes a Gzip flag, but the
	// upstream body schema has no gzip property, so the wrapper cannot forward
	// it. The exact key set is asserted so a future spec change that adds one is
	// visible as an addition rather than a silent change.
	var keys map[string]any
	decodeRequestBody(t, req, &keys)
	if _, ok := keys["gzip"]; ok {
		t.Errorf("body = %s; the generated model has no gzip field, so it is never sent", req.Body)
	}

	// An explicit Format must be forwarded rather than overwritten.
	if _, err := surface.TradeConfirmations().Generate(context.Background(), TradeConfirmationRequest{
		AccountID: AccountID("U1234567"),
		StartDate: "20260102",
		EndDate:   "20260103",
		Format:    "text/html",
	}); err != nil {
		t.Fatalf("Generate with Format: %v", err)
	}
	req = lastRESTRequest(t, gw)
	decodeRequestBody(t, req, &got)
	if got.MimeType != "text/html" {
		t.Errorf("mimeType = %q; want the explicit text/html", got.MimeType)
	}
}

// TestRESTTradeConfirmations_Generate_GzipNotSent pins the documented limitation
// on TradeConfirmationRequest.Gzip. The upstream createTradeConfirmations body
// schema has no gzip property (unlike StmtRequest's, which the statements wrapper
// does forward — see TestRESTStatements_Generate), so a caller asking for gzip
// gets the same request body as one that did not. The field is kept for
// compatibility, so the contract to assert is that the flag is inert and the rest
// of the body is unaffected — the exact key set, so a spec change that adds a
// gzip property fails here instead of passing silently.
func TestRESTTradeConfirmations_Generate_GzipNotSent(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)

	conf, err := surface.TradeConfirmations().Generate(context.Background(), TradeConfirmationRequest{
		AccountID: AccountID("U1234567"),
		StartDate: "20260102",
		EndDate:   "20260103",
		Format:    "text/html",
		Gzip:      true,
	})
	if err != nil {
		t.Fatalf("Generate with Gzip: %v", err)
	}
	// The request still succeeds and still decodes: the flag is ignored, not
	// fatal.
	if conf.ContentType != "application/pdf" {
		t.Errorf("contentType = %q; want application/pdf from the fixture", conf.ContentType)
	}
	if string(conf.Data) != "JVBERi0xLjQK" {
		t.Errorf("data = %q; want the fixture's base64 payload", conf.Data)
	}

	req := lastRESTRequest(t, gw)
	var keys map[string]any
	decodeRequestBody(t, req, &keys)
	if _, ok := keys["gzip"]; ok {
		t.Errorf("body = %s; want no gzip key — the upstream schema has none to fill", req.Body)
	}
	want := []string{"accountId", "endDate", "mimeType", "startDate"}
	got := make([]string, 0, len(keys))
	for k := range keys {
		got = append(got, k)
	}
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("body keys = %v; want exactly %v", got, want)
	}
	// Everything else still reaches the wire, so the ignored flag is the only
	// difference from a request without it.
	var fields struct {
		AccountID string `json:"accountId"`
		StartDate string `json:"startDate"`
		EndDate   string `json:"endDate"`
		MimeType  string `json:"mimeType"`
	}
	decodeRequestBody(t, req, &fields)
	if fields.AccountID != "U1234567" || fields.StartDate != "20260102" ||
		fields.EndDate != "20260103" || fields.MimeType != "text/html" {
		t.Errorf("body = %s; want the caller's fields unchanged", req.Body)
	}
}

func TestRESTStatements_Generate(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)

	// Defaults: Format -> application/pdf, Language -> en, and an empty
	// AccountIDs slice must be omitted from the body.
	stmt, err := surface.Statements().Generate(context.Background(), StatementRequest{
		AccountID: AccountID("U1234567"),
		StartDate: "2026-01-01",
		EndDate:   "2026-01-31",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if stmt.ContentType != "application/pdf" {
		t.Errorf("contentType = %q; want application/pdf from the fixture", stmt.ContentType)
	}
	if string(stmt.Data) != "JVBERi0xLjQK" {
		t.Errorf("data = %q; want the fixture's base64 payload", stmt.Data)
	}
	if stmt.Gzip {
		t.Error("gzip = true; the fixture encodes false")
	}

	req := lastRESTRequest(t, gw)
	if req.Method != http.MethodPost {
		t.Errorf("method = %s; want POST", req.Method)
	}
	if req.Path != "/gw/api/v1/statements" {
		t.Errorf("path = %q; want /gw/api/v1/statements", req.Path)
	}
	if got := req.Headers.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q; want application/json", got)
	}
	var got struct {
		AccountID  string   `json:"accountId"`
		StartDate  string   `json:"startDate"`
		EndDate    string   `json:"endDate"`
		MimeType   string   `json:"mimeType"`
		Language   string   `json:"language"`
		Gzip       bool     `json:"gzip"`
		AccountIDs []string `json:"accountIds"`
	}
	decodeRequestBody(t, req, &got)
	if got.AccountID != "U1234567" {
		t.Errorf("accountId = %q; want U1234567", got.AccountID)
	}
	if got.StartDate != "2026-01-01" || got.EndDate != "2026-01-31" {
		t.Errorf("dates = %q..%q; want 2026-01-01..2026-01-31", got.StartDate, got.EndDate)
	}
	if got.MimeType != "application/pdf" {
		t.Errorf("mimeType = %q; want the defaulted application/pdf", got.MimeType)
	}
	if got.Language != "en" {
		t.Errorf("language = %q; want the defaulted en", got.Language)
	}
	if got.Gzip {
		t.Error("gzip = true; the request's Gzip was false")
	}
	// accountIds is omitempty on the generated model, so a single-account
	// request must not carry the key at all.
	var keys map[string]any
	decodeRequestBody(t, req, &keys)
	if _, ok := keys["accountIds"]; ok {
		t.Errorf("body = %s; want no accountIds key when AccountIDs is empty", req.Body)
	}

	// Every explicit value must reach the wire, including a populated
	// AccountIDs list and a true Gzip.
	if _, err := surface.Statements().Generate(context.Background(), StatementRequest{
		AccountID:  AccountID("U1234567"),
		AccountIDs: []AccountID{AccountID("U1234567"), AccountID("U7654321")},
		StartDate:  "2026-01-01",
		EndDate:    "2026-01-31",
		Format:     "text/csv",
		Language:   "fr",
		Gzip:       true,
	}); err != nil {
		t.Fatalf("Generate with explicit fields: %v", err)
	}
	req = lastRESTRequest(t, gw)
	decodeRequestBody(t, req, &got)
	if got.MimeType != "text/csv" {
		t.Errorf("mimeType = %q; want the explicit text/csv", got.MimeType)
	}
	if got.Language != "fr" {
		t.Errorf("language = %q; want the explicit fr", got.Language)
	}
	if !got.Gzip {
		t.Error("gzip = false; want the explicit true to be sent")
	}
	if len(got.AccountIDs) != 2 || got.AccountIDs[0] != "U1234567" || got.AccountIDs[1] != "U7654321" {
		t.Errorf("accountIds = %v; want [U1234567 U7654321]", got.AccountIDs)
	}
}

func TestRESTStatements_ListAvailable(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)

	dates, err := surface.Statements().ListAvailable(context.Background(), AccountID("U1234567"))
	if err != nil {
		t.Fatalf("ListAvailable: %v", err)
	}
	if len(dates.Annual) != 1 || dates.Annual[0] != "2025" {
		t.Errorf("annual = %v; want [2025]", dates.Annual)
	}
	if len(dates.Monthly) != 1 || dates.Monthly[0] != "2026-01" {
		t.Errorf("monthly = %v; want [2026-01]", dates.Monthly)
	}
	if dates.DailyStart != "2026-01-01" {
		t.Errorf("dailyStart = %q; want 2026-01-01", dates.DailyStart)
	}
	if dates.DailyEnd != "2026-01-31" {
		t.Errorf("dailyEnd = %q; want 2026-01-31", dates.DailyEnd)
	}

	req := lastRESTRequest(t, gw)
	if req.Method != http.MethodGet {
		t.Errorf("method = %s; want GET", req.Method)
	}
	if req.Path != "/gw/api/v1/statements/available" {
		t.Errorf("path = %q; want /gw/api/v1/statements/available", req.Path)
	}
	if got := req.Query.Get("accountId"); got != "U1234567" {
		t.Errorf("accountId = %q; want U1234567", got)
	}
}

// TestRESTStatements_ListAvailable_EmptyEnvelope pins the absent-data branch: a
// 200 carrying no data key must decode to a zero-valued result, not an error.
func TestRESTStatements_ListAvailable_EmptyEnvelope(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)
	gw.srv.Fixtures().Set(mockgateway.OpListStatementsAvailable, mockgateway.Fixture{Body: `{}`})

	dates, err := surface.Statements().ListAvailable(context.Background(), AccountID("U1234567"))
	if err != nil {
		t.Fatalf("ListAvailable: %v", err)
	}
	if dates == nil {
		t.Fatal("ListAvailable = nil; want a zero-valued struct")
	}
	if len(dates.Annual) != 0 || len(dates.Monthly) != 0 {
		t.Errorf("dates = %+v; want empty lists", dates)
	}
	if dates.DailyStart != "" || dates.DailyEnd != "" {
		t.Errorf("daily = %q..%q; want empty", dates.DailyStart, dates.DailyEnd)
	}
}
