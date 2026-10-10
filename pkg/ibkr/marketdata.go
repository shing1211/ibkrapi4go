// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/shing1211/ibkrapi4go/client"
)

// Snapshot is a point-in-time market-data snapshot for one contract. Field
// values are decimal strings exactly as returned by IBKR (ADR 0008).
type Snapshot struct {
	// ConID is the contract identifier.
	ConID ConID
	// Fields maps requested field ids (e.g. FieldLastPrice) to their values.
	Fields map[Field]string
	// Updated is the UTC timestamp of the data.
	Updated time.Time
	// Status is the market-data availability (field 6509), or nil.
	// Availability values: R=real-time, D=delayed, Z=frozen,
	// Y=frozen-delayed, N=not-subscribed, i=incomplete, v=VDR-exempt.
	Status *MarketDataStatus
}

// MarketDataStatus describes market-data availability for a contract.
// Field 6509: first char = R(real-time)/D(delayed)/Z(frozen)/Y(frozen-delayed)/N(not-subscribed)/i(incomplete)/v(VDR-exempt),
// second char = P(snapshot)/p(consolidated),
// third char = B(book).
type MarketDataStatus struct {
	Availability    string
	Consolidated    string
	Book            string
	IsDelayed       bool
	IsFrozen        bool
	IsNotSubscribed bool
}

// Bar is a single OHLC bar; prices are decimal strings (ADR 0008).
type Bar struct {
	// Time is the bar start time (UTC).
	Time time.Time
	// Open is the open price.
	Open string
	// High is the high price.
	High string
	// Low is the low price.
	Low string
	// Close is the close price.
	Close string
	// Volume is the traded volume.
	Volume string
}

// History is a series of OHLC bars for a contract.
type History struct {
	// Symbol is the instrument symbol.
	Symbol string
	// Bars are the returned bars, oldest first.
	Bars []Bar
}

// HistoryOptions selects a historical bar series.
type HistoryOptions struct {
	// ConID is the contract identifier (required).
	ConID ConID
	// Period is the duration away from the start, e.g. "1d", "1w", "1m" (required).
	Period string
	// Bar is the bar width, e.g. "1min", "1h", "1d" (required).
	Bar string
	// Exchange selects the exchange; empty means SMART.
	Exchange string
	// OutsideRTH includes trades outside regular trading hours.
	OutsideRTH bool
}

// MarketDataManager exposes market-data snapshot and history. It is safe for
// concurrent use.
type MarketDataManager struct {
	client *Client
}

// Snapshot returns market-data snapshots for the given contracts and fields. An
// empty fields slice requests the gateway default field set.
func (m *MarketDataManager) Snapshot(ctx context.Context, conids []ConID, fields []Field) ([]Snapshot, error) {
	const op = "MarketData.Snapshot"
	if len(conids) == 0 {
		return nil, &Error{Op: op, Message: "no conids provided", Err: ErrInvalidRequest}
	}
	params := &client.GetMdSnapshotParams{Conids: joinConIDs(conids)}
	if len(fields) > 0 {
		f := joinFields(fields)
		params.Fields = &f
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetMdSnapshot(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]Snapshot, 0, len(raw))
	for _, item := range raw {
		out = append(out, snapshotFromRaw(item))
	}
	return out, nil
}

// History returns historical OHLC bars for a contract.
func (m *MarketDataManager) History(ctx context.Context, opts HistoryOptions) (*History, error) {
	const op = "MarketData.History"
	params := &client.GetMdHistoryParams{
		Conid:  int64(opts.ConID),
		Period: opts.Period,
		Bar:    opts.Bar,
	}
	if opts.Exchange != "" {
		params.Exchange = &opts.Exchange
	}
	if opts.OutsideRTH {
		params.OutsideRth = &opts.OutsideRTH
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetMdHistory(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw struct {
		Symbol string `json:"symbol"`
		Data   []struct {
			T json.Number `json:"t"`
			O json.Number `json:"o"`
			H json.Number `json:"h"`
			L json.Number `json:"l"`
			C json.Number `json:"c"`
			V json.Number `json:"v"`
		} `json:"data"`
	}
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	h := &History{Symbol: raw.Symbol, Bars: make([]Bar, 0, len(raw.Data))}
	for _, b := range raw.Data {
		h.Bars = append(h.Bars, Bar{
			Time:   time.Unix(jsonNumberToInt64(b.T), 0).UTC(),
			Open:   b.O.String(),
			High:   b.H.String(),
			Low:    b.L.String(),
			Close:  b.C.String(),
			Volume: b.V.String(),
		})
	}
	return h, nil
}

// Unsubscribe closes the backend market-data stream for a contract.
func (m *MarketDataManager) Unsubscribe(ctx context.Context, conid ConID) error {
	const op = "MarketData.Unsubscribe"
	id := int64(conid)
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.CloseMdStream(ctx, client.CloseMdStreamJSONRequestBody(client.CloseMdStreamJSONBody{Conid: &id}))
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// UnsubscribeAll closes all backend market-data streams for the session.
func (m *MarketDataManager) UnsubscribeAll(ctx context.Context) error {
	const op = "MarketData.UnsubscribeAll"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.CloseAllMdStreams(ctx)
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// --- helpers ----------------------------------------------------------------

func snapshotFromRaw(item map[string]json.RawMessage) Snapshot {
	s := Snapshot{Fields: make(map[Field]string, len(item))}
	for key, val := range item {
		switch key {
		case "conid":
			var n json.Number
			if json.Unmarshal(val, &n) == nil {
				s.ConID = ConID(jsonNumberToInt(n))
			}
		case "_updated":
			var n json.Number
			if json.Unmarshal(val, &n) == nil {
				s.Updated = time.Unix(jsonNumberToInt64(n), 0).UTC()
			}
		case "6509":
			s.Status = parseMarketDataStatus(val)
		case "server_id", "6119":
			// metadata, not field values
		default:
			s.Fields[Field(key)] = rawScalarString(val)
		}
	}
	return s
}

// rawScalarString renders a snapshot field value, which may be a string,
// number, or boolean, as a string.
func rawScalarString(val json.RawMessage) string {
	var n json.Number
	if json.Unmarshal(val, &n) == nil {
		return n.String()
	}
	var s string
	if json.Unmarshal(val, &s) == nil {
		return s
	}
	var b bool
	if json.Unmarshal(val, &b) == nil {
		return strconv.FormatBool(b)
	}
	return strings.Trim(string(val), `"`)
}

func parseMarketDataStatus(raw json.RawMessage) *MarketDataStatus {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil || len(s) == 0 {
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

func joinConIDs(conids []ConID) string {
	parts := make([]string, len(conids))
	for i, c := range conids {
		parts[i] = strconv.Itoa(int(c))
	}
	return strings.Join(parts, ",")
}

func joinFields(fields []Field) string {
	parts := make([]string, len(fields))
	for i, f := range fields {
		parts[i] = string(f)
	}
	return strings.Join(parts, ",")
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
