// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

// errors_retries.go verifies docs/design/06-errors-retries.md against the code
// it describes. Every check here pins one load-bearing fact the document states
// as fact, and each reports the file:line on both sides so the failure names the
// line to edit.
//
// What is deliberately NOT checked: whole-document diffing. The RetryPolicy
// block's field *set* is pinned by name and type in both directions only for
// the fields the document lists; a field the document omits is reported in
// review rather than enforced here, because the defaults check has no claim to
// compare against. What a field's default is, by contrast, is enforced in both
// directions per field — see checkRetryPolicyDefaults, which now requires every
// documented field to say either what it defaults to or that it has no
// assigned default, and checks each claim against the code.
//
// A field the code declares but the document does not list remains a
// review-time finding, not a build failure. This program is not authorised to
// decide which side of that disagreement is authoritative.

import (
	"fmt"
	"go/ast"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	errDoc        = "docs/design/06-errors-retries.md"
	retrySrcPath  = "internal/retry.go"
	errorsSrcPath = "internal/errors.go"
	breakerPath   = "internal/breaker.go"
	transportPath = "internal/transport.go"
)

// httpMethodConsts maps the net/http method constants to the method strings the
// document writes. Resolved through a table rather than assumed, so a constant
// this checker has never seen is reported instead of silently skipped.
var httpMethodConsts = map[string]string{
	"MethodGet":     http.MethodGet,
	"MethodHead":    http.MethodHead,
	"MethodPost":    http.MethodPost,
	"MethodPut":     http.MethodPut,
	"MethodPatch":   http.MethodPatch,
	"MethodDelete":  http.MethodDelete,
	"MethodConnect": http.MethodConnect,
	"MethodOptions": http.MethodOptions,
	"MethodTrace":   http.MethodTrace,
}

// --- claim 1: the Error struct's field set ----------------------------------

// checkErrorStruct verifies that the `type Error struct` block in
// 06-errors-retries.md lists exactly the fields internal.Error declares, in the
// same order. docs/ERRORS.md and every `errors.As(err, &e); e.HTTPStatus` guard
// in the tree depend on that set, so a field added, removed or reordered is a
// public-contract change and must reach the document.
func checkErrorStruct(repoRoot string) error {
	doc, err := parseDoc(repoRoot, errDoc)
	if err != nil {
		return err
	}
	code, err := parseGo(repoRoot, errorsSrcPath)
	if err != nil {
		return err
	}

	docFields, err := doc.structFields("Error")
	if err != nil {
		return err
	}
	if len(docFields) == 0 {
		return fmt.Errorf("%s: the Error code block lists no fields; the field-set check cannot run", doc.rel)
	}
	realFields, err := code.structFields("Error")
	if err != nil {
		return err
	}

	if len(docFields) != len(realFields) {
		return fmt.Errorf("error struct drift: %s documents %d fields (%s) but %s declares %d (%s); "+
			"callers read this set as the error contract (docs/ERRORS.md)",
			doc.rel, len(docFields), joinDocFields(docFields), code.rel, len(realFields), joinCodeFields(realFields))
	}
	for i, df := range docFields {
		rf := realFields[i]
		if df.name != rf.name {
			return fmt.Errorf("error struct drift: %s documents field %d as %q but %s:%d declares %q; "+
				"the documented field set must match the declaration in name and order",
				doc.at(df.lineNo), i+1, df.name, code.rel, code.line(rf.node), rf.name)
		}
		if df.typ != rf.typ {
			return fmt.Errorf("error struct drift: %s documents field %s as %s but %s:%d declares %s",
				doc.at(df.lineNo), df.name, df.typ, code.rel, code.line(rf.node), rf.typ)
		}
	}
	return nil
}

// checkErrorTypeAlias verifies that pkg/ibkr exposes the internal struct as a
// type alias rather than redeclaring it. The document's struct block is the
// public contract; it only describes what callers see while the public name is
// an alias. A second, hand-maintained copy would drift silently.
func checkErrorTypeAlias(repoRoot string) error {
	code, err := parseGo(repoRoot, "pkg/ibkr/errors.go")
	if err != nil {
		return err
	}
	var spec *ast.TypeSpec
	for _, d := range code.file.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, s := range gd.Specs {
			ts, ok := s.(*ast.TypeSpec)
			if ok && ts.Name.Name == "Error" {
				spec = ts
			}
		}
	}
	if spec == nil {
		return fmt.Errorf("%s: no type named Error is declared; the alias check cannot run", code.rel)
	}
	if !spec.Assign.IsValid() {
		return fmt.Errorf("%s: `type Error = ...` is not an alias, so %s and the public ibkr.Error are "+
			"separate types and the struct documented in %s is no longer the public contract",
			code.at(spec), errorsSrcPath, errDoc)
	}
	sel, ok := spec.Type.(*ast.SelectorExpr)
	if !ok {
		return fmt.Errorf("%s: `type Error = %s` does not alias the %s type declared in %s",
			code.at(spec), exprString(spec.Type), "internal.Error", errorsSrcPath)
	}
	if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "internal" || sel.Sel.Name != "Error" {
		return fmt.Errorf("%s: `type Error = %s` does not alias %s, the type %s documents",
			code.at(spec), exprString(spec.Type), "internal.Error", errDoc)
	}
	return nil
}

// --- claim 2: the RetryPolicy defaults ---------------------------------------

// checkRetryPolicyDefaults verifies the default each field of the document's
// RetryPolicy block claims. A field's comment makes exactly one of two claims,
// and each is checked against the code:
//
//   - `// default X` — DefaultRetryPolicy must assign that field the value X.
//     A field the document defaults but the code never assigns is drift, and so
//     is one whose assigned value differs from X.
//   - A no-default marker (`// optional;` — see noDefaultMarkerRe) —
//     DefaultRetryPolicy must not assign the field at all. Its zero value is
//     then what an unconfigured policy gets. This is a real design choice, not
//     a gap: `Metrics` is never assigned because nil is its zero value and a
//     nil Metrics is a valid no-op, so insisting on an assignment would be
//     insisting on production code bending to satisfy a checker.
//
// The claim is checked in both directions, so the marker cannot become a
// silent exemption: if the code later starts assigning the field, the document
// now understates the policy and the check says so.
//
// A documented field that claims neither is a failure. The check does not
// infer a default from a field having none, and it does not exempt a comment
// it cannot read: a malformed marker, a comment with no `default` clause and a
// missing comment all fail.
//
// Each default is pinned on its own so a failure names the one constant that
// moved, not "the retry policy". The second place the BaseDelay default is
// applied is pinned by checkBaseDelayFallback.
func checkRetryPolicyDefaults(repoRoot string) error {
	doc, err := parseDoc(repoRoot, errDoc)
	if err != nil {
		return err
	}
	code, err := parseGo(repoRoot, retrySrcPath)
	if err != nil {
		return err
	}

	docFields, err := doc.structFields("RetryPolicy")
	if err != nil {
		return err
	}
	if len(docFields) == 0 {
		return fmt.Errorf("%s: the RetryPolicy code block lists no fields; the defaults check cannot run", doc.rel)
	}
	realFields, err := code.structFields("RetryPolicy")
	if err != nil {
		return err
	}
	realByName := map[string]codeField{}
	for _, rf := range realFields {
		if rf.name != "" {
			realByName[rf.name] = rf
		}
	}

	assigned, err := defaultRetryValues(code)
	if err != nil {
		return err
	}

	for _, df := range docFields {
		rf, ok := realByName[df.name]
		if !ok {
			return fmt.Errorf("retry policy drift: %s documents field %q, which %s does not declare",
				doc.at(df.lineNo), df.name, retrySrcPath)
		}
		if df.typ != rf.typ {
			return fmt.Errorf("retry policy drift: %s documents field %s as %s but %s:%d declares %s",
				doc.at(df.lineNo), df.name, df.typ, retrySrcPath, code.line(rf.node), rf.typ)
		}
		av, isAssigned := assigned[df.name]
		// The no-default marker is tested first because its own words contain
		// "default", so a `// no default: ...` comment also matches
		// defaultCommentRe. Testing it second would read that comment as a
		// default claim whose value is "nil is fine".
		if claimsNoDefault(df.comm) {
			// The claim is that the zero value is correct, so the code side of
			// it is an absence. An assignment here is drift in the other
			// direction: the document now understates the policy.
			if isAssigned {
				return fmt.Errorf("retry policy drift: %s marks %s as having no assigned default, but %s:%d "+
					"assigns %s to it; document the default or drop the marker",
					doc.at(df.lineNo), df.name, retrySrcPath, code.line(av.node), exprString(av.expr))
			}
			continue
		}
		if !isAssigned {
			if !claimsDefault(df.comm) {
				return fmt.Errorf("retry policy drift: %s documents %s with neither a `default` clause nor a "+
					"no-default marker (comment %q); say what the field defaults to, or prefix the comment with "+
					"`optional;` if its zero value is the intended value",
					doc.at(df.lineNo), df.name, df.comm)
			}
			return fmt.Errorf("retry policy drift: %s documents a default for %s but %s:DefaultRetryPolicy "+
				"never assigns it, so a zero RetryPolicy falls back to some other value",
				doc.at(df.lineNo), df.name, retrySrcPath)
		}
		want, err := canonDocValue(df.typ, df.comm, doc, df)
		if err != nil {
			return err
		}
		got, err := canonCodeValue(df.typ, av.expr)
		if err != nil {
			return fmt.Errorf("retry policy drift: cannot read the value %s:%d assigns to %s (%s): %w",
				retrySrcPath, code.line(av.node), df.name, exprString(av.expr), err)
		}
		if want != got {
			return fmt.Errorf("retry policy drift: %s documents %s default as %q but %s:%d assigns %q",
				doc.at(df.lineNo), df.name, want, retrySrcPath, code.line(av.node), got)
		}
	}

	return checkBaseDelayFallback(doc, code, docFields)
}

// defaultRetryValues reads the RetryPolicy literal returned by
// DefaultRetryPolicy. The literal's type is checked before its keys are read,
// so a returned `SomeOther{...}` cannot pass as the policy.
func defaultRetryValues(code *goSrc) (map[string]keyedValue, error) {
	fn := code.funcDecl("DefaultRetryPolicy")
	if fn == nil {
		return nil, fmt.Errorf("%s: no DefaultRetryPolicy function is declared; the defaults check cannot run", retrySrcPath)
	}
	for _, stmt := range fn.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			continue
		}
		lit, ok := ret.Results[0].(*ast.CompositeLit)
		if !ok {
			continue
		}
		if got := structLitType(lit); got != "RetryPolicy" {
			return nil, fmt.Errorf("%s: DefaultRetryPolicy returns a %s literal, not a RetryPolicy literal",
				retrySrcPath, got)
		}
		out := map[string]keyedValue{}
		for _, kv := range keyedValues(lit) {
			out[kv.key] = kv
		}
		return out, nil
	}
	return nil, fmt.Errorf("%s: DefaultRetryPolicy does not return a RetryPolicy literal; "+
		"the defaults check cannot run", retrySrcPath)
}

// checkBaseDelayFallback verifies the second place BaseDelay's default is
// applied. delayFor substitutes 200ms for a zero BaseDelay, so a caller that
// builds RetryPolicy{MaxAttempts: 3} by hand gets the documented 200ms even
// without DefaultRetryPolicy. Two copies of the constant is a drift risk: this
// pins them to the same value as the documented default.
func checkBaseDelayFallback(doc *docFile, code *goSrc, docFields []docField) error {
	var docDefault time.Duration
	var haveDefault bool
	for _, df := range docFields {
		if df.name != "BaseDelay" {
			continue
		}
		s, err := canonDocValue(df.typ, df.comm, doc, df)
		if err != nil {
			return err
		}
		d, err := time.ParseDuration(s)
		if err != nil {
			return fmt.Errorf("%s: documented BaseDelay default %q is not a duration: %w", doc.at(df.lineNo), s, err)
		}
		docDefault, haveDefault = d, true
	}
	if !haveDefault {
		return fmt.Errorf("%s: the RetryPolicy block does not document a BaseDelay default; the check cannot run", doc.rel)
	}

	fn := code.method("RetryPolicy", "delayFor")
	if fn == nil {
		return fmt.Errorf("%s: no RetryPolicy.delayFor method is declared; the check cannot run", retrySrcPath)
	}

	// The identifier the fallback assigns is resolved from the declaration it
	// was initialized from (`base := p.BaseDelay`), not assumed to be "base".
	src := baseDelayParam(fn)
	if src == "" {
		return fmt.Errorf("%s:%d: delayFor does not read p.BaseDelay into a local; the check cannot run",
			retrySrcPath, code.line(fn))
	}
	for _, stmt := range fn.Body.List {
		guard, ok := stmt.(*ast.IfStmt)
		if !ok || !isIntLEZero(guard.Cond, src) {
			continue
		}
		for _, inner := range guard.Body.List {
			as, ok := inner.(*ast.AssignStmt)
			if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
				continue
			}
			if id, ok := as.Lhs[0].(*ast.Ident); !ok || id.Name != src {
				continue
			}
			got, ok := durationValue(as.Rhs[0])
			if !ok {
				return fmt.Errorf("%s:%d: the BaseDelay fallback assigns %s, which is not a duration literal",
					retrySrcPath, code.line(as), exprString(as.Rhs[0]))
			}
			if got != docDefault {
				return fmt.Errorf("retry policy drift: %s documents BaseDelay default as %s but the fallback "+
					"for a zero BaseDelay at %s:%d uses %s; a hand-built RetryPolicy would not honour the documented default",
					doc.rel, docDefault, retrySrcPath, code.line(as), got)
			}
			return nil
		}
	}
	return fmt.Errorf("%s: delayFor has no `if %s <= 0` fallback assigning a duration to %s; "+
		"a zero BaseDelay would no longer mean the documented %s",
		retrySrcPath, src, src, docDefault)
}

// baseDelayParam returns the name of the local that delayFor initializes from
// p.BaseDelay, or "" if there is no such local.
func baseDelayParam(fn *ast.FuncDecl) string {
	for _, stmt := range fn.Body.List {
		as, ok := stmt.(*ast.AssignStmt)
		if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
			continue
		}
		sel, ok := as.Rhs[0].(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "BaseDelay" {
			continue
		}
		if recv, ok := sel.X.(*ast.Ident); !ok || recv.Name != receiverName(fn) {
			continue
		}
		if id, ok := as.Lhs[0].(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

func receiverName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	if len(fn.Recv.List[0].Names) == 0 {
		return ""
	}
	return fn.Recv.List[0].Names[0].Name
}

// isIntLEZero reports whether cond is `<ident> <= 0`.
func isIntLEZero(cond ast.Expr, ident string) bool {
	b, ok := cond.(*ast.BinaryExpr)
	if !ok || b.Op != token.LEQ {
		return false
	}
	l, ok := b.X.(*ast.Ident)
	if !ok || l.Name != ident {
		return false
	}
	v, ok := intValue(b.Y)
	return ok && v == 0
}

// --- claim 3: the safe-method set -------------------------------------------

// safeMethodSet re-reads isSafeMethod from the source rather than hardcoding
// GET/HEAD/OPTIONS, so checkSafeMethods compares the document against the code
// and checkOrderMutationsNotRetried uses the same list. One reader, so the two
// checks cannot disagree about what "safe" means.
func safeMethodSet(code *goSrc) ([]string, bool, string) {
	fn := code.funcDecl("isSafeMethod")
	if fn == nil {
		return nil, false, fmt.Sprintf("%s: no isSafeMethod function is declared; the check cannot run", retrySrcPath)
	}
	// The switch must be on the method parameter, resolved from the signature.
	param := paramOfType(fn.Type, "string")
	if param == "" {
		return nil, false, fmt.Sprintf("%s: isSafeMethod takes no string parameter; the check cannot run", retrySrcPath)
	}
	var sw *ast.SwitchStmt
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if s, ok := n.(*ast.SwitchStmt); ok && sw == nil {
			sw = s
		}
		return sw == nil
	})
	if sw == nil {
		return nil, false, fmt.Sprintf("%s: isSafeMethod has no switch statement; the check cannot run", retrySrcPath)
	}
	if tag, ok := sw.Tag.(*ast.Ident); !ok || tag.Name != param {
		return nil, false, fmt.Sprintf("%s: the switch in isSafeMethod does not test the %s parameter; the check cannot run",
			retrySrcPath, param)
	}
	var out []string
	hasDefault := false
	for _, c := range sw.Body.List {
		cc, ok := c.(*ast.CaseClause)
		if !ok {
			continue
		}
		if cc.List == nil {
			hasDefault = true
			continue
		}
		for _, e := range cc.List {
			sel, ok := e.(*ast.SelectorExpr)
			if !ok {
				return nil, false, fmt.Sprintf("%s: isSafeMethod has a case that is not an http.MethodX constant; the check cannot run",
					retrySrcPath)
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "http" {
				return nil, false, fmt.Sprintf("%s: isSafeMethod has a case that does not come from net/http; the check cannot run",
					retrySrcPath)
			}
			m, ok := httpMethodConsts[sel.Sel.Name]
			if !ok {
				return nil, false, fmt.Sprintf("%s: isSafeMethod references http.%s, which is not a method constant this checker knows",
					retrySrcPath, sel.Sel.Name)
			}
			out = append(out, m)
		}
	}
	return out, hasDefault, ""
}

// checkSafeMethods verifies rule 1 of the document ("Safe methods only: GET,
// HEAD, OPTIONS") against the cases isSafeMethod accepts, in order.
func checkSafeMethods(repoRoot string) error {
	doc, err := parseDoc(repoRoot, errDoc)
	if err != nil {
		return err
	}
	code, err := parseGo(repoRoot, retrySrcPath)
	if err != nil {
		return err
	}
	actual, hasDefault, fatal := safeMethodSet(code)
	if fatal != "" {
		return fmt.Errorf("%s", fatal)
	}
	if len(actual) == 0 {
		return fmt.Errorf("%s: isSafeMethod accepts no method; the safe-method check cannot run", retrySrcPath)
	}
	// Without a default clause the whitelist would stop being one. Go would
	// reject a switch with no fallthrough return, but a future refactor to a
	// lookup map would not, so the shape is pinned.
	if !hasDefault {
		return fmt.Errorf("%s: isSafeMethod has no default clause, so it is not a whitelist; "+
			"%s rule 1 claims only %v are safe",
			retrySrcPath, errDoc, actual)
	}

	no, want, err := docSafeMethods(doc)
	if err != nil {
		return err
	}
	if !equalStrings(want, actual) {
		return fmt.Errorf("safe-method drift: %s documents safe methods as %s but %s:isSafeMethod accepts %s; "+
			"%s rule 1 and ADR 0009 both rest on this set",
			doc.at(no), strings.Join(want, ", "), retrySrcPath, strings.Join(actual, ", "), errDoc)
	}
	return nil
}

var safeMethodsRe = regexp.MustCompile(`^\s*\d+\.\s+\*\*Safe methods only\*\*:\s*(.+?)\s*$`)

// docSafeMethods reads the method list out of the document's rule 1.
func docSafeMethods(doc *docFile) (int, []string, error) {
	for i, l := range doc.lines {
		m := safeMethodsRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		var out []string
		for _, tok := range regexp.MustCompile("`([^`]+)`").FindAllStringSubmatch(m[1], -1) {
			out = append(out, strings.TrimSpace(tok[1]))
		}
		if len(out) == 0 {
			return 0, nil, fmt.Errorf("%s:%d: the safe-method rule lists no backtick-quoted methods; the check cannot run",
				doc.rel, i+1)
		}
		return i + 1, out, nil
	}
	return 0, nil, fmt.Errorf("%s: no numbered `**Safe methods only**` rule found; the check cannot run", doc.rel)
}

// --- claim 4: order/instruction mutations are never retried (ADR 0009) -------

// checkOrderMutationsNotRetried verifies the mechanism behind the document's
// rule 2, not the sentence. Two halves:
//
//   - The Retry middleware's first statement must be a guard that returns the
//     base transport when `!isSafeMethod(req.Method)`. Without this, isSafeMethod
//     could stay correct while the middleware stops consulting it, and the
//     document's "POST, 500 -> 1 attempt" row would quietly become false.
//   - TradeManager.mutate must convert a netDo failure into the `ambiguous`
//     reconcile error the document's "Ambiguous outcomes" section promises.
//
// The runtime half (a POST really is attempted once) is already covered by
// internal/retry_test.go:82 and internal/fault_injection_test.go:389. What is
// not covered anywhere is the document agreeing with the code, which is what
// this checks.
func checkOrderMutationsNotRetried(repoRoot string) error {
	if err := checkRetryGateIsWhitelist(repoRoot); err != nil {
		return err
	}
	return checkAmbiguousReconcileError(repoRoot)
}

func checkRetryGateIsWhitelist(repoRoot string) error {
	code, err := parseGo(repoRoot, retrySrcPath)
	if err != nil {
		return err
	}
	fn := code.funcDecl("Retry")
	if fn == nil {
		return fmt.Errorf("%s: no Retry middleware is declared; the no-auto-retry check cannot run", retrySrcPath)
	}
	lit := roundTripFuncLit(fn)
	if lit == nil {
		return fmt.Errorf("%s: the Retry middleware builds no RoundTripFunc; the no-auto-retry check cannot run", retrySrcPath)
	}
	reqParam := paramOfType(lit.Type, "*http.Request")
	if reqParam == "" {
		return fmt.Errorf("%s: the Retry RoundTripFunc takes no *http.Request; the no-auto-retry check cannot run",
			retrySrcPath)
	}
	// base is the base transport the guard hands an unsafe request to. It is
	// the parameter of the enclosing middleware closure, not of the
	// RoundTripFunc, so it is resolved from whichever literal declares an
	// http.RoundTripper parameter.
	baseParam := ""
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		fl, ok := n.(*ast.FuncLit)
		if !ok {
			return baseParam == ""
		}
		if p := paramOfType(fl.Type, "http.RoundTripper"); p != "" {
			baseParam = p
			return false
		}
		return true
	})
	if baseParam == "" {
		return fmt.Errorf("%s: no middleware closure in Retry takes an http.RoundTripper; "+
			"the no-auto-retry check cannot run", retrySrcPath)
	}

	if len(lit.Body.List) == 0 {
		return fmt.Errorf("%s: the Retry RoundTripFunc is empty; the no-auto-retry check cannot run", retrySrcPath)
	}
	guard, ok := lit.Body.List[0].(*ast.IfStmt)
	if !ok {
		return fmt.Errorf("%s: the first statement of the Retry middleware is a %T, not the safe-method guard; "+
			"%s rule 2 and ADR 0009 rely on that guard",
			retrySrcPath, lit.Body.List[0], errDoc)
	}
	if !returnsBaseRoundTrip(guard.Body, baseParam, reqParam) {
		return fmt.Errorf("%s: the safe-method guard at %s does not return %s.RoundTrip(%s); "+
			"an unsafe method would proceed to the retry loop",
			retrySrcPath, code.at(guard), baseParam, reqParam)
	}
	if !guardDeniesUnsafe(guard.Cond, reqParam) {
		return fmt.Errorf("%s: the guard at %s does not test !isSafeMethod(%s.Method); "+
			"the retry middleware no longer implements %s rule 2 (order/instruction mutations never retried)",
			retrySrcPath, code.at(guard), reqParam, errDoc)
	}
	return nil
}

func checkAmbiguousReconcileError(repoRoot string) error {
	trade, err := parseGo(repoRoot, "pkg/ibkr/trade.go")
	if err != nil {
		return err
	}
	fn := trade.method("TradeManager", "mutate")
	if fn == nil {
		return fmt.Errorf("%s: no TradeManager.mutate method is declared; the reconcile-error check cannot run",
			"pkg/ibkr/trade.go")
	}
	if !callsNetDo(fn) {
		return fmt.Errorf("pkg/ibkr/trade.go: TradeManager.mutate (%s) does not go through Client.netDo; "+
			"%s describes order mutations as a single attempt that reports an ambiguous outcome",
			trade.at(fn), errDoc)
	}
	var lit *ast.CompositeLit
	for _, stmt := range fn.Body.List {
		g, ok := stmt.(*ast.IfStmt)
		if !ok {
			continue
		}
		ast.Inspect(g.Body, func(n ast.Node) bool {
			cl, ok := n.(*ast.CompositeLit)
			if ok && structLitType(cl) == "Error" && lit == nil {
				lit = cl
			}
			return lit == nil
		})
		if lit != nil {
			break
		}
	}
	if lit == nil {
		return fmt.Errorf("pkg/ibkr/trade.go: TradeManager.mutate (%s) never constructs an Error for a failed "+
			"mutation; %s promises a distinct error instructing the caller to reconcile",
			trade.at(fn), errDoc)
	}
	fields := map[string]keyedValue{}
	for _, kv := range keyedValues(lit) {
		fields[kv.key] = kv
	}
	codeKV, ok := fields["Code"]
	if !ok {
		return fmt.Errorf("pkg/ibkr/trade.go:%d: the reconcile error sets no Code; %s and docs/ERRORS.md both "+
			"document Code == \"ambiguous\" for an ambiguous order outcome",
			trade.line(lit), errDoc)
	}
	code, ok := stringLit(codeKV.expr)
	if !ok {
		return fmt.Errorf("pkg/ibkr/trade.go:%d: the reconcile error's Code is %s, not a string literal",
			trade.line(codeKV.node), exprString(codeKV.expr))
	}
	if code != "ambiguous" {
		return fmt.Errorf("pkg/ibkr/trade.go:%d: the reconcile error's Code is %q but %s and docs/ERRORS.md "+
			"document %q; callers match on it to tell an ambiguous outcome apart",
			trade.line(codeKV.node), code, errDoc, "ambiguous")
	}
	msgKV, ok := fields["Message"]
	if !ok {
		return fmt.Errorf("pkg/ibkr/trade.go:%d: the reconcile error sets no Message; %s requires the message to "+
			"instruct the caller to reconcile", trade.line(lit), errDoc)
	}
	msg, ok := stringLit(msgKV.expr)
	if !ok {
		return fmt.Errorf("pkg/ibkr/trade.go:%d: the reconcile error's Message is not a string literal",
			trade.line(msgKV.node))
	}
	if !strings.Contains(msg, "reconcile") {
		return fmt.Errorf("%s: the reconcile error's Message (%s) does not tell the caller to reconcile; "+
			"%s states the SDK \"returns a distinct error instructing the caller to reconcile\"",
			trade.at(msgKV.node), msg, errDoc)
	}
	return nil
}

// --- claim 5: the circuit breaker is disabled by default --------------------

// checkBreakerDisabledByDefault verifies the three independent things that
// together make "Optional, disabled by default" true: a non-positive threshold
// yields no breaker, the transport installs the middleware only for a non-nil
// breaker, and nothing in pkg/ibkr sets a breaker except the opt-in options.
func checkBreakerDisabledByDefault(repoRoot string) error {
	if err := checkBreakerNilOnNonPositiveThreshold(repoRoot); err != nil {
		return err
	}
	if err := checkBreakerMiddlewareIsConditional(repoRoot); err != nil {
		return err
	}
	return checkNoImplicitBreakerDefault(repoRoot)
}

func checkBreakerNilOnNonPositiveThreshold(repoRoot string) error {
	code, err := parseGo(repoRoot, breakerPath)
	if err != nil {
		return err
	}
	fn := code.funcDecl("NewBreaker")
	if fn == nil {
		return fmt.Errorf("%s: no NewBreaker function is declared; the disabled-by-default check cannot run", breakerPath)
	}
	// The threshold identifier is resolved from the signature.
	param := paramOfType(fn.Type, "int")
	if param == "" {
		return fmt.Errorf("%s: NewBreaker takes no int parameter; the disabled-by-default check cannot run", breakerPath)
	}
	for _, stmt := range fn.Body.List {
		guard, ok := stmt.(*ast.IfStmt)
		if !ok || !isIntLEZero(guard.Cond, param) {
			continue
		}
		for _, inner := range guard.Body.List {
			ret, ok := inner.(*ast.ReturnStmt)
			if !ok || len(ret.Results) != 1 {
				continue
			}
			if id, ok := ret.Results[0].(*ast.Ident); ok && id.Name == "nil" {
				return nil
			}
		}
		return fmt.Errorf("%s: the `if %s <= 0` guard at %s does not return nil; %s documents the breaker as "+
			"disabled by default, which a non-positive threshold must produce",
			breakerPath, param, code.at(guard), errDoc)
	}
	return fmt.Errorf("%s: NewBreaker has no `if %s <= 0 { return nil }` guard; %s documents the breaker as "+
		"disabled by default and a non-positive threshold is how a caller asks for no breaker",
		breakerPath, param, errDoc)
}

func checkBreakerMiddlewareIsConditional(repoRoot string) error {
	code, err := parseGo(repoRoot, transportPath)
	if err != nil {
		return err
	}
	fn := code.funcDecl("NewClientTransport")
	if fn == nil {
		return fmt.Errorf("%s: no NewClientTransport function is declared; the disabled-by-default check cannot run",
			transportPath)
	}
	cfgParam := paramOfType(fn.Type, "TransportConfig")
	if cfgParam == "" {
		return fmt.Errorf("%s: NewClientTransport takes no TransportConfig parameter; the disabled-by-default check cannot run",
			transportPath)
	}
	// Three outcomes, each with its own message: the middleware is gone, it is
	// installed unconditionally, or it is installed under some other condition.
	// Collapsing them would describe an unconditional install as "no longer
	// installs", which is the opposite of what happened.
	var anywhere bool
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) < 1 {
			return true
		}
		if calleeName(call.Fun) == "CircuitBreaker" {
			if arg, ok := call.Args[0].(*ast.SelectorExpr); ok && arg.Sel.Name == "Breaker" {
				anywhere = true
			}
		}
		return true
	})
	if !anywhere {
		return fmt.Errorf("%s: NewClientTransport no longer installs the CircuitBreaker middleware; "+
			"%s documents it as an available layer", transportPath, errDoc)
	}
	for _, stmt := range fn.Body.List {
		guard, ok := stmt.(*ast.IfStmt)
		if !ok || !guardInstalls(guard, "CircuitBreaker", cfgParam, "Breaker") {
			continue
		}
		if isNotNilField(guard.Cond, cfgParam, "Breaker") {
			return nil
		}
		return fmt.Errorf("%s:%d: the CircuitBreaker middleware is installed under %s, not a nil check on "+
			"%s.%s; %s documents the breaker as optional and disabled by default, so a nil breaker must be "+
			"the only way the layer is skipped",
			transportPath, code.line(guard.Cond), exprString(guard.Cond), cfgParam, "Breaker", errDoc)
	}
	return fmt.Errorf("%s: the CircuitBreaker middleware is installed unconditionally; %s documents the "+
		"breaker as optional and disabled by default, so `if %s.Breaker != nil` must guard it",
		transportPath, errDoc, cfgParam)
}

// guardInstalls reports whether the guarded block appends a call to mw
// configured from <cfgParam>.<cfgField>. The append is looked for anywhere in
// the block rather than at a fixed argument position: the chain is assembled
// as `ms = append(ms, mw(cfg.X))`, so the middleware call is not the append's
// first argument. The callee is matched on its final name, which covers both
// `mw(cfg.X)` and `pkg.Mw(cfg.X)`.
func guardInstalls(guard *ast.IfStmt, mw, cfgParam, cfgField string) bool {
	found := false
	ast.Inspect(guard.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) < 1 || calleeName(call.Fun) != mw {
			return true
		}
		arg, ok := call.Args[0].(*ast.SelectorExpr)
		if !ok || arg.Sel.Name != cfgField {
			return true
		}
		if id, ok := arg.X.(*ast.Ident); ok && id.Name == cfgParam {
			found = true
		}
		return true
	})
	return found
}

// calleeName returns the final name of a call target: "Mw" for both `mw(...)`
// and `pkg.Mw(...)`.
func calleeName(fun ast.Expr) string {
	switch t := fun.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	case *ast.IndexExpr:
		return calleeName(t.X)
	}
	return ""
}

// isNotNilField reports whether cond is `<ident>.<field> != nil`.
func isNotNilField(cond ast.Expr, ident, field string) bool {
	b, ok := cond.(*ast.BinaryExpr)
	if !ok || b.Op != token.NEQ {
		return false
	}
	sel, ok := b.X.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != field {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	if !ok || id.Name != ident {
		return false
	}
	n, ok := b.Y.(*ast.Ident)
	return ok && n.Name == "nil"
}

// checkNoImplicitBreakerDefault verifies that no pkg/ibkr code other than the
// opt-in WithCircuitBreaker options writes the config's breaker field. A
// default threshold set anywhere else would switch the breaker on for every
// client, which is the opposite of what the document says and would not fail a
// test that only covers the explicit option.
func checkNoImplicitBreakerDefault(repoRoot string) error {
	dir := filepath.Join(repoRoot, "pkg", "ibkr")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("pkg/ibkr: %w", err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		code, err := parseGo(repoRoot, "pkg/ibkr/"+name)
		if err != nil {
			return err
		}
		for _, d := range code.file.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			// Declaring the field is not setting it; a struct FieldDecl is
			// therefore not a writer and is skipped by inspecting bodies only.
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch s := n.(type) {
				case *ast.AssignStmt:
					for _, lhs := range s.Lhs {
						if fieldIdent(lhs) == "breaker" {
							err = breakerWriteError(code, fn, fieldIdent(lhs))
							return false
						}
					}
				case *ast.KeyValueExpr:
					if fieldIdent(s.Key) == "breaker" {
						err = breakerWriteError(code, fn, "breaker")
						return false
					}
				}
				return err == nil
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func breakerWriteError(code *goSrc, fn *ast.FuncDecl, name string) error {
	if strings.HasPrefix(fn.Name.Name, "WithCircuitBreaker") {
		return nil
	}
	return fmt.Errorf("%s: %s writes the client config's %q field, but %s documents the circuit breaker as "+
		"\"Optional, disabled by default\"; only the opt-in WithCircuitBreaker options may set it",
		code.at(fn.Body), fn.Name.Name, name, errDoc)
}

// fieldIdent returns the field name an assignment target or composite literal
// key selects, or "".
func fieldIdent(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	}
	return ""
}

// --- shared helpers ---------------------------------------------------------

// canonDocValue canonicalizes a documented `// default X` value using the type
// the document itself declares, so "200ms" and a coded 200*time.Millisecond
// become the same string. Reading the value through the declared type is what
// keeps "3" and "200ms" from both being treated as opaque text.
func canonDocValue(typ, comm string, doc *docFile, df docField) (string, error) {
	raw, err := defaultComment(comm)
	if err != nil {
		return "", fmt.Errorf("%s: field %s has no readable `default` comment: %w", doc.at(df.lineNo), df.name, err)
	}
	switch typ {
	case "int":
		f := strings.Fields(raw)
		if len(f) == 0 {
			return "", fmt.Errorf("%s: empty default for int field %s", doc.at(df.lineNo), df.name)
		}
		v, err := strconv.Atoi(strings.TrimSuffix(f[0], ","))
		if err != nil {
			return "", fmt.Errorf("%s: documented default %q for int field %s is not an integer", doc.at(df.lineNo), raw, df.name)
		}
		return strconv.Itoa(v), nil
	case "bool":
		f := strings.Fields(raw)
		if len(f) == 0 || (f[0] != "true" && f[0] != "false") {
			return "", fmt.Errorf("%s: documented default %q for bool field %s is not true/false",
				doc.at(df.lineNo), raw, df.name)
		}
		return f[0], nil
	case "time.Duration":
		d, err := time.ParseDuration(strings.TrimSpace(raw))
		if err != nil {
			return "", fmt.Errorf("%s: documented default %q for %s is not a duration", doc.at(df.lineNo), raw, df.name)
		}
		return d.String(), nil
	case "[]int":
		var out []int
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			v, err := strconv.Atoi(part)
			if err != nil {
				return "", fmt.Errorf("%s: documented default %q for %s is not a list of integers",
					doc.at(df.lineNo), raw, df.name)
			}
			out = append(out, v)
		}
		if len(out) == 0 {
			return "", fmt.Errorf("%s: empty documented default for %s", doc.at(df.lineNo), df.name)
		}
		return fmt.Sprint(out), nil
	}
	return "", fmt.Errorf("%s: no canonicalizer for documented type %q of field %s; the check cannot run",
		doc.at(df.lineNo), typ, df.name)
}

// defaultCommentRe recognizes a `// default X` clause anywhere in a field's
// comment. The `[: ]` separator is part of the clause, so "default 200ms" and
// "default: 200ms" are both a claim and "defaults are applied per attempt" is
// not.
var defaultCommentRe = regexp.MustCompile(`(?i)\bdefault\b[: ]\s*(.+)$`)

// noDefaultMarkerRe recognizes the clause a document author writes to state that
// a field has no assigned default, so that its zero value is what an
// unconfigured policy gets. `// optional;` is the form 06-errors-retries.md
// uses for `Metrics`; `// no assigned default;` and `// no default:` are the
// explicit spellings.
//
// Two properties of this pattern are load-bearing, and both exist so a comment
// cannot opt itself out of checking by accident:
//
//   - It is anchored to the start of the comment. A comment that mentions
//     defaults somewhere in its prose (`// see DefaultRetryPolicy`) is not a
//     marker, so the field still has to say what it defaults to.
//   - The accepted words are a closed list and each must be followed by a
//     separator. A near-miss (`optionall;`, a bare `optional`) does not match,
//     so it falls through to the "claims neither" failure instead of silently
//     exempting the field.
//
// The marker is only half the check: claimsNoDefault is paired with an assertion
// that DefaultRetryPolicy assigns the field nothing, so a marker that stops
// being true fails in the other direction.
var noDefaultMarkerRe = regexp.MustCompile(`(?i)^(?:no assigned default|no default|optional)\s*[:;-]`)

// claimsNoDefault reports whether a field's comment carries the no-default
// marker. The comment arrives already trimmed, so the anchor sees the first
// clause rather than leading whitespace.
func claimsNoDefault(comm string) bool { return noDefaultMarkerRe.MatchString(comm) }

// claimsDefault reports whether a field's comment carries a `default` clause.
func claimsDefault(comm string) bool { return defaultCommentRe.MatchString(comm) }

func defaultComment(comm string) (string, error) {
	if comm == "" {
		return "", fmt.Errorf("field carries no comment")
	}
	m := defaultCommentRe.FindStringSubmatch(comm)
	if m == nil {
		return "", fmt.Errorf("comment %q has no `default` clause", comm)
	}
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(m[1]), ".")), nil
}

// canonCodeValue canonicalizes a coded value using the same canonical form as
// canonDocValue, so the two sides are directly comparable.
func canonCodeValue(typ string, e ast.Expr) (string, error) {
	switch typ {
	case "int":
		v, ok := intValue(e)
		if !ok {
			return "", fmt.Errorf("%s is not an integer literal", exprString(e))
		}
		return strconv.Itoa(v), nil
	case "bool":
		v, ok := boolValue(e)
		if !ok {
			return "", fmt.Errorf("%s is not a bool literal", exprString(e))
		}
		return v, nil
	case "time.Duration":
		d, ok := durationValue(e)
		if !ok {
			return "", fmt.Errorf("%s is not a duration literal", exprString(e))
		}
		return d.String(), nil
	case "[]int":
		v, ok := intSliceValue(e)
		if !ok {
			return "", fmt.Errorf("%s is not an []int literal", exprString(e))
		}
		return fmt.Sprint(v), nil
	}
	return "", fmt.Errorf("no canonicalizer for type %q", typ)
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func joinDocFields(fs []docField) string {
	out := make([]string, 0, len(fs))
	for _, f := range fs {
		out = append(out, f.describe())
	}
	return strings.Join(out, ", ")
}

func joinCodeFields(fs []codeField) string {
	out := make([]string, 0, len(fs))
	for _, f := range fs {
		out = append(out, f.describe())
	}
	return strings.Join(out, ", ")
}

// roundTripFuncLit returns the function literal handed to RoundTripFunc inside
// fn. Resolving the RoundTripFunc argument rather than taking the first literal
// in the body matters: a middleware is typically an outer
// `func(base http.RoundTripper) ...` closure wrapping the inner
// `func(req *http.Request) ...`, and the first literal is the outer one.
func roundTripFuncLit(fn *ast.FuncDecl) *ast.FuncLit {
	var lit *ast.FuncLit
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || lit != nil {
			return lit == nil
		}
		id, ok := call.Fun.(*ast.Ident)
		if !ok || id.Name != "RoundTripFunc" || len(call.Args) != 1 {
			return true
		}
		fl, ok := call.Args[0].(*ast.FuncLit)
		if !ok {
			return true
		}
		lit = fl
		return false
	})
	return lit
}

// returnsBaseRoundTrip reports whether the block's last statement returns
// base.RoundTrip(req).
func returnsBaseRoundTrip(body *ast.BlockStmt, base, req string) bool {
	var ret *ast.ReturnStmt
	for _, stmt := range body.List {
		if r, ok := stmt.(*ast.ReturnStmt); ok {
			ret = r
		}
	}
	if ret == nil || len(ret.Results) != 1 {
		return false
	}
	call, ok := ret.Results[0].(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "RoundTrip" {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	if !ok || id.Name != base {
		return false
	}
	arg, ok := call.Args[0].(*ast.Ident)
	return ok && arg.Name == req
}

// guardDeniesUnsafe reports whether cond contains `!isSafeMethod(<req>.Method)`.
func guardDeniesUnsafe(cond ast.Expr, req string) bool {
	found := false
	ast.Inspect(cond, func(n ast.Node) bool {
		if found {
			return false
		}
		un, ok := n.(*ast.UnaryExpr)
		if !ok || un.Op != token.NOT {
			return true
		}
		call, ok := un.X.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return true
		}
		id, ok := call.Fun.(*ast.Ident)
		if !ok || id.Name != "isSafeMethod" {
			return true
		}
		sel, ok := call.Args[0].(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Method" {
			return true
		}
		if recv, ok := sel.X.(*ast.Ident); ok && recv.Name == req {
			found = true
		}
		return true
	})
	return found
}

// callsNetDo reports whether fn calls <x>.client.netDo.
func callsNetDo(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "netDo" || found {
			return true
		}
		inner, ok := sel.X.(*ast.SelectorExpr)
		if !ok || inner.Sel.Name != "client" {
			return true
		}
		if m, ok := inner.X.(*ast.Ident); ok && m.Name == "m" {
			found = true
		}
		return true
	})
	return found
}
