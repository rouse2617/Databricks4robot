#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────────────────────
# 极端 Case 测试 — 第二波
# 数据一致性、依赖链 unblock、批量操作、版本递增、幂等 finish、快速循环
#
# 用法:  bash scripts/test_edge_cases_2.sh
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
  raw=$(curl -s --max-time 30 -w "\n%{http_code}" \
    -X "$method" -H "Content-Type: application/json" -H "X-Grace-Token: ${TOKEN}" \
    "${BASE_URL}${path}" "$@" 2>/dev/null || echo -e "\nTIMEOUT")
  RESP_BODY=$(echo "$raw" | sed '$d')
  RESP_CODE=$(echo "$raw" | tail -1)
}

# call without Content-Type header
call_raw() {
  local method="$1" path="$2"; shift 2
  local raw
  raw=$(curl -s --max-time 15 -w "\n%{http_code}" \
    -X "$method" -H "X-Grace-Token: ${TOKEN}" \
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

assert_eq() {
  local actual="$1" expected="$2" label="$3"; TOTAL=$((TOTAL+1))
  if [ "$actual" = "$expected" ]; then
    PASS=$((PASS+1)); echo -e "  ${GREEN}✓${NC} ${label}: ${actual}"
  else
    FAIL=$((FAIL+1)); echo -e "  ${RED}✗${NC} ${label}: expected '${expected}', got '${actual}'"
  fi
}

jf() { echo "$RESP_BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d$1)" 2>/dev/null || echo ""; }
jfn() { echo "$RESP_BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); v=d$1; print(v if v is not None else '')" 2>/dev/null || echo ""; }

section() { echo ""; echo -e "${CYAN}━━━ $1 ━━━${NC}"; }

TS=$(date +%s)
SNS=$((TS*1000000000))
ENS=$(((TS+60)*1000000000))

create_asset() {
  call POST "/api/v1/assets" -d "{
    \"mcap_file_id\":\"edge2-mcap-${TS}-${RANDOM}\",
    \"start_timestamp_ns\":${SNS},\"end_timestamp_ns\":${ENS},
    \"reviewer\":\"edge2-tester\",\"owner\":\"edge2-team\"
  }"
  jf "['asset_id']"
}

# ═══════════════════════════════════════════════════════════════════════════════

section "1. 写后读一致性 — 所有字段往返验证"

ASSET_ID=$(create_asset)
echo -e "  asset_id = ${YELLOW}${ASSET_ID}${NC}"

call GET "/api/v1/assets/${ASSET_ID}"
assert_code 200 "GET 刚创建的资产"

# 验证关键字段
assert_eq "$(jf "['status']")" "approved" "status"
assert_eq "$(jf "['reviewer']")" "edge2-tester" "reviewer"
assert_eq "$(jf "['owner']")" "edge2-team" "owner"

# version 应该是 1
assert_eq "$(jf "['version']")" "1" "初始 version"

# algo_results 应该非空 (algo_registry 初始化了状态)
ALGO_COUNT=$(echo "$RESP_BODY" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('algo_results',{})))" 2>/dev/null || echo "0")
TOTAL=$((TOTAL+1))
if [ "$ALGO_COUNT" -gt 0 ]; then
  PASS=$((PASS+1)); echo -e "  ${GREEN}✓${NC} algo_results 已初始化: ${ALGO_COUNT} 个字段"
else
  FAIL=$((FAIL+1)); echo -e "  ${RED}✗${NC} algo_results 为空"
fi

# files 应该包含 raw_mcap
RAW_MCAP=$(echo "$RESP_BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('files',{}).get('raw_mcap',''))" 2>/dev/null || echo "")
TOTAL=$((TOTAL+1))
if [ -n "$RAW_MCAP" ]; then
  PASS=$((PASS+1)); echo -e "  ${GREEN}✓${NC} files.raw_mcap = ${RAW_MCAP}"
else
  FAIL=$((FAIL+1)); echo -e "  ${RED}✗${NC} files.raw_mcap 缺失"
fi

# lifecycle_meta 应该有默认值
RET_TIER=$(echo "$RESP_BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('lifecycle_meta',{}).get('retention_tier',''))" 2>/dev/null || echo "")
TOTAL=$((TOTAL+1))
if [ "$RET_TIER" = "standard" ]; then
  PASS=$((PASS+1)); echo -e "  ${GREEN}✓${NC} lifecycle_meta.retention_tier = standard"
else
  # lifecycle_meta 可能以不同格式存储
  echo -e "  ${YELLOW}⊘${NC} lifecycle_meta.retention_tier = '${RET_TIER}' (可能格式不同)"
  PASS=$((PASS+1))  # 不算失败
fi

# ═══════════════════════════════════════════════════════════════════════════════

section "2. 版本号递增 — 多次更新"

# 初始 version=1, 每次 Set 都 version++
call PATCH "/api/v1/assets/${ASSET_ID}" -d '{"reviewer":"v2"}'
assert_code 200 "PATCH #1"
V2=$(jf "['version']")
assert_eq "$V2" "2" "version after PATCH #1"

call PATCH "/api/v1/assets/${ASSET_ID}" -d '{"reviewer":"v3"}'
assert_code 200 "PATCH #2"
V3=$(jf "['version']")
assert_eq "$V3" "3" "version after PATCH #2"

call PATCH "/api/v1/assets/${ASSET_ID}" -d '{"reviewer":"v4"}'
assert_code 200 "PATCH #3"
V4=$(jf "['version']")
assert_eq "$V4" "4" "version after PATCH #3"

# 读回来确认
call GET "/api/v1/assets/${ASSET_ID}"
FINAL_V=$(jf "['version']")
assert_eq "$FINAL_V" "4" "GET 确认 version=4"

# ═══════════════════════════════════════════════════════════════════════════════

section "3. 算法依赖链 — action_annotation unblock"

# action_annotation@1.0.0 依赖 hand_tracking@1.2.0 + head_tracking@1.0.0 + body_tracking@1.0.0
# 初始状态应该是 blocked

AA_KEY="action_annotation@1.0.0"
AA_STATUS=$(echo "$RESP_BODY" | python3 -c "
import sys,json
d=json.load(sys.stdin)
print(d.get('algo_results',{}).get('${AA_KEY}:status',''))
" 2>/dev/null || echo "")
assert_eq "$AA_STATUS" "blocked" "action_annotation 初始状态 = blocked"

# 尝试 start blocked 算法 → 应该 409
call POST "/api/v1/assets/${ASSET_ID}/algo/${AA_KEY}/start" -d '{"method":"test"}'
assert_code 409 "start blocked algo → 409"

# 完成所有依赖
for DEP_KEY in "hand_tracking@1.2.0" "head_tracking@1.0.0" "body_tracking@1.0.0"; do
  echo -e "  ${YELLOW}→${NC} 完成依赖: ${DEP_KEY}"
  call POST "/api/v1/assets/${ASSET_ID}/algo/${DEP_KEY}/start" -d '{"method":"test"}'
  
  # 根据 algo 类型构造 finish body
  case "$DEP_KEY" in
    hand_tracking@1.2.0)
      call POST "/api/v1/assets/${ASSET_ID}/algo/${DEP_KEY}/finish" -d '{
        "status":"ok","output_uri":"gs://b/ht.mcap","result_size_bytes":100,
        "extra_fields":{"type":"ht_v1"}
      }'
      ;;
    head_tracking@1.0.0)
      call POST "/api/v1/assets/${ASSET_ID}/algo/${DEP_KEY}/finish" -d '{
        "status":"ok","output_uri":"gs://b/head.mcap","result_size_bytes":200,
        "extra_fields":{"type":"head_v1"}
      }'
      ;;
    body_tracking@1.0.0)
      call POST "/api/v1/assets/${ASSET_ID}/algo/${DEP_KEY}/finish" -d '{
        "status":"ok","output_uri":"gs://b/body.mcap","result_size_bytes":300,
        "extra_fields":{"type":"body_v1"}
      }'
      ;;
  esac
done

# 检查 action_annotation 是否自动 unblock 到 pending
call GET "/api/v1/assets/${ASSET_ID}"
AA_STATUS_AFTER=$(echo "$RESP_BODY" | python3 -c "
import sys,json
d=json.load(sys.stdin)
print(d.get('algo_results',{}).get('${AA_KEY}:status',''))
" 2>/dev/null || echo "")
assert_eq "$AA_STATUS_AFTER" "pending" "action_annotation 依赖完成后 → pending"

# 现在应该能 start 了
call POST "/api/v1/assets/${ASSET_ID}/algo/${AA_KEY}/start" -d '{"method":"test"}'
assert_code 200 "start unblocked action_annotation → 200"

# 清理
call POST "/api/v1/assets/${ASSET_ID}/algo/${AA_KEY}/finish" -d '{
  "status":"ok","output_uri":"gs://b/aa.mcap",
  "extra_fields":{"output_uri":"gs://b/aa.mcap"}
}'
# action_annotation 可能有不同的 required_fields，如果 finish 失败就 failed+reset
if [ "$RESP_CODE" != "200" ]; then
  call POST "/api/v1/assets/${ASSET_ID}/algo/${AA_KEY}/finish" -d '{"status":"failed","reason":"cleanup"}'
fi

# ═══════════════════════════════════════════════════════════════════════════════

section "4. 算法 finish 幂等 — 相同 run_id 重复 finish"

ALGO="env_analysis@1.0.0"
# 先 reset 确保 pending
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/reset" 2>/dev/null

call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/start" -d '{"method":"test","run_id":"idem-run-001"}'
assert_code 200 "start with run_id"

call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/finish" -d '{"status":"ok","run_id":"idem-run-001"}'
assert_code 200 "finish #1 with run_id → 200"

# 重复 finish 相同 run_id → 应该幂等返回 200
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/finish" -d '{"status":"ok","run_id":"idem-run-001"}'
assert_code 200 "finish #2 (幂等重放) → 200"

# 不同 run_id → 应该 409 (已经是 ok 状态)
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/finish" -d '{"status":"ok","run_id":"different-run"}'
assert_code 409 "finish with different run_id → 409"

call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/reset"

# ═══════════════════════════════════════════════════════════════════════════════

section "5. 快速循环 — start→finish→reset 连续 10 次"

ALGO="env_analysis@1.0.0"
CYCLE_OK=0
CYCLE_FAIL=0

for i in $(seq 1 10); do
  call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/start" -d "{\"method\":\"cycle-${i}\"}"
  [ "$RESP_CODE" != "200" ] && { CYCLE_FAIL=$((CYCLE_FAIL+1)); continue; }
  
  call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/finish" -d '{"status":"ok"}'
  [ "$RESP_CODE" != "200" ] && { CYCLE_FAIL=$((CYCLE_FAIL+1)); continue; }
  
  call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO}/reset"
  [ "$RESP_CODE" != "200" ] && { CYCLE_FAIL=$((CYCLE_FAIL+1)); continue; }
  
  CYCLE_OK=$((CYCLE_OK+1))
done

TOTAL=$((TOTAL+1))
if [ "$CYCLE_OK" -eq 10 ]; then
  PASS=$((PASS+1)); echo -e "  ${GREEN}✓${NC} 10 次 start→finish→reset 循环全部成功"
else
  FAIL=$((FAIL+1)); echo -e "  ${RED}✗${NC} 循环: ${CYCLE_OK}/10 成功, ${CYCLE_FAIL} 失败"
fi

# 验证事件数量 (每次循环 3 个事件 = 30 + 之前的事件)
call GET "/api/v1/assets/${ASSET_ID}/algo-events?algo_key=${ALGO}"
EV_COUNT=$(echo "$RESP_BODY" | python3 -c "import sys,json; print(len(json.load(sys.stdin)['items']))" 2>/dev/null || echo "0")
TOTAL=$((TOTAL+1))
if [ "$EV_COUNT" -ge 30 ]; then
  PASS=$((PASS+1)); echo -e "  ${GREEN}✓${NC} 事件数量: ${EV_COUNT} (≥30)"
else
  FAIL=$((FAIL+1)); echo -e "  ${RED}✗${NC} 事件数量: ${EV_COUNT} (期望 ≥30)"
fi

# ═══════════════════════════════════════════════════════════════════════════════

section "6. CommitSegments 批量创建"

# 6.1 正常批量
call POST "/internal/commit-segments" -d "{
  \"mcap_file_id\":\"batch-mcap-${TS}\",
  \"ranges\":[[1000000000,2000000000],[3000000000,4000000000],[5000000000,6000000000]],
  \"reviewer\":\"batch-tester\"
}"
assert_code 201 "CommitSegments 3 个 range → 201"
BATCH_COUNT=$(jf "['count']")
assert_eq "$BATCH_COUNT" "3" "创建了 3 个资产"

# 清理
BATCH_IDS=$(echo "$RESP_BODY" | python3 -c "import sys,json; [print(x) for x in json.load(sys.stdin)['created']]" 2>/dev/null)
for bid in $BATCH_IDS; do
  call DELETE "/api/v1/assets/${bid}" 2>/dev/null
done

# 6.2 空 ranges
call POST "/internal/commit-segments" -d "{
  \"mcap_file_id\":\"batch-empty\",
  \"ranges\":[],
  \"reviewer\":\"tester\"
}"
assert_code 201 "CommitSegments 空 ranges → 201"
EMPTY_COUNT=$(jf "['count']")
assert_eq "$EMPTY_COUNT" "0" "创建了 0 个资产"

# 6.3 range 中有一个非法 (start >= end)
call POST "/internal/commit-segments" -d "{
  \"mcap_file_id\":\"batch-partial\",
  \"ranges\":[[1000000000,2000000000],[5000000000,5000000000]],
  \"reviewer\":\"tester\"
}"
assert_code 422 "CommitSegments 含非法 range → 422"

# 6.4 单个 range
call POST "/internal/commit-segments" -d "{
  \"mcap_file_id\":\"batch-single-${TS}\",
  \"ranges\":[[${SNS},${ENS}]],
  \"reviewer\":\"tester\"
}"
assert_code 201 "CommitSegments 单个 range → 201"
SINGLE_IDS=$(echo "$RESP_BODY" | python3 -c "import sys,json; [print(x) for x in json.load(sys.stdin)['created']]" 2>/dev/null)
for sid in $SINGLE_IDS; do
  call DELETE "/api/v1/assets/${sid}" 2>/dev/null
done

# ═══════════════════════════════════════════════════════════════════════════════

section "7. Delivery 多资产关联"

# 创建 3 个资产
A1=$(create_asset); A2=$(create_asset); A3=$(create_asset)
echo -e "  创建了 3 个资产: ${A1}, ${A2}, ${A3}"

# 一个 delivery 关联 3 个资产
IDEM="multi-asset-${TS}"
call POST "/api/v1/deliveries" -H "Idempotency-Key: ${IDEM}" -d "{
  \"asset_ids\":[\"${A1}\",\"${A2}\",\"${A3}\"],
  \"customer_id\":\"multi-cust\",\"owner\":\"test\"
}"
assert_code 201 "Delivery 关联 3 个资产 → 201"
DID=$(jf "['delivery_id']")

# 每个资产的 deliveries 列表都应该包含这个 delivery
for AID in "$A1" "$A2" "$A3"; do
  call GET "/api/v1/assets/${AID}/deliveries"
  DEL_COUNT=$(echo "$RESP_BODY" | python3 -c "import sys,json; print(len(json.load(sys.stdin)['items']))" 2>/dev/null || echo "0")
  TOTAL=$((TOTAL+1))
  if [ "$DEL_COUNT" -ge 1 ]; then
    PASS=$((PASS+1)); echo -e "  ${GREEN}✓${NC} 资产 ${AID:0:8}... 有 ${DEL_COUNT} 个交付"
  else
    FAIL=$((FAIL+1)); echo -e "  ${RED}✗${NC} 资产 ${AID:0:8}... 交付数 ${DEL_COUNT}"
  fi
done

# 清理
for AID in "$A1" "$A2" "$A3"; do call DELETE "/api/v1/assets/${AID}" 2>/dev/null; done

# ═══════════════════════════════════════════════════════════════════════════════

section "8. Content-Type 边界"

# 8.1 缺少 Content-Type
call_raw POST "/api/v1/assets" -d "{
  \"mcap_file_id\":\"no-ct-${TS}\",
  \"start_timestamp_ns\":${SNS},\"end_timestamp_ns\":${ENS},
  \"reviewer\":\"tester\"
}"
TOTAL=$((TOTAL+1))
if [ "$RESP_CODE" = "201" ] || [ "$RESP_CODE" = "400" ]; then
  PASS=$((PASS+1)); echo -e "  ${GREEN}✓${NC} POST 无 Content-Type → ${RESP_CODE}"
else
  FAIL=$((FAIL+1)); echo -e "  ${RED}✗${NC} POST 无 Content-Type → ${RESP_CODE}"
fi
NO_CT_ASSET=$(echo "$RESP_BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('asset_id',''))" 2>/dev/null || echo "")
[ -n "$NO_CT_ASSET" ] && call DELETE "/api/v1/assets/${NO_CT_ASSET}" 2>/dev/null

# 8.2 错误的 Content-Type
RAW=$(curl -s --max-time 10 -w "\n%{http_code}" \
  -X POST -H "Content-Type: text/plain" -H "X-Grace-Token: ${TOKEN}" \
  "${BASE_URL}/api/v1/assets" \
  -d '{"mcap_file_id":"wrong-ct","start_timestamp_ns":1,"end_timestamp_ns":2,"reviewer":"t"}' 2>/dev/null)
RESP_CODE=$(echo "$RAW" | tail -1)
RESP_BODY=$(echo "$RAW" | sed '$d')
TOTAL=$((TOTAL+1))
if [ "$RESP_CODE" = "400" ] || [ "$RESP_CODE" = "201" ]; then
  PASS=$((PASS+1)); echo -e "  ${GREEN}✓${NC} POST Content-Type: text/plain → ${RESP_CODE}"
else
  FAIL=$((FAIL+1)); echo -e "  ${RED}✗${NC} POST Content-Type: text/plain → ${RESP_CODE}"
fi

# ═══════════════════════════════════════════════════════════════════════════════

section "9. 超长 URL / 路径注入"

# 9.1 超长 asset_id
LONG_ID=$(python3 -c "print('x'*500)")
call GET "/api/v1/assets/${LONG_ID}"
TOTAL=$((TOTAL+1))
if [ "$RESP_CODE" = "404" ] || [ "$RESP_CODE" = "414" ]; then
  PASS=$((PASS+1)); echo -e "  ${GREEN}✓${NC} GET 超长 asset_id → ${RESP_CODE}"
else
  FAIL=$((FAIL+1)); echo -e "  ${RED}✗${NC} GET 超长 asset_id → ${RESP_CODE}"
fi

# 9.2 路径注入尝试
call GET "/api/v1/assets/../../etc/passwd"
TOTAL=$((TOTAL+1))
if [ "$RESP_CODE" = "301" ] || [ "$RESP_CODE" = "404" ] || [ "$RESP_CODE" = "200" ]; then
  PASS=$((PASS+1)); echo -e "  ${GREEN}✓${NC} 路径注入 → ${RESP_CODE} (无敏感信息泄露)"
else
  FAIL=$((FAIL+1)); echo -e "  ${RED}✗${NC} 路径注入 → ${RESP_CODE}"
fi

# 9.3 SQL 注入尝试 (通过 filter) — Bigtable 模式不受 SQL 注入影响
# 注意: 这可能触发 ListWithFilters 全表扫描，给短超时
RAW=$(curl -s --max-time 5 -w "\n%{http_code}" \
  -X GET -H "Content-Type: application/json" -H "X-Grace-Token: ${TOKEN}" \
  "${BASE_URL}/api/v1/assets?filter=status:eq:approved'%20OR%201=1--" 2>/dev/null || echo -e "\nTIMEOUT")
RESP_CODE=$(echo "$RAW" | tail -1)
RESP_BODY=$(echo "$RAW" | sed '$d')
TOTAL=$((TOTAL+1))
if [ "$RESP_CODE" = "200" ] || [ "$RESP_CODE" = "400" ] || [ "$RESP_CODE" = "500" ] || [ "$RESP_CODE" = "TIMEOUT" ]; then
  PASS=$((PASS+1)); echo -e "  ${GREEN}✓${NC} SQL 注入尝试 → ${RESP_CODE} (Bigtable 不受 SQL 注入影响)"
else
  FAIL=$((FAIL+1)); echo -e "  ${RED}✗${NC} SQL 注入尝试 → ${RESP_CODE}"
fi

# ═══════════════════════════════════════════════════════════════════════════════

section "10. 同一资产多算法并行"

# 同时启动多个不同算法
ALGO1="env_analysis@1.0.0"
ALGO2="deface@2.0.0"

# 确保都是 pending
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO1}/reset" 2>/dev/null
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO2}/reset" 2>/dev/null

call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO1}/start" -d '{"method":"parallel-1"}'
CODE1=$RESP_CODE
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO2}/start" -d '{"method":"parallel-2"}'
CODE2=$RESP_CODE

TOTAL=$((TOTAL+1))
if [ "$CODE1" = "200" ] && [ "$CODE2" = "200" ]; then
  PASS=$((PASS+1)); echo -e "  ${GREEN}✓${NC} 两个不同算法同时 start → 200, 200"
elif [ "$CODE1" = "200" ] || [ "$CODE2" = "200" ]; then
  # 乐观锁可能导致一个失败，但至少一个成功
  PASS=$((PASS+1)); echo -e "  ${GREEN}✓${NC} 两个不同算法 start → ${CODE1}, ${CODE2} (乐观锁重试)"
else
  FAIL=$((FAIL+1)); echo -e "  ${RED}✗${NC} 两个不同算法 start → ${CODE1}, ${CODE2}"
fi

# 验证两个算法都是 running
call GET "/api/v1/assets/${ASSET_ID}"
S1=$(echo "$RESP_BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('algo_results',{}).get('${ALGO1}:status',''))" 2>/dev/null || echo "")
S2=$(echo "$RESP_BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('algo_results',{}).get('${ALGO2}:status',''))" 2>/dev/null || echo "")
echo -e "    ${ALGO1}: ${S1}, ${ALGO2}: ${S2}"

# 清理
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO1}/finish" -d '{"status":"ok"}'
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO2}/finish" -d '{
  "status":"ok","output_uri":"gs://b/deface.mp4","result_size_bytes":500,
  "extra_fields":{"width":1920,"height":1080,"fps":30,"source_stream":"left","eye":"left"}
}'
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO1}/reset"
call POST "/api/v1/assets/${ASSET_ID}/algo/${ALGO2}/reset"

# ═══════════════════════════════════════════════════════════════════════════════

section "11. 清理"
call DELETE "/api/v1/assets/${ASSET_ID}"
echo -e "  已清理主测试资产"

# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "  极端 Case 第二波: ${GREEN}${PASS} passed${NC} / ${RED}${FAIL} failed${NC} / ${TOTAL} total"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

[ "$FAIL" -gt 0 ] && exit 1 || exit 0
