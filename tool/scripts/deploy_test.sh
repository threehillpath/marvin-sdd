#!/usr/bin/env bash
# deploy_test.sh — TDD entry point for Phase 3: deploy.sh Build Integration
#
# Tests:
#   1. deploy.sh compiles marvin into PLUGIN_DIR/bin/marvin; `marvin version` exits 0
#   2. deploy.sh exits non-zero with the Go-missing message when Go is masked from PATH

set -uo pipefail
# Note: -e is intentionally omitted to allow test helper functions to use arithmetic
# without triggering early exit (bash arithmetic returns non-zero on zero result).

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
DEPLOY_SH="${REPO_ROOT}/deploy.sh"

PASS=0
FAIL=0

pass() { echo "  PASS: $1"; PASS=$((PASS + 1)); }
fail() { echo "  FAIL: $1"; FAIL=$((FAIL + 1)); }

echo "=== deploy_test.sh ==="
echo ""

# ── Cleanup on exit ──────────────────────────────────────────────────────────
TMPDIR1=""
TMPDIR2=""
cleanup() {
  [ -n "${TMPDIR1}" ] && rm -rf "${TMPDIR1}"
  [ -n "${TMPDIR2}" ] && rm -rf "${TMPDIR2}"
}
trap cleanup EXIT

# ── Test 1: deploy.sh compiles marvin into PLUGIN_DIR/bin/marvin ────────────

echo "Test 1: deploy.sh produces bin/marvin and 'marvin version' exits 0"

TMPDIR1=$(mktemp -d)

if PLUGIN_DIR="${TMPDIR1}" /bin/bash "${DEPLOY_SH}" 2>&1; then
  if [ -f "${TMPDIR1}/bin/marvin" ]; then
    if [ -x "${TMPDIR1}/bin/marvin" ]; then
      OUTPUT=$("${TMPDIR1}/bin/marvin" version 2>&1)
      VERSION_EXIT=$?
      if [ "${VERSION_EXIT}" -eq 0 ] && [ -n "${OUTPUT}" ]; then
        pass "bin/marvin exists, is executable, and 'marvin version' exits 0 with output: ${OUTPUT}"
      else
        fail "bin/marvin exists but 'marvin version' failed (exit=${VERSION_EXIT}) or produced no output"
      fi
    else
      fail "bin/marvin exists but is not executable"
    fi
  else
    fail "deploy.sh succeeded but bin/marvin was not created at ${TMPDIR1}/bin/marvin"
  fi
else
  fail "deploy.sh exited non-zero (unexpectedly)"
fi

# ── Test 2: deploy.sh hard-fails with Go-missing message when Go is masked ──

echo ""
echo "Test 2: deploy.sh exits non-zero with Go-missing message when PATH masks go"

TMPDIR2=$(mktemp -d)

# Run with only /usr/bin in PATH (no go).
# Use /bin/bash explicitly because 'bash' may not be in /usr/bin on macOS.
# Capture both stdout and stderr; deploy.sh error message goes to stderr.
OUTPUT2=$(PLUGIN_DIR="${TMPDIR2}" PATH=/usr/bin /bin/bash "${DEPLOY_SH}" 2>&1) && EXIT2=0 || EXIT2=$?

if [ "${EXIT2}" -ne 0 ]; then
  if echo "${OUTPUT2}" | grep -qi "go"; then
    pass "deploy.sh exited ${EXIT2} with Go-related message: ${OUTPUT2}"
  else
    fail "deploy.sh exited non-zero but output did not mention Go SDK: ${OUTPUT2}"
  fi
else
  fail "deploy.sh should have exited non-zero when go is masked from PATH, but exited 0"
fi

# ── Results ──────────────────────────────────────────────────────────────────

echo ""
echo "=== Results: ${PASS} passed, ${FAIL} failed ==="

if [ "${FAIL}" -gt 0 ]; then
  exit 1
fi
