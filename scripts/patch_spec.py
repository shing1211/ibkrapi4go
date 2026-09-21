#!/usr/bin/env python3
# Copyright 2026 shing1211
# SPDX-License-Identifier: Apache-2.0
"""
patch_spec.py — fix known defects in the IBKR OpenAPI spec before codegen.

The published spec (v2.39.0/v2.40.0) does not generate cleanly with oapi-codegen.
This script applies seven deterministic, idempotent fixes:

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

   5. Replace `type: float` with `type: integer` for ID-named fields.
       oapi-codegen maps `type: float` → Go `float32`.  ConIDs can exceed 2^24
       (16,777,216) causing silent precision loss in float32.  The affected
       fields are: conid, clientInstructionId, instructionId, instructionSetId,
       ibReferenceId.

    6. Rename `twsInvestDivestResponse` schema via `x-go-name` to avoid collision
        with the auto-generated HTTP response wrapper type of the same name in
        oapi-codegen v2.8.  v2.40.0 introduced `/fa/model/tws-invest-divest`
        which returns this schema; oapi-codegen generates a wrapper struct with
        the same name, producing a "redeclared" compile error.  Renaming the
        schema to `TwsInvestDivestResponseData` breaks the collision.

    7. Replace `type: number` with `type: string` for money-amount fields.
        ADR 0008 requires money quantities (SMA, Cash, Balance, Equity, Margin,
        etc.) to be `string`/`json.Number`, never `float64`.  This prevents
        precision loss on large values.  Rate/percentage fields (Weight,
        ExchangeRate, OwnershipPercentage) are left unchanged.

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

    # 5. Replace `type: float` with `type: integer` for ID-named fields.
    # ConIDs exceed 2^24 causing precision loss in float32.  The fields are:
    # conid, clientInstructionId, instructionId, instructionSetId, ibReferenceId.
    ID_NAMES = frozenset([
        "conid",
        "clientInstructionId",
        "instructionId",
        "instructionSetId",
        "ibReferenceId",
    ])

    # 7. Replace `type: number` with `type: string` for money-amount fields.
    # ADR 0008: money quantities must be string/json.Number, never float64.
    MONEY_FIELDS = frozenset([
        "sma",
        "accruedinterest",
        "availablefunds",
        "balance",
        "buyingpower",
        "equitywithloanvalue",
        "excessliquidity",
        "initialmargin",
        "maintenancemargin",
        "netliquidationvalue",
        "regtloan",
        "regtmargin",
        "securitiesgvp",
        "totalcashvalue",
        "settledcash",
        "nav",
        "payout",
    ])

    def patch_schema(schema: dict) -> None:
        if not isinstance(schema, dict):
            return
        if "$ref" in schema:
            ref = schema["$ref"].split("/")[-1]
            if ref in schemas:
                patch_schema(schemas[ref])
            return
        props = schema.get("properties")
        if isinstance(props, dict):
            for name, prop in props.items():
                if name in ID_NAMES and prop.get("type") in ("float", "number"):
                    prop["type"] = "integer"
                    report["float_id_patched"] += 1
        for key in ("allOf", "anyOf", "oneOf"):
            for sub in schema.get(key, []):
                patch_schema(sub)
        if schema.get("type") == "array":
            patch_schema(schema.get("items"))

    for schema in schemas.values():
        patch_schema(schema)

    # 6. Rename twsInvestDivestResponse to avoid collision with the
    #    oapi-codegen HTTP response wrapper type of the same name.
    if "twsInvestDivestResponse" in schemas:
        schemas["twsInvestDivestResponse"]["x-go-name"] = "TwsInvestDivestResponseData"
        report["tws_rename"] = 1

    # 7. Replace money-amount number fields with type: string.
    def patch_money_schema(schema: dict) -> None:
        if not isinstance(schema, dict):
            return
        if "$ref" in schema:
            ref = schema["$ref"].split("/")[-1]
            if ref in schemas:
                patch_money_schema(schemas[ref])
            return
        props = schema.get("properties")
        if isinstance(props, dict):
            for name, prop in props.items():
                if name.lower() in MONEY_FIELDS and prop.get("type") == "number":
                    prop["type"] = "string"
                    report["money_field_patched"] += 1
        for key in ("allOf", "anyOf", "oneOf"):
            for sub in schema.get(key, []):
                patch_money_schema(sub)
        if schema.get("type") == "array":
            patch_money_schema(schema.get("items"))

    for schema in schemas.values():
        patch_money_schema(schema)

    return report


def main() -> None:
    src = sys.argv[1] if len(sys.argv) > 1 else "-"
    with open(src if src != "-" else 0, encoding="utf-8") as fh:
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
