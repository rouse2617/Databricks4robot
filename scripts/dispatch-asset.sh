#!/usr/bin/env bash
# Dispatch asset(s) to Databrew subscription tasks via GCP Pub/Sub (CYB-3801).
#
# 1 asset  → Databrew creates one pipeline run.
# 2+ assets → Databrew creates one batch of N assets.
# Fan-out: each binding on the receiving subscription task dispatches once.

set -euo pipefail

PROJECT="${DATABREW_PROJECT:-green-valley-442103}"
TOPIC="${DATABREW_TOPIC:-cyber-databrew-asset-events}"
BACKEND_URL="${DATABREW_BACKEND_URL:-https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app}"
CLOUDRUN_SERVICE="${DATABREW_CLOUDRUN_SERVICE:-cyber-databrew-backend-dev}"

TOPIC_TAG=""
FILE=""
VERIFY=0
ASSETS=()

usage() {
    cat <<EOF
Dispatch asset(s) to Databrew subscription tasks via Pub/Sub.

Usage:
  $(basename "$0") <asset-id> [asset-id ...]
  $(basename "$0") --file assets.txt              # one asset id per line
  $(basename "$0") -t <label> <asset-id>          # attach reserved topic field
  $(basename "$0") --verify <asset-id>            # tail dispatch log after publish

Options:
  -t, --topic-tag <label>   Reserved 'topic' field in message (currently parsed
                            but not acted on; may drive future routing).
  -f, --file <path>         Read asset ids from file (one per line, # comments).
      --verify              After publish, poll Cloud Run logs for a matching
                            "subtask: dispatched" record (up to 30s).
  -h, --help                Show this help.

Env overrides (defaults shown):
  DATABREW_PROJECT=$PROJECT
  DATABREW_TOPIC=$TOPIC
  DATABREW_CLOUDRUN_SERVICE=$CLOUDRUN_SERVICE   # used by --verify

Semantics (CYB-3801):
  Exactly 1 asset id → single pipeline run (进「执行记录」).
  2+ asset ids       → one batch of N assets (进「批量任务」).
  Fan-out            → 2 bindings on the task + 1-asset msg = 2 runs.

Examples:
  # Single asset → run
  $(basename "$0") 019de9a4-176a-705a-987b-8bf49e0c9075

  # Batch of 3 → one backfill job
  $(basename "$0") asset-a asset-b asset-c

  # Tag the message for future routing (reserved field)
  $(basename "$0") -t mcap-slimmer 019de9a4-...

  # Verify dispatch happened
  $(basename "$0") --verify 019de9a4-...

  # Batch from a file
  $(basename "$0") --file my-assets.txt
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help) usage; exit 0 ;;
        -t|--topic-tag) TOPIC_TAG="$2"; shift 2 ;;
        -f|--file) FILE="$2"; shift 2 ;;
        --verify) VERIFY=1; shift ;;
        --) shift; while [[ $# -gt 0 ]]; do ASSETS+=("$1"); shift; done ;;
        -*) echo "error: unknown flag: $1" >&2; usage >&2; exit 2 ;;
        *) ASSETS+=("$1"); shift ;;
    esac
done

if [[ -n "$FILE" ]]; then
    while IFS= read -r line; do
        line="${line%%#*}"
        line="$(printf '%s' "$line" | tr -d '[:space:]')"
        [[ -z "$line" ]] && continue
        ASSETS+=("$line")
    done < "$FILE"
fi

if [[ ${#ASSETS[@]} -eq 0 ]]; then
    echo "error: no asset ids provided" >&2
    usage >&2
    exit 2
fi

# Build JSON payload via python3 for correct escaping (bash quoting is a
# footgun for JSON — python's json.dumps handles it cleanly).
PAYLOAD="$(python3 -c '
import json, sys
ids = [a for a in sys.argv[1].split("\x1f") if a]
msg = {"asset_ids": ids}
tag = sys.argv[2]
if tag:
    msg["topic"] = tag
print(json.dumps(msg))
' "$(printf '%s\x1f' "${ASSETS[@]}")" "$TOPIC_TAG")"

echo "→ project:  $PROJECT"
echo "→ topic:    $TOPIC"
if [[ ${#ASSETS[@]} -eq 1 ]]; then
    echo "→ mode:     single run (1 asset)"
else
    echo "→ mode:     batch (${#ASSETS[@]} assets)"
fi
echo "→ payload:  $PAYLOAD"
echo

MSG_ID="$(gcloud pubsub topics publish "$TOPIC" \
    --project="$PROJECT" \
    --message="$PAYLOAD" \
    --format='value(messageIds[0])')"

echo "✓ published: messageId=$MSG_ID"

if [[ "$VERIFY" -eq 1 ]]; then
    echo "→ verifying dispatch (up to 30s)..."
    END=$((SECONDS + 30))
    while (( SECONDS < END )); do
        HITS="$(gcloud logging read \
            "resource.type=\"cloud_run_revision\"
             AND resource.labels.service_name=\"$CLOUDRUN_SERVICE\"
             AND jsonPayload.msg=\"subtask: dispatched\"" \
            --project="$PROJECT" --freshness=2m --limit=3 \
            --format='value(timestamp,jsonPayload.runs,jsonPayload.batches,jsonPayload.messages)' 2>/dev/null || true)"
        if [[ -n "$HITS" ]]; then
            echo "✓ dispatched (most-recent 3 subtask log records):"
            printf '  %s\n' "$HITS"
            echo
            echo "→ open UI: 流水线 → 订阅任务 → 点任务名 → 下发历史抽屉"
            exit 0
        fi
        sleep 3
    done
    echo "⚠ no 'subtask: dispatched' log within 30s — task disabled? wrong subscription? Check the UI." >&2
    exit 3
fi
