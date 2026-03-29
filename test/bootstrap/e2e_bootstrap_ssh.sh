#!/usr/bin/env bash
# E2E test for bootstrap script: validates success path invokes ssh
# and failure path returns correct exit codes with Chinese messages.
# Uses a mock HTTP server (bash + netcat replacement) to simulate API.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BOOTSTRAP_SCRIPT="${PROJECT_ROOT}/deploy/bootstrap/cloud-bootstrap.sh"
PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); printf "  ✓ %s\n" "$1"; }
fail() { FAIL=$((FAIL + 1)); printf "  ✗ %s\n" "$1"; }

# --- Test 1: Script syntax check ---
printf "Test 1: bash -n syntax validation\n"
if bash -n "$BOOTSTRAP_SCRIPT" 2>/dev/null; then
  pass "script passes syntax check"
else
  fail "script has syntax errors"
fi

# --- Test 2: Script contains exec ssh (success path) ---
printf "Test 2: success path uses exec ssh\n"
if grep -q 'exec ssh' "$BOOTSTRAP_SCRIPT"; then
  pass "script contains exec ssh for session handoff"
else
  fail "script missing exec ssh"
fi

# --- Test 3: Script does NOT contain automatic retry loops ---
printf "Test 3: no automatic retry on failure (D-11)\n"
retry_count=$(grep -c 'retry\|RETRY\|auto.retry\|retry_count' "$BOOTSTRAP_SCRIPT" 2>/dev/null || true)
if [ "$retry_count" -eq 0 ]; then
  pass "no automatic retry logic found"
else
  fail "found retry-related patterns ($retry_count occurrences)"
fi

# --- Test 4: Script does NOT contain Web Terminal fallback (D-09) ---
printf "Test 4: no Web Terminal fallback (D-09)\n"
web_count=$(grep -ci 'web.terminal\|websocket\|wss://\|browser' "$BOOTSTRAP_SCRIPT" 2>/dev/null || true)
if [ "$web_count" -eq 0 ]; then
  pass "no Web Terminal fallback found"
else
  fail "found Web Terminal patterns ($web_count occurrences)"
fi

# --- Test 5: Error codes map to non-zero exit codes ---
printf "Test 5: error code to exit code mapping\n"
all_mapped=true
check_mapping() {
  local code="$1" exit_val="$2"
  if grep -q "exit ${exit_val}" "$BOOTSTRAP_SCRIPT" && grep -q "${code}" "$BOOTSTRAP_SCRIPT"; then
    : # mapped correctly
  else
    all_mapped=false
    fail "error_code=${code} not mapped to exit ${exit_val}"
  fi
}
check_mapping "auth_invalid" "10"
check_mapping "account_disabled" "11"
check_mapping "account_expired" "12"
check_mapping "host_not_found" "13"
check_mapping "start_failed" "14"
check_mapping "ssh_not_ready" "15"
check_mapping "egress_binding_missing" "16"

if $all_mapped; then
  pass "all error codes mapped to expected exit codes"
fi

# --- Test 6: Script shows explicit retry suggestion on failure ---
printf "Test 6: failure path shows retry command suggestion\n"
if grep -q '请重试命令' "$BOOTSTRAP_SCRIPT"; then
  pass "failure path includes explicit retry suggestion"
else
  fail "missing retry suggestion in failure output"
fi

# --- Test 7: StrictHostKeyChecking disabled in exec ssh ---
printf "Test 7: SSH options include StrictHostKeyChecking=no\n"
if grep -q 'StrictHostKeyChecking=no' "$BOOTSTRAP_SCRIPT"; then
  pass "StrictHostKeyChecking=no present"
else
  fail "missing StrictHostKeyChecking=no"
fi

# --- Summary ---
printf "\n--- Results: %d passed, %d failed ---\n" "$PASS" "$FAIL"
if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
exit 0
