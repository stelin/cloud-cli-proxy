#!/usr/bin/env bash
# bump-version.sh — Sync version.txt with GSD milestone version
# Called before git tag during /gsd:complete-milestone
#
# Usage: ./scripts/bump-version.sh 1.2.0
#        ./scripts/bump-version.sh v1.2.0  (v prefix stripped automatically)

set -euo pipefail

VERSION="${1:?Usage: bump-version.sh <version>}"
VERSION="${VERSION#v}"  # strip leading v

echo "$VERSION" > version.txt
git add version.txt
git commit -m "chore: bump version to ${VERSION}" --allow-empty 2>/dev/null || true

echo "version.txt updated to ${VERSION}"
