// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"strings"

	"github.com/shing1211/ibkrapi4go/internal"
)

// DefaultRESTGatewayURL is the hosted IB REST API base URL.
const DefaultRESTGatewayURL = "https://api.ibkr.com"

// OAuth2Config configures OAuth2 access-token acquisition for the REST surface.
// It is passed to WithOAuth2, or the individual WithOAuth2* options can be used.
type OAuth2Config = internal.OAuthConfig

// WithRESTGateway sets the IB REST API base URL for the oauth2Bearer surface.
func WithRESTGateway(url string) Option {
	return func(c *config) error {
		if strings.TrimSpace(url) == "" {
			return &ConfigError{Field: "RESTGateway", Message: "must not be empty"}
		}
		c.restGatewayURL = url
		return nil
	}
}

// WithOAuth2 sets the OAuth2 configuration used to obtain bearer tokens for the
// REST surface.
func WithOAuth2(cfg OAuth2Config) Option {
	return func(c *config) error {
		c.oauth2 = cfg
		return nil
	}
}

// WithOAuth2ClientCredentials sets the OAuth2 client id and secret
// (client_credentials grant).
func WithOAuth2ClientCredentials(clientID, clientSecret string) Option {
	return func(c *config) error {
		c.oauth2.ClientID = clientID
		c.oauth2.ClientSecret = clientSecret
		return nil
	}
}

// WithOAuth2RefreshToken selects the refresh_token grant with the given token.
func WithOAuth2RefreshToken(refreshToken string) Option {
	return func(c *config) error {
		c.oauth2.RefreshToken = refreshToken
		return nil
	}
}

// WithOAuth2Scope sets the requested OAuth2 scope.
func WithOAuth2Scope(scope string) Option {
	return func(c *config) error {
		c.oauth2.Scope = scope
		return nil
	}
}

// WithOAuth2TokenURL overrides the OAuth2 token endpoint.
func WithOAuth2TokenURL(url string) Option {
	return func(c *config) error {
		c.oauth2.TokenURL = url
		return nil
	}
}
