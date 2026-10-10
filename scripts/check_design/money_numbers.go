// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// checkMoneyNumberFields verifies the part of docs/design/07-money-and-numbers.md
// that scripts/check_money.py cannot reach.
//
// check_money.py proves the negative claim: no exported field under pkg/ibkr is a
// float. It says nothing about a field the document positively *names*. The
// document claims the tax voucher's money fields are carried as json.Number
// rather than a float, because the gateway sends them as bare JSON numbers and a
// float32 mantissa would round them. That claim is only true while the generated
// client actually declares them that way - and the generated client is
// regenerated from the spec, so a spec edit can quietly undo it.
//
// So this check ties the two together: every field the document names must be
// json.Number in client/client.gen.go, and no float may have crept back into the
// struct the document points at.
func checkMoneyNumberFields(repoRoot string) error {
	const docRel = "docs/design/07-money-and-numbers.md"
	docPath := filepath.Join(repoRoot, filepath.FromSlash(docRel))
	//nolint:gosec // docPath is repoRoot joined with a constant, and repoRoot is the process working directory
	doc, err := os.ReadFile(docPath)
	if err != nil {
		return fmt.Errorf("%s: %w", docRel, err)
	}

	structName, fields, err := documentedJSONNumberFields(string(doc))
	if err != nil {
		return fmt.Errorf("%s: %w", docRel, err)
	}

	clientRel := "client/client.gen.go"
	clientPath := filepath.Join(repoRoot, filepath.FromSlash(clientRel))
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, clientPath, nil, 0)
	if err != nil {
		return fmt.Errorf("%s: parse error: %w", clientRel, err)
	}

	var structType *ast.StructType
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != structName {
				continue
			}
			if st, ok := ts.Type.(*ast.StructType); ok {
				structType = st
			}
		}
	}
	if structType == nil {
		return fmt.Errorf("%s documents %s as carrying json.Number, but %s declares no type %s",
			docRel, structName, clientRel, structName)
	}

	declared := map[string]string{}
	for _, f := range structType.Fields.List {
		if f.Tag == nil || len(f.Names) == 0 {
			continue
		}
		tag := strings.Trim(f.Tag.Value, "`")
		for _, part := range strings.Split(tag, ",") {
			if jsonName, ok := strings.CutPrefix(part, `json:"`); ok {
				key, _, _ := strings.Cut(jsonName, ",")
				declared[key] = exprString(f.Type)
			}
		}
	}

	var errs []string
	for _, field := range fields {
		got, ok := declared[field]
		if !ok {
			errs = append(errs, fmt.Sprintf(
				"%s documents %s.%s as a json.Number money field, but %s declares no such field",
				docRel, structName, field, clientRel))
			continue
		}
		if !strings.Contains(got, "json.Number") {
			errs = append(errs, fmt.Sprintf(
				"%s documents %s.%s as a json.Number money field, but %s declares it as %s - a float mantissa would round it",
				docRel, structName, field, clientRel, got))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "\n"))
	}
	return nil
}

// docJSONNumberFieldsRe captures the sentence shape the document uses to name the
// fields: the backticked field list that follows the struct name in the
// "tax voucher's ... are the second case" clause.
var docJSONNumberFieldsRe = regexp.MustCompile(
	`(?s)voucher's\s+(.+?)\s+are the second case`)

var backtickFieldRe = regexp.MustCompile("`([A-Za-z0-9_]+)`")

func documentedJSONNumberFields(doc string) (structName string, fields []string, err error) {
	m := docJSONNumberFieldsRe.FindStringSubmatch(doc)
	if m == nil {
		return "", nil, fmt.Errorf(
			"no sentence naming the json.Number money fields was found; expected a clause of the form " +
				"\"The tax voucher's `a`, `b` are the second case\" so this check knows which fields to verify")
	}
	for _, f := range backtickFieldRe.FindAllStringSubmatch(m[1], -1) {
		fields = append(fields, f[1])
	}
	if len(fields) == 0 {
		return "", nil, fmt.Errorf(
			"the clause naming the json.Number money fields lists no backticked field names, so there is nothing to verify")
	}
	sort.Strings(fields)
	return "TaxVoucherDTO", fields, nil
}
