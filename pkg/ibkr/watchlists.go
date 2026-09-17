// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/shing1211/ibkrapi4go/client"
)

// WatchlistManager exposes watchlist operations. It is safe for concurrent use.
type WatchlistManager struct {
	client *Client
}

// Watchlist holds a watchlist definition.
type Watchlist struct {
	// ID is the watchlist identifier.
	ID string `json:"id"`
	// Name is the watchlist name.
	Name string `json:"name"`
	// Instruments is the list of instruments in the watchlist.
	Instruments []WatchlistInstrument `json:"instruments"`
}

// WatchlistInstrument is an instrument in a watchlist.
type WatchlistInstrument struct {
	// ConID is the contract identifier.
	ConID ConID `json:"conId"`
	// Symbol is the ticker symbol.
	Symbol string `json:"symbol"`
}

// DeleteWatchlist deletes a watchlist by ID.
func (m *WatchlistManager) DeleteWatchlist(ctx context.Context, watchlistID string) error {
	const op = "Watchlist.DeleteWatchlist"
	params := &client.DeleteWatchlistParams{Id: watchlistID}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.DeleteWatchlist(ctx, params)
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// GetSpecificWatchlist returns a specific watchlist by ID.
func (m *WatchlistManager) GetSpecificWatchlist(ctx context.Context, watchlistID string) (*Watchlist, error) {
	const op = "Watchlist.GetSpecificWatchlist"
	params := &client.GetSpecificWatchlistParams{Id: watchlistID}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetSpecificWatchlist(ctx, params)
	})
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	wl := &Watchlist{
		ID:   rawToString(raw, "id"),
		Name: rawToString(raw, "name"),
	}
	if instrumentsRaw, ok := raw["instruments"]; ok {
		var instruments []map[string]json.RawMessage
		if json.Unmarshal(instrumentsRaw, &instruments) == nil {
			for _, item := range instruments {
				wl.Instruments = append(wl.Instruments, WatchlistInstrument{
					ConID:  ConID(jsonNumberToInt(jsonNumber(item["conId"]))),
					Symbol: rawToString(item, "symbol"),
				})
			}
		}
	}
	return wl, nil
}

// PostNewWatchlist creates a new watchlist.
func (m *WatchlistManager) PostNewWatchlist(ctx context.Context, id, name string, conids []ConID) error {
	const op = "Watchlist.PostNewWatchlist"
	rowsJSON := make([]map[string]interface{}, len(conids))
	for i, c := range conids {
		rowsJSON[i] = map[string]interface{}{
			"C": int(c),
		}
	}
	bodyJSON := map[string]interface{}{
		"id":   id,
		"name": name,
		"rows": rowsJSON,
	}
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		return &Error{Op: op, Message: "encode request: " + err.Error(), Err: err}
	}
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.PostNewWatchlistWithBody(ctx,
			"application/json", bytes.NewReader(body))
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// GetAllWatchlists returns all watchlists.
func (m *WatchlistManager) GetAllWatchlists(ctx context.Context) ([]Watchlist, error) {
	const op = "Watchlist.GetAllWatchlists"
	resp, err := m.client.netDo(ctx, op, func() (*http.Response, error) {
		return m.client.generated.GetAllWatchlists(ctx, nil)
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]json.RawMessage
	if err := decodeJSON(resp, op, &raw); err != nil {
		return nil, err
	}
	out := make([]Watchlist, 0, len(raw))
	for _, item := range raw {
		wl := Watchlist{
			ID:   rawToString(item, "id"),
			Name: rawToString(item, "name"),
		}
		if instrumentsRaw, ok := item["instruments"]; ok {
			var instruments []map[string]json.RawMessage
			if json.Unmarshal(instrumentsRaw, &instruments) == nil {
				for _, inst := range instruments {
					wl.Instruments = append(wl.Instruments, WatchlistInstrument{
						ConID:  ConID(jsonNumberToInt(jsonNumber(inst["conId"]))),
						Symbol: rawToString(inst, "symbol"),
					})
				}
			}
		}
		out = append(out, wl)
	}
	return out, nil
}
