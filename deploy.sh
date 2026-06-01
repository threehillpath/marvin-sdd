#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLAUDE_DIR="${HOME}/.claude"
MARKETPLACE_NAME="plan-workflow-local"
PLUGIN_NAME="plan-workflow"
MARKETPLACE_DIR="${CLAUDE_DIR}/plugins/marketplaces/${MARKETPLACE_NAME}"
SETTINGS="${CLAUDE_DIR}/settings.json"

# PLUGIN_DIR may be overridden by the caller (e.g. in tests or CI).
# When not set, use the standard install location.
PLUGIN_DIR="${PLUGIN_DIR:-${MARKETPLACE_DIR}/${PLUGIN_NAME}}"

# ── Go SDK detection ─────────────────────────────────────────────────────────
# marvin must be compiled during install. Fail early with a clear message if
# the Go SDK is not available so the user knows exactly what to fix.
if ! command -v go &>/dev/null; then
  echo "Error: Go SDK not found in PATH." >&2
  echo "Install Go from https://go.dev/dl/ and ensure 'go' is on your PATH, then re-run deploy.sh." >&2
  exit 1
fi

echo "Deploying ${PLUGIN_NAME}..."

# Install directory structure
mkdir -p "${MARKETPLACE_DIR}/.claude-plugin"
mkdir -p "${PLUGIN_DIR}"
mkdir -p "${PLUGIN_DIR}/bin"

# Copy plugin files, excluding dev-only artifacts and the Go source tree.
# tool/ is excluded here because the binary is compiled and placed in bin/
# after rsync — keeping the Go source out of the install footprint and
# ensuring --delete cannot wipe the compiled binary.
rsync -a --delete \
  --exclude='.git/' \
  --exclude='.gitignore' \
  --exclude='deploy.sh' \
  --exclude='.claude/' \
  --exclude='.claude-plugin/marketplace.json' \
  --exclude='tool/' \
  "${SCRIPT_DIR}/" "${PLUGIN_DIR}/"

# ── Compile marvin ───────────────────────────────────────────────────────────
# Build after rsync so --delete cannot remove the binary.
echo "  Building marvin..."
(cd "${SCRIPT_DIR}/tool" && go build -o "${PLUGIN_DIR}/bin/marvin" ./cmd/marvin)

# Write the installed marketplace manifest (not sourced from the repo)
cat > "${MARKETPLACE_DIR}/.claude-plugin/marketplace.json" <<MANIFEST
{
  "name": "${MARKETPLACE_NAME}",
  "owner": { "name": "Bryan Walker" },
  "plugins": [
    {
      "name": "${PLUGIN_NAME}",
      "source": "./${PLUGIN_NAME}",
      "description": "Structured architecture-to-implementation workflow using GitHub issues and project boards."
    }
  ]
}
MANIFEST

# Ensure settings.json exists
if [ ! -f "$SETTINGS" ]; then
  echo '{}' > "$SETTINGS"
fi

# Update settings.json:
#   - Register the installed marketplace (directory source)
#   - Remove the dev live-link entry if present
#   - Enable the plugin globally
jq \
  --arg marketplace "$MARKETPLACE_NAME" \
  --arg plugin "$PLUGIN_NAME" \
  --arg dir "$MARKETPLACE_DIR" '
    .extraKnownMarketplaces //= {} |
    .extraKnownMarketplaces[$marketplace] = {
      "source": { "source": "directory", "path": $dir }
    } |
    del(.extraKnownMarketplaces["plan-workflow-marketplace"]) |
    .enabledPlugins //= {} |
    .enabledPlugins[($plugin + "@" + $marketplace)] = true
  ' "$SETTINGS" > "${SETTINGS}.tmp" && mv "${SETTINGS}.tmp" "$SETTINGS"

echo "  Installed to: ${PLUGIN_DIR}"
echo "  Updated:      ${SETTINGS}"
echo ""
echo "Restart Claude Code for changes to take effect."
