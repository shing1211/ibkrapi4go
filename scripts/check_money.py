#!/usr/bin/env python3
# Copyright 2026 shing1211
# SPDX-License-Identifier: Apache-2.0
"""Check public money and quantity fields against ADR 0008.

The scanner examines exported fields on exported structs in hand-written
``pkg/ibkr`` and ``internal`` code. Generated client code, unexported wire
adapters, function bodies, observability aggregates, and legacy ID fields are
outside this public money-field gate.
"""

import glob
import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
FLOAT_TYPE = re.compile(r"\b(?:float32|float64)\b")
STRUCT_START = re.compile(r"^\s*type\s+([A-Za-z_][A-Za-z0-9_]*)\s+struct\s*\{")
FIELD_START = re.compile(r"^\s*([A-Z][A-Za-z0-9_]*(?:\s*,\s*[A-Z][A-Za-z0-9_]*)*)\s+(.+?)\s*$")

ALLOWED_FIELDS = {
    ("internal/metrics.go", "MetricsSnapshot"): {"Sum", "Min", "Max", "Gauges"},
    ("internal/metrics.go", "HistogramSnapshot"): {"Sum", "Min", "Max"},
    ("pkg/ibkr/rest_banking.go", "BankInstructionCreateRequest"): {"ClientInstructionID"},
}


def _code(line: str) -> str:
    result = []
    quote = None
    escaped = False
    for char in line:
        if quote == '"':
            if escaped:
                escaped = False
            elif char == "\\":
                escaped = True
            elif char == quote:
                quote = None
            continue
        if quote == "'":
            if escaped:
                escaped = False
            elif char == "\\":
                escaped = True
            elif char == quote:
                quote = None
            continue
        if char == '"' or char == "'":
            quote = char
            continue
        if char == "/" and len(result) > 0 and result[-1] == "/":
            break
        result.append(char)
    return "".join(result).strip()


def _brace_delta(line: str) -> int:
    code = _code(line)
    return code.count("{") - code.count("}")


def _is_float_type(type_expr: str) -> bool:
    value = type_expr.strip()
    if value.startswith("func") or value.startswith("chan"):
        return False
    return bool(FLOAT_TYPE.search(value))


def scan_source(source: str, path: str = "") -> list[tuple[int, str, str]]:
    lines = source.splitlines()
    violations = []
    index = 0
    while index < len(lines):
        start = STRUCT_START.match(_code(lines[index]))
        if not start:
            index += 1
            continue

        type_name = start.group(1)
        if not type_name[0].isupper():
            index += 1
            continue

        depth = _brace_delta(lines[index])
        end = index + 1
        while end < len(lines) and depth > 0:
            depth += _brace_delta(lines[end])
            end += 1

        field_depth = 1
        for line_index in range(index + 1, end - 1):
            line = _code(lines[line_index])
            if field_depth >= 1:
                match = FIELD_START.match(line)
                if match:
                    names = [name.strip() for name in match.group(1).split(",")]
                    type_expr = match.group(2).split("`", 1)[0].strip()
                    allowed = ALLOWED_FIELDS.get((path.replace("\\", "/"), type_name), set())
                    if _is_float_type(type_expr) and any(name not in allowed for name in names):
                        violations.append((line_index + 1, match.group(1), type_expr))
            field_depth += _brace_delta(lines[line_index])
        index = max(index + 1, end)
    return violations


def scan_file(path: str) -> list[tuple[int, str, str]]:
    with open(path, encoding="utf-8") as handle:
        return scan_source(handle.read(), os.path.relpath(path, ROOT))


def _self_test() -> bool:
    cases = [
        ("type Public struct {\n Money float64\n}", True),
        ("type private struct {\n Money float64\n}", False),
        ("func f() { Value := float64(1) }", False),
        ("type Public struct {\n Handler func() float64\n}", False),
        ("type Public struct {\n Values map[string]float64\n}", True),
    ]
    return all(bool(scan_source(source, "pkg/ibkr/test.go")) == expected for source, expected in cases)


def main() -> int:
    if not _self_test():
        print("money-check: internal self-test failed", file=sys.stderr)
        return 1

    violations = []
    scanned = 0
    for subdir in ("pkg/ibkr", "internal"):
        for path in sorted(glob.glob(os.path.join(ROOT, subdir, "*.go"))):
            if path.endswith("_test.go"):
                continue
            scanned += 1
            for lineno, names, ftype in scan_file(path):
                rel = os.path.relpath(path, ROOT)
                violations.append(f"{rel}:{lineno}: {names} {ftype}")

    if violations:
        print("money-check: float fields found (ADR 0008):", file=sys.stderr)
        for violation in violations:
            print("  " + violation, file=sys.stderr)
        return 1
    print(f"money-check OK ({scanned} files scanned)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
