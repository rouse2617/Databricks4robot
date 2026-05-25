#!/usr/bin/env bash
# Detect stale openspec/changes/ directories whose branches have been merged
# but the change directory was never archived.
#
# Usage:
#   bash scripts/agent-harness/stale-changes.sh          # list all stale dirs
#   bash scripts/agent-harness/stale-changes.sh --pr     # PR-friendly output
#   bash scripts/agent-harness/stale-changes.sh --json   # machine-readable
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT}"

CHANGES_DIR="openspec/changes"
MODE="${1:---list}"

STALE=()
REASON=()

# Resolve merged branches (local + remote) once
MERGED_BRANCHES=$( {
  git branch --merged origin/dev 2>/dev/null || true
  git branch -r --merged origin/dev 2>/dev/null || true
} | sed 's/^[* ]*//; s|^origin/||' | sort -u)

for d in "${CHANGES_DIR}"/CYB-*/; do
  [[ -d "$d" ]] || continue
  dirname="$(basename "$d")"
  cyb_id=$(echo "$dirname" | grep -oE 'CYB-[0-9]+' || true)
  [[ -n "$cyb_id" ]] || continue

  stale=0
  why=""

  # Check if any merged branch name contains this CYB id
  if echo "$MERGED_BRANCHES" | grep -qi "$cyb_id" 2>/dev/null; then
    stale=1
    why="branch merged to dev"
  fi

  # Also check: does the directory have proposal.md? If not, likely stale.
  if [[ ! -f "${d}proposal.md" ]]; then
    stale=1
    why="${why:+$why; }missing proposal.md"
  fi

  if [[ $stale -eq 1 ]]; then
    STALE+=("$dirname")
    REASON+=("$why")
  fi
done

if [[ ${#STALE[@]} -eq 0 ]]; then
  [[ "$MODE" != "--json" ]] && echo "stale-changes: 0 stale change directories found."
  exit 0
fi

plural="ies"
[[ ${#STALE[@]} -eq 1 ]] && plural="y"

case "$MODE" in
  --json)
    echo "["
    for i in "${!STALE[@]}"; do
      comma=","
      [[ $i -eq $((${#STALE[@]} - 1)) ]] && comma=""
      echo "  {\"dir\": \"${STALE[$i]}\", \"reason\": \"${REASON[$i]}\"}$comma"
    done
    echo "]"
    ;;
  --pr)
    echo "### Stale OpenSpec change directories (${#STALE[@]})"
    echo ""
    echo "These change dirs have merged branches but were never archived:"
    echo ""
    for i in "${!STALE[@]}"; do
      echo "- [ ] \`${CHANGES_DIR}/${STALE[$i]}/\` — ${REASON[$i]}"
    done
    echo ""
    echo "**Action:** Archive deltas into \`openspec/specs/\`, then delete or move to \`archive/\`."
    ;;
  *)
    echo "stale-changes: ${#STALE[@]} stale change director${plural} found:"
    for i in "${!STALE[@]}"; do
      echo "  ${CHANGES_DIR}/${STALE[$i]}/ — ${REASON[$i]}"
    done
    echo ""
    echo "Run with --pr for PR checklist format, or --json for machine output."
    ;;
esac

exit 0
