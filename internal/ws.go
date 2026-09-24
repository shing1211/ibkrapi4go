// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

// Streaming errors.
var (
	// ErrWSDisconnected is delivered to subscribers when the connection drops.
	ErrWSDisconnected = errors.New("ibkr: ws: disconnected")
	// ErrWSReconnected is delivered to subscribers after a successful reconnect.
	ErrWSReconnected = errors.New("ibkr: ws: reconnected")
)

// WSGapError signals a sequence discontinuity for a contract. A gap of more
// than 1 in the _updated sequence indicates missed messages.
type WSGapError struct {
	Conid       int
	LastSeq     int64
	ReceivedSeq int64
}

func (e *WSGapError) Error() string {
	return "ibkr: ws: sequence gap for conid " + itoa(int64(e.Conid)) +
		": last=" + itoa(e.LastSeq) + " received=" + itoa(e.ReceivedSeq)
}

// itoa converts an int to a string without importing fmt.
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + uitoa(uint64(-n))
	}
	return uitoa(uint64(n))
}

func uitoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// WSUpdate is a single field update dispatched to a sink.
type WSUpdate struct {
	ConID int
	Field string
	Value string
}

// WSSystemFrame is a non-market-data (system) frame received from the gateway.
type WSSystemFrame struct {
	Type    string
	Status  string
	Topic   string
	Payload []byte
}

// WSSink receives routed market-data updates for a subscription. Implementations
// must not block: Deliver is called from the connection's reader goroutine.
type WSSink interface {
	// Wants reports whether the sink is interested in the given conid.
	Wants(conid int) bool
	// Deliver receives a scalar field update.
	Deliver(WSUpdate)
	// Fail receives connection-level events (drops, errors, reconnects).
	Fail(error)
}

// WSSystemSink receives non-market-data frames (sts, ntf, sor, usr).
type WSSystemSink interface {
	// WantsSystem reports whether the sink wants system updates.
	WantsSystem() bool
	// DeliverSystem receives a system frame.
	DeliverSystem(WSSystemFrame)
	// Fail receives connection-level events (drops, errors, reconnects).
	Fail(error)
}

// DialWSFunc dials a WebSocket connection.
type DialWSFunc func(ctx context.Context, wsURL string, httpClient *http.Client) (*websocket.Conn, error)

// WSOptions configures a WSConn.
type WSOptions struct {
	HTTPClient    *http.Client
	Logger        *slog.Logger
	Metrics       Metrics
	PingInterval  time.Duration
	PongTimeout   time.Duration
	Reconnect     bool
	ReconnectBase time.Duration
	ReconnectMax  time.Duration
	Clock         *Clock
	DialWS        DialWSFunc
	Telemetry     Telemetry
}

// WSHandle is a registered subscription on a WSConn.
type WSHandle struct {
	conn *WSConn
	sub  *wsSub
	once sync.Once
}

type wsSub struct {
	sink       WSSink
	systemSink WSSystemSink
	conids     []int
	fields     []string
}

// WSConn is a single multiplexed WebSocket connection to the gateway.
type WSConn struct {
	wsURL string
	opts  WSOptions
	ctx   context.Context

	mu   sync.Mutex
	conn *websocket.Conn
	subs map[*wsSub]struct{}

	lastUpdated map[int]int64

	out    chan []byte
	stopCh chan struct{}
	doneCh chan struct{}
	closed atomic.Bool
	nextID atomic.Int64
	wg     sync.WaitGroup
}

// DialWS connects to the gateway WebSocket derived from gatewayURL and starts
// the reader, writer, and ping goroutines. The returned WSConn owns them until
// Close.
func DialWS(ctx context.Context, gatewayURL string, opts WSOptions) (*WSConn, error) {
	wsURL, err := wsURLFromGateway(gatewayURL)
	if err != nil {
		return nil, err
	}
	if opts.PingInterval <= 0 {
		opts.PingInterval = 30 * time.Second
	}
	if opts.PongTimeout <= 0 {
		opts.PongTimeout = 10 * time.Second
	}
	if opts.ReconnectBase <= 0 {
		opts.ReconnectBase = time.Second
	}
	if opts.ReconnectMax <= 0 {
		opts.ReconnectMax = 30 * time.Second
	}
	if opts.Logger == nil {
		opts.Logger = NopLogger()
	}
	if opts.Clock == nil {
		opts.Clock = &Clock{}
	}
	if opts.DialWS == nil {
		opts.DialWS = func(ctx context.Context, wsURL string, httpClient *http.Client) (*websocket.Conn, error) {
			dctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			conn, _, err := websocket.Dial(dctx, wsURL, &websocket.DialOptions{HTTPClient: httpClient})
			return conn, err
		}
	}
	if opts.Telemetry == nil {
		opts.Telemetry = NopTelemetry()
	}
	c := &WSConn{
		wsURL:  wsURL,
		opts:   opts,
		ctx:    ctx,
		subs:   map[*wsSub]struct{}{},
		out:    make(chan []byte, 64),
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
	if err := c.dial(ctx); err != nil {
		return nil, err
	}
	incrCounter(ctx, c.opts.Metrics, MetricWSConnects, 1)
	c.opts.Telemetry.OnWSConnect(ctx, WSConnInfo{Event: "connect", URL: c.wsURL, Subscriptions: 0})
	c.wg.Add(3)
	go c.wgDoneWrapper(c.readLoop)
	go c.wgDoneWrapper(c.writeLoop)
	go c.wgDoneWrapper(c.pingLoop)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.opts.Logger.Error("ibkr.ws goroutine panicked", "panic", r)
			}
		}()
		c.wg.Wait()
		close(c.doneCh)
	}()
	return c, nil
}

// Subscribe registers a subscription for conids/fields and sends the subscribe
// frame. Deliveries begin once frames arrive.
func (c *WSConn) Subscribe(ctx context.Context, sink WSSink, systemSink WSSystemSink, conids []int, fields []string) (*WSHandle, error) {
	if c.closed.Load() {
		return nil, ErrClosed
	}
	s := &wsSub{sink: sink, systemSink: systemSink, conids: conids, fields: fields}
	c.mu.Lock()
	c.subs[s] = struct{}{}
	c.mu.Unlock()
	setGauge(c.ctx, c.opts.Metrics, MetricWSActiveSubscriptions, float64(c.ActiveSubscriptions()))
	if err := c.send(ctx, "subscribe", subscribeParams(s)); err != nil {
		c.mu.Lock()
		delete(c.subs, s)
		c.mu.Unlock()
		return nil, err
	}
	c.opts.Telemetry.OnWSSubscribe(ctx, WSSubInfo{Event: "subscribe", ConIDs: conids, Fields: fields})
	return &WSHandle{conn: c, sub: s}, nil
}

// Close unsubscribes and removes the handle. It is idempotent.
func (h *WSHandle) Close() error {
	h.once.Do(func() {
		h.conn.mu.Lock()
		delete(h.conn.subs, h.sub)
		h.conn.mu.Unlock()
		setGauge(h.conn.ctx, h.conn.opts.Metrics, MetricWSActiveSubscriptions, float64(h.conn.ActiveSubscriptions()))
		h.conn.opts.Telemetry.OnWSUnsubscribe(context.Background(), WSSubInfo{Event: "unsubscribe", ConIDs: h.sub.conids})
		_ = h.conn.send(context.Background(), "unsubscribe", map[string]any{"conids": h.sub.conids})
	})
	return nil
}

// ActiveSubscriptions reports the number of registered subscriptions.
func (c *WSConn) ActiveSubscriptions() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.subs)
}

// Close stops all goroutines and closes the connection. It is idempotent and
// waits (bounded) for background goroutines to exit.
func (c *WSConn) Close() error {
	if c.closed.Swap(true) {
		return nil
	}
	c.opts.Telemetry.OnWSDisconnect(c.ctx, WSConnInfo{Event: "disconnect", URL: c.wsURL, Subscriptions: c.ActiveSubscriptions()})
	close(c.stopCh)
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn != nil {
		_ = conn.Close(websocket.StatusNormalClosure, "client closing")
	}
	select {
	case <-c.doneCh:
	case <-c.opts.Clock.After(3 * time.Second):
	}
	return nil
}

// --- internals --------------------------------------------------------------

func (c *WSConn) dial(ctx context.Context) error {
	conn, err := c.opts.DialWS(ctx, c.wsURL, c.opts.HTTPClient)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	return nil
}

func (c *WSConn) currentConn() *websocket.Conn {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn
}

func (c *WSConn) readLoop() {
	attempt := 0
	for {
		if c.closed.Load() {
			return
		}
		select {
		case <-c.ctx.Done():
			return
		default:
		}
		conn := c.currentConn()
		if conn == nil {
			if !c.opts.Reconnect || c.reconnect(&attempt) != nil {
				return
			}
			continue
		}
		_, data, err := conn.Read(c.ctx)
		if err == nil {
			attempt = 0
			c.dispatch(data)
			continue
		}
		if c.closed.Load() {
			return
		}
		if c.ctx.Err() != nil {
			return
		}
		if !c.opts.Reconnect {
			c.failAll(err)
			return
		}
		c.failAll(ErrWSDisconnected)
		if c.reconnect(&attempt) != nil {
			return
		}
	}
}

func (c *WSConn) reconnect(attempt *int) error {
	for {
		delay := backoffDelay(*attempt, c.opts.ReconnectBase, c.opts.ReconnectMax)
		select {
		case <-c.opts.Clock.After(delay):
		case <-c.stopCh:
			return ErrClosed
		case <-c.ctx.Done():
			return c.ctx.Err()
		}
		if c.closed.Load() {
			return ErrClosed
		}
		if c.ctx.Err() != nil {
			return c.ctx.Err()
		}
		incrCounter(c.ctx, c.opts.Metrics, MetricWSReconnects, 1)
		if err := c.dial(c.ctx); err != nil {
			c.opts.Logger.Warn("ibkr.ws reconnect failed", "err", err)
			*attempt++
			continue
		}
		*attempt = 0
		c.notifyReconnect()
		c.resubscribeAll()
		c.opts.Telemetry.OnWSConnect(c.ctx, WSConnInfo{Event: "reconnect", URL: c.wsURL, Subscriptions: c.ActiveSubscriptions()})
		return nil
	}
}

func (c *WSConn) resubscribeAll() {
	for _, s := range c.snapshotSubs() {
		_ = c.send(c.ctx, "subscribe", subscribeParams(s))
	}
}

func (c *WSConn) writeLoop() {
	for {
		setGauge(c.ctx, c.opts.Metrics, MetricWSQueueDepth, float64(len(c.out)))
		select {
		case b := <-c.out:
			conn := c.currentConn()
			if conn == nil {
				continue
			}
			ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
			if err := conn.Write(ctx, websocket.MessageText, b); err != nil {
				c.opts.Logger.Warn("ibkr.ws write failed", "err", err)
			}
			cancel()
		case <-c.stopCh:
			return
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *WSConn) pingLoop() {
	ticker := c.opts.Clock.NewTicker(c.opts.PingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			conn := c.currentConn()
			if conn == nil {
				continue
			}
			ctx, cancel := context.WithTimeout(c.ctx, c.opts.PongTimeout)
			err := conn.Ping(ctx)
			cancel()
			if err != nil {
				c.opts.Logger.Warn("ibkr.ws ping failed", "err", err)
				incrCounter(c.ctx, c.opts.Metrics, MetricWSHeartbeatFailures, 1)
				_ = conn.Close(websocket.StatusPolicyViolation, "ping timeout")
			}
		case <-c.stopCh:
			return
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *WSConn) send(ctx context.Context, method string, params map[string]any) error {
	if c.closed.Load() {
		return ErrClosed
	}
	frame := map[string]any{"id": c.nextID.Add(1), "method": method, "params": params}
	b, err := json.Marshal(frame)
	if err != nil {
		return err
	}
	select {
	case c.out <- b:
		return nil
	case <-c.stopCh:
		return ErrClosed
	case <-ctx.Done():
		return ctx.Err()
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}

func (c *WSConn) wgDoneWrapper(fn func()) {
	defer c.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			c.opts.Logger.Error("ibkr.ws goroutine panicked", "panic", r)
		}
	}()
	fn()
}

func (c *WSConn) waitForDone() {
	c.wg.Wait()
	close(c.doneCh)
}

func (c *WSConn) dispatch(data []byte) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return
	}
	if raw, ok := m["error"]; ok {
		var msg string
		if json.Unmarshal(raw, &msg) == nil && msg != "" {
			c.failAll(errors.New("ibkr: ws: " + msg))
			return
		}
	}

	if frame := parseSystemFrame(m); frame != nil {
		c.deliverSystem(frame)
	}

	rawConid, ok := m["conid"]
	if !ok {
		return
	}
	var num json.Number
	if json.Unmarshal(rawConid, &num) != nil {
		return
	}
	conid := int(jsonNumberToInt64(num))

	// Gap detection: check _updated sequence before reserved-field filtering.
	if rawUpdated, ok := m["_updated"]; ok {
		var seqNum json.Number
		if json.Unmarshal(rawUpdated, &seqNum) == nil {
			newSeq, _ := seqNum.Int64()
			c.mu.Lock()
			lastSeq, seen := c.lastUpdated[conid]
			c.lastUpdated[conid] = newSeq
			c.mu.Unlock()
			if seen && newSeq < lastSeq {
				gap := lastSeq - newSeq
				if gap > 1 {
					c.failAll(&WSGapError{Conid: conid, LastSeq: lastSeq, ReceivedSeq: newSeq})
				}
			}
		}
	}

	updates := make([]WSUpdate, 0, len(m))
	for k, v := range m {
		if wsReservedField(k) {
			continue
		}
		val, ok := wsScalarString(v)
		if !ok {
			continue
		}
		updates = append(updates, WSUpdate{ConID: conid, Field: k, Value: val})
	}
	if len(updates) == 0 {
		return
	}
	delivered := false
	for _, s := range c.snapshotSubs() {
		if !s.sink.Wants(conid) {
			continue
		}
		delivered = true
		for _, u := range updates {
			s.sink.Deliver(u)
		}
	}
	if !delivered {
		incrCounter(c.ctx, c.opts.Metrics, MetricWSDroppedEvents, int64(len(updates)))
	}
}

func parseSystemFrame(m map[string]json.RawMessage) *WSSystemFrame {
	if len(m) == 0 {
		return nil
	}
	hasConid := false
	for k := range m {
		if k == "conid" {
			hasConid = true
			break
		}
	}
	if hasConid {
		return nil
	}

	var frame WSSystemFrame
	switch {
	case m["sts"] != nil:
		frame.Type = "sts"
		var status string
		if json.Unmarshal(m["sts"], &status) == nil {
			frame.Status = status
		}
		if t, ok := m["topic"]; ok {
			json.Unmarshal(t, &frame.Topic)
		}
		return &frame
	case m["ntf"] != nil:
		frame.Type = "ntf"
		if t, ok := m["topic"]; ok {
			json.Unmarshal(t, &frame.Topic)
		}
		frame.Payload = m["ntf"]
		return &frame
	case m["sor"] != nil:
		frame.Type = "sor"
		frame.Payload = m["sor"]
		return &frame
	case m["usr"] != nil:
		frame.Type = "usr"
		frame.Payload = m["usr"]
		return &frame
	default:
		return nil
	}
}

func (c *WSConn) deliverSystem(frame *WSSystemFrame) {
	for _, s := range c.snapshotSubs() {
		if s.systemSink != nil && s.systemSink.WantsSystem() {
			s.systemSink.DeliverSystem(*frame)
		}
	}
}

func (c *WSConn) snapshotSubs() []*wsSub {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*wsSub, 0, len(c.subs))
	for s := range c.subs {
		out = append(out, s)
	}
	return out
}

func (c *WSConn) failAll(err error) {
	for _, s := range c.snapshotSubs() {
		s.sink.Fail(err)
	}
}

func (c *WSConn) notifyReconnect() {
	for _, s := range c.snapshotSubs() {
		s.sink.Fail(ErrWSReconnected)
	}
}

func subscribeParams(s *wsSub) map[string]any {
	params := map[string]any{"conids": s.conids}
	if len(s.fields) > 0 {
		params["fields"] = s.fields
	}
	return params
}

// wsReservedField reports whether a frame key is metadata rather than a field.
func wsReservedField(k string) bool {
	switch k {
	case "conid", "_updated", "server_id", "6119", "topic", "method", "id":
		return true
	default:
		return false
	}
}

// wsScalarString renders a scalar JSON value (string, number, bool) as a string.
// It returns false for objects, arrays, and null.
func wsScalarString(v json.RawMessage) (string, bool) {
	s := strings.TrimSpace(string(v))
	if s == "" || s == "null" {
		return "", false
	}
	switch s[0] {
	case '{', '[':
		return "", false
	case '"':
		var str string
		if json.Unmarshal(v, &str) != nil {
			return "", false
		}
		return str, true
	case 't', 'f':
		var b bool
		if json.Unmarshal(v, &b) != nil {
			return "", false
		}
		if b {
			return "true", true
		}
		return "false", true
	default:
		var n json.Number
		if json.Unmarshal(v, &n) != nil {
			return "", false
		}
		return n.String(), true
	}
}

// wsURLFromGateway converts an http(s) gateway URL into a ws(s) /v1/api/ws URL.
func wsURLFromGateway(gatewayURL string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(gatewayURL))
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	case "wss", "ws":
	default:
		return "", errors.New("ibkr: ws: unsupported gateway scheme")
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + "/v1/api/ws"
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

// backoffDelay returns an exponential backoff with full jitter, capped at max.
func backoffDelay(attempt int, base, max time.Duration) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	if attempt > 5 {
		attempt = 5
	}
	d := base << attempt
	if d > max {
		d = max
	}
	return time.Duration(rand.Int63n(int64(d) + 1))
}

func jsonNumberToInt64(n json.Number) int64 {
	if n == "" {
		return 0
	}
	if i, err := n.Int64(); err == nil {
		return i
	}
	if f, err := n.Float64(); err == nil {
		return int64(f)
	}
	return 0
}

// NewTestWSHandle returns a *WSHandle whose Close method is a no-op. It is
// intended for use in unit tests where a WSClient fake needs to return a
// concrete handle without a live connection.
func NewTestWSHandle() *WSHandle {
	h := &WSHandle{}
	h.once.Do(func() {}) // pre-trigger so Close() is safe
	return h
}
