#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────────────────────
# es-pg-audit.sh — PG↔ES 一致性对账
#
# 对应任务:
#   5.2 scripts/es-pg-audit.sh：PG↔ES 一致率 > 99.9%
#
# 检查项:
#   1. PG 非删除资产总数 vs ES 文档总数
#   2. PG 中存在但 ES 缺失的 asset_id
#   3. ES 中存在但 PG 缺失的 asset_id（孤儿文档）
#   4. 一致率计算（目标 > 99.9%）
#
# 前置条件:
#   - PostgreSQL 可达
#   - Elasticsearch 可达
#   - psql + curl + jq 已安装
#
# 用法:
#   bash backend/scripts/es-pg-audit.sh
#
# 可选环境变量:
#   DB_HOST            — PG 主机 (默认 localhost)
#   DB_PORT            — PG 端口 (默认 5432)
#   DB_USER            — PG 用户 (默认 postgres)
#   DB_PASSWORD        — PG 密码 (默认 postgres)
#   DB_NAME            — PG 数据库 (默认 data4cyber)
#   ELASTICSEARCH_URL  — ES 地址 (默认 http://localhost:9200)
#   ES_INDEX           — ES 索引名 (默认 assets)
#   CONSISTENCY_TARGET — 一致率目标 (默认 0.999 即 99.9%)
# ──────────────────────────────────────────────────────────────────────────────
set -euo pipefail

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
DB_NAME="${DB_NAME:-data4cyber}"
ELASTICSEARCH_URL="${ELASTICSEARCH_URL:-http://localhost:9200}"
ES_INDEX="${ES_INDEX:-assets}"
CONSISTENCY_TARGET="${CONSISTENCY_TARGET:-0.999}"

export PGPASSWORD="$DB_PASSWORD"

PASS=0
FAIL=0
TOTAL=0

# ── 颜色 ─────────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# ── 工具函数 ─────────────────────────────────────────────────────────────────

psql_query() {
  psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
    -t -A -c "$1" 2>/dev/null
}

section() {
  echo ""
  echo -e "${CYAN}━━━ $1 ━━━${NC}"
}

check_pass() {
  TOTAL=$((TOTAL + 1))
  PASS=$((PASS + 1))
  echo -e "  ${GREEN}✓${NC} $1"
}

check_fail() {
  TOTAL=$((TOTAL + 1))
  FAIL=$((FAIL + 1))
  echo -e "  ${RED}✗${NC} $1"
}

# ── 0. 连接检查 ──────────────────────────────────────────────────────────────

section "0. 连接检查"

PG_COUNT=$(psql_query "SELECT COUNT(*) FROM assets WHERE is_deleted = FALSE;" || echo "ERROR")
if [ "$PG_COUNT" = "ERROR" ] || [ -z "$PG_COUNT" ]; then
  echo -e "  ${RED}✗ 无法连接到 PostgreSQL ($DB_HOST:$DB_PORT/$DB_NAME)${NC}"
  exit 1
fi
echo -e "  ${GREEN}✓${NC} PostgreSQL 已连接 — 活跃资产: ${YELLOW}${PG_COUNT}${NC}"

ES_HEALTH=$(curl -sf "${ELASTICSEARCH_URL}/_cluster/health" 2>/dev/null || echo "ERROR")
if [ "$ES_HEALTH" = "ERROR" ]; then
  echo -e "  ${RED}✗ 无法连接到 Elasticsearch ($ELASTICSEARCH_URL)${NC}"
  exit 1
fi
echo -e "  ${GREEN}✓${NC} Elasticsearch 已连接"

# Check if index exists
ES_INDEX_EXISTS=$(curl -sf -o /dev/null -w "%{http_code}" "${ELASTICSEARCH_URL}/${ES_INDEX}" 2>/dev/null || echo "000")
if [ "$ES_INDEX_EXISTS" != "200" ]; then
  echo -e "  ${RED}✗ ES 索引 '${ES_INDEX}' 不存在${NC}"
  exit 1
fi
echo -e "  ${GREEN}✓${NC} ES 索引 '${ES_INDEX}' 存在"

# ── 1. 文档计数对比 ─────────────────────────────────────────────────────────

section "1. 文档计数对比"

# Force ES refresh to get accurate count
curl -sf -X POST "${ELASTICSEARCH_URL}/${ES_INDEX}/_refresh" > /dev/null 2>&1

ES_COUNT=$(curl -sf "${ELASTICSEARCH_URL}/${ES_INDEX}/_count" 2>/dev/null | jq -r '.count // 0')

echo -e "  PG 活跃资产数: ${YELLOW}${PG_COUNT}${NC}"
echo -e "  ES 文档数:     ${YELLOW}${ES_COUNT}${NC}"

DIFF=$((PG_COUNT - ES_COUNT))
if [ "$DIFF" -lt 0 ]; then
  DIFF=$((-DIFF))
fi

if [ "$DIFF" -eq 0 ]; then
  check_pass "计数完全一致 (差异: 0)"
else
  check_fail "计数不一致 (差异: ${DIFF})"
fi

# ── 2. 逐条 ID 对比 ─────────────────────────────────────────────────────────

section "2. 逐条 asset_id 对比"

# Get all PG asset IDs (non-deleted)
PG_IDS_FILE=$(mktemp)
psql_query "SELECT asset_id FROM assets WHERE is_deleted = FALSE ORDER BY asset_id;" > "$PG_IDS_FILE"
PG_ID_COUNT=$(wc -l < "$PG_IDS_FILE" | tr -d ' ')

# Get all ES asset IDs via scroll API
ES_IDS_FILE=$(mktemp)
SCROLL_SIZE=1000

# Initial scroll request
SCROLL_RESPONSE=$(curl -sf "${ELASTICSEARCH_URL}/${ES_INDEX}/_search?scroll=2m" \
  -H 'Content-Type: application/json' \
  -d "{
    \"size\": ${SCROLL_SIZE},
    \"_source\": false,
    \"query\": { \"match_all\": {} },
    \"sort\": [\"_doc\"]
  }" 2>/dev/null)

SCROLL_ID=$(echo "$SCROLL_RESPONSE" | jq -r '._scroll_id // empty')
echo "$SCROLL_RESPONSE" | jq -r '.hits.hits[]._id' >> "$ES_IDS_FILE"
HIT_COUNT=$(echo "$SCROLL_RESPONSE" | jq -r '.hits.hits | length')

# Continue scrolling
while [ "$HIT_COUNT" -gt 0 ] && [ -n "$SCROLL_ID" ]; do
  SCROLL_RESPONSE=$(curl -sf "${ELASTICSEARCH_URL}/_search/scroll" \
    -H 'Content-Type: application/json' \
    -d "{\"scroll\": \"2m\", \"scroll_id\": \"${SCROLL_ID}\"}" 2>/dev/null)
  SCROLL_ID=$(echo "$SCROLL_RESPONSE" | jq -r '._scroll_id // empty')
  echo "$SCROLL_RESPONSE" | jq -r '.hits.hits[]._id' >> "$ES_IDS_FILE"
  HIT_COUNT=$(echo "$SCROLL_RESPONSE" | jq -r '.hits.hits | length')
done

# Clear scroll
if [ -n "$SCROLL_ID" ]; then
  curl -sf -X DELETE "${ELASTICSEARCH_URL}/_search/scroll" \
    -H 'Content-Type: application/json' \
    -d "{\"scroll_id\": \"${SCROLL_ID}\"}" > /dev/null 2>&1 || true
fi

sort "$ES_IDS_FILE" -o "$ES_IDS_FILE"
ES_ID_COUNT=$(wc -l < "$ES_IDS_FILE" | tr -d ' ')

echo -e "  PG asset_id 数: ${YELLOW}${PG_ID_COUNT}${NC}"
echo -e "  ES asset_id 数: ${YELLOW}${ES_ID_COUNT}${NC}"

# Find IDs in PG but missing from ES
MISSING_IN_ES_FILE=$(mktemp)
comm -23 "$PG_IDS_FILE" "$ES_IDS_FILE" > "$MISSING_IN_ES_FILE"
MISSING_IN_ES=$(wc -l < "$MISSING_IN_ES_FILE" | tr -d ' ')

# Find IDs in ES but missing from PG (orphans)
ORPHAN_IN_ES_FILE=$(mktemp)
comm -13 "$PG_IDS_FILE" "$ES_IDS_FILE" > "$ORPHAN_IN_ES_FILE"
ORPHAN_IN_ES=$(wc -l < "$ORPHAN_IN_ES_FILE" | tr -d ' ')

if [ "$MISSING_IN_ES" -eq 0 ]; then
  check_pass "PG→ES 缺失: 0"
else
  check_fail "PG→ES 缺失: ${MISSING_IN_ES} 条"
  echo -e "    ${YELLOW}前 10 条缺失 ID:${NC}"
  head -10 "$MISSING_IN_ES_FILE" | while read -r id; do
    echo -e "      $id"
  done
  if [ "$MISSING_IN_ES" -gt 10 ]; then
    echo -e "      ... 共 ${MISSING_IN_ES} 条"
  fi
fi

if [ "$ORPHAN_IN_ES" -eq 0 ]; then
  check_pass "ES 孤儿文档: 0"
else
  check_fail "ES 孤儿文档: ${ORPHAN_IN_ES} 条 (ES 有但 PG 无)"
  echo -e "    ${YELLOW}前 10 条孤儿 ID:${NC}"
  head -10 "$ORPHAN_IN_ES_FILE" | while read -r id; do
    echo -e "      $id"
  done
  if [ "$ORPHAN_IN_ES" -gt 10 ]; then
    echo -e "      ... 共 ${ORPHAN_IN_ES} 条"
  fi
fi

# ── 3. 一致率计算 ────────────────────────────────────────────────────────────

section "3. 一致率"

if [ "$PG_ID_COUNT" -eq 0 ]; then
  echo -e "  ${YELLOW}⚠ PG 无活跃资产，跳过一致率计算${NC}"
  CONSISTENCY_RATE="1.000000"
  CONSISTENCY_PCT="100.0000%"
else
  TOTAL_DISCREPANCY=$((MISSING_IN_ES + ORPHAN_IN_ES))
  CONSISTENCY_RATE=$(echo "$TOTAL_DISCREPANCY $PG_ID_COUNT" | awk '{printf "%.6f", 1 - ($1/$2)}')
  CONSISTENCY_PCT=$(echo "$CONSISTENCY_RATE" | awk '{printf "%.4f%%", $1*100}')
fi

MEETS_TARGET=$(echo "$CONSISTENCY_RATE $CONSISTENCY_TARGET" | awk '{print ($1 >= $2) ? "yes" : "no"}')
TARGET_PCT=$(echo "$CONSISTENCY_TARGET" | awk '{printf "%.1f%%", $1*100}')

if [ "$MEETS_TARGET" = "yes" ]; then
  check_pass "一致率: ${CONSISTENCY_PCT} (目标 ≥ ${TARGET_PCT})"
else
  check_fail "一致率: ${CONSISTENCY_PCT} (目标 ≥ ${TARGET_PCT})"
fi

# ── 清理临时文件 ─────────────────────────────────────────────────────────────

rm -f "$PG_IDS_FILE" "$ES_IDS_FILE" "$MISSING_IN_ES_FILE" "$ORPHAN_IN_ES_FILE"

# ── 结果汇总 ─────────────────────────────────────────────────────────────────

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "  ${CYAN}PG↔ES 对账结果${NC}"
echo -e "  测试结果: ${GREEN}${PASS} passed${NC} / ${RED}${FAIL} failed${NC} / ${TOTAL} total"
echo -e "  PG 活跃资产: ${PG_COUNT}"
echo -e "  ES 文档数:   ${ES_COUNT}"
echo -e "  一致率:      ${CONSISTENCY_PCT}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if [ "$FAIL" -gt 0 ]; then
  echo -e "\n${RED}FAIL${NC} — PG↔ES 对账未通过"
  exit 1
else
  echo -e "\n${GREEN}PASS${NC} — PG↔ES 一致率 ≥ ${TARGET_PCT}"
  exit 0
fi
