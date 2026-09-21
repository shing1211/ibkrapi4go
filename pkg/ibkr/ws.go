// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shing1211/ibkrapi4go/internal"
)

// Update is a single market-data field update delivered on a Subscription.
// Values are strings exactly as returned by IBKR (ADR 0008).
type Update struct {
	// ConID is the contract the update belongs to.
	ConID ConID
	// Field is the market-data field code.
	Field Field
	// Value is the field value as a decimal/string.
	Value string
	// Received is when the client received the update.
	Received time.Time
}

// StreamingLimits bounds streaming subscriptions and buffering. Zero fields are
// replaced with defaults by WithStreamingLimits.
type StreamingLimits struct {
	// MaxConIDsPerRequest is the maximum conids per subscribe call (default 100).
	MaxConIDsPerRequest int
	// MaxFieldsPerRequest is the maximum fields per subscribe call (default 50).
	MaxFieldsPerRequest int
	// MaxSubscriptions is the maximum concurrent subscriptions per client (default 10).
	MaxSubscriptions int
	// BufferSize is the per-subscription update buffer (default 256).
	BufferSize int
	// ReconnectBase is the initial reconnect backoff (default 1s).
	ReconnectBase time.Duration
	// ReconnectMax caps the reconnect backoff (default 30s).
	ReconnectMax time.Duration
}

func defaultStreamingLimits() StreamingLimits {
	return StreamingLimits{
		MaxConIDsPerRequest: 100,
		MaxFieldsPerRequest: 50,
		MaxSubscriptions:    10,
		BufferSize:          256,
		ReconnectBase:       time.Second,
		ReconnectMax:        30 * time.Second,
	}
}

func (l StreamingLimits) withDefaults() StreamingLimits {
	d := defaultStreamingLimits()
	if l.MaxConIDsPerRequest <= 0 {
		l.MaxConIDsPerRequest = d.MaxConIDsPerRequest
	}
	if l.MaxFieldsPerRequest <= 0 {
		l.MaxFieldsPerRequest = d.MaxFieldsPerRequest
	}
	if l.MaxSubscriptions <= 0 {
		l.MaxSubscriptions = d.MaxSubscriptions
	}
	if l.BufferSize <= 0 {
		l.BufferSize = d.BufferSize
	}
	if l.ReconnectBase <= 0 {
		l.ReconnectBase = d.ReconnectBase
	}
	if l.ReconnectMax <= 0 {
		l.ReconnectMax = d.ReconnectMax
	}
	return l
}

// Subscription is a channel-based market-data subscription. Callers consume
// Updates and Errors, and must Close it to release resources.
type Subscription struct {
	handle  *internal.WSHandle
	wants   map[int]struct{}
	updates chan Update
	errs    chan error
	dropped atomic.Int64

	closedCh chan struct{}

	mu        sync.RWMutex
	isClosed  bool
	cancel    context.CancelFunc
	closeOnce sync.Once
}

func newSubscription(conids []ConID, buffer int) *Subscription {
	if buffer <= 0 {
		buffer = 256
	}
	wants := make(map[int]struct{}, len(conids))
	for _, c := range conids {
		wants[int(c)] = struct{}{}
	}
	return &Subscription{
		wants:    wants,
		updates:  make(chan Update, buffer),
		errs:     make(chan error, buffer),
		closedCh: make(chan struct{}),
	}
}

// Updates returns the stream of field updates. The channel is closed by Close.
func (s *Subscription) Updates() <-chan Update { return s.updates }

// Errors returns the stream of connection-level events (errors and reconnect
// notices). The channel is closed by Close.
func (s *Subscription) Errors() <-chan error { return s.errs }

// Dropped returns the number of updates dropped due to a full buffer.
func (s *Subscription) Dropped() int64 { return s.dropped.Load() }

// Close unsubscribes and closes the update/error channels. It is idempotent.
func (s *Subscription) Close() error {
	s.closeOnce.Do(func() {
		close(s.closedCh)
		if s.cancel != nil {
			s.cancel()
		}
		if s.handle != nil {
			_ = s.handle.Close()
		}
		s.mu.Lock()
		s.isClosed = true
		close(s.updates)
		close(s.errs)
		s.mu.Unlock()
	})
	return nil
}

// Deliver implements internal.WSSink. On a full buffer it drops the oldest
// update and increments Dropped.
func (s *Subscription) Deliver(u internal.WSUpdate) {
	up := Update{ConID: ConID(u.ConID), Field: Field(u.Field), Value: u.Value, Received: time.Now()}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.isClosed {
		return
	}
	select {
	case s.updates <- up:
	default:
		select {
		case <-s.updates:
		default:
		}
		select {
		case s.updates <- up:
		default:
		}
		s.dropped.Add(1)
	}
}

// Fail implements internal.WSSink.
func (s *Subscription) Fail(err error) {
	if err == nil {
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.isClosed {
		return
	}
	select {
	case s.errs <- err:
	default:
	}
}

// Wants implements internal.WSSink.
func (s *Subscription) Wants(conid int) bool {
	_, ok := s.wants[conid]
	return ok
}

// Subscribe opens a real-time market-data subscription. The returned
// Subscription delivers updates until Close or context cancellation.
func (m *MarketDataManager) Subscribe(ctx context.Context, conids []ConID, fields []Field) (*Subscription, error) {
	const op = "MarketData.Subscribe"
	if err := m.client.checkOpen(); err != nil {
		return nil, err
	}
	lim := m.client.cfg.streamingLimits
	if len(conids) == 0 {
		return nil, &Error{Op: op, Code: "streaming_limit", Message: "no conids provided", Err: ErrInvalidRequest}
	}
	if len(conids) > lim.MaxConIDsPerRequest {
		return nil, &Error{Op: op, Code: "streaming_limit",
			Message: fmt.Sprintf("conids exceed limit %d", lim.MaxConIDsPerRequest), Err: ErrStreamingLimit}
	}
	if len(fields) > lim.MaxFieldsPerRequest {
		return nil, &Error{Op: op, Code: "streaming_limit",
			Message: fmt.Sprintf("fields exceed limit %d", lim.MaxFieldsPerRequest), Err: ErrStreamingLimit}
	}

	conn, err := m.client.ensureWS(ctx)
	if err != nil {
		return nil, err
	}
	if conn.ActiveSubscriptions() >= lim.MaxSubscriptions {
		return nil, &Error{Op: op, Code: "streaming_limit",
			Message: fmt.Sprintf("max subscriptions %d reached", lim.MaxSubscriptions), Err: ErrStreamingLimit}
	}

	sub := newSubscription(conids, lim.BufferSize)
	subCtx, cancel := context.WithCancel(ctx)
	sub.cancel = cancel

	intConids := make([]int, len(conids))
	for i, c := range conids {
		intConids[i] = int(c)
	}
	strFields := make([]string, len(fields))
	for i, f := range fields {
		strFields[i] = string(f)
	}

	handle, err := conn.Subscribe(subCtx, sub, intConids, strFields)
	if err != nil {
		cancel()
		_ = sub.Close()
		return nil, &Error{Op: op, Message: err.Error(), Err: err}
	}
	sub.handle = handle

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// sink is already gone; log and exit
			}
		}()
		select {
		case <-subCtx.Done():
			_ = sub.Close()
		case <-sub.closedCh:
		}
	}()
	return sub, nil
}

// ensureWS lazily dials and caches the multiplexed WebSocket connection.
func (c *Client) ensureWS(ctx context.Context) (*internal.WSConn, error) {
	c.wsMu.Lock()
	defer c.wsMu.Unlock()
	if c.ws != nil {
		return c.ws, nil
	}
	lim := c.cfg.streamingLimits
	conn, err := internal.DialWS(ctx, c.cfg.gatewayURL, internal.WSOptions{
		HTTPClient:    c.httpClient,
		Logger:        c.cfg.logger,
		Metrics:       c.cfg.metrics,
		Reconnect:     true,
		ReconnectBase: lim.ReconnectBase,
		ReconnectMax:  lim.ReconnectMax,
	})
	if err != nil {
		return nil, &Error{Op: "MarketData.Subscribe", Message: err.Error(), Err: err}
	}
	c.ws = conn
	return conn, nil
}
