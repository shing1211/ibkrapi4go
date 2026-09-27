// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package mockgateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shing1211/ibkrapi4go/client"
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

// statusRe matches the generated wrapper field names that carry a decoded
// body: "JSON200", "Textplain200", "ApplicationproblemJSON400",
// "ApplicationjsonCharsetUtf8503" and so on. oapi-codegen always appends the
// three-digit HTTP status, so anchoring on the trailing digits is exact.
var statusRe = regexp.MustCompile(`^(\D.*?)(\d{3})$`)

// bodySuffixes are the request-body variants oapi-codegen appends to a method
// name; the response wrapper type is unaffected by them, so they are stripped
// before deriving the opId.
var bodySuffixes = []string{"WithBody", "WithTextBody", "WithFormdataBody", "WithMultipartBody"}

var (
	jsonUnmarshalerType = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
	rawMessageType      = reflect.TypeOf(json.RawMessage(nil))
	anyType             = reflect.TypeOf((*any)(nil)).Elem()
	// timeTimeType decodes an RFC 3339 scalar. Its UnmarshalJSON takes over, but
	// there is no key set to reject, so it is not a strictness gap.
	timeTimeType = reflect.TypeOf(time.Time{})
)

// opResponseTypes maps every opId oapi-codegen generated to the wrapper struct
// that holds its decoded response bodies (client.<Op>Response). The map is
// derived by reflecting over the generated client interface rather than by
// scanning JSON tags: a codebase-wide tag scan cannot tell which type an
// operation actually decodes into, and `executedAt` is a valid tag on some
// unrelated operation.
var opResponseTypes = sync.OnceValue(func() map[string]reflect.Type {
	iface := reflect.TypeOf((*client.ClientWithResponsesInterface)(nil)).Elem()
	out := make(map[string]reflect.Type, iface.NumMethod())
	for i := range iface.NumMethod() {
		m := iface.Method(i)
		if !strings.HasSuffix(m.Name, "WithResponse") {
			continue
		}
		op := opIDFromMethod(m.Name)
		if op == "" || m.Type.NumOut() == 0 {
			continue
		}
		resp := m.Type.Out(0)
		if resp.Kind() != reflect.Pointer || resp.Elem().Kind() != reflect.Struct {
			continue
		}
		if prev, ok := out[op]; ok && prev != resp.Elem() {
			continue
		}
		out[op] = resp.Elem()
	}
	return out
})

// TestFixtureShapeConformance validates that every registered fixture contains
// well-formed JSON whose keys match the response type the SDK decodes for that
// operation. Each fixture body is decoded into the generated success body for
// its operation and HTTP status with DisallowUnknownFields, so a fixture that
// invents a key — or renames one the generated type does not carry — fails
// here instead of being silently dropped by the non-strict production decode.
//
// The key check only applies where the generated type is what production
// actually reads. For operations whose pkg/ibkr wrapper unmarshals resp.Body
// into its own struct, the generated type is irrelevant to production and
// comparing the fixture against it is invalid; those are counted and listed
// with the file:line of the decode that justifies them. See
// TestProductionDecodePathsResolve and decode_path_test.go for how that is
// derived from call sites.
//
// Fixtures that are intentionally non-standard (dynamic, paginated, or mutation
// acks) are listed in skipShapeCheck and skipped.
//
// Coverage gaps are reported, never silent: operations with no discoverable
// response type, and those whose type cannot reject an unknown field, are
// counted and listed.
//
// This test would have caught the executedAt (getRequestsStatus) and
// accessToken (createSsoSessions) key-naming defects before they reached the
// SDK, where both decoded to a permanently zero value.
func TestFixtureShapeConformance(t *testing.T) {
	fixtures := DefaultFixtures().All()

	// Buckets for ops the strict check cannot or does not apply to. Each is
	// reported with its op list so coverage is read, not assumed.
	var (
		noResponseType  []string
		noSuccessBody   []string
		notEnforcing    []string
		reduced         []string
		scalarBodies    []string
		unionBodies     []string
		strictBodies    []string
		bypassed        []string
		unknownPath     []string
		bodyUnused      []string
		generatedPath   []string
		unmappedDetails = map[string]string{}
	)

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

		// The production decode path is recorded for every op, whatever the
		// outcome, so each failure below states whether the generated type is
		// what production reads for it.
		switch ev := productionDecodePaths().ops[op]; ev.path {
		case pathGenerated:
			generatedPath = append(generatedPath, op)
		case pathBypassed:
			// The bucket itself is filled from validateShape's own reason, which
			// names the decode that justifies the exemption.
			unmappedDetails[op] = fmt.Sprintf("%s decodes the raw body at %s (%s)", ev.path, ev.at, ev.via)
		case pathBodyUnused:
			bodyUnused = append(bodyUnused, op)
			unmappedDetails[op] = fmt.Sprintf("production calls %s but reads no body, so the key check is advisory there", ev.via)
		case pathUnknown:
			unknownPath = append(unknownPath, op)
			unmappedDetails[op] = "no method in pkg/ibkr calls the generated operation, so nothing establishes that the generated type is irrelevant; the key check still runs"
		}

		res := validateShape(op, fx.Status, fx.Body)
		switch res.class {
		case shapeFailed:
			if ev := productionDecodePaths().ops[op]; ev.path != pathGenerated {
				t.Errorf("op %q: %v [%s]", op, res.err, ev.path)
			} else {
				t.Errorf("op %q: %v [production reads the generated type at %s (%s)]", op, res.err, ev.at, ev.via)
			}
			failed++
		case shapeScalar:
			// Top-level scalars are whitelisted (see validateShape); a fixture
			// that is one is outside the key check by construction.
			scalarBodies = append(scalarBodies, op)
			passed++
		case shapeNoResponseType:
			noResponseType = append(noResponseType, op)
			unmappedDetails[op] = res.reason
		case shapeNoSuccessBody:
			noSuccessBody = append(noSuccessBody, op)
			unmappedDetails[op] = res.reason
		case shapeNotEnforcing:
			notEnforcing = append(notEnforcing, op)
			unmappedDetails[op] = res.reason
		case shapeUnion:
			unionBodies = append(unionBodies, op)
			passed++
		case shapeStrict:
			strictBodies = append(strictBodies, op)
			passed++
		case shapeStrictReduced:
			reduced = append(reduced, op)
			unmappedDetails[op] = res.reason
			passed++
		case shapeBypassedGenerated:
			bypassed = append(bypassed, op)
			unmappedDetails[op] = res.reason
			passed++
		}
	}

	// Coverage report. Every fixture is in exactly one bucket above; the
	// un-enforced ones are listed so a future change that shrinks strict
	// coverage is visible in the test log.
	t.Logf("shape conformance: %d passed, %d skipped, %d failed (of %d fixtures)",
		passed, skipped, failed, len(fixtures))
	t.Logf("  key-checked against one response type: %d %v", len(strictBodies), sorted(strictBodies))
	t.Logf("  key-checked per key across a oneOf union: %d %v", len(unionBodies), sorted(unionBodies))
	t.Logf("  top-level scalar (whitelisted, not key-checked): %d %v", len(scalarBodies), sorted(scalarBodies))
	t.Logf("  production reads the generated type: %d %v", len(generatedPath), sorted(generatedPath))
	report := func(label string, ops []string) {
		if len(ops) == 0 {
			return
		}
		t.Logf("  NOT FULLY KEY-CHECKED — %s: %d", label, len(ops))
		for _, op := range sorted(ops) {
			t.Logf("      %-40s %s", op, unmappedDetails[op])
		}
	}
	report("production decodes the raw body, not the generated type", bypassed)
	report("no pkg/ibkr call site, decode path unknown", unknownPath)
	report("production reads no body, key check is advisory", bodyUnused)
	report("no generated response type for the op", noResponseType)
	report("generated type declares no success body for the fixture status", noSuccessBody)
	report("generated type cannot reject an unknown field", notEnforcing)
	report("outer keys checked, a nested position is not", reduced)
}

// shapeClass is the outcome of one fixture check.
type shapeClass int

const (
	shapeFailed shapeClass = iota
	shapeScalar
	shapeNoResponseType
	shapeNoSuccessBody
	shapeNotEnforcing
	shapeUnion
	shapeStrict
	shapeStrictReduced
	shapeBypassedGenerated
)

type shapeResult struct {
	class  shapeClass
	err    error
	reason string
}

func (c shapeClass) String() string {
	switch c {
	case shapeFailed:
		return "shapeFailed"
	case shapeScalar:
		return "shapeScalar"
	case shapeNoResponseType:
		return "shapeNoResponseType"
	case shapeNoSuccessBody:
		return "shapeNoSuccessBody"
	case shapeNotEnforcing:
		return "shapeNotEnforcing"
	case shapeUnion:
		return "shapeUnion"
	case shapeStrict:
		return "shapeStrict"
	case shapeStrictReduced:
		return "shapeStrictReduced"
	case shapeBypassedGenerated:
		return "shapeBypassedGenerated"
	}
	return "shapeClass(?)"
}

// validateShape checks that body is valid JSON and that its keys match the
// response type the SDK decodes for op at the given HTTP status (zero means
// 200, matching Fixture.Status).
//
// For a plain success body the body is decoded with DisallowUnknownFields
// against the generated type. For a oneOf union the check runs per key against
// the union of the variants' fields — see checkUnion for why — and for a
// top-level scalar it keeps the historical whitelist. Operations with no
// discoverable response type are reported (shapeNoResponseType) rather than
// silently accepted.
func validateShape(op string, status int, body string) shapeResult {
	if status == 0 {
		status = 200
	}

	// Well-formedness first, preserved from the original check.
	var js any
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&js); err != nil {
		return shapeResult{class: shapeFailed, err: err}
	}

	// Top-level scalars are whitelisted rather than blocklisted, as before:
	// a handful of endpoints legitimately answer with a bare string. A fixture
	// reaching this branch is outside the key check, and is counted as such by
	// the caller.
	switch js.(type) {
	case string, float64, bool, nil:
		return shapeResult{class: shapeScalar}
	}

	wrapper, ok := opResponseTypes()[op]
	if !ok {
		return shapeResult{
			class:  shapeNoResponseType,
			reason: "opId has no method on client.ClientWithResponsesInterface, so no generated response wrapper exists",
		}
	}

	// An operation whose generated type production never decodes cannot be
	// checked against that type. pkg/ibkr calls the bare generated method, gets
	// the *http.Response, and unmarshals the body into its own struct, so a key
	// the generated type does not declare is exactly what production expects to
	// see. The classification comes from those call sites, not from the opId.
	if ev := productionDecodePaths().ops[op]; ev.path == pathBypassed {
		return shapeResult{
			class: shapeBypassedGenerated,
			reason: fmt.Sprintf("production decodes the raw body itself at %s (%s), so the generated type is not what it reads: %s",
				ev.at, ev.via, wrapper.Name()),
		}
	}

	success, ok := successBodyType(wrapper, status)
	if !ok {
		return shapeResult{
			class:  shapeNoSuccessBody,
			reason: fmt.Sprintf("client.%s declares no 2xx response body for HTTP %d", wrapper.Name(), status),
		}
	}

	if variants := unionMembers(success); len(variants) > 0 {
		if hasCatchAllMember(variants) {
			return shapeResult{
				class:  shapeNotEnforcing,
				reason: fmt.Sprintf("%s is a oneOf with an interface{} member, so the spec accepts any key", success),
			}
		}
		if err := checkUnion(body, variants); err != nil {
			return shapeResult{class: shapeFailed, err: err}
		}
		if gaps := nestedGaps(success, map[reflect.Type]bool{}); len(gaps) > 0 {
			return shapeResult{
				class:  shapeStrictReduced,
				reason: fmt.Sprintf("%s: union keys checked, but %s", success, strings.Join(gaps, "; ")),
			}
		}
		return shapeResult{class: shapeUnion}
	}

	// The root decoder must be one that can reject a key. A root that swallows
	// its body (interface{}, a map, a hand-written UnmarshalJSON) cannot be
	// checked at all and is reported rather than counted as a pass.
	if reason, ok := swallowsKeys(success); ok {
		return shapeResult{
			class:  shapeNotEnforcing,
			reason: fmt.Sprintf("%s cannot reject an unknown field (%s)", success, reason),
		}
	}
	if err := strictDecode(body, success); err != nil {
		return shapeResult{class: shapeFailed, err: err}
	}
	// A nested position that swallows its body does not stop the outer decode
	// from rejecting the outer keys, so this still counts as checked — but the
	// blind spot is recorded rather than forgotten.
	if gaps := nestedGaps(success, map[reflect.Type]bool{}); len(gaps) > 0 {
		return shapeResult{
			class:  shapeStrictReduced,
			reason: fmt.Sprintf("%s: outer keys checked, but %s", success, strings.Join(gaps, "; ")),
		}
	}
	return shapeResult{class: shapeStrict}
}

// successBodyType returns the generated Go type the SDK decodes into for a 2xx
// response of the given status. It reads only the wrapper's own fields, so the
// answer is per operation — never a codebase-wide tag guess.
func successBodyType(wrapper reflect.Type, status int) (reflect.Type, bool) {
	for i := range wrapper.NumField() {
		f := wrapper.Field(i)
		m := statusRe.FindStringSubmatch(f.Name)
		if m == nil || m[2] != fmt.Sprint(status) {
			continue
		}
		if status < 200 || status > 299 {
			continue
		}
		bt := f.Type
		if bt.Kind() == reflect.Pointer {
			bt = bt.Elem()
		}
		return bt, true
	}
	return nil, false
}

// unionMembers reports the oneOf variants of a generated union type. oapi-codegen
// emits a struct with a single unexported json.RawMessage field and one
// As<Variant>() method per member; there is no other marker.
func unionMembers(t reflect.Type) []reflect.Type {
	if t.Kind() != reflect.Struct {
		return nil
	}
	var out []reflect.Type
	for i := range t.NumMethod() {
		m := t.Method(i)
		if !strings.HasPrefix(m.Name, "As") || m.Type.NumIn() != 1 || m.Type.NumOut() != 2 {
			continue
		}
		out = append(out, m.Type.Out(0))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
}

// hasCatchAllMember reports whether any union member is interface{}, which makes
// the union's key set unbounded.
func hasCatchAllMember(variants []reflect.Type) bool {
	for _, v := range variants {
		if v == anyType {
			return true
		}
	}
	return false
}

// checkUnion validates a body against the variants of a oneOf union.
//
// The variants are mutually exclusive and, for getRequestsStatus, have
// conflicting field types (StatusResponse.requestId is an int64,
// AmRequestStatusResponse.requestId is a string), so no single strict decode
// can describe the wire format — and the SDK itself does not decode strictly:
// it calls As<Variant>() on the union and takes whichever variant yields the
// fields it needs. A body that mixes a key from each variant is therefore
// legitimate here, and rejecting it would be a false failure.
//
// The check is per key instead: every key in the body must be declared by at
// least one variant, and its value must strictly decode into that variant's
// field for the declared type. A key no variant declares — the executedAt
// defect — fails; a key whose type fits no variant fails too.
func checkUnion(body string, variants []reflect.Type) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &obj); err != nil {
		// A non-object body for a union of objects: fall back to "strict decode
		// against each variant, accept any".
		var firstErr error
		for _, v := range variants {
			if err := strictDecode(body, v); err == nil {
				return nil
			} else if firstErr == nil {
				firstErr = err
			}
		}
		return fmt.Errorf("body is not an object; no oneOf member decodes it: %w", firstErr)
	}

	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if err := matchUnionKey(k, obj[k], variants); err != nil {
			return err
		}
	}
	return nil
}

// matchUnionKey checks one key of a union body against every variant.
func matchUnionKey(key string, raw json.RawMessage, variants []reflect.Type) error {
	names := make([]string, 0, len(variants))
	declared := 0
	var firstErr error
	for _, v := range variants {
		names = append(names, v.String())
		ft, ok := fieldByJSONTag(v, key)
		if !ok {
			continue
		}
		declared++
		err := checkValue(raw, ft, map[reflect.Type]bool{})
		if err == nil {
			return nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	if declared == 0 {
		return fmt.Errorf("unknown field %q: no oneOf member of the response declares it (members: %s)",
			key, strings.Join(names, ", "))
	}
	return fmt.Errorf("field %q is declared by the oneOf but its value matches no member's type: %w",
		key, firstErr)
}

// checkValue strictly decodes raw into t, recursing through nested oneOf unions
// so a union nested inside a variant is not decoded by its opaque
// UnmarshalJSON (which would accept anything).
func checkValue(raw json.RawMessage, t reflect.Type, seen map[reflect.Type]bool) error {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t != anyType && !seen[t] {
		seen[t] = true
		if variants := unionMembers(t); len(variants) > 0 && !hasCatchAllMember(variants) {
			return checkUnion(string(raw), variants)
		}
	}
	return strictDecode(string(raw), t)
}

// swallowsKeys reports whether encoding/json's own decoder for t accepts any
// body without matching keys at all: interface{}, an open map, or a type with a
// hand-written UnmarshalJSON. Scalar decoders are exempt — they have no key set.
func swallowsKeys(t reflect.Type) (string, bool) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch {
	case t == anyType, t == rawMessageType:
		return t.String() + " accepts any key", true
	case isScalarKind(t.Kind()), t == timeTimeType:
		return "", false
	case t.Kind() == reflect.Map:
		return t.String() + " is an open map, so every key is legal", true
	}
	// encoding/json uses a type's UnmarshalJSON both for T and for an addressable
	// *T, so a pointer-receiver method defeats the unknown-field check just as
	// much as a value-receiver one.
	if t.Implements(jsonUnmarshalerType) || reflect.PointerTo(t).Implements(jsonUnmarshalerType) {
		if variants := unionMembers(t); len(variants) > 0 {
			if hasCatchAllMember(variants) {
				return t.String() + " is a oneOf with an interface{} member", true
			}
			return "", false
		}
		// A scalar-shaped custom decoder — openapi_types.Date is struct{time.Time}
		// and parses a "2006-01-02" string — has no key set to reject either.
		if !hasJSONKeyedField(t) {
			return "", false
		}
		return t.String() + " defines its own UnmarshalJSON", true
	}
	return "", false
}

// isScalarKind reports whether a json value of this kind carries no keys.
func isScalarKind(k reflect.Kind) bool {
	switch k {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

// nestedGaps lists every position inside t where a key could not be checked.
// It does not judge t's own root — that is swallowsKeys' job — and it reports
// only the leaf that actually swallows the body, so a container of opaque
// elements is named by the element. The result is coverage reporting: each
// position listed is one where a wrong key stays invisible to this check.
func nestedGaps(t reflect.Type, seen map[reflect.Type]bool) []string {
	if seen[t] {
		return nil
	}
	seen[t] = true

	container := false
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
		container = true
	}
	if !container {
		if reason, ok := swallowsKeys(t); ok {
			return []string{reason}
		}
	}
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
		return nestedGaps(t.Elem(), seen)
	case reflect.Struct:
		var out []string
		for i := range t.NumField() {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue // unexported: not decodable, and json ignores it
			}
			out = append(out, nestedGaps(f.Type, seen)...)
		}
		return out
	}
	return nil
}

// hasJSONKeyedField reports whether struct t declares at least one field that
// json would match against an incoming key.
func hasJSONKeyedField(t reflect.Type) bool {
	if t.Kind() != reflect.Struct {
		return false
	}
	for i := range t.NumField() {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		}
		if _, ok := f.Tag.Lookup("json"); ok {
			return true
		}
		if f.Anonymous && hasJSONKeyedField(f.Type) {
			return true
		}
	}
	return false
}

// strictDecode decodes body into t with unknown fields rejected.
func strictDecode(body string, t reflect.Type) error {
	dec := json.NewDecoder(bytes.NewReader([]byte(body)))
	dec.DisallowUnknownFields()
	return dec.Decode(reflect.New(t).Interface())
}

// fieldByJSONTag finds the field of struct t whose JSON name is key.
func fieldByJSONTag(t reflect.Type, key string) (reflect.Type, bool) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, false
	}
	for i := range t.NumField() {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		}
		tag, ok := f.Tag.Lookup("json")
		if !ok {
			if f.Anonymous {
				if ft, found := fieldByJSONTag(f.Type, key); found {
					return ft, true
				}
			}
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		if name == key {
			ft := f.Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			return ft, true
		}
	}
	return nil, false
}

func sorted(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

// TestOpResponseTypeResolutionIsUnambiguous guards the premise the whole check
// rests on: one opId resolves to exactly one generated response wrapper, and
// within a wrapper no two fields claim the same HTTP status. If either stopped
// holding, validateShape would be picking a type at random.
func TestOpResponseTypeResolutionIsUnambiguous(t *testing.T) {
	iface := reflect.TypeOf((*client.ClientWithResponsesInterface)(nil)).Elem()

	wrappersPerOp := map[string]map[string]bool{}
	for i := range iface.NumMethod() {
		m := iface.Method(i)
		if !strings.HasSuffix(m.Name, "WithResponse") {
			continue
		}
		op := opIDFromMethod(m.Name)
		if op == "" || m.Type.NumOut() == 0 {
			continue
		}
		if wrappersPerOp[op] == nil {
			wrappersPerOp[op] = map[string]bool{}
		}
		wrappersPerOp[op][m.Type.Out(0).String()] = true
	}

	for op, types := range wrappersPerOp {
		if len(types) > 1 {
			t.Errorf("op %q resolves to %d response wrappers %v; the check would be guessing", op, len(types), types)
		}
	}

	collisions := 0
	for op, w := range opResponseTypes() {
		byStatus := map[string]string{}
		for i := range w.NumField() {
			m := statusRe.FindStringSubmatch(w.Field(i).Name)
			if m == nil {
				continue
			}
			if prev, ok := byStatus[m[2]]; ok {
				collisions++
				t.Errorf("op %q: client.%s has two fields for HTTP %s (%s, %s)", op, w.Name(), m[2], prev, w.Field(i).Name)
			}
			byStatus[m[2]] = w.Field(i).Name
		}
	}
	t.Logf("%d generated ops, %d with a response wrapper, %d status collisions", iface.NumMethod(), len(wrappersPerOp), collisions)
}

// TestValidateShapeRejectsMismatchedKeys is the regression guard for the two
// key-naming defects that validateShape could not see while it decoded into
// `any`. Each case is a real fixture body with the original defect reintroduced;
// the check must reject it and accept its repaired form.
func TestValidateShapeRejectsMismatchedKeys(t *testing.T) {
	tests := []struct {
		name    string
		op      string
		status  int
		good    string
		bad     string
		wantErr string
	}{
		{
			name: "getRequestsStatus executedAt is not a field of either oneOf member",
			op:   OpGetRequestsStatus,
			good: `{"requestId":5001,"status":"COMPLETED","dateSubmitted":"2026-01-02T20:30:00-05:00"}`,
			bad:  `{"requestId":5001,"status":"COMPLETED","executedAt":"2026-01-02T20:30:00-05:00"}`,
			// The repaired body mixes StatusResponse.dateSubmitted with
			// AmRequestStatusResponse.status, so it passes per key but a key from
			// neither member must not.
		},
		{
			name:    "createSsoSessions accessToken is not the generated tag",
			op:      OpCreateSsoSessions,
			good:    `{"access_token":"sso-access-token","active":true,"token_type":"Bearer"}`,
			bad:     `{"accessToken":"sso-access-token","active":true,"tokenType":"Bearer"}`,
			wantErr: `unknown field "accessToken"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if res := validateShape(tc.op, 0, tc.good); res.class == shapeFailed {
				t.Fatalf("repaired body rejected for %s: %v", tc.op, res.err)
			} else if res.class != shapeStrict && res.class != shapeUnion {
				t.Fatalf("repaired body was not key-checked for %s: class=%v reason=%s", tc.op, res.class, res.reason)
			}

			res := validateShape(tc.op, 0, tc.bad)
			if res.class != shapeFailed {
				t.Fatalf("bad body accepted for %s: class=%v reason=%s", tc.op, res.class, res.reason)
			}
			t.Logf("%s bad body: %v", tc.op, res.err)
			if tc.wantErr != "" && !strings.Contains(res.err.Error(), tc.wantErr) {
				t.Errorf("error %q does not mention %q", res.err, tc.wantErr)
			}
		})
	}
}

// TestValidateShapePreservesWellFormednessAndScalars pins the guarantees the
// original check did provide, so the stricter key check cannot quietly drop them.
func TestValidateShapePreservesWellFormednessAndScalars(t *testing.T) {
	if res := validateShape(OpGetBrokerageStatus, 0, `{"authenticated":true,`); res.class != shapeFailed {
		t.Errorf("malformed JSON accepted: class=%v", res.class)
	} else {
		t.Logf("malformed JSON: %v", res.err)
	}

	// downloadFile answers with a bare JSON string and declares no 2xx body.
	if res := validateShape(OpDownloadFile, 0, `"c3ludGhldGljLXRheC12b3VjaGVy"`); res.class != shapeScalar {
		t.Errorf("top-level scalar not whitelisted: class=%v reason=%s", res.class, res.reason)
	}

	// An op with no generated response type is reported, not silently accepted.
	if res := validateShape("notAnOperation", 0, `{"a":1}`); res.class != shapeNoResponseType {
		t.Errorf("unknown op not reported: class=%v", res.class)
	}

	// A well-formed body whose keys all match is still accepted.
	if res := validateShape(OpGetBrokerageStatus, 0, `{"authenticated":true,"established":true}`); res.class != shapeStrict {
		t.Errorf("conforming body rejected: class=%v reason=%s err=%v", res.class, res.reason, res.err)
	}

	// A key that no response type for the op declares fails, which is the whole
	// point of the check.
	if res := validateShape(OpGetBrokerageStatus, 0, `{"autheticated":true}`); res.class != shapeFailed {
		t.Errorf("mismatched key accepted: class=%v reason=%s", res.class, res.reason)
	} else {
		t.Logf("mismatched key: %v", res.err)
	}
}
