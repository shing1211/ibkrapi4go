// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package mockgateway

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

// streamWriteTimeout bounds a single frame write so a stalled peer cannot block
// a broadcast forever.
const streamWriteTimeout = 5 * time.Second

// Tick is one scripted market-data update: the value of a single field for a
// single contract. Ticks are emitted deterministically from a StreamScript or
// pushed explicitly by a caller.
type Tick struct {
	// ConID is the contract the update belongs to.
	ConID int
	// Field is the market-data field code (for example "31" for last price).
	Field string
	// Value is the field value rendered as a string (ADR 0008).
	Value string
}

// StreamLimits bounds the subscriptions a StreamHub will accept. A zero field
// means unlimited.
type StreamLimits struct {
	// MaxConIDsPerRequest is the maximum conids accepted in one subscribe frame.
	MaxConIDsPerRequest int
	// MaxSubscriptions bounds the distinct conids a single connection may
	// subscribe to. Re-subscribing an existing conid and duplicate conids
	// within a frame do not increase the count, matching the client-side
	// distinct accounting in pkg/ibkr.
	MaxSubscriptions int
}

// StreamScript controls the scripted behavior of the WebSocket mock. All fields
// are optional; the zero value emits a single default tick per subscribed conid.
type StreamScript struct {
	// DropFirstConnection closes the first accepted connection immediately after
	// it sends its first subscribe frame, exercising the client's reconnect
	// path.
	DropFirstConnection bool
	// OnSubscribe, when set, returns the ticks to emit for a subscribe frame.
	// It is called synchronously from the connection's read loop, so it must not
	// block. When nil the default script is used.
	OnSubscribe func(conids []int, fields []string) []Tick
	// ErrorOnSubscribe, when it returns a non-empty message, makes the mock send
	// an error frame instead of ticks for that subscribe. This models
	// server-side streaming limits.
	ErrorOnSubscribe func(conids []int, fields []string) string
	// Limits enforces server-side subscription ceilings and emits an error frame
	// when exceeded.
	Limits StreamLimits
	// Hello emits an `sts` connection-status frame on upgrade and an `ntf`
	// notice on subscribe. The pkg/ibkr client ignores these frames; they are
	// provided for protocol fidelity and for callers that inspect raw frames.
	Hello bool
}

// StreamHub coordinates the scripted WebSocket connections of a Server. It is
// safe for concurrent use.
type StreamHub struct {
	mu     sync.Mutex
	conns  map[*streamConn]struct{}
	script StreamScript
	seed   int64

	accepted atomic.Int64
	frames   atomic.Int64
}

// newStreamHub builds a StreamHub from a script (value-copied) and seed.
func newStreamHub(script StreamScript, seed int64) *StreamHub {
	return &StreamHub{
		conns:  make(map[*streamConn]struct{}),
		script: script,
		seed:   seed,
	}
}

// Script returns the active stream script.
func (h *StreamHub) Script() StreamScript {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.script
}

// Subscribers returns the number of currently connected WebSocket clients.
func (h *StreamHub) Subscribers() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.conns)
}

// Push broadcasts a tick to every connection subscribed to its conid and returns
// the number of connections the frame was written to. The emitted frame is
// deterministic for a given tick and server seed.
func (h *StreamHub) Push(t Tick) int {
	frame := t.frame(h.nextUpdated())
	h.mu.Lock()
	conns := make([]*streamConn, 0, len(h.conns))
	for sc := range h.conns {
		conns = append(conns, sc)
	}
	h.mu.Unlock()

	n := 0
	for _, sc := range conns {
		if !sc.wants(t.ConID) {
			continue
		}
		if sc.sendFrame(frame) == nil {
			n++
		}
	}
	return n
}

// Broadcast sends an arbitrary frame value to every connected client and returns
// the number of successful writes. Use the StatusFrame, NotificationFrame,
// UserFrame, and OrderFrame helpers for the `sts`/`ntf`/`usr`/`sor` frame
// families.
func (h *StreamHub) Broadcast(frame any) int {
	h.mu.Lock()
	conns := make([]*streamConn, 0, len(h.conns))
	for sc := range h.conns {
		conns = append(conns, sc)
	}
	h.mu.Unlock()

	n := 0
	for _, sc := range conns {
		if sc.sendFrame(frame) == nil {
			n++
		}
	}
	return n
}

func (h *StreamHub) add(sc *streamConn) {
	sc.ordinal = int(h.accepted.Add(1))
	h.mu.Lock()
	h.conns[sc] = struct{}{}
	h.mu.Unlock()
}

func (h *StreamHub) remove(sc *streamConn) {
	h.mu.Lock()
	delete(h.conns, sc)
	h.mu.Unlock()
}

// nextUpdated returns a monotonic, seed-derived `_updated` stamp. It is
// deterministic for a given seed and emission order, unlike wall-clock time.
func (h *StreamHub) nextUpdated() int64 {
	return h.seed + h.frames.Add(1)
}

// defaultTicks returns one tick per conid using the first requested field (or
// "31") and a fixed value.
func defaultTicks(conids []int, fields []string) []Tick {
	field := "31"
	if len(fields) > 0 && fields[0] != "" {
		field = fields[0]
	}
	out := make([]Tick, 0, len(conids))
	for _, c := range conids {
		out = append(out, Tick{ConID: c, Field: field, Value: "150.25"})
	}
	return out
}

// frame renders a tick as a market-data frame consumable by internal/ws.go.
func (t Tick) frame(updated int64) map[string]any {
	return map[string]any{"conid": t.ConID, t.Field: t.Value, "_updated": updated}
}

// StatusFrame returns an `sts` (connection status) frame.
func StatusFrame(status string) map[string]any {
	return map[string]any{"sts": status, "topic": "sts"}
}

// NotificationFrame returns an `ntf` (notification) frame.
func NotificationFrame(topic string, payload any) map[string]any {
	return map[string]any{"ntf": payload, "topic": topic}
}

// UserFrame returns a `usr` (user message) frame.
func UserFrame(payload any) map[string]any {
	return map[string]any{"usr": payload}
}

// OrderFrame returns a `sor` (order status) frame.
func OrderFrame(payload any) map[string]any {
	return map[string]any{"sor": payload}
}

// streamConn is one accepted WebSocket connection and its subscription state.
type streamConn struct {
	conn    *websocket.Conn
	ordinal int
	writeMu sync.Mutex
	mu      sync.Mutex
	subs    map[int]struct{}
	closed  atomic.Bool
}

// subscribe records conids as wanted by this connection.
func (sc *streamConn) subscribe(conids []int) {
	sc.mu.Lock()
	for _, c := range conids {
		sc.subs[c] = struct{}{}
	}
	sc.mu.Unlock()
}

// unsubscribe removes conids from this connection's wants.
func (sc *streamConn) unsubscribe(conids []int) {
	sc.mu.Lock()
	for _, c := range conids {
		delete(sc.subs, c)
	}
	sc.mu.Unlock()
}

// wants reports whether conid is currently subscribed on this connection.
func (sc *streamConn) wants(conid int) bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	_, ok := sc.subs[conid]
	return ok
}

// projectedSubscriptionCount returns the number of distinct conids that would
// be subscribed after applying conids to this connection's current set.
// Already-subscribed and duplicate conids do not increase the count.
func (sc *streamConn) projectedSubscriptionCount(conids []int) int {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	seen := make(map[int]struct{}, len(sc.subs)+len(conids))
	for c := range sc.subs {
		seen[c] = struct{}{}
	}
	for _, c := range conids {
		seen[c] = struct{}{}
	}
	return len(seen)
}

// sendFrame serializes v as JSON and writes it as a text frame. Writes are
// serialized so the read loop and Push/Broadcast callers never interleave.
func (sc *streamConn) sendFrame(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	sc.writeMu.Lock()
	defer sc.writeMu.Unlock()
	if sc.closed.Load() {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), streamWriteTimeout)
	defer cancel()
	return sc.conn.Write(ctx, websocket.MessageText, b)
}

// close marks the connection closed; the handler closes the socket itself.
func (sc *streamConn) close() { sc.closed.Store(true) }

// serveWS upgrades the request to a WebSocket and runs the scripted frame loop.
// It is called from Server.ServeHTTP for OpOpenWebsocket.
func (s *Server) serveWS(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	sc := &streamConn{conn: c, subs: make(map[int]struct{})}
	s.stream.add(sc)
	defer func() {
		sc.close()
		s.stream.remove(sc)
		_ = c.Close(websocket.StatusNormalClosure, "closed")
	}()

	if s.stream.script.Hello {
		_ = sc.sendFrame(StatusFrame("connected"))
	}

	for {
		_, data, err := c.Read(context.Background())
		if err != nil {
			return
		}
		op, conids, fields := parseStreamFrame(data)
		switch op {
		case streamOpSubscribe:
			if len(conids) == 0 {
				continue
			}
			if s.stream.script.DropFirstConnection && sc.ordinal == 1 {
				return
			}
			if !s.handleSubscribe(sc, conids, fields) {
				return
			}
		case streamOpUnsubscribe:
			sc.unsubscribe(conids)
		}
	}
}

// handleSubscribe applies limits, records the subscription, and emits the
// scripted ticks. It reports whether the connection should stay open.
func (s *Server) handleSubscribe(sc *streamConn, conids []int, fields []string) bool {
	script := s.stream.script

	if msg := limitMessage(script.Limits, sc.projectedSubscriptionCount(conids), len(conids)); msg != "" {
		_ = sc.sendFrame(map[string]any{"error": msg})
		return true
	}
	if script.ErrorOnSubscribe != nil {
		if msg := script.ErrorOnSubscribe(conids, fields); msg != "" {
			_ = sc.sendFrame(map[string]any{"error": msg})
			return true
		}
	}
	sc.subscribe(conids)

	if script.Hello {
		_ = sc.sendFrame(NotificationFrame("smd", map[string]any{"conids": conids}))
	}

	var emitted []Tick
	if script.OnSubscribe != nil {
		emitted = script.OnSubscribe(conids, fields)
	} else {
		emitted = defaultTicks(conids, fields)
	}
	for _, t := range emitted {
		if !sc.wants(t.ConID) {
			continue
		}
		if err := sc.sendFrame(t.frame(s.stream.nextUpdated())); err != nil {
			return false
		}
	}
	return true
}

// limitMessage returns a non-empty error message when the subscribe exceeds the
// configured limits. distinct is the projected number of distinct conids held
// after the frame is applied; adding is the raw conid count of the frame.
func limitMessage(lim StreamLimits, distinct, adding int) string {
	if lim.MaxConIDsPerRequest > 0 && adding > lim.MaxConIDsPerRequest {
		return "streaming limit: conids exceed " + strconv.Itoa(lim.MaxConIDsPerRequest)
	}
	if lim.MaxSubscriptions > 0 && distinct > lim.MaxSubscriptions {
		return "streaming limit: subscriptions exceed " + strconv.Itoa(lim.MaxSubscriptions)
	}
	return ""
}

// streamOp classifies an inbound frame.
type streamOp int

const (
	streamOpIgnore streamOp = iota
	streamOpSubscribe
	streamOpUnsubscribe
)

// parseStreamFrame decodes a subscribe or unsubscribe frame. It accepts the
// JSON protocol used by internal/ws.go:
//
//	{"method":"subscribe","params":{"conids":[265598],"fields":["31"]}}
//
// and the raw text forms used by the CPAPI gateway:
//
//	smd+265598+{"fields":["31"]}
//	umd+265598
func parseStreamFrame(data []byte) (streamOp, []int, []string) {
	var f struct {
		Method string `json:"method"`
		Params struct {
			Conids []int    `json:"conids"`
			Fields []string `json:"fields"`
		} `json:"params"`
	}
	if err := json.Unmarshal(data, &f); err == nil && f.Method != "" {
		switch strings.ToLower(f.Method) {
		case "subscribe":
			return streamOpSubscribe, f.Params.Conids, f.Params.Fields
		case "unsubscribe":
			return streamOpUnsubscribe, f.Params.Conids, nil
		default:
			return streamOpIgnore, nil, nil
		}
	}

	msg := strings.TrimSpace(string(data))
	switch {
	case strings.HasPrefix(msg, "smd+"):
		parts := strings.SplitN(strings.TrimPrefix(msg, "smd+"), "+", 2)
		conid, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return streamOpIgnore, nil, nil
		}
		var fields []string
		if len(parts) == 2 {
			fields = parseStreamFields(parts[1])
		}
		return streamOpSubscribe, []int{conid}, fields
	case strings.HasPrefix(msg, "umd+"):
		return streamOpUnsubscribe, parseStreamConids(strings.TrimPrefix(msg, "umd+")), nil
	default:
		return streamOpIgnore, nil, nil
	}
}

// parseStreamFields decodes a text-frame field list. It accepts a JSON object
// ({"fields":[...]}), a JSON array ([...]), or a comma-separated list.
func parseStreamFields(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var obj struct {
		Fields []string `json:"fields"`
	}
	if json.Unmarshal([]byte(raw), &obj) == nil && len(obj.Fields) > 0 {
		return obj.Fields
	}
	var arr []string
	if json.Unmarshal([]byte(raw), &arr) == nil && len(arr) > 0 {
		return arr
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(strings.Trim(p, `"`)); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// parseStreamConids decodes a comma-separated conid list.
func parseStreamConids(raw string) []int {
	var out []int
	for _, p := range strings.Split(raw, ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
			out = append(out, n)
		}
	}
	return out
}
