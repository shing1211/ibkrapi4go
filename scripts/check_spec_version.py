#!/usr/bin/env python3
# Copyright 2026 shing1211
# SPDX-License-Identifier: Apache-2.0

"""
check_spec_version.py — compare the upstream IBKR OpenAPI spec version
against the version pinned in docs/SPEC.md.

Exit codes:
    0 — upstream and pinned versions match
    1 — versions differ; a ::error is printed for CI consumption
   >1 — fetch or parse error
"""

import urllib.request
import re
import sys
import os

UPSTREAM_URL = "https://api.ibkr.com/gw/api/v3/api-docs"
SPEC_FILE = "docs/SPEC.md"


def get_upstream_version() -> str:
    with urllib.request.urlopen(UPSTREAM_URL, timeout=30) as r:
        spec = __import__("json").load(r)
    return spec["info"]["version"]


def get_pinned_version() -> str:
    text = open(SPEC_FILE).read()
    m = re.search(r"Version:\s*(\d+\.\d+\.\d+)", text)
    if not m:
        raise RuntimeError(f"could not find Version in {SPEC_FILE}")
    return m.group(1)


def main() -> None:
    upstream = get_upstream_version()
    pinned = get_pinned_version()
    print(f"upstream={upstream}  pinned={pinned}", file=sys.stderr)
    if upstream != pinned:
        print(f"::error::Spec version drifted: pinned={pinned} upstream={upstream}")
        sys.exit(1)
    print("OK: upstream and pinned spec versions match", file=sys.stderr)
    sys.exit(0)


if __name__ == "__main__":
    main()
