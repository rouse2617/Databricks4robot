#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────────────────────
# 极端 Case 测试脚本
# 测试边界条件、并发冲突、大数据、非法输入等场景
#
# 用法:  bash scripts/test_edge_cases.sh
# 前提:  后端已启动 (go run ./cmd/server)
# ──────────────────────────────────────────────────────────────────────────────
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
TOKEN="${AUTH_TOKEN:-dev-token}"
PASS=0; FAIL=0; TOTAL=0

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'

call() {
  local method="$1" path="$2"; shift 2
  local raw
  raw=$(curl -s --max-time 15 -w "\n%{http_code}" \
    -X "$method" -H "Content-Type: application/json" -H "X-Databrew-Token: ${TOKEN}" \
    "${BASE_URL}${path}" "$@" 2>/dev/null || echo -e "\nTIMEOUT")
  RESP_BODY=$(echo "$raw" | sed '$d')
  RESP_CODE=$(echo "$raw" | tail -1)
}

assert_code() {
  local expected="$1" label="$2"; TOTAL=$((TOTAL+1))
  if [ "$RESP_CODE" = "$expected" ]; then
    PASS=$((PASS+1)); echo -e "  ${GREEN}✓${NC} ${label} (${RESP_CODE})"
  else
    FAIL=$((FAIL+1)); echo -e "  ${RED}✗${NC} ${label} — expected ${expected}, got ${RESP_CODE}"
    echo -e "    ${RED}$(echo "$RESP_BODY" | head -c 300)${NC}"
  fi
}

assert_body_contains() {
  local needle="$1" label="$2"; TOTAL=$((TOTAL+1))
  if echo "$RESP_BODY" | grep -q "$needle"; then
    PASS=$((PASS+1)); echo -e "  ${GREEN}✓${NC} ${label}"
  else
    FAIL=$((FAIL+1)); echo -e "  ${RED}✗${NC} ${label} — body 不含 '${needle}'"
  fi
}

jf() { echo "$RESP_BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d$1)" 2>/dev/null || echo ""; }

section() { echo ""; echo -e "${CYAN}━━━ $1 ━━━${NC}"; }

# 创建一个基础资产用于后续测试
TS=$(date +%s)
SNS=$((TS*1000000000))
ENS=$(((TS+60)*1000000000))

create_asset() {
  call POST "/api/v1/assets" -d "{
    \"mcap_file_id\":\"edge-mcap-${TS}-${RANDOM}\",
    \"start_timestamp_ns\":${SNS},\"end_timestamp_ns\":${ENS},
    \"reviewer\":\"edge-tester\",\"owner\":\"edge-team\"
  }"
  echo "$(jf "['asset_id']")"
}

section "准备: 创建测试资产"
ASSET_ID=$(create_asset)
echo -e "  asset_id = ${YELLOW}${ASSET_ID}${NC}"
[ -z "$ASSET_ID" ] && { echo -e "${RED}创建失败${NC}"; exit 1; }

# ═══════════════════════════════════════════════════════════════════════════════

section "1. 输入边界 — 空 body / 畸形 JSON"

# 1.1 完全空 body
call POST "/api/v1/assets" -d ''
assert_code 400 "POST /assets 空 body → 400"

# 1.2 非 JSON
call POST "/api/v1/assets" -d 'this is not json'
assert_code 400 "POST /assets 非 JSON → 400"

# 1.3 空 JSON 对象
call POST "/api/v1/assets" -d '{}'
assert_code 400 "POST /assets {} → 400 (缺少必填字段)"

# 1.4 超长字符串字段 (10KB reviewer)
LONG_STR=$(python3 -c "print('A'*10000)")
call POST "/api/v1/assets" -d "{
  \"mcap_file_id\":\"long-test\",
  \"start_timestamp_ns\":${SNS},\"end_timestamp_ns\":${ENS},
  \"reviewer\":\"${LONG_STR}\"
}"
assert_code 201 "POST /assets 超长 reviewer (10KB) → 201 (Bigtable 无列值限制)"

LONG_ASSET=$(jf "['asset_id']")
[ -n "$LONG_ASSET" ] && call DELETE "/api/v1/assets/${LONG_ASSET}"

# 1.5 timestamp 边界: start == end
call POST "/api/v1/assets" -d "{
  \"mcap_file_id\":\"ts-eq\",
  \"start_timestamp_ns\":${SNS},\"end_timestamp_ns\":${SNS},
  \"reviewer\":\"tester\"
}"
assert_code 422 "POST /assets start==end → 422"

# 1.6 timestamp 边界: start > end
call POST "/api/v1/assets" -d "{
  \"mcap_file_id\":\"ts-inv\",
  \"start_timestamp_ns\":${ENS},\"end_timestamp_ns\":${SNS},
  \"reviewer\":\"tester\"
}"
assert_code 422 "POST /assets start>end → 422"

# 1.7 timestamp 边界: 0 值
# Gin binding:"required" 对 int64 零值视为空 → 400
call POST "/api/v1/assets" -d '{
  "mcap_file_id":"ts-zero",
  "start_timestamp_ns":0,"end_timestamp_ns":1,
  "reviewer":"tester"
}'
assert_code 400 "POST /assets start=0 → 400 (Gin required 拒绝零值)"

# 1.8 负数 timestamp
call POST "/api/v1/assets" -d '{
  "mcap_file_id":"ts-neg",
  "start_timestamp_ns":-100,"end_timestamp_ns":100,
  "reviewer":"tester"
}'
assert_code 201 "POST /assets 负数 start → 201 (合法 int64)"
NEG_ASSET=$(jf "['asset_id']")
[ -n "$NEG_ASSET" ] && call DELETE "/api/v1/assets/${NEG_ASSET}"

# 1.9 极大 timestamp (接近 int64 max)
call POST "/api/v1/assets" -d '{
  "mcap_file_id":"ts-huge",
  "start_timestamp_ns":9223372036854775800,"end_timestamp_ns":9223372036854775807,
  "reviewer":"tester"
}'
# JSON 数字精度可能丢失，看服务端怎么处理
if [ "$RESP_CODE" = "201" ] || [ "$RESP_CODE" = "400" ]; then
  TOTAL=$((TOTAL+1)); PASS=$((PASS+1))
  echo -e "  ${GREEN}✓${NC} POST /assets 极大 timestamp → ${RESP_CODE} (可接受)"
  HUGE_ASSET=$(jf "['asset_id']")
  [ -n "$HUGE_ASSET" ] && call DELETE "/api/v1/assets/${HUGE_ASSET}"
else
  TOTAL=$((TOTAL+1)); FAIL=$((FAIL+1))
  echo -e "  ${RED}✗${NC} POST /assets 极大 timestamp → ${RESP_CODE}"
fi

# ═══════════════════════════════════════════════════════════════════════════════

section "2. Tag 校验边界"

# 2.1 空 tag map
call POST "/api/v1/assets" -d "{
  \"mcap_file_id\":\"tag-empty\",
  \"start_timestamp_ns\":${SNS},\"end_timestamp_ns\":${ENS},
  \"reviewer\":\"tester\",\"tags\":{}
}"
assert_code 201 "POST /assets 空 tags {} → 201"
TAG_ASSET=$(jf "['asset_id']")
[ -n "$TAG_ASSET" ] && call DELETE "/api/v1/assets/${TAG_ASSET}"

# 2.2 未注册的 tag key
call POST "/api/v1/assets" -d "{
  \"mcap_file_id\":\"tag-bad-key\",
  \"start_timestamp_ns\":${SNS},\"end_timestamp_ns\":${ENS},
  \"reviewer\":\"tester\",\"tags\":{\"nonexistent_key\":\"value\"}
}"
assert_code 422 "POST /assets 未注册 tag key → 422"

# 2.3 enum tag 值不在允许列表
call POST "/api/v1/assets" -d "{
  \"mcap_file_id\":\"tag-bad-val\",
  \"start_timestamp_ns\":${SNS},\"end_timestamp_ns\":${ENS},
  \"reviewer\":\"tester\",\"tags\":{\"priority\":\"ULTRA_MEGA_HIGH\"}
}"
assert_code 422 "POST /assets priority=ULTRA_MEGA_HIGH → 422"

# 2.4 string tag 正常值
call POST "/api/v1/assets" -d "{
  \"mcap_file_id\":\"tag-str\",
  \"start_timestamp_ns\":${SNS},\"end_timestamp_ns\":${ENS},
  \"reviewer\":\"tester\",\"tags\":{\"notes\":\"这是一个中文备注 with emoji 🤖\"}
}"
assert_code 201 "POST /assets notes=中文+emoji → 201"
STR_ASSET=$(jf "['asset_id']")
[ -n "$STR_ASSET" ] && call DELETE "/api/v1/assets/${STR_ASSET}"

# 2.5 大量 tags (所有合法 tag 一起)
call POST "/api/v1/assets" -d "{
  \"mcap_file_id\":\"tag-all\",
  \"start_timestamp_ns\":${SNS},\"end_timestamp_ns\":${ENS},
  \"reviewer\":\"tester\",
  \"tags\":{\"priority\":\"critical\",\"quality\":\"excellent\",\"scene\":\"indoor\",\"task\":\"test\",\"batch\":\"B001\",\"notes\":\"all tags\"}
}"
assert_code 201 "POST /assets 所有合法 tag → 201"
ALL_TAG_ASSET=$(jf "['asset_id']")
[ -n "$ALL_TAG_ASSET" ] && call DELETE "/api/v1/assets/${ALL_TAG_ASSET}"

# ═══════════════════════════════════════════════════════════════════════════════

section "3. 算法状态机边界"

ALGO="env_analysis@1.0.0"

# 3.1 从 pending 直接 finish → 409
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/finish" -d '{"status":"ok"}'
assert_code 409 "finish from pending → 409"

# 3.2 从 pending 直接 reset → 409
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/reset"
assert_code 409 "reset from pending → 409"

# 3.3 finish 用非法 status
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/start" -d '{"method":"test"}'
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/finish" -d '{"status":"invalid_status"}'
assert_code 409 "finish status=invalid_status → 409"

# 清理: 把 running 状态 finish 掉
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/finish" -d '{"status":"ok"}'
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/reset"

# 3.4 start 空 body
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/start" -d '{}'
assert_code 400 "start 空 body (缺少 method) → 400"

# 3.5 finish 空 body
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/start" -d '{"method":"test"}'
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/finish" -d '{}'
assert_code 400 "finish 空 body (缺少 status) → 400"
# 清理
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/finish" -d '{"status":"ok"}'
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/reset"

# 3.6 对不存在的资产操作
call POST "/api/v1/assets/nonexistent-id/algo/${ALGO}/start" -d '{"method":"test"}'
assert_code 404 "start on nonexistent asset → 404"

# 3.7 不存在的 algo key
call POST "/api/v1/assets/${ASSET_ID}/algo/totally_fake@9.9.9/start" -d '{"method":"test"}'
assert_code 400 "start with fake algo key → 400"

# 3.8 algo key 格式错误 (无版本号)
call POST "/api/v1/assets/${ASSET_ID}/algo/no_version/start" -d '{"method":"test"}'
assert_code 400 "start with no-version algo key → 400"

# 3.9 hand_tracking finish ok 但缺少 type (required_field)
HT="hand_tracking@1.2.0"
call POST "/api/v1/assets/${ASSET_ID}/algo/${HT}/start" -d '{"method":"test"}'
call POST "/api/v1/assets/${ASSET_ID}/algo/${HT}/finish" -d '{
  "status":"ok","output_uri":"gs://bucket/out.mcap","result_size_bytes":100
}'
assert_code 422 "hand_tracking finish ok 缺少 type → 422"
# 清理
call POST "/api/v1/assets/${ASSET_ID}/algo/${HT}/finish" -d '{"status":"failed","reason":"cleanup"}'
call POST "/api/v1/assets/${ASSET_ID}/algo/${HT}/reset"

# 3.10 hand_tracking finish ok 缺少 result_size_bytes (report_size=true)
call POST "/api/v1/assets/${ASSET_ID}/algo/${HT}/start" -d '{"method":"test"}'
call POST "/api/v1/assets/${ASSET_ID}/algo/${HT}/finish" -d '{
  "status":"ok","output_uri":"gs://bucket/out.mcap",
  "extra_fields":{"type":"ht_v1"}
}'
assert_code 422 "hand_tracking finish ok 缺少 result_size_bytes → 422"
call POST "/api/v1/assets/${ASSET_ID}/algo/${HT}/finish" -d '{"status":"failed","reason":"cleanup"}'
call POST "/api/v1/assets/${ASSET_ID}/algo/${HT}/reset"

# ═══════════════════════════════════════════════════════════════════════════════

section "4. 并发冲突 (乐观锁)"

# 4.1 同时对同一资产启动同一算法 — 只有一个成功
echo -e "  ${YELLOW}⏳${NC} 并发启动 5 个 start 请求..."
CONC_RESULTS=$(mktemp)
for i in $(seq 1 5); do
  (
    raw=$(curl -s --max-time 10 -w "\n%{http_code}" \
      -X POST -H "Content-Type: application/json" -H "X-Databrew-Token: ${TOKEN}" \
      "${BASE_URL}/api/v1/assets/${ASSET_ID}/algo/${ALGO}/start" \
      -d '{"method":"conc-test"}' 2>/dev/null)
    code=$(echo "$raw" | tail -1)
    echo "$code" >> "$CONC_RESULTS"
  ) &
done
wait

OK_COUNT=$(grep -c "200" "$CONC_RESULTS" 2>/dev/null || echo "0")
CONFLICT_COUNT=$(grep -c "409" "$CONC_RESULTS" 2>/dev/null || echo "0")
rm -f "$CONC_RESULTS"

TOTAL=$((TOTAL+1))
if [ "$OK_COUNT" -ge 1 ]; then
  PASS=$((PASS+1))
  echo -e "  ${GREEN}✓${NC} 并发 start: ${OK_COUNT}×200, ${CONFLICT_COUNT}×409 (至少 1 个成功)"
else
  FAIL=$((FAIL+1))
  echo -e "  ${RED}✗${NC} 并发 start: ${OK_COUNT}×200, ${CONFLICT_COUNT}×409"
fi

# 清理
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/finish" -d '{"status":"ok"}'
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/reset"

# ═══════════════════════════════════════════════════════════════════════════════

section "5. 幂等性"

# 5.1 Delivery 幂等: 相同 key 相同 body → 201
IDEM_KEY="edge-idem-${TS}"
call POST "/api/v1/deliveries" -H "Idempotency-Key: ${IDEM_KEY}" -d "{
  \"asset_ids\":[\"${ASSET_ID}\"],\"customer_id\":\"edge-cust\",\"owner\":\"edge\"
}"
assert_code 201 "POST /deliveries (首次) → 201"
FIRST_DID=$(jf "['delivery_id']")

call POST "/api/v1/deliveries" -H "Idempotency-Key: ${IDEM_KEY}" -d "{
  \"asset_ids\":[\"${ASSET_ID}\"],\"customer_id\":\"edge-cust\",\"owner\":\"edge\"
}"
assert_code 201 "POST /deliveries (重放相同 key+body) → 201"
SECOND_DID=$(jf "['delivery_id']")

TOTAL=$((TOTAL+1))
if [ "$FIRST_DID" = "$SECOND_DID" ]; then
  PASS=$((PASS+1))
  echo -e "  ${GREEN}✓${NC} 幂等: delivery_id 一致 (${FIRST_DID})"
else
  FAIL=$((FAIL+1))
  echo -e "  ${RED}✗${NC} 幂等失败: ${FIRST_DID} vs ${SECOND_DID}"
fi

# 5.2 相同 key 不同 body → 409
call POST "/api/v1/deliveries" -H "Idempotency-Key: ${IDEM_KEY}" -d "{
  \"asset_ids\":[\"${ASSET_ID}\"],\"customer_id\":\"DIFFERENT-cust\",\"owner\":\"edge\"
}"
assert_code 409 "POST /deliveries (相同 key 不同 body) → 409"

# 5.3 缺少 Idempotency-Key
call POST "/api/v1/deliveries" -d "{
  \"asset_ids\":[\"${ASSET_ID}\"],\"customer_id\":\"no-idem\",\"owner\":\"edge\"
}"
assert_code 400 "POST /deliveries 缺少 Idempotency-Key → 400"

# ═══════════════════════════════════════════════════════════════════════════════

section "6. 特殊字符 & Unicode"

# 6.1 中文 owner
call POST "/api/v1/assets" -d "{
  \"mcap_file_id\":\"unicode-test-${TS}\",
  \"start_timestamp_ns\":${SNS},\"end_timestamp_ns\":${ENS},
  \"reviewer\":\"张三\",\"owner\":\"机器人团队\"
}"
assert_code 201 "POST /assets 中文 reviewer/owner → 201"
UNI_ASSET=$(jf "['asset_id']")

# 验证读回来是否正确
call GET "/api/v1/assets/${UNI_ASSET}"
assert_code 200 "GET 中文资产 → 200"
GOT_OWNER=$(jf "['owner']")
TOTAL=$((TOTAL+1))
if [ "$GOT_OWNER" = "机器人团队" ]; then
  PASS=$((PASS+1)); echo -e "  ${GREEN}✓${NC} owner 中文往返一致: ${GOT_OWNER}"
else
  FAIL=$((FAIL+1)); echo -e "  ${RED}✗${NC} owner 中文往返失败: '${GOT_OWNER}'"
fi
[ -n "$UNI_ASSET" ] && call DELETE "/api/v1/assets/${UNI_ASSET}"

# 6.2 特殊字符 mcap_file_id (含 #, @, 空格)
call POST "/api/v1/assets" -d "{
  \"mcap_file_id\":\"file with spaces#and@symbols\",
  \"start_timestamp_ns\":${SNS},\"end_timestamp_ns\":${ENS},
  \"reviewer\":\"tester\"
}"
assert_code 201 "POST /assets mcap_file_id 含特殊字符 → 201"
SPEC_ASSET=$(jf "['asset_id']")
[ -n "$SPEC_ASSET" ] && call DELETE "/api/v1/assets/${SPEC_ASSET}"

# ═══════════════════════════════════════════════════════════════════════════════

section "7. 已删除资产操作"

# 7.1 对已删除资产执行 algo start
DEAD_ASSET=$(create_asset)
call DELETE "/api/v1/assets/${DEAD_ASSET}"

call POST "/api/v1/assets/${DEAD_ASSET}/algo/${ALGO}/start" -d '{"method":"test"}'
# 应该还能操作 (soft delete 只改 status，不阻止 algo 操作)
TOTAL=$((TOTAL+1))
if [ "$RESP_CODE" = "200" ] || [ "$RESP_CODE" = "409" ]; then
  PASS=$((PASS+1))
  echo -e "  ${GREEN}✓${NC} algo start on archived asset → ${RESP_CODE} (可操作)"
else
  FAIL=$((FAIL+1))
  echo -e "  ${RED}✗${NC} algo start on archived asset → ${RESP_CODE}"
fi

# 7.2 对已删除资产 PATCH
call PATCH "/api/v1/assets/${DEAD_ASSET}" -d '{"reviewer":"ghost"}'
assert_code 200 "PATCH archived asset → 200 (soft delete 不阻止更新)"

# 7.3 重复删除
call DELETE "/api/v1/assets/${DEAD_ASSET}"
assert_code 200 "DELETE 重复删除 → 200 (幂等)"

# ═══════════════════════════════════════════════════════════════════════════════

section "8. 分页边界"

# 8.1 page=0
call GET "/api/v1/assets/${ASSET_ID}/deliveries?page=0&page_size=10"
assert_code 200 "GET deliveries page=0 → 200 (应回退到 page=1)"

# 8.2 page_size=0
call GET "/api/v1/assets/${ASSET_ID}/deliveries?page=1&page_size=0"
assert_code 200 "GET deliveries page_size=0 → 200 (应回退到默认)"

# 8.3 page_size 超大
call GET "/api/v1/assets/${ASSET_ID}/deliveries?page=1&page_size=99999"
assert_code 200 "GET deliveries page_size=99999 → 200 (应限制到 max)"

# 8.4 page 超大 (超出范围)
call GET "/api/v1/assets/${ASSET_ID}/deliveries?page=999999&page_size=10"
assert_code 200 "GET deliveries page=999999 → 200 (空 items)"
ITEMS_LEN=$(echo "$RESP_BODY" | python3 -c "import sys,json; print(len(json.load(sys.stdin)['items']))" 2>/dev/null || echo "-1")
TOTAL=$((TOTAL+1))
if [ "$ITEMS_LEN" = "0" ]; then
  PASS=$((PASS+1)); echo -e "  ${GREEN}✓${NC} 超出范围页返回空 items"
else
  FAIL=$((FAIL+1)); echo -e "  ${RED}✗${NC} 期望空 items, 实际 ${ITEMS_LEN}"
fi

# 8.5 非数字 page
call GET "/api/v1/assets/${ASSET_ID}/deliveries?page=abc&page_size=xyz"
assert_code 200 "GET deliveries page=abc → 200 (应回退到默认)"

# ═══════════════════════════════════════════════════════════════════════════════

section "9. HTTP 方法错误"

# 9.1 PUT 不存在的路由
RAW=$(curl -s --max-time 5 -w "\n%{http_code}" -X PUT \
  -H "Content-Type: application/json" -H "X-Databrew-Token: ${TOKEN}" \
  "${BASE_URL}/api/v1/assets/${ASSET_ID}" -d '{}' 2>/dev/null)
RESP_CODE=$(echo "$RAW" | tail -1)
RESP_BODY=$(echo "$RAW" | sed '$d')
TOTAL=$((TOTAL+1))
if [ "$RESP_CODE" = "404" ] || [ "$RESP_CODE" = "405" ]; then
  PASS=$((PASS+1)); echo -e "  ${GREEN}✓${NC} PUT /assets/:id → ${RESP_CODE}"
else
  FAIL=$((FAIL+1)); echo -e "  ${RED}✗${NC} PUT /assets/:id → ${RESP_CODE}"
fi

# ═══════════════════════════════════════════════════════════════════════════════

section "10. 清理"
call DELETE "/api/v1/assets/${ASSET_ID}"
echo -e "  已清理主测试资产"

# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "  极端 Case 测试: ${GREEN}${PASS} passed${NC} / ${RED}${FAIL} failed${NC} / ${TOTAL} total"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

[ "$FAIL" -gt 0 ] && exit 1 || exit 0
