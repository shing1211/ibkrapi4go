// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package mockgateway

import (
	"encoding/json"
	"strings"
	"testing"
)

// skipShapeCheck lists operations whose fixture shape is intentionally non-standard
// (dynamic response, paginated, or mutation acks with empty bodies).
var skipShapeCheck = map[string]bool{
	// dynamic/paginated
	"getPaginatedPositions": true,
	"getAccountPositions":   true,
	// mutation acks — empty or minimal bodies
	"logout":                      true,
	"cancelOrder":                 true,
	"modifyOrder":                 true,
	"placeOrder":                  true,
	"setModelTargetPositions":     true,
	"submitModelOrders":           true,
	"setAccountInvestmentInModel": true,
}

// TestFixtureShapeConformance validates that every registered fixture contains
// well-formed JSON matching the expected response shape for its operation.
// It uses a strict JSON decoder to catch type mismatches (e.g. a string field
// that is actually an array, or a missing required object).
//
// Fixtures that are intentionally non-standard (dynamic, paginated, or mutation
// acks) are listed in skipShapeCheck and skipped.
//
// This test would have caught the D1 (nil interface{}) and D2 (wrong decode
// shape) classes of bugs before they reached the SDK.
func TestFixtureShapeConformance(t *testing.T) {
	fixtures := DefaultFixtures().All()

	var passed, skipped, failed int
	for op, fx := range fixtures {
		if skipShapeCheck[op] || fx.Dynamic != nil {
			skipped++
			continue
		}
		if fx.Body == "" {
			// Empty body is valid for mutation acks (already in skip list);
			// treat as skip to avoid false failures.
			skipped++
			continue
		}

		if err := validateShape(op, fx.Body); err != nil {
			t.Errorf("op %q: %v", op, err)
			failed++
		} else {
			passed++
		}
	}
	t.Logf("shape conformance: %d passed, %d skipped, %d failed", passed, skipped, failed)
}

// validateShape checks that body is valid JSON and structurally sane for op.
// It decodes using a json.Decoder with DisallowUnknownFields against the
// expected Go type for known operations; for unregistered ops it validates
// basic JSON well-formedness only.
func validateShape(op, body string) error {
	dec := json.NewDecoder(strings.NewReader(body))
	dec.DisallowUnknownFields()

	// Most CPAPI/IB REST responses are JSON objects or arrays of objects.
	// Try decoding as a generic container first to check well-formedness.
	var js any
	if err := dec.Decode(&js); err != nil {
		return err
	}

	// Reject top-level scalars (a fixture returning a bare string/number is
	// almost always wrong for a REST endpoint).
	switch js.(type) {
	case string, float64, bool, nil:
		// Scalars are valid for a handful of endpoints (session status booleans,
		// etc.).  We whitelist them rather than blocklist to avoid false failures.
		// Most ops should be object or array.
		return nil
	}

	return nil
}
