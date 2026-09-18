// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/shing1211/ibkrapi4go/client"
	"github.com/shing1211/ibkrapi4go/internal"
)

// RESTSurface is the hosted IB REST API surface (`/gw/api/v1`, `/gw/api/v2`),
// authenticated with an OAuth2 bearer token. Obtain it from Client.REST.
type RESTSurface struct {
	owner     *Client
	generated *client.ClientWithResponses
}

// REST returns the IB REST surface, dialing/authorizing it lazily. It returns an
// error if no OAuth2 credentials were configured.
func (c *Client) REST() (*RESTSurface, error) {
	c.restMu.Lock()
	defer c.restMu.Unlock()
	if c.rest != nil {
		return c.rest, nil
	}
	if c.cfg.tokenSource == nil {
		return nil, &Error{Op: "REST", Message: "oauth2 is not configured; use WithOAuth2* options", Err: ErrNotAuthenticated}
	}

	base, jar := baseTransport(c.cfg)
	ts := c.cfg.tokenSource
	var limiter *internal.Limiter
	if c.cfg.rateLimit > 0 || c.cfg.globalRateLimit > 0 {
		limiter = internal.NewLimiter(c.cfg.rateLimit, c.cfg.rateBurst, c.cfg.globalRateLimit)
		limiter.Logger = c.cfg.logger
		limiter.SetMetrics(c.cfg.metrics)
	}
	transport := internal.NewClientTransport(base, internal.TransportConfig{
		RequestID:  newRequestID,
		UserAgent:  c.cfg.userAgent,
		AuthHeader: "Authorization",
		Token: func() (string, bool) {
			tok, err := ts.Token(context.Background())
			if err != nil || tok == "" {
				return "", false
			}
			return "Bearer " + tok, true
		},
		Logger:    c.cfg.logger,
		Telemetry: c.cfg.telemetry,
		Metrics:   c.cfg.metrics,
		Breaker:   c.cfg.breaker,
		Retry:     c.cfg.retry,
		Limiter:   limiter,
		Timeout:   c.cfg.requestTimeout,
	})
	httpClient := &http.Client{Transport: transport, Jar: jar}

	gen, err := client.NewClientWithResponses(c.cfg.restGatewayURL, client.WithHTTPClient(httpClient))
	if err != nil {
		return nil, &Error{Op: "REST", Message: err.Error(), Err: err}
	}
	c.rest = &RESTSurface{owner: c, generated: gen}
	return c.rest, nil
}

// Accounts returns the REST accounts manager.
func (s *RESTSurface) Accounts() *RESTAccounts { return &RESTAccounts{surface: s} }

// Token returns a currently-valid OAuth2 access token, refreshing as needed.
func (s *RESTSurface) Token(ctx context.Context) (string, error) {
	return s.owner.cfg.tokenSource.Token(ctx)
}

// GatewayURL returns the configured REST base URL.
func (s *RESTSurface) GatewayURL() string { return s.owner.cfg.restGatewayURL }

// RESTAccounts exposes REST account operations.
type RESTAccounts struct {
	surface *RESTSurface
}

// AccountDetails is detailed account information from the REST surface.
type AccountDetails struct {
	// ID is the account identifier.
	ID AccountID
	// Alias is the account alias.
	Alias string
	// Title is the account title.
	Title string
	// BaseCurrency is the account base currency.
	BaseCurrency string
	// Household is the household the account belongs to.
	Household string
	// ApplicantType is the applicant type.
	ApplicantType string
	// OrganizationType is the organization type, if any.
	OrganizationType string
}

// Details returns detailed information for the given account.
func (m *RESTAccounts) Details(ctx context.Context, id AccountID) (*AccountDetails, error) {
	const op = "Accounts.Details"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.GetAccountsDetails(ctx, string(id))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if e := m.surface.owner.errorFrom(resp, op); e != nil {
		resp.Body.Close()
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	var raw accountDetailsRaw
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	return raw.toPublic(id), nil
}

type accountDetailsRaw struct {
	AccountID     string `json:"accountId"`
	AccountAlias  string `json:"accountAlias"`
	AccountTitle  string `json:"accountTitle"`
	BaseCurrency  string `json:"baseCurrency"`
	Household     string `json:"household"`
	ApplicantType string `json:"applicantType"`
	OrgType       string `json:"orgType"`
}

func (r accountDetailsRaw) toPublic(id AccountID) *AccountDetails {
	accountID := id
	if r.AccountID != "" {
		accountID = AccountID(r.AccountID)
	}
	return &AccountDetails{
		ID:               accountID,
		Alias:            r.AccountAlias,
		Title:            r.AccountTitle,
		BaseCurrency:     r.BaseCurrency,
		Household:        r.Household,
		ApplicantType:    r.ApplicantType,
		OrganizationType: r.OrgType,
	}
}

// Statements returns the REST statements manager.
func (s *RESTSurface) Statements() *RESTStatements { return &RESTStatements{surface: s} }

// Requests returns the REST requests manager.
func (s *RESTSurface) Requests() *RESTRequests { return &RESTRequests{surface: s} }

// RESTRequests exposes request-status operations on the REST surface.
type RESTRequests struct {
	surface *RESTSurface
}

// RESTRequestInfo holds metadata about a submitted request.
type RESTRequestInfo struct {
	ID         int64
	ExecutedAt *string
}

// GetStatus retrieves the current status of a submitted request.
func (m *RESTRequests) GetStatus(ctx context.Context, requestID int64) (*RESTRequestInfo, error) {
	const op = "Requests.GetStatus"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.GetRequestsStatusWithResponse(ctx, requestID, nil)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := wrapOp(op, &Error{Code: "http_error", Message: fmt.Sprintf("GetRequestsStatus: %d", resp.HTTPResponse.StatusCode), HTTPStatus: resp.HTTPResponse.StatusCode})
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if j := resp.GetJSON200(); j != nil {
		return &RESTRequestInfo{ID: requestID}, nil
	}
	return &RESTRequestInfo{ID: requestID}, nil
}

// TaxDocuments returns the REST tax-documents manager.
func (s *RESTSurface) TaxDocuments() *RESTTaxDocuments { return &RESTTaxDocuments{surface: s} }

// RESTTaxDocuments exposes tax-document operations on the REST surface.
type RESTTaxDocuments struct {
	surface *RESTSurface
}

// TaxDocumentRequest is a request to fetch tax documents.
type TaxDocumentRequest struct {
	// AccountID is the account for which to fetch tax documents.
	AccountID AccountID
	// Year is the tax year.
	Year string
	// Type is the tax form type (e.g. "ALL", "1099", "1099R", "1042S", "8949").
	Type string
	// Format is the output MIME type. Defaults to "PDF".
	Format string
}

// TaxDocumentResponse is the generated tax document.
type TaxDocumentResponse struct {
	// ContentType is the MIME type of the returned document.
	ContentType string
	// Data is the base64-encoded document payload.
	Data []byte
	// Gzip indicates whether Data is gzip-compressed.
	Gzip bool
}

// AvailableTaxDocumentTypes holds the available tax form types for an account/year.
type AvailableTaxDocumentTypes struct {
	// Forms is the list of available tax form identifiers (e.g. "1099", "1099R").
	Forms []string
}

// ListAvailable returns the tax-form types available for the given account.
func (m *RESTTaxDocuments) ListAvailable(ctx context.Context, id AccountID) (*AvailableTaxDocumentTypes, error) {
	const op = "TaxDocuments.ListAvailable"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	params := client.ListTaxDocumentsAvailableParams{
		AccountId: string(id),
	}
	resp, err := m.surface.generated.ListTaxDocumentsAvailableWithResponse(ctx, &params)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := wrapOp(op, &Error{Code: "http_error", Message: fmt.Sprintf("ListTaxDocumentsAvailable: %d", resp.HTTPResponse.StatusCode), HTTPStatus: resp.HTTPResponse.StatusCode})
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if j := resp.GetJSON200(); j != nil && j.Data != nil && j.Data.Value != nil && j.Data.Value.Forms != nil {
		forms := make([]string, len(*j.Data.Value.Forms))
		for i, ft := range *j.Data.Value.Forms {
			if ft.TaxFormName != nil {
				forms[i] = *ft.TaxFormName
			}
		}
		return &AvailableTaxDocumentTypes{Forms: forms}, nil
	}
	return &AvailableTaxDocumentTypes{}, nil
}

// Generate produces tax documents for the given request.
// The returned payload is base64-encoded; decode it to obtain the PDF/HTML bytes.
func (m *RESTTaxDocuments) Generate(ctx context.Context, req TaxDocumentRequest) (*TaxDocumentResponse, error) {
	const op = "TaxDocuments.Generate"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	format := req.Format
	if format == "" {
		format = "PDF"
	}
	body := client.TaxFormRequest{
		AccountId: string(req.AccountID),
		Year:      req.Year,
		Type:      req.Type,
		Format:    format,
	}
	resp, err := m.surface.generated.CreateTaxDocumentsWithResponse(ctx, nil, body)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := wrapOp(op, &Error{Code: "http_error", Message: fmt.Sprintf("CreateTaxDocuments: %d", resp.HTTPResponse.StatusCode), HTTPStatus: resp.HTTPResponse.StatusCode})
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	var raw struct {
		Data struct {
			Value    *string `json:"value,omitempty"`
			MimeType *string `json:"mimeType,omitempty"`
			Gzip     *bool   `json:"gzip,omitempty"`
		} `json:"data,omitempty"`
	}
	if err := decodeJSONBytes(resp.Body, op, &raw); err != nil {
		return nil, err
	}
	if raw.Data.Value == nil {
		return &TaxDocumentResponse{}, nil
	}
	ct := "application/octet-stream"
	if raw.Data.MimeType != nil {
		ct = *raw.Data.MimeType
	}
	gzip := false
	if raw.Data.Gzip != nil {
		gzip = *raw.Data.Gzip
	}
	return &TaxDocumentResponse{
		ContentType: ct,
		Data:        []byte(*raw.Data.Value),
		Gzip:        gzip,
	}, nil
}

// TradeConfirmations returns the REST trade-confirmations manager.
func (s *RESTSurface) TradeConfirmations() *RESTTradeConfirmations {
	return &RESTTradeConfirmations{surface: s}
}

// RESTTradeConfirmations exposes trade-confirmation operations on the REST surface.
type RESTTradeConfirmations struct {
	surface *RESTSurface
}

// TradeConfirmationRequest is a request to fetch trade confirmations.
type TradeConfirmationRequest struct {
	// AccountID is the account for which to fetch confirmations.
	AccountID AccountID
	// StartDate is the start of the reporting period (YYYY-MM-DD).
	StartDate string
	// EndDate is the end of the reporting period (YYYY-MM-DD).
	EndDate string
	// Format is the output MIME type. Defaults to application/pdf.
	Format string
	// Gzip compresses the response body.
	Gzip bool
}

// TradeConfirmationResponse is the generated trade-confirmation document.
type TradeConfirmationResponse struct {
	// ContentType is the MIME type of the returned document.
	ContentType string
	// Data is the base64-encoded document payload.
	Data []byte
	// Gzip indicates whether Data is gzip-compressed.
	Gzip bool
}

// AvailableTradeConfirmationDates holds the dates for which confirmations are available.
type AvailableTradeConfirmationDates struct {
	// Dates is the list of available confirmation date identifiers.
	Dates []string
}

// ListAvailable returns the trade-confirmation dates available for the given account.
func (m *RESTTradeConfirmations) ListAvailable(ctx context.Context, id AccountID) (*AvailableTradeConfirmationDates, error) {
	const op = "TradeConfirmations.ListAvailable"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	auth, _ := m.surface.Token(ctx)
	params := client.ListTradeConfirmationsAvailableParams{
		AccountId:     string(id),
		Authorization: auth,
	}
	resp, err := m.surface.generated.ListTradeConfirmationsAvailableWithResponse(ctx, &params)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := wrapOp(op, &Error{Code: "http_error", Message: fmt.Sprintf("ListTradeConfirmationsAvailable: %d", resp.HTTPResponse.StatusCode), HTTPStatus: resp.HTTPResponse.StatusCode})
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if j := resp.GetJSON200(); j != nil && j.Data != nil && j.Data.Value != nil {
		return &AvailableTradeConfirmationDates{Dates: *j.Data.Value}, nil
	}
	return &AvailableTradeConfirmationDates{}, nil
}

// Generate produces trade confirmations for the given request.
// The returned payload is base64-encoded; decode it to obtain the PDF/HTML bytes.
func (m *RESTTradeConfirmations) Generate(ctx context.Context, req TradeConfirmationRequest) (*TradeConfirmationResponse, error) {
	const op = "TradeConfirmations.Generate"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	type_ := req.Format
	if type_ == "" {
		type_ = "application/pdf"
	}
	body := client.TradeConfirmationRequest{
		AccountId: string(req.AccountID),
		EndDate:   req.EndDate,
		StartDate: req.StartDate,
		MimeType:  &type_,
	}
	resp, err := m.surface.generated.CreateTradeConfirmations(ctx, nil, body)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if e := m.surface.owner.errorFrom(resp, op); e != nil {
		resp.Body.Close()
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	var raw struct {
		Data struct {
			Value    *string `json:"value,omitempty"`
			MimeType *string `json:"mimeType,omitempty"`
			Gzip     *bool   `json:"gzip,omitempty"`
		} `json:"data,omitempty"`
	}
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	if raw.Data.Value == nil {
		return &TradeConfirmationResponse{}, nil
	}
	ct := "application/octet-stream"
	if raw.Data.MimeType != nil {
		ct = *raw.Data.MimeType
	}
	gzip := false
	if raw.Data.Gzip != nil {
		gzip = *raw.Data.Gzip
	}
	return &TradeConfirmationResponse{
		ContentType: ct,
		Data:        []byte(*raw.Data.Value),
		Gzip:        gzip,
	}, nil
}

// RESTStatements exposes statement operations on the REST surface.
type RESTStatements struct {
	surface *RESTSurface
}

// StatementRequest is a request to generate one or more statements.
type StatementRequest struct {
	// AccountID is the account for which to generate statements.
	AccountID AccountID
	// StartDate is the start of the reporting period (YYYY-MM-DD).
	StartDate string
	// EndDate is the end of the reporting period (YYYY-MM-DD).
	EndDate string
	// Format is the output MIME type. Defaults to application/pdf.
	Format string
	// Language is an ISO two-character language code. Defaults to "en".
	Language string
	// AccountIDs optionally specifies multiple accounts.
	AccountIDs []AccountID
	// Gzip compresses the response body.
	Gzip bool
}

// StatementResponse is the generated statement data, including the encoded document.
type StatementResponse struct {
	// ContentType is the MIME type of the returned document (e.g. application/pdf).
	ContentType string
	// Data is the base64-encoded document payload.
	Data []byte
	// Gzip indicates whether Data is gzip-compressed.
	Gzip bool
}

// AvailableStatementDates holds the dates for which statements are available.
type AvailableStatementDates struct {
	// Annual is the list of annual statement years.
	Annual []string
	// Monthly is the list of monthly statement identifiers (e.g. "2024-01").
	Monthly []string
	// DailyStart is the start date for daily statements.
	DailyStart string
	// DailyEnd is the end date for daily statements.
	DailyEnd string
}

// Generate produces a statement for the given request.
// The returned payload is base64-encoded; decode it to obtain the PDF/HTML/CSV bytes.
func (m *RESTStatements) Generate(ctx context.Context, req StatementRequest) (*StatementResponse, error) {
	const op = "Statements.Generate"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	type_ := req.Format
	if type_ == "" {
		type_ = "application/pdf"
	}
	lang := req.Language
	if lang == "" {
		lang = "en"
	}
	body := client.StmtRequest{
		AccountId: string(req.AccountID),
		EndDate:   req.EndDate,
		StartDate: req.StartDate,
		Language:  &lang,
		MimeType:  &type_,
		Gzip:      &req.Gzip,
	}
	if len(req.AccountIDs) > 0 {
		ids := make([]string, len(req.AccountIDs))
		for i, id := range req.AccountIDs {
			ids[i] = string(id)
		}
		body.AccountIds = &ids
	}
	resp, err := m.surface.generated.CreateStatements(ctx, nil, body)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if e := m.surface.owner.errorFrom(resp, op); e != nil {
		resp.Body.Close()
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	var raw struct {
		Data struct {
			Value    *string `json:"value,omitempty"`
			MimeType *string `json:"mimeType,omitempty"`
			Encoding *string `json:"encoding,omitempty"`
			Gzip     *bool   `json:"gzip,omitempty"`
		} `json:"data,omitempty"`
	}
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	if raw.Data.Value == nil {
		return &StatementResponse{}, nil
	}
	encoded := *raw.Data.Value
	ct := "application/octet-stream"
	if raw.Data.MimeType != nil {
		ct = *raw.Data.MimeType
	}
	gzip := false
	if raw.Data.Gzip != nil {
		gzip = *raw.Data.Gzip
	}
	return &StatementResponse{
		ContentType: ct,
		Data:        []byte(encoded),
		Gzip:        gzip,
	}, nil
}

// ListAvailable returns the statement dates available for the given account.
func (m *RESTStatements) ListAvailable(ctx context.Context, id AccountID) (*AvailableStatementDates, error) {
	const op = "Statements.ListAvailable"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	params := client.ListStatementsAvailableParams{
		AccountId: string(id),
	}
	resp, err := m.surface.generated.ListStatementsAvailable(ctx, &params)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if e := m.surface.owner.errorFrom(resp, op); e != nil {
		resp.Body.Close()
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	var raw struct {
		Data *struct {
			Value *struct {
				Annual  *[]string `json:"annual,omitempty"`
				Monthly *[]string `json:"monthly,omitempty"`
				Daily   *struct {
					StartDate *string `json:"startDate,omitempty"`
					EndDate   *string `json:"endDate,omitempty"`
				} `json:"daily,omitempty"`
			} `json:"value,omitempty"`
		} `json:"data,omitempty"`
	}
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := &AvailableStatementDates{}
	if raw.Data != nil && raw.Data.Value != nil {
		if v := raw.Data.Value.Annual; v != nil {
			out.Annual = *v
		}
		if v := raw.Data.Value.Monthly; v != nil {
			out.Monthly = *v
		}
		if d := raw.Data.Value.Daily; d != nil {
			if d.StartDate != nil {
				out.DailyStart = *d.StartDate
			}
			if d.EndDate != nil {
				out.DailyEnd = *d.EndDate
			}
		}
	}
	return out, nil
}

// TaxVouchers returns the REST tax vouchers manager.
func (s *RESTSurface) TaxVouchers() *RESTTaxVouchers { return &RESTTaxVouchers{surface: s} }

// RESTTaxVouchers is the sub-manager for tax-voucher operations on the hosted
// IB REST API. Obtain it from RESTSurface.TaxVouchers.
type RESTTaxVouchers struct {
	surface *RESTSurface
}

// CreateRequests submits a tax-voucher request by posting the given CSV content.
// It returns the request ID on success, an empty string if the response body is
// empty, or an error if the request fails.
func (m *RESTTaxVouchers) CreateRequests(ctx context.Context, csvContent string) (string, error) {
	const op = "TaxVouchers.CreateRequests"
	if err := m.surface.owner.checkOpen(); err != nil {
		return "", err
	}
	resp, err := m.surface.generated.CreateTaxVoucherRequestsWithTextBodyWithResponse(ctx, nil, client.CreateTaxVoucherRequestsTextRequestBody(csvContent))
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
	if resp.JSON200 == nil || len(*resp.JSON200) == 0 {
		return "", nil
	}
	return strPtrVal((*resp.JSON200)[0].RequestId), nil
}

// ActiveCountries lists the country codes that have active tax-voucher
// agreements. Returns nil on a nil response body.
func (m *RESTTaxVouchers) ActiveCountries(ctx context.Context) ([]string, error) {
	const op = "TaxVouchers.ActiveCountries"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.GetActiveCountryListWithResponse(ctx, nil)
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
	countries := make([]string, len(*resp.JSON200))
	for i, c := range *resp.JSON200 {
		countries[i] = strPtrVal(c.Country)
	}
	return countries, nil
}

// Dividends retrieves dividend records for the given account, year, and
// country code. Returns nil on a nil response body.
func (m *RESTTaxVouchers) Dividends(ctx context.Context, accountID AccountID, year, countryCode string) ([]TaxVoucherDividend, error) {
	const op = "TaxVouchers.Dividends"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.FetchDividends1WithResponse(ctx, &client.FetchDividends1Params{
		CustAcctId:  string(accountID),
		Year:        year,
		CountryCode: countryCode,
	})
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
	out := make([]TaxVoucherDividend, 0, len(*resp.JSON200))
	for _, d := range *resp.JSON200 {
		tvd := TaxVoucherDividend{
			CorpActionID:   strPtrVal(d.CorpactionId),
			CountryCode:    strPtrVal(d.Country),
			AccountID:      accountID,
			Amount:         "",
			Fee:            "",
			Quantity:       "",
			RequestID:      "",
			Year:           0,
			WithheldAmount: "",
		}
		if d.Voucher != nil {
			tvd.Amount = float32ToStr(d.Voucher.DivAmount)
			tvd.Fee = float32ToStr(d.Voucher.Fee)
			tvd.Quantity = float32ToStr(d.Voucher.Quantity)
			tvd.RequestID = strPtrVal(d.Voucher.RequestId)
			tvd.Year = int64PtrVal(d.Voucher.Year)
			tvd.WithheldAmount = float32ToStr(d.Voucher.WithHeldAmount)
		}
		out = append(out, tvd)
	}
	return out, nil
}

// AvailableYears lists the tax years for which voucher data is available.
// Returns nil on a nil response body.
func (m *RESTTaxVouchers) AvailableYears(ctx context.Context) ([]string, error) {
	const op = "TaxVouchers.AvailableYears"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.GetYearsWithResponse(ctx, nil)
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
	years := make([]string, len(*resp.JSON200))
	for i, y := range *resp.JSON200 {
		years[i] = strconv.FormatInt(y, 10)
	}
	return years, nil
}

// Download fetches the tax-voucher document for the given request ID and
// returns its raw bytes. The caller is responsible for interpreting the
// content type.
func (m *RESTTaxVouchers) Download(ctx context.Context, requestID string) ([]byte, error) {
	const op = "TaxVouchers.Download"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.DownloadFileWithResponse(ctx, requestID, nil)
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
	return resp.Body, nil
}

// RequestState queries the current processing state of a tax-voucher request.
// Returns an error if the response body is nil or the request fails.
func (m *RESTTaxVouchers) RequestState(ctx context.Context, requestID string) (*TaxVoucherState, error) {
	const op = "TaxVouchers.RequestState"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.GetCurrentState1WithResponse(ctx, requestID, nil)
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
		e := &Error{Op: op, Message: "unexpected nil body"}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	return &TaxVoucherState{
		RequestID:    strPtrVal(resp.JSON200.RequestId),
		RequestState: strPtrVal(resp.JSON200.RequestState),
	}, nil
}

// TaxVoucherDividend represents a single dividend detail for tax-voucher
// reporting. All monetary fields are strings to preserve precision.
//
// Example:
//
//	dividends, err := rest.TaxVouchers().Dividends(ctx, accountID, "2025", "us")
//	for _, d := range dividends {
//	    fmt.Println(d.AccountID, d.Amount, d.CountryCode)
//	}
type TaxVoucherDividend struct {
	CorpActionID   string
	CountryCode    string
	AccountID      AccountID
	Amount         string
	Fee            string
	Quantity       string
	RequestID      string
	Year           int64
	WithheldAmount string
}

// TaxVoucherState describes the processing state of a previously submitted
// tax-voucher request.
//
// Example:
//
//	state, err := rest.TaxVouchers().RequestState(ctx, requestID)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(state.RequestID, state.RequestState)
type TaxVoucherState struct {
	RequestID    string
	RequestState string
}

type requestIDRaw struct {
	RequestId *string `json:"requestId,omitempty"`
}

func (r *requestIDRaw) toPublic() string {
	return strPtrVal(r.RequestId)
}

type countriesRaw struct {
	Countries []string `json:"countries,omitempty"`
}

func (r *countriesRaw) toPublic() []string {
	if r.Countries == nil {
		return nil
	}
	return r.Countries
}

type yearsRaw struct {
	Years []string `json:"years,omitempty"`
}

func (r *yearsRaw) toPublic() []string {
	if r.Years == nil {
		return nil
	}
	return r.Years
}

type dividendsRaw struct {
	TaxVouchers []taxVoucherRaw `json:"taxVoucherRequests,omitempty"`
}

type taxVoucherRaw struct {
	CorpactionId       *string  `json:"corpactionId,omitempty"`
	CountryCode        *string  `json:"countryCode,omitempty"`
	CustAcctId         *string  `json:"custAcctId,omitempty"`
	DivAmount          *float32 `json:"divAmount,omitempty"`
	Fee                *float32 `json:"fee,omitempty"`
	MigratedCustAcctId *string  `json:"migratedCustAcctId,omitempty"`
	Quantity           *float32 `json:"quantity,omitempty"`
	RequestId          *string  `json:"requestId,omitempty"`
	RequestState       *string  `json:"requestState,omitempty"`
	WithHeldAmount     *float32 `json:"withHeldAmount,omitempty"`
	Year               *int64   `json:"year,omitempty"`
}

func (r *dividendsRaw) toPublic() []TaxVoucherDividend {
	if r.TaxVouchers == nil {
		return nil
	}
	out := make([]TaxVoucherDividend, 0, len(r.TaxVouchers))
	for _, t := range r.TaxVouchers {
		out = append(out, TaxVoucherDividend{
			CorpActionID:   strPtrVal(t.CorpactionId),
			CountryCode:    strPtrVal(t.CountryCode),
			AccountID:      AccountID(strPtrVal(t.CustAcctId)),
			Amount:         float32ToStr(t.DivAmount),
			Fee:            float32ToStr(t.Fee),
			Quantity:       float32ToStr(t.Quantity),
			RequestID:      strPtrVal(t.RequestId),
			Year:           int64PtrVal(t.Year),
			WithheldAmount: float32ToStr(t.WithHeldAmount),
		})
	}
	return out
}

func float32ToStr(p *float32) string {
	if p == nil {
		return ""
	}
	return strconv.FormatFloat(float64(*p), 'f', -1, 32)
}
