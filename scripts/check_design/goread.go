// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// goread.go holds the readers the design checks share: a parsed Go file that can
// name the file:line of any node in it, a markdown file split into numbered
// lines, and canonicalizers that turn a documented default and a coded default
// into the same string so they can be compared.
//
// Everything resolves a declaration before using it. Matching on a bare name
// is how an earlier check in this repo blessed a real bug: a name is only
// meaningful once you know which declaration it resolves to.

// goSrc is a parsed Go file.
type goSrc struct {
	rel  string
	path string
	fset *token.FileSet
	file *ast.File
}

func parseGo(repoRoot, rel string) (*goSrc, error) {
	path := filepath.Join(repoRoot, filepath.FromSlash(rel))
	//nolint:gosec // rel is a repo-relative path fixed in this program, never caller input
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("%s: parse error: %w", rel, err)
	}
	return &goSrc{rel: rel, path: path, fset: fset, file: file}, nil
}

// at formats a node's position as file:line.
func (g *goSrc) at(n ast.Node) string {
	if n == nil {
		return g.rel
	}
	return fmt.Sprintf("%s:%d", g.rel, g.fset.Position(n.Pos()).Line)
}

func (g *goSrc) line(n ast.Node) int {
	if n == nil {
		return 0
	}
	return g.fset.Position(n.Pos()).Line
}

// funcDecl returns the package-level func named name, or nil.
func (g *goSrc) funcDecl(name string) *ast.FuncDecl {
	for _, d := range g.file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if ok && fn.Recv == nil && fn.Name.Name == name {
			return fn
		}
	}
	return nil
}

// method returns the method named name whose receiver base type is recv.
func (g *goSrc) method(recv, name string) *ast.FuncDecl {
	for _, d := range g.file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name.Name != name {
			continue
		}
		if recvBaseType(fn.Recv.List[0].Type) == recv {
			return fn
		}
	}
	return nil
}

func recvBaseType(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return recvBaseType(t.X)
	case *ast.IndexExpr: // generic receiver instantiation
		return recvBaseType(t.X)
	case *ast.IndexListExpr:
		return recvBaseType(t.X)
	case *ast.Ident:
		return t.Name
	}
	return ""
}

// codeField is one field of a declared struct.
type codeField struct {
	name string
	typ  string
	node ast.Node
}

func (c codeField) describe() string {
	if c.name == "" {
		return "embedded " + c.typ
	}
	return c.name + " " + c.typ
}

// structFields returns the fields of the struct type named name, in
// declaration order. Embedded fields get an empty name.
func (g *goSrc) structFields(name string) ([]codeField, error) {
	for _, d := range g.file.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, s := range gd.Specs {
			ts, ok := s.(*ast.TypeSpec)
			if !ok || ts.Name.Name != name {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return nil, fmt.Errorf("%s: type %s is declared but is not a struct", g.at(ts), name)
			}
			var out []codeField
			for _, f := range st.Fields.List {
				typ := exprString(f.Type)
				if len(f.Names) == 0 {
					out = append(out, codeField{typ: typ, node: f})
					continue
				}
				for _, n := range f.Names {
					out = append(out, codeField{name: n.Name, typ: typ, node: f})
				}
			}
			return out, nil
		}
	}
	return nil, fmt.Errorf("%s: no struct type named %s is declared here", g.rel, name)
}

// taggedField is one field of a declared struct together with the json tag it
// carries. tag is the tag's name, the first comma-separated component; tagFull is
// the whole tag value, so an option like omitempty stays readable.
type taggedField struct {
	name    string
	tag     string
	tagFull string
	typ     string
	node    ast.Node
}

// taggedFields returns the fields of the struct type named name, keeping each
// field's `json:"..."` tag. A field with no json tag gets an empty tag.
//
// This is deliberately a second reader rather than an extension of
// structFields. structFields is what the 06-errors-retries.md field-set checks
// read, and their reader must not change underneath them; the json tag is what
// makes a wire contract checkable, and structFields drops it.
func (g *goSrc) taggedFields(name string) ([]taggedField, error) {
	for _, d := range g.file.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, s := range gd.Specs {
			ts, ok := s.(*ast.TypeSpec)
			if !ok || ts.Name.Name != name {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return nil, fmt.Errorf("%s: type %s is declared but is not a struct", g.at(ts), name)
			}
			out := make([]taggedField, 0, len(st.Fields.List))
			for _, f := range st.Fields.List {
				typ := exprString(f.Type)
				tag, full := jsonTag(f)
				if len(f.Names) == 0 {
					out = append(out, taggedField{typ: typ, tag: tag, tagFull: full, node: f})
					continue
				}
				for _, n := range f.Names {
					out = append(out, taggedField{name: n.Name, tag: tag, tagFull: full, typ: typ, node: f})
				}
			}
			return out, nil
		}
	}
	return nil, fmt.Errorf("%s: no struct type named %s is declared here", g.rel, name)
}

// jsonTag reads a field's `json:"name,..."` tag, returning the tag's name and the
// whole tag value. A field with no tag, or a tag this cannot read, yields two
// empty strings — which is itself a finding for any check that requires a tagged
// field, so a missing tag is never a silent pass.
func jsonTag(f *ast.Field) (name, full string) {
	if f.Tag == nil {
		return "", ""
	}
	raw, ok := stringLit(f.Tag)
	if !ok {
		return "", ""
	}
	v, ok := reflect.StructTag(raw).Lookup("json")
	if !ok {
		return "", ""
	}
	name, _, _ = strings.Cut(v, ",")
	return name, v
}

// typeSpec returns the type declaration named name, or nil.
func (g *goSrc) typeSpec(name string) *ast.TypeSpec {
	for _, d := range g.file.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, s := range gd.Specs {
			ts, ok := s.(*ast.TypeSpec)
			if ok && ts.Name.Name == name {
				return ts
			}
		}
	}
	return nil
}

// ifaceMethod returns the signature of the method named name declared by the
// interface type named iface, or nil. A generated operation is declared twice —
// once on the concrete *Client and once on the ClientInterface callers hold —
// and only the interface declaration carries the typed request body, so a
// contract check has to read the interface.
func (g *goSrc) ifaceMethod(iface, name string) *ast.FuncType {
	ts := g.typeSpec(iface)
	if ts == nil {
		return nil
	}
	it, ok := ts.Type.(*ast.InterfaceType)
	if !ok {
		return nil
	}
	for _, f := range it.Methods.List {
		ft, ok := f.Type.(*ast.FuncType)
		if !ok || len(f.Names) != 1 || f.Names[0].Name != name {
			continue
		}
		return ft
	}
	return nil
}

// pkgGoFiles returns the non-test .go files directly inside the package
// directory rel, sorted. _test.go files are excluded: a check that scanned them
// would report test scaffolding as if it were the contract.
func pkgGoFiles(repoRoot, rel string) ([]string, error) {
	dir := filepath.Join(repoRoot, filepath.FromSlash(rel))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	var out []string
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		out = append(out, rel+"/"+n)
	}
	sort.Strings(out)
	return out, nil
}

// keyedValue is one `Key: value` entry of a composite literal.
type keyedValue struct {
	key  string
	expr ast.Expr
	node ast.Node
}

// keyedValues returns the keyed entries of a struct composite literal. The
// caller is responsible for confirming the literal's type first: a slice
// literal with elided element types (`[]Point{{X: 1}}`) also keys elements by
// a bare Ident, so the keys only mean fields once the type is resolved.
func keyedValues(lit *ast.CompositeLit) []keyedValue {
	var out []keyedValue
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		out = append(out, keyedValue{key: exprString(kv.Key), expr: kv.Value, node: kv})
	}
	return out
}

// structLitType returns the struct type name a composite literal constructs, so
// `&Error{...}` and `Error{...}` both resolve to "Error". A literal over an
// unnamed type (`struct{...}{...}`) returns "".
func structLitType(lit *ast.CompositeLit) string {
	e := lit.Type
	if u, ok := e.(*ast.UnaryExpr); ok && u.Op == token.AND {
		e = u.X
	}
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// paramOfType returns the name of the first parameter of fn whose rendered type
// is typ. Resolving the parameter rather than assuming a name is deliberate: a
// parameter called something else is still the same declaration, and a
// different parameter with a guessed name is not.
func paramOfType(ft *ast.FuncType, typ string) string {
	if ft == nil || ft.Params == nil {
		return ""
	}
	for _, p := range ft.Params.List {
		if exprString(p.Type) != typ {
			continue
		}
		if len(p.Names) > 0 {
			return p.Names[0].Name
		}
	}
	return ""
}

// exprString renders an expression as the single-space-joined source form.
func exprString(n ast.Expr) string {
	var b bytes.Buffer
	if err := printer.Fprint(&b, token.NewFileSet(), n); err != nil {
		return ""
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// intValue reads an integer literal.
func intValue(n ast.Expr) (int, bool) {
	bl, ok := n.(*ast.BasicLit)
	if !ok || bl.Kind != token.INT {
		return 0, false
	}
	v, err := strconv.Atoi(strings.ReplaceAll(bl.Value, "_", ""))
	if err != nil {
		return 0, false
	}
	return v, true
}

// boolValue reads a true/false identifier.
func boolValue(n ast.Expr) (string, bool) {
	id, ok := n.(*ast.Ident)
	if !ok || (id.Name != "true" && id.Name != "false") {
		return "", false
	}
	return id.Name, true
}

// timeUnit resolves a `time.<Unit>` selector to its duration.
func timeUnit(n ast.Expr) (time.Duration, bool) {
	sel, ok := n.(*ast.SelectorExpr)
	if !ok {
		return 0, false
	}
	if id, ok := sel.X.(*ast.Ident); !ok || id.Name != "time" {
		return 0, false
	}
	switch sel.Sel.Name {
	case "Nanosecond":
		return time.Nanosecond, true
	case "Microsecond":
		return time.Microsecond, true
	case "Millisecond":
		return time.Millisecond, true
	case "Second":
		return time.Second, true
	case "Minute":
		return time.Minute, true
	case "Hour":
		return time.Hour, true
	}
	return 0, false
}

// durationValue reads an `<int> * time.<Unit>` expression, which is how Go
// spells a Duration literal.
func durationValue(n ast.Expr) (time.Duration, bool) {
	if p, ok := n.(*ast.ParenExpr); ok {
		return durationValue(p.X)
	}
	b, ok := n.(*ast.BinaryExpr)
	if !ok || b.Op != token.MUL {
		return 0, false
	}
	if v, ok1 := intValue(b.X); ok1 {
		if u, ok2 := timeUnit(b.Y); ok2 {
			return time.Duration(v) * u, true
		}
	}
	if v, ok1 := intValue(b.Y); ok1 {
		if u, ok2 := timeUnit(b.X); ok2 {
			return time.Duration(v) * u, true
		}
	}
	return 0, false
}

// intSliceValue reads a `[]int{...}` composite literal.
func intSliceValue(n ast.Expr) ([]int, bool) {
	cl, ok := n.(*ast.CompositeLit)
	if !ok || exprString(cl.Type) != "[]int" {
		return nil, false
	}
	out := make([]int, 0, len(cl.Elts))
	for _, e := range cl.Elts {
		v, ok := intValue(e)
		if !ok {
			return nil, false
		}
		out = append(out, v)
	}
	return out, true
}

// stringLit reads a string literal, unquoting it.
func stringLit(n ast.Expr) (string, bool) {
	bl, ok := n.(*ast.BasicLit)
	if !ok || bl.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(bl.Value)
	if err != nil {
		return "", false
	}
	return s, true
}

// --- markdown ---------------------------------------------------------------

// docFile is a markdown file split into 1-based numbered lines.
type docFile struct {
	rel   string
	lines []string
}

func parseDoc(repoRoot, rel string) (*docFile, error) {
	//nolint:gosec // rel is a repo-relative path fixed in this program, never caller input
	b, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(rel)))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	return &docFile{
		rel:   rel,
		lines: strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n"),
	}, nil
}

func (d *docFile) at(no int) string { return fmt.Sprintf("%s:%d", d.rel, no) }

// goBlock is an inclusive 1-based line range of a fenced ```go block.
type goBlock struct{ first, last int }

// content returns the block's content lines as an inclusive 1-based range.
// goBlocks records first as the first content line and last as the last one, so
// the range is those two values; a reader that loops with `<` would silently drop
// the block's final line, which is where a signature often sits. The 06 checks
// slice d.lines[b.first-1:b.last] and loop with `<`, and their reader must not
// change underneath them, so the correction lives here.
func (b goBlock) content() (first, last int) { return b.first, b.last }

// goBlocks returns every fenced ```go block in the document.
func (d *docFile) goBlocks() []goBlock {
	var out []goBlock
	in, start := false, 0
	for i, l := range d.lines {
		t := strings.TrimSpace(l)
		if !in && strings.EqualFold(t, "```go") {
			in, start = true, i+2
			continue
		}
		if in && strings.HasPrefix(t, "```") {
			in = false
			out = append(out, goBlock{first: start, last: i})
		}
	}
	return out
}

// docField is one field line of a documented struct block.
type docField struct {
	lineNo int
	raw    string
	name   string
	typ    string
	comm   string
}

func (f docField) describe() string { return f.name + " " + f.typ }

// structFields returns the field lines of the ```go block declaring
// `type <name> struct {`, keeping each field's document line number so a
// failure can name the line a maintainer has to edit.
func (d *docFile) structFields(name string) ([]docField, error) {
	needle := "type " + name + " struct {"
	var found []docField
	blocks := d.goBlocks()
	for _, b := range blocks {
		body := strings.Join(d.lines[b.first-1:b.last], "\n")
		if !strings.Contains(body, needle) {
			continue
		}
		found = nil
		for no := b.first; no < b.last; no++ {
			text := d.lines[no-1]
			trimmed := strings.TrimSpace(text)
			if trimmed == "" || trimmed == "}" || strings.HasPrefix(trimmed, "type ") {
				continue
			}
			code, comm := text, ""
			if i := strings.Index(code, "//"); i >= 0 {
				comm, code = strings.TrimSpace(code[i+2:]), strings.TrimSpace(code[:i])
			}
			parts := strings.Fields(code)
			if len(parts) < 2 {
				return nil, fmt.Errorf("%s: cannot read struct field %q of %s",
					d.at(no), strings.TrimSpace(text), name)
			}
			found = append(found, docField{
				lineNo: no,
				raw:    strings.TrimRight(text, " \t"),
				name:   parts[0],
				typ:    strings.Join(parts[1:], ""),
				comm:   comm,
			})
		}
		return found, nil
	}
	return nil, fmt.Errorf("%s: no ```go block declares %q; the check cannot run", d.rel, needle)
}

// structBlockFields returns the field lines of the documented `type <name>
// struct` block, stopping at that struct's own closing brace.
//
// structFields cannot be reused for a block that declares more than one type: it
// walks to the end of the ```go fence, so in a document that lists two structs
// back to back it reads the second struct's fields as part of the first. This
// reader is bounded by the brace instead, which is what a caller comparing two
// structs in one block needs.
func (d *docFile) structBlockFields(name string) ([]docField, error) {
	needle := "type " + name + " struct {"
	for _, b := range d.goBlocks() {
		first, last := b.content()
		open := 0
		for no := first; no <= last; no++ {
			trimmed := strings.TrimSpace(d.lines[no-1])
			if strings.HasPrefix(trimmed, needle) {
				open = no
				break
			}
		}
		if open == 0 {
			continue
		}
		var found []docField
		for no := open + 1; no <= last; no++ {
			trimmed := strings.TrimSpace(d.lines[no-1])
			// The struct's own closing brace ends the block. Comment lines and
			// blank lines are not fields; anything else must read as one, so a
			// block that turns into prose is reported rather than half-read.
			if trimmed == "}" {
				if len(found) == 0 {
					return nil, fmt.Errorf("%s: the documented %s block lists no fields; the check cannot run",
						d.at(no), name)
				}
				return found, nil
			}
			if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
				continue
			}
			code, comm := d.lines[no-1], ""
			if i := strings.Index(code, "//"); i >= 0 {
				comm, code = strings.TrimSpace(code[i+2:]), strings.TrimSpace(code[:i])
			}
			parts := strings.Fields(code)
			if len(parts) < 2 {
				return nil, fmt.Errorf("%s: cannot read struct field %q of %s", d.at(no), trimmed, name)
			}
			found = append(found, docField{
				lineNo: no,
				raw:    d.lines[no-1],
				name:   parts[0],
				typ:    strings.Join(parts[1:], ""),
				comm:   comm,
			})
		}
	}
	return nil, fmt.Errorf("%s: no ```go block declares %q with a closing brace; the check cannot run", d.rel, needle)
}

// docFunc is a func signature written in a ```go block, reduced to the parts a
// caller can depend on: the receiver type, the name, each parameter's type and
// the result types. Parameter *names* are deliberately not part of the
// comparison, because renaming a parameter changes no contract; where a name
// matters — the confirmation flag — the check reads it from the code.
type docFunc struct {
	lineNo   int
	recvType string
	name     string
	params   []string
	results  []string
}

// render prints the signature the way a Go declaration would, minus the
// receiver identifier, so a document and a declaration are compared like for
// like.
func (f docFunc) render() string {
	s := "func (" + f.recvType + ") " + f.name + "(" + strings.Join(f.params, ", ") + ")"
	if len(f.results) == 0 {
		return s
	}
	return s + " (" + strings.Join(f.results, ", ") + ")"
}

var docFuncRe = regexp.MustCompile(
	`^func\s+\(\s*\w+\s+([^)]+?)\s*\)\s*(\w+)\s*\(([^)]*)\)\s*(.*?)\s*$`)

// funcs returns every func signature declared in a ```go block, in document
// order. A `func` line that cannot be read is an error: silently skipping it
// would let a signature drift out of the document without any check noticing.
func (d *docFile) funcs() ([]docFunc, error) {
	var out []docFunc
	for _, b := range d.goBlocks() {
		first, last := b.content()
		for no := first; no <= last; no++ {
			text := strings.TrimSpace(d.lines[no-1])
			if !strings.HasPrefix(text, "func ") {
				continue
			}
			m := docFuncRe.FindStringSubmatch(text)
			if m == nil {
				return nil, fmt.Errorf("%s: cannot read the documented signature %q; the check cannot run",
					d.at(no), text)
			}
			out = append(out, docFunc{
				lineNo:   no,
				recvType: m[1],
				name:     m[2],
				params:   splitParams(m[3]),
				results:  splitParams(strings.Trim(m[4], "()")),
			})
		}
	}
	return out, nil
}

// splitParams splits a comma-separated parameter or result list, keeping
// bracketed groups such as `[]Order` or `(*SubmitResult, error)` intact.
func splitParams(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	depth, start := 0, 0
	for i, r := range s {
		switch r {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, paramType(strings.TrimSpace(s[start:i])))
				start = i + 1
			}
		}
	}
	return append(out, paramType(strings.TrimSpace(s[start:])))
}

// paramType drops a parameter's name, keeping only its type. A parameter
// written without a name is its own type.
func paramType(p string) string {
	if p == "" {
		return ""
	}
	if _, rest, ok := strings.Cut(p, " "); ok {
		return strings.Join(strings.Fields(rest), " ")
	}
	return p
}

// codeFunc reduces a declaration to the same shape as docFunc, so the two can
// be compared field by field.
func codeFunc(fn *ast.FuncDecl) docFunc {
	recv := ""
	if fn.Recv != nil && len(fn.Recv.List) == 1 {
		recv = exprString(fn.Recv.List[0].Type)
	}
	return docFunc{
		recvType: recv,
		name:     fn.Name.Name,
		params:   fieldTypes(fn.Type.Params),
		results:  fieldTypes(fn.Type.Results),
	}
}

func fieldTypes(fl *ast.FieldList) []string {
	if fl == nil {
		return nil
	}
	var out []string
	for _, f := range fl.List {
		n := len(f.Names)
		if n == 0 {
			n = 1
		}
		for range n {
			out = append(out, exprString(f.Type))
		}
	}
	return out
}

// --- signatures with or without a receiver ------------------------------------

// docSignature is a func signature in a ```go block, with or without a receiver.
// recvType is empty for a package-level func.
//
// It is a second reader rather than an extension of docFunc for two reasons, and
// both are about not changing a reader the 06/09 checks already depend on:
//
//   - docFunc only matches a method, so `func NewClient(opts ...Option) ...` — the
//     signature 02-client.md documents for a constructor — is invisible to it.
//   - docFunc does not strip a trailing `//` comment, so a documented line such
//     as `func (c *Client) REST() (*RESTSurface, error)   // IB REST` would be
//     parsed with the comment glued to the last result, and the result list would
//     come out as one mangled string.
type docSignature struct {
	lineNo   int
	recvType string
	name     string
	params   []string
	results  []string
}

// render prints the signature the way a Go declaration would, dropping the
// receiver identifier and keeping a package-level func receiver-less.
func (s docSignature) render() string {
	var b strings.Builder
	b.WriteString("func ")
	if s.recvType != "" {
		b.WriteString("(" + s.recvType + ") ")
	}
	b.WriteString(s.name + "(" + strings.Join(s.params, ", ") + ")")
	if len(s.results) > 0 {
		b.WriteString(" (" + strings.Join(s.results, ", ") + ")")
	}
	return b.String()
}

// docSignatureRe matches both spellings. Go's regexp has no non-capturing
// groups, so the whole receiver is the optional group and the bare type is the
// nested one: group 2 is the receiver type, 3 the name, 4 the parameters, 5 the
// results.
var docSignatureRe = regexp.MustCompile(
	`^func\s+(\(\s*\w+\s+([^)]+?)\s*\))?\s*(\w+)\s*\(([^)]*)\)\s*(.*?)\s*$`)

// signatures returns every func signature in a ```go block, in document order,
// with or without a receiver. A `func` line that cannot be read is an error, for
// the same reason funcs treats it as one: skipping it would let a signature drift
// out of a document with nothing noticing.
func (d *docFile) signatures() ([]docSignature, error) {
	var out []docSignature
	for _, b := range d.goBlocks() {
		first, last := b.content()
		for no := first; no <= last; no++ {
			text := strings.TrimSpace(d.lines[no-1])
			if !strings.HasPrefix(text, "func ") {
				continue
			}
			// A trailing comment is prose. `func (c *Client) REST() (*S, error)
			//   // IB REST` is a two-result signature, and cutting the comment is
			// what keeps the trailing `)` from being mistaken for a parameter
			// group's closer.
			if i := strings.Index(text, "//"); i >= 0 {
				text = strings.TrimSpace(text[:i])
			}
			m := docSignatureRe.FindStringSubmatch(text)
			if m == nil {
				return nil, fmt.Errorf("%s: cannot read the documented signature %q; the check cannot run",
					d.at(no), strings.TrimSpace(d.lines[no-1]))
			}
			out = append(out, docSignature{
				lineNo:   no,
				recvType: m[2],
				name:     m[3],
				params:   splitParams(m[4]),
				results:  splitParams(strings.Trim(m[5], "()")),
			})
		}
	}
	return out, nil
}

// findSignature returns the documented signature named name on the receiver type
// recvType. An empty recvType matches a package-level func.
func findSignature(sigs []docSignature, recvType, name string) (docSignature, bool) {
	for _, s := range sigs {
		if s.name == name && s.recvType == recvType {
			return s, true
		}
	}
	return docSignature{}, false
}

// --- a struct block whose field names are the whole claim ---------------------

// structBlockNames returns the *names* of the fields of the documented
// `type <name> struct` block, in document order.
//
// It is a separate reader because structBlockFields cannot read a line that
// lists several names at once, which is how 02-client.md writes the Client's
// managers:
//
//	sessionManager, accountManager, portfolioManager, tradeManager,
//
// structBlockFields would read that as a field named "sessionManager," whose type
// is "accountManager,portfolioManager,tradeManager," — a field set that exists in
// no language. So a field line is split on commas and the first word of each part
// is a name; a line written the ordinary way yields exactly one part. Types are
// dropped, which is what makes a check written on top of this a *name* check and
// nothing more.
//
// One shape is still unreadable, and deliberately so: a line carrying a single
// trailing-comma name with no type after it. structBlockFields rejects it, and a
// reader here that quietly accepted it would let a document drift into a form the
// 06/09 checks cannot read either.
func (d *docFile) structBlockNames(name string) ([]docField, error) {
	fields, err := d.structBlockFields(name)
	if err != nil {
		return nil, err
	}
	var out []docField
	for _, f := range fields {
		for _, part := range strings.Split(strings.ReplaceAll(f.raw, "\t", " "), ",") {
			code, _ := cutComment(part)
			words := strings.Fields(code)
			if len(words) == 0 {
				continue
			}
			out = append(out, docField{lineNo: f.lineNo, raw: f.raw, name: words[0]})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: the documented %s block lists no field names; the check cannot run", d.rel, name)
	}
	return out, nil
}

// cutComment splits a line at its first `//`, returning the code and the comment.
func cutComment(line string) (code, comm string) {
	if i := strings.Index(line, "//"); i >= 0 {
		return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+2:])
	}
	return strings.TrimSpace(line), ""
}

// --- the module --------------------------------------------------------------

// moduleImportPath returns the module path declared in the repository's go.mod.
// The generated package's import path is derived from it rather than hardcoded,
// so a module rename cannot leave a boundary check quietly passing.
func moduleImportPath(repoRoot string) (string, error) {
	//nolint:gosec // repoRoot is the process working directory, never caller input
	b, err := os.ReadFile(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("go.mod: %w", err)
	}
	for _, line := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			p := strings.TrimSpace(rest)
			if p == "" {
				break
			}
			return p, nil
		}
	}
	return "", fmt.Errorf("go.mod declares no module path; the check cannot run")
}

// repoGoFiles returns every .go file in the module except the generated package
// itself, as repo-relative slash paths, sorted.
//
// Unlike pkgGoFiles this walks the whole tree and keeps _test.go files, because
// the claim it serves is about which *package* reaches for the generated code and
// a test that imports it is still a package that reaches for it. .git and any
// vendor directory are skipped: neither is module code.
func repoGoFiles(repoRoot string, skipDir string) ([]string, error) {
	files, err := collectGoFiles(repoRoot, ".", skipDir)
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func collectGoFiles(root, dir, skipDir string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", dir, err)
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			if name == skipDir || name == ".git" || name == "vendor" || name == "node_modules" {
				continue
			}
			sub, err := collectGoFiles(root, path.Join(dir, name), skipDir)
			if err != nil {
				return nil, err
			}
			out = append(out, sub...)
			continue
		}
		if strings.HasSuffix(name, ".go") {
			out = append(out, path.Join(dir, name))
		}
	}
	return out, nil
}
