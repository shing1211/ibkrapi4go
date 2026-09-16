#!/usr/bin/env python3
# Copyright 2026 shing1211
# SPDX-License-Identifier: Apache-2.0
"""
check_money.py — enforce ADR 0008: no float32/float64 fields in pkg/ibkr.

Money, prices, and quantities must be strings or json.Number, never binary
floats. This scans exported struct fields under pkg/ibkr (excluding tests) and
fails if any is a float. Function parameters (e.g. WithRateLimit(float64)) are
intentionally not flagged.
"""

import glob
import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
FIELD = re.compile(r"^\s+([A-Z][A-Za-z0-9_]*)\s+\*?\[\]?(float32|float64)\b")


def main() -> int:
    violations = []
    for path in sorted(glob.glob(os.path.join(ROOT, "pkg", "ibkr", "*.go"))):
        if path.endswith("_test.go"):
            continue
        with open(path, encoding="utf-8") as fh:
            for lineno, line in enumerate(fh, 1):
                code = line.split("//", 1)[0]
                match = FIELD.match(code)
                if match:
                    rel = os.path.relpath(path, ROOT)
                    violations.append(f"{rel}:{lineno}: {match.group(1)} {match.group(2)}")
    if violations:
        print("money-check: float fields found in pkg/ibkr (ADR 0008):", file=sys.stderr)
        for v in violations:
            print("  " + v, file=sys.stderr)
        return 1
    print("money-check OK: no float fields in pkg/ibkr")
    return 0


if __name__ == "__main__":
    sys.exit(main())
