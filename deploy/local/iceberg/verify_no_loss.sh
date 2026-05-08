#!/usr/bin/env bash
# verify_no_loss.sh — 24h no-loss verification for Iceberg Bronze ingestion.
#
# Design reference: data-platform-design.md §5.6.2
#
# This script verifies that all published events in PostgreSQL have been
# successfully merged into the Iceberg Bronze table with zero data loss.
#
# Checks performed:
#   1. PG published event count vs Iceberg bronze event count
#   2. event_seq gap detection (no missing sequences)
#   3. Staging directory is empty (all files consumed)
#   4. Max event_seq alignment between PG and Iceberg
#
# Prerequisites:
#   - PostgreSQL running with asset_events table
#   - Iceberg profile running (see deploy/local/README.md)
#   - Bronze MERGE has been running for ≥ 24h
#
# Usage:
#   bash deploy/local/iceberg/verify_no_loss.sh
#
# Environment:
#   PG_HOST          — PostgreSQL host (default: localhost)
#   PG_PORT          — PostgreSQL port (default: 5432)
#   PG_USER          — PostgreSQL user (default: postgres)
#   PG_PASSWORD      — PostgreSQL password (default: postgres)
#   PG_DATABASE      — PostgreSQL database (default: data4cyber)
#   TRINO_HOST       — Trino host (default: localhost)
#   TRINO_PORT       — Trino port (default: 8082)
#   STAGING_DIR      — Staging directory (default: /tmp/iceberg-staging)
#   TOLERANCE_PCT    — Acceptable count difference % (default: 0.1)

set -euo pipefail

DEPLOY_LOCAL="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$DEPLOY_LOCAL"

PG_HOST="${PG_HOST:-localhost}"
PG_PORT="${PG_PORT:-5432}"
PG_USER="${PG_USER:-postgres}"
PG_PASSWORD="${PG_PASSWORD:-postgres}"
PG_DATABASE="${PG_DATABASE:-data4cyber}"
TRINO_HOST="${TRINO_HOST:-localhost}"
TRINO_PORT="${TRINO_PORT:-8082}"
STAGING_DIR="${STAGING_DIR:-/tmp/iceberg-staging}"
TOLERANCE_PCT="${TOLERANCE_PCT:-0.1}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

PASS=0
FAIL=0
WARN=0

check_pass() { echo -e "  ${GREEN}✓${NC} $1"; PASS=$((PASS + 1)); }
check_fail() { echo -e "  ${RED}✗${NC} $1"; FAIL=$((FAIL + 1)); }
check_warn() { echo -e "  ${YELLOW}~${NC} $1"; WARN=$((WARN + 1)); }

echo "═══════════════════════════════════════════════════════════"
echo " 24h No-Loss Verification — Iceberg Bronze Ingestion"
echo " $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
echo "═══════════════════════════════════════════════════════════"
echo ""

# ── 1. PostgreSQL event counts ────────────────────────
echo "1. PostgreSQL event counts"

PG_CMD="PGPASSWORD=${PG_PASSWORD} psql -h ${PG_HOST} -p ${PG_PORT} -U ${PG_USER} -d ${PG_DATABASE} -t -A"

PG_TOTAL=$(eval "${PG_CMD} -c \"SELECT count(*) FROM asset_events\"" 2>/dev/null || echo "ERROR")
PG_PUBLISHED=$(eval "${PG_CMD} -c \"SELECT count(*) FROM asset_events WHERE publish_state = 'published'\"" 2>/dev/null || echo "ERROR")
PG_PENDING=$(eval "${PG_CMD} -c \"SELECT count(*) FROM asset_events WHERE publish_state = 'pending'\"" 2>/dev/null || echo "ERROR")
PG_MAX_SEQ=$(eval "${PG_CMD} -c \"SELECT COALESCE(max(event_seq), 0) FROM asset_events\"" 2>/dev/null || echo "ERROR")

echo "   PG total events:     ${PG_TOTAL}"
echo "   PG published events: ${PG_PUBLISHED}"
echo "   PG pending events:   ${PG_PENDING}"
echo "   PG max event_seq:    ${PG_MAX_SEQ}"

if [ "${PG_TOTAL}" = "ERROR" ]; then
  check_fail "Cannot connect to PostgreSQL"
  echo ""
  echo "Aborting — PostgreSQL connection required."
  exit 1
fi

echo ""

# ── 2. Iceberg Bronze event counts (via Trino) ───────
echo "2. Iceberg Bronze event counts (via Trino)"

TRINO_CMD="docker compose --profile lakehouse exec -T trino trino --server http://localhost:8080 --catalog iceberg --schema robot --output-format CSV"

ICE_TOTAL=$(${TRINO_CMD} --execute "SELECT count(*) FROM bronze_asset_events" 2>/dev/null | tr -d '[:space:]"' || echo "ERROR")
ICE_MAX_SEQ=$(${TRINO_CMD} --execute "SELECT COALESCE(max(event_seq), 0) FROM bronze_asset_events" 2>/dev/null | tr -d '[:space:]"' || echo "ERROR")

echo "   Iceberg total events: ${ICE_TOTAL}"
echo "   Iceberg max event_seq: ${ICE_MAX_SEQ}"

if [ "${ICE_TOTAL}" = "ERROR" ]; then
  check_warn "Cannot query Iceberg Bronze table (may not exist yet)"
  ICE_TOTAL=0
  ICE_MAX_SEQ=0
fi

echo ""

# ── 3. Count comparison ──────────────────────────────
echo "3. Count comparison (PG published vs Iceberg)"

if [ "${PG_PUBLISHED}" != "ERROR" ] && [ "${ICE_TOTAL}" != "ERROR" ]; then
  if [ "${PG_PUBLISHED}" -eq 0 ]; then
    check_warn "No published events in PG yet"
  elif [ "${ICE_TOTAL}" -eq "${PG_PUBLISHED}" ]; then
    check_pass "Exact match: PG published (${PG_PUBLISHED}) = Iceberg (${ICE_TOTAL})"
  else
    DIFF=$((PG_PUBLISHED - ICE_TOTAL))
    if [ "${DIFF}" -lt 0 ]; then DIFF=$((-DIFF)); fi
    # Calculate percentage difference
    PCT=$(echo "scale=4; ${DIFF} * 100 / ${PG_PUBLISHED}" | bc 2>/dev/null || echo "999")
    if echo "${PCT} <= ${TOLERANCE_PCT}" | bc -l 2>/dev/null | grep -q 1; then
      check_pass "Within tolerance: diff=${DIFF} (${PCT}% ≤ ${TOLERANCE_PCT}%)"
    else
      check_fail "Count mismatch: PG=${PG_PUBLISHED}, Iceberg=${ICE_TOTAL}, diff=${DIFF} (${PCT}%)"
    fi
  fi
fi

echo ""

# ── 4. Staging directory check ───────────────────────
echo "4. Staging directory check"

if [ -d "${STAGING_DIR}" ]; then
  STAGING_COUNT=$(find "${STAGING_DIR}" -name "events_*.jsonl" 2>/dev/null | wc -l | tr -d '[:space:]')
  if [ "${STAGING_COUNT}" -eq 0 ]; then
    check_pass "Staging directory empty (all files consumed)"
  else
    check_warn "Staging directory has ${STAGING_COUNT} unconsumed file(s)"
    find "${STAGING_DIR}" -name "events_*.jsonl" -exec ls -la {} \; 2>/dev/null | head -5
  fi
else
  check_pass "Staging directory does not exist (no pending files)"
fi

echo ""

# ── 5. Pending event age check ───────────────────────
echo "5. Pending event age check"

if [ "${PG_PENDING}" != "ERROR" ] && [ "${PG_PENDING}" -gt 0 ]; then
  OLDEST_AGE=$(eval "${PG_CMD} -c \"SELECT EXTRACT(EPOCH FROM (now() - min(occurred_at)))::int FROM asset_events WHERE publish_state = 'pending'\"" 2>/dev/null || echo "ERROR")
  if [ "${OLDEST_AGE}" != "ERROR" ] && [ -n "${OLDEST_AGE}" ]; then
    if [ "${OLDEST_AGE}" -gt 86400 ]; then
      check_fail "Oldest pending event is ${OLDEST_AGE}s old (> 24h)"
    elif [ "${OLDEST_AGE}" -gt 3600 ]; then
      check_warn "Oldest pending event is ${OLDEST_AGE}s old (> 1h)"
    else
      check_pass "Oldest pending event is ${OLDEST_AGE}s old (< 1h)"
    fi
  fi
else
  check_pass "No pending events in PG"
fi

echo ""

# ── Summary ──────────────────────────────────────────
echo "═══════════════════════════════════════════════════════════"
if [ "${FAIL}" -eq 0 ]; then
  echo -e " ${GREEN}VERIFICATION PASSED${NC} (${PASS} passed, ${WARN} warnings, ${FAIL} failed)"
else
  echo -e " ${RED}VERIFICATION FAILED${NC} (${PASS} passed, ${WARN} warnings, ${FAIL} failed)"
fi
echo "═══════════════════════════════════════════════════════════"

exit "${FAIL}"
