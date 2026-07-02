#!/usr/bin/env bash
# Smoke: canonical pipeline runtime baseline on dev Cloud Run.
#
# The baseline covers three fixed workload shapes:
#   1. sleep + terminal: Pod stays alive long enough to create/attach/terminate a terminal session.
#   2. log-spam: emits many lines so SSE/log viewer paths can be stress-tested.
#   3. resource request: requests CPU/memory (and optional GPU) so monitoring, Pod diagnostics,
#      and scheduling diagnostics have stable data to inspect.
#
# Safe default: validate provided workflows only. To create fresh smoke runs:
#   RUN_SMOKE_PIPELINES=1 TARGET_ID=<execution-target-id> bash scripts/smoke-pipeline-runtime-baseline-dev.sh
#
# Useful overrides:
#   LOG_SPAM_LINES=10000
#   GPU_REQUEST=1                 # opt-in; may require cluster capacity
#   REQUIRE_TERMINAL=0            # allow target policy to keep terminal disabled
#   SLEEP_WORKFLOW=<name> SLEEP_NODE_ID=<node-id>
#   LOG_WORKFLOW=<name> LOG_NODE_ID=<node-id>
#   RESOURCE_WORKFLOW=<name> RESOURCE_NODE_ID=<node-id>
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/scripts/dev-backend-env.sh"

RUN_SMOKE_PIPELINES="${RUN_SMOKE_PIPELINES:-0}"
TARGET_ID="${TARGET_ID:-}"
LOG_SPAM_LINES="${LOG_SPAM_LINES:-2000}"
GPU_REQUEST="${GPU_REQUEST:-0}"
REQUIRE_TERMINAL="${REQUIRE_TERMINAL:-1}"
WAIT_SECONDS="${WAIT_SECONDS:-180}"

PASS=0
FAIL=0

ok() { PASS=$((PASS + 1)); echo "  OK  $1"; }
skip() { echo "  SKIP $1"; }
bad() {
  FAIL=$((FAIL + 1))
  echo "  FAIL $1"
  if [[ -n "${2:-}" ]]; then
    printf '%s\n' "$2" | head -c 1200
    echo
  fi
}

urlencode() {
  python3 -c 'import sys, urllib.parse; print(urllib.parse.quote(sys.argv[1], safe=""))' "$1"
}

request_json() {
  local method="$1" path="$2" payload="${3:-}"
  local tmp code
  tmp="$(mktemp)"
  if [[ -n "$payload" ]]; then
    code="$(curl -sS -o "$tmp" -w "%{http_code}" -X "$method" "${API_HDR[@]}" --data-binary "@$payload" "$BASE$path")"
  else
    code="$(curl -sS -o "$tmp" -w "%{http_code}" -X "$method" "${API_HDR[@]}" "$BASE$path")"
  fi
  printf '%s %s\n' "$code" "$tmp"
}

make_pipeline_payload() {
  local kind="$1" name="$2" out="$3"
  python3 - "$kind" "$name" "$LOG_SPAM_LINES" "$GPU_REQUEST" "$TARGET_ID" >"$out" <<'PY'
import json
import sys

kind, name, log_lines, gpu_request, target_id = sys.argv[1:6]

commands = {
    "sleep": "echo smoke-terminal-ready; sleep 180; echo smoke-terminal-done",
    "log": f"for i in $(seq 1 {int(log_lines)}); do echo log-spam-$i; done",
    "resource": "echo resource-smoke-start; dd if=/dev/zero of=/tmp/databrew-smoke.bin bs=1M count=32 2>/dev/null || true; sleep 30; echo resource-smoke-done",
}
resources = {
    "sleep": {"cpu": "100m", "memory": "128Mi", "disk": "1Gi"},
    "log": {"cpu": "100m", "memory": "128Mi", "disk": "1Gi"},
    "resource": {"cpu": "500m", "memory": "512Mi", "disk": "2Gi"},
}
if kind == "resource" and gpu_request not in ("", "0", "false", "False"):
    resources[kind]["gpu"] = str(gpu_request)
    resources[kind]["computeTier"] = "gpu-l4"

node_id = f"{kind}-node"
payload = {
    "name": name,
    "asset_ids": [],
    "pipeline": {
        "name": name,
        "version": "smoke",
        "nodes": [
            {
                "id": node_id,
                "component": {
                    "name": node_id,
                    "image": "busybox:1.36",
                    "command": ["sh", "-lc"],
                    "args": [{"name": "script", "value": commands[kind]}],
                    "resources": resources[kind],
                },
            }
        ],
        "edges": [],
    },
}
if target_id:
    payload["target_id"] = target_id
print(json.dumps(payload))
PY
}

create_smoke_run() {
  local kind="$1" name payload code bodyfile
  name="smoke-${kind}-$(date +%Y%m%d%H%M%S)"
  payload="$(mktemp)"
  make_pipeline_payload "$kind" "$name" "$payload"
  read -r code bodyfile < <(request_json POST "/api/v1/pipeline-runs" "$payload")
  rm -f "$payload"
  if [[ "$code" != "201" ]]; then
    bad "create ${kind} smoke run returned HTTP $code" "$(cat "$bodyfile")"
    rm -f "$bodyfile"
    return 1
  fi
  python3 - "$bodyfile" <<'PY'
import json, sys
body = json.load(open(sys.argv[1]))
print(body.get("workflowName") or body.get("workflow_name") or "")
print(body.get("id") or "")
PY
  rm -f "$bodyfile"
}

find_first_pod_node_id() {
  local workflow="$1" bodyfile="$2"
  python3 - "$bodyfile" <<'PY'
import json, sys
body = json.load(open(sys.argv[1]))
nodes = body.get("nodes") or []
for node in nodes:
    node_type = (node.get("type") or "").lower()
    if node_type in ("dag", "steps", "stepgroup"):
        continue
    print(node.get("id") or node.get("name") or "")
    raise SystemExit(0)
print("")
PY
}

wait_for_node() {
  local workflow="$1" explicit_node="${2:-}" deadline code bodyfile node_id
  if [[ -n "$explicit_node" ]]; then
    printf '%s\n' "$explicit_node"
    return 0
  fi
  deadline=$((SECONDS + WAIT_SECONDS))
  while (( SECONDS < deadline )); do
    read -r code bodyfile < <(request_json GET "/api/v1/workflows/$(urlencode "$workflow")")
    if [[ "$code" == "200" ]]; then
      node_id="$(find_first_pod_node_id "$workflow" "$bodyfile")"
      rm -f "$bodyfile"
      if [[ -n "$node_id" ]]; then
        printf '%s\n' "$node_id"
        return 0
      fi
    else
      rm -f "$bodyfile"
    fi
    sleep 3
  done
  return 1
}

pod_has_running_container() {
  local bodyfile="$1"
  python3 - "$bodyfile" <<'PY'
import json, sys
body = json.load(open(sys.argv[1]))
for container in body.get("containers") or []:
    if (container.get("state") or "").lower() == "running":
        raise SystemExit(0)
raise SystemExit(1)
PY
}

try_create_terminal() {
  local workflow="$1" node="$2" payload code bodyfile session_id term_code term_bodyfile
  payload="$(mktemp)"
  printf '{"command":"sh"}' >"$payload"
  read -r code bodyfile < <(request_json POST "/api/v1/workflows/$(urlencode "$workflow")/nodes/$(urlencode "$node")/terminal-sessions" "$payload")
  rm -f "$payload"
  if [[ "$code" == "201" ]]; then
    session_id="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1])).get("id",""))' "$bodyfile")"
    rm -f "$bodyfile"
    if [[ -n "$session_id" ]]; then
      read -r term_code term_bodyfile < <(request_json POST "/api/v1/pod-terminal/sessions/$(urlencode "$session_id")/terminate" || true)
      rm -f "${term_bodyfile:-}"
    fi
    return 0
  fi
  rm -f "$bodyfile"
  if [[ "$code" == "409" || "$code" == "503" ]]; then
    return 1
  fi
  if [[ "$code" == "403" && "$REQUIRE_TERMINAL" != "1" ]]; then
    return 2
  fi
  return 3
}

wait_and_check_terminal() {
  local workflow="$1" node="$2" deadline code bodyfile rc
  deadline=$((SECONDS + WAIT_SECONDS))
  while (( SECONDS < deadline )); do
    read -r code bodyfile < <(request_json GET "/api/v1/workflows/$(urlencode "$workflow")/nodes/$(urlencode "$node")/pod")
    if [[ "$code" == "200" ]] && pod_has_running_container "$bodyfile"; then
      rm -f "$bodyfile"
      if try_create_terminal "$workflow" "$node"; then
        ok "terminal create/terminate: $workflow/$node"
        return 0
      fi
    else
      rm -f "$bodyfile"
    fi
    sleep 1
  done
  bad "terminal never attached while pod was running: $workflow/$node"
  return 1
}

check_logs() {
  local workflow="$1" node="$2" code bodyfile streamfile log_count
  read -r code bodyfile < <(request_json GET "/api/v1/workflows/$(urlencode "$workflow")/logs?nodeId=$(urlencode "$node")&tailLines=2000&limitBytes=1048576")
  if [[ "$code" == "200" ]] && python3 - "$bodyfile" <<'PY'
import json, sys
body = json.load(open(sys.argv[1]))
assert "logs" in body
assert isinstance(body.get("lineCount", 0), int)
PY
  then
    ok "bounded logs: $workflow/$node"
  else
    bad "bounded logs failed for $workflow/$node (HTTP $code)" "$(cat "$bodyfile")"
  fi
  rm -f "$bodyfile"

  streamfile="$(mktemp)"
  curl -sS -N -m 12 "${API_HDR[@]}" \
    "$BASE/api/v1/workflows/$(urlencode "$workflow")/logs/stream?nodeId=$(urlencode "$node")&limitBytes=1048576" \
    >"$streamfile" || true
  log_count="$(grep -c '^event: log' "$streamfile" || true)"
  if [[ "$log_count" -gt 0 ]] || grep -q '^event: end' "$streamfile"; then
    ok "SSE logs streamed ${log_count} log events: $workflow/$node"
  else
    bad "SSE logs produced no events for $workflow/$node" "$(cat "$streamfile")"
  fi
  rm -f "$streamfile"
}

check_terminal() {
  local workflow="$1" node="$2" rc
  try_create_terminal "$workflow" "$node"
  rc=$?
  if [[ "$rc" == "0" ]]; then
    ok "terminal create/terminate: $workflow/$node"
    return 0
  fi
  if [[ "$rc" == "2" ]]; then
    skip "terminal disabled by policy: $workflow/$node"
    return 0
  fi
  wait_and_check_terminal "$workflow" "$node"
}

check_pod_and_resources() {
  local workflow="$1" node="$2" code bodyfile
  read -r code bodyfile < <(request_json GET "/api/v1/workflows/$(urlencode "$workflow")/nodes/$(urlencode "$node")/pod")
  if [[ "$code" == "200" ]] && grep -q '"podName"' "$bodyfile"; then
    ok "Pod diagnostics: $workflow/$node"
  else
    bad "Pod diagnostics failed for $workflow/$node (HTTP $code)" "$(cat "$bodyfile")"
  fi
  rm -f "$bodyfile"

  read -r code bodyfile < <(request_json GET "/api/v1/workflows/$(urlencode "$workflow")/nodes/$(urlencode "$node")/resources")
  if [[ "$code" == "200" ]] && python3 - "$bodyfile" <<'PY'
import json, sys
body = json.load(open(sys.argv[1]))
assert "pods" in body
PY
  then
    ok "resource snapshot: $workflow/$node"
  else
    bad "resource snapshot failed for $workflow/$node (HTTP $code)" "$(cat "$bodyfile")"
  fi
  rm -f "$bodyfile"
}

run_or_skip() {
  local label="$1" workflow="$2" node="$3" check_fn="$4"
  if [[ -z "$workflow" ]]; then
    skip "$label workflow not set"
    return
  fi
  if [[ -z "$node" ]]; then
    if ! node="$(wait_for_node "$workflow")"; then
      bad "$label could not resolve a Pod node for $workflow"
      return
    fi
  fi
  "$check_fn" "$workflow" "$node"
}

echo "=== smoke-pipeline-runtime-baseline-dev === BASE=$BASE"

SLEEP_WORKFLOW="${SLEEP_WORKFLOW:-}"
SLEEP_NODE_ID="${SLEEP_NODE_ID:-}"
LOG_WORKFLOW="${LOG_WORKFLOW:-}"
LOG_NODE_ID="${LOG_NODE_ID:-}"
RESOURCE_WORKFLOW="${RESOURCE_WORKFLOW:-}"
RESOURCE_NODE_ID="${RESOURCE_NODE_ID:-}"
SLEEP_TERMINAL_CHECKED=0

if [[ "$RUN_SMOKE_PIPELINES" == "1" ]]; then
  read -r SLEEP_WORKFLOW _ < <(create_smoke_run sleep)
  if [[ -n "$SLEEP_WORKFLOW" ]]; then
    SLEEP_NODE_ID="$(wait_for_node "$SLEEP_WORKFLOW" || true)"
    if [[ -n "$SLEEP_NODE_ID" ]] && wait_and_check_terminal "$SLEEP_WORKFLOW" "$SLEEP_NODE_ID"; then
      SLEEP_TERMINAL_CHECKED=1
    elif [[ -n "$SLEEP_NODE_ID" ]]; then
      bad "sleep terminal check failed: $SLEEP_WORKFLOW"
    fi
  fi
  read -r LOG_WORKFLOW _ < <(create_smoke_run log)
  read -r RESOURCE_WORKFLOW _ < <(create_smoke_run resource)
fi

if [[ -n "$SLEEP_WORKFLOW" && -z "$SLEEP_NODE_ID" ]]; then
  SLEEP_NODE_ID="$(wait_for_node "$SLEEP_WORKFLOW" || true)"
fi
if [[ -n "$LOG_WORKFLOW" && -z "$LOG_NODE_ID" ]]; then
  LOG_NODE_ID="$(wait_for_node "$LOG_WORKFLOW" || true)"
fi
if [[ -n "$RESOURCE_WORKFLOW" && -z "$RESOURCE_NODE_ID" ]]; then
  RESOURCE_NODE_ID="$(wait_for_node "$RESOURCE_WORKFLOW" || true)"
fi

if [[ "$SLEEP_TERMINAL_CHECKED" != "1" ]]; then
  run_or_skip "sleep terminal" "$SLEEP_WORKFLOW" "$SLEEP_NODE_ID" check_terminal
fi
run_or_skip "sleep Pod diagnostics" "$SLEEP_WORKFLOW" "$SLEEP_NODE_ID" check_pod_and_resources
run_or_skip "log-spam" "$LOG_WORKFLOW" "$LOG_NODE_ID" check_logs
run_or_skip "resource request" "$RESOURCE_WORKFLOW" "$RESOURCE_NODE_ID" check_pod_and_resources

echo "=== done: ${PASS} passed, ${FAIL} failed ==="
[[ "$FAIL" -eq 0 ]]
