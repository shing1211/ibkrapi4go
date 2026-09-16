#!/usr/bin/env bash
# codegen.sh — Regenerate Go types and client from IBKR OpenAPI spec
# Usage: ./scripts/codegen.sh [spec_file]
#
# The official spec has bugs that prevent oapi-codegen from generating cleanly.
# This script patches the spec before codegen to work around those bugs.
#
# Known issues:
#   - /gw/api/v1/balances/query: path has 0 {param} but spec declares 1

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
SPEC_URL="https://api.ibkr.com/gw/api/v3/api-docs"
SPEC_FILE="${1:-$ROOT_DIR/specs/ibkr_spec.json}"
OUTPUT_DIR="$ROOT_DIR/client"
PATCH_SCRIPT="$SCRIPT_DIR/patch_spec.py"

# ---------------------------------------------------------------------------
# Step 1: Fetch spec if not provided or --force-fetch flag passed
# ---------------------------------------------------------------------------
if [ ! -f "$SPEC_FILE" ] || [[ "${2:-}" == "--force-fetch" ]]; then
    echo "Fetching latest IBKR API spec..."
    mkdir -p "$(dirname "$SPEC_FILE")"
    curl -s "$SPEC_URL" -o "$SPEC_FILE"
    echo "Saved to $SPEC_FILE ($(wc -c < "$SPEC_FILE") bytes)"
fi

# ---------------------------------------------------------------------------
# Step 2: Patch spec bugs
# ---------------------------------------------------------------------------
echo "Patching spec..."
python3 "$PATCH_SCRIPT" "$SPEC_FILE" > "$ROOT_DIR/specs/ibkr_patched.json"

# ---------------------------------------------------------------------------
# Step 3: Generate Go code
# ---------------------------------------------------------------------------
echo "Generating Go types and client..."
mkdir -p "$OUTPUT_DIR"

oapi-codegen \
    --package=ibkr \
    --generate=types,client \
    "$ROOT_DIR/specs/ibkr_patched.json" \
    > "$OUTPUT_DIR/client.gen.go"

# ---------------------------------------------------------------------------
# Step 4: Report
# ---------------------------------------------------------------------------
echo
echo "Generated: $OUTPUT_DIR/client.gen.go"
echo "Lines: $(wc -l < "$OUTPUT_DIR/client.gen.go")"
echo "Done."
