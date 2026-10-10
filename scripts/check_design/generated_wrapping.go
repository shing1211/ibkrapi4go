// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

// generated_wrapping.go verifies docs/design/04-generated-wrapping.md against
// the code it describes.
//
// The document's whole point is a boundary: generated code is an implementation
// detail, and a caller must never be able to depend on it. A boundary that is
// written down and not checked is a boundary that erodes one signature at a time,
// and the erosion is invisible until a spec change breaks a caller who never knew
// they depended on the generated types.
//
// The checks are:
//
//   - the import boundary, in both directions. Every package in the module that
//     imports the generated package must be one the document names, and both
//     packages the document names must actually import it — a boundary satisfied
//     vacuously would be the cheapest possible way for the check to pass.
//   - the exported surface: no exported declaration in pkg/ibkr may name a
//     generated type, directly or through an exported struct's field types.
//
// What is deliberately NOT checked:
//
//   - The Adapter pattern code block. It is a sketch, and it is a *stale* sketch:
//     it names `accountSummaryGenerated` and `toAccountSummary`, neither of which
//     is declared anywhere, and it gives the public type a `NetLiq` and a
//     `Currency` field that pkg/ibkr.AccountSummary does not have
//     (pkg/ibkr/account.go:36 has NetLiquidationValue and no Currency at all).
//     The block's `{ ... }` and `// ...` markers say it is illustrative; a check
//     over it would be checking an example, not the code. Recorded here because
//     it is a real disagreement a reader can trip over, and left to a human.
//   - "Public types can carry domain semantics (validated ids, string money)". The
//     string-money half is enforced over the whole tree by scripts/check_money.py,
//     which is the gate 07-money-and-numbers.md exists to point at; repeating it
//     here would make one fact two checks and neither stronger. The
//     validated-ids half is a design intent, not a fact a checker can read.
//   - "`oneOf`/`anyOf` unions are modeled as a tagged union in public types, not
//     as raw generated types". This is covered by the exported-surface check: a
//     `client.OneOf…` in an exported signature is exactly the leak it describes,
//     and there is nothing further to say about how a union is shaped.
//   - "This is enforced by review, not by a linter (see `.golangci.yml`), to avoid
//     false positives in tests." A statement about a policy, and the false
//     positives it avoids are real: this check reads the *packages* that import
//     the generated code, tests included, because a test that imports it is still
//     a package coupled to it. That is deliberately stricter than the linter the
//     document declines to configure, and the difference is called out in
//     checkImportBoundary.

import (
	"fmt"
	"go/ast"
	"path"
	"sort"
	"strings"
)

const (
	wrapDoc      = "docs/design/04-generated-wrapping.md"
	pubIbdAlias  = "pkg/ibkr"
	genPkgSuffix = "/client"
)

// checkGeneratedWrapping runs the docs/design/04-generated-wrapping.md checks as
// one unit.
func checkGeneratedWrapping(repoRoot string) error {
	return collectDocErrors(repoRoot, wrapDoc, []docCheck{
		{"import boundary", checkImportBoundary},
		{"no generated type is exported", checkNoGeneratedTypeIsExported},
	})
}

// --- claim 1: the import boundary --------------------------------------------

// checkImportBoundary verifies "`client/*.gen.go` is imported **only** by
// `pkg/ibkr` and `internal/`".
//
// Both halves are read from the module rather than hardcoded. The import path is
// derived from go.mod's module line and the generated package's own directory, so
// a module rename cannot leave this check comparing against a path nothing
// imports any more — which would pass, and mean nothing. The allowed directories
// come from the document's own sentence, matched by name, so re-reading the
// document is what changes what is allowed.
//
// Test files are included. The document's enforcement note says the boundary is
// enforced "by review, not by a linter ... to avoid false positives in tests",
// and this check is not that linter: a _test.go file that imports the generated
// package puts its package inside the boundary, and a package outside the boundary
// that reaches for generated code is a leak whether the import sits in a test or
// not. The directories under internal/ are part of `internal` — internal/mockgateway
// imports the generated package from its tests today, and it is not a third
// importer.
func checkImportBoundary(repoRoot string) error {
	files, err := repoGoFiles(repoRoot, "client")
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no Go files found in the module; the import-boundary check cannot run")
	}
	doc, err := parseDoc(repoRoot, wrapDoc)
	if err != nil {
		return err
	}
	roots, allowed, err := docImportBoundaryDirs(doc, moduleDirs(files))
	if err != nil {
		return err
	}
	module, err := moduleImportPath(repoRoot)
	if err != nil {
		return err
	}
	genPath := module + genPkgSuffix
	importers := map[string][]string{}
	for _, rel := range files {
		code, err := parseGo(repoRoot, rel)
		if err != nil {
			return err
		}
		if !importsPath(code.file, genPath) {
			continue
		}
		dir := path.Dir(rel)
		importers[dir] = append(importers[dir], rel)
	}
	if len(importers) == 0 {
		return fmt.Errorf("no package in the module imports %s, so %s's boundary names an importer that does "+
			"not exist; either the generated package moved or the document is stale", genPath, wrapDoc)
	}
	var dirs []string
	for d := range importers {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	for _, d := range dirs {
		if !allowed[d] {
			return fmt.Errorf("import-boundary drift: package %s imports the generated package %s (in %s); "+
				"%s documents the generated code as importable only by %s",
				d, genPath, importers[d][0], wrapDoc, allowedList(roots))
		}
	}
	// The other direction: the boundary names two importers, so each of them has
	// to reach the generated code somewhere. "internal" is a directory tree, not a
	// single package — the document writes it with a trailing slash — so what is
	// required is that the root itself or a package beneath it imports the
	// generated package. Without this half, deleting every generated call from
	// internal/ would satisfy the document's sentence.
	//
	// The requirement is on the *roots* the document named, not on every directory
	// the expansion added: internal/ contains packages that legitimately do not
	// touch the generated code, and demanding that they all do would be a rule the
	// document does not state.
	for want := range roots {
		if importerTreeContains(importers, want) {
			continue
		}
		return fmt.Errorf("import-boundary drift: neither package %s nor anything beneath it imports the generated "+
			"package, but %s names it as one of the only two permitted importers", want, wrapDoc)
	}
	return nil
}

// importerTreeContains reports whether root is itself an importer or an ancestor
// directory of one.
func importerTreeContains(importers map[string][]string, root string) bool {
	for dir := range importers {
		if dir == root || strings.HasPrefix(dir, root+"/") {
			return true
		}
	}
	return false
}

// moduleDirs returns the set of directories that contain a Go file.
func moduleDirs(files []string) map[string]bool {
	out := map[string]bool{}
	for _, f := range files {
		out[path.Dir(f)] = true
	}
	return out
}

// boundaryDirsRe reads the allowed directories out of the document's boundary
// rule. The sentence is matched for its shape — a line that says the generated
// code is imported "only" by some backticked names — and the directories are read
// out of it, so the document, not this program, decides the boundary.
// docImportBoundaryDirs returns the roots the document permits and the full set of
// directories those roots cover.
//
// The roots are the backticked names in the boundary sentence, path-cleaned. The
// covered set is the roots plus everything beneath a root called `internal`. Both
// are kept because the two directions need different ones: an importer is allowed
// if it is in the covered set, and a root is *required* to reach the generated code
// if the document names it.
func docImportBoundaryDirs(doc *docFile, dirs map[string]bool) (roots, allowed map[string]bool, err error) {
	for _, l := range doc.lines {
		if !strings.Contains(l, "is imported") || !strings.Contains(l, "only") {
			continue
		}
		out := map[string]bool{}
		for _, m := range backtickRe.FindAllStringSubmatch(l, -1) {
			n := m[1]
			// The first span names the generated files, not a directory.
			if strings.Contains(n, ".go") {
				continue
			}
			// The document writes `internal/`; the module path is `internal`.
			out[path.Clean(n)] = true
		}
		if len(out) < 2 {
			return nil, nil, fmt.Errorf("%s: the boundary rule names %d importer directories; it has to name the "+
				"two permitted importers, and the check cannot run on fewer", doc.rel, len(out))
		}
		return out, expandUnderInternal(out, dirs), nil
	}
	return nil, nil, fmt.Errorf("%s: no boundary rule naming the permitted importers was found; the check cannot run", doc.rel)
}

// expandUnderInternal turns the `internal` entry into itself and every directory
// beneath it. A path segment is required, so a package called `internalx` is not
// swept in. The directories come from the module walk rather than from a
// hardcoded list, because "everything under internal/" is a claim about the
// directory tree and the tree is the authority on itself.
func expandUnderInternal(in, dirs map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k := range in {
		out[k] = true
	}
	var roots []string
	for k := range in {
		if path.Base(k) == "internal" {
			roots = append(roots, k)
		}
	}
	sort.Strings(roots)
	for _, r := range roots {
		for d := range dirs {
			if strings.HasPrefix(d, r+"/") {
				out[d] = true
			}
		}
	}
	return out
}

func allowedList(allowed map[string]bool) string {
	var names []string
	for k := range allowed {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, " and ")
}

// importsPath reports whether a file imports the named path, read from the import
// declaration rather than from a line of text: an aliased or dot import is
// resolved the same way a plain one is.
func importsPath(file *ast.File, want string) bool {
	for _, im := range file.Imports {
		if im.Path == nil {
			continue
		}
		p, ok := stringLit(im.Path)
		if ok && p == want {
			return true
		}
	}
	return false
}

// --- claim 2: nothing generated is exported -----------------------------------

// checkNoGeneratedTypeIsExported verifies "No exported symbol in `pkg/ibkr` may
// be a generated type."
//
// The scan is over declarations, not lines. For every exported top-level
// declaration in the package — a type, a function, a method — the types its
// signature or type expression mentions are walked, and any reference qualified by
// the generated package's import name is drift. An exported struct's *exported*
// field types count, because a caller can read a field as readily as a parameter.
// An unexported field does not: `Client.generated` is a generated type the SDK
// holds privately, and reaching it would take a caller writing to this package.
//
// Only signatures are read, never bodies: an exported constructor may legitimately
// build a generated value internally. What it must not do is hand one to a caller.
//
// The generated package is named by its import declaration rather than by the
// literal `client`, so a file that aliases the import (`gen "…/client"`) is read
// under its alias and a file that happens to have a local variable called `client`
// is not mistaken for the package.
func checkNoGeneratedTypeIsExported(repoRoot string) error {
	files, err := pkgGoFiles(repoRoot, pubIbdAlias)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("%s: no Go files found; the exported-surface check cannot run", pubIbdAlias)
	}
	for _, rel := range files {
		code, err := parseGo(repoRoot, rel)
		if err != nil {
			return err
		}
		gen := generatedImportName(code)
		if err != nil {
			return err
		}
		if gen == "" {
			continue
		}
		if err := noGeneratedTypeInExported(code, gen); err != nil {
			return fmt.Errorf("boundary drift: %s: %w; %s documents that no exported symbol in %s may be a "+
				"generated type, because a spec update must not break callers' code",
				rel, err, wrapDoc, pubIbdAlias)
		}
	}
	return nil
}

// generatedImportName returns the local name the file uses for the generated
// package: the alias if it declared one, the package's own name otherwise, and ""
// if the file does not import it.
func generatedImportName(code *goSrc) string {
	for _, im := range code.file.Imports {
		if im.Path == nil {
			continue
		}
		p, ok := stringLit(im.Path)
		if !ok || !strings.HasSuffix(p, genPkgSuffix) {
			continue
		}
		if im.Name != nil {
			return im.Name.Name
		}
		return path.Base(p)
	}
	return ""
}

// noGeneratedTypeInExported walks the file's exported declarations and returns the
// first generated type it finds qualified by gen.
func noGeneratedTypeInExported(code *goSrc, gen string) error {
	for _, d := range code.file.Decls {
		switch decl := d.(type) {
		case *ast.FuncDecl:
			if !decl.Name.IsExported() {
				continue
			}
			// Only the signature is read, not the body: an exported constructor may
			// legitimately build a generated value internally. What it must not do
			// is hand one to a caller.
			if sel := firstGeneratedType(decl.Type, gen); sel != "" {
				return fmt.Errorf("exported %s has the generated type %s in its signature",
					declName(decl), sel)
			}
		case *ast.GenDecl:
			for _, s := range decl.Specs {
				ts, ok := s.(*ast.TypeSpec)
				if !ok || !ts.Name.IsExported() {
					continue
				}
				// A struct's own type expression is `struct{...}`, which names no
				// generated type; its *fields* are the contract and are checked
				// below. Reading the whole expression for every type would descend
				// into the fields and flag `Client.generated`, which is unexported
				// and unreachable from outside the package.
				st, isStruct := ts.Type.(*ast.StructType)
				if !isStruct {
					if sel := firstGeneratedType(ts.Type, gen); sel != "" {
						return fmt.Errorf("exported type %s is declared as %s", ts.Name.Name, sel)
					}
					continue
				}
				// An exported struct's *exported* field types are part of the
				// contract: a caller reads a field as readily as a parameter. An
				// unexported field is not reachable, and `Client.generated` is a
				// generated type the SDK is entitled to hold privately.
				for _, f := range st.Fields.List {
					if !fieldIsExported(f) {
						continue
					}
					if sel := firstGeneratedType(f.Type, gen); sel != "" {
						return fmt.Errorf("exported type %s has an exported field of the generated type %s",
							ts.Name.Name, sel)
					}
				}
			}
		}
	}
	return nil
}

func declName(fn *ast.FuncDecl) string {
	if fn.Recv == nil {
		return "func " + fn.Name.Name
	}
	return "method " + recvBaseType(fn.Recv.List[0].Type) + "." + fn.Name.Name
}

// fieldIsExported reports whether a struct field is part of the exported surface.
// An embedded field counts: its name is the type's, so an embedded generated type
// is reachable as `x.Field` on any value a caller holds.
func fieldIsExported(f *ast.Field) bool {
	if len(f.Names) == 0 {
		return true
	}
	return f.Names[0].IsExported()
}

// firstGeneratedType returns the first expression in e qualified by gen, as the
// source text of that expression, or "".
func firstGeneratedType(e ast.Expr, gen string) string {
	var found string
	ast.Inspect(e, func(n ast.Node) bool {
		if found != "" {
			return false
		}
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == gen {
			found = exprString(sel)
		}
		return true
	})
	return found
}
