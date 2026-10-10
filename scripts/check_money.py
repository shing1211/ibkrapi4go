#!/usr/bin/env python3
# Copyright 2026 shing1211
# SPDX-License-Identifier: Apache-2.0
"""Check public money and quantity fields against ADR 0008.

The scanner examines exported fields on exported structs in hand-written
``pkg/ibkr`` and ``internal`` code. Generated client code, unexported wire
adapters, function bodies, observability aggregates, and legacy ID fields are
outside this public money-field gate.

A second rule covers what the field scan structurally cannot see: a money value
that reaches the caller as a string produced by formatting a binary float. The
field scan only inspects struct declarations, so it is blind to function bodies -
and that is exactly where the tax-voucher defect lived, where a float32 decoded
from the wire was re-rendered with ``strconv.FormatFloat`` and the cents silently
vanished. Two vectors are rejected:

* ``strconv.FormatFloat`` anywhere in production ``pkg/ibkr`` code, and
* any helper whose signature takes a binary float and returns a string, which is
  how the same defect returns after the direct call is removed.

The only legitimate sources of a money string are a quoted ``string`` field from
the gateway and ``json.Number`` via ``rawToString``/``jsonNumberToStr``; both
preserve the gateway's own digits. ``ALLOWED_FLOAT_FORMATS`` is the escape hatch
for a genuine need, and an entry there has to say why.
"""

import glob
import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
FLOAT_TYPE = re.compile(r"\b(?:float32|float64)\b")
STRUCT_START = re.compile(r"^\s*type\s+([A-Za-z_][A-Za-z0-9_]*)\s+struct\s*\{")
FIELD_START = re.compile(r"^\s*([A-Z][A-Za-z0-9_]*(?:\s*,\s*[A-Z][A-Za-z0-9_]*)*)\s+(.+?)\s*$")

# Production sites permitted to format a float, each with the reason it is safe.
# Empty today: no production code needs it, which is the point.
ALLOWED_FLOAT_FORMATS = {
    # "pkg/ibkr/example.go:42": "formats a latency histogram, not a money value",
}

FORMAT_FLOAT = re.compile(r"\bstrconv\.FormatFloat\b")
# func name(any ... float32|float64 ... ) string   - a float-to-string helper
FLOAT_TO_STRING_HELPER = re.compile(
    r"^func\s+([A-Za-z_][A-Za-z0-9_]*)\s*\([^)]*\b(?:float32|float64|\*\s*(?:float32|float64))\b[^)]*\)\s*string\b"
)

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


def scan_float_formatting(source: str, path: str) -> list[tuple[int, str]]:
    """Reject float-to-string conversions in production pkg/ibkr code.

    The struct-field scan cannot see these: it inspects declarations, not
    function bodies, and the tax-voucher defect was entirely inside one.
    """
    violations = []
    for lineno, line in enumerate(source.splitlines(), 1):
        key = f"{path}:{lineno}"
        if key in ALLOWED_FLOAT_FORMATS:
            continue
        if FORMAT_FLOAT.search(_code(line)):
            violations.append(
                (lineno, "strconv.FormatFloat formats a binary float; a money string must come "
                         "from a quoted string field or json.Number so the gateway's digits survive")
            )
            continue
        helper = FLOAT_TO_STRING_HELPER.match(_code(line))
        if helper:
            violations.append(
                (lineno, f"{helper.group(1)}() takes a binary float and returns a string; "
                         "money must not be rendered through a float mantissa (ADR 0008)")
            )
    return violations


def _self_test() -> bool:
    cases = [
        ("type Public struct {\n Money float64\n}", True),
        ("type private struct {\n Money float64\n}", False),
        ("func f() { Value := float64(1) }", False),
        ("type Public struct {\n Handler func() float64\n}", False),
        ("type Public struct {\n Values map[string]float64\n}", True),
    ]
    if not all(bool(scan_source(source, "pkg/ibkr/test.go")) == expected for source, expected in cases):
        return False

    fmt_cases = [
        # The tax-voucher defect, verbatim in shape.
        ("func float32ToStr(p *float32) string {\n return strconv.FormatFloat(0, 'f', -1, 32)\n}", True),
        # The helper reintroduced without the direct FormatFloat call.
        ("func f32ToStr(p *float32) string {\n return \"\"\n}", True),
        ("func moneyFromFloat(v float64) string {\n return \"\"\n}", True),
        # The sanctioned path: json.Number keeps the gateway's digits.
        ("func jsonNumberToStr(n *json.Number) string {\n return n.String()\n}", False),
        # An unrelated int formatting is not money.
        ("func idToString(v int) string {\n return strconv.Itoa(v)\n}", False),
        # Conversions inside a function body are not struct fields.
        ("func ratio(a, b float64) float64 {\n return a / b\n}", False),
    ]
    return all(
        bool(scan_float_formatting(source, "pkg/ibkr/test.go")) == expected
        for source, expected in fmt_cases
    )


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
            rel = os.path.relpath(path, ROOT).replace("\\", "/")
            for lineno, names, ftype in scan_file(path):
                violations.append(f"{rel}:{lineno}: {names} {ftype}")
            # The float-formatting rule is deliberately narrower than the field
            # rule: it guards pkg/ibkr, where money is produced. internal has no
            # money surface, and its float usage is metrics and breaker state.
            if subdir == "pkg/ibkr":
                with open(path, encoding="utf-8") as handle:
                    for lineno, why in scan_float_formatting(handle.read(), rel):
                        violations.append(f"{rel}:{lineno}: {why}")

    if violations:
        print("money-check: float fields found (ADR 0008):", file=sys.stderr)
        for violation in violations:
            print("  " + violation, file=sys.stderr)
        return 1
    print(f"money-check OK ({scanned} files scanned)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
