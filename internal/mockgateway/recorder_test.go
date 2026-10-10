// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package mockgateway

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
)

// The recorder is how a test asserts what was actually sent. A shallow copy would
// let a recorded snapshot mutate under the test's feet - the snapshot is taken
// before routing, and routing then enriches the live request with path parameters
// and decodes a body. An assertion that passes or fails depending on that timing is
// worse than no assertion, so the copy is pinned here.

func sampleRequest() *Request {
	return &Request{
		Method:  http.MethodPost,
		Path:    "/v1/api/iserver/account/orders",
		Query:   url.Values{"fields": {"31,84"}},
		Headers: http.Header{"Content-Type": {"application/json"}},
		Body:    []byte(`{"orders":[{"conid":1}]}`),
		Time:    time.Unix(1700000000, 0).UTC(),
	}
}

func TestRecorder_RecordTakesADeepCopy(t *testing.T) {
	r := NewRecorder()
	live := sampleRequest()
	r.Record(live)

	// Mutate every field the snapshot could share, the way routing and decoding do.
	live.Path = "/v1/api/iserver/account/CHANGED/orders"
	live.Query.Set("fields", "MUTATED")
	live.Query.Add("extra", "1")
	live.Headers.Set("Content-Type", "text/plain")
	live.Body[0] = 'X'
	live.Params = map[string]string{"accountId": "U9999999"}

	got, ok := r.LastRequest()
	if !ok {
		t.Fatal("LastRequest() reported nothing recorded")
	}
	if got.Path != "/v1/api/iserver/account/orders" {
		t.Errorf("Path = %q; the snapshot shared the live string", got.Path)
	}
	if q := got.Query.Get("fields"); q != "31,84" {
		t.Errorf("Query[fields] = %q; want 31,84 - the snapshot shared the live map", q)
	}
	if _, has := got.Query["extra"]; has {
		t.Error("the snapshot picked up a key added after recording")
	}
	if ct := got.Headers.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q; the snapshot shared the live header map", ct)
	}
	if string(got.Body) != `{"orders":[{"conid":1}]}` {
		t.Errorf("Body = %s; the snapshot shared the live byte slice", got.Body)
	}
	if got.Params != nil {
		t.Errorf("Params = %v; want nil - the snapshot captured routing parameters", got.Params)
	}
}

func TestRecorder_RecordCopiesExistingParams(t *testing.T) {
	r := NewRecorder()
	live := sampleRequest()
	live.Params = map[string]string{"accountId": "U1234567"}
	r.Record(live)

	live.Params["accountId"] = "CHANGED"

	got, _ := r.LastRequest()
	if got.Params["accountId"] != "U1234567" {
		t.Errorf("Params[accountId] = %q; the snapshot shared the live map",
			got.Params["accountId"])
	}
}

func TestRecorder_RecordIsSafeOnNil(t *testing.T) {
	var nilRec *Recorder
	nilRec.Record(sampleRequest()) // must not panic on a nil receiver

	r := NewRecorder()
	r.Record(nil) // must not panic on a nil request

	if got, ok := r.LastRequest(); ok {
		t.Errorf("LastRequest() = %+v; want nothing recorded", got)
	}
}

func TestRecorder_RequestsReturnsACopyOfTheSlice(t *testing.T) {
	r := NewRecorder()
	r.Record(sampleRequest())
	r.Record(sampleRequest())

	all := r.Requests()
	if len(all) != 2 {
		t.Fatalf("Requests() returned %d; want 2", len(all))
	}
	// Overwriting the returned slice must not corrupt the recorder's own record.
	all[0] = nil
	if again := r.Requests(); again[0] == nil {
		t.Error("Requests() handed out its backing slice; the caller mutated the recorder")
	}
}

func TestRecorder_LastRequestOnEmpty(t *testing.T) {
	r := NewRecorder()
	got, ok := r.LastRequest()
	if ok {
		t.Errorf("LastRequest() on an empty recorder = %+v, true; want nil, false", got)
	}
	if len(r.Requests()) != 0 {
		t.Errorf("Requests() on an empty recorder returned %d entries; want 0",
			len(r.Requests()))
	}
}

func TestRecorder_LastRequestIsTheMostRecent(t *testing.T) {
	r := NewRecorder()
	first := sampleRequest()
	first.Path = "/first"
	last := sampleRequest()
	last.Path = "/last"

	r.Record(first)
	r.Record(last)

	got, ok := r.LastRequest()
	if !ok {
		t.Fatal("LastRequest() reported nothing")
	}
	if got.Path != "/last" {
		t.Errorf("LastRequest() = %q; want /last", got.Path)
	}
}

func TestRecorder_ResetDiscardsEverything(t *testing.T) {
	r := NewRecorder()
	r.Record(sampleRequest())
	r.Record(sampleRequest())
	r.Reset()

	if got, ok := r.LastRequest(); ok {
		t.Errorf("after Reset, LastRequest() = %+v; want nothing", got)
	}
	if n := len(r.Requests()); n != 0 {
		t.Errorf("after Reset, Requests() returned %d entries; want 0", n)
	}

	// A reset recorder must still work.
	r.Record(sampleRequest())
	if _, ok := r.LastRequest(); !ok {
		t.Error("the recorder is unusable after Reset")
	}
}

// TestRecorder_ConcurrentRecordIsSafe is meaningful under -race only, but it is the
// property that lets a test fire concurrent requests at the gateway and then make
// assertions about what was sent.
func TestRecorder_ConcurrentRecordIsSafe(t *testing.T) {
	r := NewRecorder()
	const n = 64

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			r.Record(sampleRequest())
		}()
	}
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_ = r.Requests()
			_, _ = r.LastRequest()
		}()
	}
	wg.Wait()

	if got := len(r.Requests()); got != n {
		t.Errorf("recorded %d requests; want %d", got, n)
	}
}

// TestServer_RecordsRequestsWithoutParams pins the documented pre-routing
// contract at the server level: the snapshot is taken before the router captures
// path parameters, so a recorded request never has Params. Tests rely on this to
// assert on what arrived rather than on what the router later derived from it.
func TestServer_RecordsRequestsWithoutParams(t *testing.T) {
	s := New()
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/v1/api/iserver/currency/pairs", nil))

	recorded := s.Recorder().Requests()
	if len(recorded) != 1 {
		t.Fatalf("recorded %d requests; want 1", len(recorded))
	}
	got := recorded[0]
	if got.Params != nil {
		t.Errorf("recorded Params = %v; want nil - recording happens before routing", got.Params)
	}
	if got.Method != http.MethodGet {
		t.Errorf("Method = %q; want GET", got.Method)
	}
	if got.Path != "/v1/api/iserver/currency/pairs" {
		t.Errorf("Path = %q; want the request path without a query string", got.Path)
	}
}
