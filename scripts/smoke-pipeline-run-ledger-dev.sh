#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=/dev/null
source "${ROOT}/scripts/dev-backend-env.sh"

echo "BASE=${BASE}"

watcher_json="$(curl -sfS "${API_HDR[@]}" "${BASE}/api/v1/pipeline-runs/watcher/status")"
WATCHER_JSON="${watcher_json}" python3 - <<'PY'
import json
import os

data = json.loads(os.environ["WATCHER_JSON"])
for key in ("id", "activeScanLimit", "consecutiveFailures", "totalScans", "healthy", "stale"):
    if key not in data:
        raise SystemExit(f"missing watcher field: {key}")
print("watcher status ok:", data.get("id"), "healthy=", data.get("healthy"), "stale=", data.get("stale"))
PY

runs_json="$(curl -sfS "${API_HDR[@]}" "${BASE}/api/v1/pipeline-runs")"
run_id="$(RUNS_JSON="${runs_json}" python3 - <<'PY'
import json
import os

data = json.loads(os.environ["RUNS_JSON"])
items = data.get("items") or []
print(items[0]["id"] if items else "")
PY
)"

if [[ -z "${run_id}" ]]; then
  echo "no pipeline runs found; watcher status smoke passed"
  exit 0
fi

events_json="$(curl -sfS "${API_HDR[@]}" "${BASE}/api/v1/pipeline-runs/${run_id}/events?limit=20")"
EVENTS_JSON="${events_json}" python3 - <<'PY'
import json
import os

data = json.loads(os.environ["EVENTS_JSON"])
if "items" not in data or "total" not in data:
    raise SystemExit("run events response missing items/total")
for item in data.get("items", []):
    for key in ("runId", "eventType", "subjectType", "subjectId", "sequence", "occurredAt"):
        if key not in item:
            raise SystemExit(f"event missing {key}: {item}")
print("run events ok:", len(data.get("items", [])))
PY
