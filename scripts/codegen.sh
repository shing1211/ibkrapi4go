#!/usr/bin/env bash
# Copyright 2026 shing1211
# SPDX-License-Identifier: Apache-2.0
#
# codegen.sh — regenerate client/client.gen.go from the IBKR OpenAPI spec.
#
# Usage:
#   ./scripts/codegen.sh [spec_file]
#
# If spec_file is omitted, the spec is fetched to specs/ibkr_spec.json.
# The spec is patched by scripts/patch_spec.py before generation because the
# published spec does not generate cleanly. See docs/CODEGEN.md.
#
# A deterministic SPDX header is prepended to the generated file so that it
# passes `make license-check` while remaining reproducible (client/ is excluded
# from addlicense).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
SPEC_URL="https://api.ibkr.com/gw/api/v3/api-docs"
SPEC_FILE="${1:-$ROOT_DIR/specs/ibkr_spec.json}"
PATCHED_FILE="$ROOT_DIR/specs/ibkr_patched.json"
CONFIG_FILE="$ROOT_DIR/oapi-codegen.yaml"
OUTPUT="$ROOT_DIR/client/client.gen.go"

if ! command -v oapi-codegen >/dev/null 2>&1; then
    echo "error: oapi-codegen not found; run 'make tools'" >&2
    exit 1
fi

# 1. Fetch the spec if absent.
if [ ! -f "$SPEC_FILE" ]; then
    echo "fetching spec: $SPEC_URL"
    mkdir -p "$(dirname "$SPEC_FILE")"
    curl -fsSL "$SPEC_URL" -o "$SPEC_FILE"
fi
echo "spec: $SPEC_FILE ($(wc -c < "$SPEC_FILE") bytes)"

# 2. Patch known spec defects.
echo "patching spec..."
python3 "$SCRIPT_DIR/patch_spec.py" "$SPEC_FILE" > "$PATCHED_FILE"

# 3. Generate into a temp file, then prepend the deterministic header.
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
tmp_gen="$tmp_dir/client.gen.go"
sed "s#^output:.*#output: $tmp_gen#" "$CONFIG_FILE" > "$tmp_dir/oapi-codegen.yaml"

echo "generating..."
oapi-codegen -config "$tmp_dir/oapi-codegen.yaml" "$PATCHED_FILE"

mkdir -p "$(dirname "$OUTPUT")"
{
    printf '// Copyright 2026 shing1211\n'
    printf '// SPDX-License-Identifier: Apache-2.0\n'
    cat "$tmp_gen"
} > "$OUTPUT"

echo "wrote: client/client.gen.go ($(wc -l < "$OUTPUT") lines)"
