#!/usr/bin/env bash


# ------------------------------------------------------------------------------
# export_vars.sh - Load environment variables from pnpm and a local .env file
#
# Usage:
#   source export_vars.sh               # For local development
#   export_vars.sh $GITHUB_ENV          # In GitHub Actions (writes to env file)
#
# What it does:
# - Runs `pnpm build-env --print-dotenv` from the redhat-env package path and exports vars
# - Loads and exports additional vars from a .env file placed next to this script
# - Supports ${VAR} substitution in .env values
# - If a target file is passed (e.g., $GITHUB_ENV), all vars are written to it
# - Ensures idempotency by cleaning the file before writing
#
# Notes:
# - Use `source` to persist vars in your current shell session
# - The script assumes `pnpm` is installed and the target package path exists
# - Only use trusted .env files as values are evaluated via `eval`
# ------------------------------------------------------------------------------

set -euo pipefail

TARGET_ENV_FILE="${1:-}"
TARGET_PACKAGE_PATH="./packages/redhat-env"

# Get the directory where this script lives
SCRIPT_PATH="${(%):-%x}" # zsh-compatible
[[ -n "${BASH_SOURCE:-}" ]] && SCRIPT_PATH="${BASH_SOURCE[0]}"
SCRIPT_DIR="$(cd "$(dirname "$SCRIPT_PATH")" && pwd)"

ENV_FILE="$SCRIPT_DIR/.env"

if [[ -n "$TARGET_ENV_FILE" ]]; then
  echo "🧹 Cleaning $TARGET_ENV_FILE"
  : > "$TARGET_ENV_FILE"
fi

# Sanity check for package path
if [[ ! -d "$TARGET_PACKAGE_PATH" ]]; then
  echo "❌ Package path '$TARGET_PACKAGE_PATH' does not exist."
  exit 1
fi

# Load envs from pnpm build-env
if command -v pnpm > /dev/null; then
  while IFS= read -r line; do
    [[ "$line" =~ ^[A-Za-z_][A-Za-z0-9_]*= ]] || continue
    echo "Exporting from pnpm: $line"
    export "$line"
    [[ -n "$TARGET_ENV_FILE" ]] && echo "$line" >> "$TARGET_ENV_FILE"
  done < <(cd "$TARGET_PACKAGE_PATH" && pnpm build-env --print-dotenv)
else
  echo "❌ pnpm is not installed or not in PATH."
  exit 1
fi

# Load from .env next to this script
if [[ -f "$ENV_FILE" ]]; then
  while IFS= read -r line; do
    # Allow only lines like KEY=VALUE, skip empty/comment lines
    if [[ "$line" =~ ^[A-Za-z_][A-Za-z0-9_]*=.+ ]]; then
      echo "Exporting from .env: $line"
      eval "export $line"
      [[ -n "$TARGET_ENV_FILE" ]] && echo "$line" >> "$TARGET_ENV_FILE"
    fi
  done < "$ENV_FILE"
else
  echo "ℹ️ No .env file found at $ENV_FILE"
fi
