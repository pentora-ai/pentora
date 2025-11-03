#!/usr/bin/env bash
set -euo pipefail

echo "Checking plugin commands for PrintError regression..."

# Directory containing plugin commands
PLUGIN_CMDS_DIR="cmd/pentora/commands/plugin"

# If directory doesn't exist, nothing to check (non-fatal for other workflows)
if [[ ! -d "$PLUGIN_CMDS_DIR" ]]; then
  echo "(info) Plugin commands directory not found: $PLUGIN_CMDS_DIR"
  echo "✅ No PrintError regression found"
  exit 0
fi

# Find forbidden pattern: return formatter.PrintError( in non-test, non-comment lines
# - Exclude *_test.go
# - Exclude lines that are comments-only
# - Grep returns non-zero when no matches found

# Use ripgrep if available for robustness; fallback to grep
if command -v rg >/dev/null 2>&1; then
  matches=$(rg -n "return\s+formatter\.PrintError\s*\(" "$PLUGIN_CMDS_DIR" -g '!*_test.go' -g '!**/*.md' || true)
else
  # Escape the parenthesis properly for basic grep -E
  matches=$(grep -RInE "return[[:space:]]+formatter\\.PrintError[[:space:]]*\(" "$PLUGIN_CMDS_DIR" \
    | grep -v "_test.go" \
    | grep -vE "^[[:space:]]*//" || true)
fi

if [[ -n "$matches" ]]; then
  echo "❌ ERROR: Found PrintError usage in plugin commands (main error paths)"
  echo ""
  echo "$matches"
  echo ""
  echo "Plugin commands should use PrintTotalFailureSummary for main error paths:"
  echo "  return formatter.PrintTotalFailureSummary(operation, err, plugin.ErrorCode(err))"
  exit 1
fi

echo "✅ No PrintError regression found"
exit 0
