// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"bytes"
	"context"
	"encoding/json"
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
	// Status is the market-data availability (field 6509), or nil.
	// It is non-nil when Field is "6509".
	Status *MarketDataStatus
}

// SystemUpdateType classifies a system (non-market-data) frame.
type SystemUpdateType string

const (
	SystemUpdateStatus       SystemUpdateType = "sts"
	SystemUpdateNotification SystemUpdateType = "ntf"
	SystemUpdateOrder        SystemUpdateType = "sor"
	SystemUpdateUser         SystemUpdateType = "usr"
	SystemUpdateAccount      SystemUpdateType = "acq"
	SystemUpdatePortfolio    SystemUpdateType = "pos"
)

// SystemUpdate is a non-market-data frame delivered on a Subscription.
// It is a discriminated union: check the Type field to determine which
// payload is populated.
type SystemUpdate struct {
	// Type is the frame type: "sts", "ntf", "sor", or "usr".
	Type SystemUpdateType
	// Status is populated for "sts" frames (connection status message).
	Status string
	// Topic is populated for "ntf" frames (notification topic).
	Topic string
	// Payload is the raw JSON payload for "ntf", "sor", or "usr" frames.
	Payload []byte

	// Typed payloads — populated when parsing succeeds, nil on failure.
	// Consumers should check the Type field first, then access the
	// corresponding typed field.
	OrderEvent         *OrderEvent
	NotificationEvent  *NotificationEvent
	UserMessageEvent   *UserMessageEvent
	AccountUpdateEvent *AccountUpdateEvent
	PortfolioEvent     *PortfolioEvent

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

	systemUpdates    chan SystemUpdate
	accountUpdates   chan AccountUpdateEvent
	portfolioUpdates chan PortfolioEvent

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
		wants:            wants,
		updates:          make(chan Update, buffer),
		errs:             make(chan error, buffer),
		systemUpdates:    make(chan SystemUpdate, buffer),
		accountUpdates:   make(chan AccountUpdateEvent, buffer),
		portfolioUpdates: make(chan PortfolioEvent, buffer),
		closedCh:         make(chan struct{}),
	}
}

// Updates returns the stream of field updates. The channel is closed by Close.
func (s *Subscription) Updates() <-chan Update { return s.updates }

// Errors returns the stream of connection-level events (errors and reconnect
// notices). The channel is closed by Close.
func (s *Subscription) Errors() <-chan error { return s.errs }

// SystemUpdates returns the stream of non-market-data frames (connection status,
// notifications, order updates, user messages). The channel is closed by Close.
func (s *Subscription) SystemUpdates() <-chan SystemUpdate { return s.systemUpdates }

// AccountUpdates returns the stream of account value updates ("acq" frames).
// The channel is closed by Close.
func (s *Subscription) AccountUpdates() <-chan AccountUpdateEvent { return s.accountUpdates }

// PortfolioUpdates returns the stream of position updates ("pos" frames).
// The channel is closed by Close.
func (s *Subscription) PortfolioUpdates() <-chan PortfolioEvent { return s.portfolioUpdates }

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
		close(s.systemUpdates)
		close(s.accountUpdates)
		close(s.portfolioUpdates)
		s.mu.Unlock()
	})
	return nil
}

// Deliver implements internal.WSSink. On a full buffer it drops the oldest
// update and increments Dropped.
func (s *Subscription) Deliver(u internal.WSUpdate) {
	up := Update{ConID: ConID(u.ConID), Field: Field(u.Field), Value: u.Value, Received: time.Now()}
	if u.Field == "6509" {
		up.Status = parseMarketDataStatusString(u.Value)
	}
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

// WantsSystem implements internal.WSSystemSink. It always returns true since
// every subscription is eligible to receive system updates.
func (s *Subscription) WantsSystem() bool {
	return true
}

// DeliverSystem implements internal.WSSystemSink.
func (s *Subscription) DeliverSystem(frame internal.WSSystemFrame) {
	received := time.Now()
	up := SystemUpdate{
		Type:     SystemUpdateType(frame.Type),
		Status:   frame.Status,
		Topic:    frame.Topic,
		Payload:  frame.Payload,
		Received: received,
	}
	var portfolioEvents []PortfolioEvent
	switch frame.Type {
	case "sor":
		up.OrderEvent = parseOrderEvent(frame.Payload, received)
	case "ntf":
		up.NotificationEvent = parseNotificationEvent(frame.Topic, frame.Payload, received)
	case "usr":
		up.UserMessageEvent = parseUserMessageEvent(frame.Payload, received)
	case "acq":
		up.AccountUpdateEvent = parseAccountUpdateEvent(frame.Payload, received)
	case "pos":
		portfolioEvents = parsePortfolioEvents(frame.Payload, received)
		if len(portfolioEvents) > 0 {
			up.PortfolioEvent = &portfolioEvents[0]
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.isClosed {
		return
	}
	select {
	case s.systemUpdates <- up:
	default:
	}
	if up.AccountUpdateEvent != nil {
		select {
		case s.accountUpdates <- *up.AccountUpdateEvent:
		default:
		}
	}
	for _, pe := range portfolioEvents {
		select {
		case s.portfolioUpdates <- pe:
		default:
		}
	}
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

	handle, err := conn.Subscribe(subCtx, sub, sub, intConids, strFields)
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

// subscribeChannel opens a non-conid (channel/push) stream subscription using
// the given gateway method (for example "account" or "portfolio"). Account and
// portfolio frames are delivered on the returned Subscription's typed channels.
func (c *Client) subscribeChannel(ctx context.Context, op, method string, fields []Field) (*Subscription, error) {
	if err := c.checkOpen(); err != nil {
		return nil, err
	}
	lim := c.cfg.streamingLimits
	if len(fields) > lim.MaxFieldsPerRequest {
		return nil, &Error{Op: op, Code: "streaming_limit",
			Message: fmt.Sprintf("fields exceed limit %d", lim.MaxFieldsPerRequest), Err: ErrStreamingLimit}
	}

	conn, err := c.ensureWS(ctx)
	if err != nil {
		return nil, err
	}
	if conn.ActiveSubscriptions() >= lim.MaxSubscriptions {
		return nil, &Error{Op: op, Code: "streaming_limit",
			Message: fmt.Sprintf("max subscriptions %d reached", lim.MaxSubscriptions), Err: ErrStreamingLimit}
	}

	sub := newSubscription(nil, lim.BufferSize)
	subCtx, cancel := context.WithCancel(ctx)
	sub.cancel = cancel

	strFields := make([]string, len(fields))
	for i, f := range fields {
		strFields[i] = string(f)
	}

	handle, err := conn.SubscribeStream(subCtx, sub, sub, method, strFields)
	if err != nil {
		cancel()
		_ = sub.Close()
		return nil, &Error{Op: op, Message: err.Error(), Err: err}
	}
	sub.handle = handle

	go func() {
		defer func() {
			_ = recover()
		}()
		select {
		case <-subCtx.Done():
			_ = sub.Close()
		case <-sub.closedCh:
		}
	}()
	return sub, nil
}

// SubscribeAccount opens an account value streaming subscription. Updates are
// delivered on Subscription.AccountUpdates; connection notices on Errors.
func (m *AccountManager) SubscribeAccount(ctx context.Context, fields []Field) (*Subscription, error) {
	return m.client.subscribeChannel(ctx, "Account.SubscribeAccount", "account", fields)
}

// SubscribePortfolio opens a portfolio (position) streaming subscription.
// Updates are delivered on Subscription.PortfolioUpdates.
func (m *PortfolioManager) SubscribePortfolio(ctx context.Context, fields []Field) (*Subscription, error) {
	return m.client.subscribeChannel(ctx, "Portfolio.SubscribePortfolio", "portfolio", fields)
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

// parseOrderEvent parses the raw JSON payload of a "sor" frame into an OrderEvent.
func parseOrderEvent(payload []byte, received time.Time) *OrderEvent {
	var raw struct {
		Conid         *int64  `json:"conid"`
		OrderId       *int64  `json:"order_id"`
		ClientOrderID *string `json:"cOID"`
		Account       *string `json:"account"`
		OrderStatus   *string `json:"order_status"`
		Side          *string `json:"side"`
		OrderType     *string `json:"order_type"`
		Tif           *string `json:"tif"`
		Size          *string `json:"size"`
		CumFill       *string `json:"cum_fill"`
		AveragePrice  *string `json:"average_price"`
		TotalSize     *string `json:"total_size"`
		OrderTime     *string `json:"order_time"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil
	}
	e := &OrderEvent{Received: received}
	if raw.Conid != nil {
		e.Conid = *raw.Conid
	}
	if raw.OrderId != nil {
		e.OrderID = *raw.OrderId
	}
	if raw.ClientOrderID != nil {
		e.ClientOrderID = *raw.ClientOrderID
	}
	if raw.Account != nil {
		e.Account = *raw.Account
	}
	if raw.OrderStatus != nil {
		e.Status = WSOrderStatus(*raw.OrderStatus)
	}
	if raw.Side != nil {
		e.Side = *raw.Side
	}
	if raw.OrderType != nil {
		e.OrderType = *raw.OrderType
	}
	if raw.Tif != nil {
		e.TIF = *raw.Tif
	}
	if raw.Size != nil {
		e.Size = *raw.Size
	}
	if raw.CumFill != nil {
		e.CumFill = *raw.CumFill
	}
	if raw.AveragePrice != nil {
		e.AveragePrice = *raw.AveragePrice
	}
	if raw.TotalSize != nil {
		e.TotalSize = *raw.TotalSize
	}
	if raw.OrderTime != nil {
		e.OrderTime = *raw.OrderTime
	}
	return e
}

// parseNotificationEvent parses the raw JSON payload of an "ntf" frame.
func parseNotificationEvent(topic string, payload []byte, received time.Time) *NotificationEvent {
	var raw struct {
		Message *string `json:"message"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return &NotificationEvent{Topic: topic, Received: received}
	}
	e := &NotificationEvent{Topic: topic, Received: received}
	if raw.Message != nil {
		e.Message = *raw.Message
	}
	return e
}

// parseUserMessageEvent parses the raw JSON payload of a "usr" frame.
func parseUserMessageEvent(payload []byte, received time.Time) *UserMessageEvent {
	var raw struct {
		Message *string `json:"message"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil
	}
	e := &UserMessageEvent{Received: received}
	if raw.Message != nil {
		e.Message = *raw.Message
	}
	return e
}

// parseAccountUpdateEvent parses the raw JSON payload of an "acq" frame.
func parseAccountUpdateEvent(payload []byte, received time.Time) *AccountUpdateEvent {
	var raw struct {
		Account      *string `json:"account"`
		NetLiquidity *string `json:"net"`
		Cash         *string `json:"cash"`
		Equity       *string `json:"equity"`
		MaintMargin  *string `json:"maintmargin"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil
	}
	e := &AccountUpdateEvent{Received: received}
	if raw.Account != nil {
		e.Account = *raw.Account
	}
	if raw.NetLiquidity != nil {
		e.NetLiquidity = *raw.NetLiquidity
	}
	if raw.Cash != nil {
		e.Cash = *raw.Cash
	}
	if raw.Equity != nil {
		e.Equity = *raw.Equity
	}
	if raw.MaintMargin != nil {
		e.MaintMargin = *raw.MaintMargin
	}
	return e
}

// parsePortfolioEvents parses the raw JSON payload of a "pos" frame. The payload
// may be a single position object or an array of position objects.
func parsePortfolioEvents(payload []byte, received time.Time) []PortfolioEvent {
	type posRaw struct {
		Conid         *int64  `json:"conid"`
		Position      *string `json:"pos"`
		AvgCost       *string `json:"avgCost"`
		MarketValue   *string `json:"mktVal"`
		UnrealizedPNL *string `json:"unrealizedPnl"`
	}
	convert := func(raw posRaw) PortfolioEvent {
		e := PortfolioEvent{Received: received}
		if raw.Conid != nil {
			e.Conid = *raw.Conid
		}
		if raw.Position != nil {
			e.Position = *raw.Position
		}
		if raw.AvgCost != nil {
			e.AvgCost = *raw.AvgCost
		}
		if raw.MarketValue != nil {
			e.MarketValue = *raw.MarketValue
		}
		if raw.UnrealizedPNL != nil {
			e.UnrealizedPNL = *raw.UnrealizedPNL
		}
		return e
	}

	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var raws []posRaw
		if err := json.Unmarshal(trimmed, &raws); err != nil {
			return nil
		}
		out := make([]PortfolioEvent, 0, len(raws))
		for _, r := range raws {
			out = append(out, convert(r))
		}
		return out
	}

	var raw posRaw
	if err := json.Unmarshal(trimmed, &raw); err != nil {
		return nil
	}
	return []PortfolioEvent{convert(raw)}
}

// parseMarketDataStatusString parses the string value of field 6509 into a MarketDataStatus.
func parseMarketDataStatusString(s string) *MarketDataStatus {
	if len(s) == 0 {
		return nil
	}
	m := &MarketDataStatus{}
	if len(s) >= 1 {
		m.Availability = s[:1]
	}
	if len(s) >= 2 {
		m.Consolidated = s[1:2]
	}
	if len(s) >= 3 {
		m.Book = s[2:3]
	}
	switch m.Availability {
	case "D":
		m.IsDelayed = true
	case "Z":
		m.IsFrozen = true
	case "Y":
		m.IsFrozen = true
		m.IsDelayed = true
	case "N":
		m.IsNotSubscribed = true
	}
	return m
}
