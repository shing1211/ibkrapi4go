// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

// check_design_docs.go verifies that design documents accurately describe the codebase.
// It is run by `make docs-design-check`.
//
// Checks:
//   - The middleware chain in docs/design/01-transport.md matches internal/transport.go
//   - The manager method counts in docs/design/03-managers.md match actual exports
//
// Usage: go run scripts/check_design_docs.go
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	repoRoot, _ := os.Getwd()
	if err := run(repoRoot); err != nil {
		fmt.Fprintf(os.Stderr, "design doc check failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("design docs OK")
}

func run(repoRoot string) error {
	var errs []string

	if err := checkTransportChain(repoRoot); err != nil {
		errs = append(errs, err.Error())
	}

	if err := checkManagerCounts(repoRoot); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "\n"))
	}
	return nil
}

// checkTransportChain verifies that the middleware chain diagram in 01-transport.md
// matches the actual order in internal/transport.go NewClientTransport function.
func checkTransportChain(repoRoot string) error {
	transportPath := filepath.Join(repoRoot, "internal", "transport.go")
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

// checkManagerCounts verifies that method counts listed in 03-managers.md
// match the actual exported methods on each manager type.
func checkManagerCounts(repoRoot string) error {
	docPath := filepath.Join(repoRoot, "docs", "design", "03-managers.md")
	docSrc, err := os.ReadFile(docPath)
	if err != nil {
		return fmt.Errorf("docs/design/03-managers.md: %w", err)
	}

	// For each manager, count actual exported methods in the corresponding Go file.
	// Map manager names to their source files.
	managerFiles := map[string][]string{
		"AccountManager":        {"pkg/ibkr/account.go"},
		"PortfolioManager":      {"pkg/ibkr/portfolio.go"},
		"TradeManager":          {"pkg/ibkr/trade.go", "pkg/ibkr/contract.go"},
		"MarketDataManager":     {"pkg/ibkr/marketdata.go", "pkg/ibkr/ws.go"},
		"TradingAccountManager": {"pkg/ibkr/trading_accounts.go"},
		"AlertManager":          {"pkg/ibkr/alerts.go"},
		"ForecastManager":       {"pkg/ibkr/events.go"},
		"ScannerManager":        {"pkg/ibkr/scanner.go"},
		"AllocationManager":     {"pkg/ibkr/allocation.go"},
		"ModelManager":          {"pkg/ibkr/models.go"},
		"FYIManager":            {"pkg/ibkr/notifications.go"},
		"OAuthManager":          {"pkg/ibkr/oauth1.go"},
		"WatchlistManager":      {"pkg/ibkr/watchlists.go"},
		"PerformanceManager":    {"pkg/ibkr/performance.go"},
	}

	for manager, files := range managerFiles {
		actualCount := 0
		for _, file := range files {
			n, err := countManagerMethods(repoRoot, file, manager)
			if err != nil {
				continue // skip if file doesn't exist
			}
			actualCount += n
		}

		// Extract the documented count for this manager from the doc
		docCount := extractDocCount(string(docSrc), manager)
		if docCount == 0 {
			continue // not documented with a count
		}

		if actualCount != docCount {
			return fmt.Errorf("docs/design/03-managers.md: %s has %d methods in code but doc says %d", manager, actualCount, docCount)
		}
	}

	return nil
}

func countManagerMethods(repoRoot, file, managerType string) (int, error) {
	path := filepath.Join(repoRoot, file)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return 0, nil
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, path, src, 0)
	if err != nil {
		return 0, err
	}

	var count int
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil {
			continue
		}
		recv := fn.Recv.List[0].Type
		var name string
		switch t := recv.(type) {
		case *ast.StarExpr:
			if ident, ok := t.X.(*ast.Ident); ok {
				name = ident.Name
			}
		case *ast.Ident:
			name = t.Name
		}
		if name == managerType && fn.Name.IsExported() {
			count++
		}
	}
	return count, nil
}

func extractDocCount(doc string, manager string) int {
	escaped := strings.ReplaceAll(manager, "*", "\\*")
	re := regexp.MustCompile(`\|` + escaped + `\|[^|]*\|\s*(\d+)\s*ops\|`)
	m := re.FindStringSubmatch(doc)
	if m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	// No "N ops" — count backtick-quoted method names in the table row.
	// The manager is on a row like: | `Name` | scope | `Method1`, `Method2` |
	// Find the row by looking for the manager name (with optional surrounding space)
	// then extract the third pipe-delimited field.
	backtick := "\x60"
	pat := backtick + `\s*` + escaped + `\s*` + backtick
	loc := regexp.MustCompile(pat).FindStringIndex(doc)
	if loc == nil {
		// Try: backtick + space + name
		pat2 := backtick + `\s+` + escaped + `\s*` + backtick
		loc = regexp.MustCompile(pat2).FindStringIndex(doc)
	}
	if loc == nil {
		return 0
	}
	// Find the start of this row: go backward to the last | before loc[0]
	rowStart := 0
	for i := loc[0] - 1; i >= 0; i-- {
		if doc[i] == '|' {
			rowStart = i
			break
		}
	}
	// Find the end of the row: the next | after the third field
	row := doc[rowStart:]
	fields := strings.Split(row, "|")
	if len(fields) < 4 {
		return 0
	}
	methodCol := fields[3]
	count := len(regexp.MustCompile(backtick+"[^"+backtick+"]+"+backtick).FindAllString(methodCol, -1))
	return count
}
