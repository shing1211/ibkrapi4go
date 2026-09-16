// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/shing1211/ibkrapi4go/client"
	"github.com/shing1211/ibkrapi4go/internal"
)

// DefaultGatewayURL is the base URL of the locally-run Client Portal Gateway.
const DefaultGatewayURL = "https://localhost:5000"

// DefaultRequestTimeout is the default per-request timeout.
const DefaultRequestTimeout = 15 * time.Second

// DefaultTickleInterval is the default session heartbeat interval.
const DefaultTickleInterval = 60 * time.Second

// closeLogoutTimeout bounds the best-effort logout performed by Close.
const closeLogoutTimeout = 3 * time.Second

// DefaultServerURL is a backwards-compatible alias for DefaultGatewayURL.
//
// Deprecated: use DefaultGatewayURL.
const DefaultServerURL = DefaultGatewayURL

// UserAgent returns the default outbound User-Agent.
func defaultUserAgent() string { return "ibkrapi4go/" + strings.TrimPrefix(Version, "v") }

// config is the resolved client configuration.
type config struct {
	gatewayURL         string
	httpClient         *http.Client
	requestTimeout     time.Duration
	tickleInterval     time.Duration
	userAgent          string
	logger             *slog.Logger
	insecureSkipVerify bool
}

// Option customizes a Client during construction. Options are applied in order;
// later options override earlier ones and the environment.
type Option func(*config) error

// WithGatewayURL sets the Client Portal Gateway base URL.
func WithGatewayURL(u string) Option {
	return func(c *config) error {
		if strings.TrimSpace(u) == "" {
			return &ConfigError{Field: "GatewayURL", Message: "must not be empty"}
		}
		parsed, err := url.Parse(u)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return &ConfigError{Field: "GatewayURL", Message: "must be an absolute URL"}
		}
		c.gatewayURL = u
		return nil
	}
}

// WithHTTPClient supplies a base *http.Client whose Transport and cookie jar are
// used by the SDK. The SDK wraps the provided Transport with its own middleware
// chain (request id, User-Agent, auth, error decoding).
func WithHTTPClient(hc *http.Client) Option {
	return func(c *config) error {
		c.httpClient = hc
		return nil
	}
}

// WithRequestTimeout sets the per-request timeout.
func WithRequestTimeout(d time.Duration) Option {
	return func(c *config) error {
		if d < 0 {
			return &ConfigError{Field: "RequestTimeout", Message: "must not be negative"}
		}
		c.requestTimeout = d
		return nil
	}
}

// WithTickleInterval sets the session heartbeat interval.
func WithTickleInterval(d time.Duration) Option {
	return func(c *config) error {
		if d < 0 {
			return &ConfigError{Field: "TickleInterval", Message: "must not be negative"}
		}
		c.tickleInterval = d
		return nil
	}
}

// WithUserAgent overrides the outbound User-Agent header.
func WithUserAgent(ua string) Option {
	return func(c *config) error {
		c.userAgent = ua
		return nil
	}
}

// WithLogger sets the structured logger. When unset the SDK discards output.
func WithLogger(l *slog.Logger) Option {
	return func(c *config) error {
		c.logger = l
		return nil
	}
}

// WithInsecureSkipVerify disables TLS certificate verification. It is intended
// only for the local gateway, which uses a self-signed certificate. The SDK logs
// a warning when it is enabled against a non-loopback host.
func WithInsecureSkipVerify(skip bool) Option {
	return func(c *config) error {
		c.insecureSkipVerify = skip
		return nil
	}
}

// Client is the composition root for the SDK. It owns the HTTP transport, the
// session state machine, and the domain managers. It is safe for concurrent use.
type Client struct {
	cfg        config
	httpClient *http.Client
	session    *internal.Session
	generated  *client.ClientWithResponses

	closed atomic.Bool

	sessionManager    *SessionManager
	accountManager    *AccountManager
	portfolioManager  *PortfolioManager
	tradeManager      *TradeManager
	marketDataManager *MarketDataManager
}

// NewClient builds a Client from functional options, falling back to environment
// variables and then compiled-in defaults. It performs no I/O and does not
// authenticate; call Session().Initialize to start the session.
func NewClient(opts ...Option) (*Client, error) {
	cfg := config{}
	for _, o := range opts {
		if err := o(&cfg); err != nil {
			return nil, err
		}
	}
	applyEnv(&cfg)

	if cfg.gatewayURL == "" {
		cfg.gatewayURL = DefaultGatewayURL
	}
	if cfg.requestTimeout == 0 {
		cfg.requestTimeout = DefaultRequestTimeout
	}
	if cfg.tickleInterval == 0 {
		cfg.tickleInterval = DefaultTickleInterval
	}
	if cfg.userAgent == "" {
		cfg.userAgent = defaultUserAgent()
	}

	base, jar := baseTransport(cfg)
	if cfg.insecureSkipVerify && isNonLoopback(cfg.gatewayURL) && cfg.logger != nil {
		cfg.logger.Warn("insecure TLS skip-verify enabled for non-loopback host",
			"gateway", cfg.gatewayURL)
	}

	var session *internal.Session
	transport := internal.NewTransport(base,
		internal.WithRequestID(newRequestID),
		internal.WithUserAgent(cfg.userAgent),
		internal.WithToken("Authorization", func() (string, bool) {
			if session == nil {
				return "", false
			}
			return session.Token()
		}),
	)
	httpClient := &http.Client{Transport: transport, Jar: jar}

	session = internal.NewSession(internal.SessionConfig{
		HTTPClient:     httpClient,
		ServerURL:      cfg.gatewayURL,
		TickleInterval: cfg.tickleInterval,
		RequestTimeout: cfg.requestTimeout,
		Logger:         cfg.logger,
	})

	generated, err := client.NewClientWithResponses(
		cfg.gatewayURL,
		client.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, err
	}

	c := &Client{
		cfg:        cfg,
		httpClient: httpClient,
		session:    session,
		generated:  generated,
	}
	c.sessionManager = &SessionManager{client: c}
	c.accountManager = &AccountManager{client: c}
	c.portfolioManager = &PortfolioManager{client: c}
	c.tradeManager = &TradeManager{client: c}
	c.marketDataManager = &MarketDataManager{client: c}
	return c, nil
}

// Session returns the session manager.
func (c *Client) Session() *SessionManager { return c.sessionManager }

// Account returns the account manager.
func (c *Client) Account() *AccountManager { return c.accountManager }

// Portfolio returns the portfolio manager.
func (c *Client) Portfolio() *PortfolioManager { return c.portfolioManager }

// Trade returns the trade manager, which also exposes contract lookups.
func (c *Client) Trade() *TradeManager { return c.tradeManager }

// MarketData returns the market-data manager.
func (c *Client) MarketData() *MarketDataManager { return c.marketDataManager }

// GatewayURL returns the configured gateway base URL.
func (c *Client) GatewayURL() string { return c.cfg.gatewayURL }

// HTTPClient returns the underlying *http.Client, including the SDK middleware
// chain. Callers should not replace its Transport.
func (c *Client) HTTPClient() *http.Client { return c.httpClient }

// Close stops the tickle goroutine and performs a best-effort logout. It is
// idempotent: after the first call, subsequent calls return nil. After Close,
// manager methods return ErrClosed.
func (c *Client) Close() error {
	if c.closed.Swap(true) {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), closeLogoutTimeout)
	defer cancel()
	return c.session.Close(ctx)
}

// checkOpen returns ErrClosed if the client has been closed.
func (c *Client) checkOpen() error {
	if c.closed.Load() {
		return internal.ErrClosed
	}
	return nil
}

// errorFrom maps a non-2xx response to *Error, tagging it with op.
func (c *Client) errorFrom(resp *http.Response, op string) *Error {
	e := internal.ResponseError(resp)
	if e == nil {
		return nil
	}
	e.Op = op
	return e
}

// wrapOp wraps a transport-level error as *Error so callers can use errors.As.
func wrapOp(op string, err error) *Error {
	return &Error{Op: op, Message: err.Error(), Err: err}
}

// baseTransport selects the base RoundTripper and cookie jar. When the caller
// supplies a client, its Transport and Jar are honored.
func baseTransport(cfg config) (http.RoundTripper, http.CookieJar) {
	var base http.RoundTripper = http.DefaultTransport
	var jar http.CookieJar
	if cfg.httpClient != nil {
		if cfg.httpClient.Transport != nil {
			base = cfg.httpClient.Transport
		}
		jar = cfg.httpClient.Jar
	} else if cfg.insecureSkipVerify {
		base = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}} //nolint:gosec // opt-in, localhost only
	}
	if jar == nil {
		if j, err := cookiejar.New(nil); err == nil {
			jar = j
		}
	}
	return base, jar
}

// newRequestID generates a random request correlation id.
func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

// isNonLoopback reports whether the URL host is not localhost/loopback.
func isNonLoopback(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return true
	}
	host := u.Hostname()
	if host == "localhost" {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return !ip.IsLoopback()
	}
	return true
}

// applyEnv fills unset configuration from the environment. Options always win
// because applyEnv only sets fields that are still zero-valued.
func applyEnv(cfg *config) {
	if cfg.gatewayURL == "" {
		if v := os.Getenv("IBKR_GATEWAY_URL"); v != "" {
			cfg.gatewayURL = v
		}
	}
	if cfg.requestTimeout == 0 {
		if v := os.Getenv("IBKR_TIMEOUT"); v != "" {
			if d, err := time.ParseDuration(v); err == nil {
				cfg.requestTimeout = d
			}
		}
	}
	if cfg.tickleInterval == 0 {
		if v := os.Getenv("IBKR_TICKLE_INTERVAL"); v != "" {
			if d, err := time.ParseDuration(v); err == nil {
				cfg.tickleInterval = d
			}
		}
	}
	if cfg.userAgent == "" {
		cfg.userAgent = os.Getenv("IBKR_USER_AGENT")
	}
	if !cfg.insecureSkipVerify {
		if v := os.Getenv("IBKR_INSECURE_SKIP_VERIFY"); v == "true" || v == "1" {
			cfg.insecureSkipVerify = true
		}
	}
	if cfg.logger == nil {
		cfg.logger = loggerFromEnv()
	}
}

// loggerFromEnv builds a logger when IBKR_LOG_LEVEL is set, else returns nil
// (discard output).
func loggerFromEnv() *slog.Logger {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("IBKR_LOG_LEVEL")))
	if v == "" {
		return nil
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(v)); err != nil {
		return nil
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}
