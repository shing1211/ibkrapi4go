// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

// concurrency.go verifies docs/design/08-concurrency.md against the code it
// describes.
//
// The document's claims divide into two kinds, and only one of them is a fact a
// checker can read. "Public types are safe for concurrent use", "No exported
// method panics on concurrent use" and "Every public method takes
// `context.Context` first" are properties of the whole package, asserted rather
// than derived, and two of the three are already false as written — Client.Close
// takes no context, and neither do Subscribe's and WithGatewayURL's siblings. The
// checks below are the three claims that have a named declaration behind them:
//
//   - the redaction list. The document names `Authorization` and `Cookie` and says
//     the redaction reaches both the logs and `*ibkr.Error.Message`, so the header
//     alternation, the two call sites and the error field are each read out of the
//     code that implements them.
//   - the insecure-skip-verify warning. The document says the SDK warns when the
//     option is enabled against a non-loopback host, which is a claim about a
//     guarded Warn at each place a client is built.
//   - the OAuth generation counter. The document says an in-flight result from an
//     older generation is returned to its caller but never repopulates the cache,
//     which is precisely a captured generation, a guard, and a return outside the
//     guard — all three of which are read from the code.
//
// What is deliberately NOT checked:
//
//   - The whole concurrency-contract table. Each row is a ✅ with a prose note
//     ("mutex-guarded state machine", "immutable after construction"). Those are
//     claims about every method of a type, not about one declaration, and a check
//     that had to re-derive them would be a concurrency audit rather than a
//     document check. `Limiter`'s mutex and `*Error`'s immutability in particular
//     are properties of a whole file's worth of statements.
//   - "Every public method takes `context.Context` first." False as written — see
//     the header note. Recorded here rather than enforced, because the code is
//     authoritative and the document is the side that needs correcting, and
//     because `Client.Close`, `Client.GatewayURL` and the twenty-two `WithXxx`
//     options are not defects: an option that took a context it could not honour,
//     and a Close that took one it had to invent, would both be worse.
//   - The goroutine-ownership table. "Started by" and "Stopped by" for the ws
//     reader, writer, ping and reconnect goroutines are spread across a read loop,
//     a write loop, a ping loop and a reconnect loop, and the honest version of
//     this check is a liveness proof. What the document's TLS and secrets rules
//     imply about Close is covered by the *Close* idempotence check in
//     client_composition.go, which reads the same code.
//   - "`Client.Close` cancels the WebSocket's owned I/O context and signals
//     session shutdown". The cancellation is real and lives in the ws handle's own
//     context; pinning it here would need to follow the value across three types,
//     and the close-semantics check already pins the part of Close that decides
//     whether any of that runs at all.
//   - "OAuth token invalidation ... never repopulates the cache" for the *refresh
//     token*. The guard covers the access token, the expiry and the refresh token
//     together — they are in one block — so the check reads the block, not one
//     assignment.
//   - "Never hold a mutex while performing I/O or blocking on a channel" and
//     "Channel sends select on `ctx.Done()`". Whole-file rules, in the same
//     category as the table.
//   - "CI runs `go test -race ./...`". A claim about a workflow file, not about
//     code, and it belongs to the CI configuration rather than to a design
//     document's contract with the code.

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strings"
)

const (
	concDoc        = "docs/design/08-concurrency.md"
	transportSrc   = "internal/transport.go"
	observabilityP = "internal/observability.go"
	oauthSrc       = "internal/oauth.go"
	poolSrcPath    = "pkg/ibkr/transport_pool.go"
)

// checkConcurrency runs the docs/design/08-concurrency.md checks as one unit.
func checkConcurrency(repoRoot string) error {
	return collectDocErrors(repoRoot, concDoc, []docCheck{
		{"redaction", checkRedactionCoversNamedHeaders},
		{"insecure-skip-verify warning", checkInsecureSkipVerifyWarns},
		{"OAuth generations", checkOAuthGenerations},
	})
}

// --- claim 1: the redaction list ----------------------------------------------

// redactionRe reads the headers the document says are redacted. The list is
// backtick-quoted in prose, so the two names the document commits to are read from
// the sentence and compared with the alternation the code compiles. A header the
// code redacts and the document does not name is not drift — the code may know
// about more; a header the document names and the code does not is drift, because
// a caller reading the document is relying on it.
//
// The whitespace runs are `\s+` throughout because the document wraps this
// sentence across two lines, and a pattern that needs the whole claim on one line
// reports "the check cannot run" for a sentence that is present and correct.
var redactionRe = regexp.MustCompile(
	"redacts\\s+((?:`[^`]+`,?\\s*)+)and\\s+token-bearing\\s+headers")

// checkRedactionCoversNamedHeaders verifies the document's redaction claim in the
// four places it can actually fail:
//
//   - the header alternation regexp must name every header the document names;
//   - redact must apply that alternation *and* the token patterns, because the
//     document claims both halves — a redact that applied only the header list
//     would leave a bearer token in a log line;
//   - the *Error the transport builds must carry a redacted Message, which is the
//     "`*ibkr.Error.Message`" half of the sentence and the half a caller sees when
//     a token ends up in a gateway error body;
//   - the logger must be handed the redacted text, at both the transport-failure
//     site and the decoded-error site.
func checkRedactionCoversNamedHeaders(repoRoot string) error {
	doc, err := parseDoc(repoRoot, concDoc)
	if err != nil {
		return err
	}
	trans, err := parseGo(repoRoot, transportSrc)
	if err != nil {
		return err
	}
	obs, err := parseGo(repoRoot, observabilityP)
	if err != nil {
		return err
	}
	claimLine, want, err := docRedactedHeaders(doc)
	if err != nil {
		return err
	}
	got, err := headerRedactAlternatives(trans)
	if err != nil {
		return err
	}
	for _, h := range want {
		if !got[strings.ToLower(h)] {
			return fmt.Errorf("redaction drift: %s: says the transport redacts %s, but %s's headerRedact "+
				"alternation names only %s", doc.at(claimLine), h, transportSrc, renderSet(got))
		}
	}
	redact := trans.funcDecl("redact")
	if redact == nil {
		return fmt.Errorf("%s: no redact function is declared; the redaction check cannot run", transportSrc)
	}
	applied := map[string]bool{}
	ast.Inspect(redact.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		switch calleeName(call.Fun) {
		case "ReplaceAllString", "ReplaceAll":
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if id, ok := sel.X.(*ast.Ident); ok {
					applied[id.Name] = true
				}
			}
		}
		return true
	})
	for _, pattern := range []string{"headerRedact", "secretRedact"} {
		if !applied[pattern] {
			return fmt.Errorf("redaction drift: %s:%d: redact does not apply %s; %s says the transport redacts "+
				"%s as well as token-bearing headers, and a redact that skipped one of the two patterns would leave "+
				"that half unredacted", transportSrc, trans.line(redact), pattern, doc.rel, strings.Join(want, " and "))
		}
	}

	// The *Error the transport builds.
	errorFrom := trans.funcDecl("ResponseError")
	if errorFrom == nil {
		return fmt.Errorf("%s: no ResponseError function is declared; %s says the redaction reaches "+
			"*ibkr.Error.Message, so there is nothing to check", transportSrc, doc.rel)
	}
	lit := firstErrorLit(errorFrom.Body)
	if lit == nil {
		return fmt.Errorf("redaction drift: %s: ResponseError builds no *Error; %s says the redaction "+
			"reaches *ibkr.Error.Message, so there is nothing to check", transportSrc, doc.rel)
	}
	msg := ""
	var msgNode ast.Expr
	for _, kv := range keyedValues(lit) {
		if kv.key == "Message" {
			msg, msgNode = exprString(kv.expr), kv.expr
		}
	}
	if !isCallTo(msgNode, "redact") {
		return fmt.Errorf("redaction drift: %s: the *Error the transport builds sets Message to %s, not a redacted "+
			"string; %s says the redaction reaches *ibkr.Error.Message, which is where a caller would read a token "+
			"echoed back by the gateway", trans.at(lit), msg, doc.rel)
	}

	// The two log sites.
	for _, c := range []struct {
		what string
		fn   string
	}{
		{"the transport-failure log line", "logRequest"},
		{"the decoded-error log line", "LogError"},
	} {
		fn := obs.funcDecl(c.fn)
		if fn == nil {
			return fmt.Errorf("%s: no %s function is declared; the redaction check cannot run", observabilityP, c.fn)
		}
		if !logsRedactedText(fn) {
			return fmt.Errorf("redaction drift: %s: %s does not pass redact(...) to the logger; %s says the "+
				"transport redacts token-bearing headers from logs, and an unredacted attribute is a token in a log "+
				"file", observabilityP, c.what, doc.rel)
		}
	}
	return nil
}

func docRedactedHeaders(doc *docFile) (int, []string, error) {
	text := strings.Join(doc.lines, "\n")
	m := redactionRe.FindStringSubmatch(text)
	if m == nil {
		return 0, nil, fmt.Errorf("%s: the secrets section no longer names the headers the transport redacts; "+
			"the redaction check cannot run", doc.rel)
	}
	var out []string
	for _, h := range backtickRe.FindAllStringSubmatch(m[1], -1) {
		out = append(out, h[1])
	}
	if len(out) == 0 {
		return 0, nil, fmt.Errorf("%s: the redaction sentence names no backtick-quoted header; the check cannot run", doc.rel)
	}
	return strings.Count(text[:strings.Index(text, m[0])], "\n") + 1, out, nil
}

// headerRedactAlternatives returns the lowercased alternatives of the regexp the
// transport compiles into headerRedact.
//
// The alternatives are read out of the pattern rather than matched against, so a
// header that is still there but can no longer match — because the `\s*:\s*` tail
// was dropped, say — is still seen as named here and the *Error/log call sites
// below are what would catch the rest. Reading the list is what makes the message
// name the header a maintainer has to add back.
func headerRedactAlternatives(code *goSrc) (map[string]bool, error) {
	for _, d := range code.file.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}
		for _, s := range gd.Specs {
			vs, ok := s.(*ast.ValueSpec)
			if !ok || len(vs.Names) == 0 || vs.Names[0].Name != "headerRedact" || len(vs.Values) != 1 {
				continue
			}
			call, ok := vs.Values[0].(*ast.CallExpr)
			if !ok || calleeName(call.Fun) != "MustCompile" || len(call.Args) != 1 {
				return nil, fmt.Errorf("%s: headerRedact is not a single regexp.MustCompile; the redaction check "+
					"cannot read its header list", transportSrc)
			}
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return nil, fmt.Errorf("%s: headerRedact's pattern is not a string literal; the redaction check "+
					"cannot read its header list", transportSrc)
			}
			pattern, ok := stringLit(lit)
			if !ok {
				return nil, fmt.Errorf("%s: headerRedact's pattern is not a readable string literal", transportSrc)
			}
			return alternationOf(pattern), nil
		}
	}
	return nil, fmt.Errorf("%s: no headerRedact variable is declared; the redaction check cannot run", transportSrc)
}

// alternationOf returns the alternatives of the first `(a|b|c)` group in a
// pattern, lowercased, ignoring `(?i)` and any non-capturing prefix.
func alternationOf(pattern string) map[string]bool {
	out := map[string]bool{}
	start := strings.Index(pattern, "(")
	for start >= 0 && strings.HasPrefix(pattern[start:], "(?") {
		start = strings.Index(pattern[start+2:], "(")
		if start >= 0 {
			start += 2
		}
	}
	if start < 0 {
		return out
	}
	end := strings.Index(pattern[start:], ")")
	if end < 0 {
		return out
	}
	for _, alt := range strings.Split(pattern[start+1:start+end], "|") {
		out[strings.ToLower(strings.TrimSpace(alt))] = true
	}
	return out
}

func renderSet(s map[string]bool) string {
	var parts []string
	for k := range s {
		parts = append(parts, k)
	}
	return strings.Join(parts, ", ")
}

func isCallTo(e ast.Expr, callee string) bool {
	call, ok := e.(*ast.CallExpr)
	return ok && calleeName(call.Fun) == callee
}

// logsRedactedText reports whether a function passes a redact(...) result to a
// logging call.
//
// The redact call is located first and the logging call second, so a redact that
// computed a value nobody logged does not pass. A logging argument counts when it
// *contains* a redact call rather than when it is one: logRequest appends the
// redacted value into an attribute slice (`append(attrs, "err", redact(...))`),
// and demanding the call be a direct argument would report a log line that does
// redact its error text as one that does not.
func logsRedactedText(fn *ast.FuncDecl) bool {
	redacts := 0
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok && calleeName(call.Fun) == "redact" {
			redacts++
		}
		return true
	})
	if redacts == 0 {
		return false
	}
	logged := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if recv, ok := sel.X.(*ast.Ident); !ok || recv.Name != "logger" {
			return true
		}
		for _, a := range call.Args {
			if containsRedactCall(a) {
				logged = true
			}
		}
		return true
	})
	return logged
}

// containsRedactCall reports whether an expression contains a call to redact.
func containsRedactCall(e ast.Expr) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok && calleeName(call.Fun) == "redact" {
			found = true
		}
		return !found
	})
	return found
}

// --- claim 2: the insecure-skip-verify warning --------------------------------

// skipVerifyWarnRe recognises the rule: the option is local-only and the SDK warns
// when it is used against a non-loopback host. Both halves are required, so
// removing either makes the check report that the rule is gone rather than
// silently checking less.
var skipVerifyWarnRe = regexp.MustCompile(
	"(?i)warns if enabled against a non-loopback host")

// warnSites are the two places a client is built. A warning that fired for only
// one of them would leave the other silent, and the document's rule is about the
// SDK rather than about one constructor, so both are required. The pool's builder
// is named here because it exists, and the reason NewClient alone is not enough to
// check is the reason it is a separate site: a client taken from the transport
// pool is configured by a different function.
var warnSites = []struct {
	rel string
	fn  string
}{
	{clientSrcPath, "NewClient"},
	{poolSrcPath, "NewTransportPool"},
}

// checkInsecureSkipVerifyWarns verifies the document's TLS rule:
//
//   - `WithInsecureSkipVerify` really writes the configuration field the guard
//     reads, resolved from the option's own assignment rather than assumed;
//   - each client-building function guards a Warn on that field *and* on
//     isNonLoopback of the configured gateway URL, so the warning is about a
//     non-loopback host rather than about the option alone.
func checkInsecureSkipVerifyWarns(repoRoot string) error {
	doc, err := parseDoc(repoRoot, concDoc)
	if err != nil {
		return err
	}
	code, err := parseGo(repoRoot, clientSrcPath)
	if err != nil {
		return err
	}
	text := strings.Join(doc.lines, "\n")
	m := skipVerifyWarnRe.FindStringIndex(text)
	if m == nil {
		return fmt.Errorf("%s: the secrets section no longer states that the SDK warns when skip-verify is enabled "+
			"against a non-loopback host; the check cannot run", doc.rel)
	}
	claimLine := strings.Count(text[:m[0]], "\n") + 1

	opt := code.funcDecl("WithInsecureSkipVerify")
	if opt == nil {
		return fmt.Errorf("%s: no WithInsecureSkipVerify function is declared; the TLS check cannot run", clientSrcPath)
	}
	field := optionWrittenField(opt)
	if field == "" {
		return fmt.Errorf("tls drift: %s:%d: WithInsecureSkipVerify writes no field of the config, so %s's rule "+
			"about enabling it against a non-loopback host has nothing to test", clientSrcPath, code.line(opt), doc.rel)
	}
	for _, site := range warnSites {
		src, err := parseGo(repoRoot, site.rel)
		if err != nil {
			return err
		}
		fn := src.funcDecl(site.fn)
		if fn == nil {
			return fmt.Errorf("%s: no %s function is declared; the TLS check cannot run", site.rel, site.fn)
		}
		guard := skipVerifyGuard(fn, field)
		if guard == nil {
			return fmt.Errorf("tls drift: %s: %s has no `if <config>.%s && isNonLoopback(...)` guard; %s:%d "+
				"says the SDK warns if skip-verify is enabled against a non-loopback host, and a client built here "+
				"would enable it silently", site.rel, site.fn, field, doc.rel, claimLine)
		}
		if !logsGatewayWarning(guard.Body) {
			return fmt.Errorf("tls drift: %s: the skip-verify guard at %s logs no warning; %s:%d says the SDK "+
				"warns if skip-verify is enabled against a non-loopback host, and a client built here would enable "+
				"it silently", site.rel, src.at(guard), doc.rel, claimLine)
		}
	}
	return nil
}

// optionWrittenField returns the config field an option's closure assigns to.
func optionWrittenField(fn *ast.FuncDecl) string {
	name := ""
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
			return true
		}
		sel, ok := as.Lhs[0].(*ast.SelectorExpr)
		if !ok || sel.Sel.Name == "" {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == "c" {
			name = sel.Sel.Name
		}
		return true
	})
	return name
}

// skipVerifyGuard returns the guard in fn that tests the skip-verify field against
// isNonLoopback. The config value is a *local*, not a parameter: NewClient takes
// only its options and builds `cfg` itself. The local is resolved from the
// composite literal it is initialised from, so a builder that took a *config
// parameter instead would be reported rather than misread — and a builder that
// shadowed `cfg` with something else would be a different function, not a silent
// pass. The predicate comes from the call itself, so neither the URL field nor the
// predicate name is assumed.
func skipVerifyGuard(fn *ast.FuncDecl, field string) *ast.IfStmt {
	cfg := configLocal(fn)
	if cfg == "" {
		return nil
	}
	for _, stmt := range fn.Body.List {
		guard, ok := stmt.(*ast.IfStmt)
		if !ok {
			continue
		}
		if testsFieldAndNonLoopback(guard.Cond, cfg, field) {
			return guard
		}
	}
	return nil
}

// configLocal returns the name of the local a function builds a config value in,
// read from the composite literal it is initialised from. A function that takes a
// *config parameter is treated as having no such local, because the two are
// different constructions and guessing between them is how a check reads the wrong
// declaration.
func configLocal(fn *ast.FuncDecl) string {
	if p := paramOfType(fn.Type, "*config"); p != "" {
		return ""
	}
	for _, stmt := range fn.Body.List {
		as, ok := stmt.(*ast.AssignStmt)
		if !ok || as.Tok != token.DEFINE || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
			continue
		}
		cl, ok := as.Rhs[0].(*ast.CompositeLit)
		if !ok || structLitType(cl) != "config" {
			continue
		}
		if id, ok := as.Lhs[0].(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

// testsFieldAndNonLoopback reports whether cond is
// `<cfg>.<field> && isNonLoopback(<cfg>.<urlField>)`. The URL field is read out of
// the call, so a guard that tested a different field than the gateway's URL is
// reported rather than accepted.
func testsFieldAndNonLoopback(cond ast.Expr, cfg, field string) bool {
	and, ok := cond.(*ast.BinaryExpr)
	if !ok || and.Op != token.LAND {
		return false
	}
	if !isFieldOf(and.X, cfg, field) {
		return false
	}
	call, ok := and.Y.(*ast.CallExpr)
	if !ok || calleeName(call.Fun) != "isNonLoopback" || len(call.Args) != 1 {
		return false
	}
	sel, ok := call.Args[0].(*ast.SelectorExpr)
	if !ok {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == cfg && sel.Sel.Name != ""
}

// logsGatewayWarning reports whether a guard logs a Warn that names the gateway
// URL.
//
// The URL is the substance of the warning — a caller who enabled skip-verify
// against a remote host has to be told *which* host — so requiring it is
// wording-independent. Matching on the word "insecure" in the message text would
// instead fail a maintainer who reworded a perfectly good warning, and a check
// that fires on prose is a check people learn to route around.
func logsGatewayWarning(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Warn" {
			return true
		}
		for _, a := range call.Args {
			if isAnySelectorField(a, "gatewayURL") {
				found = true
			}
		}
		return true
	})
	return found
}

// isAnySelectorField reports whether an expression mentions `<recv>.<field>` for
// some receiver. The receiver is not constrained because the guard's own condition
// has already established which config value is at issue; what is being checked
// here is that the warning names the URL, not which local it read.
func isAnySelectorField(e ast.Expr, field string) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if ok && sel.Sel.Name == field {
			found = true
		}
		return !found
	})
	return found
}

// --- claim 3: the OAuth generation counter ------------------------------------

// oauthGenerationRe recognises the rule the counter implements. The wording is
// required, so a document that stopped claiming it is reported rather than leaving
// the check with nothing to verify. The whitespace runs are `\s+` because the
// document wraps this sentence across two lines.
var oauthGenerationRe = regexp.MustCompile(
	"(?i)invalidation uses generations;\\s+an in-flight result from an older\\s+" +
		"generation is returned to its caller\\s+but never repopulates the cache")

// checkOAuthGenerations verifies the deadlock-avoidance rule that keeps a stale
// token from overwriting a fresh one.
//
// Three declarations, and the rule needs all three:
//
//   - invalidateLocked must bump the generation counter. Without the bump, nothing
//     distinguishes the in-flight result from the invalidated state.
//   - Token must capture the counter into a local *before* it fetches, and every
//     cache write must sit inside a guard comparing the counter against that local.
//     The local is the point: a guard comparing the counter to itself would always
//     hold.
//   - the return must sit outside that guard, or an invalidated fetch would fail
//     its caller as well as its cache — which is the half of the sentence a
//     cache-only check cannot see.
func checkOAuthGenerations(repoRoot string) error {
	doc, err := parseDoc(repoRoot, concDoc)
	if err != nil {
		return err
	}
	code, err := parseGo(repoRoot, oauthSrc)
	if err != nil {
		return err
	}
	text := strings.Join(doc.lines, "\n")
	m := oauthGenerationRe.FindStringIndex(text)
	if m == nil {
		return fmt.Errorf("%s: the deadlock-avoidance section no longer states the generations rule; the "+
			"OAuth-generation check cannot run", doc.rel)
	}
	claimLine := strings.Count(text[:m[0]], "\n") + 1

	source := code.typeSpec("TokenSource")
	if source == nil {
		return fmt.Errorf("%s: no TokenSource type is declared; the OAuth-generation check cannot run", oauthSrc)
	}
	gen, _ := structHasField(code, "TokenSource", "generation")
	if !gen {
		return fmt.Errorf("oauth generation drift: %s: TokenSource declares no generation field; %s:%d says "+
			"token invalidation uses generations, so an in-flight result from an older generation has no way to "+
			"tell it is stale", oauthSrc, oauthSrc, claimLine)
	}

	invalidate := code.method("TokenSource", "invalidateLocked")
	if invalidate == nil {
		return fmt.Errorf("%s: no TokenSource.invalidateLocked method is declared; the OAuth-generation check "+
			"cannot run", oauthSrc)
	}
	if !incrementsField(invalidate.Body, receiverName(invalidate), "generation") {
		return fmt.Errorf("oauth generation drift: %s: TokenSource.invalidateLocked does not increment the "+
			"generation counter; %s:%d says invalidation uses generations, so a fetch already in flight would "+
			"repopulate the cache it was told to discard", code.at(invalidate), doc.rel, claimLine)
	}

	token := code.method("TokenSource", "Token")
	if token == nil {
		return fmt.Errorf("%s: no TokenSource.Token method is declared; the OAuth-generation check cannot run", oauthSrc)
	}
	captured, _ := capturedGeneration(code, token, receiverName(token), "generation")
	if captured == "" {
		return fmt.Errorf("oauth generation drift: %s: Token does not read the generation counter into a local "+
			"before fetching; %s:%d says an in-flight result from an older generation is recognised as older, "+
			"which needs the value it was started with", code.at(token), doc.rel, claimLine)
	}
	guard := guardComparingField(token.Body, receiverName(token), "generation", captured)
	if guard == nil {
		return fmt.Errorf("oauth generation drift: %s: Token has no `if %s.generation == %s` guard around the "+
			"cache write; %s:%d says an in-flight result from an older generation never repopulates the cache",
			code.at(token), receiverName(token), captured, doc.rel, claimLine)
	}
	for _, field := range []string{"token", "expiry", "refreshToken"} {
		if !assignsReceiverFieldInside(guard.Body, receiverName(token), field) {
			return fmt.Errorf("oauth generation drift: %s: the generation guard does not assign %s.%s; "+
				"%s:%d says an in-flight result from an older generation never repopulates the cache, and %s is "+
				"part of that cache", code.at(guard), receiverName(token), field, doc.rel, claimLine, field)
		}
	}
	if guardReturns(guard.Body) {
		return fmt.Errorf("oauth generation drift: %s: Token returns from inside the generation guard, so a "+
			"superseded fetch would fail its caller as well as its cache; %s:%d says the result is returned to "+
			"its caller and only the cache write is suppressed", code.at(guard), doc.rel, claimLine)
	}
	return nil
}

// guardReturns reports whether a block contains a return of any shape. A
// generation guard that returns is not a guard over the cache any more: it has
// become a caller-visible early exit, which is the half of the document's sentence
// that a cache-write-only check cannot see.
func guardReturns(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if _, ok := n.(*ast.ReturnStmt); ok {
			found = true
		}
		return !found
	})
	return found
}

// capturedGeneration returns the local a method copies the named field into
// before it does any I/O, and the assignment's node. The assignment is required to
// come before the fetch call so the value captured is the one the fetch started
// with.
func capturedGeneration(code *goSrc, fn *ast.FuncDecl, recv, field string) (string, ast.Node) {
	fetchAt := -1
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if fetchAt >= 0 {
			return false
		}
		if call, ok := n.(*ast.CallExpr); ok && calleeName(call.Fun) == "fetch" {
			fetchAt = code.line(call)
		}
		return true
	})
	if fetchAt < 0 {
		return "", nil
	}
	name, node := "", ast.Node(nil)
	for _, stmt := range fn.Body.List {
		as, ok := stmt.(*ast.AssignStmt)
		if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 || code.line(as) > fetchAt {
			continue
		}
		if !isFieldOf(as.Rhs[0], recv, field) {
			continue
		}
		if id, ok := as.Lhs[0].(*ast.Ident); ok {
			name, node = id.Name, as
		}
	}
	return name, node
}

// guardComparingField returns the if-statement whose condition compares
// `<recv>.<field>` to a local, or nil. The comparison has to be equality: an
// inequality guard would write the cache precisely when the generation differs,
// which is the inverse of the documented rule and the reason the operator is
// checked rather than the presence of any comparison.
func guardComparingField(body *ast.BlockStmt, recv, field, local string) *ast.IfStmt {
	for _, stmt := range body.List {
		guard, ok := stmt.(*ast.IfStmt)
		if !ok {
			continue
		}
		b, ok := guard.Cond.(*ast.BinaryExpr)
		if !ok || b.Op != token.EQL {
			continue
		}
		if !isFieldOf(b.X, recv, field) {
			continue
		}
		if id, ok := b.Y.(*ast.Ident); ok && id.Name == local {
			return guard
		}
	}
	return nil
}

// assignsReceiverFieldInside reports whether a block assigns a value to
// `<recv>.<field>`.
func assignsReceiverFieldInside(body *ast.BlockStmt, recv, field string) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for _, lhs := range as.Lhs {
			if isFieldOf(lhs, recv, field) {
				found = true
			}
		}
		return true
	})
	return found
}

// incrementsField reports whether a body increments `<recv>.<field>`.
func incrementsField(body *ast.BlockStmt, recv, field string) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		inc, ok := n.(*ast.IncDecStmt)
		if !ok || inc.Tok != token.INC {
			return true
		}
		if isFieldOf(inc.X, recv, field) {
			found = true
		}
		return true
	})
	return found
}

// structHasField reports whether a struct declares a field with the given name, and
// returns the field's node. The type is not constrained: the claim is that a
// counter exists, and a counter that changed from uint64 to int is not drift.
func structHasField(code *goSrc, typ, field string) (bool, ast.Node) {
	fields, err := code.structFields(typ)
	if err != nil {
		return false, nil
	}
	for _, f := range fields {
		if f.name == field {
			return true, f.node
		}
	}
	return false, nil
}
