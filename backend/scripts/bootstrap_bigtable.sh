#!/usr/bin/env bash
set -euo pipefail

# Bootstrap Bigtable tables/column families.
# Source of truth:
#   schemas/sql.md in the cyber-databrew repository
#
# Requires:
#   - cbt installed
#   - CBT_PROJECT + CBT_INSTANCE set (or in ~/.cbtrc)

TABLES=(
  "assets:meta,algo,tag,files"
  "mcap_files:meta,process"
  "deliveries:meta"
  "idx_segments_by_file:ref"
  "idx_asset_deliveries:ref"
  "idx_customer_deliveries:ref"
  "idempotency_keys:meta"
  "asset_algo_events:meta"
)

exists_table() {
  local table="$1"
  cbt ls 2>/dev/null | grep -qx "${table}"
}

exists_family() {
  local table="$1"
  local family="$2"
  cbt ls "${table}" 2>/dev/null | awk '{print $1}' | grep -qx "${family}"
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
