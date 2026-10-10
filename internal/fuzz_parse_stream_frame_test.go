// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"encoding/json"
	"testing"
)

func FuzzParseStreamFrame(f *testing.F) {
	f.Add([]byte(`{"conid":265598,"31":"150.25","_updated":1}`))
	f.Add([]byte(`{"sts":"connected","topic":"sts"}`))
	f.Add([]byte(`{"ntf":"test","topic":"ntf"}`))
	f.Add([]byte(`{"sor":"order","topic":"sor"}`))
	f.Add([]byte(`{"usr":"user","topic":"usr"}`))
	f.Add([]byte(`{"conid":265598,"31":"150.25","_updated":1,"server_id":"abc"}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(data, &m); err != nil {
			t.Skip()
		}
		parseSystemFrame(m)
	})
}
