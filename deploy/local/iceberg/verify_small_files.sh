#!/usr/bin/env bash
# verify_small_files.sh — Check small file count stability after 7 days.
#
# Design reference: data-platform-design.md §5.6, tasks.md 6.8
#
# This script queries Iceberg table metadata via Trino to report:
#   1. Total data file count per table
#   2. Small file count (< 128 MB) per table
#   3. Average file size per table
#   4. Snapshot count per table
#
# Run this daily for 7+ days and compare results to verify that the
# compact.py CronJob keeps small file counts stable.
#
# Acceptance criteria (task 6.8):
#   - After 7 days of continuous Bronze MERGE + daily compaction:
#   - Small file count for Silver/Gold tables should be stable (not growing)
#   - Bronze table small files are acceptable (append-heavy pattern)
#
# Usage:
#   bash deploy/local/iceberg/verify_small_files.sh
#   bash deploy/local/iceberg/verify_small_files.sh >> /tmp/small_files_log.csv
#
# Environment:
#   TRINO_HOST  — Trino host (default: localhost)
#   TRINO_PORT  — Trino port (default: 8082)

set -euo pipefail

TRINO_HOST="${TRINO_HOST:-localhost}"
TRINO_PORT="${TRINO_PORT:-8082}"
TARGET_SIZE_BYTES=$((128 * 1024 * 1024))  # 128 MB

TRINO_CMD="docker compose -f deploy/local/docker-compose.iceberg.yml exec -T trino trino --server http://localhost:8080 --catalog iceberg --schema robot --output-format CSV_HEADER"

TABLES=(
  "bronze_asset_events"
  "silver_mcap_files_current"
  "silver_assets_current"
  "silver_deliveries_current"
  "silver_asset_algo_latest"
  "silver_asset_tags"
  "gold_dataset_snapshot_items"
)

NOW=$(date -u '+%Y-%m-%d %H:%M:%S')

echo "═══════════════════════════════════════════════════════════"
echo " Small File Count Report — ${NOW} UTC"
echo "═══════════════════════════════════════════════════════════"
echo ""
printf "%-35s %10s %12s %12s %10s\n" "TABLE" "FILES" "SMALL_FILES" "AVG_SIZE_MB" "SNAPSHOTS"
printf "%-35s %10s %12s %12s %10s\n" "---" "---" "---" "---" "---"

for TABLE in "${TABLES[@]}"; do
  # Query file metadata from Iceberg metadata tables
  RESULT=$(${TRINO_CMD} --execute "
    SELECT
      count(*) AS total_files,
      count(CASE WHEN file_size_in_bytes < ${TARGET_SIZE_BYTES} THEN 1 END) AS small_files,
      COALESCE(round(avg(file_size_in_bytes) / 1048576.0, 2), 0) AS avg_size_mb
    FROM iceberg.robot.\"${TABLE}\$files\"
  " 2>/dev/null | tail -1 || echo "0,0,0")

  SNAP_COUNT=$(${TRINO_CMD} --execute "
    SELECT count(*) FROM iceberg.robot.\"${TABLE}\$snapshots\"
  " 2>/dev/null | tail -1 | tr -d '[:space:]"' || echo "0")

  TOTAL=$(echo "${RESULT}" | cut -d',' -f1 | tr -d '"[:space:]')
  SMALL=$(echo "${RESULT}" | cut -d',' -f2 | tr -d '"[:space:]')
  AVG_MB=$(echo "${RESULT}" | cut -d',' -f3 | tr -d '"[:space:]')

  printf "%-35s %10s %12s %12s %10s\n" "${TABLE}" "${TOTAL}" "${SMALL}" "${AVG_MB}" "${SNAP_COUNT}"
done

echo ""
echo "═══════════════════════════════════════════════════════════"
echo " Stability criteria:"
echo "   - Silver/Gold small_files should NOT grow over 7 days"
echo "   - Bronze small_files may grow (append-heavy, compacted less)"
echo "   - Run daily: bash verify_small_files.sh >> /tmp/file_report.log"
echo "═══════════════════════════════════════════════════════════"
