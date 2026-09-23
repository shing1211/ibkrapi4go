// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"net/http"
	"sync"

	"github.com/shing1211/ibkrapi4go/client"
	"github.com/shing1211/ibkrapi4go/internal"
)

// TransportPool holds shared resources for multiple Client instances. It allows
// multiple clients to share one *http.Client, session, and rate limiter,
// reducing connection overhead in multi-account setups (family offices, prop
// trading firms).
//
// TransportPool is safe for concurrent use. Close must be called when all
// clients obtained from the pool are done; shared resources are released only
// when the reference count reaches zero.
type TransportPool struct {
	mu          sync.Mutex
	cfg         config
	httpClient  *http.Client
	session     *internal.Session
	rateLimiter *internal.Limiter
	releaseFn   func()
	refCount    int
	closed      bool
}

// NewTransportPool creates a shared transport pool. Options configure the
// shared HTTP client, session, and rate limiter that all pool clients will use.
// Options that are per-client (like WithAccount) should be passed to
// TransportPool.Client instead.
func NewTransportPool(opts ...Option) (*TransportPool, error) {
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
		if cfg.logger != nil {
			cfg.logger.Warn("ibkr.config insecure TLS skip-verify enabled for non-loopback host",
				"gateway", cfg.gatewayURL)
		}
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
	session = internal.NewSession(internal.SessionConfig{
		ServerURL:      cfg.gatewayURL,
		TickleInterval: cfg.tickleInterval,
		RequestTimeout: cfg.requestTimeout,
		Logger:         cfg.logger,
	})

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

	session.SetHTTPClient(httpClient)

	generated, err := client.NewClientWithResponses(
		cfg.gatewayURL,
		client.WithHTTPClient(httpClient),
	)
	if err != nil {
		_ = session.Close(context.Background())
		return nil, err
	}
	_ = generated

	p := &TransportPool{
		cfg:         cfg,
		httpClient:  httpClient,
		session:     session,
		rateLimiter: limiter,
	}
	p.releaseFn = func() { p.releaseClient() }

	return p, nil
}

// releaseClient decrements the reference count and tears down shared resources
// when the last client is released.
func (p *TransportPool) releaseClient() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.refCount--
	if p.refCount > 0 {
		return
	}
	p.closeLocked()
}

// Client creates a new Client that shares the pool's HTTP client, session, and
// rate limiter. The returned Client's Close method decrements the pool's
// reference count instead of tearing down the shared session.
func (p *TransportPool) Client(opts ...Option) (*Client, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, &ConfigError{Field: "TransportPool", Message: "pool is closed"}
	}
	p.refCount++
	p.mu.Unlock()

	cfg := p.cfg
	for _, o := range opts {
		if err := o(&cfg); err != nil {
			p.releaseClient()
			return nil, err
		}
	}

	c := &Client{
		cfg:        cfg,
		httpClient: p.httpClient,
		session:    p.session,
		release:    p.releaseFn,
	}

	c.sessionManager = &SessionManager{client: c}
	c.accountManager = &AccountManager{client: c}
	c.portfolioManager = &PortfolioManager{client: c}
	c.tradeManager = &TradeManager{client: c}
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

// Close releases all shared resources. It must be called after all clients
// obtained from the pool are done. Shared resources are torn down only when the
// last client's Close has been called (reference count reaches zero).
func (p *TransportPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	if p.refCount == 0 {
		p.closeLocked()
	}
	return nil
}

// closeLocked tears down shared resources. Caller must hold p.mu.
func (p *TransportPool) closeLocked() {
	if p.session != nil {
		ctx, cancel := context.WithTimeout(context.Background(), closeLogoutTimeout)
		defer cancel()
		_ = p.session.Close(ctx)
	}
	p.httpClient = nil
	p.session = nil
	p.rateLimiter = nil
}
