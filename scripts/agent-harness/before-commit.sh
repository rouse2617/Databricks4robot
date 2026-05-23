#!/usr/bin/env bash
# Hard guardrails before git commit. Does not replace .githooks/commit-msg.
# Usage: scripts/agent-harness/before-commit.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT}"

STAGED=()
while IFS= read -r line; do
  [[ -n "${line}" ]] && STAGED+=("${line}")
done < <(git diff --name-only --cached 2>/dev/null || true)

SOURCE="staged"
FILES=()
if ((${#STAGED[@]} > 0)); then
  FILES=("${STAGED[@]}")
fi
if [[ ${#FILES[@]} -eq 0 ]]; then
  SOURCE="unstaged+untracked (no staged files)"
  while IFS= read -r line; do
    [[ -n "${line}" ]] && FILES+=("${line}")
  done < <( {
    git diff --name-only 2>/dev/null || true
    git ls-files --others --exclude-standard 2>/dev/null || true
  } | awk 'NF && !seen[$0]++')
fi

if [[ ${#FILES[@]} -eq 0 ]]; then
  exit 0
fi

fail() {
  echo "agent-harness (before-commit): ERROR: $*" >&2
  exit 1
}

warn() {
  echo "agent-harness (before-commit): WARNING: $*" >&2
}

echo "agent-harness (before-commit): checking ${#FILES[@]} file(s) from ${SOURCE}" >&2

has_file() {
  local needle="$1"
  shift
  local f
  for f in "$@"; do
    [[ "${f}" == "${needle}" ]] && return 0
  done
  return 1
}

has_glob() {
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

for f in "${FILES[@]}"; do
  case "${f}" in
    .env|.env.*|*/.env|*/.env.*)
      fail "refusing commit: credential file '${f}' must not be committed."
      ;;
  esac
done

if has_glob 'backend/migrations/*' "${FILES[@]}"; then
  warn "migrations/ changed — off-limits without explicit approval; apply-migration-dev.sh before backend deploy if approved."
fi

API_HANDLER=0
if has_glob 'backend/routes/*' "${FILES[@]}" || has_glob 'backend/internal/handlers/*' "${FILES[@]}"; then
  API_HANDLER=1
fi

if [[ ${API_HANDLER} -eq 1 ]]; then
  HAS_OPENAPI=0
  HAS_GUIDE=0
  HAS_SMOKE_OR_SPEC=0

  has_file 'api/openapi.yaml' "${FILES[@]}" && HAS_OPENAPI=1
  has_file 'docs/review/api-guide.md' "${FILES[@]}" && HAS_GUIDE=1

  for f in "${FILES[@]}"; do
    case "${f}" in
      scripts/*smoke*.sh|scripts/api-guide-smoke.sh) HAS_SMOKE_OR_SPEC=1 ;;
      openspec/changes/*/specs/*) HAS_SMOKE_OR_SPEC=1 ;;
    esac
  done

  if [[ ${HAS_OPENAPI} -eq 0 || ${HAS_GUIDE} -eq 0 || ${HAS_SMOKE_OR_SPEC} -eq 0 ]]; then
    fail "handler/route changes require minimum API contract artifacts in the same commit: api/openapi.yaml, docs/review/api-guide.md, and (smoke script or openspec/changes/*/specs/*). Missing: openapi=${HAS_OPENAPI} api-guide=${HAS_GUIDE} smoke-or-spec=${HAS_SMOKE_OR_SPEC}"
  fi
fi

if has_glob 'Frontend/*' "${FILES[@]}"; then
  warn "Frontend/ in commit — deploy-before-commit and Chrome DevTools MCP verification required before push (see deploy-before-commit.md)."
fi

exit 0
