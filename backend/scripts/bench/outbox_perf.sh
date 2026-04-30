#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────────────────────
# outbox_perf.sh — Outbox Worker 性能压测
#
# 对应任务: 9.2 P1-T-2: Outbox 性能压测（50 events/s × 60s → P99 ≤ 60s）
#
# 测试流程:
#   1. 以 50 events/s 的速率持续创建资产 60 秒（共 3000 events）
#   2. 等待所有 events 被 outbox worker 消费并同步到 ES
#   3. 计算每个 event 从创建到 ES 可查的延迟
#   4. 报告 P50/P95/P99 延迟（目标 P99 ≤ 60s）
#
# 前置条件:
#   - docker compose 全栈启动 (PG + ES + Backend with OUTBOX_WORKER_ENABLED=true)
#   - curl + jq + python3 已安装
#
# 用法:
#   bash backend/scripts/bench/outbox_perf.sh
#
# 可选环境变量:
#   BASE_URL         — 后端地址 (默认 http://localhost:8080)
#   ES_URL           — Elasticsearch 地址 (默认 http://localhost:9200)
#   AUTH_TOKEN       — 认证 token (默认 dev-token)
#   EVENTS_PER_SEC   — 每秒事件数 (默认 50)
#   DURATION_SEC     — 持续秒数 (默认 60)
#   ES_TIMEOUT_SEC   — ES 同步等待超时 (默认 120)
#   P99_TARGET_SEC   — P99 延迟目标 (默认 60)
# ──────────────────────────────────────────────────────────────────────────────
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
ES_URL="${ES_URL:-http://localhost:9200}"
AUTH_TOKEN="${AUTH_TOKEN:-dev-token}"
EVENTS_PER_SEC="${EVENTS_PER_SEC:-50}"
DURATION_SEC="${DURATION_SEC:-60}"
ES_TIMEOUT_SEC="${ES_TIMEOUT_SEC:-120}"
P99_TARGET_SEC="${P99_TARGET_SEC:-60}"

TOTAL_EVENTS=$((EVENTS_PER_SEC * DURATION_SEC))
INTERVAL_US=$(( 1000000 / EVENTS_PER_SEC ))  # microseconds between events
BENCH_RUN_ID=$(date +%s)
BENCH_OWNER="bench-team-${BENCH_RUN_ID}"
BENCH_REVIEWER="bench-perf-${BENCH_RUN_ID}"

# ── 颜色 ─────────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

section() {
  echo ""
  echo -e "${CYAN}━━━ $1 ━━━${NC}"
}

# ── 临时文件 ─────────────────────────────────────────────────────────────────
ASSET_IDS_FILE=$(mktemp)
CREATE_TIMES_FILE=$(mktemp)
LATENCIES_FILE=$(mktemp)
trap 'rm -f "$ASSET_IDS_FILE" "$CREATE_TIMES_FILE" "$LATENCIES_FILE"' EXIT

# ── 0. 前置检查 ──────────────────────────────────────────────────────────────

section "0. 前置检查"

echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "  Outbox 性能压测"
echo -e "  速率: ${YELLOW}${EVENTS_PER_SEC} events/s${NC}"
echo -e "  持续: ${YELLOW}${DURATION_SEC}s${NC}"
echo -e "  总量: ${YELLOW}${TOTAL_EVENTS} events${NC}"
echo -e "  P99 目标: ${YELLOW}≤ ${P99_TARGET_SEC}s${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Health check
HTTP_CODE=$(curl -sf -o /dev/null -w "%{http_code}" "${BASE_URL}/healthz" 2>/dev/null || echo "000")
if [ "$HTTP_CODE" != "200" ]; then
  echo -e "  ${RED}✗ 后端未就绪 (${BASE_URL}/healthz → ${HTTP_CODE})${NC}"
  exit 1
fi
echo -e "  ${GREEN}✓${NC} 后端就绪"

ES_CODE=$(curl -sf -o /dev/null -w "%{http_code}" "${ES_URL}/_cluster/health" 2>/dev/null || echo "000")
if [ "$ES_CODE" != "200" ]; then
  echo -e "  ${RED}✗ Elasticsearch 未就绪 (${ES_URL} → ${ES_CODE})${NC}"
  exit 1
fi
echo -e "  ${GREEN}✓${NC} Elasticsearch 就绪"

MCAP_FILE_ID=$(curl -sf "${BASE_URL}/api/v1/mcap-files?page=1&page_size=1" \
  -H "X-Grace-Token: ${AUTH_TOKEN}" 2>/dev/null \
  | python3 -c "import sys, json; print(json.load(sys.stdin)['items'][0]['mcap_file_id'])" 2>/dev/null || echo "")
if [ -z "$MCAP_FILE_ID" ]; then
  echo -e "  ${RED}✗ 无法获取可用 mcap_file_id${NC}"
  exit 1
fi
echo -e "  ${GREEN}✓${NC} 使用 mcap_file_id=${MCAP_FILE_ID}"

# ── 1. 生成事件 ──────────────────────────────────────────────────────────────

section "1. 生成 ${TOTAL_EVENTS} 个事件 (${EVENTS_PER_SEC}/s × ${DURATION_SEC}s)"

CREATED=0
FAILED=0
BENCH_START=$BENCH_RUN_ID

for sec in $(seq 1 "$DURATION_SEC"); do
  SEC_START=$(python3 -c "import time; print(time.time())")

  for i in $(seq 1 "$EVENTS_PER_SEC"); do
    TS_NS=$(python3 -c "import time; print(int(time.time() * 1e9))")
    END_NS=$((TS_NS + 120000000000))
    CREATE_TS=$(python3 -c "import time; print(time.time())")

    RESP=$(curl -sf -w "\n%{http_code}" \
      -X POST "${BASE_URL}/api/v1/assets" \
      -H "Content-Type: application/json" \
      -H "X-Grace-Token: ${AUTH_TOKEN}" \
      -d "{
        \"mcap_file_id\": \"${MCAP_FILE_ID}\",
        \"start_timestamp_ns\": ${TS_NS},
        \"end_timestamp_ns\": ${END_NS},
        \"reviewer\": \"${BENCH_REVIEWER}\",
        \"owner\": \"${BENCH_OWNER}\",
        \"type\": \"task_demo\",
        \"env\": \"indoor\",
        \"task\": \"navigation\"
      }" 2>/dev/null || echo -e "\n000")

    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')

    if [ "$HTTP_CODE" = "201" ]; then
      ASSET_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['asset_id'])" 2>/dev/null || echo "")
      if [ -n "$ASSET_ID" ]; then
        echo "$ASSET_ID" >> "$ASSET_IDS_FILE"
        echo "${ASSET_ID} ${CREATE_TS}" >> "$CREATE_TIMES_FILE"
        CREATED=$((CREATED + 1))
      else
        FAILED=$((FAILED + 1))
      fi
    else
      FAILED=$((FAILED + 1))
    fi
  done

  # Rate limiting: sleep until the second is up
  SEC_ELAPSED=$(python3 -c "import time; print(time.time() - ${SEC_START})")
  SLEEP_TIME=$(python3 -c "s = 1.0 - ${SEC_ELAPSED}; print(max(0, s))")
  if python3 -c "exit(0 if ${SLEEP_TIME} > 0.01 else 1)"; then
    sleep "$SLEEP_TIME"
  fi

  # Progress every 10 seconds
  if [ $((sec % 10)) -eq 0 ]; then
    echo -e "  ${YELLOW}${sec}/${DURATION_SEC}s${NC} — 已创建 ${CREATED}, 失败 ${FAILED}"
  fi
done

GENERATE_ELAPSED=$(( $(date +%s) - BENCH_START ))
echo ""
echo -e "  ${GREEN}✓${NC} 生成完成: ${YELLOW}${CREATED}${NC} 成功, ${RED}${FAILED}${NC} 失败, 耗时 ${GENERATE_ELAPSED}s"
echo -e "  实际速率: $(python3 -c "print(f'{${CREATED}/${GENERATE_ELAPSED}:.1f}')" 2>/dev/null || echo "N/A") events/s"

if [ "$CREATED" -eq 0 ]; then
  echo -e "  ${RED}没有成功创建任何事件，终止${NC}"
  exit 1
fi

# ── 2. 等待所有事件同步到 ES ─────────────────────────────────────────────────

section "2. 等待 ES 同步 (超时 ${ES_TIMEOUT_SEC}s)"

SYNC_START=$(date +%s)
SYNCED=0
POLL_INTERVAL=5

while true; do
  ELAPSED=$(( $(date +%s) - SYNC_START ))
  if [ "$ELAPSED" -ge "$ES_TIMEOUT_SEC" ]; then
    echo -e "  ${RED}✗ 超时 — ${SYNCED}/${CREATED} 已同步${NC}"
    break
  fi

  # Refresh ES index
  curl -sf -X POST "${ES_URL}/assets/_refresh" > /dev/null 2>&1 || true

  # Count how many of our bench assets are in ES
  SYNCED=$(curl -sf -X POST "${ES_URL}/assets/_count" \
    -H "Content-Type: application/json" \
    -d "{\"query\":{\"term\":{\"owner.keyword\":\"${BENCH_OWNER}\"}}}" 2>/dev/null \
    | python3 -c "import sys,json; print(json.load(sys.stdin).get('count',0))" 2>/dev/null || echo "0")

  echo -e "  ${ELAPSED}s — ${YELLOW}${SYNCED}/${CREATED}${NC} 已同步到 ES"

  if [ "$SYNCED" -ge "$CREATED" ]; then
    echo -e "  ${GREEN}✓${NC} 全部同步完成 — 耗时 ${ELAPSED}s"
    break
  fi

  sleep "$POLL_INTERVAL"
done

SYNC_ELAPSED=$(( $(date +%s) - SYNC_START ))

# ── 3. 计算逐条延迟 ─────────────────────────────────────────────────────────

section "3. 计算延迟分布"

# For each asset, check when it appeared in ES (approximate: use sync completion time)
# Since we can't get per-doc indexing time from ES easily, we measure:
# - Per-batch: time from last event creation to all events synced
# - Overall P99 approximation: sync_elapsed is the upper bound for the last batch
#
# More precise: sample individual assets to measure actual latency
SAMPLE_SIZE=100
if [ "$CREATED" -lt "$SAMPLE_SIZE" ]; then
  SAMPLE_SIZE="$CREATED"
fi

echo -e "  采样 ${SAMPLE_SIZE} 个资产测量逐条延迟..."

# Take a random sample of asset IDs
SAMPLE_FILE=$(mktemp)
shuf -n "$SAMPLE_SIZE" "$ASSET_IDS_FILE" > "$SAMPLE_FILE" 2>/dev/null || head -n "$SAMPLE_SIZE" "$ASSET_IDS_FILE" > "$SAMPLE_FILE"

CHECK_TS=$(python3 -c "import time; print(time.time())")

while IFS= read -r aid; do
  # Look up create time
  CREATE_TS=$(grep "^${aid} " "$CREATE_TIMES_FILE" | awk '{print $2}')
  if [ -z "$CREATE_TS" ]; then
    continue
  fi

  # Check if in ES
  ES_RESP=$(curl -sf "${ES_URL}/assets/_doc/${aid}" 2>/dev/null || echo '{"found":false}')
  FOUND=$(echo "$ES_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('found',False))" 2>/dev/null || echo "False")

  if [ "$FOUND" = "True" ]; then
    # Latency = check_time - create_time (upper bound)
    LATENCY=$(python3 -c "print(f'{${CHECK_TS} - ${CREATE_TS}:.2f}')")
    echo "$LATENCY" >> "$LATENCIES_FILE"
  fi
done < "$SAMPLE_FILE"
rm -f "$SAMPLE_FILE"

MEASURED=$(wc -l < "$LATENCIES_FILE" | tr -d ' ')

if [ "$MEASURED" -gt 0 ]; then
  # Calculate percentiles using Python
  STATS=$(python3 -c "
import sys

latencies = sorted(float(l.strip()) for l in open('${LATENCIES_FILE}') if l.strip())
n = len(latencies)
if n == 0:
    print('0 0 0 0 0 0')
    sys.exit()

avg = sum(latencies) / n
p50 = latencies[int(n * 0.50)]
p90 = latencies[int(n * 0.90)]
p95 = latencies[int(n * 0.95)]
p99 = latencies[min(int(n * 0.99), n - 1)]
mx  = latencies[-1]
print(f'{avg:.2f} {p50:.2f} {p90:.2f} {p95:.2f} {p99:.2f} {mx:.2f}')
")

  AVG=$(echo "$STATS" | awk '{print $1}')
  P50=$(echo "$STATS" | awk '{print $2}')
  P90=$(echo "$STATS" | awk '{print $3}')
  P95=$(echo "$STATS" | awk '{print $4}')
  P99=$(echo "$STATS" | awk '{print $5}')
  MAX=$(echo "$STATS" | awk '{print $6}')

  echo ""
  echo -e "  ${CYAN}延迟分布 (${MEASURED} 个样本):${NC}"
  echo -e "    Avg:  ${YELLOW}${AVG}s${NC}"
  echo -e "    P50:  ${YELLOW}${P50}s${NC}"
  echo -e "    P90:  ${YELLOW}${P90}s${NC}"
  echo -e "    P95:  ${YELLOW}${P95}s${NC}"
  echo -e "    P99:  ${YELLOW}${P99}s${NC}"
  echo -e "    Max:  ${YELLOW}${MAX}s${NC}"
else
  P99="999"
  echo -e "  ${RED}无法测量延迟 — 没有采样到已同步的资产${NC}"
fi

# ── 4. 清理测试数据 ──────────────────────────────────────────────────────────

section "4. 清理测试数据"

CLEANED=0
while IFS= read -r aid; do
  curl -sf -X DELETE "${BASE_URL}/api/v1/assets/${aid}" \
    -H "X-Grace-Token: ${AUTH_TOKEN}" > /dev/null 2>&1 || true
  CLEANED=$((CLEANED + 1))
done < "$ASSET_IDS_FILE"
echo -e "  已清理 ${CLEANED} 个测试资产"

# ── 5. 结果汇总 ──────────────────────────────────────────────────────────────

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "  ${CYAN}Outbox 性能压测结果${NC}"
echo -e "  目标速率:   ${EVENTS_PER_SEC} events/s × ${DURATION_SEC}s = ${TOTAL_EVENTS} events"
echo -e "  实际创建:   ${CREATED} events"
echo -e "  ES 已同步:  ${SYNCED} events"
echo -e "  同步耗时:   ${SYNC_ELAPSED}s"
if [ "$MEASURED" -gt 0 ]; then
  echo -e "  P99 延迟:   ${P99}s (目标 ≤ ${P99_TARGET_SEC}s)"
fi
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Check P99 target
P99_OK=$(python3 -c "print('yes' if float('${P99}') <= float('${P99_TARGET_SEC}') else 'no')")
SYNC_RATE_OK=$(python3 -c "print('yes' if ${SYNCED} >= ${CREATED} else 'no')")

if [ "$P99_OK" = "yes" ] && [ "$SYNC_RATE_OK" = "yes" ]; then
  echo -e "\n${GREEN}PASS${NC} — P99 ${P99}s ≤ ${P99_TARGET_SEC}s, 全部同步完成"
  exit 0
else
  if [ "$P99_OK" != "yes" ]; then
    echo -e "\n${RED}FAIL${NC} — P99 ${P99}s > ${P99_TARGET_SEC}s 目标"
  fi
  if [ "$SYNC_RATE_OK" != "yes" ]; then
    echo -e "\n${RED}FAIL${NC} — 仅 ${SYNCED}/${CREATED} 同步完成"
  fi
  exit 1
fi
