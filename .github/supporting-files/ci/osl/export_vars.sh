#!/usr/bin/env bash
#
# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements.  See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership.  The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License.  You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.
#

#
# export_vars.sh
#
# Merges:
#  1) `pnpm build-env --print-dotenv` (from packages/redhat-env)
#  2) a local `.env` (alongside this script)
#
# • In GitHub Actions ($GITHUB_ENV set): appends the merged KEY=VALUE
#   lines into $GITHUB_ENV so later steps see them.
# • Locally: writes merged KEY=VALUE lines to stdout, so you can:
#     eval "$(bash path/to/export_vars.sh)"
#   or
#     source <(bash path/to/export_vars.sh)
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../../" && pwd)"
PNPM_DIR="$REPO_ROOT/packages/redhat-env"
ENV_FILE="$SCRIPT_DIR/.env"

MERGED="$(mktemp)"
trap 'rm -f "$MERGED"' EXIT

if [[ ! -d "$PNPM_DIR" ]]; then
  echo "ERROR: cannot find $PNPM_DIR" >&2
  exit 1
fi

(
  cd "$PNPM_DIR"
  pnpm build-env --print-dotenv > "$MERGED"
)

set -o allexport
# shellcheck disable=SC1090
source "$MERGED"
set +o allexport

if [[ -z "${KIE_TOOLS_BUILD__streamName:-}" ]]; then
  echo "ERROR: pnpm build-env did not set KIE_TOOLS_BUILD__streamName" >&2
  exit 1
fi

if [[ -f "$ENV_FILE" ]]; then
  set -o allexport
  source "$ENV_FILE"
  set +o allexport

  while IFS= read -r line || [[ -n "$line" ]]; do
    # skip empty or comment lines
    case "$line" in
      ''|\#*) continue ;;
    esac

    # expand ${VAR} now that pnpm vars are loaded
    set +u
    expanded_line="$(eval echo "\"$line\"" )"
    set -u

    printf '%s\n' "$expanded_line" >> "$MERGED"
  done < "$ENV_FILE"
fi

if [[ -n "${GITHUB_ENV:-}" ]]; then
  # GitHub Actions: append into $GITHUB_ENV removing empty lines
  grep -v -E '^\s*$' "$MERGED" >> "$GITHUB_ENV"
else
  # Local: print to stdout
  cat "$MERGED"
fi
