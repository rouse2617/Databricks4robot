#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# Historical docs are explicitly exempt from runtime wording checks.
EXCLUDE_GLOB='!docs/review/{outbox-test-plan.md,outbox-worker-design.md,doc-alignment-pure-cdc-checklist.md}'

FORBIDDEN_PATTERN='ES \+ outbox|不动 outbox cursor|同步链路全部基于「自写 Go Outbox Worker|不引 Debezium / Kafka'
EXCLUDED_FILES=(
  "docs/review/outbox-test-plan.md"
  "docs/review/outbox-worker-design.md"
  "docs/review/doc-alignment-pure-cdc-checklist.md"
)

if command -v rg >/dev/null 2>&1; then
  if rg -n -e "$FORBIDDEN_PATTERN" docs/review/*.md --glob "$EXCLUDE_GLOB"; then
    echo "Found non-Pure-CDC wording in docs/review (excluding historical docs)." >&2
    exit 1
  fi
else
  matches="$(grep -nE "$FORBIDDEN_PATTERN" docs/review/*.md || true)"
  for excluded in "${EXCLUDED_FILES[@]}"; do
    matches="$(printf '%s\n' "$matches" | grep -v "^${excluded}:" || true)"
  done
  if [[ -n "${matches// }" ]]; then
    printf '%s\n' "$matches"
    echo "Found non-Pure-CDC wording in docs/review (excluding historical docs)." >&2
    exit 1
  fi
fi

echo "docs/review wording check passed (Pure CDC baseline)."
