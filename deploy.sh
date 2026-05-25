#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLAUDE_DIR="${HOME}/.claude"
MARKETPLACE_NAME="plan-workflow-local"
PLUGIN_NAME="plan-workflow"
MARKETPLACE_DIR="${CLAUDE_DIR}/plugins/marketplaces/${MARKETPLACE_NAME}"
PLUGIN_DIR="${MARKETPLACE_DIR}/${PLUGIN_NAME}"
SETTINGS="${CLAUDE_DIR}/settings.json"

echo "Deploying ${PLUGIN_NAME}..."

# Install directory structure
mkdir -p "${MARKETPLACE_DIR}/.claude-plugin"
mkdir -p "${PLUGIN_DIR}"

# Copy plugin files, excluding dev-only artifacts
rsync -a --delete \
  --exclude='.git/' \
  --exclude='.gitignore' \
  --exclude='deploy.sh' \
  --exclude='.claude/' \
  --exclude='.claude-plugin/marketplace.json' \
  "${SCRIPT_DIR}/" "${PLUGIN_DIR}/"

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
