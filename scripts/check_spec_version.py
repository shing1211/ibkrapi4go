#!/usr/bin/env python3
# Copyright 2026 shing1211
# SPDX-License-Identifier: Apache-2.0
"""
check_spec_version.py — compare live IBKR OpenAPI spec version against pinned version.

Exit codes:
    0 = live version matches pinned (no drift)
    1 = live version is newer than pinned (drift detected)
    2 = error fetching or parsing spec
"""

import json
import sys
import urllib.request

PINNED_VERSION = "v2.40.0"
SPEC_URL = "https://api.ibkr.com/gw/api/v3/api-docs"


def parse_version(v: str) -> tuple:
    if v.startswith("v"):
        v = v[1:]
    return tuple(int(x) for x in v.split("."))


def main() -> int:
    try:
        with urllib.request.urlopen(SPEC_URL, timeout=30) as resp:
            spec = json.load(resp)
    except Exception as exc:
        print(f"ERROR: failed to fetch spec from {SPEC_URL}: {exc}", file=sys.stderr)
        return 2

    live_version = spec.get("info", {}).get("version", "")

    if not live_version:
        print("ERROR: could not extract version from spec (missing info.version)", file=sys.stderr)
        return 2

    live_tuple = parse_version(live_version)
    pinned_tuple = parse_version(PINNED_VERSION)

    print(f"Live version:   {live_version}")
    print(f"Pinned version: {PINNED_VERSION}")

    if live_tuple == pinned_tuple:
        print("Status: OK — spec is current (no drift)")
        return 0
    elif live_tuple > pinned_tuple:
        print("Status: DRIFT — live spec is newer than pinned version")
        return 1
    else:
        print("Status: OK — live spec is older than pinned (expected in dev)")
        return 0


if __name__ == "__main__":
    sys.exit(main())
