// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

// orders_confirmation.go verifies docs/design/09-orders-and-confirmation.md
// against the code it describes. Order submission is the highest-risk part of
// this SDK, and the defect that motivated this checker lived here: an
// acknowledgement id rendered through a 32-bit float, so a caller polling by that
// id addressed the wrong instruction. Every check below pins one claim the
// document states as fact, and each failure names the file:line on both sides so
// the message points at the line to edit.
//
// What is deliberately NOT checked:
//
//   - The documented field *order* of OrderRequest and Reply. Struct field order
//     is not a contract a caller can observe, so pinning it would make the check
//     fail on a reordering that changes nothing. The two field sets are compared
//     in both directions; the sequence is not. SubmitResult is compared in order
//     too, because the document and the declaration agree on it exactly.
//   - The documented `OrderRequest` and `Reply` field sets are now enforced in
//     both directions. They were not, while the document and the code disagreed:
//     the block said `Message string` where pkg/ibkr/trade.go declares
//     `Messages []string`, and it omitted `ParentID` and `IsSingleGroup`. A
//     human resolved both in favour of the code, so there is nothing left to
//     decline, and the subset-only direction is exactly what let the two missing
//     fields go unnoticed.
//   - The document's Tests section. It is a statement about the test suite, not
//     about production behaviour, and a check that failed on a renamed test would
//     rot into noise. Its one claim about the public API — money fields in
//     OrderRequest are strings — is enforced by checkHandBuiltWireBody.
//   - The `Code == "ambiguous"` half of the error-mapping table.
//     06-errors-retries.md's checkAmbiguousReconcileError already pins that
//     string on the same code line. Duplicating it here would make one fact two
//     checks to maintain and neither stronger. What 09 adds is a claim 06 does
//     not make: that the reconcile instruction names the API this document tells
//     the caller to reach for.

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strconv"
	"strings"
)

const (
	ordersDoc     = "docs/design/09-orders-and-confirmation.md"
	tradeSrcPath  = "pkg/ibkr/trade.go"
	genClientPath = "client/client.gen.go"
	pubClientPath = "pkg/ibkr/client.go"
	pubIbdPkg     = "pkg/ibkr"
)

// moneyTags are the order ticket's money and quantity keys. The document claims
// the wire body carries them as strings because the generated request type
// marshals them as float32, so each is checked on both sides: the hand-built body
// must declare it a string, and the generated type the submit operation actually
// takes must declare the same tag a float32.
var moneyTags = []string{"quantity", "price", "auxPrice"}

// --- claim 1: the endpoints the flow list names -----------------------------

// flowEndpointClaim binds one documented flow step to the TradeManager method
// that implements it. The HTTP verb and the path are not written here: both are
// read out of the document, so editing the document's endpoint is what makes this
// check fail, and the generated client is the only authority on the real one.
//
// index selects which verb+path pair of the step this claim is, which is what
// lets one numbered step ("Modify / Cancel") carry two endpoints. suffixOnly
// marks a step that elides the account prefix with `...`.
type flowEndpointClaim struct {
	what       string
	title      string
	index      int
	suffixOnly bool
	method     string
}

var flowEndpointClaims = []flowEndpointClaim{
	{what: "Submit", title: "Submit", method: "Submit"},
	{what: "Confirm", title: "Confirm", method: "Confirm"},
	{what: "order status", title: "Result", method: "OrderStatus"},
	{what: "Modify", title: "Modify / Cancel", suffixOnly: true, method: "Modify"},
	{what: "Cancel", title: "Modify / Cancel", index: 1, suffixOnly: true, method: "Cancel"},
}

// stepRe recognises a numbered, bold-titled step of the flow list.
var stepRe = regexp.MustCompile(`^(\d+)\.\s+\*\*(.+?)\*\*`)

// verbPathRe recognises a backtick-quoted "<METHOD> <path>" pair.
var verbPathRe = regexp.MustCompile("`(GET|POST|PUT|PATCH|DELETE)\\s+([^`]+)`")

// checkOrderFlowEndpoints verifies steps 1-4 of the document's "IBKR order flow"
// against the endpoints the generated client really requests, and against the
// TradeManager method that makes each call. Three things have to line up for a
// step to be true, and each is resolved from a declaration rather than assumed:
//
//   - the document states a verb and a path,
//   - the method the step names calls exactly one generated operation, and
//   - that operation's request builder builds the documented verb and path.
//
// The middle link is the one a name-only check would miss. Reading the endpoint
// from the operation name alone would still pass if Submit had been rewired to
// Modify's endpoint, which is exactly the rewiring this check exists to catch.
func checkOrderFlowEndpoints(repoRoot string) error {
	doc, err := parseDoc(repoRoot, ordersDoc)
	if err != nil {
		return err
	}
	trade, err := parseGo(repoRoot, tradeSrcPath)
	if err != nil {
		return err
	}
	gen, err := parseGo(repoRoot, genClientPath)
	if err != nil {
		return err
	}
	steps, err := docFlowSteps(doc)
	if err != nil {
		return err
	}

	for _, c := range flowEndpointClaims {
		step, ok := findFlowStep(steps, c.title)
		if !ok {
			return fmt.Errorf("%s: the flow list has no step titled %q; the endpoint check cannot run", doc.rel, c.title)
		}
		pairs := verbPathRe.FindAllStringSubmatch(step.text, -1)
		if c.index >= len(pairs) {
			return fmt.Errorf("%s: step %q states %d HTTP endpoint(s) but the %s claim needs %d; the check cannot run",
				doc.at(step.line), c.title, len(pairs), c.what, c.index+1)
		}
		docVerb, docPath := pairs[c.index][1], pairs[c.index][2]

		fn := trade.method("TradeManager", c.method)
		if fn == nil {
			return fmt.Errorf("%s: no TradeManager.%s method is declared; the endpoint check cannot run", tradeSrcPath, c.method)
		}
		calls := generatedCalls(fn)
		if len(calls) != 1 {
			return fmt.Errorf("%s: TradeManager.%s makes %d generated-client calls but the %s step of %s names one "+
				"endpoint; the check cannot run", trade.at(fn), c.method, len(calls), c.what, doc.rel)
		}
		codeConst, codePath, at, err := generatedOperation(gen, calls[0].name)
		if err != nil {
			return err
		}
		wantConst, known := httpVerbConsts[docVerb]
		if !known {
			return fmt.Errorf("%s: step %d (%s) states the HTTP verb %q, which this checker cannot map to a net/http "+
				"constant; the endpoint check cannot run", doc.at(step.line), step.label, c.what, docVerb)
		}
		codeVerb, knownCode := httpMethodConsts[codeConst]
		if !knownCode {
			return fmt.Errorf("%s: the request builder for %s uses http.%s, which is not a method constant this "+
				"checker knows, so the verb it really requests cannot be read", at, calls[0].name, codeConst)
		}
		if codeConst != wantConst {
			return fmt.Errorf("order flow endpoint drift: %s documents step %d (%s) as %s %s but %s builds %s %s",
				doc.at(step.line), step.label, c.what, docVerb, docPath, at, codeVerb, codePath)
		}
		want := pathParamsToVerbs(docPath)
		// The document elides the account prefix with `...`, so only the tail it
		// spells out is a claim.
		if c.suffixOnly {
			if !strings.HasSuffix(codePath, strings.TrimPrefix(want, "...")) {
				return fmt.Errorf("order flow endpoint drift: %s documents step %d (%s) as %s%s but %s builds %s %s",
					doc.at(step.line), step.label, c.what, docVerb, want, at, codeVerb, codePath)
			}
			continue
		}
		if codePath != want {
			return fmt.Errorf("order flow endpoint drift: %s documents step %d (%s) as %s %s but %s builds %s %s",
				doc.at(step.line), step.label, c.what, docVerb, docPath, at, codeVerb, codePath)
		}
	}
	return nil
}

// flowStep is one numbered step of the document's flow list, with the
// continuation lines that belong to it joined on.
type flowStep struct {
	label int
	title string
	line  int
	text  string
}

// docFlowSteps reads the numbered list under the document's "IBKR order flow"
// heading. The list ends at the first blank line or heading, so a later list in
// the document cannot contribute a step.
func docFlowSteps(doc *docFile) ([]flowStep, error) {
	var out []flowStep
	open := false
	for i, l := range doc.lines {
		if m := stepRe.FindStringSubmatch(l); m != nil {
			label, err := strconv.Atoi(m[1])
			if err != nil {
				return nil, fmt.Errorf("%s: cannot read step number %q; the endpoint check cannot run", doc.at(i+1), m[1])
			}
			out = append(out, flowStep{label: label, title: m[2], line: i + 1, text: l})
			open = true
			continue
		}
		if !open {
			continue
		}
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(t, "```") {
			break
		}
		out[len(out)-1].text += " " + t
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: no numbered flow steps found; the endpoint check cannot run", doc.rel)
	}
	return out, nil
}

func findFlowStep(steps []flowStep, title string) (flowStep, bool) {
	for _, s := range steps {
		if s.title == title {
			return s, true
		}
	}
	return flowStep{}, false
}

// pathParamsRe matches a documented `{param}` placeholder.
var pathParamsRe = regexp.MustCompile(`\{[^}]*\}`)

// pathParamsToVerbs rewrites a documented path's `{param}` placeholders to the
// `%s` verbs the generated builder uses, so the two spellings compare equal
// without either side being normalised into something vaguer.
func pathParamsToVerbs(p string) string {
	return pathParamsRe.ReplaceAllString(p, "%s")
}

// generatedCall is one call a method makes on the generated client, with the
// operation name taken from the declaration it selects.
type generatedCall struct {
	name string
	node ast.Node
}

// generatedCalls returns the generated-client operations fn calls. The receiver
// chain is resolved, not matched: `m.client.generated.Op(...)` and
// `m.generated.Op(...)` are the same call, and a field of some other name on the
// way down is not.
func generatedCalls(fn *ast.FuncDecl) []generatedCall {
	var out []generatedCall
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		switch x := sel.X.(type) {
		case *ast.Ident:
			if x.Name == "generated" {
				out = append(out, generatedCall{name: sel.Sel.Name, node: sel})
			}
		case *ast.SelectorExpr:
			if x.Sel.Name == "generated" {
				out = append(out, generatedCall{name: sel.Sel.Name, node: sel})
			}
		}
		return true
	})
	return out
}

// httpVerbConsts maps a documented HTTP verb to the net/http constant the
// generated builder has to name. The direction is deliberate: the document
// supplies the verb, so the expected constant is derived from the claim rather
// than read back out of the code, and a builder that names some other method
// constant is drift rather than a second opinion. A verb this table does not know
// is reported, not skipped, so a document that grows a verb cannot make the check
// pass by being unrecognised.
var httpVerbConsts = map[string]string{
	"GET":    "MethodGet",
	"HEAD":   "MethodHead",
	"POST":   "MethodPost",
	"PUT":    "MethodPut",
	"PATCH":  "MethodPatch",
	"DELETE": "MethodDelete",
}

// generatedOperation resolves the verb and path a generated client method
// requests. Neither lives on the method: it delegates to a request builder, and
// that builder declares both. The delegation is followed rather than
// reconstructed from the method name, so a method whose builder is spelled
// differently resolves the same way, and a method that reaches no builder is
// reported instead of skipped.
//
// The verb is returned as the net/http constant the builder names, not as a
// method string, so the caller compares it against the constant the document's
// verb implies.
func generatedOperation(gen *goSrc, method string) (verbConst, path, at string, err error) {
	fn := gen.method("Client", method)
	if fn == nil {
		return "", "", "", fmt.Errorf("%s: no Client.%s method is declared; the endpoint check cannot run", genClientPath, method)
	}
	cur := fn
	for hop := 0; hop < 4; hop++ {
		gotVerb, gotPath, verbOK, pathOK := requestShape(cur)
		if verbOK && pathOK {
			return gotVerb, gotPath, gen.at(cur), nil
		}
		if verbOK != pathOK {
			declared, missing := "a verb", "a path"
			if pathOK {
				declared, missing = "a path", "a verb"
			}
			return "", "", "", fmt.Errorf("%s: the request builder for %s declares %s but not %s; "+
				"the endpoint check cannot run", gen.at(cur), method, declared, missing)
		}
		next := requestDelegates(gen, cur)
		if len(next) != 1 {
			return "", "", "", fmt.Errorf("%s: %s reaches %d request builders; the endpoint check cannot run",
				gen.at(cur), method, len(next))
		}
		cur = next[0]
	}
	return "", "", "", fmt.Errorf("%s: could not follow %s to a request builder within 4 hops; "+
		"the endpoint check cannot run", gen.rel, method)
}

// requestShape reads the method constant and path a request builder builds,
// reporting which of the two it found. The verb comes from the http.NewRequest
// argument and the path from the operationPath assignment, because those are the
// two statements that make the request; nothing here trusts a comment.
func requestShape(fn *ast.FuncDecl) (verbConst, path string, verbOK, pathOK bool) {
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.AssignStmt:
			if len(t.Lhs) != 1 || len(t.Rhs) != 1 {
				return true
			}
			id, ok := t.Lhs[0].(*ast.Ident)
			if !ok || id.Name != "operationPath" {
				return true
			}
			call, ok := t.Rhs[0].(*ast.CallExpr)
			if !ok || calleeName(call.Fun) != "Sprintf" || len(call.Args) == 0 {
				return true
			}
			if lit, ok := stringLit(call.Args[0]); ok {
				path, pathOK = lit, true
			}
		case *ast.CallExpr:
			if calleeName(t.Fun) != "NewRequest" || len(t.Args) < 2 {
				return true
			}
			sel, ok := t.Args[0].(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "http" {
				return true
			}
			verbConst, verbOK = sel.Sel.Name, true
		}
		return true
	})
	return verbConst, path, verbOK, pathOK
}

// requestDelegates returns the request-builder functions fn calls.
func requestDelegates(gen *goSrc, fn *ast.FuncDecl) []*ast.FuncDecl {
	var out []*ast.FuncDecl
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		id, ok := call.Fun.(*ast.Ident)
		if !ok || !strings.HasPrefix(id.Name, "New") || !strings.Contains(id.Name, "Request") {
			return true
		}
		if d := gen.funcDecl(id.Name); d != nil {
			out = append(out, d)
		}
		return true
	})
	return out
}

// --- claim 2: the public API the document publishes -------------------------

// checkPublicOrderAPI verifies the document's "Public API" block against the
// declarations it publishes. Every struct in the block is checked in both
// directions, and the two directions are the point:
//
//   - every field the document lists must be declared, with the documented type.
//     This is what a caller writing `OrderRequest{ConID: ..., Quantity: "1.5"}`
//     depends on, and a documented name the code does not declare is a compile
//     error the reader pays for.
//   - every field the code declares must be documented. This is the direction
//     that was missing when the block listed `TimeInForce` before `StopPrice` and
//     omitted `ParentID` and `IsSingleGroup` altogether: a subset check passes
//     happily while a field is missing, and the document is the only place a
//     caller would look for one.
//
// SubmitResult is additionally checked in order, because the document and the
// declaration agree on it exactly. OrderRequest and Reply are not: field order in
// a struct is not a contract a caller can observe, so reordering the document
// would fail the build over a change that means nothing.
//
// The two documented signatures are compared on receiver type, name, parameter
// types and result types. Parameter *names* are not compared, because renaming
// one changes no contract a caller can observe.
func checkPublicOrderAPI(repoRoot string) error {
	doc, err := parseDoc(repoRoot, ordersDoc)
	if err != nil {
		return err
	}
	trade, err := parseGo(repoRoot, tradeSrcPath)
	if err != nil {
		return err
	}

	if err := checkDocumentedStruct(doc, trade, "OrderRequest", matchSet); err != nil {
		return err
	}
	if err := checkDocumentedStruct(doc, trade, "SubmitResult", matchExact); err != nil {
		return err
	}
	if err := checkDocumentedStruct(doc, trade, "Reply", matchSet); err != nil {
		return err
	}

	docFuncs, err := doc.funcs()
	if err != nil {
		return err
	}
	for _, want := range []string{"Submit", "Confirm"} {
		df, ok := findDocFunc(docFuncs, "*TradeManager", want)
		if !ok {
			return fmt.Errorf("%s: no documented TradeManager.%s signature; the public-API check cannot run", doc.rel, want)
		}
		fn := trade.method("TradeManager", want)
		if fn == nil {
			return fmt.Errorf("%s: no TradeManager.%s method is declared; the public-API check cannot run", tradeSrcPath, want)
		}
		got := codeFunc(fn)
		if df.recvType != got.recvType || !equalStrings(df.params, got.params) || !equalStrings(df.results, got.results) {
			return fmt.Errorf("order api drift: %s documents %q but %s declares %q",
				doc.at(df.lineNo), df.render(), trade.at(fn), got.render())
		}
	}

	return checkReconcileEntryPoint(repoRoot)
}

// fieldSetMode says how strictly a documented struct block has to match the
// declaration behind it.
type fieldSetMode int

const (
	// matchSet requires the documented and declared field sets to be equal in
	// both directions, and compares each documented field's type against the
	// declaration. Field *order* is deliberately not compared: struct field order
	// is not a contract a caller can observe, so a reordering the build would
	// reject changes no behaviour.
	matchSet fieldSetMode = iota
	// matchExact additionally requires the documented fields to appear in
	// declaration order and compares the two counts up front, so a block that
	// gained or lost a field is reported with both sides spelled out.
	matchExact
)

// checkDocumentedStruct compares one documented struct block against its
// declaration.
//
// Both modes run the two per-field directions — documented field must be declared
// with the documented type, declared field must be documented — because a check
// that only runs one of them is satisfied by an incomplete document, which is
// exactly the defect that let ParentID and IsSingleGroup go undocumented. Only
// matchExact compares field counts and positions.
func checkDocumentedStruct(doc *docFile, code *goSrc, name string, mode fieldSetMode) error {
	docFields, err := doc.structBlockFields(name)
	if err != nil {
		return err
	}
	realFields, err := code.structFields(name)
	if err != nil {
		return err
	}
	realByName := map[string]codeField{}
	for _, rf := range realFields {
		if rf.name != "" {
			realByName[rf.name] = rf
		}
	}
	docByName := map[string]int{}
	if mode == matchExact && len(docFields) != len(realFields) {
		return fmt.Errorf("order api drift: %s documents %s as %d fields (%s) but %s declares %d (%s)",
			doc.rel, name, len(docFields), joinDocFields(docFields), tradeSrcPath, len(realFields), joinCodeFields(realFields))
	}
	// Direction one: the document to the code. A field the code does not declare,
	// or declares with another type, is a compile error or a silent
	// misreading for whoever wrote their client from this document.
	for i, df := range docFields {
		if prev, dup := docByName[df.name]; dup {
			return fmt.Errorf("order api drift: %s documents %s field %q at both %s and %s",
				doc.rel, name, df.name, doc.at(docFields[prev].lineNo), doc.at(df.lineNo))
		}
		docByName[df.name] = i
		rf, ok := realByName[df.name]
		if !ok {
			return fmt.Errorf("order api drift: %s documents %s field %q, which %s does not declare",
				doc.at(df.lineNo), name, df.name, tradeSrcPath)
		}
		if df.typ != rf.typ {
			return fmt.Errorf("order api drift: %s documents %s field %s as %s but %s declares %s",
				doc.at(df.lineNo), name, df.name, df.typ, code.at(rf.node), rf.typ)
		}
	}
	// Direction two: the code to the document. This is the direction that was
	// missing, and it is the one a reader of the document is relying on: a field
	// the code declares and the document omits is a field nobody can find.
	for _, rf := range realFields {
		if rf.name == "" {
			return fmt.Errorf("order api drift: %s embeds %s at %s, which the documented %s block does not show",
				tradeSrcPath, rf.typ, code.at(rf.node), name)
		}
		if _, ok := docByName[rf.name]; !ok {
			return fmt.Errorf("order api drift: %s declares %s field %s at %s, which %s does not document",
				tradeSrcPath, name, rf.name, code.at(rf.node), doc.rel)
		}
	}
	if mode != matchExact {
		return nil
	}
	for i, df := range docFields {
		if realFields[i].name != df.name {
			return fmt.Errorf("order api drift: %s documents %s field %d as %q but %s declares %q",
				doc.at(df.lineNo), name, i+1, df.name, code.at(realFields[i].node), realFields[i].name)
		}
	}
	return nil
}

func findDocFunc(fs []docFunc, recvType, name string) (docFunc, bool) {
	for _, f := range fs {
		if f.name == name && f.recvType == recvType {
			return f, true
		}
	}
	return docFunc{}, false
}

// checkReconcileEntryPoint verifies the recovery path the document's
// Reconciliation section names. The ambiguous-outcome rule is that the SDK
// returns an error directing the caller to reconcile rather than resubmit, and
// the only way a caller can follow that instruction is for the entry point to
// exist with the documented shape: a caller reading this document and calling
// cli.Trade().OpenOrders(ctx) must compile and reach the gateway.
func checkReconcileEntryPoint(repoRoot string) error {
	client, err := parseGo(repoRoot, pubClientPath)
	if err != nil {
		return err
	}
	trade, err := parseGo(repoRoot, tradeSrcPath)
	if err != nil {
		return err
	}
	accessor := client.method("Client", "Trade")
	if accessor == nil {
		return fmt.Errorf("%s: no Client.Trade method is declared; the reconciliation check cannot run", pubClientPath)
	}
	ret := ""
	if accessor.Type.Results != nil && len(accessor.Type.Results.List) == 1 {
		ret = exprString(accessor.Type.Results.List[0].Type)
	}
	if ret != "*TradeManager" {
		return fmt.Errorf("%s: the reconciliation example calls cli.Trade().OpenOrders(ctx) but Client.Trade returns %s, not *TradeManager",
			client.at(accessor), ret)
	}
	open := trade.method("TradeManager", "OpenOrders")
	if open == nil {
		return fmt.Errorf("%s: no TradeManager.OpenOrders method is declared; the reconciliation check cannot run", tradeSrcPath)
	}
	// The documented call passes only a context. Any other parameter is a
	// signature the document's example no longer satisfies.
	if p := open.Type.Params; p == nil || len(p.List) != 1 || exprString(p.List[0].Type) != "context.Context" {
		return fmt.Errorf("%s: the reconciliation example calls OpenOrders(ctx) but the method declares %q; "+
			"the documented recovery path would not compile", trade.at(open), codeFunc(open).render())
	}
	return nil
}

// --- claim 3: the wire body is built by hand ---------------------------------

// handBuiltBodyRe recognises the sentence that justifies the hand-built wire
// body. It is required rather than assumed so a document that stops making the
// claim is reported instead of quietly leaving the check with nothing to verify.
// The whitespace between the words is matched loosely because the document wraps
// the sentence; the words themselves are not, so a reworded justification fails.
var handBuiltBodyRe = regexp.MustCompile(
	"built by hand \\(`orderTicketJSON`\\) with money/quantity as\\s+strings")

// checkHandBuiltWireBody verifies the document's justification for not using the
// generated request type, in both directions. The claim is a comparison, and
// either side alone would be a check that could be satisfied by accident:
//
//   - the hand-built orderTicketJSON declares quantity, price and auxPrice as
//     strings, and OrderRequest.toJSON copies each one across untouched. A parse
//     or a reformat on the way through would reintroduce the precision loss ADR
//     0008 exists to prevent, and would not change any type.
//   - Submit marshals that hand-built envelope and hands the generated operation
//     a reader, so the generated model is bypassed rather than merely unused.
//   - and the generated request type the submit operation actually takes declares
//     the same three tags as float32. That one is resolved through the alias the
//     generated code declares and then the element type of the `orders` slice,
//     because client.gen.go declares `json:"quantity"` on five unrelated types
//     and a tag search would read one of those and pass on a type nobody submits.
func checkHandBuiltWireBody(repoRoot string) error {
	doc, err := parseDoc(repoRoot, ordersDoc)
	if err != nil {
		return err
	}
	trade, err := parseGo(repoRoot, tradeSrcPath)
	if err != nil {
		return err
	}
	gen, err := parseGo(repoRoot, genClientPath)
	if err != nil {
		return err
	}
	claimLine, ok := docSentence(handBuiltBodyRe, doc)
	if !ok {
		return fmt.Errorf("%s: the sentence justifying the hand-built wire body is gone or reworded; "+
			"the money-precision check cannot run", doc.rel)
	}

	tickets, err := trade.taggedFields("orderTicketJSON")
	if err != nil {
		return err
	}
	for _, tag := range moneyTags {
		f, ok := fieldByTag(tickets, tag)
		if !ok {
			return fmt.Errorf("money precision drift: %s: says the wire body carries money/quantity as strings but "+
				"%s declares no %q field, so there is nothing to check", doc.at(claimLine), tradeSrcPath, tag)
		}
		if f.typ != "string" {
			return fmt.Errorf("money precision drift: %s: declares %q as %s, but %s says the wire body carries "+
				"money/quantity as strings and justifies it by the generated type's float32",
				trade.at(f.node), tag, f.typ, doc.at(claimLine))
		}
	}

	if err := checkToJSONCopiesVerbatim(trade, tickets); err != nil {
		return err
	}
	if err := checkSubmitMarshalsHandBuiltBody(trade); err != nil {
		return err
	}
	return checkGeneratedTypeUsesFloat32(doc, gen, claimLine)
}

// checkToJSONCopiesVerbatim requires each money key to be filled from the
// corresponding OrderRequest field and nothing else. A bare selector is the
// mechanism: any conversion, reformat or arithmetic between the two sides is the
// class of defect that truncated an acknowledgement id, and it would not change
// any type this checker can see. The wire key is taken from the json tag, so
// renaming the Go field is not drift while repointing the value is.
func checkToJSONCopiesVerbatim(trade *goSrc, tickets []taggedField) error {
	fn := trade.method("OrderRequest", "toJSON")
	if fn == nil {
		return fmt.Errorf("%s: no OrderRequest.toJSON method is declared; the money-precision check cannot run", tradeSrcPath)
	}
	recv := receiverName(fn)
	lit := firstStructLit(fn.Body, "orderTicketJSON")
	if lit == nil {
		return fmt.Errorf("%s: OrderRequest.toJSON builds no orderTicketJSON literal; the money-precision check "+
			"cannot run", trade.at(fn))
	}
	byKey := map[string]keyedValue{}
	for _, kv := range keyedValues(lit) {
		byKey[kv.key] = kv
	}
	for _, want := range []struct{ tag, from string }{
		{"quantity", "Quantity"},
		{"price", "LimitPrice"},
		{"auxPrice", "StopPrice"},
	} {
		f, ok := fieldByTag(tickets, want.tag)
		if !ok {
			return fmt.Errorf("%s: the wire body declares no %q field; the money-precision check cannot run", tradeSrcPath, want.tag)
		}
		kv, ok := byKey[f.name]
		if !ok {
			return fmt.Errorf("money precision drift: %s: the wire body sets no %q value, so the order ticket no "+
				"longer carries the documented one", trade.at(lit), f.name)
		}
		sel, ok := kv.expr.(*ast.SelectorExpr)
		if !ok {
			return fmt.Errorf("money precision drift: %s: the wire body's %s is %s, not a copy of OrderRequest.%s; "+
				"the documented value has to reach the wire untouched",
				trade.at(kv.node), f.name, exprString(kv.expr), want.from)
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok || id.Name != recv || sel.Sel.Name != want.from {
			return fmt.Errorf("money precision drift: %s: the wire body's %s is %s, but %s documents it carrying "+
				"OrderRequest.%s",
				trade.at(kv.node), f.name, exprString(kv.expr), ordersDoc, want.from)
		}
	}
	return nil
}

// checkSubmitMarshalsHandBuiltBody requires Submit to marshal the hand-built
// envelope and to hand the generated operation a reader. Using the typed-body
// variant would put the generated model back on the wire while the document still
// claimed the body was built by hand, and no field-type check would see it.
func checkSubmitMarshalsHandBuiltBody(trade *goSrc) error {
	fn := trade.method("TradeManager", "Submit")
	if fn == nil {
		return fmt.Errorf("%s: no TradeManager.Submit method is declared; the money-precision check cannot run", tradeSrcPath)
	}
	env := firstCompositeLitArg(fn, "Marshal")
	if env == nil {
		return fmt.Errorf("money precision drift: %s: Submit marshals no composite literal, so the hand-built wire "+
			"body the document describes is not what this method sends", trade.at(fn))
	}
	if got := litTypeName(env); got != "ordersSubmissionJSON" {
		return fmt.Errorf("money precision drift: %s: Submit marshals a %s literal, not the hand-built "+
			"ordersSubmissionJSON envelope the document describes", trade.at(env), got)
	}
	// The envelope's orders slice has to hold the hand-built ticket type, read
	// from the declaration rather than from the call site.
	envFields, err := trade.taggedFields("ordersSubmissionJSON")
	if err != nil {
		return err
	}
	orders, ok := fieldByTag(envFields, "orders")
	if !ok {
		return fmt.Errorf("%s declares no `orders` field; the money-precision check cannot run", tradeSrcPath)
	}
	if orders.typ != "[]orderTicketJSON" {
		return fmt.Errorf("money precision drift: %s declares orders as %q, not []orderTicketJSON, so the hand-built "+
			"ticket type is not what the envelope sends", tradeSrcPath, orders.typ)
	}
	calls := generatedCalls(fn)
	if len(calls) != 1 {
		return fmt.Errorf("%s: Submit makes %d generated-client calls but the hand-built body check needs one; "+
			"it cannot run", trade.at(fn), len(calls))
	}
	if !strings.HasSuffix(calls[0].name, "WithBody") {
		return fmt.Errorf("money precision drift: %s: Submit calls the typed-body variant %s, which marshals the "+
			"generated request model; %s says the wire body is built by hand because that model is float32",
			trade.at(calls[0].node), calls[0].name, ordersDoc)
	}
	return nil
}

// checkGeneratedTypeUsesFloat32 reads the request body type the submit operation
// declares, follows the alias the generated code uses, then the element type of
// its `orders` field, and requires the three money tags to be float32 there. The
// resolution is the point: see the note on checkHandBuiltWireBody.
func checkGeneratedTypeUsesFloat32(doc *docFile, gen *goSrc, claimLine int) error {
	ticket, err := generatedOrderTicketType(gen, "SubmitNewOrder")
	if err != nil {
		return err
	}
	fields, err := gen.taggedFields(ticket)
	if err != nil {
		return err
	}
	for _, tag := range moneyTags {
		f, ok := fieldByTag(fields, tag)
		if !ok {
			return fmt.Errorf("money precision drift: %s: justifies the hand-built wire body by saying the generated "+
				"request type marshals money/quantity as float32, but the generated type the submit operation takes (%s) "+
				"declares no %q field", doc.at(claimLine), ticket, tag)
		}
		if f.typ != "float32" && strings.TrimPrefix(f.typ, "*") != "float32" {
			return fmt.Errorf("money precision drift: %s: justifies the hand-built wire body by saying the generated "+
				"request type marshals money/quantity as float32, but %s declares %q as %s",
				doc.at(claimLine), gen.at(f.node), tag, f.typ)
		}
	}
	return nil
}

// scalarTypeNames are the parameter types an operation's signature can carry
// without carrying a request body.
var scalarTypeNames = map[string]bool{
	"context.Context": true, "string": true, "int": true, "int64": true,
	"bool": true, "float32": true, "float64": true, "interface{}": true,
}

// generatedOrderTicketType resolves the generated struct that carries one order
// ticket for the named operation: the operation's typed request body, through the
// alias the generated code declares for it, then the element type of its `orders`
// field.
func generatedOrderTicketType(gen *goSrc, op string) (string, error) {
	ft := gen.ifaceMethod("ClientInterface", op)
	if ft == nil {
		return "", fmt.Errorf("%s: ClientInterface declares no %s method; the money-precision check cannot run", gen.rel, op)
	}
	body := ""
	for _, f := range ft.Params.List {
		id, ok := f.Type.(*ast.Ident)
		if !ok || scalarTypeNames[id.Name] {
			continue
		}
		body = id.Name
	}
	if body == "" {
		return "", fmt.Errorf("%s: the %s method on ClientInterface declares no typed request body; "+
			"the money-precision check cannot run", gen.rel, op)
	}
	// `type X = Y` is an alias, and Y is the real declaration; `type X Y` is a
	// distinct type, so the loop stops at the first non-alias. Following the
	// alias is what stops a redeclared type from being read as the one the
	// operation takes.
	resolved := body
	for range 4 {
		ts := gen.typeSpec(resolved)
		if ts == nil {
			return "", fmt.Errorf("%s: type %s is not declared; the money-precision check cannot run", gen.rel, resolved)
		}
		if !ts.Assign.IsValid() {
			break
		}
		id, ok := ts.Type.(*ast.Ident)
		if !ok {
			return "", fmt.Errorf("%s: type %s aliases %s, which is not a named type; the money-precision check cannot run",
				gen.rel, resolved, exprString(ts.Type))
		}
		resolved = id.Name
	}
	ts := gen.typeSpec(resolved)
	if ts == nil {
		return "", fmt.Errorf("%s: type %s is not declared; the money-precision check cannot run", gen.rel, resolved)
	}
	if _, ok := ts.Type.(*ast.StructType); !ok {
		return "", fmt.Errorf("%s: type %s is declared but is not a struct, so it cannot carry the order ticket; "+
			"the money-precision check cannot run", gen.rel, resolved)
	}
	fields, err := gen.taggedFields(resolved)
	if err != nil {
		return "", err
	}
	orders, ok := fieldByTag(fields, "orders")
	if !ok {
		return "", fmt.Errorf("%s: %s declares no `orders` field; the money-precision check cannot run", gen.rel, resolved)
	}
	elem := strings.TrimPrefix(strings.TrimPrefix(orders.typ, "*"), "[]")
	if elem == "" || elem == orders.typ {
		return "", fmt.Errorf("%s: %s.orders is declared as %q, not a slice of a named order-ticket type; "+
			"the money-precision check cannot run", gen.rel, resolved, orders.typ)
	}
	return elem, nil
}

// fieldByTag returns the single field carrying the given json tag.
func fieldByTag(fields []taggedField, tag string) (taggedField, bool) {
	var found taggedField
	n := 0
	for _, f := range fields {
		if f.tag == tag {
			found, n = f, n+1
		}
	}
	return found, n == 1
}

// litTypeName names the type a composite literal constructs, keeping a package
// qualifier: `&client.OrdersSubmissionRequest{...}` is named
// "client.OrdersSubmissionRequest", not "". The shared structLitType only
// resolves an unqualified name, and a message that said "marshals a  literal"
// would name no type at all for a maintainer to look up.
func litTypeName(lit *ast.CompositeLit) string {
	e := lit.Type
	if u, ok := e.(*ast.UnaryExpr); ok && u.Op == token.AND {
		e = u.X
	}
	return exprString(e)
}

// firstStructLit returns the first composite literal of the named struct type in
// a body, or nil.
func firstStructLit(body *ast.BlockStmt, typ string) *ast.CompositeLit {
	var found *ast.CompositeLit
	ast.Inspect(body, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		if cl, ok := n.(*ast.CompositeLit); ok && structLitType(cl) == typ {
			found = cl
		}
		return true
	})
	return found
}

// firstCompositeLitArg returns the first composite-literal argument to a call to
// the named function.
func firstCompositeLitArg(fn *ast.FuncDecl, callee string) *ast.CompositeLit {
	var found *ast.CompositeLit
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 || calleeName(call.Fun) != callee {
			return true
		}
		if cl, ok := call.Args[0].(*ast.CompositeLit); ok {
			found = cl
		}
		return true
	})
	return found
}

// docSentence returns the 1-based line where the first match of re starts. The
// pattern is matched against the whole document rather than line by line, because
// a claim the author wrapped across two lines is still the claim; a pattern that
// needs one line to itself must anchor with (?m) so `^` and `$` keep working.
func docSentence(re *regexp.Regexp, doc *docFile) (int, bool) {
	text := strings.Join(doc.lines, "\n")
	loc := re.FindStringIndex(text)
	if loc == nil {
		return 0, false
	}
	return strings.Count(text[:loc[0]], "\n") + 1, true
}

// --- claim 4: confirmation is explicit ---------------------------------------

// explicitConfirmRe recognises the safety rule that confirmation is the caller's
// decision.
var explicitConfirmRe = regexp.MustCompile(
	`Confirmation is \*\*explicit\*\*: the manager does not silently auto-confirm`)

// checkConfirmationIsExplicit verifies the document's third safety rule: the SDK
// does not silently auto-confirm order warnings. The rule is a statement about
// every call site, not about one function, so it is checked in both directions:
//
//   - TradeManager.Confirm is the only place in pkg/ibkr that reaches the reply
//     endpoint. A second caller anywhere in the package — a helper, a manager, a
//     retry path — would break the rule whatever Confirm itself does, and the
//     package scan is what finds it.
//   - Confirm forwards the caller's reply id and the caller's confirmed flag
//     untouched. Auto-confirming is expressible here without any new code:
//     replacing the flag with a literal true. The flag also has to be a
//     `confirmed` json field with no omitempty, because omitempty would drop
//     false and leave the gateway an empty body — the opposite of the
//     documented "false cancels the pending order".
func checkConfirmationIsExplicit(repoRoot string) error {
	doc, err := parseDoc(repoRoot, ordersDoc)
	if err != nil {
		return err
	}
	trade, err := parseGo(repoRoot, tradeSrcPath)
	if err != nil {
		return err
	}
	gen, err := parseGo(repoRoot, genClientPath)
	if err != nil {
		return err
	}
	if _, ok := docSentence(explicitConfirmRe, doc); !ok {
		return fmt.Errorf("%s: the document no longer states that confirmation is explicit; "+
			"the auto-confirm check cannot run", doc.rel)
	}

	confirm := trade.method("TradeManager", "Confirm")
	if confirm == nil {
		return fmt.Errorf("%s: no TradeManager.Confirm method is declared; the auto-confirm check cannot run", tradeSrcPath)
	}
	calls := generatedCalls(confirm)
	if len(calls) != 1 {
		return fmt.Errorf("%s: TradeManager.Confirm makes %d generated-client calls but the auto-confirm check needs "+
			"the one that answers a reply; it cannot run", trade.at(confirm), len(calls))
	}
	replyID := paramOfType(confirm.Type, "string")
	confirmed := paramOfType(confirm.Type, "bool")
	if replyID == "" || confirmed == "" {
		return fmt.Errorf("%s: TradeManager.Confirm declares %s, so the reply id or the confirmed flag cannot be "+
			"traced; the auto-confirm check cannot run", trade.at(confirm), codeFunc(confirm).render())
	}

	if err := checkReplyCallForwardsArgs(trade, gen, confirm, calls[0].name, replyID, confirmed); err != nil {
		return err
	}
	if err := checkConfirmBodyFlag(trade, confirmed); err != nil {
		return err
	}
	return checkOnlyConfirmAnswersReplies(repoRoot, calls[0].name, doc.rel)
}

// checkReplyCallForwardsArgs requires the call Confirm makes to pass the reply id
// it was given, and to put the caller's confirmed flag on the wire. The two
// positions are resolved from the generated method's own signature: the reader
// parameter is the only one of its type, and the reply id is found by the name the
// generated declaration gives it. A name here only locates a position — what
// decides the claim is the argument the call actually passes.
func checkReplyCallForwardsArgs(trade, gen *goSrc, confirm *ast.FuncDecl, op, replyID, confirmed string) error {
	genFn := gen.method("Client", op)
	if genFn == nil {
		return fmt.Errorf("%s: no Client.%s method is declared; the auto-confirm check cannot run", genClientPath, op)
	}
	params := fieldTypes(genFn.Type.Params)
	replyIdx := paramNamed(genFn.Type.Params, "replyId")
	if replyIdx < 0 {
		return fmt.Errorf("%s: the generated %s signature names no reply-id parameter, so the position the "+
			"documented reply id must reach cannot be resolved; the auto-confirm check cannot run",
			gen.at(genFn), op)
	}
	readerIdx := -1
	for i, t := range params {
		if t == "io.Reader" {
			readerIdx = i
		}
	}
	if readerIdx < 0 {
		return fmt.Errorf("%s: the generated %s signature takes no io.Reader, so it has no hand-built body to send; "+
			"the auto-confirm check cannot run", gen.rel, op)
	}

	call := callNamed(confirm, op)
	if call == nil {
		return fmt.Errorf("%s: TradeManager.Confirm no longer calls %s; %s documents the confirmation step as a "+
			"call to the reply endpoint with the reply id set", trade.at(confirm), op, ordersDoc)
	}
	if replyIdx >= len(call.Args) {
		return fmt.Errorf("%s: %s is called with %d arguments but its signature declares more; "+
			"the auto-confirm check cannot run", trade.at(call), op, len(call.Args))
	}
	if got := exprString(call.Args[replyIdx]); got != replyID {
		return fmt.Errorf("auto-confirm drift: %s: %s is called with %s where the reply id %q belongs; %s documents "+
			"the confirmation step as a call with the reply id set", trade.at(call.Args[replyIdx]), op, got, replyID, ordersDoc)
	}
	if readerIdx >= len(call.Args) {
		return fmt.Errorf("%s: %s is called with %d arguments but its signature declares more; "+
			"the auto-confirm check cannot run", trade.at(call), op, len(call.Args))
	}
	if !readsMarshalOf(confirm, call.Args[readerIdx], confirmed) {
		return fmt.Errorf("auto-confirm drift: %s: the confirmation body is %s, not the caller's confirmed flag "+
			"marshalled; %s documents that the caller decides whether to confirm",
			trade.at(call.Args[readerIdx]), exprString(call.Args[readerIdx]), ordersDoc)
	}
	return nil
}

// paramNamed returns the position of the parameter with the given name, or -1.
func paramNamed(fl *ast.FieldList, name string) int {
	i := 0
	for _, f := range fl.List {
		for _, n := range f.Names {
			if n.Name == name {
				return i
			}
			i++
		}
		if len(f.Names) == 0 {
			i++
		}
	}
	return -1
}

// callNamed returns the first call in fn to the named method, or nil.
func callNamed(fn *ast.FuncDecl, name string) *ast.CallExpr {
	var found *ast.CallExpr
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == name {
			found = call
		}
		return true
	})
	return found
}

// readsMarshalOf reports whether expr is, directly or through a local, a reader
// over a marshal of a composite literal that assigns the identifier name to some
// field. The local hop is not a convenience: the code under test marshals the
// body into a variable and hands the generated call a reader over that variable,
// so a checker that only understood the inline form would never see this claim.
func readsMarshalOf(fn *ast.FuncDecl, expr ast.Expr, name string) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	if isReaderConstructor(calleeName(call.Fun)) {
		expr = call.Args[0]
	}
	if id, ok := expr.(*ast.Ident); ok {
		expr = assignedValue(fn, id.Name)
	}
	marshal, ok := expr.(*ast.CallExpr)
	if !ok || len(marshal.Args) != 1 || calleeName(marshal.Fun) != "Marshal" {
		return false
	}
	lit, ok := marshal.Args[0].(*ast.CompositeLit)
	if !ok {
		return false
	}
	for _, kv := range keyedValues(lit) {
		if exprString(kv.expr) == name {
			return true
		}
	}
	return false
}

// isReaderConstructor reports whether a callee name builds a reader over a byte
// slice. The set is closed so a new reader type is reported rather than skipped.
func isReaderConstructor(name string) bool {
	switch name {
	case "NewReader", "NewBuffer", "NewReadSeeker", "NewStringReader":
		return true
	}
	return false
}

// assignedValue returns the expression a local of that name was last assigned
// from, or nil.
func assignedValue(fn *ast.FuncDecl, name string) ast.Expr {
	var found ast.Expr
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		as, ok := n.(*ast.AssignStmt)
		if !ok || len(as.Lhs) == 0 {
			return true
		}
		for i, lhs := range as.Lhs {
			id, ok := lhs.(*ast.Ident)
			if !ok || id.Name != name || i >= len(as.Rhs) {
				continue
			}
			found = as.Rhs[i]
		}
		return true
	})
	return found
}

// checkConfirmBodyFlag requires the confirmation envelope to carry the caller's
// flag as a `confirmed` field with no omitempty. omitempty on a bool would drop
// false, leaving the gateway an empty body, so a false would no longer mean "cancel
// the pending order" the way the documented signature says it does.
//
// The flag is found by its Go name case-insensitively — the parameter is
// `confirmed` and the field `Confirmed`, which is Go's own convention — or by its
// tag. Matching case-sensitively on the parameter name would miss the field that
// is actually the flag and report a missing field where the real fault is a
// renamed tag, which sends a maintainer looking in the wrong place.
func checkConfirmBodyFlag(trade *goSrc, confirmed string) error {
	fields, err := trade.taggedFields("confirmReplyJSON")
	if err != nil {
		return err
	}
	var found *taggedField
	for i := range fields {
		if strings.EqualFold(fields[i].name, confirmed) || fields[i].tag == "confirmed" {
			found = &fields[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("auto-confirm drift: %s declares no confirmation flag field; %s documents Confirm's "+
			"confirmed argument as what the gateway is told", tradeSrcPath, ordersDoc)
	}
	if found.typ != "bool" {
		return fmt.Errorf("auto-confirm drift: %s: the confirmation flag is %s, not a bool; %s documents "+
			"Confirm(ctx, replyID, confirmed bool)", trade.at(found.node), found.typ, ordersDoc)
	}
	if found.tag != "confirmed" {
		return fmt.Errorf("auto-confirm drift: %s: the confirmation flag is serialised as %q, not \"confirmed\", so "+
			"the gateway would not see the caller's decision", trade.at(found.node), found.tag)
	}
	if strings.Contains(found.tagFull, "omitempty") {
		return fmt.Errorf("auto-confirm drift: %s: the confirmation flag's json tag carries omitempty, so "+
			"confirmed=false would serialise as an empty body; %s documents false as cancelling the pending order",
			trade.at(found.node), ordersDoc)
	}
	return nil
}

// checkOnlyConfirmAnswersReplies scans every non-test file in pkg/ibkr for a call
// to the reply endpoint and requires the only one to be inside TradeManager.Confirm.
// The document's rule is about the whole package: a helper that answered a warning
// on the caller's behalf would satisfy every check above and still break the
// contract.
func checkOnlyConfirmAnswersReplies(repoRoot, op, docRel string) error {
	files, err := pkgGoFiles(repoRoot, pubIbdPkg)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("%s: no Go files found; the auto-confirm check cannot run", pubIbdPkg)
	}
	for _, rel := range files {
		code, err := parseGo(repoRoot, rel)
		if err != nil {
			return err
		}
		for _, d := range code.file.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			for _, c := range generatedCalls(fn) {
				if c.name != op {
					continue
				}
				where := "package-level function " + fn.Name.Name
				if fn.Recv != nil {
					where = recvBaseType(fn.Recv.List[0].Type) + "." + fn.Name.Name
				}
				if fn.Recv != nil && recvBaseType(fn.Recv.List[0].Type) == "TradeManager" && fn.Name.Name == "Confirm" {
					continue
				}
				return fmt.Errorf("auto-confirm drift: %s: %s calls the reply-confirmation operation %s, so a "+
					"caller could reach a confirmation %s did not ask for; only TradeManager.Confirm may",
					code.at(c.node), where, op, docRel)
			}
		}
	}
	return nil
}

// --- claim 5: order mutations go through the single-attempt path --------------

// noRetryVerbsRe reads the operations out of the document's no-automatic-retry
// rule, so the set of mutations this checks is the set the document names.
var noRetryVerbsRe = regexp.MustCompile(`^\s*[-*]\s+\*\*No automatic retry\*\*\s+of\s+(.+?)\s*$`)

// mutationMethods maps a documented operation to the TradeManager method that
// performs it. An operation the document names and this table does not know is
// reported, so the rule cannot grow an unchecked mutation.
var mutationMethods = map[string]string{
	"submit":  "Submit",
	"modify":  "Modify",
	"cancel":  "Cancel",
	"confirm": "Confirm",
}

// checkOrderCallsGoThroughMutate verifies the document's first safety rule — no
// automatic retry of submit/modify/cancel/confirm — by checking the call path
// rather than the sentence. Each named method has to reach the gateway through
// TradeManager.mutate, and must not call the retrying netDo itself. A method can
// keep its name, its comment and its ADR reference while a refactor moves it onto
// netDo; only the call path shows that.
//
// Two more things are checked from the same call graph, both of which the sentence
// implies and neither of which it states: mutate must still reach the gateway
// through netDo, and every method that routes through mutate must be one the rule
// names — otherwise an undocumented mutation would be silently covered by the
// rule instead of documented by it.
//
// The half of the rule that turns a timeout into the `ambiguous` reconcile error
// is not repeated here: 06-errors-retries.md's checkAmbiguousReconcileError pins
// that error on this same file, and one fact does not need two checks.
func checkOrderCallsGoThroughMutate(repoRoot string) error {
	doc, err := parseDoc(repoRoot, ordersDoc)
	if err != nil {
		return err
	}
	trade, err := parseGo(repoRoot, tradeSrcPath)
	if err != nil {
		return err
	}
	verbs, line, ok := docNoRetryVerbs(doc)
	if !ok {
		return fmt.Errorf("%s: the no-automatic-retry rule does not name the operations it covers; "+
			"the single-attempt check cannot run", doc.rel)
	}
	if len(verbs) == 0 {
		return fmt.Errorf("%s: the no-automatic-retry rule names no operations; the single-attempt check cannot run",
			doc.at(line))
	}
	mutate := trade.method("TradeManager", "mutate")
	if mutate == nil {
		return fmt.Errorf("%s: no TradeManager.mutate method is declared; the single-attempt check cannot run", tradeSrcPath)
	}
	mutRecv := receiverName(mutate)
	if !callsMethod(mutate, mutRecv, "netDo") {
		return fmt.Errorf("%s: TradeManager.mutate no longer reaches the gateway through Client.netDo; the "+
			"single-attempt path %s documents for %v is gone", trade.at(mutate), ordersDoc, verbs)
	}
	covered := map[string]bool{}
	for _, v := range verbs {
		method, known := mutationMethods[v]
		if !known {
			return fmt.Errorf("%s: the no-automatic-retry rule names %q, which this checker does not know how to "+
				"verify; the check cannot run", doc.at(line), v)
		}
		covered[method] = true
		fn := trade.method("TradeManager", method)
		if fn == nil {
			return fmt.Errorf("%s: no TradeManager.%s method is declared; the single-attempt check cannot run",
				tradeSrcPath, method)
		}
		recv := receiverName(fn)
		if !callsMethod(fn, recv, "mutate") {
			return fmt.Errorf("order retry drift: %s: TradeManager.%s does not go through %s.mutate, so %s's rule "+
				"that %s is attempted once no longer describes this method",
				trade.at(fn), method, recv, ordersDoc, v)
		}
		if callsMethod(fn, recv, "netDo") {
			return fmt.Errorf("order retry drift: %s: TradeManager.%s calls netDo directly instead of through "+
				"%s.mutate; %s documents %s as a single attempt that reports an ambiguous outcome",
				trade.at(fn), method, recv, ordersDoc, v)
		}
	}
	for _, caller := range callersOfMethod(trade, "TradeManager", "mutate") {
		if !covered[caller] {
			return fmt.Errorf("order retry drift: %s routes TradeManager.%s through mutate but %s's no-automatic-retry "+
				"rule does not name it, so a single-attempt mutation is undocumented",
				tradeSrcPath, caller, ordersDoc)
		}
	}
	return nil
}

// docNoRetryVerbs reads the operations the no-automatic-retry rule names.
func docNoRetryVerbs(doc *docFile) ([]string, int, bool) {
	for i, l := range doc.lines {
		m := noRetryVerbsRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		raw := strings.TrimSpace(m[1])
		var out []string
		for _, p := range strings.Split(raw, "/") {
			p = strings.ToLower(strings.Trim(strings.TrimSpace(p), "`."))
			if p != "" {
				out = append(out, p)
			}
		}
		return out, i + 1, true
	}
	return nil, 0, false
}

// callsMethod reports whether fn calls a field or method named name on its
// receiver, however deep the receiver chain is: `m.mutate` and `m.client.netDo`
// are both a call on m.
func callsMethod(fn *ast.FuncDecl, recv, name string) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != name {
			return true
		}
		if id, ok := rootIdent(sel.X); ok && id.Name == recv {
			found = true
		}
		return true
	})
	return found
}

// rootIdent returns the identifier at the end of a receiver chain: m in
// m, m in m.client and m in m.client.netDo.
func rootIdent(e ast.Expr) (*ast.Ident, bool) {
	for {
		switch t := e.(type) {
		case *ast.Ident:
			return t, true
		case *ast.SelectorExpr:
			e = t.X
		default:
			return nil, false
		}
	}
}

// callersOfMethod returns the names of the methods on recv that call recv.name.
func callersOfMethod(code *goSrc, recv, name string) []string {
	var out []string
	for _, d := range code.file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || recvBaseType(fn.Recv.List[0].Type) != recv || fn.Name.Name == name {
			continue
		}
		if callsMethod(fn, receiverName(fn), name) {
			out = append(out, fn.Name.Name)
		}
	}
	return out
}

// --- claim 6: the error-mapping table ----------------------------------------

// The three rows of the document's error-mapping table, each requiring the row to
// still say what it says. A table that has been reworded or dropped leaves these
// checks with nothing to verify, which must be reported rather than assumed.
var errorTableRows = []struct {
	what string
	re   *regexp.Regexp
}{
	{"Rejected by IBKR", regexp.MustCompile(`(?m)^\|\s*Rejected by IBKR\s*\|.*` + "`\\*ibkr\\.Error` wrapping `ErrOrderRejected`")},
	{"Requires confirmation", regexp.MustCompile(`(?m)^\|\s*Requires confirmation\s*\|.*` + "`SubmitResult\\.Replies` \\(not an error\\)")},
	{"Ambiguous timeout", regexp.MustCompile(`(?m)^\|\s*Ambiguous timeout\s*\|.*` + "`\\*ibkr\\.Error` with `Code == \"ambiguous\"`")},
}

// checkErrorMapping verifies the three rows of the document's error-mapping table
// against the code that produces each outcome.
//
//   - Rejected by IBKR: the reply-item reader returns an *Error wrapping
//     ErrOrderRejected, carrying the gateway's code and message, on the branch
//     that reads a non-empty `error`. The identifier is compared as it is written
//     in the code; a wrapper around anything else is drift, because that is how a
//     caller tells a rejection from a transport failure.
//   - Requires confirmation: a reply item becomes a SubmitResult.Replies entry
//     and the loop continues, and the "unexpected order response" rejection is
//     reachable only when there is neither an order id nor a reply. Weakening that
//     guard is the failure mode — a reply-only response would be reported as a
//     rejection and the caller would give up on an order the gateway is still
//     asking about.
//   - Ambiguous timeout: what this document adds is which API the caller should
//     reconcile through. The error's message has to name the entry point the
//     Reconciliation section tells the caller to use, or the instruction is wrong
//     even though it is an instruction. The Code string itself is pinned by
//     06-errors-retries.md.
//
// Accepted() is checked with the second row because it is what a caller reads to
// decide whether a confirmation is outstanding, and the document's claim — a
// non-empty Replies means the order is not yet accepted — holds only if Accepted
// tests the replies.
func checkErrorMapping(repoRoot string) error {
	doc, err := parseDoc(repoRoot, ordersDoc)
	if err != nil {
		return err
	}
	trade, err := parseGo(repoRoot, tradeSrcPath)
	if err != nil {
		return err
	}
	for _, row := range errorTableRows {
		if _, ok := docSentence(row.re, doc); !ok {
			return fmt.Errorf("%s: the error-mapping table no longer has the %q row this check verifies; "+
				"the check cannot run", doc.rel, row.what)
		}
	}
	if err := checkRejectedIsOrderRejected(trade); err != nil {
		return err
	}
	if err := checkRepliesAreNotErrors(trade); err != nil {
		return err
	}
	if err := checkAcceptedTestsReplies(trade); err != nil {
		return err
	}
	return checkReconcileNamesTheDocumentedAPI(doc, trade)
}

// checkRejectedIsOrderRejected verifies the first table row.
func checkRejectedIsOrderRejected(trade *goSrc) error {
	fn := trade.method("TradeManager", "parseSubmitResult")
	if fn == nil {
		return fmt.Errorf("%s: no TradeManager.parseSubmitResult method is declared; the error-mapping check cannot run",
			tradeSrcPath)
	}
	// The branch is found by the response field it reads, not by position, so a
	// reordered parse cannot make the check read a different branch.
	branch := branchOnItemField(fn, "error")
	if branch == nil {
		return fmt.Errorf("%s: parseSubmitResult has no branch for a non-empty \"error\" field, so the document's "+
			"rejection row is not implemented; the error-mapping check cannot run", trade.at(fn))
	}
	lit := firstErrorLit(branch.Body)
	if lit == nil {
		return fmt.Errorf("%s: the rejection branch builds no *Error; %s documents a rejected order as "+
			"*ibkr.Error", trade.at(branch), ordersDoc)
	}
	fields := map[string]keyedValue{}
	for _, kv := range keyedValues(lit) {
		fields[kv.key] = kv
	}
	for _, k := range []string{"Code", "Message"} {
		if _, ok := fields[k]; !ok {
			return fmt.Errorf("%s: the rejection error sets no %s; %s documents a rejection as carrying the "+
				"gateway's code and message", trade.at(lit), k, ordersDoc)
		}
	}
	wrapped, ok := fields["Err"]
	if !ok {
		return fmt.Errorf("%s: the rejection error wraps nothing; %s documents it as wrapping ErrOrderRejected",
			trade.at(lit), ordersDoc)
	}
	if exprString(wrapped.expr) != "ErrOrderRejected" {
		return fmt.Errorf("%s: the rejection error wraps %s, not ErrOrderRejected; %s documents a rejection as "+
			"*ibkr.Error wrapping ErrOrderRejected, which is how a caller tells a rejection from a transport failure",
			trade.at(wrapped.node), exprString(wrapped.expr), ordersDoc)
	}
	return nil
}

// branchOnItemField returns the if-statement in fn that handles a response item
// carrying the named field, or nil. The field is resolved rather than matched at a
// fixed position, the whole body is searched rather than only its top-level
// statements — the branches live inside the loop over response items — and both
// spellings the parse uses are accepted:
//
//	if rawToString(item, "id") != "" { ... }
//	if id := rawToString(item, "id"); id != "" { ... }
//
// The second form is the one the code actually uses, and reading only the first
// would make the check report "not implemented" for a branch that is there.
func branchOnItemField(fn *ast.FuncDecl, field string) *ast.IfStmt {
	var found *ast.IfStmt
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		g, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		if testedField(g) == field {
			found = g
		}
		return true
	})
	return found
}

// testedField returns the response field an if-statement's condition tests
// against the empty string, or "".
func testedField(g *ast.IfStmt) string {
	b, ok := g.Cond.(*ast.BinaryExpr)
	if !ok || b.Op != token.NEQ {
		return ""
	}
	if lit, ok := b.Y.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		if s, ok := stringLit(lit); !ok || s != "" {
			return ""
		}
	} else {
		return ""
	}
	switch x := b.X.(type) {
	case *ast.CallExpr:
		return stringArg(x, 0)
	case *ast.Ident:
		// The value came from the if-statement's own init.
		if g.Init == nil {
			return ""
		}
		as, ok := g.Init.(*ast.AssignStmt)
		if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
			return ""
		}
		if id, ok := as.Lhs[0].(*ast.Ident); !ok || id.Name != x.Name {
			return ""
		}
		call, ok := as.Rhs[0].(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			return ""
		}
		return stringArg(call, 1)
	}
	return ""
}

// stringArg returns the nth argument of a call when it is a string literal.
func stringArg(call *ast.CallExpr, n int) string {
	if n >= len(call.Args) {
		return ""
	}
	lit, ok := call.Args[n].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	s, _ := stringLit(lit)
	return s
}

// firstErrorLit returns the first *Error{...} literal in a block, or nil.
func firstErrorLit(body *ast.BlockStmt) *ast.CompositeLit {
	var found *ast.CompositeLit
	ast.Inspect(body, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		if cl, ok := n.(*ast.CompositeLit); ok && structLitType(cl) == "Error" {
			found = cl
		}
		return true
	})
	return found
}

// checkRepliesAreNotErrors verifies the second table row: a reply item is
// returned in SubmitResult.Replies, never as an error.
func checkRepliesAreNotErrors(trade *goSrc) error {
	fn := trade.method("TradeManager", "parseSubmitResult")
	if fn == nil {
		return fmt.Errorf("%s: no TradeManager.parseSubmitResult method is declared; the error-mapping check cannot run",
			tradeSrcPath)
	}
	branch := branchOnItemField(fn, "id")
	if branch == nil {
		return fmt.Errorf("%s: parseSubmitResult has no branch for a reply \"id\" field, so the document's "+
			"confirmation row is not implemented; the error-mapping check cannot run", trade.at(fn))
	}
	if firstStructLit(branch.Body, "Reply") == nil {
		return fmt.Errorf("%s: the confirmation branch builds no Reply, so a required confirmation could not be "+
			"reported in SubmitResult.Replies; %s documents it as a result, not an error",
			trade.at(branch), ordersDoc)
	}
	appendOK := false
	ast.Inspect(branch.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || calleeName(call.Fun) != "append" || len(call.Args) != 2 {
			return true
		}
		if sel, ok := call.Args[0].(*ast.SelectorExpr); ok && sel.Sel.Name == "Replies" {
			appendOK = true
		}
		return true
	})
	if !appendOK {
		return fmt.Errorf("%s: the confirmation branch never appends to SubmitResult.Replies; %s documents a "+
			"required confirmation as returned in SubmitResult.Replies", trade.at(branch), ordersDoc)
	}
	if ret := errorReturnIn(branch.Body); ret != nil {
		return fmt.Errorf("%s: the confirmation branch returns an error, so a reply the gateway asked about "+
			"would be reported as a failure; %s documents it as returned in SubmitResult.Replies, not an error",
			trade.at(ret), ordersDoc)
	}
	if !endsWithContinue(branch.Body) {
		return fmt.Errorf("%s: the confirmation branch does not continue to the next item, so a reply falls "+
			"through the accepted-order handling; %s documents a reply as a pending confirmation",
			trade.at(branch), ordersDoc)
	}
	return checkEmptyResponseRejection(trade, fn)
}

// errorReturnIn returns the first statement in a block that returns an error, or
// nil.
func errorReturnIn(body *ast.BlockStmt) ast.Stmt {
	for _, s := range body.List {
		ret, ok := s.(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 2 {
			continue
		}
		if id, ok := ret.Results[0].(*ast.Ident); ok && id.Name == "nil" {
			return ret
		}
	}
	return nil
}

func endsWithContinue(body *ast.BlockStmt) bool {
	if len(body.List) == 0 {
		return false
	}
	br, ok := body.List[len(body.List)-1].(*ast.BranchStmt)
	return ok && br.Tok == token.CONTINUE
}

// checkEmptyResponseRejection verifies the guard that makes a reply-only response
// a success. The document promises a confirmation is returned in the result; if
// this guard tested only the order id, a response carrying only replies would be
// rejected and the caller would never get the chance to confirm.
//
// The result being tested is resolved from the local the function builds, because
// the guard reads a local and not the receiver: assuming it reads the receiver
// would make this check report a missing guard for one that is present.
func checkEmptyResponseRejection(trade *goSrc, fn *ast.FuncDecl) error {
	result := submitResultVar(fn)
	if result == "" {
		return fmt.Errorf("%s: parseSubmitResult builds no *SubmitResult local, so the guard that decides whether a "+
			"response is a rejection cannot be found; the error-mapping check cannot run", tradeSrcPath)
	}
	var found *ast.IfStmt
	for _, stmt := range fn.Body.List {
		g, ok := stmt.(*ast.IfStmt)
		if !ok {
			continue
		}
		if bothEmptyTests(g.Cond, result) {
			found = g
			break
		}
	}
	if found == nil {
		return fmt.Errorf("%s: parseSubmitResult has no guard rejecting a response that carries neither an order "+
			"id nor a reply; %s documents a required confirmation as a result rather than a failure",
			trade.at(fn), ordersDoc)
	}
	if firstErrorLit(found.Body) == nil {
		return fmt.Errorf("%s: the empty-response guard builds no *Error, so an unrecognised response would be "+
			"reported as a successful submission with nothing in it", trade.at(found))
	}
	return nil
}

// bothEmptyTests reports whether cond is `<result>.OrderID == "" && len(<result>.Replies) == 0`.
// Both halves are required, and the replies half is the one that keeps a
// reply-only response out of the rejection path.
func bothEmptyTests(cond ast.Expr, recv string) bool {
	and, ok := cond.(*ast.BinaryExpr)
	if !ok || and.Op != token.LAND {
		return false
	}
	return isEmptyFieldTest(and.X, recv, "OrderID") && isLenZeroTest(and.Y, recv, "Replies")
}

// submitResultVar returns the name of the local a function builds a
// *SubmitResult into, or "".
func submitResultVar(fn *ast.FuncDecl) string {
	found := ""
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found != "" {
			return false
		}
		as, ok := n.(*ast.AssignStmt)
		if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
			return true
		}
		id, ok := as.Lhs[0].(*ast.Ident)
		if !ok {
			return true
		}
		un, ok := as.Rhs[0].(*ast.UnaryExpr)
		if !ok || un.Op != token.AND {
			return true
		}
		cl, ok := un.X.(*ast.CompositeLit)
		if !ok || structLitType(cl) != "SubmitResult" {
			return true
		}
		found = id.Name
		return true
	})
	return found
}

func isEmptyFieldTest(e ast.Expr, recv, field string) bool {
	b, ok := e.(*ast.BinaryExpr)
	if !ok || b.Op != token.EQL {
		return false
	}
	return isFieldOf(b.X, recv, field) && isEmptyStringLit(b.Y)
}

func isNonEmptyFieldTest(e ast.Expr, recv, field string) bool {
	b, ok := e.(*ast.BinaryExpr)
	if !ok || b.Op != token.NEQ {
		return false
	}
	return isFieldOf(b.X, recv, field) && isEmptyStringLit(b.Y)
}

func isEmptyStringLit(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return false
	}
	s, ok := stringLit(lit)
	return ok && s == ""
}

func isFieldOf(e ast.Expr, recv, field string) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != field {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == recv
}

func isLenZeroTest(e ast.Expr, recv, field string) bool {
	b, ok := e.(*ast.BinaryExpr)
	if !ok || b.Op != token.EQL {
		return false
	}
	if v, ok := intValue(b.Y); !ok || v != 0 {
		return false
	}
	call, ok := b.X.(*ast.CallExpr)
	if !ok || calleeName(call.Fun) != "len" || len(call.Args) != 1 {
		return false
	}
	return isFieldOf(call.Args[0], recv, field)
}

// checkAcceptedTestsReplies verifies SubmitResult.Accepted, which is how a caller
// reads the document's "a non-empty Replies means the order is not yet accepted".
// Accepted has to test both halves: a result carrying an order id alongside a
// pending reply is not accepted, and a result carrying neither is not accepted.
func checkAcceptedTestsReplies(trade *goSrc) error {
	fn := trade.method("SubmitResult", "Accepted")
	if fn == nil {
		return fmt.Errorf("%s: no SubmitResult.Accepted method is declared; the error-mapping check cannot run", tradeSrcPath)
	}
	recv := receiverName(fn)
	if len(fn.Body.List) != 1 {
		return fmt.Errorf("%s: SubmitResult.Accepted is not a single return, so the documented "+
			"\"non-empty Replies means not accepted\" rule cannot be read from it", trade.at(fn))
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return fmt.Errorf("%s: SubmitResult.Accepted does not return one value; the error-mapping check cannot run",
			trade.at(fn))
	}
	if !acceptedIsRepliesEmptyAndOrderIDSet(ret.Results[0], recv) {
		return fmt.Errorf("confirmation drift: %s: Accepted is %q, but %s documents a non-empty "+
			"SubmitResult.Replies as meaning the order is not yet accepted and the caller's confirmation is required",
			trade.at(fn), exprString(ret.Results[0]), ordersDoc)
	}
	return nil
}

func acceptedIsRepliesEmptyAndOrderIDSet(e ast.Expr, recv string) bool {
	and, ok := e.(*ast.BinaryExpr)
	if !ok || and.Op != token.LAND {
		return false
	}
	return isLenZeroTest(and.X, recv, "Replies") && isNonEmptyFieldTest(and.Y, recv, "OrderID")
}

// reconcileAPIRe reads the entry point the document tells a caller to reconcile
// through.
var reconcileAPIRe = regexp.MustCompile("callers should use `(\\w+)`")

// checkReconcileNamesTheDocumentedAPI verifies the third table row's claim: the
// ambiguous-outcome error directs the caller to the API this document names.
// 06-errors-retries.md pins the error's Code and that its message tells the caller
// to reconcile at all; what it does not check is that the instruction names an
// entry point that exists, and that the two documents agree on which one.
func checkReconcileNamesTheDocumentedAPI(doc *docFile, trade *goSrc) error {
	idx := -1
	for i, l := range doc.lines {
		if reconcileAPIRe.MatchString(l) {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("%s: the Reconciliation section does not name the API to reconcile through; "+
			"the error-mapping check cannot run", doc.rel)
	}
	name := reconcileAPIRe.FindStringSubmatch(doc.lines[idx])[1]
	if trade.method("TradeManager", name) == nil {
		return fmt.Errorf("%s: the document tells a caller to reconcile through %s, which no TradeManager method "+
			"implements; the ambiguous-outcome error cannot point anyone at it", doc.at(idx+1), name)
	}
	mutate := trade.method("TradeManager", "mutate")
	if mutate == nil {
		return fmt.Errorf("%s: no TradeManager.mutate method is declared; the error-mapping check cannot run", tradeSrcPath)
	}
	lit := firstErrorLit(mutate.Body)
	if lit == nil {
		return fmt.Errorf("%s: TradeManager.mutate builds no *Error, so the ambiguous-outcome row of the error "+
			"mapping table in %s has nothing to check", trade.at(mutate), ordersDoc)
	}
	msg := ""
	for _, kv := range keyedValues(lit) {
		if kv.key != "Message" {
			continue
		}
		s, ok := stringLit(kv.expr)
		if !ok {
			return fmt.Errorf("%s: the ambiguous-outcome error's Message is not a string literal, so the "+
				"reconcile instruction it carries cannot be read", trade.at(kv.node))
		}
		msg = s
	}
	if msg == "" {
		return fmt.Errorf("%s: the ambiguous-outcome error sets no Message, so it directs the caller nowhere; "+
			"%s documents an ambiguous outcome as an error the caller reconciles from", trade.at(lit), ordersDoc)
	}
	if !strings.Contains(msg, name) {
		return fmt.Errorf("%s: the ambiguous-outcome error's Message (%s) does not name %s, the API %s tells a "+
			"caller to reconcile through", trade.at(lit), msg, name, doc.rel)
	}
	return nil
}
