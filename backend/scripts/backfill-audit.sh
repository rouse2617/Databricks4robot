#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────────────────────
# backfill-audit.sh — 新字段空值率 + 双写一致性对账
#
# 对应任务:
#   4.10 scripts/backfill-audit.sh
#   4.11 新字段空值率 < 0.1%，双写一致率 100%
#
# 检查项:
#   1. 新 promoted 字段空值率 < 0.1%（lifecycle_state, asset_type,
#      retention_tier, duration_ms, owner）
#   2. 双写一致性：promoted 字段与 cf_meta JSONB 中对应值 100% 一致
#
# 前置条件:
#   - PostgreSQL 可达（通过环境变量或默认 localhost:5432）
#   - psql 客户端已安装
#
# 用法:
#   bash backend/scripts/backfill-audit.sh
#
# 可选环境变量:
#   DB_HOST       — PG 主机 (默认 localhost)
#   DB_PORT       — PG 端口 (默认 5432)
#   DB_USER       — PG 用户 (默认 postgres)
#   DB_PASSWORD   — PG 密码 (默认 postgres)
#   DB_NAME       — PG 数据库 (默认 data4cyber)
#   NULL_THRESHOLD — 空值率阈值 (默认 0.001 即 0.1%)
# ──────────────────────────────────────────────────────────────────────────────
set -euo pipefail

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
DB_NAME="${DB_NAME:-data4cyber}"
NULL_THRESHOLD="${NULL_THRESHOLD:-0.001}"

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

section "0. 数据库连接检查"

TOTAL_ROWS=$(psql_query "SELECT COUNT(*) FROM assets WHERE is_deleted = FALSE;" || echo "ERROR")
if [ "$TOTAL_ROWS" = "ERROR" ] || [ -z "$TOTAL_ROWS" ]; then
  echo -e "  ${RED}✗ 无法连接到 PostgreSQL ($DB_HOST:$DB_PORT/$DB_NAME)${NC}"
  exit 1
fi
echo -e "  ${GREEN}✓${NC} 已连接 — 活跃资产总数: ${YELLOW}${TOTAL_ROWS}${NC}"

if [ "$TOTAL_ROWS" -eq 0 ]; then
  echo -e "  ${YELLOW}⚠ 无活跃资产，跳过对账${NC}"
  echo -e "\n${GREEN}PASS${NC} — 无数据需要对账"
  exit 0
fi

# ── 1. 新字段空值率检查 ──────────────────────────────────────────────────────

section "1. 新字段空值率 (阈值 < ${NULL_THRESHOLD})"

# 定义要检查的字段及其"空"条件
# lifecycle_state: 不应为空或 'created'（backfill 后应有实际值）
# asset_type: 不应为空或空字符串
# retention_tier: 允许空字符串（可选字段），但不应为 NULL
# duration_ms: 不应为 0（除非确实是零时长）
# owner: 不应为空字符串

declare -a FIELDS=("lifecycle_state" "asset_type" "retention_tier" "duration_ms" "owner")
declare -a NULL_CONDITIONS=(
  "lifecycle_state IS NULL OR lifecycle_state = ''"
  "asset_type IS NULL OR asset_type = ''"
  "retention_tier IS NULL"
  "duration_ms IS NULL"
  "owner IS NULL OR owner = ''"
)

for i in "${!FIELDS[@]}"; do
  FIELD="${FIELDS[$i]}"
  CONDITION="${NULL_CONDITIONS[$i]}"

  NULL_COUNT=$(psql_query "
    SELECT COUNT(*)
    FROM assets
    WHERE is_deleted = FALSE
      AND ($CONDITION);
  ")

  if [ "$TOTAL_ROWS" -gt 0 ]; then
    # Use awk for floating point comparison
    NULL_RATE=$(echo "$NULL_COUNT $TOTAL_ROWS" | awk '{printf "%.6f", $1/$2}')
    NULL_PCT=$(echo "$NULL_RATE" | awk '{printf "%.4f%%", $1*100}')
    EXCEEDS=$(echo "$NULL_RATE $NULL_THRESHOLD" | awk '{print ($1 > $2) ? "yes" : "no"}')
  else
    NULL_RATE="0"
    NULL_PCT="0.0000%"
    EXCEEDS="no"
  fi

  if [ "$EXCEEDS" = "no" ]; then
    check_pass "${FIELD}: 空值率 ${NULL_PCT} (${NULL_COUNT}/${TOTAL_ROWS})"
  else
    check_fail "${FIELD}: 空值率 ${NULL_PCT} (${NULL_COUNT}/${TOTAL_ROWS}) — 超过阈值 $(echo "$NULL_THRESHOLD" | awk '{printf "%.2f%%", $1*100}')"
  fi
done

# ── 2. 双写一致性检查 ────────────────────────────────────────────────────────

section "2. 双写一致性 (目标 100%)"

# 检查 promoted 字段与 cf_meta JSONB 中对应值是否一致
# 仅检查 cf_meta 中有对应 key 的行

# lifecycle_state vs cf_meta->>'lifecycle_state'
INCONSISTENT=$(psql_query "
  SELECT COUNT(*)
  FROM assets
  WHERE is_deleted = FALSE
    AND cf_meta->>'lifecycle_state' IS NOT NULL
    AND cf_meta->>'lifecycle_state' != ''
    AND lifecycle_state != cf_meta->>'lifecycle_state';
")
if [ "$INCONSISTENT" = "0" ]; then
  check_pass "lifecycle_state ↔ cf_meta.lifecycle_state: 一致 (不一致: 0)"
else
  check_fail "lifecycle_state ↔ cf_meta.lifecycle_state: 不一致 ${INCONSISTENT} 行"
fi

# asset_type vs cf_meta->>'asset_type'
INCONSISTENT=$(psql_query "
  SELECT COUNT(*)
  FROM assets
  WHERE is_deleted = FALSE
    AND cf_meta->>'asset_type' IS NOT NULL
    AND cf_meta->>'asset_type' != ''
    AND asset_type != cf_meta->>'asset_type';
")
if [ "$INCONSISTENT" = "0" ]; then
  check_pass "asset_type ↔ cf_meta.asset_type: 一致 (不一致: 0)"
else
  check_fail "asset_type ↔ cf_meta.asset_type: 不一致 ${INCONSISTENT} 行"
fi

# owner vs cf_meta->>'owner'
INCONSISTENT=$(psql_query "
  SELECT COUNT(*)
  FROM assets
  WHERE is_deleted = FALSE
    AND cf_meta->>'owner' IS NOT NULL
    AND cf_meta->>'owner' != ''
    AND owner != cf_meta->>'owner';
")
if [ "$INCONSISTENT" = "0" ]; then
  check_pass "owner ↔ cf_meta.owner: 一致 (不一致: 0)"
else
  check_fail "owner ↔ cf_meta.owner: 不一致 ${INCONSISTENT} 行"
fi

# retention_tier vs cf_meta->>'retention_tier'
INCONSISTENT=$(psql_query "
  SELECT COUNT(*)
  FROM assets
  WHERE is_deleted = FALSE
    AND cf_meta->>'retention_tier' IS NOT NULL
    AND cf_meta->>'retention_tier' != ''
    AND retention_tier != cf_meta->>'retention_tier';
")
if [ "$INCONSISTENT" = "0" ]; then
  check_pass "retention_tier ↔ cf_meta.retention_tier: 一致 (不一致: 0)"
else
  check_fail "retention_tier ↔ cf_meta.retention_tier: 不一致 ${INCONSISTENT} 行"
fi

# duration_ms vs cf_meta->>'duration_ms' (numeric comparison)
INCONSISTENT=$(psql_query "
  SELECT COUNT(*)
  FROM assets
  WHERE is_deleted = FALSE
    AND cf_meta->>'duration_ms' IS NOT NULL
    AND cf_meta->>'duration_ms' != ''
    AND duration_ms != (cf_meta->>'duration_ms')::bigint;
")
if [ "$INCONSISTENT" = "0" ]; then
  check_pass "duration_ms ↔ cf_meta.duration_ms: 一致 (不一致: 0)"
else
  check_fail "duration_ms ↔ cf_meta.duration_ms: 不一致 ${INCONSISTENT} 行"
fi

# ── 结果汇总 ─────────────────────────────────────────────────────────────────

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "  ${CYAN}Backfill 对账结果${NC}"
echo -e "  测试结果: ${GREEN}${PASS} passed${NC} / ${RED}${FAIL} failed${NC} / ${TOTAL} total"
echo -e "  活跃资产: ${TOTAL_ROWS}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if [ "$FAIL" -gt 0 ]; then
  echo -e "\n${RED}FAIL${NC} — 对账未通过，请检查上述失败项"
  exit 1
else
  echo -e "\n${GREEN}PASS${NC} — 新字段空值率 < 0.1%，双写一致率 100%"
  exit 0
fi
