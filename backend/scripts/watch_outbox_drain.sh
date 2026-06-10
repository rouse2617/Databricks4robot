#!/usr/bin/env bash
# Watch outbox relay drain rate by polling /api/v1/search/sync-progress.
#
# Usage:
#   bash backend/scripts/watch_outbox_drain.sh              # 30 samples × 10s
#   INTERVAL=5 SAMPLES=60 bash backend/scripts/watch_outbox_drain.sh
#   BASE_URL=https://my-backend ... bash backend/scripts/watch_outbox_drain.sh
#
# Pulls DATABREW_TOKEN from the live dev Cloud Run env if not set.
set -euo pipefail

PROJECT="${GCP_PROJECT:-green-valley-442103}"
REGION="${GCP_REGION:-us-central1}"
SERVICE="${BACKEND_SERVICE:-cyber-databrew-backend-dev}"
BASE_URL="${BASE_URL:-https://cyber-databrew-backend-dev-234851712830.us-central1.run.app}"
INTERVAL="${INTERVAL:-10}"
SAMPLES="${SAMPLES:-30}"

if [[ -z "${DATABREW_TOKEN:-}" ]]; then
  DATABREW_TOKEN=$(gcloud run services describe "$SERVICE" \
    --region="$REGION" --project="$PROJECT" --format=json 2>/dev/null \
    | python3 -c "import sys,json; svc=json.load(sys.stdin); envs={e['name']:e.get('value','') for e in svc['spec']['template']['spec']['containers'][0].get('env',[])}; print(envs.get('DATABREW_TOKEN',''))")
fi

URL="$BASE_URL/api/v1/search/sync-progress"
prev_lag=""
prev_mq=""

printf '%-10s %-14s %-14s %-12s %-12s %-15s\n' "TIME" "PG_SEQ" "MQ_SEQ" "LAG" "PENDING" "RATE(ev/s)"
for ((i=0; i<SAMPLES; i++)); do
  resp=$(curl -s -H "X-Databrew-Token: $DATABREW_TOKEN" "$URL")
  pg=$(echo "$resp"   | python3 -c "import sys,json;d=json.load(sys.stdin);print(d['pg_max_event_seq'])" 2>/dev/null || echo "?")
  mq=$(echo "$resp"   | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('outbox_published_max_seq',0))" 2>/dev/null || echo "?")
  lag=$(echo "$resp"  | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('seq_lag',0))" 2>/dev/null || echo "?")
  pend=$(echo "$resp" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('outbox_pending_events',0))" 2>/dev/null || echo "?")
  ts=$(date +%H:%M:%S)
  rate=""
  if [[ -n "$prev_mq" && "$mq" =~ ^[0-9]+$ && "$prev_mq" =~ ^[0-9]+$ ]]; then
    delta=$(( mq - prev_mq ))
    rate=$(( delta / INTERVAL ))
    rate="${rate}"
  fi
  printf '%-10s %-14s %-14s %-12s %-12s %-15s\n' "$ts" "$pg" "$mq" "$lag" "$pend" "$rate"
  prev_lag="$lag"; prev_mq="$mq"
  if [[ "$lag" =~ ^[0-9]+$ && "$lag" -lt 100 ]]; then
    echo "lag < 100 — caught up."
    break
  fi
  sleep "$INTERVAL"
done
