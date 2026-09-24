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
	"strconv"
	"strings"
	"sync"
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
	endpointTimeout    time.Duration
	tickleInterval     time.Duration
	userAgent          string
	logger             *slog.Logger
	insecureSkipVerify bool
	streamingLimits    StreamingLimits
	rateLimit          float64
	rateBurst          int
	globalRateLimit    float64
	retry              internal.RetryPolicy
	telemetry          internal.Telemetry
	metrics            internal.Metrics
	breaker            *internal.Breaker
	restGatewayURL     string
	oauth2             internal.OAuthConfig
	tokenSource        *internal.TokenSource
	userMiddleware     []internal.Middleware
	maxResponseBytes   int64
}

// Rate-limit defaults (see docs/RATE-LIMITING.md).
const (
	DefaultPerEndpointRPS = 10
	DefaultGlobalRPS      = 50
)

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

// WithRateLimit sets the per-endpoint request rate (requests/second) and burst.
// rps<=0 disables per-endpoint limiting; burst<=0 defaults to 2*rps.
func WithRateLimit(rps float64, burst int) Option {
	return func(c *config) error {
		if rps < 0 {
			return &ConfigError{Field: "RateLimit", Message: "must not be negative"}
		}
		c.rateLimit = rps
		c.rateBurst = burst
		return nil
	}
}

// WithMaxResponseBytes caps the size of any single HTTP response body.
// Responses exceeding the cap return an error instead of consuming unbounded
// memory. A value <= 0 disables the limit (default).
func WithMaxResponseBytes(n int64) Option {
	return func(c *config) error {
		c.maxResponseBytes = n
		return nil
	}
}

// RetryPolicy controls automatic retries for safe (idempotent) requests. Order
// and other unsafe mutations are never retried (ADR 0009).
type RetryPolicy = internal.RetryPolicy

// DefaultRetryPolicy returns the SDK's default retry policy.
func DefaultRetryPolicy() RetryPolicy { return internal.DefaultRetryPolicy() }

// Telemetry receives request lifecycle callbacks. Implementations must be safe
// for concurrent use. It is OTel-compatible but carries no OTel dependency.
type Telemetry = internal.Telemetry

// RequestInfo describes an outbound request for telemetry.
type RequestInfo = internal.RequestInfo

// ResponseInfo describes a request outcome for telemetry.
type ResponseInfo = internal.ResponseInfo

// SpanContext carries trace context across the API boundary.
type SpanContext = internal.SpanContext

// WSConnInfo describes a WebSocket connection event.
type WSConnInfo = internal.WSConnInfo

// WSSubInfo describes a subscription change.
type WSSubInfo = internal.WSSubInfo

// OrderEventInfo describes an order lifecycle event.
type OrderEventInfo = internal.OrderEventInfo

// WithTelemetry installs telemetry hooks invoked around each HTTP request.
func WithTelemetry(t Telemetry) Option {
	return func(c *config) error {
		c.telemetry = t
		return nil
	}
}

// Metrics receives metric observations. Implementations must be safe for
// concurrent use and must not block. It is shaped to bridge directly onto
// OpenTelemetry instruments but carries no OTel dependency (ADR 0013).
type Metrics = internal.Metrics

// Attr is a low-cardinality metric attribute. Keys and values must never carry
// PII, account ids, conids, order ids, or tokens.
type Attr = internal.Attr

// InMemoryMetrics is a dependency-free, thread-safe Metrics implementation
// suitable for tests and simple applications.
type InMemoryMetrics = internal.InMemoryMetrics

// MetricsSnapshot is a point-in-time copy of an InMemoryMetrics.
type MetricsSnapshot = internal.MetricsSnapshot

// NewInMemoryMetrics returns an empty in-memory metrics sink suitable for tests
// and simple applications.
func NewInMemoryMetrics() *InMemoryMetrics { return internal.NewInMemoryMetrics() }

// NopMetrics returns a metrics sink that discards every observation.
func NopMetrics() Metrics { return internal.NopMetrics() }

// Metric names emitted by the SDK. All attributes are low-cardinality.
const (
	MetricHTTPRequests        = internal.MetricHTTPRequests
	MetricHTTPErrors          = internal.MetricHTTPErrors
	MetricHTTPDuration        = internal.MetricHTTPDuration
	MetricOrdersSubmitted     = internal.MetricOrdersSubmitted
	MetricOrdersConfirmed     = internal.MetricOrdersConfirmed
	MetricOrdersModified      = internal.MetricOrdersModified
	MetricOrdersCancelled     = internal.MetricOrdersCancelled
	MetricOrdersRejected      = internal.MetricOrdersRejected
	MetricRateLimitWaits      = internal.MetricRateLimitWaits
	MetricRateLimitWaitMS     = internal.MetricRateLimitWaitMS
	MetricBreakerState        = internal.MetricBreakerState
	MetricWSConnects          = internal.MetricWSConnects
	MetricWSReconnects        = internal.MetricWSReconnects
	MetricOAuthTokenRefreshes = internal.MetricOAuthTokenRefreshes
	MetricOAuthTokenFailures  = internal.MetricOAuthTokenFailures
)

// SeriesKey builds the stable, attribute-ordered snapshot key used by
// InMemoryMetrics.
func SeriesKey(name string, attrs ...Attr) string { return internal.SeriesKey(name, attrs...) }

// WithMetrics installs a metrics sink. Every SDK subsystem (HTTP transport,
// circuit breaker, rate limiter, OAuth token source, WebSocket) reports to it.
// A nil sink disables metrics.
func WithMetrics(m Metrics) Option {
	return func(c *config) error {
		c.metrics = m
		return nil
	}
}

// WithCircuitBreaker enables a circuit breaker that opens after threshold
// consecutive transport failures, short-circuits for cooldown, then allows a
// half-open probe. threshold<=0 disables it (the default).
func WithCircuitBreaker(threshold int, cooldown time.Duration) Option {
	return func(c *config) error {
		if threshold < 0 {
			return &ConfigError{Field: "CircuitBreaker", Message: "threshold must not be negative"}
		}
		c.breaker = internal.NewBreaker(threshold, cooldown)
		return nil
	}
}

// WithCircuitBreakerBudget installs a sliding-window error budget on the
// circuit breaker. When budget failures occur within the last size outcomes
// the breaker opens, even if the consecutive-failure threshold has not been
// reached. Requires WithCircuitBreaker to be set first. budget<=0 disables
// the budget. size should be >= budget.
func WithCircuitBreakerBudget(budget, size int) Option {
	return func(c *config) error {
		if c.breaker == nil {
			return &ConfigError{Field: "CircuitBreakerBudget", Message: "requires WithCircuitBreaker"}
		}
		c.breaker.SetErrorBudget(budget, size)
		return nil
	}
}

// WithRetryPolicy overrides the retry policy. Set MaxAttempts to 1 to disable
// retries.
func WithRetryPolicy(p RetryPolicy) Option {
	return func(c *config) error {
		c.retry = p
		return nil
	}
}

// WithGlobalRateLimit sets the client-wide request rate (requests/second).
// rps<=0 disables the global bucket.
func WithGlobalRateLimit(rps float64) Option {
	return func(c *config) error {
		if rps < 0 {
			return &ConfigError{Field: "GlobalRateLimit", Message: "must not be negative"}
		}
		c.globalRateLimit = rps
		return nil
	}
}

// WithStreamingLimits overrides the streaming subscription limits. Zero fields
// keep their defaults.
func WithStreamingLimits(l StreamingLimits) Option {
	return func(c *config) error {
		c.streamingLimits = l
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

	closed  atomic.Bool
	release func() // pool-managed cleanup callback; nil for standalone clients

	wsMu sync.Mutex
	ws   *internal.WSConn

	restMu sync.Mutex
	rest   *RESTSurface

	sessionManager        *SessionManager
	accountManager        *AccountManager
	portfolioManager      *PortfolioManager
	tradeManager          *TradeManager
	marketDataManager     *MarketDataManager
	tradingAccountManager *TradingAccountManager
	alertManager          *AlertManager
	forecastManager       *ForecastManager
	scannerManager        *ScannerManager
	allocationManager     *AllocationManager
	modelManager          *ModelManager
	fyiManager            *FYIManager
	oauthManager          *OAuthManager
	watchlistManager      *WatchlistManager
	performanceManager    *PerformanceManager
}

// NewClient builds a Client from functional options, falling back to environment
// variables and then compiled-in defaults. It performs no I/O and does not
// authenticate; call Session().Initialize to start the session.
func NewClient(opts ...Option) (*Client, error) {
	cfg := config{rateLimit: -1, globalRateLimit: -1}
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
	if cfg.rateLimit < 0 {
		cfg.rateLimit = DefaultPerEndpointRPS
	}
	if cfg.globalRateLimit < 0 {
		cfg.globalRateLimit = DefaultGlobalRPS
	}
	if cfg.retry.MaxAttempts == 0 {
		cfg.retry = internal.DefaultRetryPolicy()
	}
	if cfg.restGatewayURL == "" {
		cfg.restGatewayURL = DefaultRESTGatewayURL
	}
	if cfg.oauth2.Logger == nil {
		cfg.oauth2.Logger = cfg.logger
	}
	cfg.oauth2.Metrics = cfg.metrics
	if cfg.tokenSource == nil && (cfg.oauth2.ClientID != "" || cfg.oauth2.RefreshToken != "") {
		cfg.tokenSource = internal.NewTokenSource(cfg.oauth2)
	}
	if cfg.tokenSource != nil {
		cfg.tokenSource.SetMetrics(cfg.metrics)
	}
	cfg.streamingLimits = cfg.streamingLimits.withDefaults()

	base, jar := baseTransport(cfg)
	if cfg.insecureSkipVerify && isNonLoopback(cfg.gatewayURL) {
		cfg.logger.Warn("ibkr.config insecure TLS skip-verify enabled for non-loopback host",
			"gateway", cfg.gatewayURL)
	}

	var limiter *internal.Limiter
	if cfg.rateLimit > 0 || cfg.globalRateLimit > 0 {
		limiter = internal.NewLimiter(cfg.rateLimit, cfg.rateBurst, cfg.globalRateLimit)
		limiter.Logger = cfg.logger
		limiter.SetMetrics(cfg.metrics)
	}
	if cfg.breaker != nil {
		cfg.breaker.Logger = cfg.logger
		cfg.breaker.SetMetrics(cfg.metrics)
	}

	transportTimeout := cfg.requestTimeout
	if cfg.endpointTimeout > 0 {
		transportTimeout = cfg.endpointTimeout
	}

	var session *internal.Session
	transport := internal.NewClientTransport(base, internal.TransportConfig{
		RequestID:  newRequestID,
		UserAgent:  cfg.userAgent,
		AuthHeader: "Authorization",
		Token: func() (string, bool) {
			if session == nil {
				return "", false
			}
			return session.Token()
		},
		Logger:           cfg.logger,
		Telemetry:        cfg.telemetry,
		Metrics:          cfg.metrics,
		Breaker:          cfg.breaker,
		Retry:            cfg.retry,
		Limiter:          limiter,
		Timeout:          transportTimeout,
		MaxResponseBytes: cfg.maxResponseBytes,
		UserMiddleware:   cfg.userMiddleware,
	})
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
	c.tradeManager = &TradeManager{client: c, coidRegistry: newCOIDRegistry()}
	c.marketDataManager = &MarketDataManager{client: c}
	c.tradingAccountManager = &TradingAccountManager{client: c}
	c.alertManager = &AlertManager{client: c}
	c.forecastManager = &ForecastManager{client: c}
	c.scannerManager = &ScannerManager{client: c}
	c.allocationManager = &AllocationManager{client: c}
	c.modelManager = &ModelManager{client: c}
	c.fyiManager = &FYIManager{client: c}
	c.oauthManager = &OAuthManager{client: c}
	c.watchlistManager = &WatchlistManager{client: c}
	c.performanceManager = &PerformanceManager{client: c}
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

// TradingAccount returns the trading account manager.
func (c *Client) TradingAccount() *TradingAccountManager { return c.tradingAccountManager }

// Alert returns the alert manager.
func (c *Client) Alert() *AlertManager { return c.alertManager }

// Forecast returns the forecast manager.
func (c *Client) Forecast() *ForecastManager { return c.forecastManager }

// Scanner returns the scanner manager.
func (c *Client) Scanner() *ScannerManager { return c.scannerManager }

// Allocation returns the allocation manager.
func (c *Client) Allocation() *AllocationManager { return c.allocationManager }

// Model returns the model portfolio manager.
func (c *Client) Model() *ModelManager { return c.modelManager }

// FYI returns the FYI notification manager.
func (c *Client) FYI() *FYIManager { return c.fyiManager }

// OAuth returns the OAuth manager.
func (c *Client) OAuth() *OAuthManager { return c.oauthManager }

// Watchlist returns the watchlist manager.
func (c *Client) Watchlist() *WatchlistManager { return c.watchlistManager }

// Performance returns the performance analyst manager.
func (c *Client) Performance() *PerformanceManager { return c.performanceManager }

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
	c.wsMu.Lock()
	ws := c.ws
	c.ws = nil
	c.wsMu.Unlock()
	if ws != nil {
		_ = ws.Close()
	}
	if c.release != nil {
		c.release()
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

// metricsSink returns the configured metrics sink, or nil when metrics are
// disabled.
func (c *Client) metricsSink() internal.Metrics { return c.cfg.metrics }

// telemetrySink returns the configured telemetry sink, or nil when disabled.
func (c *Client) telemetrySink() internal.Telemetry { return c.cfg.telemetry }

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
		base = &http.Transport{ //nolint:gosec // opt-in, localhost only
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			ForceAttemptHTTP2:   true,
			TLSHandshakeTimeout: 10 * time.Second,
		}
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
	if cfg.rateLimit < 0 {
		if v := os.Getenv("IBKR_RATE_LIMIT"); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				cfg.rateLimit = f
			}
		}
	}
	if cfg.globalRateLimit < 0 {
		if v := os.Getenv("IBKR_GLOBAL_RATE_LIMIT"); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				cfg.globalRateLimit = f
			}
		}
	}
	if cfg.restGatewayURL == "" {
		if v := os.Getenv("IBKR_REST_GATEWAY_URL"); v != "" {
			cfg.restGatewayURL = v
		}
	}
	if cfg.oauth2.ClientID == "" {
		cfg.oauth2.ClientID = os.Getenv("IBKR_CLIENT_ID")
	}
	if cfg.oauth2.ClientSecret == "" {
		cfg.oauth2.ClientSecret = os.Getenv("IBKR_CLIENT_SECRET")
	}
	if cfg.oauth2.RefreshToken == "" {
		cfg.oauth2.RefreshToken = os.Getenv("IBKR_CLIENT_REFRESH_TOKEN")
	}
	if cfg.logger == nil {
		cfg.logger = loggerFromEnv()
	}
	if cfg.logger == nil {
		cfg.logger = internal.NopLogger()
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
