// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"crypto/rsa"
	"io"
	"os"
	"strings"
	"time"

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

// WithOAuth2JWTKey sets the RSA private key for JWT-bearer token exchange
// (private_key_jwt grant type). Use this when the REST API requires
// Signed JWT authentication instead of client credentials or refresh token.
func WithOAuth2JWTKey(key *rsa.PrivateKey) Option {
	return func(c *config) error {
		c.oauth2.JWTKey = key
		return nil
	}
}

// WithOAuth2JWTKeyFile loads an RSA private key from a PEM file for
// JWT-bearer token exchange.
func WithOAuth2JWTKeyFile(path string) Option {
	return func(c *config) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return &ConfigError{Field: "JWTKeyFile", Message: "read key file: " + err.Error()}
		}
		c.oauth2.JWTKeyPEM = data
		return nil
	}
}

// WithOAuth2JWTKeyPEM sets raw PEM-encoded RSA private key bytes for
// JWT-bearer token exchange.
func WithOAuth2JWTKeyPEM(pemData []byte) Option {
	return func(c *config) error {
		c.oauth2.JWTKeyPEM = pemData
		return nil
	}
}

// WithOAuth2JWTKeyReader loads an RSA private key from an io.Reader for
// JWT-bearer token exchange. The reader must return PEM-encoded data.
func WithOAuth2JWTKeyReader(r io.Reader) Option {
	return func(c *config) error {
		data, err := io.ReadAll(r)
		if err != nil {
			return &ConfigError{Field: "JWTKeyReader", Message: "read key data: " + err.Error()}
		}
		c.oauth2.JWTKeyPEM = data
		return nil
	}
}

// WithOAuth2JWTExpiry sets the JWT assertion expiry duration. Defaults to 60s.
func WithOAuth2JWTExpiry(duration string) Option {
	return func(c *config) error {
		d, err := time.ParseDuration(duration)
		if err != nil {
			return &ConfigError{Field: "JWTExpiry", Message: "parse duration: " + err.Error()}
		}
		c.oauth2.JWTExpiry = d
		return nil
	}
}
