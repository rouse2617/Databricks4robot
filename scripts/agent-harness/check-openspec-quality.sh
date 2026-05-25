#!/usr/bin/env bash
# Advisory quality checks for OpenSpec change artifacts.
# Run before the OpenSpec checkpoint — warns on common issues.
#
# Usage:
#   bash scripts/agent-harness/check-openspec-quality.sh openspec/changes/CYB-123-slug
#   bash scripts/agent-harness/check-openspec-quality.sh --all   # check all active changes
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT}"

WARNINGS=0
ERRORS=0

warn() { echo "  WARN: $*" >&2; WARNINGS=$((WARNINGS + 1)); }
err()  { echo "  ERR:  $*" >&2; ERRORS=$((ERRORS + 1)); }
ok()   { echo "  OK:   $*"; }

check_change() {
  local dir="$1"
  local name
  name="$(basename "$dir")"

  echo "=== ${name} ==="

  # ── proposal.md ──
  if [[ ! -f "${dir}proposal.md" ]]; then
    err "proposal.md missing"
  else
    local p="${dir}proposal.md"
    ok "proposal.md exists"

    grep -qi '## Why' "$p" || warn "proposal.md: missing '## Why' section"
    grep -qi '## What Changes' "$p" || warn "proposal.md: missing '## What Changes' section"
    grep -qi '## Impact' "$p" || warn "proposal.md: missing '## Impact' section"
    grep -qi '## Scope' "$p" || warn "proposal.md: missing '## Scope' section"

    # Why should be short (heuristic: ≤150 chars of actual content after heading)
    local why_text
    why_text=$(sed -n '/## Why/,/## /p' "$p" | grep -v '## ' | tr -d '\n' | sed 's/^[[:space:]]*//')
    if [[ ${#why_text} -gt 300 ]]; then
      warn "proposal.md: 'Why' section is long (${#why_text} chars). Aim for ≤3 sentences."
    fi

    # In/Out scope
    if grep -qi '## Scope' "$p" && ! grep -qi 'out of scope\|out-of-scope' "$p"; then
      warn "proposal.md: 'Scope' should include both In scope and Out of scope"
    fi
  fi

  # ── specs/ delta ──
  local spec_files
  spec_files=$(find "${dir}specs" -name 'spec.md' 2>/dev/null || true)
  if [[ -z "$spec_files" ]]; then
    warn "spec delta missing (no specs/*/spec.md found)"
  else
    while IFS= read -r sf; do
      ok "spec delta: ${sf#${dir}}"

      # Each Requirement must have Priority
      if grep -q '### Requirement:' "$sf"; then
        local req_count
        req_count=$(grep -c '### Requirement:' "$sf" || true)
        local pri_count
        pri_count=$(grep -c '\*\*Priority\*\*' "$sf" || true)
        if [[ $pri_count -lt $req_count ]]; then
          warn "spec: $req_count requirements, but only $pri_count have Priority"
        fi

        # Rationale
        local rat_count
        rat_count=$(grep -c '\*\*Rationale\*\*' "$sf" || true)
        if [[ $rat_count -lt $req_count ]]; then
          warn "spec: $req_count requirements, but only $rat_count have Rationale"
        fi

        # Scenarios
        local scen_count
        scen_count=$(grep -c '#### Scenario:' "$sf" || true)
        if [[ $scen_count -lt $req_count ]]; then
          warn "spec: $req_count requirements, but only $scen_count scenarios (need ≥1 per requirement)"
        fi

        # Error path: each req should have ≥2 scenarios (happy + error)
        if [[ $scen_count -lt $((req_count * 2)) ]]; then
          warn "spec: consider adding error-path scenarios (currently $scen_count scenarios for $req_count requirements)"
        fi
      fi

      # No API paths / field names in spec
      if grep -qE '(POST|GET|DELETE|PUT|PATCH) /api/' "$sf" 2>/dev/null; then
        warn "spec: contains API paths — behavior specs should not mention HTTP routes"
      fi
    done <<< "$spec_files"
  fi

  # ── design.md (if present) ──
  if [[ -f "${dir}design.md" ]]; then
    ok "design.md exists"

    # Each Decision should have Approach + Alternative + Rationale
    local dec_count
    dec_count=$(grep -c '### Decision' "${dir}design.md" || true)
    if [[ $dec_count -gt 0 ]]; then
      local app_count alt_count rat_count
      app_count=$(grep -c '\*\*Approach\*\*' "${dir}design.md" || true)
      alt_count=$(grep -c '\*\*Alternative\*\*' "${dir}design.md" || true)
      rat_count=$(grep -c '\*\*Rationale\*\*' "${dir}design.md" || true)

      [[ $app_count -ge $dec_count ]] || warn "design.md: $dec_count decisions but only $app_count have Approach"
      [[ $alt_count -ge $dec_count ]] || warn "design.md: $dec_count decisions but only $alt_count have Alternative"
      [[ $rat_count -ge $dec_count ]] || warn "design.md: $dec_count decisions but only $rat_count have Rationale"
    fi
  fi

  # ── tasks.md ──
  if [[ ! -f "${dir}tasks.md" ]]; then
    err "tasks.md missing"
  else
    ok "tasks.md exists"

    # Deploy verification section
    if ! grep -qi 'Deploy verification\|deploy verification' "${dir}tasks.md"; then
      warn "tasks.md: missing 'Deploy verification' section"
    fi

    # Checkbox tags: [backend] / [Frontend] / [sdk]
    local cb_count
    cb_count=$(grep -c '^\s*- \[ \]' "${dir}tasks.md" || true)
    if [[ $cb_count -gt 0 ]]; then
      local tagged
      tagged=$(grep -cE '^\s*- \[ \].*\[(backend|Frontend|sdk)\]' "${dir}tasks.md" || true)
      if [[ $tagged -lt $cb_count ]]; then
        warn "tasks.md: $cb_count checkboxes, but only $tagged tagged with [backend]/[Frontend]/[sdk]"
      fi
    fi

    # Context files section
    if ! grep -qi 'Context files' "${dir}tasks.md" && [[ ! -f "${dir}context-files.md" ]]; then
      warn "tasks.md: missing 'Context files' section (or context-files.md)"
    fi
  fi

  # ── decisions.md (warn if off-limits or precedence conflict expected but no log) ──
  # Only a soft check: if the change touches off-limits paths, decisions.md should exist
  if [[ ! -f "${dir}decisions.md" ]]; then
    :
    # Not an error — decisions.md is only needed when rules are overridden.
  fi

  echo ""
}

check_all() {
  local found=0
  for d in openspec/changes/CYB-*/; do
    [[ -d "$d" ]] || continue
    [[ "$(basename "$d")" == "README.md" ]] && continue
    found=1
    check_change "$d"
  done
  if [[ $found -eq 0 ]]; then
    echo "No active change directories found."
  fi
}

if [[ "${1:-}" == "--all" ]]; then
  check_all
elif [[ $# -ge 1 ]]; then
  check_change "${1%/}/"
else
  echo "usage: $0 <openspec/changes/CYB-xxx-slug/> | --all" >&2
  exit 1
fi

echo "=== Summary: ${ERRORS} error(s), ${WARNINGS} warning(s) ==="
[[ $ERRORS -gt 0 ]] && exit 1
exit 0
