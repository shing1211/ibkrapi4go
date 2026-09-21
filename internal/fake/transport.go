// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package fake

import (
	"net/http"
)

// RoundTripper is a test double for http.RoundTripper. By default it returns
// a 200 OK with an empty body. Set RoundTripFn to customize responses.
type RoundTripper struct {
	// RoundTripFn, when set, is called by RoundTrip. It receives the request
	// and returns a response and error.
	RoundTripFn func(req *http.Request) (*http.Response, error)
	// CallCount records the number of times RoundTrip was called.
	CallCount int
	// LastRequest captures the most recent request.
	LastRequest *http.Request
}

// RoundTrip satisfies http.RoundTripper.
func (rt *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.CallCount++
	rt.LastRequest = req
	if rt.RoundTripFn != nil {
		return rt.RoundTripFn(req)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       http.NoBody,
	}, nil
}
