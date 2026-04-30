#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────────────────────
# Outbox Worker E2E 验证脚本
# 验证：写 asset → outbox worker 消费 → ≤60s ES 可查
#
# 对应任务: 2.18 本地 docker compose：写 event → ≤60s ES 可查
# 设计依据: outbox-worker-design.md §1.1 G1, §17 Phase 2
#
# 前置条件:
#   1. docker compose 全栈启动 (PG + ES + Backend):
#      cd deploy/local
#      docker compose -f docker-compose.all.yml up -d --build
#
#   2. 后端启用 outbox worker:
#      OUTBOX_WORKER_ENABLED=true (在 docker-compose 或 .env 中设置)
#
#   3. 等待所有服务就绪:
#      curl -sf http://localhost:8080/healthz
#      curl -sf http://localhost:9200/_cluster/health
#
# 用法:
#   bash backend/scripts/test_outbox_e2e.sh
#
# 可选环境变量:
#   BASE_URL       — 后端地址 (默认 http://localhost:8080)
#   ES_URL         — Elasticsearch 地址 (默认 http://localhost:9200)
#   AUTH_TOKEN     — 认证 token (默认 dev-token)
#   TIMEOUT_SEC    — ES 可查等待超时秒数 (默认 60)
#   CURL_TIMEOUT   — 单次 HTTP 请求超时秒数 (默认 10)
#   POLL_INTERVAL  — ES 轮询间隔秒数 (默认 2)
# ──────────────────────────────────────────────────────────────────────────────
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
ES_URL="${ES_URL:-http://localhost:9200}"
AUTH_TOKEN="${AUTH_TOKEN:-dev-token}"
TIMEOUT_SEC="${TIMEOUT_SEC:-60}"
CURL_TIMEOUT="${CURL_TIMEOUT:-10}"
POLL_INTERVAL="${POLL_INTERVAL:-2}"

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

api() {
  local method="$1" path="$2"
  shift 2
  curl -s --max-time "${CURL_TIMEOUT}" -w "\n%{http_code}" \
    -X "$method" \
    -H "Content-Type: application/json" \
    -H "X-Grace-Token: ${AUTH_TOKEN}" \
    "${BASE_URL}${path}" "$@" 2>/dev/null || echo -e "\nTIMEOUT"
}

es_api() {
  local method="$1" path="$2"
  shift 2
  curl -s --max-time "${CURL_TIMEOUT}" -w "\n%{http_code}" \
    -X "$method" \
    -H "Content-Type: application/json" \
    "${ES_URL}${path}" "$@" 2>/dev/null || echo -e "\nTIMEOUT"
}

call() {
  local method="$1" path="$2"
  shift 2
  local raw
  raw=$(api "$method" "$path" "$@")
  RESP_BODY=$(echo "$raw" | sed '$d')
  RESP_CODE=$(echo "$raw" | tail -1)
}

call_es() {
  local method="$1" path="$2"
  shift 2
  local raw
  raw=$(es_api "$method" "$path" "$@")
  RESP_BODY=$(echo "$raw" | sed '$d')
  RESP_CODE=$(echo "$raw" | tail -1)
}

assert_code() {
  local expected="$1" label="$2"
  TOTAL=$((TOTAL + 1))
  if [ "$RESP_CODE" = "TIMEOUT" ]; then
    FAIL=$((FAIL + 1))
    echo -e "  ${RED}✗${NC} ${label} — TIMEOUT (>${CURL_TIMEOUT}s)"
    return 1
  fi
  if [ "$RESP_CODE" = "$expected" ]; then
    PASS=$((PASS + 1))
    echo -e "  ${GREEN}✓${NC} ${label} (${RESP_CODE})"
    return 0
  else
    FAIL=$((FAIL + 1))
    echo -e "  ${RED}✗${NC} ${label} — expected ${expected}, got ${RESP_CODE}"
    echo -e "    ${RED}body: $(echo "$RESP_BODY" | head -c 200)${NC}"
    return 1
  fi
}

json_field() {
  echo "$RESP_BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d$1)" 2>/dev/null || echo ""
}

section() {
  echo ""
  echo -e "${CYAN}━━━ $1 ━━━${NC}"
}

# ── 0. 前置检查 ──────────────────────────────────────────────────────────────

section "0. 前置检查"

# 0.1 后端健康
call GET "/healthz"
if ! assert_code 200 "GET /healthz — 后端就绪"; then
  echo -e "${RED}后端未就绪，请先启动: docker compose -f docker-compose.all.yml up -d --build${NC}"
  exit 1
fi

# 0.2 ES 集群健康
call_es GET "/_cluster/health"
if ! assert_code 200 "GET /_cluster/health — ES 就绪"; then
  echo -e "${RED}Elasticsearch 未就绪，请检查 docker compose 状态${NC}"
  exit 1
fi

# 0.3 Outbox worker 健康 (可选，不阻塞)
call GET "/healthz/outbox"
if [ "$RESP_CODE" = "200" ]; then
  TOTAL=$((TOTAL + 1))
  PASS=$((PASS + 1))
  echo -e "  ${GREEN}✓${NC} GET /healthz/outbox — outbox worker 健康 (${RESP_CODE})"
else
  echo -e "  ${YELLOW}⚠${NC}  GET /healthz/outbox — ${RESP_CODE} (确认 OUTBOX_WORKER_ENABLED=true)"
fi

# ── 1. 创建资产 ──────────────────────────────────────────────────────────────

section "1. 创建资产 (触发 outbox event)"

TS_NOW=$(date +%s)
UNIQUE_ID="outbox-e2e-${TS_NOW}-$$"
START_NS=$((TS_NOW * 1000000000))
END_NS=$(( (TS_NOW + 120) * 1000000000 ))

call POST "/api/v1/assets" -d "{
  \"mcap_file_id\": \"${UNIQUE_ID}\",
  \"start_timestamp_ns\": ${START_NS},
  \"end_timestamp_ns\": ${END_NS},
  \"reviewer\": \"outbox-e2e-tester\",
  \"owner\": \"outbox-e2e-team\",
  \"type\": \"task_demo\",
  \"env\": \"indoor\",
  \"task\": \"pick_and_place\",
  \"tags\": {\"priority\": \"high\", \"quality\": \"good\"}
}"
if ! assert_code 201 "POST /api/v1/assets — 创建资产"; then
  echo -e "${RED}创建资产失败，终止测试${NC}"
  exit 1
fi

ASSET_ID=$(json_field "['asset_id']")
echo -e "    asset_id    = ${YELLOW}${ASSET_ID}${NC}"
echo -e "    mcap_file_id = ${YELLOW}${UNIQUE_ID}${NC}"

if [ -z "$ASSET_ID" ] || [ "$ASSET_ID" = "None" ]; then
  echo -e "${RED}无法获取 asset_id，终止测试${NC}"
  exit 1
fi

CREATE_TIME=$(date +%s)

# ── 2. 轮询 ES 等待资产出现 ──────────────────────────────────────────────────

section "2. 等待 ES 索引 (超时 ${TIMEOUT_SEC}s)"

echo -e "  ${YELLOW}⏳${NC} 轮询 ES，每 ${POLL_INTERVAL}s 检查一次..."

FOUND=0
ELAPSED=0

while [ "$ELAPSED" -lt "$TIMEOUT_SEC" ]; do
  # 方式 1: 直接按 doc_id 查 ES (outbox worker 用 asset_id 作为 doc_id)
  call_es GET "/assets/_doc/${ASSET_ID}"
  if [ "$RESP_CODE" = "200" ]; then
    ES_FOUND=$(echo "$RESP_BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('found', False))" 2>/dev/null || echo "False")
    if [ "$ES_FOUND" = "True" ]; then
      FOUND=1
      break
    fi
  fi

  # 方式 2: 通过搜索 API 查 (备选验证)
  call GET "/api/v1/search/assets?q=${UNIQUE_ID}&page=1&page_size=1"
  if [ "$RESP_CODE" = "200" ]; then
    SEARCH_TOTAL=$(echo "$RESP_BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('total', 0))" 2>/dev/null || echo "0")
    if [ "$SEARCH_TOTAL" -gt 0 ]; then
      FOUND=1
      break
    fi
  fi

  sleep "$POLL_INTERVAL"
  ELAPSED=$(( $(date +%s) - CREATE_TIME ))
  echo -e "    ${ELAPSED}s / ${TIMEOUT_SEC}s ..."
done

TOTAL=$((TOTAL + 1))
LATENCY=$(( $(date +%s) - CREATE_TIME ))

if [ "$FOUND" -eq 1 ]; then
  PASS=$((PASS + 1))
  echo -e "  ${GREEN}✓${NC} 资产在 ES 中可查 — 延迟 ${LATENCY}s (SLA ≤ 60s)"
else
  FAIL=$((FAIL + 1))
  echo -e "  ${RED}✗${NC} 资产在 ${TIMEOUT_SEC}s 内未出现在 ES 中"
  echo -e "    ${RED}可能原因:${NC}"
  echo -e "    ${RED}  - OUTBOX_WORKER_ENABLED 未设为 true${NC}"
  echo -e "    ${RED}  - outbox worker tick 间隔过长${NC}"
  echo -e "    ${RED}  - ES 索引 mapping 不兼容${NC}"
  echo -e "    ${RED}  - 查看后端日志: docker compose logs backend | grep outbox${NC}"
fi

# ── 3. 验证 ES 文档内容 ─────────────────────────────────────────────────────

if [ "$FOUND" -eq 1 ]; then
  section "3. 验证 ES 文档字段"

  call_es GET "/assets/_doc/${ASSET_ID}"
  if [ "$RESP_CODE" = "200" ]; then
    # 验证关键字段
    ES_ASSET_ID=$(echo "$RESP_BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['_source'].get('asset_id',''))" 2>/dev/null || echo "")
    ES_OWNER=$(echo "$RESP_BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['_source'].get('owner',''))" 2>/dev/null || echo "")
    ES_REVIEWER=$(echo "$RESP_BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['_source'].get('reviewer',''))" 2>/dev/null || echo "")

    TOTAL=$((TOTAL + 1))
    if [ "$ES_ASSET_ID" = "$ASSET_ID" ]; then
      PASS=$((PASS + 1))
      echo -e "  ${GREEN}✓${NC} ES doc asset_id 匹配"
    else
      FAIL=$((FAIL + 1))
      echo -e "  ${RED}✗${NC} ES doc asset_id 不匹配: expected=${ASSET_ID}, got=${ES_ASSET_ID}"
    fi

    TOTAL=$((TOTAL + 1))
    if [ "$ES_OWNER" = "outbox-e2e-team" ]; then
      PASS=$((PASS + 1))
      echo -e "  ${GREEN}✓${NC} ES doc owner = outbox-e2e-team"
    else
      FAIL=$((FAIL + 1))
      echo -e "  ${RED}✗${NC} ES doc owner: expected=outbox-e2e-team, got=${ES_OWNER}"
    fi

    TOTAL=$((TOTAL + 1))
    if [ "$ES_REVIEWER" = "outbox-e2e-tester" ]; then
      PASS=$((PASS + 1))
      echo -e "  ${GREEN}✓${NC} ES doc reviewer = outbox-e2e-tester"
    else
      FAIL=$((FAIL + 1))
      echo -e "  ${RED}✗${NC} ES doc reviewer: expected=outbox-e2e-tester, got=${ES_REVIEWER}"
    fi
  fi

  # ── 4. 验证更新也能同步到 ES ──────────────────────────────────────────────

  section "4. 验证更新同步 (PATCH → ES)"

  call PATCH "/api/v1/assets/${ASSET_ID}" -d '{
    "reviewer": "outbox-e2e-updated"
  }'
  assert_code 200 "PATCH /api/v1/assets/:id — 更新资产"

  UPDATE_TIME=$(date +%s)
  UPDATE_FOUND=0
  UPDATE_ELAPSED=0

  echo -e "  ${YELLOW}⏳${NC} 等待更新同步到 ES..."

  while [ "$UPDATE_ELAPSED" -lt "$TIMEOUT_SEC" ]; do
    call_es GET "/assets/_doc/${ASSET_ID}"
    if [ "$RESP_CODE" = "200" ]; then
      ES_UPDATED_REVIEWER=$(echo "$RESP_BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['_source'].get('reviewer',''))" 2>/dev/null || echo "")
      if [ "$ES_UPDATED_REVIEWER" = "outbox-e2e-updated" ]; then
        UPDATE_FOUND=1
        break
      fi
    fi

    sleep "$POLL_INTERVAL"
    UPDATE_ELAPSED=$(( $(date +%s) - UPDATE_TIME ))
    echo -e "    ${UPDATE_ELAPSED}s / ${TIMEOUT_SEC}s ..."
  done

  UPDATE_LATENCY=$(( $(date +%s) - UPDATE_TIME ))
  TOTAL=$((TOTAL + 1))
  if [ "$UPDATE_FOUND" -eq 1 ]; then
    PASS=$((PASS + 1))
    echo -e "  ${GREEN}✓${NC} 更新在 ES 中同步 — 延迟 ${UPDATE_LATENCY}s"
  else
    FAIL=$((FAIL + 1))
    echo -e "  ${RED}✗${NC} 更新在 ${TIMEOUT_SEC}s 内未同步到 ES"
  fi
fi

# ── 5. 清理 ──────────────────────────────────────────────────────────────────

section "5. 清理"

call DELETE "/api/v1/assets/${ASSET_ID}"
echo -e "  已软删除测试资产: ${ASSET_ID}"

# ── 结果汇总 ─────────────────────────────────────────────────────────────────

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "  ${CYAN}Outbox E2E 验证${NC}"
echo -e "  测试结果: ${GREEN}${PASS} passed${NC} / ${RED}${FAIL} failed${NC} / ${TOTAL} total"
if [ "$FOUND" -eq 1 ]; then
  echo -e "  创建 → ES 可查延迟: ${GREEN}${LATENCY}s${NC} (SLA ≤ 60s)"
fi
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
