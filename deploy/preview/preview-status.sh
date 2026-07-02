#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  deploy/preview/preview-status.sh [--watch] [<pipelinerun-name>]

Without arguments, lists recent fast-preview PipelineRuns.
With a PipelineRun name, prints step progress, latest log hints, and URLs.
With --watch, polls every 5s until the run succeeds or fails.

Examples:
  deploy/preview/preview-status.sh
  deploy/preview/preview-status.sh preview-fast-b723ff083383-151151
  deploy/preview/preview-status.sh --watch preview-fast-b723ff083383-151151
USAGE
}

die() {
  echo "ERROR: $*" >&2
  exit 1
}

tekton_namespace="${TEKTON_NAMESPACE:-cyber-databrew-dev}"
preview_public_base_url="${PREVIEW_PUBLIC_BASE_URL:-https://cyber-databrew-dev.cyberorigin.ai}"
preview_public_base_url="${preview_public_base_url%/}"
watch_mode=false
run_name=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --watch)
      watch_mode=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    -*)
      die "unknown option: $1"
      ;;
    *)
      run_name="$1"
      shift
      ;;
  esac
done

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

require_cmd kubectl
require_cmd awk

list_recent_runs() {
  echo "Recent fast-preview PipelineRuns in ${tekton_namespace}:"
  kubectl get pipelinerun -n "${tekton_namespace}" -l preview=true \
    --sort-by=.metadata.creationTimestamp 2>/dev/null \
    | tail -n 10 || echo "(none)"
}

pipeline_condition() {
  local name="$1"
  kubectl get pipelinerun "${name}" -n "${tekton_namespace}" \
    -o jsonpath='{.status.conditions[?(@.type=="Succeeded")].status}{"\t"}{.status.conditions[?(@.type=="Succeeded")].reason}' 2>/dev/null || true
}

preview_id_for_run() {
  local name="$1"
  kubectl get pipelinerun "${name}" -n "${tekton_namespace}" \
    -o jsonpath='{.metadata.labels.preview-id}' 2>/dev/null || true
}

taskrun_for_pipeline() {
  local name="$1"
  kubectl get taskrun -n "${tekton_namespace}" \
    -l "tekton.dev/pipelineRun=${name}" \
    -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true
}

pod_for_pipeline() {
  local name="$1"
  kubectl get pods -n "${tekton_namespace}" \
    -l "tekton.dev/pipelineRun=${name}" \
    -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true
}

step_state() {
  local taskrun="$1"
  local step="$2"
  kubectl get taskrun "${taskrun}" -n "${tekton_namespace}" -o json 2>/dev/null \
    | STEP="${step}" python3 -c '
import json, os, sys
data = json.load(sys.stdin)
step = os.environ["STEP"]
for item in data.get("status", {}).get("steps", []):
    if item.get("name") != step:
        continue
    if item.get("running") is not None:
        print("running")
        raise SystemExit(0)
    reason = item.get("terminated", {}).get("reason", "")
    if reason == "Completed":
        print("done")
    elif reason:
        print(f"failed:{reason}")
    else:
        print("pending")
    raise SystemExit(0)
print("pending")
' || echo "pending"
}

step_label() {
  case "$1" in
    fetch-source) echo "1/3 拉取 GitHub 源码" ;;
    build-images) echo "2/3 BuildKit 构建镜像" ;;
    deploy-previews) echo "3/3 部署 GKE Preview Pod" ;;
    *) echo "$1" ;;
  esac
}

latest_log_hint() {
  local pod="$1"
  local container="$2"
  [[ -n "${pod}" && -n "${container}" ]] || return 0
  kubectl logs -n "${tekton_namespace}" "${pod}" -c "${container}" --tail=80 2>/dev/null \
    | awk '
      /^TIMING / { line=$0; next }
      /^#[0-9]+ [0-9.]+ TIMING / { sub(/^#[0-9]+ [0-9.]+ /, ""); line=$0; next }
      /^Skipping frontend image build/ { line=$0; next }
      /^Registry credentials written/ { line=$0; next }
      /^Fast preview deployment complete/ { line=$0; next }
      END { if (line != "") print line }
    ' || true
}

print_run_status() {
  local name="$1"
  local cond status reason preview_id taskrun pod
  cond="$(pipeline_condition "${name}")"
  status="${cond%%$'\t'*}"
  reason="${cond#*$'\t'}"
  preview_id="$(preview_id_for_run "${name}")"
  taskrun="$(taskrun_for_pipeline "${name}")"
  pod="$(pod_for_pipeline "${name}")"

  echo "PipelineRun: ${name}"
  echo "namespace:   ${tekton_namespace}"
  echo "preview-id:  ${preview_id:-unknown}"
  echo "condition:   status=${status:-Pending} reason=${reason:-Pending}"
  if [[ -n "${preview_id}" ]]; then
    echo "backend url: ${preview_public_base_url}/preview/${preview_id}/api"
    echo "frontend:    ${preview_public_base_url}/preview/${preview_id}/"
  fi
  echo ""
  echo "Steps:"

  local current_step="" current_container="" hint=""
  for step in fetch-source build-images deploy-previews; do
    local state
    state="$(step_state "${taskrun}" "${step}")"
    printf '  - %-16s %s (%s)\n' "${step}" "$(step_label "${step}")" "${state}"
    if [[ "${state}" == "running" ]]; then
      current_step="${step}"
      current_container="step-${step}"
    fi
  done

  if [[ -n "${current_step}" ]]; then
    hint="$(latest_log_hint "${pod}" "${current_container}")"
    echo ""
    echo "Current: ${current_step}"
    if [[ -n "${hint}" ]]; then
      echo "Latest:  ${hint}"
    else
      echo "Latest:  (waiting for logs...)"
    fi
  fi

  if [[ "${status}" == "True" ]]; then
    echo ""
    echo "Done. Timing summary:"
    for container in step-fetch-source step-build-images step-deploy-previews; do
      kubectl logs -n "${tekton_namespace}" "${pod}" -c "${container}" --tail=200 2>/dev/null \
        | awk '
          /^TIMING / { print; next }
          /^#[0-9]+ [0-9.]+ TIMING / { sub(/^#[0-9]+ [0-9.]+ /, ""); print; next }
        ' || true
    done
  elif [[ "${status}" == "False" ]]; then
    echo ""
    echo "Failed. Recent logs:"
    kubectl logs -n "${tekton_namespace}" "${pod}" --all-containers --tail=40 2>/dev/null || true
  fi
}

if [[ -z "${run_name}" ]]; then
  list_recent_runs
  exit 0
fi

if [[ "${watch_mode}" != "true" ]]; then
  print_run_status "${run_name}"
  exit 0
fi

echo "Watching ${run_name} (Ctrl+C to stop)..."
while true; do
  clear || true
  print_run_status "${run_name}"
  cond="$(pipeline_condition "${run_name}")"
  status="${cond%%$'\t'*}"
  case "${status}" in
    True|False) break ;;
  esac
  sleep 5
done
