#!/usr/bin/env bash
# Advisory path-based reminders after file edits.
# Usage: scripts/agent-harness/after-edit.sh [file...]
# Exit 0 unless a clearly dangerous credential path is detected.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT}"

collect_files() {
  if [[ $# -gt 0 ]]; then
    printf '%s\n' "$@"
    return
  fi
  {
    git diff --name-only --cached 2>/dev/null || true
    git diff --name-only 2>/dev/null || true
  } | awk 'NF && !seen[$0]++'
}

matches_any() {
  local pattern="$1"
  shift
  local f
  for f in "$@"; do
    case "${f}" in
      ${pattern}) return 0 ;;
    esac
  done
  return 1
}

FILES=()
while IFS= read -r line; do
  [[ -n "${line}" ]] && FILES+=("${line}")
done < <(collect_files "$@")

if [[ ${#FILES[@]} -eq 0 ]]; then
  exit 0
fi

WARN=0
CREDENTIAL_BLOCK=0

note() {
  echo "agent-harness (after-edit): $*" >&2
  WARN=1
}

if matches_any '.env*' "${FILES[@]}" || matches_any '*/.env*' "${FILES[@]}"; then
  note "CRITICAL: .env* files changed — do not commit credentials."
  CREDENTIAL_BLOCK=1
fi

if matches_any 'backend/routes/*' "${FILES[@]}" || matches_any 'backend/internal/handlers/*' "${FILES[@]}"; then
  note "API contract sync likely required: api/openapi.yaml, docs/review/api-guide.md, smoke script, openspec spec delta; SDK/Frontend if public surface or UI calls API."
fi

if matches_any 'api/openapi.yaml' "${FILES[@]}"; then
  note "OpenAPI changed — use verification Tier L; align api-guide and smoke scripts."
fi

if matches_any 'backend/migrations/*' "${FILES[@]}"; then
  note "Migrations are off-limits without explicit approval — run scripts/apply-migration-dev.sh before backend deploy if approved."
fi

if matches_any 'Frontend/*' "${FILES[@]}"; then
  note "Frontend changed — deploy frontend dev and run Chrome DevTools MCP verification before commit/push (see deploy-verification.md)."
fi

if matches_any 'sdk/*' "${FILES[@]}"; then
  note "SDK changed — run: cd sdk && uv run ruff check src/ and pytest for touched modules."
fi

if [[ ${WARN} -eq 1 ]]; then
  note "See docs/agents/AI-RULES.md and docs/agents/HARNESS.md."
fi

if [[ ${CREDENTIAL_BLOCK} -eq 1 ]]; then
  exit 1
fi
exit 0
