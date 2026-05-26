#!/usr/bin/env bash
# Run commitlint with the same packages and .commitlintrc.json as CI.
# Usage: scripts/commitlint-run.sh [--edit FILE | --from SHA --to SHA | --last ...]
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TOOLS="${ROOT}/tools/commitlint"
BIN="${TOOLS}/node_modules/.bin/commitlint"

if [[ ! -x "${BIN}" ]]; then
  if ! command -v npm >/dev/null 2>&1; then
    echo "commitlint-run: install Node.js 20+ (npm required)." >&2
    exit 1
  fi
  echo "commitlint-run: installing tools/commitlint (first run)..." >&2
  (cd "${TOOLS}" && npm ci --no-fund --no-audit 2>/dev/null) || (cd "${TOOLS}" && npm install --no-fund --no-audit)
fi

# Resolve "extends" from repo root .commitlintrc.json (packages live under tools/commitlint).
export NODE_PATH="${TOOLS}/node_modules${NODE_PATH:+:${NODE_PATH}}"
cd "${ROOT}"
exec "${BIN}" "$@"
