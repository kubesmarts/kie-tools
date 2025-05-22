#!/usr/bin/env bash

# ----------------------------------------------------------------------------
# export_vars.sh - Load environment variables from pnpm and a local .env file
#
# Usage:
#   source export_vars.sh               # For local development (bash or zsh)
#   ./export_vars.sh $GITHUB_ENV        # In GitHub Actions (writes to env file)
#
# What it does:
# - Runs `pnpm build-env --print-dotenv` from the redhat-env package path and exports vars
# - Loads and exports additional vars from a .env file placed next to this script
# - Supports ${VAR} substitution in .env values
# - If a target file is passed (e.g., $GITHUB_ENV), all vars are written to it
# - Ensures idempotency by cleaning the file before writing
#
# Notes:
# - Works under both bash and zsh
# - Use `source` to persist vars in your current shell session
# - The script assumes `pnpm` is installed and the target package path exists
# - Only use trusted .env files as values are evaluated via `eval`
# ----------------------------------------------------------------------------

# Workaround for VSCode zsh preexec hook error when RPROMPT is unset
if [ -n "${ZSH_VERSION-}" ]; then
  : ${RPROMPT:=""}
fi

# Detect shell and determine script source and sourced vs executed
if [ -n "${ZSH_VERSION-}" ]; then
  # zsh
  SCRIPT_SOURCE="${(%):-%x}"
  [[ "${(%):-%x}" != "$0" ]] && IS_SOURCED=true || IS_SOURCED=false
elif [ -n "${BASH_VERSION-}" ]; then
  # bash
  SCRIPT_SOURCE="${BASH_SOURCE[0]}"
  [[ "${BASH_SOURCE[0]}" != "$0" ]] && IS_SOURCED=true || IS_SOURCED=false
else
  # fallback
  SCRIPT_SOURCE="$0"
  IS_SOURCED=false
fi

# Helper to exit or return based on context
die() {
  echo "❌ $1" >&2
  if [ "$IS_SOURCED" = true ]; then
    return 1
  else
    exit 1
  fi
}

set -euo pipefail

TARGET_ENV_FILE="${1:-}"
TARGET_PACKAGE_PATH="./packages/redhat-env"
SCRIPT_DIR="$(cd "$(dirname "$SCRIPT_SOURCE")" && pwd)"
ENV_FILE="$SCRIPT_DIR/.env"

# Clean target env file if specified
if [ -n "$TARGET_ENV_FILE" ]; then
  echo "🧹 Cleaning $TARGET_ENV_FILE"
  : > "$TARGET_ENV_FILE"
fi

# Sanity check for package path
[ ! -d "$TARGET_PACKAGE_PATH" ] && die "Package path '$TARGET_PACKAGE_PATH' does not exist."

# Ensure pnpm is available
command -v pnpm > /dev/null || die "pnpm is not installed or not in PATH."

# Load envs from pnpm build-env
while IFS= read -r line; do
  [[ "$line" =~ ^[A-Za-z_][A-Za-z0-9_]*= ]] || continue
  echo "Exporting from pnpm: $line"
  export "$line"
  [ -n "$TARGET_ENV_FILE" ] && echo "$line" >> "$TARGET_ENV_FILE"
done < <(cd "$TARGET_PACKAGE_PATH" && pnpm build-env --print-dotenv)

# Load from .env next to this script
if [ -f "$ENV_FILE" ]; then
  while IFS= read -r line; do
    # Allow only lines like KEY=VALUE, skip empty/comment lines
    if [[ "$line" =~ ^[A-Za-z_][A-Za-z0-9_]*=.+ ]]; then
      echo "Exporting from .env: $line"
      eval "export $line"
      [ -n "$TARGET_ENV_FILE" ] && echo "$line" >> "$TARGET_ENV_FILE"
    fi
  done < "$ENV_FILE"
else
  echo "ℹ️ No .env file found at $ENV_FILE"
fi
