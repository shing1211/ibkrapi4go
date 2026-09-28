// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"strings"
	"testing"
)

const moneyDoc = "docs/design/07-money-and-numbers.md"

// TestMoneyNumberChecksAreLoadBearing mutates one fact per case and requires the
// message that names it. A check that has never been observed failing is not a
// check, so these cases exist to prove checkMoneyNumberFields actually bites.
//
// The fact under test is a documented claim about generated code, so each case
// breaks one side of it: the document, or the generated type it names.
func TestMoneyNumberChecksAreLoadBearing(t *testing.T) {
	cases := []struct {
		name  string
		edits []edit
		want  string
	}{
		{
			// The doc drops a field it used to name, so the checker must notice
			// the named set no longer covers what the prose implies. Removing
			// the only mention of `quantity` leaves the clause intact, which the
			// checker accepts; instead retyping the field in code is the
			// regression that matters, and is covered below.
			name: "code/divamount-back-to-float",
			edits: []edit{replace("client/client.gen.go",
				"DivAmount          *json.Number `json:\"divAmount,omitempty\"`",
				"DivAmount          *float32 `json:\"divAmount,omitempty\"`")},
			want: "documents TaxVoucherDTO.divAmount as a json.Number money field",
		},
		{
			name: "code/withheld-back-to-float64",
			edits: []edit{replace("client/client.gen.go",
				"WithHeldAmount     *json.Number `json:\"withHeldAmount,omitempty\"`",
				"WithHeldAmount     *float64 `json:\"withHeldAmount,omitempty\"`")},
			want: "documents TaxVoucherDTO.withHeldAmount as a json.Number money field",
		},
		{
			name: "code/fee-back-to-float",
			edits: []edit{replace("client/client.gen.go",
				"Fee                *json.Number `json:\"fee,omitempty\"`",
				"Fee                *float32 `json:\"fee,omitempty\"`")},
			want: "documents TaxVoucherDTO.fee as a json.Number money field",
		},
		{
			// Removing the sentence the checker parses must fail loudly, not
			// silently pass with an empty field set.
			name: "doc/sentence-removed",
			edits: []edit{replace(moneyDoc,
				"are the second case",
				"are handled some other way")},
			want: "no sentence naming the json.Number money fields was found",
		},
		{
			// Stripping the backticks leaves the clause with no parseable field
			// names; an empty set must not be treated as "nothing to check".
			name: "doc/field-names-unparseable",
			edits: []edit{replace(moneyDoc,
				"`divAmount`, `withHeldAmount`, `fee` and `quantity`",
				"the amount fields")},
			want: "lists no backticked field names",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, e := range tc.edits {
				mutate(t, e.rel, e.old, e.repl)
			}
			err := checkMoneyNumberFields(scratchRoot)
			if err == nil {
				t.Fatalf("checkMoneyNumberFields passed; expected a failure containing %q", tc.want)
			}
			if !strings.Contains(stripLineNumbers(err.Error()), stripLineNumbers(tc.want)) {
				t.Fatalf("checkMoneyNumberFields failed for the wrong reason.\nwant substring: %q\ngot: %v",
					tc.want, err)
			}
		})
	}
}

// TestMoneyNumberChecksPassOnUnmutatedTree is the control for every red case
// above: without it, a checker that always failed would satisfy them all.
func TestMoneyNumberChecksPassOnUnmutatedTree(t *testing.T) {
	if err := checkMoneyNumberFields(scratchRoot); err != nil {
		t.Fatalf("checkMoneyNumberFields on the unmutated tree: %v", err)
	}
}
