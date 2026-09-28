// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package mockgateway

import (
	"reflect"
	"testing"
)

// The streaming layer is parsed from the wire formats IBKR actually sends, in
// several shapes. These are pure functions, and they are where a subscription
// silently stops receiving the fields a caller asked for: a parser that accepts
// something it should reject, or drops a field, produces a stream that looks
// healthy and is wrong.

func TestParseStreamFields(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want []string
	}{
		{"empty", "", nil},
		{"whitespace only", "   ", nil},
		{"comma separated", "last,bid,ask", []string{"last", "bid", "ask"}},
		{"comma separated with spaces", "last, bid , ask", []string{"last", "bid", "ask"}},
		{"trailing comma", "last,bid,", []string{"last", "bid"}},
		{"leading comma", ",last", []string{"last"}},
		{"quoted entries", `"last","bid"`, []string{"last", "bid"}},
		{"json array", `["last","bid"]`, []string{"last", "bid"}},
		{"json object", `{"fields":["last","bid"]}`, []string{"last", "bid"}},
		{"json object with one field", `{"fields":["last"]}`, []string{"last"}},
		{"numeric field ids", "31,84,86", []string{"31", "84", "86"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseStreamFields(tc.raw); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("parseStreamFields(%q) = %#v; want %#v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestParseStreamConids(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want []int
	}{
		{"empty", "", nil},
		{"single", "265598", []int{265598}},
		{"several", "1,2,3", []int{1, 2, 3}},
		{"with spaces", "1, 2 , 3", []int{1, 2, 3}},
		{"non-numeric entries are skipped", "1,abc,3", []int{1, 3}},
		{"empty entries are skipped", "1,,2", []int{1, 2}},
		{"all invalid", "abc,def", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseStreamConids(tc.raw); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("parseStreamConids(%q) = %#v; want %#v", tc.raw, got, tc.want)
			}
		})
	}
}

// TestParseStreamFrame covers both wire shapes the mock accepts: the JSON control
// frame the SDK sends, and the legacy text protocol. A frame the parser should
// ignore must be ignored - if an unknown method were treated as a subscribe, a
// client would get frames it never asked for.
func TestParseStreamFrame(t *testing.T) {
	for _, tc := range []struct {
		name       string
		data       string
		wantOp     streamOp
		wantConids []int
		wantFields []string
	}{
		{
			name:   "json subscribe",
			data:   `{"method":"subscribe","params":{"conids":[265598],"fields":["last","bid"]}}`,
			wantOp: streamOpSubscribe, wantConids: []int{265598}, wantFields: []string{"last", "bid"},
		},
		{
			name:   "json subscribe without fields",
			data:   `{"method":"subscribe","params":{"conids":[1]}}`,
			wantOp: streamOpSubscribe, wantConids: []int{1},
		},
		{
			name:   "json method is case-insensitive",
			data:   `{"method":"SUBSCRIBE","params":{"conids":[7],"fields":["last"]}}`,
			wantOp: streamOpSubscribe, wantConids: []int{7}, wantFields: []string{"last"},
		},
		{
			name:   "json unsubscribe",
			data:   `{"method":"unsubscribe","params":{"conids":[1,2]}}`,
			wantOp: streamOpUnsubscribe, wantConids: []int{1, 2},
		},
		{
			name:   "json account carries fields but no conids",
			data:   `{"method":"account","params":{"fields":["1","2"]}}`,
			wantOp: streamOpAccount, wantFields: []string{"1", "2"},
		},
		{
			name:   "json portfolio carries fields but no conids",
			data:   `{"method":"portfolio","params":{"fields":["3"]}}`,
			wantOp: streamOpPortfolio, wantFields: []string{"3"},
		},
		{
			name:   "unknown json method is ignored",
			data:   `{"method":"somethingelse","params":{"conids":[1]}}`,
			wantOp: streamOpIgnore,
		},
		{
			name:   "json without a method is ignored",
			data:   `{"params":{"conids":[1]}}`,
			wantOp: streamOpIgnore,
		},
		{
			name:   "malformed json is ignored",
			data:   `{"method":`,
			wantOp: streamOpIgnore,
		},
		{
			name:   "legacy smd+ subscribe",
			data:   "smd+265598+last,bid",
			wantOp: streamOpSubscribe, wantConids: []int{265598}, wantFields: []string{"last", "bid"},
		},
		{
			name:   "legacy smd+ without a field list",
			data:   "smd+265598",
			wantOp: streamOpSubscribe, wantConids: []int{265598},
		},
		{
			name:   "legacy smd+ with a non-numeric conid is ignored",
			data:   "smd+notanumber+last",
			wantOp: streamOpIgnore,
		},
		{
			name:   "legacy umd+ unsubscribe",
			data:   "umd+1,2,3",
			wantOp: streamOpUnsubscribe, wantConids: []int{1, 2, 3},
		},
		{
			name:   "unrecognised text is ignored",
			data:   "hello",
			wantOp: streamOpIgnore,
		},
		{
			name:   "empty text is ignored",
			data:   "   ",
			wantOp: streamOpIgnore,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			op, conids, fields := parseStreamFrame([]byte(tc.data))
			if op != tc.wantOp {
				t.Errorf("op = %v; want %v", op, tc.wantOp)
			}
			if !reflect.DeepEqual(conids, tc.wantConids) {
				t.Errorf("conids = %#v; want %#v", conids, tc.wantConids)
			}
			if !reflect.DeepEqual(fields, tc.wantFields) {
				t.Errorf("fields = %#v; want %#v", fields, tc.wantFields)
			}
		})
	}
}
