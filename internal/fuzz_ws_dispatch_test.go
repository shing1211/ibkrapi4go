// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"
)

func FuzzWSDispatch(f *testing.F) {
	f.Add([]byte(`{"conid":265598,"31":"150.25","_updated":1}`))
	f.Add([]byte(`invalid json`))
	f.Add([]byte(`{"conid":265598}`))
	f.Add([]byte(`{"sts":"connected","topic":"sts"}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}
		wsScalarString(data)
		for _, key := range []string{"conid", "_updated", "server_id", "6119", "6509", "topic", "method", "id", "random"} {
			wsReservedField(key)
		}
	})
}
