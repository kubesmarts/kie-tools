#!/usr/bin/env bash
#
# export_vars.sh
#
# This script merges two sources of environment variables:
#   1) `pnpm build-env --print-dotenv` (run from packages/redhat-env)
#   2) A local `.env` file (located alongside this script)
#
# Behavior:
#  • If run in GitHub Actions (i.e. if $GITHUB_ENV is defined and writable), it
#    appends every merged KEY=VALUE line directly into $GITHUB_ENV—so all later
#    steps in the same job will see those variables automatically.
#
#  • If run locally (no $GITHUB_ENV), it writes all merged KEY=VALUE lines to
#    stdout.  You can then do one of:
#       eval "$(bash .github/supporting-files/ci/osl/export_vars.sh)"
#    or
#       source <(bash .github/supporting-files/ci/osl/export_vars.sh)
#    to export those variables into your current shell session.
#
# Requirements:
#  • This script must live in:
#       .github/supporting-files/ci/osl/export_vars.sh
#  • A `.env` file must sit in the same directory as this script.
#  • `pnpm build-env --print-dotenv` must be run from:
#       packages/redhat-env   (relative to repo root)
#
# Usage (locally):
#   cd <repo-root>
#   # Pipe the output into eval so your interactive shell gets the variables:
#   eval "$(bash .github/supporting-files/ci/osl/export_vars.sh)"
#
#   # or, equivalently:
#   source <(bash .github/supporting-files/ci/osl/export_vars.sh)
#
#   # Now test:
#   echo "$KIE_TOOLS_BUILD__runTests"
#   echo "$SONATAFLOW_OPERATOR__kogitoDataIndexPostgresqlImage"
#
# Usage (in GitHub Actions):
#   steps:
#     - uses: actions/checkout@v3
#     - name: Install pnpm (if needed)
#       run: |
#         curl -fsSL https://get.pnpm.io/install.sh | sh -
#         pnpm install
#
#     - name: Merge & export env (pnpm + .env)
#       run: bash ./.github/supporting-files/ci/osl/export_vars.sh
#
#     - name: Debug some vars
#       run: |
#         echo "runTests = $KIE_TOOLS_BUILD__runTests"
#         echo "dataIndexImage = $SONATAFLOW_OPERATOR__kogitoDataIndexPostgresqlImage"
#
#     # All subsequent steps will automatically see the variables defined above.
#

set -euo pipefail


SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../../" && pwd)"
PNPM_DIR="$REPO_ROOT/packages/redhat-env"
ENV_FILE="$SCRIPT_DIR/.env"


MERGED_FILE="$(mktemp /tmp/merged-env.XXXXXX)"
cleanup() {
  rm -f "$MERGED_FILE"
}
trap cleanup EXIT


if [[ ! -d "$PNPM_DIR" ]]; then
  echo "ERROR: Expected pnpm directory not found: $PNPM_DIR" >&2
  exit 1
fi

(
  cd "$PNPM_DIR"
  pnpm build-env --print-dotenv > "$MERGED_FILE"
)


if [[ -f "$ENV_FILE" ]]; then
  cat "$ENV_FILE" >> "$MERGED_FILE"
fi


if [[ -n "${GITHUB_ENV:-}" && -w "$GITHUB_ENV" ]]; then
  #
  # Running inside GitHub Actions → append to $GITHUB_ENV
  #
  cat "$MERGED_FILE" >> "$GITHUB_ENV"
else
  #
  # Running locally → export into this shell
  #
  cat "$MERGED_FILE"
fi

# (The trap on EXIT will remove $MERGED_FILE automatically)
