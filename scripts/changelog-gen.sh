#!/usr/bin/env bash
set -euo pipefail

LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
if [ -z "$LAST_TAG" ]; then
    RANGE="HEAD"
else
    RANGE="${LAST_TAG}..HEAD"
fi

echo "## [Unreleased]"
echo ""
echo "### Features"
git log "$RANGE" --oneline --grep="^feat" | sed 's/^/- /'
echo ""
echo "### Fixes"
git log "$RANGE" --oneline --grep="^fix" | sed 's/^/- /'
echo ""
echo "### Documentation"
git log "$RANGE" --oneline --grep="^docs" | sed 's/^/- /'
echo ""
echo "### Internal"
git log "$RANGE" --oneline --grep="^refactor\|^perf\|^test\|^ci\|^chore" | sed 's/^/- /'
