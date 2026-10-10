// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"testing"

	"github.com/shing1211/ibkrapi4go/internal/mockgateway"
)

// voucherCSV is the request body CreateRequests must forward verbatim: the
// wrapper posts the caller's CSV as text/plain and does not re-encode it.
const voucherCSV = "corpactionId,countryCode,custAcctId\nca-1,US,U1234567\n"

func TestRESTTaxVouchers_CreateRequests(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)

	id, err := surface.TaxVouchers().CreateRequests(context.Background(), voucherCSV)
	if err != nil {
		t.Fatalf("CreateRequests: %v", err)
	}
	if id != "tv-1" {
		t.Errorf("requestID = %q; want tv-1", id)
	}

	req := lastRESTRequest(t, gw)
	if req.Method != http.MethodPost {
		t.Errorf("method = %s; want POST", req.Method)
	}
	if req.Path != "/gw/api/v1/tax-vouchers" {
		t.Errorf("path = %q; want /gw/api/v1/tax-vouchers", req.Path)
	}
	if got := req.Headers.Get("Content-Type"); got != "text/plain" {
		t.Errorf("Content-Type = %q; want text/plain", got)
	}
	if string(req.Body) != voucherCSV {
		t.Errorf("body = %q; want the CSV verbatim", req.Body)
	}
}

// TestRESTTaxVouchers_CreateRequests_EmptyResult pins the documented
// empty-body case: a 200 whose JSON array is empty yields an empty request ID
// and no error, rather than panicking on the zero-th element.
func TestRESTTaxVouchers_CreateRequests_EmptyResult(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)
	gw.srv.Fixtures().Set(mockgateway.OpCreateTaxVoucherRequests, mockgateway.Fixture{Body: `[]`})

	id, err := surface.TaxVouchers().CreateRequests(context.Background(), voucherCSV)
	if err != nil {
		t.Fatalf("CreateRequests: %v", err)
	}
	if id != "" {
		t.Errorf("requestID = %q; want empty for an empty result array", id)
	}
}

func TestRESTTaxVouchers_ActiveCountries(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)

	countries, err := surface.TaxVouchers().ActiveCountries(context.Background())
	if err != nil {
		t.Fatalf("ActiveCountries: %v", err)
	}
	if len(countries) != 2 {
		t.Fatalf("countries = %v; want two", countries)
	}
	// The spec's Country schema describes neither field, so the wrapper follows
	// the field names: `country` is the display name, `countryCode` the ISO code.
	// Country is the only schema in the spec that carries both, which is what
	// settles it — elsewhere a schema that wants a code puts it in `country`
	// (ResidenceAddress.example is "GBR"), because there is no sibling to hold it.
	// Asserted on the names so a switch to countryCode is a visible change, and
	// checked against the codes too, so the test cannot pass on a value that is
	// both.
	if countries[0] != "United States" || countries[1] != "United Kingdom" {
		t.Errorf("countries = %v; want the country display names", countries)
	}
	for _, c := range countries {
		if c == "US" || c == "GB" {
			t.Errorf("countries = %v; contains an ISO code, want display names only", countries)
		}
	}

	req := lastRESTRequest(t, gw)
	if req.Method != http.MethodGet {
		t.Errorf("method = %s; want GET", req.Method)
	}
	if req.Path != "/gw/api/v1/tax-vouchers/active-countries" {
		t.Errorf("path = %q; want /gw/api/v1/tax-vouchers/active-countries", req.Path)
	}
}

func TestRESTTaxVouchers_Dividends(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)
	// The shared dividend fixture carries no `voucher` object, so every money
	// field would decode to its zero value and the float32 formatting in the
	// wrapper would never run. Override it with a response that does carry one.
	gw.srv.Fixtures().Set(mockgateway.OpFetchDividends1, mockgateway.Fixture{
		Body: `[{"corpactionId":"ca-1","country":"US","currency":"USD","exDate":"2026-01-02",` +
			`"isin":"US0378331005","payDate":"2026-01-15","securityDesc":"Apple Inc","symbol":"AAPL",` +
			`"voucher":{"corpactionId":"ca-1","countryCode":"US","custAcctId":"U7654321",` +
			`"divAmount":12.5,"fee":0.75,"quantity":10,"requestId":"tv-1",` +
			`"withHeldAmount":1.25,"year":2025}},` +
			`{"corpactionId":"ca-2","country":"GB","symbol":"VOD","voucher":null}]`,
	})

	dividends, err := surface.TaxVouchers().Dividends(context.Background(), AccountID("U1234567"), "2025", "US")
	if err != nil {
		t.Fatalf("Dividends: %v", err)
	}
	if len(dividends) != 2 {
		t.Fatalf("dividends = %+v; want two", dividends)
	}

	got := dividends[0]
	if got.CorpActionID != "ca-1" {
		t.Errorf("corpActionId = %q; want ca-1", got.CorpActionID)
	}
	// CountryCode comes from the envelope's `country`, not the voucher's
	// countryCode.
	if got.CountryCode != "US" {
		t.Errorf("countryCode = %q; want US", got.CountryCode)
	}
	// AccountID is the account the caller asked about; the voucher's own
	// custAcctId (U7654321) is deliberately not used.
	if got.AccountID != AccountID("U1234567") {
		t.Errorf("accountId = %q; want the requested U1234567", got.AccountID)
	}
	// Money and quantity stay strings (ADR 0008) rather than binary floats.
	if got.Amount != "12.5" {
		t.Errorf("amount = %q; want \"12.5\"", got.Amount)
	}
	if got.Fee != "0.75" {
		t.Errorf("fee = %q; want \"0.75\"", got.Fee)
	}
	if got.Quantity != "10" {
		t.Errorf("quantity = %q; want \"10\"", got.Quantity)
	}
	if got.WithheldAmount != "1.25" {
		t.Errorf("withheldAmount = %q; want \"1.25\"", got.WithheldAmount)
	}
	if got.RequestID != "tv-1" {
		t.Errorf("requestId = %q; want tv-1", got.RequestID)
	}
	if got.Year != 2025 {
		t.Errorf("year = %d; want 2025", got.Year)
	}

	// A record with no voucher must fall back to empty money fields and year 0
	// rather than panicking on the nil pointer.
	empty := dividends[1]
	if empty.CorpActionID != "ca-2" || empty.CountryCode != "GB" {
		t.Errorf("dividends[1] = %+v; want ca-2/GB", empty)
	}
	if empty.Amount != "" || empty.Fee != "" || empty.Quantity != "" || empty.WithheldAmount != "" {
		t.Errorf("dividends[1] money = %q/%q/%q/%q; want all empty for a null voucher",
			empty.Amount, empty.Fee, empty.Quantity, empty.WithheldAmount)
	}
	if empty.RequestID != "" {
		t.Errorf("dividends[1].requestId = %q; want empty", empty.RequestID)
	}
	if empty.Year != 0 {
		t.Errorf("dividends[1].year = %d; want 0", empty.Year)
	}
	if empty.AccountID != AccountID("U1234567") {
		t.Errorf("dividends[1].accountId = %q; want the requested U1234567", empty.AccountID)
	}

	req := lastRESTRequest(t, gw)
	if req.Method != http.MethodGet {
		t.Errorf("method = %s; want GET", req.Method)
	}
	if req.Path != "/gw/api/v1/tax-vouchers/dividends" {
		t.Errorf("path = %q; want /gw/api/v1/tax-vouchers/dividends", req.Path)
	}
	for _, q := range []struct{ key, want string }{
		{"custAcctId", "U1234567"},
		{"year", "2025"},
		{"countryCode", "US"},
	} {
		if got := req.Query.Get(q.key); got != q.want {
			t.Errorf("query %s = %q; want %q", q.key, got, q.want)
		}
	}
}

func TestRESTTaxVouchers_AvailableYears(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)

	years, err := surface.TaxVouchers().AvailableYears(context.Background())
	if err != nil {
		t.Fatalf("AvailableYears: %v", err)
	}
	if len(years) != 2 {
		t.Fatalf("years = %v; want two", years)
	}
	// The API returns JSON integers; the wrapper renders them as decimal strings.
	if years[0] != "2025" || years[1] != "2024" {
		t.Errorf("years = %v; want [2025 2024]", years)
	}

	req := lastRESTRequest(t, gw)
	if req.Method != http.MethodGet {
		t.Errorf("method = %s; want GET", req.Method)
	}
	if req.Path != "/gw/api/v1/tax-vouchers/years" {
		t.Errorf("path = %q; want /gw/api/v1/tax-vouchers/years", req.Path)
	}
}

func TestRESTTaxVouchers_Download(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)

	data, err := surface.TaxVouchers().Download(context.Background(), "tv-1")
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	// The wrapper hands back the undecoded response body, so the JSON string
	// literal arrives with its quotes intact. The caller is responsible for
	// interpreting the content type, so nothing here base64-decodes it.
	if string(data) != `"c3ludGhldGljLXRheC12b3VjaGVy"` {
		t.Errorf("data = %q; want the raw undecoded body", data)
	}

	req := lastRESTRequest(t, gw)
	if req.Method != http.MethodGet {
		t.Errorf("method = %s; want GET", req.Method)
	}
	if req.Path != "/gw/api/v1/tax-vouchers/tv-1/download" {
		t.Errorf("path = %q; want /gw/api/v1/tax-vouchers/tv-1/download", req.Path)
	}
}

func TestRESTTaxVouchers_RequestState(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)

	state, err := surface.TaxVouchers().RequestState(context.Background(), "tv-1")
	if err != nil {
		t.Fatalf("RequestState: %v", err)
	}
	if state.RequestID != "tv-1" {
		t.Errorf("requestId = %q; want tv-1", state.RequestID)
	}
	if state.RequestState != "COMPLETED" {
		t.Errorf("requestState = %q; want COMPLETED", state.RequestState)
	}

	req := lastRESTRequest(t, gw)
	if req.Method != http.MethodGet {
		t.Errorf("method = %s; want GET", req.Method)
	}
	if req.Path != "/gw/api/v1/tax-vouchers/tv-1/state" {
		t.Errorf("path = %q; want /gw/api/v1/tax-vouchers/tv-1/state", req.Path)
	}
}

// TestRESTTaxVouchers_RequestState_ServerError pins the failure branch: a 4xx/5xx
// must surface an error rather than a zero-valued state that looks like a real
// answer.
func TestRESTTaxVouchers_RequestState_ServerError(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)
	gw.srv.Fixtures().Set(mockgateway.OpGetCurrentState1, mockgateway.Fixture{
		Status: http.StatusNotFound,
		Body:   `{"error":"not found","message":"no such voucher request"}`,
	})

	state, err := surface.TaxVouchers().RequestState(context.Background(), "missing")
	if err == nil {
		t.Fatalf("RequestState on 404 = %+v, nil error; want an error", state)
	}
	if state != nil {
		t.Errorf("state = %+v; want nil alongside the error", state)
	}
	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("err = %v (%T); want *Error", err, err)
	}
	if e.Op != "TaxVouchers.RequestState" {
		t.Errorf("op = %q; want TaxVouchers.RequestState", e.Op)
	}
	if e.HTTPStatus != http.StatusNotFound {
		t.Errorf("httpStatus = %d; want 404", e.HTTPStatus)
	}
}

// TestRESTTaxVouchers_Dividends_MoneyPrecision pins the reason the voucher's
// money fields are json.Number rather than float32.
//
// Every other tax-voucher assertion uses 12.5 / 0.75 / 1.25, all of which a
// float32 represents exactly - so they pass whether the field is a float32 or a
// json.Number and prove nothing about precision. A float32 mantissa is 24 bits,
// so it rounds anything above 2^24 (16777216); an aggregate withholding figure
// can exceed that, and the old code re-emitted the rounded value as the
// caller's money string with nothing to indicate a loss.
func TestRESTTaxVouchers_Dividends_MoneyPrecision(t *testing.T) {
	surface, gw := restSurfaceWithGateway(t)
	gw.srv.Fixtures().Set(mockgateway.OpFetchDividends1, mockgateway.Fixture{
		Body: `[{"corpactionId":"ca-1","country":"US","currency":"USD","exDate":"2026-01-02",` +
			`"isin":"US0378331005","payDate":"2026-01-15","securityDesc":"Apple Inc","symbol":"AAPL",` +
			`"voucher":{"corpactionId":"ca-1","countryCode":"US","custAcctId":"U7654321",` +
			`"divAmount":12345678.91,"fee":0.007,"quantity":10,"requestId":"tv-prec",` +
			`"withHeldAmount":33554432.55,"year":2025}}]`,
	})

	dividends, err := surface.TaxVouchers().Dividends(context.Background(), AccountID("U1234567"), "2025", "US")
	if err != nil {
		t.Fatalf("Dividends: %v", err)
	}
	if len(dividends) != 1 {
		t.Fatalf("dividends = %+v; want one", dividends)
	}

	// Each expected value is the gateway's own literal, byte for byte. A float32
	// decode would yield 12345678 and 33554432 here.
	for _, tc := range []struct{ name, got, want string }{
		{"amount", dividends[0].Amount, "12345678.91"},
		{"fee", dividends[0].Fee, "0.007"},
		{"withheldAmount", dividends[0].WithheldAmount, "33554432.55"},
		{"quantity", dividends[0].Quantity, "10"},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q; want %q - the gateway's digits must survive verbatim", tc.name, tc.got, tc.want)
		}
	}

	// Show the loss the old path produced, so the expectation above is not just
	// a literal someone picked. float32 holds 24 mantissa bits; these two values
	// exceed 2^24 and are rounded to a whole number of units.
	var divAmount float32 = 12345678.91
	var withheld float32 = 33554432.55
	if old := strconv.FormatFloat(float64(divAmount), 'f', -1, 32); old == "12345678.91" {
		t.Errorf("float32 held 12345678.91 exactly (%q); this test no longer proves the precision fix", old)
	}
	if old := strconv.FormatFloat(float64(withheld), 'f', -1, 32); old == "33554432.55" {
		t.Errorf("float32 held 33554432.55 exactly (%q); this test no longer proves the precision fix", old)
	}
}
