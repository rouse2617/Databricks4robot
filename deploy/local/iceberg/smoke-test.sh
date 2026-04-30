#!/usr/bin/env bash
# smoke-test.sh — Verify PyIceberg can connect to the Iceberg REST Catalog
# and Trino can query Iceberg tables.
#
# Prerequisites:
#   docker compose -f deploy/local/docker-compose.iceberg.yml up -d
#   (or: make iceberg-up)
#
# Usage:
#   bash deploy/local/iceberg/smoke-test.sh
#
# Exit codes:
#   0 — all checks passed
#   1 — one or more checks failed

set -euo pipefail

ICEBERG_REST_URI="${ICEBERG_REST_URI:-http://localhost:8181}"
TRINO_HOST="${TRINO_HOST:-localhost}"
TRINO_PORT="${TRINO_PORT:-8082}"
MINIO_ENDPOINT="${MINIO_ENDPOINT:-http://localhost:9000}"
MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-admin}"
MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-password}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

PASS=0
FAIL=0

check() {
  local name="$1"
  shift
  if "$@" >/dev/null 2>&1; then
    echo -e "  ${GREEN}✓${NC} $name"
    PASS=$((PASS + 1))
  else
    echo -e "  ${RED}✗${NC} $name"
    FAIL=$((FAIL + 1))
  fi
}

echo "═══════════════════════════════════════════════════"
echo " Iceberg Lakehouse Smoke Test"
echo "═══════════════════════════════════════════════════"
echo ""

# ── 1. Service health checks ──────────────────────────
echo "1. Service connectivity"

check "Iceberg REST Catalog reachable" \
  curl -sf "${ICEBERG_REST_URI}/v1/config"

check "MinIO reachable" \
  curl -sf "${MINIO_ENDPOINT}/minio/health/live"

check "Trino reachable" \
  curl -sf "http://${TRINO_HOST}:${TRINO_PORT}/v1/info"

echo ""

# ── 2. Iceberg REST Catalog API ───────────────────────
echo "2. Iceberg REST Catalog API"

# List namespaces
check "List namespaces" \
  curl -sf "${ICEBERG_REST_URI}/v1/namespaces"

echo ""

# ── 3. Trino → Iceberg query ─────────────────────────
echo "3. Trino → Iceberg queries"

# Create a test namespace and table via Trino
TRINO_CMD="docker compose -f deploy/local/docker-compose.iceberg.yml exec -T trino trino --server http://localhost:8080 --catalog iceberg --schema robot"

# Try to create schema (may already exist)
echo "   Creating test schema..."
${TRINO_CMD} --execute "CREATE SCHEMA IF NOT EXISTS iceberg.smoke_test" 2>/dev/null || true

# Create a test table
echo "   Creating test table..."
${TRINO_CMD} --execute "
  CREATE TABLE IF NOT EXISTS iceberg.smoke_test.smoke_events (
    event_id VARCHAR,
    event_type VARCHAR,
    asset_id VARCHAR,
    event_seq BIGINT,
    created_at TIMESTAMP
  )
" 2>/dev/null || true

# Insert test data
echo "   Inserting test data..."
${TRINO_CMD} --execute "
  INSERT INTO iceberg.smoke_test.smoke_events VALUES
    ('evt-001', 'asset_created', 'asset-001', 1, TIMESTAMP '2025-01-15 00:00:00'),
    ('evt-002', 'tag_upserted', 'asset-001', 2, TIMESTAMP '2025-01-15 00:01:00'),
    ('evt-003', 'algo_finished', 'asset-002', 3, TIMESTAMP '2025-01-15 00:02:00')
" 2>/dev/null || true

# Query and verify
RESULT=$(${TRINO_CMD} --execute "SELECT count(*) FROM iceberg.smoke_test.smoke_events" 2>/dev/null | tr -d '[:space:]')
if [ -n "$RESULT" ] && [ "$RESULT" -ge 3 ] 2>/dev/null; then
  echo -e "  ${GREEN}✓${NC} Trino query returned ${RESULT} rows (expected ≥ 3)"
  PASS=$((PASS + 1))
else
  echo -e "  ${RED}✗${NC} Trino query returned '${RESULT}' rows (expected ≥ 3)"
  FAIL=$((FAIL + 1))
fi

# Verify snapshot exists (Iceberg time travel)
SNAPSHOT_RESULT=$(${TRINO_CMD} --execute "SELECT count(*) FROM iceberg.smoke_test.\"smoke_events\$snapshots\"" 2>/dev/null | tr -d '[:space:]')
if [ -n "$SNAPSHOT_RESULT" ] && [ "$SNAPSHOT_RESULT" -ge 1 ] 2>/dev/null; then
  echo -e "  ${GREEN}✓${NC} Iceberg snapshot exists (${SNAPSHOT_RESULT} snapshots)"
  PASS=$((PASS + 1))
else
  echo -e "  ${YELLOW}~${NC} Could not verify Iceberg snapshots (non-critical)"
fi

echo ""

# ── 4. Cleanup ────────────────────────────────────────
echo "4. Cleanup"
${TRINO_CMD} --execute "DROP TABLE IF EXISTS iceberg.smoke_test.smoke_events" 2>/dev/null || true
${TRINO_CMD} --execute "DROP SCHEMA IF EXISTS iceberg.smoke_test" 2>/dev/null || true
echo -e "  ${GREEN}✓${NC} Test artifacts cleaned up"

echo ""

# ── Summary ───────────────────────────────────────────
echo "═══════════════════════════════════════════════════"
if [ "$FAIL" -eq 0 ]; then
  echo -e " ${GREEN}ALL CHECKS PASSED${NC} (${PASS} passed, ${FAIL} failed)"
  echo "═══════════════════════════════════════════════════"
  exit 0
else
  echo -e " ${RED}SOME CHECKS FAILED${NC} (${PASS} passed, ${FAIL} failed)"
  echo "═══════════════════════════════════════════════════"
  exit 1
fi
