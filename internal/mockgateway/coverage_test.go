// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package mockgateway

import (
	"os"
	"sort"
	"strings"
	"testing"
)

// specPath is docs/SPEC.md relative to this package directory. It is packaged
// with the repository, so the guard never touches the network.
const specPath = "../../docs/SPEC.md"

// Canonical operation counts from docs/SPEC.md (AGENTS.md rule 6).
const (
	cpapiOperationCount  = 123
	ibRESTOperationCount = 70
	totalOperationCount  = cpapiOperationCount + ibRESTOperationCount
)

// specOp is one canonical operation row extracted from docs/SPEC.md.
type specOp struct {
	method string
	path   string
	opID   string
}

// routeKey identifies a method/path pair in the normalized route table.
type routeKey struct {
	method string
	path   string
}

// TestCPAPICoverage asserts that every CPAPI (ssoBearer) operation in
// docs/SPEC.md has a matching route registered by the mock gateway. Dynamic
// path segments are normalized to "{}" on both sides before comparison.
func TestCPAPICoverage(t *testing.T) {
	specOps := parseSpecSurface(t, "CPAPI")
	if len(specOps) != cpapiOperationCount {
		t.Fatalf("parsed %d CPAPI rows from %s; want %d", len(specOps), specPath, cpapiOperationCount)
	}
	assertCoverage(t, "CPAPI", specOps)
}

// TestIBRESTCoverage asserts that every IB REST operation in docs/SPEC.md has a
// matching route registered by the mock gateway. The IB REST surface includes
// the OAuth2 token endpoint (generateToken), whose route is unauthenticated.
func TestIBRESTCoverage(t *testing.T) {
	specOps := parseSpecSurface(t, "IB REST")
	if len(specOps) != ibRESTOperationCount {
		t.Fatalf("parsed %d IB REST rows from %s; want %d", len(specOps), specPath, ibRESTOperationCount)
	}
	assertCoverage(t, "IB REST", specOps)
}

// TestAllOperationsCoverage asserts the combined CPAPI + IB REST surface has
// zero unrouted operations and that docs/SPEC.md still declares 193.
func TestAllOperationsCoverage(t *testing.T) {
	cpapi := parseSpecSurface(t, "CPAPI")
	rest := parseSpecSurface(t, "IB REST")
	total := len(cpapi) + len(rest)
	if total != totalOperationCount {
		t.Fatalf("parsed %d operations from %s; want %d", total, specPath, totalOperationCount)
	}
	assertCoverage(t, "CPAPI + IB REST", append(append([]specOp{}, cpapi...), rest...))
}

// assertCoverage fails with an explicit missing list when any spec operation
// has no registered route.
func assertCoverage(t *testing.T, surface string, specOps []specOp) {
	t.Helper()
	if len(specOps) == 0 {
		t.Fatalf("parsed 0 %s rows from %s", surface, specPath)
	}

	table := make(map[routeKey]string)
	for _, rt := range defaultRoutes() {
		path := normalizeSegments(rt.segments)
		for _, method := range rt.methods {
			table[routeKey{method: method, path: path}] = rt.op
		}
	}

	var missing []string
	for _, op := range specOps {
		k := routeKey{method: op.method, path: normalizePath(op.path)}
		if _, ok := table[k]; !ok {
			missing = append(missing, op.method+" "+op.path+" ("+op.opID+")")
		}
	}
	if len(missing) > 0 {
		t.Fatalf("missing %d of %d %s routes:\n  %s",
			len(missing), len(specOps), surface, strings.Join(missing, "\n  "))
	}
	t.Logf("%s coverage: %d/%d operations routed", surface, len(specOps), len(specOps))
}

// pathCollisionOps lists SPEC operations that are deliberately given no route of
// their own, because another operation's route covers the same normalized method
// and path and always wins the scoring tie. The value names the operation doing
// the shadowing, so the reason travels with the data rather than living only in a
// comment beside the route table.
//
// This map exists because the coverage tests above are structurally unable to see
// this class of gap. They match on normalized method and path, so an operation
// that shares a path with another one satisfies them through the other one's
// route: the operation is unroutable, and no test says so. Matching on opId is
// the only thing that notices, and an unexplained entry here fails the build
// rather than quietly shrinking the router.
var (
	pathCollisionOps = map[string]string{
		// Both are POST /v1/api/iserver/account/{segment}/orders. The SDK sends a
		// byte-identical payload for each, so neither a path nor a body predicate can
		// separate them. See TestModelOrderRouteCollision for the router-level view.
		OpSubmitModelPortfolioOrder: OpSubmitNewOrder,
	}

	// sharedOpIDs names operations that docs/SPEC.md labels for more than one
	// route. The mock gateway keys routes and fixtures by opId, so two endpoints
	// sharing an opId also share a fixture. The spec does this once, deliberately,
	// so it is listed here with its route count rather than left implicit.
	sharedOpIDs = map[string]int{
		// Two distinct endpoints: a contract trading schedule and a security
		// definition schedule. Both routes register under OpGetTradingSchedule.
		OpGetTradingSchedule: 2,
	}
)

// TestEveryOpIDIsRoutedOrExplained is the opId-aware companion to the method+path
// coverage tests. Every distinct operation in docs/SPEC.md must either be
// registered in the route table or appear in pathCollisionOps with a reason that
// this test independently verifies.
func TestEveryOpIDIsRoutedOrExplained(t *testing.T) {
	specOps := append(parseSpecSurface(t, "CPAPI"), parseSpecSurface(t, "IB REST")...)

	// Collapse to distinct opIds, tracking how many distinct routes each covers.
	distinct := make(map[string]specOp, len(specOps))
	routesPerOp := make(map[string]map[routeKey]bool, len(specOps))
	for _, op := range specOps {
		if _, ok := distinct[op.opID]; !ok {
			distinct[op.opID] = op
		}
		if routesPerOp[op.opID] == nil {
			routesPerOp[op.opID] = make(map[routeKey]bool, 1)
		}
		routesPerOp[op.opID][routeKey{method: op.method, path: normalizePath(op.path)}] = true
	}

	// One opId spanning several routes is surprising enough to be explicit, in
	// both directions: a new one must be declared, and a declared one that no
	// longer applies must be removed.
	for opID, keys := range routesPerOp {
		if len(keys) > 1 {
			want, declared := sharedOpIDs[opID]
			switch {
			case !declared:
				t.Errorf("opId %q covers %d routes in %s; declare it in sharedOpIDs if that is intended",
					opID, len(keys), specPath)
			case len(keys) != want:
				t.Errorf("sharedOpIDs says %q covers %d routes, but %s gives it %d",
					opID, want, specPath, len(keys))
			}
		}
	}
	for opID, want := range sharedOpIDs {
		if _, isOp := distinct[opID]; !isOp {
			t.Errorf("sharedOpIDs lists %q, which is not an operation in %s", opID, specPath)
			continue
		}
		if got := len(routesPerOp[opID]); got != want {
			t.Errorf("sharedOpIDs says %q covers %d routes, but %s gives it %d",
				opID, want, specPath, got)
		}
	}

	routed := make(map[string]bool)
	for _, rt := range defaultRoutes() {
		routed[rt.op] = true
	}

	// Each exception must be justified, or it is just a hole with a comment on it.
	var explained []string
	for opID, shadow := range pathCollisionOps {
		self, isOp := distinct[opID]
		if !isOp {
			t.Errorf("pathCollisionOps lists %q, which is not an operation in %s", opID, specPath)
			continue
		}
		other, isShadow := distinct[shadow]
		if !isShadow {
			t.Errorf("pathCollisionOps says %q is shadowed by %q, which is not an operation in %s",
				opID, shadow, specPath)
			continue
		}
		if !routed[shadow] {
			t.Errorf("pathCollisionOps says %q is shadowed by %q, but %q has no route either - "+
				"this entry hides a missing route", opID, shadow, shadow)
			continue
		}
		if self.method != other.method || normalizePath(self.path) != normalizePath(other.path) {
			t.Errorf("pathCollisionOps says %q collides with %q, but %s %s and %s %s do not",
				opID, shadow, self.method, self.path, other.method, other.path)
			continue
		}
		explained = append(explained, opID)
	}

	var unrouted []string
	for opID, op := range distinct {
		if routed[opID] {
			continue
		}
		if _, ok := pathCollisionOps[opID]; ok {
			continue
		}
		unrouted = append(unrouted, op.method+" "+op.path+" ("+opID+")")
	}
	if len(unrouted) > 0 {
		sort.Strings(unrouted)
		t.Fatalf("%d operation(s) have neither a route nor a pathCollisionOps entry:\n  %s\n"+
			"Add the route, or add the operation to pathCollisionOps with the operation that shadows it.",
			len(unrouted), strings.Join(unrouted, "\n  "))
	}

	sort.Strings(explained)
	t.Logf("opId coverage: %d distinct operations, %d routed, %d explained by a path collision%s",
		len(distinct), len(distinct)-len(explained), len(explained),
		map[bool]string{true: ": " + strings.Join(explained, ", "), false: ""}[len(explained) > 0])
}

// TestRoutesHaveFixtures asserts every routed operation has a fixture, so a
// matched route never falls through to the generic 404.
func TestRoutesHaveFixtures(t *testing.T) {
	fixtures := DefaultFixtures()
	for _, rt := range defaultRoutes() {
		if _, ok := fixtures.Get(rt.op); !ok {
			t.Errorf("route %v %s (op %q) has no fixture", rt.methods, normalizeSegments(rt.segments), rt.op)
		}
	}
}

// parseSpecSurface extracts the rows for the given surface from docs/SPEC.md.
func parseSpecSurface(t *testing.T, surface string) []specOp {
	t.Helper()
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read %s: %v", specPath, err)
	}
	var ops []specOp
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cols := strings.Split(line, "|")
		// | METHOD | `path` | surface | auth | summary | `opId` |
		if len(cols) < 7 {
			continue
		}
		method := strings.TrimSpace(cols[1])
		path := strings.Trim(strings.TrimSpace(cols[2]), "`")
		rowSurface := strings.TrimSpace(cols[3])
		opID := strings.Trim(strings.TrimSpace(cols[6]), "`")
		if rowSurface != surface || opID == "" || path == "" {
			continue
		}
		switch method {
		case "GET", "POST", "PUT", "DELETE", "PATCH":
		default:
			continue
		}
		ops = append(ops, specOp{method: method, path: path, opID: opID})
	}
	return ops
}

// normalizePath normalizes a URL path's dynamic segments to "{}".
func normalizePath(path string) string {
	return normalizeSegments(splitPath(path))
}

// normalizeSegments joins path segments, replacing placeholders with "{}".
func normalizeSegments(segments []string) string {
	if len(segments) == 0 {
		return "/"
	}
	out := make([]string, len(segments))
	for i, seg := range segments {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			out[i] = "{}"
			continue
		}
		out[i] = seg
	}
	return "/" + strings.Join(out, "/")
}

// TestModelOrderRouteCollision pins a known limitation of the path-based mock
// router. IBKR exposes submitNewOrder
// (POST /v1/api/iserver/account/{accountId}/orders) and
// submitModelPortfolioOrder (POST /v1/api/iserver/account/{modelCode}/orders)
// as separate operations, but once placeholders are normalized the two are
// indistinguishable: same method, same segment count, same literal count. The
// router therefore scores them equally and the first-declared route wins.
//
// The SDK sends a byte-identical payload for both, so a body predicate cannot
// separate them either, and submitModelPortfolioOrder is deliberately left
// unrouted rather than shadowed by a route that could never be selected. The
// docs/SPEC.md coverage check still passes because it compares normalized
// method and path. Callers needing the model-portfolio response decoded install
// the shape on the shared route; see TestModels_SubmitModelPortfolioOrder in
// pkg/ibkr.
//
// This test documents the collision so a future router change that alters which
// route wins is noticed deliberately rather than by accident.
func TestModelOrderRouteCollision(t *testing.T) {
	const modelOrderPath = "/v1/api/iserver/account/U1234567/orders"

	winner, params, ok := matchRoute(defaultRoutes(), "POST", modelOrderPath)
	if !ok {
		t.Fatal("no route matched the model order path")
	}
	if winner.op != OpSubmitNewOrder {
		t.Fatalf("winning route = %q, want %q (first-declared wins on a score tie)",
			winner.op, OpSubmitNewOrder)
	}
	if params["accountId"] != "U1234567" {
		t.Errorf("captured accountId = %q, want %q", params["accountId"], "U1234567")
	}

	// The model operation is declared for documentation but must not be
	// registered, because a registered route here could never be selected and
	// would only add a fixture that is never served.
	for _, rt := range defaultRoutes() {
		if rt.op == OpSubmitModelPortfolioOrder {
			t.Errorf("OpSubmitModelPortfolioOrder is registered at %v but can never be matched",
				rt.segments)
		}
	}

	// Contrast: an operation with a genuinely different path shape still routes
	// to its own route, so the collision is specific to the shared shape.
	openOrders, _, ok := matchRoute(defaultRoutes(), "GET", "/v1/api/iserver/account/orders")
	if !ok {
		t.Fatal("no route matched the open-orders path")
	}
	if openOrders.op != OpGetOpenOrders {
		t.Errorf("GET /v1/api/iserver/account/orders resolved to %q, want %q",
			openOrders.op, OpGetOpenOrders)
	}
}
