// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// check_design_docs.go verifies that design documents accurately describe the codebase.
// It is run by `make design-check`, and by the docs-guards CI job.
//
// Checks:
//   - The middleware chain in docs/design/01-transport.md matches internal/transport.go
//   - The Client struct block, accessor list, NewClient contract and close semantics
//     in docs/design/02-client.md match pkg/ibkr
//   - The manager method counts in docs/design/03-managers.md match actual exports
//   - The import boundary and the no-generated-type-exposed rule in
//     docs/design/04-generated-wrapping.md match the module layout
//   - The subscription surface, buffer default and ping/backoff parameters in
//     docs/design/05-streaming.md match pkg/ibkr/ws.go and internal/ws.go
//   - The Error struct, RetryPolicy defaults, safe-method set, no-auto-retry gate and
//     circuit-breaker default in docs/design/06-errors-retries.md match the code
//   - The redaction list, the insecure-skip-verify warning and the OAuth generation
//     mechanism in docs/design/08-concurrency.md match the code
//   - The order-flow endpoints, public order API, hand-built wire body, explicit
//     confirmation, single-attempt mutation path and error mapping in
//     docs/design/09-orders-and-confirmation.md match the code
//
// Usage: go run ./scripts/check_design [-fix]
//
//	-fix  rewrite the generated manager table in docs/design/03-managers.md
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// designCheck is one document assertion: the document it verifies and the
// function that compares that document against the code.
type designCheck struct {
	doc string
	fn  func(repoRoot string) error
}

// designChecks is the registry of verified documents. A document is listed here
// only if some check actually reads it; an entry here is a claim that drift in
// that document fails the build, so adding one without a check is exactly the
// gap this program exists to close.
//
// docs/design/07-money-and-numbers.md is covered by checkMoneyNumberFields, not
// omitted. Its main claim - money and quantities never reach a float - is
// already enforced by scripts/check_money.py over the whole tree, and re-checking
// that here would be one more thing to keep in step and no stronger. But the
// document also names specific fields as json.Number, and check_money.py cannot
// see those: it proves the absence of a float, not the presence of a particular
// type in generated code. That claim is checked here, so a spec edit that
// regenerates a money field back to a float fails the build.
var designChecks = []designCheck{
	{"docs/design/01-transport.md", checkTransportChain},
	{"docs/design/07-money-and-numbers.md", checkMoneyNumberFields},
	{clientDoc, checkClientComposition},
	{"docs/design/03-managers.md", checkManagerCounts},
	{wrapDoc, checkGeneratedWrapping},
	{streamDoc, checkStreaming},
	{errDoc, checkErrorRetries},
	{concDoc, checkConcurrency},
	{ordersDoc, checkOrdersAndConfirmation},
}

// checkClientComposition runs the docs/design/02-client.md checks as one unit.
// The rationale for collecting rather than returning on the first failure is the
// one checkErrorRetries gives: a maintainer who introduced drift in two of these
// facts should see both in one run.
func checkClientComposition(repoRoot string) error {
	return collectDocErrors(repoRoot, clientDoc, []docCheck{
		{"Client struct block", checkClientStructBlock},
		{"accessor list", checkClientAccessors},
		{"NewClient contract", checkNewClientContract},
		{"options block", checkClientOptions},
		{"close semantics", checkClientCloseContract},
	})
}

// checkOrdersAndConfirmation runs the docs/design/09-orders-and-confirmation.md
// checks as one unit.
func checkOrdersAndConfirmation(repoRoot string) error {
	return collectDocErrors(repoRoot, ordersDoc, []docCheck{
		{"order flow endpoints", checkOrderFlowEndpoints},
		{"public order API", checkPublicOrderAPI},
		{"hand-built wire body", checkHandBuiltWireBody},
		{"confirmation is explicit", checkConfirmationIsExplicit},
		{"order mutations are single-attempt", checkOrderCallsGoThroughMutate},
		{"error mapping", checkErrorMapping},
	})
}

// checkErrorRetries runs the docs/design/06-errors-retries.md checks as one
// unit.
func checkErrorRetries(repoRoot string) error {
	return collectDocErrors(repoRoot, errDoc, []docCheck{
		{"Error struct field set", checkErrorStruct},
		{"public Error type alias", checkErrorTypeAlias},
		{"RetryPolicy defaults", checkRetryPolicyDefaults},
		{"safe-method set", checkSafeMethods},
		{"order mutations never retried", checkOrderMutationsNotRetried},
		{"circuit breaker disabled by default", checkBreakerDisabledByDefault},
	})
}

// docCheck is one assertion about a document, with a short label for the message.
type docCheck struct {
	what string
	fn   func(string) error
}

// collectDocErrors runs a document's checks as one unit, collecting every failure
// rather than returning on the first, for the reason checkErrorRetries gives: a
// maintainer who introduced drift in two of these facts should see both in one
// run instead of playing whack-a-mole. Each failure is prefixed with the document
// and the label, so one line names the file to edit and the claim it broke.
func collectDocErrors(repoRoot, doc string, checks []docCheck) error {
	var errs []string
	for _, c := range checks {
		if err := c.fn(repoRoot); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %s: %v", doc, c.what, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "\n"))
	}
	return nil
}

// -fix rewrites the generated manager table in docs/design/03-managers.md from
// the source and exits without running the checks, in the shape gofmt uses: -w
// writes, the default compares. It is deliberately not wired into `make check`,
// because a gate that edits its own subject is not a gate.
func main() {
	repoRoot, _ := os.Getwd()
	fix := flag.Bool("fix", false, "rewrite the generated manager table in docs/design/03-managers.md")
	flag.Parse()

	if *fix {
		if err := fixManagerTable(repoRoot); err != nil {
			fmt.Fprintf(os.Stderr, "design doc fix failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("regenerated docs/design/03-managers.md manager table")
		return
	}

	verified, err := run(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "design doc check failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("design docs OK")
	for _, doc := range verified {
		fmt.Printf("  verified %s\n", doc)
	}
}

func run(repoRoot string) ([]string, error) {
	var errs []string
	var verified []string

	for _, c := range designChecks {
		if err := c.fn(repoRoot); err != nil {
			errs = append(errs, err.Error())
			continue
		}
		verified = append(verified, c.doc)
	}

	if len(errs) > 0 {
		return verified, fmt.Errorf("%s", strings.Join(errs, "\n"))
	}
	return verified, nil
}

// checkTransportChain verifies that the middleware chain diagram in 01-transport.md
// matches the actual order in internal/transport.go NewClientTransport function.
func checkTransportChain(repoRoot string) error {
	transportPath := filepath.Join(repoRoot, "internal", "transport.go")
	//nolint:gosec // transportPath is repoRoot joined with a constant, and repoRoot is the process working directory, never caller input
	src, err := os.ReadFile(transportPath)
	if err != nil {
		return fmt.Errorf("internal/transport.go: %w", err)
	}

	// Extract the actual middleware order from NewClientTransport by parsing
	// the if blocks in order.
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, transportPath, src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("internal/transport.go: parse error: %w", err)
	}

	var actualOrder []string
	var transportFn *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "NewClientTransport" {
			continue
		}
		transportFn = fn
	}

	// Collect the middleware names in assembly order. Walking the
	// `ms = append(ms, Name(...))` calls directly is deliberate: the guards are
	// a mix of `cfg.Field != nil`, `cfg.Field != ""`, and method calls such as
	// `cfg.Retry.enabled()`, so matching on the condition shape is brittle.
	// Reading the appends covers guarded and unconditional layers alike.
	appendName := map[string]string{
		"RequestID":      "requestID",
		"UserAgent":      "userAgent",
		"Auth":           "auth",
		"Logging":        "logging/telemetry",
		"Instrument":     "Instrument",
		"CircuitBreaker": "circuitBreaker",
		"Retry":          "retry",
		"RateLimit":      "rateLimit",
		"Timeout":        "timeout",
		"MaxBytes":       "MaxBytes",
		"ErrorDecode":    "ErrorDecode",
	}
	if transportFn != nil {
		ast.Inspect(transportFn.Body, func(n ast.Node) bool {
			assign, ok := n.(*ast.AssignStmt)
			if !ok || len(assign.Rhs) != 1 {
				return true
			}
			call, ok := assign.Rhs[0].(*ast.CallExpr)
			if !ok || len(call.Args) < 2 {
				return true
			}
			// ms = append(ms, X) -> Args[1] is either a middleware call or, for
			// the variadic user slice, a bare selector.
			switch arg := call.Args[1].(type) {
			case *ast.SelectorExpr:
				if id, ok := arg.X.(*ast.Ident); ok && id.Name == "cfg" && arg.Sel.Name == "UserMiddleware" {
					actualOrder = append(actualOrder, "UserMiddleware")
				}
			case *ast.CallExpr:
				ident, ok := arg.Fun.(*ast.Ident)
				if !ok {
					return true
				}
				if mapped, ok := appendName[ident.Name]; ok {
					actualOrder = append(actualOrder, mapped)
				}
			}
			return true
		})
	}
	if len(actualOrder) == 0 {
		return fmt.Errorf("internal/transport.go: no middleware appends found in NewClientTransport; the order check cannot run")
	}

	var deduped []string
	for _, m := range actualOrder {
		if len(deduped) == 0 || deduped[len(deduped)-1] != m {
			deduped = append(deduped, m)
		}
	}

	// Build the expected chain from the doc
	docPath := filepath.Join(repoRoot, "docs", "design", "01-transport.md")
	//nolint:gosec // docPath is repoRoot joined with a constant, and repoRoot is the process working directory, never caller input
	docSrc, err := os.ReadFile(docPath)
	if err != nil {
		return fmt.Errorf("docs/design/01-transport.md: %w", err)
	}

	// Extract the chain from the doc using the code block
	chainRe := regexp.MustCompile(`(?s)request\s+└─(.+?)\s+response`)
	m := chainRe.FindSubmatch(docSrc)
	if m == nil {
		return fmt.Errorf("docs/design/01-transport.md: could not find middleware chain diagram")
	}

	chainStr := string(m[1])

	// Normalize whitespace and box-drawing characters before splitting by arrow.
	// First collapse each line's leading noise (spaces, box chars) then join lines.
	lines := strings.Split(chainStr, "\n")
	var cleanLineParts []string
	for _, line := range lines {
		// Remove box-drawing characters and trim
		for _, r := range line {
			if r == ' ' || r == '│' || r == '├' || r == '└' || r == '┌' || r == '┐' || r == '┘' || r == '┴' || r == '┬' {
				continue
			}
			cleanLineParts = append(cleanLineParts, string(r))
		}
	}
	normalized := strings.Join(cleanLineParts, "")

	// Split by arrow (preserving spaces within parts)
	splitParts := strings.Split(normalized, "→")
	for i := range splitParts {
		splitParts[i] = strings.TrimSpace(splitParts[i])
	}

	// Filter empty parts and strip parenthetical annotations
	var docOrder []string
	for _, p := range splitParts {
		p = strings.TrimSpace(p)
		if p == "" || p == "http.Transport.Do" {
			continue
		}
		// Strip parenthetical annotations: "foo (bar)" -> "foo"
		if idx := strings.Index(p, "("); idx != -1 {
			p = strings.TrimSpace(p[:idx])
		}
		if p != "" {
			docOrder = append(docOrder, p)
		}
	}

	// errorDecode is not named in the doc diagram, it maps to "ErrorDecode"
	// Map doc names to code names
	nameMap := map[string]string{
		"logging/telemetry": "logging/telemetry",
		"Instrument":        "Instrument",
		"circuitBreaker":    "circuitBreaker",
		"retry":             "retry",
		"rateLimit":         "rateLimit",
		"timeout":           "timeout",
		"maxBytes":          "MaxBytes",
		"errorDecode":       "ErrorDecode",
		"UserMiddleware":    "UserMiddleware",
		"requestID":         "requestID",
		"userAgent":         "userAgent",
		"auth":              "auth",
	}

	// Compare: we skip requestID, userAgent, auth (they appear in both but
	// are implicit in the doc diagram that just shows the full flow)
	// The doc shows the full flow including all layers; we just verify
	// that all the key middleware layers appear in the right relative order.

	// Build doc order with mapped names
	var docMapped []string
	for _, d := range docOrder {
		if mapped, ok := nameMap[d]; ok {
			docMapped = append(docMapped, mapped)
		}
	}

	// Keep the doc's own internal ordering check: every layer the chain diagram
	// claims must appear, and in the documented sequence.
	expectedKeywords := []string{
		"requestID", "userAgent", "auth", "logging/telemetry", "Instrument",
		"circuitBreaker", "retry", "rateLimit", "timeout", "maxBytes",
		"errorDecode", "UserMiddleware",
	}
	lastIdx := -1
	for _, kw := range expectedKeywords {
		idx := -1
		for j, d := range docOrder {
			if d == kw {
				idx = j
				break
			}
		}
		if idx == -1 && (kw == "requestID" || kw == "userAgent" || kw == "auth") {
			continue
		}
		if idx == -1 {
			return fmt.Errorf("docs/design/01-transport.md: middleware %q not found in chain diagram", kw)
		}
		if idx <= lastIdx && lastIdx != -1 {
			return fmt.Errorf("docs/design/01-transport.md: %q appears before its predecessor in the chain (order: %v)", kw, docOrder)
		}
		lastIdx = idx
	}

	// Now compare the documented chain against the middleware that
	// NewClientTransport actually assembles. Without this, the diagram can
	// drift from the code and the checker still passes.
	docSet := make(map[string]int, len(docMapped))
	for i, d := range docMapped {
		docSet[d] = i
	}

	codeSet := make(map[string]bool, len(deduped))
	for _, c := range deduped {
		codeSet[c] = true
	}

	// Every layer the code assembles must be named in the diagram.
	for _, c := range deduped {
		if _, ok := docSet[c]; !ok {
			return fmt.Errorf(
				"transport drift: NewClientTransport assembles %q but docs/design/01-transport.md does not list it (code: %v, doc: %v)",
				c, deduped, docMapped)
		}
	}

	// The relative order of the shared layers must match.
	var codeShared, docShared []string
	for _, c := range deduped {
		if _, ok := docSet[c]; ok {
			codeShared = append(codeShared, c)
		}
	}
	for _, d := range docMapped {
		if codeSet[d] {
			docShared = append(docShared, d)
		}
	}
	if len(codeShared) != len(docShared) {
		return fmt.Errorf(
			"transport drift: code and document disagree on which layers are present (code: %v, doc: %v)",
			codeShared, docShared)
	}
	for i := range codeShared {
		if codeShared[i] != docShared[i] {
			return fmt.Errorf(
				"transport drift: middleware order differs. code: %v, docs/design/01-transport.md: %v",
				codeShared, docShared)
		}
	}

	return nil
}

// managerScopes is the Scope column of the 03-managers.md table, and the reason
// that table is generated rather than hand-written.
//
// The Method column is a fact about the code, so it can be rendered from the AST.
// The Scope column is prose and cannot be derived, so it lives here and is emitted
// alongside the count. That is the honest division: the checker owns the whole
// table, the prose around it stays hand-written, and a manager added to pkg/ibkr
// without a scope here is an error rather than a row with a blank cell.
var managerScopes = map[string]string{
	"AccountManager":        "accounts & summaries",
	"AlertManager":          "alerts",
	"AllocationManager":     "FA allocation",
	"FYIManager":            "FYIs / notifications",
	"ForecastManager":       "event contracts",
	"MarketDataManager":     "quotes, history & streaming",
	"ModelManager":          "model portfolios",
	"OAuthManager":          "OAuth1",
	"PerformanceManager":    "PortfolioAnalyst",
	"PortfolioManager":      "positions & ledger",
	"ScannerManager":        "market scanner",
	"SessionManager":        "session lifecycle & health",
	"TradeManager":          "orders & contracts",
	"TradingAccountManager": "trading-account ops",
	"WatchlistManager":      "watchlists",
}

// The generated region of 03-managers.md. Everything between these markers is
// rewritten by -fix and compared byte for byte otherwise, so a hand-edited count
// fails even when it happens to satisfy the per-manager check below.
const (
	managersTableStart = "<!-- generated: managers-table -->"
	managersTableEnd   = "<!-- /generated: managers-table -->"
)

// renderManagerTable produces the whole generated region, markers included, from
// the same managerMethodCounts walk that verifies it.
//
// It shares that walk with the check deliberately. A second implementation - a
// Python script parsing Go, say - could count differently from the verifier, and
// then "regenerate" a table the verifier rejects. One walk, two modes, the shape
// gofmt uses: -fix writes, the default compares.
func renderManagerTable(repoRoot string) (string, error) {
	counts, err := managerMethodCounts(repoRoot)
	if err != nil {
		return "", err
	}
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)

	var missing []string
	var b strings.Builder
	b.WriteString(managersTableStart + "\n\n")
	b.WriteString("| Manager | Scope | Methods (implemented) |\n")
	b.WriteString("|---------|-------|-----------------------|\n")
	for _, name := range names {
		scope, ok := managerScopes[name]
		if !ok {
			missing = append(missing, name)
			continue
		}
		fmt.Fprintf(&b, "| `%s` | %s | %d ops |\n", name, scope, counts[name])
	}
	if len(missing) > 0 {
		return "", fmt.Errorf(
			"no scope in managerScopes for %s; add one so the generated table can describe it",
			strings.Join(missing, ", "))
	}
	// A scope with no matching type is the same drift in the other direction: the
	// table would silently lose a row.
	var stale []string
	for name := range managerScopes {
		if _, ok := counts[name]; !ok {
			stale = append(stale, name)
		}
	}
	if len(stale) > 0 {
		sort.Strings(stale)
		return "", fmt.Errorf(
			"managerScopes names %s, which pkg/ibkr does not declare; remove the stale entries",
			strings.Join(stale, ", "))
	}
	b.WriteString("\n" + managersTableEnd + "\n")
	return b.String(), nil
}

// checkManagerTableRegion compares the committed generated region against what
// renderManagerTable produces, and reports both sides on a mismatch.
func checkManagerTableRegion(repoRoot string) error {
	docPath := filepath.Join(repoRoot, "docs", "design", "03-managers.md")
	//nolint:gosec // docPath is repoRoot joined with a constant, and repoRoot is the process working directory, never caller input
	docSrc, err := os.ReadFile(docPath)
	if err != nil {
		return fmt.Errorf("docs/design/03-managers.md: %w", err)
	}
	doc := string(docSrc)

	start := strings.Index(doc, managersTableStart)
	end := strings.Index(doc, managersTableEnd)
	if start < 0 || end < 0 || end < start {
		return fmt.Errorf(
			"docs/design/03-managers.md: the generated table markers are missing or out of order; "+
				"expected %q before %q", managersTableStart, managersTableEnd)
	}
	committed := doc[start:end+len(managersTableEnd)] + "\n"

	want, err := renderManagerTable(repoRoot)
	if err != nil {
		return err
	}
	if committed == want {
		return nil
	}
	return fmt.Errorf(
		"docs/design/03-managers.md: the generated manager table is stale; "+
			"run `go run ./scripts/check_design -fix`\n--- committed ---\n%s\n--- should be ---\n%s",
		committed, want)
}

// fixManagerTable rewrites the generated region in place. Called only by -fix.
func fixManagerTable(repoRoot string) error {
	docPath := filepath.Join(repoRoot, "docs", "design", "03-managers.md")
	//nolint:gosec // docPath is repoRoot joined with a constant, and repoRoot is the process working directory, never caller input
	docSrc, err := os.ReadFile(docPath)
	if err != nil {
		return fmt.Errorf("docs/design/03-managers.md: %w", err)
	}
	doc := string(docSrc)

	start := strings.Index(doc, managersTableStart)
	end := strings.Index(doc, managersTableEnd)
	if start < 0 || end < 0 || end < start {
		return fmt.Errorf("docs/design/03-managers.md: generated table markers missing or out of order")
	}

	want, err := renderManagerTable(repoRoot)
	if err != nil {
		return err
	}
	updated := doc[:start] + strings.TrimSuffix(want, "\n") + doc[end+len(managersTableEnd):]
	if updated == doc {
		return nil
	}
	return os.WriteFile(docPath, []byte(updated), 0o600)
}

// checkManagerCounts verifies that the method count listed for each manager in
// 03-managers.md matches the exported methods actually declared on that type.
//
// Three properties this check depends on, each of which was a way for it to pass
// without comparing anything.
//
// The manager set comes from the source, not from a table beside this function, so
// a manager added to pkg/ibkr cannot go unverified by being forgotten here.
//
// The per-manager source-file list is gone. A method declared in ws.go rather than
// in its manager's home file was previously only counted if the map named that
// file, and two independent omissions cancelled out: AccountManager's
// SubscribeAccount lived in ws.go, which the map omitted, and the doc did not list
// the method either. Undocumented method, uncounted file, row compared equal, both
// wrong. Counting across the package removes the second omission, and the strict
// "one row per manager, N ops" format removes the first.
//
// A missing doc row is a failure rather than a skip. The old code returned early
// when it could not find a count, which is what let every grouped row -
// "| `AlertManager`, `ForecastManager`, `ScannerManager` | ... | 7, 5, and 2 ops
// respectively |" - pass unverified for nine runs while the numbers in it rotted.
//
// The per-manager comparison below and checkManagerTableRegion overlap: if the
// committed table equals the rendered one, every count necessarily matches. The
// per-manager pass is kept anyway because it names the offending manager and both
// numbers, where a region diff shows two whole tables and makes the reader find it.
func checkManagerCounts(repoRoot string) error {
	if err := checkManagerTableRegion(repoRoot); err != nil {
		return err
	}

	docPath := filepath.Join(repoRoot, "docs", "design", "03-managers.md")
	//nolint:gosec // docPath is repoRoot joined with a constant, and repoRoot is the process working directory, never caller input
	docSrc, err := os.ReadFile(docPath)
	if err != nil {
		return fmt.Errorf("docs/design/03-managers.md: %w", err)
	}

	actual, err := managerMethodCounts(repoRoot)
	if err != nil {
		return err
	}

	names := make([]string, 0, len(actual))
	for name := range actual {
		names = append(names, name)
	}
	sort.Strings(names)

	var failures []string
	for _, name := range names {
		docCount, found := extractDocCount(string(docSrc), name)
		if !found {
			failures = append(failures, fmt.Sprintf(
				"docs/design/03-managers.md: %s declares %d exported methods in pkg/ibkr but has no row; "+
					"add one reading `%d ops`", name, actual[name], actual[name]))
			continue
		}
		if docCount != actual[name] {
			failures = append(failures, fmt.Sprintf(
				"docs/design/03-managers.md: %s declares %d exported methods in pkg/ibkr but the doc says %d",
				name, actual[name], docCount))
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("%d manager method count mismatch(es):\n  %s",
			len(failures), strings.Join(failures, "\n  "))
	}
	return nil
}

// managerMethodCounts returns, for every `*Manager` struct type declared in
// pkg/ibkr, the number of exported methods declared on it. Both the set of managers
// and their counts are read from the source, so nothing here needs editing when a
// manager gains a method or a method moves to a different file in the package.
//
// Types are collected separately from their methods, and seeded at zero, because a
// manager with no methods at all is exactly the case worth reporting - scanning
// methods alone would omit it, and ForecastManager is presently one of those.
func managerMethodCounts(repoRoot string) (map[string]int, error) {
	dir := filepath.Join(repoRoot, "pkg", "ibkr")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read pkg/ibkr: %w", err)
	}

	counts := map[string]int{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		//nolint:gosec // path comes from a ReadDir entry under a constant dir, never from a caller or the environment
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, path, src, 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}

		for _, decl := range parsed.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || !strings.HasSuffix(ts.Name.Name, "Manager") {
						continue
					}
					if _, ok := ts.Type.(*ast.StructType); !ok {
						continue
					}
					// Seed the type so a manager with no methods at all is still
					// reported. A method-only scan would omit it entirely, and
					// ForecastManager is presently one of those.
					if _, seen := counts[ts.Name.Name]; !seen {
						counts[ts.Name.Name] = 0
					}
				}
			case *ast.FuncDecl:
				if d.Recv == nil || !d.Name.IsExported() {
					continue
				}
				recv := d.Recv.List[0].Type
				var recvName string
				switch t := recv.(type) {
				case *ast.StarExpr:
					if ident, ok := t.X.(*ast.Ident); ok {
						recvName = ident.Name
					}
				case *ast.Ident:
					recvName = t.Name
				}
				if !strings.HasSuffix(recvName, "Manager") {
					continue
				}
				counts[recvName]++
			}
		}
	}
	return counts, nil
}

// extractDocCount returns the "N ops" count from the 03-managers.md row for one
// manager, and whether such a row exists at all. The two answers are kept distinct
// because a manager can legitimately be documented as 0 ops - ForecastManager
// currently is - and a bare int would report that identically to "not documented",
// which is precisely the skip this signature exists to delete.
//
// The name is matched in a table cell of its own, with the backticks the rest of
// the docs use written \x60 rather than a literal, since this pattern lives in a Go
// raw string. Requiring a bare unbackticked name matched no row in the file.
func extractDocCount(doc, manager string) (int, bool) {
	escaped := regexp.QuoteMeta(manager)
	re := regexp.MustCompile(`(?m)^\|\s*\x60?\s*` + escaped + `\s*\x60?\s*\|[^|]*\|\s*(\d+)\s*ops\s*\|`)
	m := re.FindStringSubmatch(doc)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return n, true
}
