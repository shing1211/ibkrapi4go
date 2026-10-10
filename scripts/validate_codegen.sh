#!/usr/bin/env bash
# Copyright 2026 shing1211
# SPDX-License-Identifier: Apache-2.0
#
# validate_codegen.sh — verify that committed generated code matches a fresh
# generation from the spec. Exits non-zero on drift.
#
# Usage:
#   ./scripts/validate_codegen.sh [spec_file]
#
# Applies the same deterministic SPDX header as scripts/codegen.sh so the diff
# is meaningful.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
SPEC_FILE="${1:-$ROOT_DIR/specs/ibkr_spec.json}"
PATCHED_FILE="$ROOT_DIR/specs/ibkr_patched.json"
CONFIG_FILE="$ROOT_DIR/oapi-codegen.yaml"
COMMITTED="$ROOT_DIR/client/client.gen.go"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

if ! command -v oapi-codegen >/dev/null 2>&1; then
    echo "error: oapi-codegen not found; run 'make tools'" >&2
    exit 1
fi

if [ ! -f "$SPEC_FILE" ]; then
    echo "fetching spec..."
    mkdir -p "$(dirname "$SPEC_FILE")"
    curl -fsSL "https://api.ibkr.com/gw/api/v3/api-docs" -o "$SPEC_FILE"
fi

python3 "$SCRIPT_DIR/patch_spec.py" "$SPEC_FILE" > "$PATCHED_FILE"

# Generate into a temp dir using a temp config that redirects output.
# oapi-codegen is a native binary, so paths handed to it must be in native
# form. On Windows under Git Bash, mktemp yields an MSYS path such as
# /tmp/tmp.XXXX, which the native binary cannot resolve. cygpath -m emits a
# Windows path with forward slashes, which oapi-codegen accepts and which does
# not need escaping inside the YAML config.
win_path() {
  if command -v cygpath >/dev/null 2>&1; then
    cygpath -m "$1"
  else
    printf '%s' "$1"
  fi
}
RAW_GEN="$TMP_DIR/raw.gen.go"
sed "s#^output:.*#output: $(win_path "$RAW_GEN")#" "$CONFIG_FILE" > "$TMP_DIR/oapi-codegen.yaml"
oapi-codegen -config "$(win_path "$TMP_DIR/oapi-codegen.yaml")" "$(win_path "$PATCHED_FILE")"

{
    printf '// Copyright 2026 shing1211\n'
    printf '// SPDX-License-Identifier: Apache-2.0\n'
    cat "$TMP_DIR/raw.gen.go"
} > "$TMP_DIR/client.gen.go"

if ! diff -q "$TMP_DIR/client.gen.go" "$COMMITTED" >/dev/null 2>&1; then
    echo "error: generated code differs from committed client/client.gen.go" >&2
    diff -u "$COMMITTED" "$TMP_DIR/client.gen.go" | head -80 >&2 || true
    exit 1
fi

echo "codegen OK: committed client matches a fresh generation"
