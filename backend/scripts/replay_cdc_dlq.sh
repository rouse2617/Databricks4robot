#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <dlq-jsonl-file> [topic-override]"
  exit 1
fi

DLQ_FILE="$1"
TOPIC_OVERRIDE="${2:-}"

ARGS=(run ./cmd/cdc-dlq-replay --file "$DLQ_FILE")
if [[ -n "$TOPIC_OVERRIDE" ]]; then
  ARGS+=(--topic "$TOPIC_OVERRIDE")
fi

go "${ARGS[@]}"
