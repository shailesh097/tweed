#!/usr/bin/env bash
# One-time helper: rename the project from "tweed" to another name.
#
#   ./scripts/rename.sh tailor
#
# Run it BEFORE your first commit, check `git status`, then delete this script.
set -euo pipefail

NEW="${1:-}"
if [[ ! "$NEW" =~ ^[a-z][a-z0-9-]*$ ]]; then
  echo "usage: $0 <new-name>   (lowercase letters, digits, dashes)" >&2
  exit 1
fi
cd "$(dirname "$0")/.."

grep -rlI --exclude-dir=.git --exclude=rename.sh 'tweed' . | xargs sed -i "s/tweed/$NEW/g"
mv cmd/tweed "cmd/$NEW"
mv configs/tweed.yaml "configs/$NEW.yaml"

echo "Renamed to '$NEW'. Now: go mod tidy && make check && rm scripts/rename.sh"
