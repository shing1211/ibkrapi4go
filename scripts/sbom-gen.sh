#!/usr/bin/env bash
set -euo pipefail

# Check for syft
if command -v syft &> /dev/null; then
  syft . -o cyclonedx-json=sbom.cdx.json
  echo "CycloneDX SBOM written to sbom.cdx.json"
else
  echo "syft not installed. Install from https://github.com/anchore/syft"
  exit 1
fi
