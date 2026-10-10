#!/usr/bin/env python3
# Copyright 2026 shing1211
# SPDX-License-Identifier: Apache-2.0
"""
gen_spec_index.py — regenerate docs/SPEC.md from the IBKR OpenAPI spec.

Usage:
    python3 scripts/gen_spec_index.py specs/ibkr_spec.json > docs/SPEC.md
"""

import collections
import json
import sys
from datetime import date


def surface(path: str) -> str:
    if path.startswith("/v1/api"):
        return "CPAPI"
    if path.startswith("/gw/api"):
        return "IB REST"
    if path.startswith("/oauth2"):
        return "IB REST"
    return "other"


def auth(op: dict) -> str:
    s = op.get("security")
    if not s:
        return "-"
    schemes = sorted({k for e in s for k in e})
    return ", ".join(schemes) if schemes else "-"


def main() -> None:
    with open(sys.argv[1], encoding="utf-8") as fh:
        spec = json.load(fh)
    paths = spec.get("paths", {})
    info = spec.get("info", {})
    schemas = len(spec.get("components", {}).get("schemas", {}))

    ops = {"CPAPI": 0, "IB REST": 0, "other": 0}
    by_tag: dict[str, list] = collections.defaultdict(list)

    for path, item in paths.items():
        for method, op in item.items():
            if not isinstance(op, dict):
                continue
            surf = surface(path)
            ops[surf] = ops.get(surf, 0) + 1
            tags = op.get("tags") or ["(untagged)"]
            by_tag[tags[0]].append((method.upper(), path, op, surf))

    total = sum(ops.values())
    out = []
    out.append("# SPEC.md — IBKR OpenAPI Specification Reference")
    out.append("")
    out.append("> Source: `https://api.ibkr.com/gw/api/v3/api-docs`")
    out.append(f"> Title: {info.get('title', '?')} | Version: {info.get('version', '?')} | OpenAPI: {spec.get('openapi', '?')}")
    out.append(f"> Endpoints: {total} | Schemas: {schemas} | Tags: {len(by_tag)} | Generated: {date.today().isoformat()}")
    out.append("")
    out.append("> **Canonical counts.** All endpoint/schema numbers in this repository")
    out.append("> derive from this file. Regenerate with `scripts/gen_spec_index.py` — do")
    out.append("> not edit by hand.")
    out.append("")
    out.append("---")
    out.append("")
    out.append("## Servers")
    out.append("")
    out.append("| Environment | URL |")
    out.append("|-------------|-----|")
    for s in spec.get("servers", []):
        out.append(f"| {s.get('description', '—')} | `{s.get('url', '')}` |")
    out.append("")
    out.append("## Surfaces and auth")
    out.append("")
    out.append("| Surface | Base path | Operations | Auth |")
    out.append("|---------|-----------|-----------:|------|")
    out.append(f"| Client Portal API (CPAPI) | `/v1/api/*` | {ops['CPAPI']} | `ssoBearer` |")
    out.append(f"| IB REST API | `/gw/api/v1/*`, `/gw/api/v2/*`, `/oauth2/*` | {ops['IB REST']} | `oauth2Bearer` |")
    out.append(f"| **Total** | | **{total}** | |")
    out.append("")
    out.append(f"Both surfaces are implemented: {ops['CPAPI']} CPAPI (`ssoBearer`) + {ops['IB REST']} IB REST (`oauth2Bearer`). See [ADR 0001](./adr/0001-two-api-surfaces.md) and [ADR 0011](./adr/0011-oauth2-surface.md).")
    out.append("")
    out.append("---")
    out.append("")
    out.append(f"## Endpoint index ({total} operations)")
    out.append("")

    for tag in sorted(by_tag):
        entries = by_tag[tag]
        entries.sort(key=lambda e: (e[1], e[0]))
        out.append(f"### {tag} *({len(entries)})*")
        out.append("")
        out.append("| Method | Path | Surface | Auth | Summary | Operation ID |")
        out.append("|--------|------|---------|------|---------|--------------|")
        for method, path, op, surf in entries:
            summary = (op.get("summary") or "").replace("|", "\\|")
            oid = op.get("operationId") or ""
            out.append(f"| {method} | `{path}` | {surf} | {auth(op)} | {summary} | `{oid}` |")
        out.append("")

    out.append("---")
    out.append("")
    out.append("## Known spec defects")
    out.append("")
    out.append("The published spec does not generate cleanly. See [CODEGEN.md](./CODEGEN.md)")
    out.append("for the patches applied by `scripts/patch_spec.py`: path-parameter")
    out.append("mismatches, a duplicate `operationId`, and Go type-name collisions.")
    out.append("")

    sys.stdout.reconfigure(encoding="utf-8")
    print("\n".join(out))


if __name__ == "__main__":
    main()
