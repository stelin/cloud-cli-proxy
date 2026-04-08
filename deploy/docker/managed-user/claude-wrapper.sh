#!/usr/bin/env bash
set -euo pipefail

REAL_CLAUDE="${CLAUDE_REAL_BIN:-/usr/local/bin/claude-real}"
if [ ! -x "${REAL_CLAUDE}" ]; then
  if command -v claude-real >/dev/null 2>&1; then
    REAL_CLAUDE="$(command -v claude-real)"
  else
    echo "claude-real not found, cannot start claude." >&2
    exit 127
  fi
fi

exec "${REAL_CLAUDE}" "$@"
