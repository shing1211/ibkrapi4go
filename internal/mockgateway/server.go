// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// Package mockgateway provides a dependency-free, in-repo mock of the IBKR
// Client Portal Gateway. It serves deterministic fixtures, records requests,
// and supports scriptable fault injection. It deliberately does not import
// testing, so it can be used from tests, examples, and a standalone binary.
package mockgateway

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"
)

// maxBodyBytes bounds how much of a request body the mock will read.
const maxBodyBytes = 1 << 20

// maxTimeoutFault caps how long a Timeout fault blocks, so a misconfigured
// fault cannot hang a test forever.
const maxTimeoutFault = 10 * time.Second

// Server is an in-memory mock IBKR gateway. It is safe for concurrent use.
type Server struct {
	routes       []route
	fixtures     *Fixtures
	recorder     *Recorder
	scenario     *Scenario
	sessions     *sessionStore
	oauth        *oauthStore
	stream       *StreamHub
	seed         int64
	authRequired bool
}

// New builds a Server from functional options.
func New(opts ...Option) *Server {
	o := options{seed: 1}
	for _, fn := range opts {
		fn(&o)
	}
	if o.fixtures == nil {
		o.fixtures = DefaultFixtures()
	}
	if o.recorder == nil {
		o.recorder = NewRecorder()
	}
	if o.scenario == nil {
		o.scenario = NewScenario()
	}
	if o.latency > 0 {
		o.scenario.SetLatency(o.latency)
	}
	if o.faults != nil {
		o.scenario.SetPolicy(o.faults)
	}
	script := StreamScript{}
	if o.streamScript != nil {
		script = *o.streamScript
	}
	return &Server{
		routes:       defaultRoutes(),
		fixtures:     o.fixtures,
		recorder:     o.recorder,
		scenario:     o.scenario,
		sessions:     &sessionStore{},
		oauth:        newOAuthStore(),
		stream:       newStreamHub(script, o.seed),
		seed:         o.seed,
		authRequired: o.authRequired,
	}
}

// Handler returns the http.Handler serving the mock gateway.
func (s *Server) Handler() http.Handler { return s }

// Recorder returns the request recorder.
func (s *Server) Recorder() *Recorder { return s.recorder }

// Fixtures returns the fixture registry.
func (s *Server) Fixtures() *Fixtures { return s.fixtures }

// Hub returns the StreamHub for the mock gateway. Tests use it to inject
// frames directly (e.g., to simulate gaps).
func (s *Server) Hub() *StreamHub { return s.stream }

// Scenario returns the fault-injection scenario.
func (s *Server) Scenario() *Scenario { return s.scenario }

// Stream returns the WebSocket streaming hub.
func (s *Server) Stream() *StreamHub { return s.stream }

// PushTick broadcasts a scripted market-data tick to subscribed WebSocket
// clients and returns the number of connections written to.
func (s *Server) PushTick(t Tick) int { return s.stream.Push(t) }

// Seed returns the configured deterministic seed.
func (s *Server) Seed() int64 { return s.seed }

// Operations returns the operation IDs of every registered route, in
// registration order and without duplicates.
func (s *Server) Operations() []string {
	seen := make(map[string]bool, len(s.routes))
	out := make([]string, 0, len(s.routes))
	for _, rt := range s.routes {
		if seen[rt.op] {
			continue
		}
		seen[rt.op] = true
		out = append(out, rt.op)
	}
	return out
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := &Request{
		Method:  r.Method,
		Path:    r.URL.Path,
		Query:   r.URL.Query(),
		Headers: r.Header.Clone(),
		Time:    time.Now(),
	}
	if r.Body != nil {
		body, _ := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
		req.Body = body
	}
	s.recorder.Record(req)

	rt, params, ok := matchRoute(s.routes, r.Method, r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, `{"error":"unknown path"}`)
		return
	}
	req.Params = params
	op := rt.op

	if !s.applyScenario(w, r, req, op) {
		return
	}

	if isSessionOp(op) {
		s.serveSession(w, req, op)
		return
	}

	if op == OpGenerateToken {
		s.serveToken(w, req)
		return
	}

	if rt.bearer {
		if !s.oauth.authenticate(req) {
			writeJSON(w, http.StatusUnauthorized,
				`{"error":"invalid_token","error_description":"bearer token is missing or invalid"}`)
			return
		}
	}

	if rt.protected {
		if s.authRequired && !s.sessions.isAuthenticated(req) {
			writeJSON(w, http.StatusUnauthorized, `{"error":"not authenticated"}`)
			return
		}
		if s.scenario.SessionExpired() {
			writeJSON(w, http.StatusUnauthorized, `{"error":"session expired"}`)
			return
		}
	}

	if op == OpOpenWebsocket {
		s.serveWS(w, r)
		return
	}

	if op == OpSubmitNewOrder && s.scenario.OrderReplyConfirm() {
		body := s.scenario.OrderReplyBody()
		if body == "" {
			body = `[{"id":"reply-1","message":["Confirm this order"],"messageIds":["o354"]}]`
		}
		writeJSON(w, http.StatusOK, body)
		return
	}

	fx, ok := s.fixtures.Get(op)
	if !ok {
		writeJSON(w, http.StatusNotFound, `{"error":"unknown path"}`)
		return
	}
	status, body := fx.Status, fx.Body
	if status == 0 {
		status = http.StatusOK
	}
	if fx.Dynamic != nil {
		status, body = fx.Dynamic(req)
		if status == 0 {
			status = http.StatusOK
		}
	}
	writeJSON(w, status, body)
}

// applyScenario resolves and applies latency and any injected fault. It returns
// false when the fault already produced the response (or dropped the
// connection).
func (s *Server) applyScenario(w http.ResponseWriter, r *http.Request, req *Request, op string) bool {
	fault := s.scenario.FaultFor(op, req)

	delay := s.scenario.Latency()
	if fault != nil && fault.Delay > delay {
		delay = fault.Delay
	}
	if delay > 0 && !sleepCtx(r.Context(), delay) {
		return false
	}
	if fault == nil {
		return true
	}
	if fault.DropConnection {
		dropConnection(w)
		return false
	}
	if fault.Timeout {
		waitCtx(r.Context())
		return false
	}
	if fault.Status != 0 {
		body := fault.Body
		if body == "" {
			body = `{"error":"injected fault"}`
		}
		writeJSON(w, fault.Status, body)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

// sleepCtx sleeps for d unless the context is cancelled first. It reports
// whether the sleep completed.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// waitCtx blocks until the context is cancelled or the safety cap elapses.
func waitCtx(ctx context.Context) {
	t := time.NewTimer(maxTimeoutFault)
	defer t.Stop()
	select {
	case <-t.C:
	case <-ctx.Done():
	}
}

// dropConnection hijacks and closes the connection without a response.
func dropConnection(w http.ResponseWriter) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		return
	}
	conn, _, err := hj.Hijack()
	if err == nil {
		_ = conn.Close()
	}
}

// route is one registered mock route.
type route struct {
	methods   []string
	segments  []string
	op        string
	protected bool
	bearer    bool
}

func mkRoute(op string, methods []string, pattern string, protected bool) route {
	return route{
		methods:   methods,
		segments:  splitPath(pattern),
		op:        op,
		protected: protected,
	}
}

// mkBearerRoute registers an OAuth2 bearer-protected route (IB REST surface).
func mkBearerRoute(op string, methods []string, pattern string) route {
	return route{
		methods:  methods,
		segments: splitPath(pattern),
		op:       op,
		bearer:   true,
	}
}

// defaultRoutes returns the Phase-1 (T1) routes, the remaining CPAPI routes
// (T2), and the IB REST + OAuth2 routes (T3). All 185 operations in
// docs/SPEC.md resolve here.
func defaultRoutes() []route {
	post := []string{http.MethodPost}
	get := []string{http.MethodGet}
	getPost := []string{http.MethodGet, http.MethodPost}
	del := []string{http.MethodDelete}

	routes := []route{
		// session/auth (unauthenticated)
		mkRoute(OpInitializeSession, post, "/v1/api/iserver/auth/ssodh/init", false),
		mkRoute(OpGetBrokerageStatus, getPost, "/v1/api/iserver/auth/status", false),
		mkRoute(OpGetSessionToken, post, "/v1/api/tickle", false),
		mkRoute(OpLogout, post, "/v1/api/logout", false),
		mkRoute(OpGetSessionValidation, get, "/v1/api/sso/validate", false),

		// account
		mkRoute(OpGetBrokerageAccounts, get, "/v1/api/iserver/accounts", true),
		mkRoute(OpGetPnl, get, "/v1/api/iserver/account/pnl/partitioned", true),
		mkRoute(OpGetAccountSummary, get, "/v1/api/iserver/account/{accountId}/summary", true),

		// contracts
		mkRoute(OpGetContractSymbols, get, "/v1/api/iserver/secdef/search", true),
		mkRoute(OpGetContractSymbolsFromBody, post, "/v1/api/iserver/secdef/search", true),
		mkRoute(OpGetInstrumentInfo, get, "/v1/api/iserver/contract/{conid}/info", true),
		mkRoute(OpGetContractRules, post, "/v1/api/iserver/contract/rules", true),
		mkRoute(OpGetContractStrikes, get, "/v1/api/iserver/secdef/strikes", true),

		// portfolio
		mkRoute(OpGetAllAccounts, get, "/v1/api/portfolio/accounts", true),
		mkRoute(OpGetAllSubaccounts, get, "/v1/api/portfolio/subaccounts", true),
		mkRoute(OpGetUncachedPositions, get, "/v1/api/portfolio2/{accountId}/positions", true),
		mkRoute(OpInvalidatePositionCache, post, "/v1/api/portfolio/{accountId}/positions/invalidate", true),
		mkRoute(OpGetPaginatedPositions, get, "/v1/api/portfolio/{accountId}/positions/{pageId}", true),
		mkRoute(OpGetPositionByConid, get, "/v1/api/portfolio/{accountId}/position/{conid}", true),
		mkRoute(OpGetPortfolioLedger, get, "/v1/api/portfolio/{accountId}/ledger", true),
		mkRoute(OpGetAssetAllocation, get, "/v1/api/portfolio/{accountId}/allocation", true),
		mkRoute(OpGetPortfolioSummary, get, "/v1/api/portfolio/{accountId}/summary", true),
		mkRoute(OpGetPortfolioMetadata, get, "/v1/api/portfolio/{accountId}/meta", true),

		// market data
		mkRoute(OpGetMdSnapshot, get, "/v1/api/iserver/marketdata/snapshot", true),
		mkRoute(OpGetMdHistory, get, "/v1/api/iserver/marketdata/history", true),
		mkRoute(OpCloseMdStream, post, "/v1/api/iserver/marketdata/unsubscribe", true),
		mkRoute(OpCloseAllMdStreams, get, "/v1/api/iserver/marketdata/unsubscribeall", true),

		// orders and trades
		mkRoute(OpSubmitNewOrder, post, "/v1/api/iserver/account/{accountId}/orders", true),
		mkRoute(OpConfirmOrderReply, post, "/v1/api/iserver/reply/{replyId}", true),
		mkRoute(OpPreviewMarginImpact, post, "/v1/api/iserver/account/{accountId}/orders/whatif", true),
		mkRoute(OpModifyOpenOrder, post, "/v1/api/iserver/account/{accountId}/order/{orderId}", true),
		mkRoute(OpCancelOpenOrder, del, "/v1/api/iserver/account/{accountId}/order/{orderId}", true),
		mkRoute(OpGetOpenOrders, get, "/v1/api/iserver/account/orders", true),
		mkRoute(OpGetOrderStatus, get, "/v1/api/iserver/account/order/status/{orderId}", true),
		mkRoute(OpGetTradeHistory, get, "/v1/api/iserver/account/trades", true),
	}

	return append(append(routes, cpapiRoutes()...), restRoutes()...)
}

// splitPath splits a URL path into its non-empty segments.
func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

// match route methods and segment count, capturing placeholders.
func (rt route) match(parts []string) (map[string]string, bool) {
	if len(parts) != len(rt.segments) {
		return nil, false
	}
	var params map[string]string
	for i, seg := range rt.segments {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			if params == nil {
				params = make(map[string]string, 2)
			}
			params[strings.Trim(seg, "{}")] = parts[i]
			continue
		}
		if seg != parts[i] {
			return nil, false
		}
	}
	return params, true
}

func (rt route) allows(method string) bool {
	if len(rt.methods) == 0 {
		return true
	}
	for _, m := range rt.methods {
		if m == method {
			return true
		}
	}
	return false
}

// specificity scores a route by its number of literal segments.
func (rt route) specificity() int {
	n := 0
	for _, seg := range rt.segments {
		if !strings.HasPrefix(seg, "{") {
			n++
		}
	}
	return n
}

// matchRoute selects the most specific route matching method and path. A method
// match outranks segment specificity so that POST/DELETE routes sharing a path
// resolve correctly.
func matchRoute(routes []route, method, path string) (route, map[string]string, bool) {
	parts := splitPath(path)
	var (
		best      route
		bestParam map[string]string
		bestScore = -1 << 30
	)
	for _, rt := range routes {
		params, ok := rt.match(parts)
		if !ok {
			continue
		}
		score := rt.specificity()
		if rt.allows(method) {
			score += 1 << 20
		}
		if score > bestScore {
			bestScore = score
			best = rt
			bestParam = params
		}
	}
	return best, bestParam, bestScore > -1<<30
}
