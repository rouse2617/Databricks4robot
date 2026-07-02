#!/usr/bin/env bash
# 全栈重启脚本 — 切分支 + 重启前后端（默认跳过迁移）
#
# Usage:
#   bash deploy/local/restart.sh                    # 重启当前分支
#   bash deploy/local/restart.sh <branch>           # 切换到指定分支后重启
#   bash deploy/local/restart.sh <branch> --migrate # 切分支 + 应用迁移
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT}"

BRANCH="${1:-}"
RUN_MIGRATE=false
[[ "${2:-}" == "--migrate" ]] && RUN_MIGRATE=true

# ── 颜色 ──────────────────────────────────
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# ── 配置 ──────────────────────────────────
ARGO_TOKEN_FILE="/tmp/argo-token.txt"
BACKEND_PORT="${BACKEND_PORT:-8080}"
FRONTEND_PORT="${FRONTEND_PORT:-5176}"
ARGO_PORT="${ARGO_PORT:-12746}"
ARGO_NAMESPACE="${ARGO_NAMESPACE:-argo}"
ARGO_SERVER_SERVICE="${ARGO_SERVER_SERVICE:-}"

# ── 1. 切分支 ─────────────────────────────
if [[ -n "${BRANCH}" ]]; then
  echo -e "${YELLOW}[1/6] 切分支: ${BRANCH}${NC}"
  git fetch origin dev 2>/dev/null || true
  git checkout "${BRANCH}" 2>/dev/null || {
    echo -e "${RED}分支 ${BRANCH} 不存在,尝试 git checkout -b ${BRANCH} origin/dev${NC}"
    git checkout -b "${BRANCH}" origin/dev
  }
  echo -e "${GREEN}当前分支: $(git branch --show-current)${NC}"
else
  echo -e "${YELLOW}[1/6] 当前分支: $(git branch --show-current) (未指定新分支,跳过切分支)${NC}"
fi

# ── 2. 停旧进程 ────────────────────────────
echo -e "${YELLOW}[2/6] 停止旧进程...${NC}"
# 停占用后端端口的进程（不管 go run 还是编译好的 ./server）
BPID=$(lsof -nP -iTCP:"${BACKEND_PORT}" -sTCP:LISTEN -t 2>/dev/null || true)
if [[ -n "${BPID}" ]]; then
  kill ${BPID} 2>/dev/null && echo "  已停 backend 进程 (PID ${BPID})" || true
else
  echo "  backend 未运行"
fi
# 停 vite
VPID=$(lsof -nP -iTCP:"${FRONTEND_PORT}" -sTCP:LISTEN -t 2>/dev/null || true)
if [[ -n "${VPID}" ]]; then
  kill ${VPID} 2>/dev/null && echo "  已停 vite (PID ${VPID})" || true
else
  echo "  vite 未运行"
fi
sleep 1

# ── 3. 检查依赖 ────────────────────────────
echo -e "${YELLOW}[3/6] 检查依赖服务...${NC}"

# Postgres
if curl -s --max-time 2 http://localhost:5432 >/dev/null 2>&1; then
  echo "  Postgres ✅"
else
  echo -e "${YELLOW}  Postgres 未运行,启动中...${NC}"
  docker compose -f deploy/local/docker-compose.yml up -d postgres pgbouncer 2>&1 | tail -1
  sleep 3
fi

# ES
if curl -s --max-time 2 http://localhost:9200 >/dev/null 2>&1; then
  echo "  Elasticsearch ✅"
else
  echo "  Elasticsearch 未运行 (跳过,非核心依赖)"
fi

# Argo port-forward
if [[ -z "${ARGO_SERVER_SERVICE}" ]]; then
  if kubectl --context kind-argo-local -n "${ARGO_NAMESPACE}" get svc argo-workflows-server >/dev/null 2>&1; then
    ARGO_SERVER_SERVICE="argo-workflows-server"
  else
    ARGO_SERVER_SERVICE="argo-server"
  fi
fi
ARGO_PF=$(ps aux | grep "port-forward.*${ARGO_SERVER_SERVICE}.*${ARGO_PORT}" | grep -v grep | awk '{print $2}' || true)
if [[ -n "${ARGO_PF}" ]]; then
  echo "  Argo port-forward ✅ (PID ${ARGO_PF})"
else
  echo -e "${YELLOW}  Argo port-forward 未运行,启动中...${NC}"
  nohup kubectl --context kind-argo-local -n "${ARGO_NAMESPACE}" port-forward --address 0.0.0.0 "svc/${ARGO_SERVER_SERVICE}" "${ARGO_PORT}:2746" > /tmp/argo-portforward.log 2>&1 &
  echo "  Argo port-forward PID=$!"
fi

# Argo token
if [[ ! -f "${ARGO_TOKEN_FILE}" ]]; then
  echo -e "${YELLOW}  刷新 Argo token...${NC}"
  kubectl --context kind-argo-local -n "${ARGO_NAMESPACE}" create token databrew-backend --duration=24h > "${ARGO_TOKEN_FILE}" 2>/dev/null || true
fi

# ── 4. 数据库迁移 ──────────────────────────
if [[ "${RUN_MIGRATE}" == true ]]; then
  echo -e "${YELLOW}[4/6] 数据库迁移...${NC}"
  make local-migrate 2>&1 | tail -5
else
  echo -e "${YELLOW}[4/6] 数据库迁移: 跳过 (默认,需要时加 --migrate)${NC}"
fi

# ── 5. 启动后端 ────────────────────────────
echo -e "${YELLOW}[5/6] 启动后端 (go run)...${NC}"

ARGO_TOKEN="${ARGO_TOKEN:-$(cat "${ARGO_TOKEN_FILE}" 2>/dev/null || echo '')}"
cd "${ROOT}/backend"
nohup env \
  ENV=development \
  STORAGE_BACKEND=postgres \
  DB_HOST="${DB_HOST:-localhost}" \
  DB_PORT="${DB_PORT:-5432}" \
  DB_USER="${DB_USER:-postgres}" \
  DB_PASSWORD="${DB_PASSWORD:-postgres}" \
  DB_NAME="${DB_NAME:-cyber_databrew_dev}" \
  DATABREW_TOKEN="${DATABREW_TOKEN:-dev-token}" \
  ARGO_SERVER_URL="${ARGO_SERVER_URL:-http://localhost:${ARGO_PORT}}" \
  ARGO_TOKEN="${ARGO_TOKEN}" \
  ARGO_WORKFLOWS_NAMESPACE="${ARGO_WORKFLOWS_NAMESPACE:-argo}" \
  ARGO_INSECURE_SKIP_VERIFY=true \
  PRICING_CONFIG_PATH="${PRICING_CONFIG_PATH:-${ROOT}/backend/config/gcp_pricing.yaml}" \
  LAKEHOUSE_BACKEND=bigquery \
  ELASTICSEARCH_URL="${ELASTICSEARCH_URL:-http://localhost:9200}" \
  PORT="${BACKEND_PORT}" \
  go run ./cmd/server/ > /tmp/backend-dev.log 2>&1 &
BACKEND_PID=$!
echo "  backend PID=${BACKEND_PID}"

# 等后端 ready
echo -n "  等待后端就绪..."
for i in $(seq 1 30); do
  if curl -s --max-time 1 -o /dev/null "http://localhost:${BACKEND_PORT}/healthz" 2>/dev/null; then
    echo -e " ${GREEN}✅ ($(($i * 1))s)${NC}"
    break
  fi
  echo -n "."
  sleep 1
done
if ! curl -s --max-time 1 -o /dev/null "http://localhost:${BACKEND_PORT}/healthz" 2>/dev/null; then
  echo -e " ${RED}后端启动超时,检查日志: tail -20 /tmp/backend-dev.log${NC}"
fi

# ── 6. 启动前端 ────────────────────────────
echo -e "${YELLOW}[6/6] 启动前端 (vite)...${NC}"
cd "${ROOT}/Frontend"
lsof -nP -iTCP:"${FRONTEND_PORT}" -sTCP:LISTEN 2>/dev/null | awk 'NR>1{print $2}' | xargs kill 2>/dev/null || true
sleep 1
nohup env \
  VITE_API_BASE_URL="http://localhost:${BACKEND_PORT}" \
  VITE_DEV_ACCESS_TOKEN="${DATABREW_TOKEN:-dev-token}" \
  ./node_modules/.bin/vite --host 0.0.0.0 --port "${FRONTEND_PORT}" > /tmp/vite.log 2>&1 &
VITE_PID=$!
echo "  vite PID=${VITE_PID}"

# 等前端 ready
echo -n "  等待前端就绪..."
for i in $(seq 1 15); do
  if curl -s --max-time 1 -o /dev/null "http://localhost:${FRONTEND_PORT}" 2>/dev/null; then
    echo -e " ${GREEN}✅ ($(($i * 1))s)${NC}"
    break
  fi
  echo -n "."
  sleep 1
done

# ── 完成 ───────────────────────────────────
echo ""
echo -e "${GREEN}═════════════════════════════════════${NC}"
echo -e "${GREEN}  全栈已启动${NC}"
echo -e "${GREEN}═════════════════════════════════════${NC}"
echo "  分支:  $(git -C "${ROOT}" branch --show-current)"
echo "  前端:  http://localhost:${FRONTEND_PORT}"
echo "  后端:  http://localhost:${BACKEND_PORT}"
echo "  Argo:  http://localhost:${ARGO_PORT}"
echo ""
echo "  日志:  tail -f /tmp/backend-dev.log"
echo "        tail -f /tmp/vite.log"
echo "  重启:  bash deploy/local/restart.sh"
