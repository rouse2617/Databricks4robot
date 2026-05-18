#!/usr/bin/env bash
# Full PG → Elasticsearch assets reindex via backend admin API.
# Requires: reachable backend, X-Grace-Token, Elasticsearch configured on the server.
#
# Usage:
#   BASE=https://cyber-databrew-backend-dev-....run.app TOKEN=dev-token ./backend/scripts/search_reindex.sh
#   DRY_RUN_ONLY=1 BASE=http://localhost:8080 TOKEN=dev-token ./backend/scripts/search_reindex.sh

set -euo pipefail
BASE="${BASE:-http://localhost:8080}"
TOKEN="${TOKEN:-dev-token}"
PAGE_SIZE="${PAGE_SIZE:-200}"

echo "=== GET outbox-stats (read-only) ==="
curl -sS "${BASE}/api/v1/admin/search/outbox-stats" \
  -H "X-Grace-Token: ${TOKEN}" | jq .

echo "=== POST reindex dry_run ==="
curl -sS -X POST "${BASE}/api/v1/admin/search/reindex" \
  -H "X-Grace-Token: ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"dry_run\":true,\"page_size\":${PAGE_SIZE}}" | jq .

if [[ "${DRY_RUN_ONLY:-}" == "1" ]]; then
  echo "DRY_RUN_ONLY=1 set; skipping destructive reindex."
  exit 0
fi

read -r -p "Run full reindex (wipes ES docs then bulk-indexes from PG)? [y/N] " ok
if [[ "${ok}" != "y" && "${ok}" != "Y" ]]; then
  echo "Aborted."
  exit 0
fi

echo "=== POST reindex dry_run=false ==="
curl -sS -X POST "${BASE}/api/v1/admin/search/reindex" \
  -H "X-Grace-Token: ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"dry_run\":false,\"page_size\":${PAGE_SIZE}}" | jq .

echo "=== GET sync-progress (optional) ==="
curl -sS "${BASE}/api/v1/search/sync-progress" \
  -H "X-Grace-Token: ${TOKEN}" | jq . || true
