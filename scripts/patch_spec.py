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

    8. Apply the same money-field retyping to inline request/response schemas
        declared under `paths`.  Defect 7 only walks `components.schemas`, so
        operations whose bodies are declared inline keep `type: number` and
        generate as `float32`.  In v2.40.0 this affects exactly eight fields in
        three operations: `amtToInvest` (x2) under
        `/v1/api/fa/model/invest-divest` and `/v1/api/fa/model/tws-invest-divest`,
        and `auxPrice`, `cashQty`, `fxQty`, `price`, `quantity`, `trailingAmt`
        under `/v1/api/iserver/account/{modelCode}/orders`.
        The field allowlist is separate from defect 7 on purpose: the same
        names also appear on `singleOrderSubmissionRequest`, `FopInstruction`,
        `DwacInstruction`, and `ComplexAssetTransferInstruction`, which are live
        banking request schemas, and retyping those would change the wire format
        of existing transfer operations.  See docs/CODEGEN.md defect 8.

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

    # 8. Reapply money retyping to inline schemas declared under `paths`.
    #    Scoped to a separate allowlist; see the module docstring.
    INLINE_MONEY_FIELDS = frozenset([
        "amttoinvest",
        "auxprice",
        "cashqty",
        "fxqty",
        "price",
        "quantity",
        "trailingamt",
    ])

    def patch_inline_money(node) -> None:
        if isinstance(node, dict):
            props = node.get("properties")
            if isinstance(props, dict):
                for name, prop in props.items():
                    if (
                        isinstance(prop, dict)
                        and name.lower() in INLINE_MONEY_FIELDS
                        and prop.get("type") == "number"
                    ):
                        prop["type"] = "string"
                        report["inline_money_patched"] += 1
            for value in node.values():
                patch_inline_money(value)
        elif isinstance(node, list):
            for value in node:
                patch_inline_money(value)

    for path_item in spec.get("paths", {}).values():
        patch_inline_money(path_item)

    # 9. Retype money fields that the gateway sends as JSON *numbers* to
    #    `json.Number` rather than `string`.
    #
    #    Defects 7 and 8 retype `type: number` to `type: string`, which is correct
    #    for the money fields the gateway already quotes on the wire (e.g.
    #    "balance":"1000.00"). It is wrong for any field the gateway sends as a
    #    bare JSON number: Go cannot unmarshal a number into a string field, so
    #    the retyped decode fails outright rather than losing precision. Those
    #    fields need `json.Number`, which accepts the number *and* preserves the
    #    literal exactly.
    #
    #    float32 is the reason this matters. A float32 mantissa holds 24 bits, so
    #    it silently rounds any amount above 2^24 (16777216) - reachable for
    #    aggregate dividend and withholding figures on a tax voucher. json.Number
    #    keeps the gateway's own digits, per ADR 0008.
    NUMBER_MONEY_SCHEMAS = {
        "TaxVoucherDTO": frozenset([
            "fee",
            "divAmount",
            "withHeldAmount",
            "quantity",
        ]),
    }

    for schema_name, field_names in NUMBER_MONEY_SCHEMAS.items():
        schema = schemas.get(schema_name)
        props = schema.get("properties") if isinstance(schema, dict) else None
        if not isinstance(props, dict):
            continue
        for name in field_names:
            prop = props.get(name)
            if isinstance(prop, dict) and prop.get("type") == "number":
                prop["x-go-type"] = "json.Number"
                report["number_money_patched"] += 1

    # 10. Make `year` optional on the "which tax years are available" operation.
    #
    #     The spec marks TaxYearRequestParam as `required: true`, so
    #     oapi-codegen emits a non-pointer string and the wrapper has no way to
    #     leave it out: calling the operation to discover the available years
    #     sends a present-and-empty `year=`. The requirement is also incoherent
    #     on its face - an endpoint whose purpose is to list which years exist
    #     cannot require the caller to already know the year.
    #
    #     `required` is a sibling of `$ref`, so overriding it in place would be
    #     ignored. The parameter is inlined for this operation only, leaving the
    #     shared component untouched for the operations that genuinely need a
    #     year. The result is a `*string` the wrapper can omit.
    OPTIONAL_YEAR_OPS = frozenset(["listTaxDocumentsAvailable"])

    for path_item in spec.get("paths", {}).values():
        if not isinstance(path_item, dict):
            continue
        for op in path_item.values():
            if not isinstance(op, dict) or op.get("operationId") not in OPTIONAL_YEAR_OPS:
                continue
            params = op.get("parameters")
            if not isinstance(params, list):
                continue
            for idx, param in enumerate(params):
                resolved = resolve(param)
                if resolved.get("name") != "year" or resolved.get("in") != "query":
                    continue
                if resolved.get("required") is not True:
                    continue
                relaxed = dict(resolved)
                relaxed["required"] = False
                params[idx] = relaxed
                report["year_optional_patched"] += 1

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
    sys.stdout.reconfigure(encoding="utf-8")
    json.dump(spec, sys.stdout, ensure_ascii=False)


if __name__ == "__main__":
    main()
