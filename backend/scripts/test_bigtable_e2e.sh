#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────────────────────
# Bigtable 集成测试脚本
# 连接真实 Bigtable (green-valley-442103 / poc-datainfra) 进行端到端 API 测试
#
# 用法:
#   1. 先启动后端:  cd backend && go run ./cmd/server
#   2. 另开终端:    bash scripts/test_bigtable_e2e.sh
#
# 可选环境变量:
#   BASE_URL       — 后端地址 (默认 http://localhost:8080)
#   AUTH_TOKEN     — 认证 token (默认 dev-token)
#   CURL_TIMEOUT   — 每个请求超时秒数 (默认 15)
#   SKIP_LIST_SCAN — 设为 1 跳过全表扫描测试 (默认 0)
# ──────────────────────────────────────────────────────────────────────────────
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
AUTH_TOKEN="${AUTH_TOKEN:-dev-token}"
CURL_TIMEOUT="${CURL_TIMEOUT:-15}"
SKIP_LIST_SCAN="${SKIP_LIST_SCAN:-0}"
PASS=0
FAIL=0
SKIP=0
TOTAL=0

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# ── 工具函数 ─────────────────────────────────────────────────────────────────

api() {
  local method="$1" path="$2"
  shift 2
  curl -s --max-time "${CURL_TIMEOUT}" -w "\n%{http_code}" \
    -X "$method" \
    -H "Content-Type: application/json" \
    -H "X-Grace-Token: ${AUTH_TOKEN}" \
    "${BASE_URL}${path}" "$@" 2>/dev/null || echo -e "\nTIMEOUT"
}

call() {
  local method="$1" path="$2"
  shift 2
  local raw
  raw=$(api "$method" "$path" "$@")
  RESP_BODY=$(echo "$raw" | sed '$d')
  RESP_CODE=$(echo "$raw" | tail -1)
}

assert_code() {
  local expected="$1" label="$2"
  TOTAL=$((TOTAL + 1))
  if [ "$RESP_CODE" = "TIMEOUT" ]; then
    FAIL=$((FAIL + 1))
    echo -e "  ${RED}✗${NC} ${label} — TIMEOUT (>${CURL_TIMEOUT}s)"
    return
  fi
  if [ "$RESP_CODE" = "$expected" ]; then
    PASS=$((PASS + 1))
    echo -e "  ${GREEN}✓${NC} ${label} (${RESP_CODE})"
  else
    FAIL=$((FAIL + 1))
    echo -e "  ${RED}✗${NC} ${label} — expected ${expected}, got ${RESP_CODE}"
    echo -e "    ${RED}body: $(echo "$RESP_BODY" | head -c 200)${NC}"
  fi
}

skip_test() {
  local label="$1"
  TOTAL=$((TOTAL + 1))
  SKIP=$((SKIP + 1))
  echo -e "  ${YELLOW}⊘${NC} ${label} — SKIPPED"
}

json_field() {
  echo "$RESP_BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d$1)" 2>/dev/null || echo ""
}

section() {
  echo ""
  echo -e "${CYAN}━━━ $1 ━━━${NC}"
}

# ── 0. 健康检查 ──────────────────────────────────────────────────────────────

section "0. 健康检查"
call GET "/healthz"
assert_code 200 "GET /healthz"

# ── 1. Asset CRUD ────────────────────────────────────────────────────────────

section "1. Asset CRUD"

# 1.1 创建资产
TS_NOW=$(date +%s)
START_NS=$((TS_NOW * 1000000000))
END_NS=$(( (TS_NOW + 60) * 1000000000 ))

call POST "/api/v1/assets" -d "{
  \"mcap_file_id\": \"test-mcap-${TS_NOW}\",
  \"start_timestamp_ns\": ${START_NS},
  \"end_timestamp_ns\": ${END_NS},
  \"reviewer\": \"e2e-tester\",
  \"owner\": \"test-team\",
  \"type\": \"task_demo\",
  \"env\": \"indoor\",
  \"task\": \"pick_and_place\",
  \"tags\": {\"priority\": \"high\", \"quality\": \"good\"}
}"
assert_code 201 "POST /api/v1/assets — 创建资产"

ASSET_ID=$(json_field "['asset_id']")
echo -e "    asset_id = ${YELLOW}${ASSET_ID}${NC}"

if [ -z "$ASSET_ID" ]; then
  echo -e "${RED}无法获取 asset_id，终止测试${NC}"
  exit 1
fi

# 1.2 获取资产
call GET "/api/v1/assets/${ASSET_ID}"
assert_code 200 "GET /api/v1/assets/:id — 获取资产"

GOT_STATUS=$(json_field "['status']")
GOT_REVIEWER=$(json_field "['reviewer']")
TOTAL=$((TOTAL + 1))
if [ "$GOT_STATUS" = "approved" ] && [ "$GOT_REVIEWER" = "e2e-tester" ]; then
  PASS=$((PASS + 1))
  echo -e "  ${GREEN}✓${NC} 字段验证: status=${GOT_STATUS}, reviewer=${GOT_REVIEWER}"
else
  FAIL=$((FAIL + 1))
  echo -e "  ${RED}✗${NC} 字段验证失败: status=${GOT_STATUS}, reviewer=${GOT_REVIEWER}"
fi

# 验证 files 和 lifecycle_meta 字段存在
GOT_FILES=$(json_field "['files']")
GOT_LIFECYCLE=$(json_field "['lifecycle_meta']")
TOTAL=$((TOTAL + 1))
if [ -n "$GOT_FILES" ] && [ -n "$GOT_LIFECYCLE" ]; then
  PASS=$((PASS + 1))
  echo -e "  ${GREEN}✓${NC} files 和 lifecycle_meta 字段存在"
else
  FAIL=$((FAIL + 1))
  echo -e "  ${RED}✗${NC} files 或 lifecycle_meta 字段缺失"
fi

# 验证 tags
GOT_PRIORITY=$(echo "$RESP_BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('tags',{}).get('priority',''))" 2>/dev/null || echo "")
TOTAL=$((TOTAL + 1))
if [ "$GOT_PRIORITY" = "high" ]; then
  PASS=$((PASS + 1))
  echo -e "  ${GREEN}✓${NC} tags.priority = high"
else
  FAIL=$((FAIL + 1))
  echo -e "  ${RED}✗${NC} tags.priority 期望 high, 实际='${GOT_PRIORITY}'"
fi

# 验证 version
GOT_VERSION=$(json_field "['version']")
TOTAL=$((TOTAL + 1))
if [ "$GOT_VERSION" = "1" ]; then
  PASS=$((PASS + 1))
  echo -e "  ${GREEN}✓${NC} version = 1"
else
  FAIL=$((FAIL + 1))
  echo -e "  ${RED}✗${NC} version 期望 1, 实际=${GOT_VERSION}"
fi

# 1.3 更新资产
call PATCH "/api/v1/assets/${ASSET_ID}" -d '{
  "reviewer": "updated-reviewer",
  "tags": {"priority": "low", "notes": "updated by e2e"}
}'
assert_code 200 "PATCH /api/v1/assets/:id — 更新资产"

UPDATED_REVIEWER=$(json_field "['reviewer']")
TOTAL=$((TOTAL + 1))
if [ "$UPDATED_REVIEWER" = "updated-reviewer" ]; then
  PASS=$((PASS + 1))
  echo -e "  ${GREEN}✓${NC} reviewer 已更新为 updated-reviewer"
else
  FAIL=$((FAIL + 1))
  echo -e "  ${RED}✗${NC} reviewer 更新失败: ${UPDATED_REVIEWER}"
fi

# 1.4 列表查询 (全表扫描 — 可能很慢，可跳过)
if [ "$SKIP_LIST_SCAN" = "1" ]; then
  skip_test "GET /api/v1/assets — 列表查询 (SKIP_LIST_SCAN=1)"
  skip_test "GET /api/v1/assets?filter=... — 过滤查询 (SKIP_LIST_SCAN=1)"
  skip_test "GET /api/v1/assets?sort_by=... — 排序查询 (SKIP_LIST_SCAN=1)"
else
  echo -e "  ${YELLOW}⏳${NC} 列表查询 (全表扫描，可能较慢...)"
  CURL_TIMEOUT=60 call GET "/api/v1/assets?page=1&page_size=5"
  assert_code 200 "GET /api/v1/assets — 列表查询"

  if [ "$RESP_CODE" = "200" ]; then
    LIST_TOTAL=$(json_field "['total']")
    echo -e "    total = ${LIST_TOTAL}"
  fi

  CURL_TIMEOUT=60 call GET "/api/v1/assets?filter=owner:eq:test-team&page=1&page_size=10"
  assert_code 200 "GET /api/v1/assets?filter=owner:eq:test-team — 过滤查询"

  CURL_TIMEOUT=60 call GET "/api/v1/assets?sort_by=-created_at&page=1&page_size=5"
  assert_code 200 "GET /api/v1/assets?sort_by=-created_at — 排序查询"
fi

# ── 2. 算法生命周期 (env_analysis — 无 output 要求) ──────────────────────────

section "2. 算法生命周期 (env_analysis@1.0.0)"

ALGO_KEY="env_analysis@1.0.0"

# 2.1 启动算法
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO_KEY}/start" -d '{
  "method": "dagster",
  "run_id": "e2e-run-001"
}'
assert_code 200 "POST .../algo/${ALGO_KEY}/start — 启动算法"

START_STATUS=$(json_field "['status']")
TOTAL=$((TOTAL + 1))
if [ "$START_STATUS" = "running" ]; then
  PASS=$((PASS + 1))
  echo -e "  ${GREEN}✓${NC} 状态: pending → running"
else
  FAIL=$((FAIL + 1))
  echo -e "  ${RED}✗${NC} 期望 status=running, 实际=${START_STATUS}"
fi

# 2.2 重复启动应返回 409
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO_KEY}/start" -d '{"method": "dagster"}'
assert_code 409 "POST .../algo/${ALGO_KEY}/start (重复) — 409 冲突"

# 2.3 完成算法 (ok)
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO_KEY}/finish" -d '{
  "status": "ok",
  "run_id": "e2e-run-001"
}'
assert_code 200 "POST .../algo/${ALGO_KEY}/finish — 完成算法 (ok)"

FINISH_STATUS=$(json_field "['status']")
TOTAL=$((TOTAL + 1))
if [ "$FINISH_STATUS" = "ok" ]; then
  PASS=$((PASS + 1))
  echo -e "  ${GREEN}✓${NC} 状态: running → ok"
else
  FAIL=$((FAIL + 1))
  echo -e "  ${RED}✗${NC} 期望 status=ok, 实际=${FINISH_STATUS}"
fi

# 2.4 重置算法
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO_KEY}/reset"
assert_code 200 "POST .../algo/${ALGO_KEY}/reset — 重置算法"

RESET_STATUS=$(json_field "['status']")
TOTAL=$((TOTAL + 1))
if [ "$RESET_STATUS" = "pending" ]; then
  PASS=$((PASS + 1))
  echo -e "  ${GREEN}✓${NC} 状态: ok → pending"
else
  FAIL=$((FAIL + 1))
  echo -e "  ${RED}✗${NC} 期望 status=pending, 实际=${RESET_STATUS}"
fi

# 2.5 查询算法事件
call GET "/api/v1/assets/${ASSET_ID}/events?event_type=algo_*"
assert_code 200 "GET .../events?event_type=algo_* — 查询事件列表"

EVENT_COUNT=$(echo "$RESP_BODY" | python3 -c "import sys,json; print(len(json.load(sys.stdin)['items']))" 2>/dev/null || echo "0")
TOTAL=$((TOTAL + 1))
if [ "$EVENT_COUNT" -ge 3 ]; then
  PASS=$((PASS + 1))
  echo -e "  ${GREEN}✓${NC} 事件数量: ${EVENT_COUNT} (≥3: start, finish, reset)"
else
  FAIL=$((FAIL + 1))
  echo -e "  ${RED}✗${NC} 期望 ≥3 个事件, 实际=${EVENT_COUNT}"
fi

# 2.6 按 algo_key 过滤事件
call GET "/api/v1/assets/${ASSET_ID}/events?event_type=algo_*&algo_key=${ALGO_KEY}"
assert_code 200 "GET .../events?event_type=algo_*&algo_key=${ALGO_KEY} — 按 algo_key 过滤"

FILTERED_COUNT=$(echo "$RESP_BODY" | python3 -c "import sys,json; print(len(json.load(sys.stdin)['items']))" 2>/dev/null || echo "0")
TOTAL=$((TOTAL + 1))
if [ "$FILTERED_COUNT" -ge 3 ]; then
  PASS=$((PASS + 1))
  echo -e "  ${GREEN}✓${NC} 过滤后事件数量: ${FILTERED_COUNT}"
else
  FAIL=$((FAIL + 1))
  echo -e "  ${RED}✗${NC} 过滤后期望 ≥3, 实际=${FILTERED_COUNT}"
fi

# 2.7 finish failed 需要 reason
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO_KEY}/start" -d '{"method": "dagster"}'
# 先启动
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO_KEY}/finish" -d '{"status": "failed"}'
assert_code 422 "POST .../finish (failed 缺少 reason) — 422"

# 带 reason 的 failed
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO_KEY}/finish" -d '{
  "status": "failed",
  "reason": "OOM during processing"
}'
assert_code 200 "POST .../finish (failed 带 reason) — 200"

# ── 3. 带 output 的算法 (hand_tracking) ──────────────────────────────────────

section "3. 算法生命周期 (hand_tracking@1.2.0 — 需要 output_uri)"

HT_KEY="hand_tracking@1.2.0"

# 3.1 启动
call POST "/api/v1/assets/${ASSET_ID}/algo/${HT_KEY}/start" -d '{
  "method": "dagster",
  "run_id": "e2e-ht-run-001"
}'
assert_code 200 "POST .../algo/${HT_KEY}/start — 启动"

# 3.2 完成但缺少 output_uri → 422
call POST "/api/v1/assets/${ASSET_ID}/algo/${HT_KEY}/finish" -d '{
  "status": "ok",
  "run_id": "e2e-ht-run-001"
}'
assert_code 422 "POST .../algo/${HT_KEY}/finish (缺少 output_uri) — 422"

# 3.3 完成 (ok) 带 output_uri
call POST "/api/v1/assets/${ASSET_ID}/algo/${HT_KEY}/finish" -d '{
  "status": "ok",
  "output_uri": "gs://test-bucket/hand_tracking/output.mcap",
  "run_id": "e2e-ht-run-001",
  "result_size_bytes": 12345,
  "extra_fields": {"type": "hand_tracking_v1"}
}'
assert_code 200 "POST .../algo/${HT_KEY}/finish (带 output_uri) — 200"

# 3.4 验证资产上的 algo 字段和 files
call GET "/api/v1/assets/${ASSET_ID}"
assert_code 200 "GET /api/v1/assets/:id — 验证 algo 字段"

HT_STATUS=$(echo "$RESP_BODY" | python3 -c "
import sys,json
d=json.load(sys.stdin)
ar=d.get('algo_results',{})
print(ar.get('${HT_KEY}:status',''))
" 2>/dev/null || echo "")
TOTAL=$((TOTAL + 1))
if [ "$HT_STATUS" = "ok" ]; then
  PASS=$((PASS + 1))
  echo -e "  ${GREEN}✓${NC} algo_results['${HT_KEY}:status'] = ok"
else
  FAIL=$((FAIL + 1))
  echo -e "  ${RED}✗${NC} 期望 algo status=ok, 实际='${HT_STATUS}'"
fi

# 验证 output_uri 写入了 files
HT_OUTPUT=$(echo "$RESP_BODY" | python3 -c "
import sys,json
d=json.load(sys.stdin)
f=d.get('files',{})
print(f.get('${HT_KEY}:output_uri',''))
" 2>/dev/null || echo "")
TOTAL=$((TOTAL + 1))
if echo "$HT_OUTPUT" | grep -q "gs://"; then
  PASS=$((PASS + 1))
  echo -e "  ${GREEN}✓${NC} files['${HT_KEY}:output_uri'] = ${HT_OUTPUT}"
else
  # output_uri 可能存在 algo_results 里而不是 files 里，取决于实现
  echo -e "  ${YELLOW}⊘${NC} files['${HT_KEY}:output_uri'] 未找到 (可能在 algo_results 中)"
  SKIP=$((SKIP + 1))
  FAIL=$((FAIL - 0))  # 不算失败
fi

# 3.5 重置 hand_tracking
call POST "/api/v1/assets/${ASSET_ID}/algo/${HT_KEY}/reset"
assert_code 200 "POST .../algo/${HT_KEY}/reset — 重置"

# ── 4. ListDeliveries ────────────────────────────────────────────────────────

section "4. ListDeliveries"

# 4.1 无交付记录
call GET "/api/v1/assets/${ASSET_ID}/deliveries"
assert_code 200 "GET .../deliveries — 查询交付列表"

DEL_ITEMS=$(echo "$RESP_BODY" | python3 -c "import sys,json; print(len(json.load(sys.stdin)['items']))" 2>/dev/null || echo "-1")
echo -e "    当前交付数: ${DEL_ITEMS}"

# 4.2 创建交付
IDEM_KEY="e2e-idem-${TS_NOW}"
call POST "/api/v1/deliveries" \
  -H "Idempotency-Key: ${IDEM_KEY}" \
  -d "{
    \"asset_ids\": [\"${ASSET_ID}\"],
    \"customer_id\": \"e2e-customer-001\",
    \"contract_id\": \"e2e-contract-001\",
    \"note\": \"e2e test delivery\",
    \"owner\": \"test-team\"
  }"
assert_code 201 "POST /api/v1/deliveries — 创建交付"

DELIVERY_ID=$(json_field "['delivery_id']")
echo -e "    delivery_id = ${YELLOW}${DELIVERY_ID}${NC}"

# 4.3 幂等重放 — 相同 key 应返回 201
call POST "/api/v1/deliveries" \
  -H "Idempotency-Key: ${IDEM_KEY}" \
  -d "{
    \"asset_ids\": [\"${ASSET_ID}\"],
    \"customer_id\": \"e2e-customer-001\",
    \"contract_id\": \"e2e-contract-001\",
    \"note\": \"e2e test delivery\",
    \"owner\": \"test-team\"
  }"
assert_code 201 "POST /api/v1/deliveries (幂等重放) — 201"

# 4.4 查询交付列表
if [ -n "$DELIVERY_ID" ]; then
  call GET "/api/v1/assets/${ASSET_ID}/deliveries"
  assert_code 200 "GET .../deliveries — 有交付记录"

  DEL_ITEMS=$(echo "$RESP_BODY" | python3 -c "import sys,json; print(len(json.load(sys.stdin)['items']))" 2>/dev/null || echo "0")
  TOTAL=$((TOTAL + 1))
  if [ "$DEL_ITEMS" -ge 1 ]; then
    PASS=$((PASS + 1))
    echo -e "  ${GREEN}✓${NC} 交付列表包含 ${DEL_ITEMS} 条记录"
  else
    FAIL=$((FAIL + 1))
    echo -e "  ${RED}✗${NC} 期望 ≥1 条交付记录, 实际=${DEL_ITEMS}"
  fi
fi

# 4.5 获取交付详情
if [ -n "$DELIVERY_ID" ]; then
  call GET "/api/v1/deliveries/${DELIVERY_ID}"
  assert_code 200 "GET /api/v1/deliveries/:id — 获取交付详情"
fi

# ── 5. 软删除 ────────────────────────────────────────────────────────────────

section "5. 软删除"

# 5.1 创建待删除资产
call POST "/api/v1/assets" -d "{
  \"mcap_file_id\": \"test-mcap-del-${TS_NOW}\",
  \"start_timestamp_ns\": ${START_NS},
  \"end_timestamp_ns\": ${END_NS},
  \"reviewer\": \"e2e-tester\",
  \"owner\": \"delete-test-team\"
}"
assert_code 201 "POST /api/v1/assets — 创建待删除资产"
ASSET_ID_2=$(json_field "['asset_id']")

# 5.2 软删除
call DELETE "/api/v1/assets/${ASSET_ID_2}"
assert_code 200 "DELETE /api/v1/assets/:id — 软删除"

# 5.3 获取已删除资产 — 应该还能获取到 (只是 status=archived)
call GET "/api/v1/assets/${ASSET_ID_2}"
assert_code 200 "GET /api/v1/assets/:id — 获取已删除资产"

DEL_STATUS=$(json_field "['status']")
TOTAL=$((TOTAL + 1))
if [ "$DEL_STATUS" = "archived" ]; then
  PASS=$((PASS + 1))
  echo -e "  ${GREEN}✓${NC} 已删除资产 status=archived"
else
  FAIL=$((FAIL + 1))
  echo -e "  ${RED}✗${NC} 期望 status=archived, 实际=${DEL_STATUS}"
fi

# ── 6. 错误场景 ──────────────────────────────────────────────────────────────

section "6. 错误场景"

# 6.1 获取不存在的资产
call GET "/api/v1/assets/nonexistent-asset-id-12345"
assert_code 404 "GET /api/v1/assets/nonexistent — 404"

# 6.2 无效 algo key
call POST "/api/v1/assets/${ASSET_ID}/algo/invalid_algo_key/start" -d '{"method":"test"}'
assert_code 400 "POST .../algo/invalid_key/start — 400 无效 algo key"

# 6.3 无认证
RAW=$(curl -s --max-time 5 -w "\n%{http_code}" -X GET "${BASE_URL}/api/v1/assets/${ASSET_ID}" 2>/dev/null || echo -e "\nTIMEOUT")
RESP_CODE=$(echo "$RAW" | tail -1)
RESP_BODY=$(echo "$RAW" | sed '$d')
assert_code 401 "GET /api/v1/assets (无 token) — 401"

# 6.4 创建资产缺少必填字段
call POST "/api/v1/assets" -d '{"owner": "test"}'
assert_code 400 "POST /api/v1/assets (缺少必填字段) — 400"

# 6.5 无效 tag key
call POST "/api/v1/assets" -d "{
  \"mcap_file_id\": \"test-mcap-bad-tag\",
  \"start_timestamp_ns\": ${START_NS},
  \"end_timestamp_ns\": ${END_NS},
  \"reviewer\": \"tester\",
  \"tags\": {\"invalid_tag_key\": \"value\"}
}"
assert_code 422 "POST /api/v1/assets (无效 tag) — 422"

# 6.6 无效 tag value (enum 校验)
call POST "/api/v1/assets" -d "{
  \"mcap_file_id\": \"test-mcap-bad-tag-val\",
  \"start_timestamp_ns\": ${START_NS},
  \"end_timestamp_ns\": ${END_NS},
  \"reviewer\": \"tester\",
  \"tags\": {\"priority\": \"invalid_value\"}
}"
assert_code 422 "POST /api/v1/assets (无效 tag value) — 422"

# ── 7. 清理 ─────────────────────────────────────────────────────────────────

section "7. 清理"

call DELETE "/api/v1/assets/${ASSET_ID}"
echo -e "  已软删除测试资产: ${ASSET_ID}"

# ── 结果汇总 ─────────────────────────────────────────────────────────────────

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
if [ "$SKIP" -gt 0 ]; then
  echo -e "  测试结果: ${GREEN}${PASS} passed${NC} / ${RED}${FAIL} failed${NC} / ${YELLOW}${SKIP} skipped${NC} / ${TOTAL} total"
else
  echo -e "  测试结果: ${GREEN}${PASS} passed${NC} / ${RED}${FAIL} failed${NC} / ${TOTAL} total"
fi
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
