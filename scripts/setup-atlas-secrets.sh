#!/usr/bin/env bash
# Set up Atlas Pro secrets for the cyber-databrew repo.
#
# Required (run once):
#   - An Atlas Pro token (https://auth.atlasgo.cloud/settings/tokens)
#     — used for `atlas migrate lint --web` so lint reports are uploaded
#       to Atlas Cloud with a shareable URL.
#
# Usage:
#   ATLAS_TOKEN=atlas_xxx bash scripts/setup-atlas-secrets.sh
#
# Add --dry-run to print the gh secret set commands without executing them.
#
# After setup, verify with: bash scripts/setup-atlas-secrets.sh --check

set -euo pipefail

if [[ "${1:-}" == "--check" ]]; then
  echo "==> Checking required Atlas secrets are set on this repo"
  REQUIRED=(ATLAS_TOKEN)
  missing=()
  for s in "${REQUIRED[@]}"; do
    if gh secret list --json name --jq '.[].name' 2>/dev/null | grep -qx "$s"; then
      echo "  ✓ $s"
    else
      echo "  ✗ $s MISSING"
      missing+=("$s")
    fi
  done
  if [[ ${#missing[@]} -gt 0 ]]; then
    echo
    echo "${#missing[@]} secret(s) missing. Run with ATLAS_TOKEN set to set them."
    exit 1
  fi
  echo
  echo "All Atlas secrets present."
  exit 0
fi

if [[ "${1:-}" == "--dry-run" ]]; then
  DRY_RUN=1
else
  DRY_RUN=0
fi

set_secret() {
  local name="$1"
  local value="$2"
  if [[ -z "$value" ]]; then
    echo "SKIP $name (empty value)"
    return
  fi
  if (( DRY_RUN )); then
    echo "DRY-RUN: gh secret set $name --body ***"
  else
    echo -n "$value" | gh secret set "$name" --body -
    echo "SET  $name"
  fi
}

: "${ATLAS_TOKEN:?ATLAS_TOKEN required (Atlas Pro token from https://auth.atlasgo.cloud/settings/tokens)}"

if (( ! DRY_RUN )); then
  if ! command -v gh >/dev/null 2>&1; then
    echo "ERROR: gh CLI not found. Install: https://cli.github.com" >&2
    exit 1
  fi
  if ! gh auth status >/dev/null 2>&1; then
    echo "ERROR: gh not authenticated. Run: gh auth login" >&2
    exit 1
  fi
fi

echo "==> Setting Atlas Pro secrets"
set_secret ATLAS_TOKEN "$ATLAS_TOKEN"

if (( DRY_RUN )); then
  echo
  echo "Dry-run complete. Re-run without --dry-run to apply."
else
  echo
  echo "Done. Verify with: bash scripts/setup-atlas-secrets.sh --check"
fi