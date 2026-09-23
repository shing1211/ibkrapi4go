#!/usr/bin/env python3
# Copyright 2026 shing1211
# SPDX-License-Identifier: Apache-2.0
"""
check_money.py — enforce ADR 0008: no float32/float64 fields in pkg/ibkr.

Money, prices, and quantities must be strings or json.Number, never binary
floats. This scanner uses a hand-written state machine (not full AST parsing)
to detect exported struct fields whose declared type is, contains, or resolves
to float32 or float64. Function parameters are intentionally excluded.

Supported type patterns:
  T                              (bare)
  *T                             (pointer)
  []T                            (slice)
  [N]T                           (array)
  map[K]T                        (map)
  []*T / []*[]T / map[K]*[]T    (pointer/slice/map nesting)
"""

import glob
import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

ALLOWLIST = {
    "client.gen.go": {
        # Generated market-data geographic coordinates, not monetary values.
        "Latitude": True,
        "Longitude": True,
    },
}

STRUCT_FIELD = re.compile(r"\b([A-Z][A-Za-z0-9_]*)\s+")
FLOAT_TYPE = re.compile(r"\b(?:float32|float64)\b")


def _skip_word(t: str, i: int) -> int:
    n = len(t)
    while i < n and (t[i].isalnum() or t[i] == "_"):
        i += 1
    return i


def _skip_type(t: str, i: int) -> int:
    n = len(t)
    while i < n:
        c = t[i]
        if c == "*":
            i += 1
            continue
        if c == "[":
            depth = 1
            i += 1
            while i < n and depth > 0:
                if t[i] == "[": depth += 1
                elif t[i] == "]": depth -= 1
                i += 1
            continue
        if c.isalpha() or c == "_":
            i = _skip_word(t, i)
            if i < n and t[i] == "[":
                i = _skip_type(t, i)
            return i
        if c == "(":
            depth = 1
            i += 1
            while i < n and depth > 0:
                if t[i] == "(": depth += 1
                elif t[i] == ")": depth -= 1
                i += 1
            continue
        if c == "<":
            while i < n and t[i] != ">":
                i += 1
            i += 1
            continue
        if c == "{":
            depth = 1
            i += 1
            while i < n and depth > 0:
                if t[i] == "{": depth += 1
                elif t[i] == "}": depth -= 1
                i += 1
            continue
        break
    return i


def _is_float_field(type_expr: str) -> bool:
    t = type_expr.strip()
    n = len(t)
    i = 0

    while i < n:
        c = t[i]

        if c == "*":
            i += 1
            continue

        if c == "[":
            depth = 1
            i += 1
            while i < n and depth > 0:
                if t[i] == "[": depth += 1
                elif t[i] == "]": depth -= 1
                i += 1
            continue

        if c.isalpha() or c == "_":
            j = i
            i = _skip_word(t, i)
            tok = t[j:i]

            if tok in ("float32", "float64"):
                return True
            if tok in ("chan", "func"):
                return False

            if i < n and t[i] == "[":
                i = _skip_type(t, i)
            if i >= n:
                return False
            if t[i] == "{":
                return True
            continue

        if c == "(":
            depth = 1
            i += 1
            while i < n and depth > 0:
                if t[i] == "(": depth += 1
                elif t[i] == ")": depth -= 1
                i += 1
            continue

        if c == "<":
            while i < n and t[i] != ">":
                i += 1
            i += 1
            continue

        i += 1

    return False


def scan_file(path: str) -> list[tuple[int, str, str]]:
    violations = []
    basename = os.path.basename(path)
    allowlist = ALLOWLIST.get(basename, {})

    with open(path, encoding="utf-8") as fh:
        for lineno, raw in enumerate(fh, 1):
            code = raw.split("//", 1)[0]
            if not code.strip():
                continue

            fm = STRUCT_FIELD.search(code)
            if not fm:
                continue
            fname = fm.group(1)
            if fname in allowlist:
                continue

            type_rest = code[fm.end():].strip()
            if _is_float_field(type_rest):
                violations.append((lineno, fname, type_rest))

    return violations


def main() -> int:
    violations = []
    scanned = 0

    for subdir in ("pkg/ibkr", "internal", "client"):
        pattern = os.path.join(ROOT, subdir, "*.go")
        for path in sorted(glob.glob(pattern)):
            if path.endswith("_test.go"):
                continue
            scanned += 1
            for lineno, fname, ftype in scan_file(path):
                rel = os.path.relpath(path, ROOT)
                violations.append(f"{rel}:{lineno}: {fname} {ftype}")

    if violations:
        print("money-check: float fields found (ADR 0008):", file=sys.stderr)
        for v in violations:
            print("  " + v, file=sys.stderr)
        return 1
    print(f"money-check OK ({scanned} files scanned)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
