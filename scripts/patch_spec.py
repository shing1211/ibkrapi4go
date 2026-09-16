#!/usr/bin/env python3
"""
patch_spec.py — Fix known bugs in the IBKR OpenAPI spec before codegen.

Bugs fixed:
  1. /gw/api/v1/balances/query (POST): path has 0 {param} in the URL path,
     but the spec's operation declares a path parameter "accountId" that
     doesn't appear in the URL pattern. oapi-codegen rejects this.
     Fix: remove the spurious path parameter from the operation.

Usage:
    python3 patch_spec.py /path/to/spec.json > patched.json
"""

import json
import sys
from pathlib import Path


def patch_balances_query(op):
    """Remove spurious 'accountId' path param from /balances/query."""
    params = op.get("parameters", [])
    params_clean = [p for p in params if p.get("name") != "accountId"]
    op["parameters"] = params_clean
    return op


def patch_spec(spec: dict) -> dict:
    """Apply all known patches to the spec."""
    paths = spec.get("paths", {})

    # Bug 1: /gw/api/v1/balances/query
    if "/gw/api/v1/balances/query" in paths:
        post_op = paths["/gw/api/v1/balances/query"].get("post", {})
        if post_op:
            paths["/gw/api/v1/balances/query"]["post"] = patch_balances_query(post_op)
            print("  Patched: /gw/api/v1/balances/query — removed spurious accountId param", file=sys.stderr)

    return spec


def main():
    spec_path = Path(sys.argv[1]).expanduser()
    spec = json.loads(spec_path.read_text())

    print(f"Loaded spec: {spec.get('info', {}).get('title', 'unknown')} "
          f"v{spec.get('info', {}).get('version', '?')}", file=sys.stderr)
    print(f"Paths before patch: {len(spec.get('paths', {}))}", file=sys.stderr)

    patched = patch_spec(spec)

    print(f"Paths after patch: {len(patched.get('paths', {}))}", file=sys.stderr)
    json.dump(patched, sys.stdout, indent=2, ensure_ascii=False)


if __name__ == "__main__":
    main()
