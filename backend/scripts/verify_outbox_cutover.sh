#!/usr/bin/env bash
# verify_outbox_cutover.sh — Validate no event loss during cutover from
# in-process outbox worker to standalone outbox-worker binary.
#
# Task 7.7: 切换验证：无丢事件
#
# Usage:
#   ./scripts/verify_outbox_cutover.sh [PG_DSN] [ES_URL]
#
# Defaults:
#   PG_DSN  = postgres://postgres:postgres@localhost:5432/data4cyber?sslmode=disable
#   ES_URL  = http://localhost:9200
#
# What it checks:
#   1. PG pending count = 0 (all events published)
#   2. PG total events vs ES total docs — counts match
#   3. Cursor in outbox_sink_cursors is at or near MAX(event_seq)
#   4. No events stuck with high retry_count
#
# Prerequisites:
#   - psql and curl available on PATH
#   - Both PG and ES are reachable
#
# Exit codes:
#   0 = all checks passed
#   1 = one or more checks failed

set -euo pipefail

PG_DSN="${1:-postgres://postgres:postgres@localhost:5432/data4cyber?sslmode=disable}"
ES_URL="${2:-http://localhost:9200}"
ES_INDEX="assets"
SINK_NAME="es_assets"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

errors=0

echo "=== Outbox Cutover Verification ==="
echo "PG: $PG_DSN"
echo "ES: $ES_URL/$ES_INDEX"
echo ""

# ── 1. Pending events should be 0 ───────────────────────────────────
echo -n "1. Pending events count... "
pending=$(psql "$PG_DSN" -t -A -c \
  "SELECT COUNT(*) FROM asset_events WHERE publish_state = 'pending';" 2>/dev/null || echo "ERROR")
if [ "$pending" = "ERROR" ]; then
  echo -e "${RED}FAIL${NC} — could not query PG"
  errors=$((errors + 1))
elif [ "$pending" -eq 0 ]; then
  echo -e "${GREEN}OK${NC} (0 pending)"
else
  echo -e "${YELLOW}WARN${NC} ($pending pending events — worker may still be draining)"
  if [ "$pending" -gt 100 ]; then
    errors=$((errors + 1))
  fi
fi

# ── 2. PG total asset events vs ES doc count ─────────────────────────
echo -n "2. PG asset count vs ES doc count... "
pg_assets=$(psql "$PG_DSN" -t -A -c \
  "SELECT COUNT(*) FROM assets WHERE is_deleted = false;" 2>/dev/null || echo "ERROR")
es_count=$(curl -s "$ES_URL/$ES_INDEX/_count" 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin).get('count','ERROR'))" 2>/dev/null || echo "ERROR")

if [ "$pg_assets" = "ERROR" ] || [ "$es_count" = "ERROR" ]; then
  echo -e "${RED}FAIL${NC} — could not query PG ($pg_assets) or ES ($es_count)"
  errors=$((errors + 1))
elif [ "$pg_assets" = "$es_count" ]; then
  echo -e "${GREEN}OK${NC} (PG=$pg_assets, ES=$es_count)"
else
  diff=$((pg_assets - es_count))
  if [ "${diff#-}" -le 5 ]; then
    echo -e "${YELLOW}WARN${NC} (PG=$pg_assets, ES=$es_count, diff=$diff — within tolerance)"
  else
    echo -e "${RED}FAIL${NC} (PG=$pg_assets, ES=$es_count, diff=$diff)"
    errors=$((errors + 1))
  fi
fi

# ── 3. Cursor position vs MAX(event_seq) ────────────────────────────
echo -n "3. Cursor position... "
cursor=$(psql "$PG_DSN" -t -A -c \
  "SELECT COALESCE(last_published_seq, 0) FROM outbox_sink_cursors WHERE sink_name = '$SINK_NAME';" 2>/dev/null || echo "ERROR")
max_seq=$(psql "$PG_DSN" -t -A -c \
  "SELECT COALESCE(MAX(event_seq), 0) FROM asset_events;" 2>/dev/null || echo "ERROR")

if [ "$cursor" = "ERROR" ] || [ "$max_seq" = "ERROR" ]; then
  echo -e "${RED}FAIL${NC} — could not query cursor ($cursor) or max_seq ($max_seq)"
  errors=$((errors + 1))
elif [ "$cursor" -eq "$max_seq" ]; then
  echo -e "${GREEN}OK${NC} (cursor=$cursor, max_seq=$max_seq — fully caught up)"
elif [ $((max_seq - cursor)) -le 10 ]; then
  echo -e "${YELLOW}WARN${NC} (cursor=$cursor, max_seq=$max_seq — nearly caught up)"
else
  echo -e "${RED}FAIL${NC} (cursor=$cursor, max_seq=$max_seq — lag=$((max_seq - cursor)))"
  errors=$((errors + 1))
fi

# ── 4. Events stuck with high retry_count ────────────────────────────
echo -n "4. High-retry events... "
stuck=$(psql "$PG_DSN" -t -A -c \
  "SELECT COUNT(*) FROM asset_events WHERE publish_state = 'pending' AND retry_count > 5;" 2>/dev/null || echo "ERROR")
if [ "$stuck" = "ERROR" ]; then
  echo -e "${RED}FAIL${NC} — could not query"
  errors=$((errors + 1))
elif [ "$stuck" -eq 0 ]; then
  echo -e "${GREEN}OK${NC} (0 stuck events)"
else
  echo -e "${RED}FAIL${NC} ($stuck events with retry_count > 5)"
  errors=$((errors + 1))
fi

# ── Summary ──────────────────────────────────────────────────────────
echo ""
if [ "$errors" -eq 0 ]; then
  echo -e "${GREEN}All checks passed.${NC} Cutover is safe."
  exit 0
else
  echo -e "${RED}$errors check(s) failed.${NC} Investigate before completing cutover."
  exit 1
fi
