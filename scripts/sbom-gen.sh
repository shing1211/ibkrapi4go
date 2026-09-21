#!/usr/bin/env bash
set -euo pipefail

# Generate a simple SBOM from go.mod for ibkrapi4go.
# This is a lightweight alternative to CycloneDX/SPDX tooling
# that avoids adding new dependencies.

echo "=== ibkrapi4go SBOM ==="
echo "Generated: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo ""

MODULE=$(head -1 go.mod | awk '{print $2}')
GO_VERSION=$(grep '^go ' go.mod | awk '{print $2}')

echo "Module: ${MODULE}"
echo "Go version: ${GO_VERSION}"
echo ""
echo "=== Direct dependencies ==="
# Parse require block (first block, non-indirect)
awk '/^require \(/,/^\)/' go.mod | grep -v '//' | grep -v '^require' | grep -v '^)' | awk '{print $1, $2}'

echo ""
echo "=== Indirect dependencies ==="
awk '/^require \(/,/^\)/' go.mod | grep '//' | grep -v '^require' | grep -v '^)' | sed 's|// indirect||' | awk '{print $1, $2}'
