#!/usr/bin/env bash
set -euo pipefail

# Bootstrap Bigtable tables/column families.
# Source of truth:
#   schemas/sql.md in the data4cyber repository
#
# Requires:
#   - cbt installed
#   - CBT_PROJECT + CBT_INSTANCE set (or in ~/.cbtrc)

TABLES=(
  "assets:cf:meta,cf:algo,cf:tag"
  "mcap_files:cf:meta,cf:process"
  "deliveries:cf:meta"
  "idx_segments_by_file:ref"
  "idx_asset_deliveries:ref"
  "idx_customer_deliveries:ref"
  "idempotency_keys:meta"
)

exists_table() {
  local table="$1"
  cbt ls | rg -n "^${table}$" >/dev/null 2>&1
}

exists_family() {
  local table="$1"
  local family="$2"
  cbt ls "${table}" | rg -n "^[[:space:]]+Families:.*\\b${family}\\b" >/dev/null 2>&1
}

for entry in "${TABLES[@]}"; do
  table="${entry%%:*}"
  families_csv="${entry#*:}"

  if exists_table "${table}"; then
    echo "[skip] table exists: ${table}"
  else
    echo "[create] table: ${table}"
    cbt createtable "${table}"
  fi

  IFS=',' read -r -a families <<< "${families_csv}"
  for family in "${families[@]}"; do
    if exists_family "${table}" "${family}"; then
      echo "  [skip] family exists: ${table}.${family}"
    else
      echo "  [create] family: ${table}.${family}"
      cbt createfamily "${table}" "${family}"
    fi
  done
done

echo "[done] Bigtable schema bootstrap completed."
