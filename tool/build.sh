#!/usr/bin/env bash
set -euo pipefail

# Compiles the marvin CLI from Go source into a target binary path, skipping
# the build when the binary already exists and is newer than every file in
# the source tree. Shared by deploy.sh (local-directory install, where the
# caller has already hard-failed if Go is missing) and the plugin's
# SessionStart hook (marketplace install, where tool/ ships alongside the
# rest of the plugin and a missing Go SDK should degrade quietly instead of
# blocking session start).
#
# Usage: build.sh <tool_source_dir> <output_binary_path>

TOOL_DIR="${1:?usage: build.sh <tool_source_dir> <output_binary_path>}"
OUT="${2:?usage: build.sh <tool_source_dir> <output_binary_path>}"

if [ ! -d "$TOOL_DIR" ]; then
  if [ ! -f "$OUT" ]; then
    echo "marvin: no Go source at ${TOOL_DIR} and no existing binary at ${OUT}; marvin is unavailable." >&2
  fi
  exit 0
fi

if [ -f "$OUT" ] && [ -z "$(find "$TOOL_DIR" -type f -newer "$OUT")" ]; then
  exit 0
fi

if ! command -v go &>/dev/null; then
  echo "marvin: Go SDK not found in PATH; skipping build. Install Go from https://go.dev/dl/ to enable the marvin CLI." >&2
  exit 0
fi

mkdir -p "$(dirname "$OUT")"
echo "  Building marvin..." >&2
(cd "$TOOL_DIR" && go build -o "$OUT" ./cmd/marvin)
