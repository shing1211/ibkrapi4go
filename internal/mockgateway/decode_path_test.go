// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package mockgateway

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/shing1211/ibkrapi4go/client"
)

// pkgIBKRSrcDir is pkg/ibkr relative to this package directory. It ships with
// the repository, so the scan never touches the network. The tree is only ever
// read.
const pkgIBKRSrcDir = "../../pkg/ibkr"

// productionPath is how pkg/ibkr reads the response body of one operation. The
// shape check can only assert a fixture's keys against the generated type when
// the generated type is what production actually decodes, so this is what
// decides whether the key check is enforced for an op.
//
// It is established from call sites, never from the operation's name: the scan
// reads the generated client method each pkg/ibkr method actually calls, and
// what that method then does with the result.
type productionPath int

const (
	// pathUnknown is the zero value: no pkg/ibkr call site resolved to the op.
	// It is not an exemption — an op in this class is still key-checked, because
	// nothing has established that the generated type is irrelevant to it.
	pathUnknown productionPath = iota
	// pathGenerated: a pkg/ibkr method reads the generated JSON2xx field
	// (resp.JSON200, resp.GetJSON200(), resp.JSON202, …).
	pathGenerated
	// pathBypassed: a pkg/ibkr method unmarshals the raw HTTP body itself into a
	// hand-written struct, so the generated type is irrelevant to it and
	// comparing the fixture against the generated type is invalid.
	pathBypassed
	// pathBodyUnused: a pkg/ibkr method calls the operation but reads no body at
	// all. The key check still runs — production ignoring a body is not evidence
	// that the fixture is right — but the class is reported.
	pathBodyUnused
)

func (p productionPath) String() string {
	switch p {
	case pathGenerated:
		return "generated"
	case pathBypassed:
		return "bypassed"
	case pathBodyUnused:
		return "bodyUnused"
	case pathUnknown:
		return "unknown"
	}
	return "productionPath(?)"
}

// pathEvidence records the production call site that classified an op, so every
// exemption names the decode that justifies it.
type pathEvidence struct {
	path productionPath
	// at is the file:line of the raw-body decode (for pathBypassed), the
	// generated-field read (pathGenerated), or the op call (pathBodyUnused).
	at string
	// via names the pkg/ibkr method and how its operation was resolved.
	via string
}

// decodeScan is one walk of pkg/ibkr.
type decodeScan struct {
	// ops is the per-operation production decode path.
	ops map[string]pathEvidence
	// mixed lists operations with both a generated-field reader and a raw-body
	// decoder. The key check is enforced for them (a wrong key would still reach
	// the generated reader), so the class is only reported.
	mixed []string
	// unattributed lists raw-body decode sites whose enclosing method reaches no
	// generated operation: shared decoders and pagers, which are classified at
	// their call sites instead.
	unattributed []string
	// ambiguous lists raw-body decode sites whose enclosing method reaches more
	// than one generated operation. Such a site does not identify which
	// operation it belongs to, so it can never justify an exemption.
	ambiguous []string
	// multiOp lists methods that reach several operations without reading a
	// body. They cannot hide a bypass, because they decode nothing.
	multiOp []string
	// inherited counts bypass classifications justified by a decoder in a callee
	// rather than in the op's own method (parseSubmitResult and friends).
	inherited int
	// err is set when pkg/ibkr could not be read at all. It is reported rather
	// than swallowed, because a failed scan must not silently disable the check.
	err error
}

// productionDecodePaths walks pkg/ibkr once and memoises the result.
var productionDecodePaths = sync.OnceValue(scanProductionDecodePaths)

// pkgFunc is one pkg/ibkr function or method, reduced to what the decode-path
// question needs.
type pkgFunc struct {
	file string
	line int
	recv string
	name string

	// bodyDecodeAt holds the line of every raw-body decode in the body:
	// json.Unmarshal or decodeJSONBytes of `x.Body`, or decodeJSON of the
	// *http.Response. A json.Unmarshal of a json.RawMessage is not one: it reads
	// a fragment production already holds, not the response.
	bodyDecodeAt []int
	// ops is every generated operation the body reaches directly.
	ops map[string]bool
	// dispatchOps are the generated operations reached only from inside a case
	// body of this function's own dispatch table. A function that is itself a
	// dispatcher reaches several operations and belongs to none of them; each
	// caller is resolved through the literal it passes.
	dispatchOps map[string]bool
	// readsGeneratedBody records a read of the generated JSON2xx field.
	readsGeneratedBody bool
	// calls names the pkg/ibkr functions and methods this body calls.
	calls map[string]bool
	// litArgs[c] is the set of string literals passed to the pkg/ibkr callee c.
	litArgs map[string]map[string]bool
	// disp maps a string case key to the single operation its case body reaches.
	// It resolves the wrappers that reach their operation through a string-keyed
	// dispatch table rather than a direct generated call.
	disp map[string]string
}

// generatedBodyRe matches the generated response-wrapper accessors oapi-codegen
// emits. Both the exported field (resp.JSON200) and the accessor oapi-codegen
// adds for unions (resp.GetJSON200()) are selectors, not necessarily calls, so
// they are matched as selectors.
var generatedBodyRe = regexp.MustCompile(`^(?:JSON\d{3}|GetJSON\d{3})$`)

// stringLitRe unwraps a Go string literal's contents.
var stringLitRe = regexp.MustCompile(`^"([^"\\]*)"$`)

// scanProductionDecodePaths derives, per operation, whether pkg/ibkr reads the
// generated response type or the raw HTTP body. See productionPath for the
// classes; the derivation is:
//
//  1. Every pkg/ibkr function is reduced to: the raw-body decodes it performs,
//     the generated operations it calls, whether it reads a generated JSON2xx
//     field, and which pkg/ibkr functions it calls.
//  2. A function "decodes the body" if it does so directly or if it calls a
//     function that does, to a fixpoint. This catches the wrappers that hand
//     resp to a shared decoder (parseSubmitResult, decodeRebalanceSubscription).
//  3. A function's operation is the single generated operation it calls. When it
//     calls none, it is resolved through a string-keyed dispatch table it
//     invokes with a literal (ModelManager.postModelJSON).
//  4. An operation is bypassed only when every pkg/ibkr method that reaches it
//     decodes the raw body and none reads the generated type.
func scanProductionDecodePaths() decodeScan {
	scan := decodeScan{ops: map[string]pathEvidence{}}

	entries, err := os.ReadDir(pkgIBKRSrcDir)
	if err != nil {
		scan.err = fmt.Errorf("read %s: %w", pkgIBKRSrcDir, err)
		return scan
	}
	fset := token.NewFileSet()
	var fns []*pkgFunc
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(pkgIBKRSrcDir, name), nil, parser.SkipObjectResolution)
		if err != nil {
			scan.err = fmt.Errorf("parse %s: %w", name, err)
			return scan
		}
		imports := importNames(file)
		for _, d := range file.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			fns = append(fns, reduceFunc(fset, name, imports, fd))
		}
	}
	sort.Slice(fns, func(i, j int) bool {
		if fns[i].file != fns[j].file {
			return fns[i].file < fns[j].file
		}
		return fns[i].line < fns[j].line
	})

	// Step 2: a function decodes the body directly or through a callee.
	byName := map[string][]int{}
	for i, f := range fns {
		byName[f.name] = append(byName[f.name], i)
	}
	isDecoder := make([]bool, len(fns))
	for changed := true; changed; {
		changed = false
		for i, f := range fns {
			if isDecoder[i] {
				continue
			}
			if len(f.bodyDecodeAt) > 0 {
				isDecoder[i], changed = true, true
				continue
			}
			for callee := range f.calls {
				for _, j := range byName[callee] {
					if j != i && isDecoder[j] {
						isDecoder[i], changed = true, true
						break
					}
				}
				if isDecoder[i] {
					break
				}
			}
		}
	}
	// The decode site that justifies a classification. A function that decodes
	// the body itself names its own json.Unmarshal/decodeJSON line; one that
	// hands resp to a single shared decoder names the line inside that decoder,
	// so every bypass justification is a real decode. A caller whose decoder
	// cannot be pinned to one callee — pkg/ibkr has several methods of the same
	// name, and only the name is syntactically visible — reports no site rather
	// than a line that is not its own.
	decoderSite := func(i int) string {
		if len(fns[i].bodyDecodeAt) > 0 {
			return fmt.Sprintf("%s:%d", fns[i].file, fns[i].bodyDecodeAt[0])
		}
		for _, callee := range sortedBoolKeys(fns[i].calls) {
			if len(byName[callee]) != 1 {
				continue
			}
			j := byName[callee][0]
			if len(fns[j].bodyDecodeAt) > 0 {
				return fmt.Sprintf("%s:%d", fns[j].file, fns[j].bodyDecodeAt[0])
			}
		}
		return ""
	}

	// Steps 3 and 4.
	type sites struct {
		generated []string
		bypassed  []string
		unused    []string
	}
	perOp := map[string]*sites{}
	via := map[string]string{}
	for i, f := range fns {
		ops, how := f.ops, "direct generated call"
		if len(ops) == 0 {
			ops, how = dispatchOps(f, byName, fns), "string dispatch"
		}
		if len(ops) > 1 {
			// A decode that reaches several operations does not identify which
			// one it belongs to, so it may never justify an exemption. One that
			// reaches none (a shared decoder) or several without decoding is
			// harmless and only reported.
			switch {
			case isDecoder[i]:
				scan.ambiguous = append(scan.ambiguous, fmt.Sprintf("%s reaches %d operations %s, so the decode%s cannot justify an exemption for any of them",
					f.qualifier(), len(ops), ops2str(ops), atOrAt(decoderSite(i))))
			default:
				scan.multiOp = append(scan.multiOp, fmt.Sprintf("%s reaches %d operations %s without reading a body",
					f.qualifier(), len(ops), ops2str(ops)))
			}
			continue
		}
		if len(ops) == 0 {
			if isDecoder[i] {
				scan.unattributed = append(scan.unattributed, fmt.Sprintf("%s reaches no generated operation, so it is classified at its call sites instead (decodes the body%s)",
					f.qualifier(), atOrAt(decoderSite(i))))
			}
			continue
		}
		op := onlyKey(ops)
		if _, ok := perOp[op]; !ok {
			perOp[op] = &sites{}
		}
		v := via[op]
		if v == "" || f.qualifier() < v {
			via[op] = f.qualifier() + " (" + how + ")"
		}
		switch {
		case isDecoder[i] && f.readsGeneratedBody:
			// Both in one method: the generated read wins so the check is
			// enforced, and the conflict is reported.
			perOp[op].generated = append(perOp[op].generated, fmt.Sprintf("%s:%d", f.file, f.line))
		case isDecoder[i]:
			if len(f.bodyDecodeAt) == 0 {
				scan.inherited++
			}
			perOp[op].bypassed = append(perOp[op].bypassed, decoderSite(i))
		case f.readsGeneratedBody:
			perOp[op].generated = append(perOp[op].generated, fmt.Sprintf("%s:%d", f.file, f.line))
		default:
			perOp[op].unused = append(perOp[op].unused, fmt.Sprintf("%s:%d", f.file, f.line))
		}
	}

	for op, s := range perOp {
		switch {
		case len(s.generated) > 0:
			scan.ops[op] = pathEvidence{pathGenerated, minSite(s.generated), via[op]}
			if len(s.bypassed) > 0 {
				scan.mixed = append(scan.mixed, fmt.Sprintf("%s is read through the generated type at %s and decoded from the raw body at %s; the generated read is what the key check is enforced against",
					op, minSite(s.generated), minSite(s.bypassed)))
			}
		case len(s.bypassed) > 0:
			scan.ops[op] = pathEvidence{pathBypassed, minSite(s.bypassed), via[op]}
		case len(s.unused) > 0:
			scan.ops[op] = pathEvidence{pathBodyUnused, minSite(s.unused), via[op]}
		}
	}
	sort.Strings(scan.mixed)
	sort.Strings(scan.unattributed)
	sort.Strings(scan.ambiguous)
	sort.Strings(scan.multiOp)
	return scan
}

// reduceFunc extracts the decode-path facts from one pkg/ibkr function body.
func reduceFunc(fset *token.FileSet, file string, imports map[string]bool, fd *ast.FuncDecl) *pkgFunc {
	f := &pkgFunc{
		file:        file,
		line:        fset.Position(fd.Pos()).Line,
		name:        fd.Name.Name,
		ops:         map[string]bool{},
		dispatchOps: map[string]bool{},
		calls:       map[string]bool{},
		litArgs:     map[string]map[string]bool{},
		disp:        map[string]string{},
	}
	if fd.Recv != nil && len(fd.Recv.List) > 0 {
		f.recv = recvTypeName(fd.Recv.List[0].Type)
	}

	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.SelectorExpr:
			// resp.JSON200 / resp.GetJSON200(): a selector, not necessarily a call.
			if generatedBodyRe.MatchString(node.Sel.Name) {
				f.readsGeneratedBody = true
			}
		case *ast.CallExpr:
			switch calleeRef(node.Fun) {
			case "json.Unmarshal", "json.NewDecoder", "decodeJSONBytes":
				// Only `x.Body` counts as a response body.
				if len(node.Args) > 0 && isBodyField(node.Args[0], imports) {
					f.bodyDecodeAt = append(f.bodyDecodeAt, fset.Position(node.Pos()).Line)
				}
			case "decodeJSON":
				// Takes the *http.Response itself.
				if len(node.Args) > 0 && isLocalIdent(node.Args[0], imports) {
					f.bodyDecodeAt = append(f.bodyDecodeAt, fset.Position(node.Pos()).Line)
				}
			}
			switch fun := node.Fun.(type) {
			case *ast.SelectorExpr:
				if op, ok := generatedMethodOps()[fun.Sel.Name]; ok {
					f.ops[op] = true
				}
			case *ast.Ident:
				if op, ok := generatedMethodOps()[fun.Name]; ok {
					f.ops[op] = true
				}
			}
			recordLocalCall(f, node, imports)
		}
		return true
	})

	// A string-keyed dispatch table: switch on a parameter, one case per
	// operation, each case body calling exactly one generated method.
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		sw, ok := n.(*ast.SwitchStmt)
		if !ok || sw.Tag == nil {
			return true
		}
		for _, stmt := range sw.Body.List {
			cc, ok := stmt.(*ast.CaseClause)
			if !ok || len(cc.List) != 1 {
				continue
			}
			lit, ok := cc.List[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			m := stringLitRe.FindStringSubmatch(lit.Value)
			if m == nil {
				continue
			}
			reached := map[string]bool{}
			ast.Inspect(cc, func(x ast.Node) bool {
				call, ok := x.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if op, ok := generatedMethodOps()[sel.Sel.Name]; ok {
					reached[op] = true
					f.dispatchOps[op] = true
				}
				return true
			})
			if len(reached) == 1 {
				f.disp[m[1]] = onlyKey(reached)
			}
		}
		return true
	})

	// A dispatcher belongs to no operation of its own: the generated calls it
	// makes are the ones its callers select between.
	if len(f.disp) > 0 {
		for op := range f.dispatchOps {
			delete(f.ops, op)
		}
	}
	return f
}

// dispatchOps resolves the operation a function reaches through a string-keyed
// dispatch table it calls with a literal.
func dispatchOps(f *pkgFunc, byName map[string][]int, fns []*pkgFunc) map[string]bool {
	out := map[string]bool{}
	for callee := range f.calls {
		for _, j := range byName[callee] {
			for key, op := range fns[j].disp {
				if f.litArgs[callee][key] {
					out[op] = true
				}
			}
		}
	}
	return out
}

// recordLocalCall notes a call to a pkg/ibkr function or method, so the bodyReader
// fixpoint can follow one hop, and remembers any string literal arguments.
func recordLocalCall(f *pkgFunc, call *ast.CallExpr, imports map[string]bool) {
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		// A method on a value the package itself declares, or on an embedded
		// pkg/ibkr manager reached through a field chain.
		if f.knows(fun.Sel.Name) {
			f.calls[fun.Sel.Name] = true
			recordLiterals(f, fun.Sel.Name, call.Args)
		}
	case *ast.Ident:
		if f.knows(fun.Name) {
			f.calls[fun.Name] = true
			recordLiterals(f, fun.Name, call.Args)
		}
	}
}

func recordLiterals(f *pkgFunc, callee string, args []ast.Expr) {
	for _, a := range args {
		lit, ok := a.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}
		m := stringLitRe.FindStringSubmatch(lit.Value)
		if m == nil {
			continue
		}
		if f.litArgs[callee] == nil {
			f.litArgs[callee] = map[string]bool{}
		}
		f.litArgs[callee][m[1]] = true
	}
}

// pkgDeclared is filled once with every function and method name pkg/ibkr
// declares, so a call can be told apart from a call into client or net/http.
var pkgDeclared = sync.OnceValue(func() map[string]bool {
	out := map[string]bool{}
	entries, err := os.ReadDir(pkgIBKRSrcDir)
	if err != nil {
		return out
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(pkgIBKRSrcDir, e.Name()), nil, parser.SkipObjectResolution)
		if err != nil {
			continue
		}
		for _, d := range file.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok {
				out[fd.Name.Name] = true
			}
		}
	}
	return out
})

func (f *pkgFunc) knows(name string) bool { return pkgDeclared()[name] }

// importNames collects the package identifiers a file imports, so `json.Body` or
// `http.Response.Body` is not mistaken for a local response variable.
func importNames(file *ast.File) map[string]bool {
	out := map[string]bool{}
	for _, imp := range file.Imports {
		p := strings.Trim(imp.Path.Value, `"`)
		name := p[strings.LastIndex(p, "/")+1:]
		if imp.Name != nil {
			name = imp.Name.Name
		}
		out[name] = true
	}
	return out
}

func isBodyField(arg ast.Expr, imports map[string]bool) bool {
	sel, ok := arg.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Body" {
		return false
	}
	return isLocalIdent(sel.X, imports)
}

func isLocalIdent(arg ast.Expr, imports map[string]bool) bool {
	id, ok := arg.(*ast.Ident)
	return ok && id.Name != "nil" && !imports[id.Name]
}

// calleeRef renders a call target as "pkg.Fn" for a selector call or "Fn".
func calleeRef(fun ast.Expr) string {
	switch e := fun.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.IndexExpr:
		return calleeRef(e.X)
	case *ast.IndexListExpr:
		return calleeRef(e.X)
	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok {
			return x.Name + "." + e.Sel.Name
		}
		return e.Sel.Name
	}
	return ""
}

func recvTypeName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + recvTypeName(e.X)
	}
	return "?"
}

func (f *pkgFunc) qualifier() string {
	if f.recv == "" {
		return f.file + ":" + strconv.Itoa(f.line) + " " + f.name
	}
	return f.file + ":" + strconv.Itoa(f.line) + " (" + f.recv + ")." + f.name
}

func onlyKey(m map[string]bool) string {
	for k := range m {
		return k
	}
	return ""
}

func sortedSet(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedBoolKeys(m map[string]bool) []string { return sortedSet(m) }

func ops2str(m map[string]bool) string { return "[" + strings.Join(sortedSet(m), " ") + "]" }

// atOrAt renders a decode site as a trailing clause, or says plainly that the
// decode was reached through a callee pkg/ibkr's name-only resolution cannot pin.
func atOrAt(site string) string {
	if site == "" {
		return " through a callee whose decode site the name-only call graph cannot pin"
	}
	return " at " + site
}

func minSite(sites []string) string {
	out := sites[0]
	for _, s := range sites[1:] {
		if s < out {
			out = s
		}
	}
	return out
}

// generatedMethodOps maps every generated client method — both the bare
// *WithResponse-less form and the response-wrapper form — to the opId the
// fixture layer files it under. Both go through opIDFromMethod, the same
// derivation opResponseTypes uses, so a call site can never be matched to the
// wrong operation.
var generatedMethodOps = sync.OnceValue(func() map[string]string {
	out := map[string]string{}
	for _, iface := range []reflect.Type{
		reflect.TypeOf((*client.ClientWithResponsesInterface)(nil)).Elem(),
		reflect.TypeOf((*client.ClientInterface)(nil)).Elem(),
	} {
		for i := range iface.NumMethod() {
			m := iface.Method(i)
			if op := opIDFromMethod(m.Name); op != "" {
				out[m.Name] = op
			}
		}
	}
	return out
})

// opIDFromMethod derives the opId a generated client method belongs to. Both the
// bare form (GetAccounts) and the response-wrapper form
// (GetAccountsWithResponse, CreateEchoSignedJwtWithBodyWithResponse) name the
// same operation, so the WithResponse and request-body suffixes are stripped
// before the first rune is lowered.
func opIDFromMethod(method string) string {
	base := strings.TrimSuffix(method, "WithResponse")
	for _, s := range bodySuffixes {
		base = strings.TrimSuffix(base, s)
	}
	if base == "" {
		return ""
	}
	r := []rune(base)
	r[0] = []rune(strings.ToLower(string(r[0])))[0]
	return string(r)
}

// TestProductionDecodePathsResolve is the guard on the exemption. It fails when
// pkg/ibkr cannot be read, or when a raw-body decode reaches more than one
// operation — such a site does not identify which operation it belongs to, so it
// must never be allowed to exempt one. Everything else is reported, so the
// classification is read rather than assumed.
func TestProductionDecodePathsResolve(t *testing.T) {
	scan := productionDecodePaths()
	if scan.err != nil {
		t.Fatalf("cannot derive the production decode path: %v", scan.err)
	}
	for _, s := range scan.ambiguous {
		t.Errorf("ambiguous body decode: %s", s)
	}

	// Every exemption must name the decode that justifies it, so the exemption
	// is auditable from the test log alone.
	var exemptions int
	for op, ev := range scan.ops {
		if ev.path != pathBypassed {
			continue
		}
		exemptions++
		if ev.at == "" || ev.via == "" {
			t.Errorf("op %q is exempt because production decodes the raw body, but the exemption names no decode site (at=%q via=%q)", op, ev.at, ev.via)
		}
	}
	t.Logf("%d operations are exempt because production decodes the raw body, each naming the decode that justifies it", exemptions)

	// Every operation a fixture serves must be classifiable, or explicitly
	// reported as unknown. Nothing is inferred from the operation's name.
	fixtures := DefaultFixtures().All()
	var unknown []string
	for op := range fixtures {
		if scan.ops[op].path == pathUnknown {
			unknown = append(unknown, op)
		}
	}
	sort.Strings(unknown)
	if len(unknown) > 0 {
		t.Logf("%d fixture ops have no pkg/ibkr call site and stay key-checked against the generated type:", len(unknown))
		for _, op := range unknown {
			t.Logf("      %-40s no method in pkg/ibkr calls the generated operation, so nothing establishes that the generated type is irrelevant to it", op)
		}
	}

	counts := map[productionPath]int{}
	for _, ev := range scan.ops {
		counts[ev.path]++
	}
	t.Logf("production decode path for %d operations reached by pkg/ibkr: %d read the generated type, %d decode the raw body, %d read no body",
		len(scan.ops), counts[pathGenerated], counts[pathBypassed], counts[pathBodyUnused])
	for _, s := range scan.mixed {
		t.Logf("  BOTH PATHS: %s", s)
	}
	for _, s := range scan.unattributed {
		t.Logf("  shared decoder, classified at its call sites: %s", s)
	}
	for _, s := range scan.multiOp {
		t.Logf("  multi-op caller that reads no body: %s", s)
	}
	if scan.inherited > 0 {
		t.Logf("  %d bypass classifications are justified by a decoder in a callee rather than in the op's own method", scan.inherited)
	}
}
