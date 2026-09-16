// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// netDo runs a raw generated call, guarding against a closed client and mapping
// non-2xx responses and transport errors to *Error. The returned response body
// is open and owned by the caller, who must close it (decodeJSON does so).
func (c *Client) netDo(ctx context.Context, op string, fn func() (*http.Response, error)) (*http.Response, error) {
	if err := c.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := fn()
	if err != nil {
		return nil, wrapOp(op, err)
	}
	if e := c.errorFrom(resp, op); e != nil {
		resp.Body.Close()
		return nil, e
	}
	return resp, nil
}

// decodeJSON reads and closes resp.Body, decoding JSON with json.Number so
// decimal values are preserved as strings rather than binary floats (ADR 0008).
func decodeJSON(resp *http.Response, op string, v any) error {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &Error{Op: op, Message: "read response: " + err.Error(), Err: err}
	}
	return decodeJSONBytes(body, op, v)
}

// decodeJSONValue decodes data with json.Number for callers that already hold the
// response body bytes.
func decodeJSONBytes(body []byte, op string, v any) error {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(v); err != nil {
		return &Error{Op: op, Message: "decode response: " + err.Error(), Err: err}
	}
	return nil
}

// jsonNumberToInt converts a json.Number to an int, tolerating decimal forms.
func jsonNumberToInt(n json.Number) int {
	if n == "" {
		return 0
	}
	if i, err := n.Int64(); err == nil {
		return int(i)
	}
	if f, err := n.Float64(); err == nil {
		return int(f)
	}
	return 0
}

// rawToString returns the value at key as a decimal/string, tolerating JSON
// strings, numbers, and booleans.
func rawToString(raw map[string]json.RawMessage, key string) string {
	b, ok := raw[key]
	if !ok {
		return ""
	}
	var s string
	if json.Unmarshal(b, &s) == nil {
		return s
	}
	var n json.Number
	if json.Unmarshal(b, &n) == nil {
		return n.String()
	}
	var v any
	if json.Unmarshal(b, &v) == nil {
		return fmt.Sprint(v)
	}
	return ""
}

// rawToBool returns the boolean at key, or false if absent/unparseable.
func rawToBool(raw map[string]json.RawMessage, key string) bool {
	b, ok := raw[key]
	if !ok {
		return false
	}
	var v bool
	_ = json.Unmarshal(b, &v)
	return v
}

// rawToStringSlice returns the string array at key, or nil.
func rawToStringSlice(raw map[string]json.RawMessage, key string) []string {
	b, ok := raw[key]
	if !ok {
		return nil
	}
	var v []string
	if json.Unmarshal(b, &v) == nil {
		return v
	}
	var nums []json.Number
	if json.Unmarshal(b, &nums) == nil {
		out := make([]string, len(nums))
		for i, n := range nums {
			out[i] = n.String()
		}
		return out
	}
	return nil
}
