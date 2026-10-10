#!/usr/bin/env bash
# Copyright 2026 shing1211
# SPDX-License-Identifier: Apache-2.0
#
# sbom-gen.sh - write a CycloneDX software bill of materials for this repository
# using syft. Exits non-zero if syft is not installed.
#
# Usage:
#   ./scripts/sbom-gen.sh

set -euo pipefail

# Check for syft
if command -v syft &> /dev/null; then
  syft . -o cyclonedx-json=sbom.cdx.json
  echo "CycloneDX SBOM written to sbom.cdx.json"
else
  echo "syft not installed. Install from https://github.com/anchore/syft"
  exit 1
fi
