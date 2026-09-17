// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package mockgateway

import (
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Request is a snapshot of an inbound gateway request. Params holds the path
// parameters captured by the router (for example "accountId"); it is populated
// on the live request during routing, but Recorder captures requests before
// routing, so recorded snapshots have no Params.
type Request struct {
	// Method is the HTTP method.
	Method string
	// Path is the request path, without the query string.
	Path string
	// Query holds the decoded query parameters.
	Query url.Values
	// Headers holds a deep copy of the request headers.
	Headers http.Header
	// Body holds the request body, up to a bounded limit.
	Body []byte
	// Params maps route placeholders to their captured values.
	Params map[string]string
	// Time is when the request was received.
	Time time.Time
}

// clone returns a deep copy of the request, so recorded snapshots are not
// mutated when the live request is enriched with route parameters.
func (r *Request) clone() *Request {
	c := &Request{
		Method:  r.Method,
		Path:    r.Path,
		Query:   cloneValues(r.Query),
		Headers: r.Headers.Clone(),
		Body:    append([]byte(nil), r.Body...),
		Time:    r.Time,
	}
	if r.Params != nil {
		c.Params = make(map[string]string, len(r.Params))
		for k, v := range r.Params {
			c.Params[k] = v
		}
	}
	return c
}

func cloneValues(v url.Values) url.Values {
	if v == nil {
		return nil
	}
	out := make(url.Values, len(v))
	for k, vs := range v {
		out[k] = append([]string(nil), vs...)
	}
	return out
}

// Recorder is a thread-safe capture of the requests a Server has served.
type Recorder struct {
	mu   sync.Mutex
	reqs []*Request
}

// NewRecorder returns an empty Recorder.
func NewRecorder() *Recorder { return &Recorder{} }

// Record stores a copy of req. It is called by the server before routing.
func (r *Recorder) Record(req *Request) {
	if r == nil || req == nil {
		return
	}
	c := req.clone()
	r.mu.Lock()
	r.reqs = append(r.reqs, c)
	r.mu.Unlock()
}

// Requests returns a copy of every captured request, in arrival order.
func (r *Recorder) Requests() []*Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*Request, len(r.reqs))
	copy(out, r.reqs)
	return out
}

// LastRequest returns the most recently captured request, if any.
func (r *Recorder) LastRequest() (*Request, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.reqs) == 0 {
		return nil, false
	}
	return r.reqs[len(r.reqs)-1], true
}

// Reset discards all captured requests.
func (r *Recorder) Reset() {
	r.mu.Lock()
	r.reqs = nil
	r.mu.Unlock()
}
