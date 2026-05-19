#!/usr/bin/env bash
set -euo pipefail

# Validate OpenSpec change deltas with the official CLI (CI only).
# Agents still author changes as plain Markdown — no local CLI required.
#
# Usage: openspec_validate_ci.sh <branch_name> <changed_files_newline_separated>
#
# Validates:
#   - openspec/changes/* dirs touched in the PR diff
#   - openspec/changes/{CYB|DAT}-{id}-* dir implied by branch (if present)
#
# Does NOT validate openspec/specs/ baseline (may lack #### Scenario until migrated).

BRANCH_NAME="${1:-}"
CHANGED_FILES="${2:-}"
OPENSPEC_TELEMETRY="${OPENSPEC_TELEMETRY:-0}"
export OPENSPEC_TELEMETRY

NODE_VERSION="${OPENSPEC_NODE_VERSION:-20}"
OPENSPEC_PKG="${OPENSPEC_PKG:-@fission-ai/openspec@latest}"

collect_change_ids() {
  local ids=()
  local line dir id

  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    if [[ "$line" =~ ^openspec/changes/([^/]+)/ ]]; then
      dir="${BASH_REMATCH[1]}"
      [[ "$dir" == "archive" || "$dir" == "README.md" ]] && continue
      ids+=("$dir")
    fi
  done <<< "$CHANGED_FILES"

  if [[ "$BRANCH_NAME" =~ (CYB-[0-9]+|DAT-[0-9]+) ]]; then
    local issue_id="${BASH_REMATCH[1]}"
    for candidate in openspec/changes/${issue_id}-*/; do
      [[ -d "$candidate" ]] || continue
      id="$(basename "$candidate")"
      ids+=("$id")
    done
  fi

  if [[ ${#ids[@]} -eq 0 ]]; then
    return 0
  fi

  printf '%s\n' "${ids[@]}" | sort -u
}

CHANGE_IDS=()
while IFS= read -r _id; do
  [[ -n "$_id" ]] && CHANGE_IDS+=("$_id")
done < <(collect_change_ids)

if [[ ${#CHANGE_IDS[@]} -eq 0 ]]; then
  echo "OK: No openspec/changes/* dirs in PR diff; OpenSpec CLI validate skipped."
  exit 0
fi

if ! command -v node >/dev/null 2>&1; then
  echo "ERROR: node is required for openspec validate CI."
  exit 1
fi

echo "OpenSpec CLI validate (package: ${OPENSPEC_PKG})"
FAILED=0

for id in "${CHANGE_IDS[@]}"; do
  change_dir="openspec/changes/${id}"
  if [[ ! -d "$change_dir" ]]; then
    echo "WARN: Change id '$id' from diff/branch but directory missing: $change_dir"
    FAILED=1
    continue
  fi
  if [[ ! -f "${change_dir}/proposal.md" ]]; then
    echo "WARN: $change_dir missing proposal.md — skipped CLI validate"
    continue
  fi

  echo "--- validate change: $id ---"
  if ! npx --yes "${OPENSPEC_PKG}" validate "$id" --no-interactive; then
    echo "ERROR: openspec validate failed for '$id'"
    echo "Hint: each Requirement needs #### Scenario: blocks (see docs/agents/spec-writing-skill.md)"
    FAILED=1
  fi
done

if [[ "$FAILED" -ne 0 ]]; then
  exit 1
fi

echo "OK: OpenSpec CLI validated ${#CHANGE_IDS[@]} change(s)."
exit 0
