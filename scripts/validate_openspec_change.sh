#!/usr/bin/env bash
set -euo pipefail

# Validate that a PR has a valid OpenSpec change linked.
# Usage: validate_openspec_change.sh <branch_name> <changed_files_list>
#
# Exit 0 = pass, Exit 1 = fail

BRANCH_NAME="${1:-}"
CHANGED_FILES="${2:-}"

# Extract Linear issue id from branch name (CYB-xxx primary; DAT-xxx legacy)
CHANGE_ID=""
if [[ "$BRANCH_NAME" =~ (CYB-[0-9]+|DAT-[0-9]+) ]]; then
  CHANGE_ID="${BASH_REMATCH[1]}"
fi

if [[ -z "$CHANGE_ID" ]]; then
  echo "ERROR: Branch name '$BRANCH_NAME' does not contain a Linear issue ID (CYB-xxx or DAT-xxx)."
  echo "Branch must follow: fix/CYB-{id}-* | feat/CYB-{id}-* | hotfix/CYB-{id}-*"
  exit 1
fi

# Check if any runtime files are changed
RUNTIME_PATHS="backend/ Frontend/ sdk/ dagster/"
HAS_RUNTIME_CHANGE=false
for path in $RUNTIME_PATHS; do
  if echo "$CHANGED_FILES" | grep -q "^${path}"; then
    HAS_RUNTIME_CHANGE=true
    break
  fi
done

if [[ "$HAS_RUNTIME_CHANGE" == "false" ]]; then
  echo "OK: No runtime files changed. OpenSpec check skipped."
  exit 0
fi

# Find matching openspec change directory
CHANGE_DIR=""
for dir in openspec/changes/${CHANGE_ID}-*/; do
  if [[ -d "$dir" ]]; then
    CHANGE_DIR="$dir"
    break
  fi
done

if [[ -z "$CHANGE_DIR" ]]; then
  echo "ERROR: No OpenSpec change directory found matching '${CHANGE_ID}-*' under openspec/changes/"
  echo "Create: openspec/changes/${CHANGE_ID}-<slug>/ with required files."
  exit 1
fi

echo "Found OpenSpec change: $CHANGE_DIR"

# Check required files
MISSING=()

if [[ ! -f "${CHANGE_DIR}proposal.md" ]]; then
  MISSING+=("proposal.md")
fi

if [[ ! -f "${CHANGE_DIR}tasks.md" ]]; then
  MISSING+=("tasks.md")
fi

# Check for at least one spec delta
SPEC_COUNT=$(find "${CHANGE_DIR}specs" -name "spec.md" 2>/dev/null | wc -l)
if [[ "$SPEC_COUNT" -eq 0 ]]; then
  MISSING+=("specs/*/spec.md (at least one spec delta)")
fi

# For feature branches, also require design.md
if [[ "$BRANCH_NAME" == feat/* ]]; then
  if [[ ! -f "${CHANGE_DIR}design.md" ]]; then
    MISSING+=("design.md (required for feature branches)")
  fi
fi

if [[ ${#MISSING[@]} -gt 0 ]]; then
  echo "ERROR: OpenSpec change '$CHANGE_DIR' is missing required files:"
  for f in "${MISSING[@]}"; do
    echo "  - $f"
  done
  exit 1
fi

# Recommended: curated context (Trellis-style) — warn only
if [[ ! -f "${CHANGE_DIR}context-files.md" ]]; then
  if ! grep -qE '^##[[:space:]]+Context files' "${CHANGE_DIR}tasks.md" 2>/dev/null; then
    echo "WARN: Add ${CHANGE_DIR}context-files.md (or ## Context files in tasks.md) — see openspec/changes/README.md"
  fi
fi

echo "OK: OpenSpec change '$CHANGE_DIR' is complete."
exit 0
