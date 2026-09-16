// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"net/http"

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
