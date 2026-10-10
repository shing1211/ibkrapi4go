// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

// client_composition.go verifies docs/design/02-client.md against the code it
// describes. This document is the map of the SDK: if the map is wrong, every
// other document is being read from the wrong starting point, and unlike a prose
// claim a wrong field name or accessor here costs a caller a compile error before
// they write a line of their own.
//
// The checks are:
//
//   - the `Client` struct block, as a name set in both directions. The document
//     states its own completeness rule — "The struct diagram shows all significant
//     fields" with the pool `release` callback omitted — and the note that states
//     it names what it omits, so the omissions are read out of the document
//     rather than hardcoded here.
//   - the accessor list, each documented accessor against the method it names.
//   - the NewClient contract: its signature, the fact that a bad option's error
//     reaches the caller unwrapped, and the fact that constructing a client starts
//     nothing on the network and authenticates nothing.
//   - the close contract: `Close() error`, idempotence by returning nil on a
//     second call, and ErrClosed reaching a caller through the one guard every
//     request passes.
//   - the options block, each documented `With*` line against the function it
//     names: the name has to be a declared package-level func, the parameter
//     types have to be the declaration's, and the single result has to be
//     `Option`, which is what the documented `NewClient(opts ...Option)` needs to
//     accept the line as an argument.
//
// Two disagreements between this document and the code were recorded rather than
// checked. Both were resolved in the code's favour by a human on 2026-09-27, and
// both halves are now checked:
//
//   - The struct diagram's completeness note contradicted the diagram it
//     qualified: the note (docs/design/02-client.md:40) said the "Internal
//     synchronization fields (`wsMu`, `restMu`) ... are omitted for clarity" while
//     the block two lines above listed both of them (lines 34 and 36). The code is
//     authoritative — both fields are declared in pkg/ibkr/client.go:341,344 — so
//     the block was right and the note was over-broad. The note was corrected and
//     now names only the pool `release` callback, which pkg/ibkr/client.go:339 does
//     declare, so it is a true omission and the contradiction is gone.
//
//     The check still *tolerates* a note name the block also lists, because a
//     redundant note name is not a defect this program can report: the field is
//     documented either way, so it is neither an undeclared field nor an omission
//     that could let one through. That tolerance is exercised by
//     struct/the-contradictory-omission-note-passes, which reintroduces the
//     contradiction in the scratch tree.
//
//   - The options block named `WithOAuth2JWTKeyPath(path string)` and
//     `WithOAuth2JWTKey(key []byte)`, neither of which compiles:
//     `WithOAuth2JWTKeyPath` is declared nowhere, and the declaration at
//     pkg/ibkr/oauth.go:80 takes a *rsa.PrivateKey. The document was corrected to
//     the four declarations at pkg/ibkr/oauth.go:80,89,102,111, so
//     checkClientOptions can now be written. Before the correction it could not:
//     a check over the block would have had to decide which side was authoritative.
//
// What is deliberately NOT checked:
//
//   - The options block's *completeness*. Every one of the twenty-two documented
//     options is now verified against its declaration, but the reverse direction
//     is not a claim and is not checked: the section states no exhaustiveness, the
//     way the accessor block states none, and seven exported options are
//     deliberately absent — WithMaxResponseBytes, WithTransportMiddleware,
//     WithEndpointTimeout, WithOAuth2, WithOAuth2Scope, WithOAuth2TokenURL and
//     WithOAuth2JWTExpiry. Closing that gap needs either seven more lines or a
//     sentence scoping the list, and both are document edits, so they are a
//     decision for a human rather than for this program.
//   - `NewClient` validates configuration. Which fields it validates, and in what
//     order, is not stated; the check verifies only the three things above.
//   - The Lifecycle list. Steps 2 and 4 are the same `Session().Initialize` and
//     `Client.Close` the close contract already pins; step 1 is NewClient; step 3
//     is prose.
//   - `Stops background goroutines; no leaks (verified with `goleak`)`. A claim
//     about a test run, not about production code. A check over it would fail on
//     a renamed test and rot.
//   - `Managers are created once and are safe to reuse`. The concurrency half is
//     08-concurrency.md's table row and is checked there.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
)

const (
	clientDoc     = "docs/design/02-client.md"
	clientSrcPath = "pkg/ibkr/client.go"
	// respSrcPath holds netDo, the single guard every request passes.
	respSrcPath = "pkg/ibkr/response.go"
	// pubErrSrcPath declares the sentinels callers match on.
	pubErrSrcPath = "pkg/ibkr/errors.go"
)

// --- claim 1: the Client struct block ----------------------------------------

// structCompletenessRe recognises the note that states what the struct diagram
// leaves out. It is required rather than assumed: the omissions this check allows
// are the omissions the document names, so a document that drops the note leaves
// the check with nothing to read and must be reported rather than passed.
var structCompletenessRe = regexp.MustCompile(`(?i)struct diagram shows all significant fields`)

// backtickRe finds the `name` spans of a line.
var backtickRe = regexp.MustCompile("`([^`]+)`")

// checkClientStructBlock verifies the documented `Client` struct block as a set of
// field names, in both directions.
//
// The directions are what matter. Document-to-code alone passes on a block that
// has fallen behind, and 09-orders-and-confirmation.md learned that the hard way:
// its OrderRequest block omitted two declared fields for as long as only one
// direction ran. Code-to-document alone would pass on a block that describes a
// client that does not exist.
//
// The document scopes its own completeness, so the omissions come from the
// document: the note beneath the block names what it leaves out. A name the note
// gives that the block already lists is redundant rather than an omission, and is
// tolerated — see the header. The declaration is what an omission is matched
// against, so a stale note cannot let a real field through.
func checkClientStructBlock(repoRoot string) error {
	doc, err := parseDoc(repoRoot, clientDoc)
	if err != nil {
		return err
	}
	code, err := parseGo(repoRoot, clientSrcPath)
	if err != nil {
		return err
	}
	note, omitted, err := docStructOmissionNote(doc)
	if err != nil {
		return err
	}
	docFields, err := doc.structBlockNames("Client")
	if err != nil {
		return err
	}
	realFields, err := code.structFields("Client")
	if err != nil {
		return err
	}

	// blockNames maps each name the block itself lists to its position, so a
	// duplicate can name both lines. allowed is the block plus the note's
	// effective omissions, and omissions keeps only the names the block does not
	// already show.
	blockNames := map[string]int{}
	for i, df := range docFields {
		if prev, dup := blockNames[df.name]; dup {
			return fmt.Errorf("client struct drift: %s lists field %q at both %s and %s",
				doc.rel, df.name, doc.at(docFields[prev].lineNo), doc.at(df.lineNo))
		}
		blockNames[df.name] = i
	}
	allowed := map[string]bool{}
	for _, df := range docFields {
		allowed[df.name] = true
	}
	var omissions []string
	for _, name := range omitted {
		if allowed[name] {
			// The block shows what the note says it omits, so the note is
			// redundant rather than wrong. The document used to be in that state
			// for a field the code declares, and the human corrected the note on
			// 2026-09-27; a redundancy is not reported, because the field is
			// documented either way and the comparisons below are unaffected.
			// struct/the-contradictory-omission-note-passes reintroduces it in the
			// scratch tree so the tolerance stays pinned.
			continue
		}
		allowed[name] = true
		omissions = append(omissions, name)
	}

	for _, df := range docFields {
		if !declaresField(realFields, df.name) {
			return fmt.Errorf("client struct drift: %s documents Client field %q, which %s does not declare",
				doc.at(df.lineNo), df.name, clientSrcPath)
		}
	}
	for _, rf := range realFields {
		if rf.name == "" {
			return fmt.Errorf("client struct drift: %s embeds %s at %s, which the documented block does not show",
				clientSrcPath, rf.typ, code.at(rf.node))
		}
		if !allowed[rf.name] {
			return fmt.Errorf("client struct drift: %s declares Client field %q at %s, which %s neither documents "+
				"nor lists among the fields its completeness note omits (note at %s)",
				clientSrcPath, rf.name, code.at(rf.node), doc.rel, doc.at(note))
		}
	}
	// A name the note claims to omit which the declaration does not have is a note
	// that has gone stale in the other direction, and would otherwise let a real
	// field through: the allowance is matched against the declaration, not applied
	// blind.
	for _, name := range omissions {
		if !declaresField(realFields, name) {
			return fmt.Errorf("client struct drift: %s:%d says the block omits %q, but %s declares no such field; "+
				"the note is stale, so it can no longer be read as the document's list of omissions",
				doc.rel, note, name, clientSrcPath)
		}
	}
	return nil
}

func declaresField(fields []codeField, name string) bool {
	for _, f := range fields {
		if f.name == name {
			return true
		}
	}
	return false
}

// docStructOmissionNote finds the note that scopes the struct block's
// completeness, and returns its line number together with the field names it says
// are left out.
//
// A note that names nothing is an error rather than an empty allowance. That is
// the failure mode worth catching: with no named omission the check would demand
// the block list every field, and the maintainer would be pushed to delete the
// note to make the build green rather than to fix either side.
func docStructOmissionNote(doc *docFile) (int, []string, error) {
	for i, l := range doc.lines {
		if !structCompletenessRe.MatchString(l) {
			continue
		}
		// The note runs until the blockquote ends or a heading starts.
		var text []string
		for j := i; j < len(doc.lines); j++ {
			t := strings.TrimSpace(doc.lines[j])
			if t == "" || strings.HasPrefix(t, "#") {
				break
			}
			text = append(text, t)
		}
		var omitted []string
		for _, m := range backtickRe.FindAllStringSubmatch(strings.Join(text, " "), -1) {
			omitted = append(omitted, m[1])
		}
		if len(omitted) == 0 {
			return 0, nil, fmt.Errorf("%s:%d: the note scoping the struct diagram's completeness names no omitted "+
				"field, so the check cannot tell which declared fields the block intends to skip",
				doc.rel, i+1)
		}
		return i + 1, omitted, nil
	}
	return 0, nil, fmt.Errorf("%s: no note scoping the struct diagram's completeness was found; the check cannot run", doc.rel)
}

// --- claim 2: the accessor list ----------------------------------------------

// accessorReceiver is the receiver type the document writes its accessors on.
const accessorReceiver = "*Client"

// checkClientAccessors verifies every signature in the document's accessor block
// against the method it names: receiver type, name, parameter types and result
// types. Parameter *names* are not compared, because renaming one changes no
// contract a caller can observe.
//
// The method is looked for across the whole public package rather than in
// client.go alone: `REST()` is documented beside the other fifteen accessors and
// declared in pkg/ibkr/rest.go, and a check that read only client.go would report
// a missing method for an accessor that exists.
//
// The reverse direction is not checked: the document's block is a curated list and
// says nothing about GatewayURL or HTTPClient, which the declaration also has.
// Requiring the reverse would make the check enforce a completeness claim the
// document never made.
func checkClientAccessors(repoRoot string) error {
	doc, err := parseDoc(repoRoot, clientDoc)
	if err != nil {
		return err
	}
	found := 0
	sigs, err := doc.signatures()
	if err != nil {
		return err
	}
	for _, want := range sigs {
		if want.recvType != accessorReceiver {
			continue
		}
		found++
		code, fn, err := resolveMethod(repoRoot, pubIbdAlias, "Client", want.name)
		if err != nil {
			return err
		}
		got := codeFunc(fn)
		if !equalStrings(want.params, got.params) || !equalStrings(want.results, got.results) {
			return fmt.Errorf("client accessor drift: %s documents %q but %s declares %q",
				doc.at(want.lineNo), want.render(), code.at(fn), got.render())
		}
	}
	if found == 0 {
		return fmt.Errorf("%s: the accessor block documents no %s signature; the check cannot run", doc.rel, accessorReceiver)
	}
	return nil
}

// resolveMethod finds the method named name on receiver type recv anywhere in a
// package, and returns the file it was declared in alongside the declaration. The
// file is part of the answer because a failure has to name the line a maintainer
// has to edit, and because "declared in rest.go, not client.go" is exactly the
// kind of thing a single-file check gets wrong.
func resolveMethod(repoRoot, pkgDir, recv, name string) (*goSrc, *ast.FuncDecl, error) {
	files, err := pkgGoFiles(repoRoot, pkgDir)
	if err != nil {
		return nil, nil, err
	}
	for _, rel := range files {
		code, err := parseGo(repoRoot, rel)
		if err != nil {
			return nil, nil, err
		}
		if fn := code.method(recv, name); fn != nil {
			return code, fn, nil
		}
	}
	return nil, nil, fmt.Errorf("no %s.%s method is declared anywhere in %s; the check cannot run", recv, name, pkgDir)
}

// --- claim 3: the NewClient contract -----------------------------------------

// ioCallNames is the closed set of calls that would perform network I/O or start
// the session. It is closed so a new one is reported rather than skipped, for the
// reason isReaderConstructor gives: an open-ended "does this look like I/O?"
// heuristic is not a check.
//
// A method named here on a *different* receiver is not flagged — only a call on
// the receiver or on a value derived from it — so `cfg.retry.MaxAttempts` is fine
// while `httpClient.Get(url)` is not.
var ioCallNames = map[string]bool{
	"Do": true, "Get": true, "Head": true, "Post": true, "PostForm": true,
	"RoundTrip": true, "NewRequest": true, "NewRequestWithContext": true,
	"Dial": true, "DialContext": true, "DialWS": true, "Connect": true,
	"Initialize": true, "Start": true, "Tickle": true, "Authenticate": true,
}

// checkNewClientContract verifies the three things 02-client.md's Construction
// section states about NewClient that have a mechanism to read.
//
//   - The signature. A caller writing `cli, err := ibkr.NewClient(...)` reads it
//     from the document.
//   - An option's error reaches the caller unwrapped. "Returns *ibkr.ConfigError on
//     bad input" is a claim about which error type arrives, and it only holds if
//     the option loop hands the option's error back as it is; a wrap or a
//     replacement here would turn every configuration error into something else,
//     and `errors.As(err, &cfgErr)` is how a caller matches it.
//   - Nothing on the network is started. "Does not perform I/O and does not
//     authenticate" is the property that makes `NewClient` safe to call in a
//     constructor, a test and a CLI, and a transport that dialed on construction
//     would break all three.
func checkNewClientContract(repoRoot string) error {
	doc, err := parseDoc(repoRoot, clientDoc)
	if err != nil {
		return err
	}
	code, err := parseGo(repoRoot, clientSrcPath)
	if err != nil {
		return err
	}
	sigs, err := doc.signatures()
	if err != nil {
		return err
	}
	want, ok := findSignature(sigs, "", "NewClient")
	if !ok {
		return fmt.Errorf("%s: no documented `func NewClient` signature; the construction check cannot run", doc.rel)
	}
	fn := code.funcDecl("NewClient")
	if fn == nil {
		return fmt.Errorf("%s: no NewClient function is declared; the construction check cannot run", clientSrcPath)
	}
	got := codeFunc(fn)
	if !equalStrings(want.params, got.params) || !equalStrings(want.results, got.results) {
		return fmt.Errorf("client construction drift: %s documents %q but %s declares %q",
			doc.at(want.lineNo), want.render(), code.at(fn), got.render())
	}
	if err := checkOptionErrorIsPropagated(code, fn); err != nil {
		return err
	}
	return checkNewClientStartsNoIO(code, fn)
}

// checkOptionErrorIsPropagated requires the option loop to return the error an
// option returned, as it is. The loop is found by the call it makes on the
// variadic parameter, so a reordering of NewClient's statements cannot move the
// check onto a different call.
func checkOptionErrorIsPropagated(code *goSrc, fn *ast.FuncDecl) error {
	opts := variadicParam(fn)
	if opts == "" {
		return fmt.Errorf("%s: NewClient takes no variadic parameter; the construction check cannot run", clientSrcPath)
	}
	for _, stmt := range fn.Body.List {
		loop, ok := stmt.(*ast.RangeStmt)
		if !ok || exprString(loop.X) != opts {
			continue
		}
		for _, inner := range loop.Body.List {
			g, ok := inner.(*ast.IfStmt)
			if !ok || g.Init == nil {
				continue
			}
			as, ok := g.Init.(*ast.AssignStmt)
			if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
				continue
			}
			// calleeName takes the *call target*, not the call: `o(&cfg)` is
			// guarded by reading the identifier `o`, which is the loop variable the
			// option is held in. Reading the wrong node here would make this guard
			// invisible, so the assertion is on the loop's own value identifier
			// instead — a fixed name in the source would be a weaker check.
			call, ok := as.Rhs[0].(*ast.CallExpr)
			if !ok || calleeName(call.Fun) != loop.Value.(*ast.Ident).Name {
				continue
			}
			id, ok := as.Lhs[0].(*ast.Ident)
			if !ok {
				continue
			}
			if err := propagatesErrorUnwrapped(g, id.Name); err != nil {
				return fmt.Errorf("client construction drift: %s: %w; %s documents that NewClient returns the "+
					"*ConfigError an option built, and a wrap or replacement here would make errors.As miss it",
					code.at(g), err, clientDoc)
			}
			return nil
		}
		return fmt.Errorf("%s: NewClient's loop over %s contains no `if err := <option>(&cfg); err != nil` guard; "+
			"%s documents that NewClient returns *ConfigError on bad input, so the option's error has to be read "+
			"and returned here", clientSrcPath, opts, clientDoc)
	}
	return fmt.Errorf("%s: NewClient does not range over its %s parameter; the construction check cannot run",
		clientSrcPath, opts)
}

// propagatesErrorUnwrapped reports whether the guard returns the init error
// directly, rather than wrapping or replacing it.
//
// The two shapes that count are `return err` and `return nil, err`. A guard that
// returns a fresh ConfigError is a replacement and a guard that returns anything
// the option did not produce is a wrap; both are reported. `nil` is recognised as
// the value half of the two-result form, not as a substitute for the error.
func propagatesErrorUnwrapped(g *ast.IfStmt, name string) error {
	if len(g.Body.List) != 1 {
		return fmt.Errorf("the guard's body is not a single return")
	}
	ret, ok := g.Body.List[0].(*ast.ReturnStmt)
	if !ok {
		return fmt.Errorf("the guard's body is a %T, not a return", g.Body.List[0])
	}
	if len(ret.Results) == 2 {
		if !isIdent(ret.Results[0], "nil") {
			return fmt.Errorf("the guard's two-result form returns %s first, not nil", exprString(ret.Results[0]))
		}
		ret.Results = ret.Results[1:]
	}
	for _, r := range ret.Results {
		switch e := r.(type) {
		case *ast.Ident:
			if e.Name == name || e.Name == "nil" {
				continue
			}
			return fmt.Errorf("the guard returns %s, not the error the option returned", e.Name)
		case *ast.UnaryExpr:
			if e.Op != token.AND {
				return fmt.Errorf("the guard returns an address of %s, not the error the option returned", exprString(e))
			}
			if lit, ok := e.X.(*ast.CompositeLit); ok && structLitType(lit) == "ConfigError" {
				return fmt.Errorf("the guard replaces the option's error with a new ConfigError")
			}
		default:
			return fmt.Errorf("the guard returns %s, which is neither the option's error nor a nil", exprString(r))
		}
	}
	return nil
}

// checkNewClientStartsNoIO requires NewClient's body to make no call that would
// reach the network or start the session. The scan covers the whole body
// including nested closures, because a `go someDialer()` inside a helper literal
// would be exactly the I/O the document rules out.
func checkNewClientStartsNoIO(code *goSrc, fn *ast.FuncDecl) error {
	var found *ast.CallExpr
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if ioCallNames[calleeName(call.Fun)] {
			found = call
			return false
		}
		return true
	})
	if found == nil {
		return nil
	}
	return fmt.Errorf("client construction drift: %s calls %s, so NewClient performs I/O or starts the session; "+
		"%s documents that NewClient does not perform I/O and does not authenticate",
		code.at(found), exprString(found.Fun), clientDoc)
}

// variadicParam returns the name of a function's variadic parameter, or "".
func variadicParam(fn *ast.FuncDecl) string {
	if fn.Type.Params == nil {
		return ""
	}
	for _, p := range fn.Type.Params.List {
		ell, ok := p.Type.(*ast.Ellipsis)
		if !ok {
			continue
		}
		if len(p.Names) > 0 {
			return p.Names[0].Name
		}
		if id, ok := ell.Elt.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

// --- claim 5: the options block -----------------------------------------------

// optionResultType is the single result an option constructor has to return.
//
// It is not read out of the document, which states no result at all — the
// options block writes bare `WithFoo(params)` lines. It follows from the
// document's own Construction section instead: `func NewClient(opts ...Option)`
// plus "Options are functional" claims that each documented line names a value
// usable where an Option is expected, and a declaration returning
// `(Option, error)` is not one. Requiring exactly one result of this type is
// therefore the document's claim, not an assumption about the current shape.
const optionResultType = "Option"

// optionLineRe is the shape of a documented option line: an identifier, a
// parenthesised parameter list, nothing else. The name is not captured — it comes
// out of the parse, so that the document and the declaration are read by the same
// reader rather than by a regex on one side and a parser on the other.
var optionLineRe = regexp.MustCompile(`^\w+\s*\(.*\)\s*$`)

// checkClientOptions verifies every line of the document's options block against
// the function it names.
//
// The name is resolved to a declared package-level func rather than matched
// against a list of known option names, which is the whole point: the defect this
// check exists to catch is a documented option no declaration backs — the block
// named `WithOAuth2JWTKeyPath` until it was corrected on 2026-09-27 — and a name
// list would have to carry that defect to find it.
//
// Parameter *types* are compared, parameter *names* are not, for the reason
// docFunc gives: `WithOAuth2RefreshToken(token string)` and the declaration's
// `WithOAuth2RefreshToken(refreshToken string)` are the same contract.
func checkClientOptions(repoRoot string) error {
	doc, err := parseDoc(repoRoot, clientDoc)
	if err != nil {
		return err
	}
	opts, err := doc.options()
	if err != nil {
		return err
	}
	// A duplicate is reported rather than tolerated: two lines for one
	// constructor means one of them is describing something else, and which one is
	// not a question this program can answer.
	first := map[string]int{}
	for i, o := range opts {
		if prev, dup := first[o.name]; dup {
			return fmt.Errorf("client options drift: %s documents %s at both %s and %s",
				doc.rel, o.name, doc.at(opts[prev].lineNo), doc.at(o.lineNo))
		}
		first[o.name] = i
	}
	for _, o := range opts {
		code, fn, err := resolveFunc(repoRoot, pubIbdAlias, o.name)
		if err != nil {
			return fmt.Errorf("client options drift: %s documents %s, but %w; a caller who copied this line "+
				"would not compile", doc.at(o.lineNo), o.name, err)
		}
		got := codeFunc(fn)
		if !equalStrings(o.params, got.params) {
			return fmt.Errorf("client options drift: %s documents %s but %s declares %s; the documented parameter "+
				"types are what a caller writes at the call site",
				doc.at(o.lineNo), o.render(), code.at(fn), renderedFunc(fn))
		}
		if !equalStrings([]string{optionResultType}, got.results) {
			return fmt.Errorf("client options drift: %s declares %s whose single result is (%s) rather than %s; "+
				"%s documents NewClient as func NewClient(opts ...Option), so a value this line names has to be "+
				"usable as an argument to it",
				code.at(fn), o.name, strings.Join(got.results, ", "), optionResultType, clientDoc)
		}
	}
	return nil
}

// renderedFunc prints a package-level declaration the way the options block
// writes it — `WithFoo(string, int)`. docFunc.render always prints a receiver, so
// it would put "func ()" in front of one of these; docSignature.render is
// receiver-aware, so it is the one used in the message.
func renderedFunc(fn *ast.FuncDecl) string {
	return docSignature{name: fn.Name.Name, params: fieldTypes(fn.Type.Params)}.render()
}

// docOption is one bare `Name(params)` line of the options block.
type docOption struct {
	lineNo int
	name   string
	params []string
}

// render prints the line the way a Go declaration would, minus the `func` and
// the result, so a document and a declaration can be named side by side.
func (o docOption) render() string {
	return o.name + "(" + strings.Join(o.params, ", ") + ")"
}

// options returns the option lines of the document's single options block.
//
// The block is located in two passes, and the split is the point. One pass over
// every ```go block would also run the struct block, the accessor list and the
// close signature through the option reader. Locating first and *then* reading
// means a line inside the options block that has stopped being an option line is
// reported rather than skipped: reading and filtering in one pass would drop
// `WithGatewayURL` out of the comparison silently the moment its parentheses
// went missing, and the check would stay green on a document that no longer
// documents the option at all.
//
// A line that cannot be read as a declaration is likewise an error rather than a
// skip, for the reason funcs gives.
func (d *docFile) options() ([]docOption, error) {
	type span struct{ first, last int }
	var candidates []span
	for _, b := range d.goBlocks() {
		first, last := b.content()
		for no := first; no <= last; no++ {
			if code, ok := d.optionCandidate(no); ok && optionLineRe.MatchString(code) {
				candidates = append(candidates, span{first, last})
				break
			}
		}
	}
	switch {
	case len(candidates) == 0:
		return nil, fmt.Errorf("%s: no ```go block lists an option line of the form `Name(params)`; the check cannot run", d.rel)
	case len(candidates) > 1:
		return nil, fmt.Errorf("%s: %d ```go blocks list option lines, so the check cannot tell which is the options "+
			"block; the check cannot run", d.rel, len(candidates))
	}

	var out []docOption
	first, last := candidates[0].first, candidates[0].last
	for no := first; no <= last; no++ {
		code, isContent := d.optionCandidate(no)
		if !isContent {
			continue
		}
		if !optionLineRe.MatchString(code) {
			return nil, fmt.Errorf("%s: line %d reads %q, which is not an option line of the form `Name(params)`, "+
				"so the check cannot compare it against a declaration", d.rel, no, code)
		}
		o, err := parseOptionLine(no, code)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", d.rel, err)
		}
		out = append(out, o)
	}
	return out, nil
}

// optionCandidate returns line no's code with any trailing comment cut, and
// whether the line carries option-block content at all. Blank lines, `//` group
// headers and `func` signatures are not content: the options block is a list of
// bare option lines, and the block's own group comments are prose a maintainer
// rewords freely.
func (d *docFile) optionCandidate(no int) (string, bool) {
	text := strings.TrimSpace(d.lines[no-1])
	if text == "" || strings.HasPrefix(text, "//") || strings.HasPrefix(text, "func ") {
		return "", false
	}
	code, _ := cutComment(text)
	return strings.TrimSpace(code), true
}

// parseOptionLine reads one bare `Name(params)` line as the Go declaration it
// abbreviates — `WithX(a, b int)` becomes `func WithX(a, b int) Option {}` — and
// reduces it with the same codeFunc the declaration side is reduced with, so the
// two are compared like for like.
//
// Parsing rather than splitting is what makes a grouped parameter come out right.
// The document writes `WithCircuitBreakerBudget(budget, size int)` and the
// declaration is `func WithCircuitBreakerBudget(budget, size int) Option`; a
// comma-splitting reader reads the leading `budget` as a type of its own and
// would report that line as disagreeing with the declaration it names — a
// permanent false positive on a correct document.
//
// The result is dropped rather than compared, because the document states none.
func parseOptionLine(lineNo int, text string) (docOption, error) {
	src := "package p\nfunc " + text + " Option {}\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "option", src, 0)
	if err != nil {
		return docOption{}, fmt.Errorf("line %d: cannot read the documented option %q, so the check cannot compare it: %w",
			lineNo, text, err)
	}
	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		return docOption{}, fmt.Errorf("line %d: cannot read the documented option %q; the check cannot run", lineNo, text)
	}
	got := codeFunc(fn)
	return docOption{lineNo: lineNo, name: got.name, params: got.params}, nil
}

// resolveFunc finds the package-level function named name anywhere in a package,
// and returns the file it was declared in alongside the declaration — the file
// because a failure has to name the line a maintainer has to edit, and the whole
// package because the options are spread over client.go, oauth.go, middleware.go
// and timeout.go.
func resolveFunc(repoRoot, pkgDir, name string) (*goSrc, *ast.FuncDecl, error) {
	files, err := pkgGoFiles(repoRoot, pkgDir)
	if err != nil {
		return nil, nil, err
	}
	for _, rel := range files {
		code, err := parseGo(repoRoot, rel)
		if err != nil {
			return nil, nil, err
		}
		if fn := code.funcDecl(name); fn != nil {
			return code, fn, nil
		}
	}
	return nil, nil, fmt.Errorf("no package-level func %s is declared anywhere in %s", name, pkgDir)
}

// --- claim 4: the close contract ---------------------------------------------

// checkClientCloseContract verifies the document's close semantics: `Close()`
// exists with the documented signature, a second call returns nil, and a request
// made after Close returns ErrClosed.
//
// Idempotence is checked on the mechanism, not the comment. `c.closed.Swap(true)`
// returning true is what makes the second call a no-op, and it is expressible
// without any new code — deleting the guard, or swapping on a plain Load, is the
// whole defect class. The ErrClosed claim is checked on the guard every request
// passes rather than on any one manager method: the managers reach the gateway
// through netDo, so a guard that stopped being there would leave Close harmless
// and every manager method talking to a closed client.
func checkClientCloseContract(repoRoot string) error {
	doc, err := parseDoc(repoRoot, clientDoc)
	if err != nil {
		return err
	}
	code, err := parseGo(repoRoot, clientSrcPath)
	if err != nil {
		return err
	}
	errs, err := parseGo(repoRoot, pubErrSrcPath)
	if err != nil {
		return err
	}
	resp, err := parseGo(repoRoot, respSrcPath)
	if err != nil {
		return err
	}
	sigs, err := doc.signatures()
	if err != nil {
		return err
	}
	want, ok := findSignature(sigs, accessorReceiver, "Close")
	if !ok {
		return fmt.Errorf("%s: no documented `func (c *Client) Close` signature; the close-semantics check cannot run", doc.rel)
	}
	fn := code.method("Client", "Close")
	if fn == nil {
		return fmt.Errorf("%s: no Client.Close method is declared; the close-semantics check cannot run", clientSrcPath)
	}
	got := codeFunc(fn)
	if !equalStrings(want.params, got.params) || !equalStrings(want.results, got.results) {
		return fmt.Errorf("close drift: %s documents %q but %s declares %q",
			doc.at(want.lineNo), want.render(), code.at(fn), got.render())
	}
	if err := checkCloseIsIdempotent(code, fn); err != nil {
		return err
	}
	return checkNetDoGuardsWithErrClosed(code, resp, errs)
}

// closedField is the Client field the document calls the "closed flag".
const closedField = "closed"

// checkCloseIsIdempotent requires Close's first statement to be a guard that
// returns nil when the closed flag is already set.
//
// The field is read out of the document's struct block rather than assumed, and
// the guard's condition has to swap *that* field on the receiver. A different
// flag — a new `closing` bool, say — is reported rather than passed: two flags is
// a state machine the document does not describe.
func checkCloseIsIdempotent(code *goSrc, fn *ast.FuncDecl) error {
	if len(fn.Body.List) == 0 {
		return fmt.Errorf("close drift: %s: Client.Close has an empty body; %s documents Close as idempotent",
			code.at(fn), clientDoc)
	}
	guard, ok := fn.Body.List[0].(*ast.IfStmt)
	if !ok {
		return fmt.Errorf("close drift: %s: the first statement of Client.Close is a %T, not the idempotence guard; "+
			"%s documents Close as idempotent", code.at(fn.Body.List[0]), fn.Body.List[0], clientDoc)
	}
	if !swapsReceiverField(guard.Cond, receiverName(fn), closedField) {
		return fmt.Errorf("close drift: %s: the first statement of Client.Close tests %s, not a swap of %s.%s; "+
			"%s documents Close as idempotent, so a second call has to observe the first and return nil",
			code.at(guard), exprString(guard.Cond), receiverName(fn), closedField, clientDoc)
	}
	if !returnsNil(guard.Body) {
		return fmt.Errorf("close drift: %s: the idempotence guard does not return nil, so a second Close would "+
			"report whatever the second call went on to do; %s documents the second call as returning nil",
			code.at(guard), clientDoc)
	}
	return nil
}

// swapsReceiverField reports whether cond is `recv.field.Swap(true)`. The method
// name and the argument are both required: a `Load`-shaped guard would read the
// flag without changing it, and so would not make the second call a no-op.
func swapsReceiverField(cond ast.Expr, recv, field string) bool {
	call, ok := cond.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	outer, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || outer.Sel.Name != "Swap" {
		return false
	}
	if stringArg(call, 0) != "true" && exprString(call.Args[0]) != "true" {
		return false
	}
	inner, ok := outer.X.(*ast.SelectorExpr)
	if !ok || inner.Sel.Name != field {
		return false
	}
	id, ok := inner.X.(*ast.Ident)
	return ok && id.Name == recv
}

// returnsNil reports whether a block's single statement returns nil.
func returnsNil(body *ast.BlockStmt) bool {
	if len(body.List) != 1 {
		return false
	}
	ret, ok := body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return false
	}
	id, ok := ret.Results[0].(*ast.Ident)
	return ok && id.Name == "nil"
}

// checkNetDoGuardsWithErrClosed requires netDo to be gated on the closed check
// and requires that check to return the public ErrClosed.
//
// Both halves are read from declarations. The public error is resolved through
// the alias in pkg/ibkr/errors.go, so a redefinition of ErrClosed away from
// internal.ErrClosed would be reported: callers match on it with errors.Is, and
// docs/ERRORS.md documents the same sentinel.
func checkNetDoGuardsWithErrClosed(client, resp, errs *goSrc) error {
	netDo := resp.method("Client", "netDo")
	if netDo == nil {
		return fmt.Errorf("%s: no Client.netDo method is declared; the close-semantics check cannot run", respSrcPath)
	}
	if len(netDo.Body.List) == 0 {
		return fmt.Errorf("%s: Client.netDo has an empty body; the close-semantics check cannot run", respSrcPath)
	}
	guard, ok := netDo.Body.List[0].(*ast.IfStmt)
	if !ok {
		return fmt.Errorf("close drift: %s: the first statement of Client.netDo is a %T, not the closed guard; "+
			"%s documents that manager methods return ErrClosed after Close, and netDo is the path every "+
			"request takes", resp.at(netDo.Body.List[0]), netDo.Body.List[0], clientDoc)
	}
	if !callsMethodOnReceiver(guard, "checkOpen") {
		return fmt.Errorf("close drift: %s: netDo's first guard is %s, not a call to Client.checkOpen; %s documents "+
			"that manager methods return ErrClosed after Close, and netDo is where that check has to happen",
			resp.at(guard), exprString(guard.Cond), clientDoc)
	}
	checkOpen := client.method("Client", "checkOpen")
	if checkOpen == nil {
		return fmt.Errorf("%s: no Client.checkOpen method is declared; the close-semantics check cannot run", clientSrcPath)
	}
	if !returnsSentinel(checkOpen.Body, "internal", "ErrClosed") {
		return fmt.Errorf("close drift: %s: Client.checkOpen does not return internal.ErrClosed; %s documents that "+
			"manager methods return ErrClosed after Close, and ErrClosed is that sentinel",
			client.at(checkOpen), clientDoc)
	}
	alias := errs.varSpecErrClosed()
	if alias != "internal.ErrClosed" {
		return fmt.Errorf("close drift: %s binds the public ErrClosed to %q rather than to internal.ErrClosed; the "+
			"sentinel callers match on with errors.Is is not the one the closed guard returns, so %s's claim that "+
			"manager methods return ErrClosed after Close is not the caller's to rely on",
			pubErrSrcPath, alias, clientDoc)
	}
	return nil
}

// varSpecErrClosed returns the right-hand side of the package-level `ErrClosed`
// declaration, or "" when there is none. A grouped `var (...)` block is handled
// because that is how pkg/ibkr/errors.go declares the sentinels.
func (g *goSrc) varSpecErrClosed() string {
	for _, d := range g.file.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}
		for _, s := range gd.Specs {
			vs, ok := s.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, n := range vs.Names {
				if n.Name == "ErrClosed" && i < len(vs.Values) {
					return exprString(vs.Values[i])
				}
			}
		}
	}
	return ""
}

// returnsSentinel reports whether a body returns a single `pkg.Name` value
// anywhere. It walks the whole body rather than the top-level statements, because
// the sentinel is returned from inside a guard — that is the only place it makes
// sense to return it from — and a top-level-only read would report the real
// declaration as absent.
func returnsSentinel(body *ast.BlockStmt, pkg, name string) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			return true
		}
		sel, ok := ret.Results[0].(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != name {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == pkg {
			found = true
		}
		return true
	})
	return found
}

// callsMethodOnReceiver reports whether an if-statement's condition calls a
// method on a value the guard names, or assigns that call to a local. Both are
// the shape a guard takes in this code, and both are read from the guard rather
// than assumed, so a rewritten guard is either understood or reported.
func callsMethodOnReceiver(guard *ast.IfStmt, method string) bool {
	for _, cond := range []ast.Expr{guard.Cond} {
		if callIsMethod(cond, method) {
			return true
		}
	}
	if guard.Init == nil {
		return false
	}
	as, ok := guard.Init.(*ast.AssignStmt)
	if !ok || len(as.Rhs) != 1 {
		return false
	}
	return callIsMethod(as.Rhs[0], method)
}

func callIsMethod(e ast.Expr, method string) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == method
}
