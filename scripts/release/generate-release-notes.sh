#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 2 ]; then
  echo "Usage: $0 <version-without-v> <git-range> [prev-tag]" >&2
  exit 1
fi

VERSION="$1"
RANGE="$2"
PREV_TAG="${3:-}"
REPO="${GITHUB_REPOSITORY:-}"

has_content=0

print_section() {
  local title="$1"
  shift

  local notes
  notes="$(git log "$RANGE" --no-merges --pretty=format:'- %s (%h)' -- "$@" || true)"
  if [ -n "$notes" ]; then
    has_content=1
    echo "### $title"
    echo "$notes"
    echo
  fi
}

echo "## What's Changed"
echo

print_section "Backend (Go / API)" cmd internal
print_section "Frontend (Admin Web)" web/admin
print_section "Runtime & Deployment" deploy docker-compose.yml docker-compose.build.yaml .github/workflows
print_section "Docs" docs README.md README.en.md

if [ "$has_content" -eq 0 ]; then
  git log "$RANGE" --no-merges --pretty=format:'- %s (%h)' || true
  echo
fi

if [ -n "$REPO" ]; then
  if [ -n "$PREV_TAG" ]; then
    echo "**Full Changelog:** https://github.com/${REPO}/compare/${PREV_TAG}...v${VERSION}"
  else
    echo "**Full Changelog:** https://github.com/${REPO}/releases/tag/v${VERSION}"
  fi
fi
