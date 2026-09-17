#!/usr/bin/env python3
# Copyright 2026 shing1211
# SPDX-License-Identifier: Apache-2.0
#
# patch_gen.py - deterministic post-generation fixups for client/client.gen.go.
#
# The oapi-codegen request-builder template emits unguarded
# runtime.StyleParamWithOptions calls for query parameters whose Go type is a
# bare `interface{}`. When such a parameter is nil, the runtime dereferences a
# nil interface and panics. This script wraps exactly those calls in a
# `if params.X != nil` guard.
#
# It is deterministic and idempotent. Both scripts/codegen.sh (which writes the
# committed client) and scripts/validate_codegen.sh (which regenerates into a
# temp dir and diffs) invoke it, so a fresh generation matches the committed
# output byte-for-byte.
#
# Usage:
#   python3 scripts/patch_gen.py <path-to-gen.go>

import re
import sys

FIX_COMMENT = "// FIX: guard nil interface{} to avoid panic"

# `func NewXRequest(server string[, extra path params], params *XParams)`
FUNC_RE = re.compile(r"^func New\w+Request\(.*\bparams \*(\w+Params)\)")

# The generated query-param emission line, e.g.
#   if queryFrag, err := runtime.StyleParamWithOptions("form", false, "enabled",
#       params.Enabled, runtime.StyleParamOptions{...}); err != nil {
CALL_RE = re.compile(
    r'^(?P<indent>\t+)if queryFrag, err := runtime\.StyleParamWithOptions\('
    r'.*, params\.(?P<field>\w+), .*\); err != nil \{$'
)

IFACE_FIELD_RE = re.compile(r"^\s+(?P<field>\w+)\s+interface\{\}", re.M)


def interface_params(src):
    """Return {(structName, fieldName)} for bare interface{} fields."""
    pairs = set()
    for m in re.finditer(r"type (\w+Params) struct \{(.*?)\n\}", src, re.S):
        struct_name, body = m.group(1), m.group(2)
        for fm in IFACE_FIELD_RE.finditer(body):
            pairs.add((struct_name, fm.group("field")))
    return pairs


def main():
    if len(sys.argv) != 2:
        print("usage: patch_gen.py <gen.go>", file=sys.stderr)
        return 2

    path = sys.argv[1]
    with open(path, "r", encoding="utf-8") as fh:
        original = fh.read()

    iface = interface_params(original)
    lines = original.split("\n")
    out = []
    patched = 0
    cur_struct = None

    i = 0
    while i < len(lines):
        line = lines[i]

        fm = FUNC_RE.match(line)
        if fm:
            cur_struct = fm.group(1)

        cm = CALL_RE.match(line)
        if cm and cur_struct and (cur_struct, cm.group("field")) in iface:
            field = cm.group("field")
            indent = cm.group("indent")

            # Idempotency: already wrapped by a previous run?
            already = (
                len(out) >= 2
                and out[-1].strip() == FIX_COMMENT
                and out[-2].strip() == f"if params.{field} != nil {{"
            )

            # Verify the fixed 7-line emission block before rewriting.
            block = lines[i : i + 7]
            ok = (
                len(block) == 7
                and block[1] == indent + "\t" + "return nil, err"
                and block[2] == indent + "} else {"
                and block[3]
                == indent + "\t" + 'for _, qp := range strings.Split(queryFrag, "&") {'
                and block[4]
                == indent + "\t\t" + "rawQueryFragments = append(rawQueryFragments, qp)"
                and block[5] == indent + "\t" + "}"
                and block[6] == indent + "}"
            )
            if not already and ok:
                out.append(f"{indent}if params.{field} != nil {{")
                out.append(f"{indent}\t{FIX_COMMENT}")
                out.append("\t" + line)
                out.extend("\t" + b for b in block[1:])
                out.append(f"{indent}}}")
                patched += 1
                i += 7
                continue

        out.append(line)
        i += 1

    with open(path, "w", encoding="utf-8") as fh:
        fh.write("\n".join(out))

    print(f"patch_gen: guarded {patched} nil-interface{{}} param(s)", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())
