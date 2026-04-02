#!/usr/bin/env bash
set -euo pipefail

REAL_CLAUDE="${CLAUDE_REAL_BIN:-/usr/local/bin/claude-real}"
if [ ! -x "${REAL_CLAUDE}" ]; then
  if command -v claude-real >/dev/null 2>&1; then
    REAL_CLAUDE="$(command -v claude-real)"
  else
    echo "claude-real not found, cannot start persistent claude session." >&2
    exit 127
  fi
fi

# Non-interactive mode keeps the original behavior.
if [ "${CLAUDE_NO_TMUX:-}" = "1" ] || ! command -v tmux >/dev/null 2>&1 || [ ! -t 0 ] || [ ! -t 1 ]; then
  exec "${REAL_CLAUDE}" "$@"
fi

WORKDIR="$(pwd -P)"
HASH="$(printf '%s' "${WORKDIR}" | sha1sum | awk '{print $1}')"
SESSION="claude_${HASH:0:12}"

build_cmd() {
  local quoted
  printf -v quoted '%q ' "${REAL_CLAUDE}" "$@"
  printf '%s' "${quoted% }"
}

CMD="$(build_cmd "$@")"

if tmux has-session -t "${SESSION}" 2>/dev/null; then
  if [ -n "${TMUX:-}" ]; then
    CURRENT="$(tmux display-message -p '#S' 2>/dev/null || true)"
    if [ "${CURRENT}" = "${SESSION}" ]; then
      echo "Already attached to ${SESSION} (${WORKDIR})" >&2
      exit 0
    fi
    exec tmux switch-client -t "${SESSION}"
  fi
  exec tmux attach -t "${SESSION}"
fi

tmux new-session -d -s "${SESSION}" -c "${WORKDIR}" "${CMD}"
tmux set-option -t "${SESSION}" destroy-unattached off >/dev/null 2>&1 || true

if [ -n "${TMUX:-}" ]; then
  exec tmux switch-client -t "${SESSION}"
fi
exec tmux attach -t "${SESSION}"
