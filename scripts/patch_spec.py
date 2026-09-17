#!/usr/bin/env python3
# Copyright 2026 shing1211
# SPDX-License-Identifier: Apache-2.0
"""
patch_spec.py — fix known defects in the IBKR OpenAPI spec before codegen.

The published spec (v2.39.0) does not generate cleanly with oapi-codegen.
This script applies four deterministic, idempotent fixes:

  1. Reconcile path parameters with the path template. Some operations declare
     a path parameter that is absent from the path (or whose name differs), and
     some paths declare placeholders with no matching parameter.
     Fix: drop spurious params, rename mismatched ones, add missing ones.

  2. De-duplicate operationIds. `getTradingSchedule` is used by two operations.
     Fix: append a numeric suffix to later occurrences.

  3. Resolve Go type-name collisions after PascalCase normalization
     (e.g. ErrorResponse/errorResponse, User/user).
     Fix: assign x-go-name to later occurrences.

  4. Replace bare `type: null` in inline query parameter schemas with
     `type: string`.  OpenAPI allows a schema to omit `type`, but oapi-codegen
     maps that to a bare `interface{}` in the generated params struct.  When
     such a field is nil, `runtime.StyleParamWithOptions` panics.  These params
     are plain strings in practice, so setting `type: string` makes
     oapi-codegen emit a concrete `string` (or a named string enum), which is
     nil-safe.  See docs/CODEGEN.md defect 4.

Usage:
    python3 scripts/patch_spec.py specs/ibkr_spec.json > specs/ibkr_patched.json

See docs/CODEGEN.md for the measured results.
"""

import collections
import json
import re
import sys


def go_name(value: str) -> str:
    """Approximate oapi-codegen's PascalCase identifier normalization."""
    parts = re.split(r"[^A-Za-z0-9]+", value)
    return "".join(p[:1].upper() + p[1:] for p in parts if p)


def patch(spec: dict) -> collections.Counter:
    comp = spec.get("components", {})
    compparams = comp.get("parameters", {})
    report = collections.Counter()

    def resolve(param: dict) -> dict:
        if "$ref" in param:
            return dict(compparams.get(param["$ref"].split("/")[-1], {}))
        return dict(param)

    # 1. Reconcile path parameters with path templates.
    for path, item in spec.get("paths", {}).items():
        placeholders = re.findall(r"\{([^}]+)\}", path)
        for op in item.values():
            if not isinstance(op, dict):
                continue
            params = op.get("parameters", [])
            nonpath = [p for p in params if resolve(p).get("in") != "path"]
            pool = [resolve(p) for p in params if resolve(p).get("in") == "path"]
            rebuilt = []
            for name in placeholders:
                chosen = None
                for cand in pool:
                    if cand.get("name") == name:
                        chosen = cand
                        pool.remove(cand)
                        break
                if chosen is None and pool:
                    chosen = pool.pop(0)
                    report["param_renamed"] += 1
                if chosen is None:
                    chosen = {"name": name, "in": "path", "required": True,
                              "schema": {"type": "string"}}
                    report["param_added"] += 1
                chosen = dict(chosen)
                chosen["name"] = name
                chosen.pop("$ref", None)
                rebuilt.append(chosen)
            report["param_dropped"] += len(pool)
            if "parameters" in op or rebuilt:
                op["parameters"] = nonpath + rebuilt

    # 2. De-duplicate operationIds.
    counts = collections.Counter()
    for item in spec.get("paths", {}).values():
        for op in item.values():
            if isinstance(op, dict) and op.get("operationId"):
                counts[op["operationId"]] += 1
    seen = collections.Counter()
    for item in spec.get("paths", {}).values():
        for op in item.values():
            if not isinstance(op, dict) or not op.get("operationId"):
                continue
            oid = op["operationId"]
            if counts[oid] > 1:
                seen[oid] += 1
                if seen[oid] > 1:
                    op["operationId"] = f"{oid}_{seen[oid]}"
                    report["opid_renamed"] += 1

    # 3. Resolve Go type-name collisions.
    schemas = comp.get("schemas", {})
    groups = collections.defaultdict(list)
    for key in schemas:
        groups[go_name(key)].append(key)
    for base, keys in groups.items():
        if len(keys) < 2:
            continue
        for i, key in enumerate(keys[1:], start=2):
            schemas[key]["x-go-name"] = f"{base}{i}"
            report["typename_renamed"] += 1

    # 4. Replace bare `type: null` in inline query parameter schemas.
    # These become `interface{}` in Go, which panics in StyleParamWithOptions when
    # nil.  The correct type for these freeform string-keyed params is `string`
    # (they are plain string values in practice).  Setting `type: string` makes
    # oapi-codegen emit `string` in Go, which is nil-safe.
    for path, item in spec.get("paths", {}).items():
        for op in item.values():
            if not isinstance(op, dict):
                continue
            for p in op.get("parameters", []):
                if not isinstance(p, dict):
                    continue
                if p.get("in") != "query":
                    continue
                if "$ref" in p:
                    continue
                schema = p.get("schema")
                if not isinstance(schema, dict):
                    continue
                if schema.get("type") is None:
                    schema["type"] = "string"
                    report["null_type_patched"] += 1

    return report


def main() -> None:
    src = sys.argv[1] if len(sys.argv) > 1 else "-"
    with open(src if src != "-" else 0) as fh:
        spec = json.load(fh)

    report = patch(spec)

    info = spec.get("info", {})
    print(
        f"Loaded {info.get('title', '?')} v{info.get('version', '?')}; "
        f"patched: " + ", ".join(f"{k}={v}" for k, v in sorted(report.items())),
        file=sys.stderr,
    )
    json.dump(spec, sys.stdout, ensure_ascii=False)


if __name__ == "__main__":
    main()
