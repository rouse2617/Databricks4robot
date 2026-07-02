#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  deploy/preview/fast-preview.sh <commit-ish> [--frontend remote|local] [--preview-id id] [--ttl-hours n] [--wait|--no-wait]

Environment overrides:
  PROJECT_ID                  default: green-valley-442103
  REGION                      default: us-central1
  ARTIFACT_REPO               default: cyber-databrew-images
  TEKTON_NAMESPACE            default: cyber-databrew-dev
  TEKTON_SERVICE_ACCOUNT      default: tekton-builder
  GITHUB_TOKEN_SECRET         default: cyber-databrew-preview-github-token
  REFRESH_GITHUB_TOKEN_SECRET default: false
  PREVIEW_NAMESPACE           default: cyber-databrew-dev
  PREVIEW_PUBLIC_BASE_URL     default: https://cyber-databrew-dev.cyberorigin.ai
  PREVIEW_BUILDKITD_NAMESPACE default: cyber-databrew-dev
  PREVIEW_BUILDKITD_DEPLOY    default: preview-buildkitd
  PREVIEW_ENV_RBAC_MANIFEST   default: deploy/preview/k8s/preview-env-reader.yaml
  PREVIEW_BOOTSTRAP           default: true

If the Kubernetes GitHub token Secret is missing, the script creates it from
GITHUB_TOKEN or `gh auth token`. The token is not printed and is not written to
the repository.
USAGE
}

die() {
  echo "ERROR: $*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

sanitize_preview_id() {
  printf '%s' "$1" \
    | tr '[:upper:]' '[:lower:]' \
    | tr -c 'a-z0-9-' '-' \
    | sed -E 's/^-+//; s/-+$//; s/-+/-/g' \
    | cut -c1-24
}

sanitize_k8s_name() {
  printf '%s' "$1" \
    | tr '[:upper:]' '[:lower:]' \
    | tr -c 'a-z0-9-' '-' \
    | sed -E 's/^-+//; s/-+$//; s/-+/-/g' \
    | cut -c1-63 \
    | sed -E 's/-+$//'
}

commitish=""
frontend_mode="remote"
preview_id=""
ttl_hours="${TTL_HOURS:-24}"
wait_for_completion=true

while [[ $# -gt 0 ]]; do
  case "$1" in
    --frontend)
      frontend_mode="${2:-}"
      shift 2
      ;;
    --preview-id)
      preview_id="${2:-}"
      shift 2
      ;;
    --ttl-hours)
      ttl_hours="${2:-}"
      shift 2
      ;;
    --wait)
      wait_for_completion=true
      shift
      ;;
    --no-wait)
      wait_for_completion=false
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
      if [[ -n "${commitish}" ]]; then
        die "unexpected extra argument: $1"
      fi
      commitish="$1"
      shift
      ;;
  esac
done

[[ -n "${commitish}" ]] || { usage >&2; exit 2; }
[[ "${frontend_mode}" == "remote" || "${frontend_mode}" == "local" ]] || die "--frontend must be remote or local"
[[ "${ttl_hours}" =~ ^[0-9]+$ ]] || die "--ttl-hours must be an integer"

require_cmd git
require_cmd kubectl
require_cmd sed
require_cmd awk

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/../.." && pwd)"
template="${script_dir}/tekton/preview-fast-pipelinerun.yaml"
buildkit_manifest="${script_dir}/k8s/preview-buildkitd.yaml"
env_rbac_manifest="${PREVIEW_ENV_RBAC_MANIFEST:-${script_dir}/k8s/preview-env-reader.yaml}"
[[ -f "${template}" ]] || die "missing template: ${template}"
[[ -f "${buildkit_manifest}" ]] || die "missing buildkitd manifest: ${buildkit_manifest}"
[[ -f "${env_rbac_manifest}" ]] || die "missing env RBAC manifest: ${env_rbac_manifest}"

project_id="${PROJECT_ID:-green-valley-442103}"
region="${REGION:-us-central1}"
artifact_repo="${ARTIFACT_REPO:-cyber-databrew-images}"
tekton_namespace="${TEKTON_NAMESPACE:-cyber-databrew-dev}"
tekton_service_account="${TEKTON_SERVICE_ACCOUNT:-tekton-builder}"
github_token_secret="${GITHUB_TOKEN_SECRET:-cyber-databrew-preview-github-token}"
refresh_github_token_secret="${REFRESH_GITHUB_TOKEN_SECRET:-false}"
repo_url="${REPO_URL:-https://github.com/CyberOrigin2077/cyber-databrew.git}"
mcap_preview_host="${MCAP_PREVIEW_HOST:-mcap-preview-dev-wtttm6suaq-uc.a.run.app}"
preview_namespace="${PREVIEW_NAMESPACE:-cyber-databrew-dev}"
preview_public_base_url="${PREVIEW_PUBLIC_BASE_URL:-https://cyber-databrew-dev.cyberorigin.ai}"
preview_public_base_url="${preview_public_base_url%/}"
buildkitd_namespace="${PREVIEW_BUILDKITD_NAMESPACE:-cyber-databrew-dev}"
buildkitd_deploy="${PREVIEW_BUILDKITD_DEPLOY:-preview-buildkitd}"
preview_bootstrap="${PREVIEW_BOOTSTRAP:-true}"

commit_sha="$(git -C "${repo_root}" rev-parse "${commitish}^{commit}")"
default_preview_id="$(git -C "${repo_root}" rev-parse --short=12 "${commit_sha}")"
preview_id="$(sanitize_preview_id "${preview_id:-${default_preview_id}}")"
[[ -n "${preview_id}" ]] || die "preview id is empty after sanitization"

run_name="$(sanitize_k8s_name "preview-fast-${preview_id}-$(date -u +%H%M%S)")"
api_service="cdb-pv-${preview_id}-api"
web_service="cdb-pv-${preview_id}-web"
web_url="${preview_public_base_url}/preview/${preview_id}/"
api_url="${preview_public_base_url}/preview/${preview_id}/api"

ensure_github_token_secret() {
  if [[ "${refresh_github_token_secret}" != "true" ]] \
    && kubectl get secret "${github_token_secret}" -n "${tekton_namespace}" >/dev/null 2>&1; then
    return 0
  fi

  local token="${GITHUB_TOKEN:-}"
  if [[ -z "${token}" ]]; then
    if command -v gh >/dev/null 2>&1; then
      token="$(gh auth token)"
    else
      die "Kubernetes Secret ${tekton_namespace}/${github_token_secret} is missing and neither GITHUB_TOKEN nor gh is available"
    fi
  fi
  [[ -n "${token}" ]] || die "empty GitHub token"

  kubectl create secret generic "${github_token_secret}" \
    -n "${tekton_namespace}" \
    --from-literal=token="${token}" \
    --dry-run=client -o yaml \
    | kubectl apply -f - >/dev/null
}

render_template() {
  sed \
    -e "s#__RUN_NAME__#${run_name}#g" \
    -e "s#__TEKTON_NAMESPACE__#${tekton_namespace}#g" \
    -e "s#__TEKTON_SERVICE_ACCOUNT__#${tekton_service_account}#g" \
    -e "s#__GITHUB_TOKEN_SECRET__#${github_token_secret}#g" \
    -e "s#__PREVIEW_ID__#${preview_id}#g" \
    -e "s#__FRONTEND_MODE__#${frontend_mode}#g" \
    -e "s#__COMMIT_SHA__#${commit_sha}#g" \
    -e "s#__REPO_URL__#${repo_url}#g" \
    -e "s#__TTL_HOURS__#${ttl_hours}#g" \
    -e "s#__PROJECT_ID__#${project_id}#g" \
    -e "s#__REGION__#${region}#g" \
    -e "s#__ARTIFACT_REPO__#${artifact_repo}#g" \
    -e "s#__MCAP_PREVIEW_HOST__#${mcap_preview_host}#g" \
    -e "s#__PREVIEW_NAMESPACE__#${preview_namespace}#g" \
    -e "s#__PREVIEW_PUBLIC_BASE_URL__#${preview_public_base_url}#g" \
    -e "s#__PREVIEW_BUILDKITD_NAMESPACE__#${buildkitd_namespace}#g" \
    -e "s#__PREVIEW_BUILDKITD_DEPLOY__#${buildkitd_deploy}#g" \
    "${template}"
}

condition_status() {
  kubectl get pipelinerun "${run_name}" -n "${tekton_namespace}" \
    -o jsonpath='{.status.conditions[?(@.type=="Succeeded")].status}' 2>/dev/null || true
}

condition_reason() {
  kubectl get pipelinerun "${run_name}" -n "${tekton_namespace}" \
    -o jsonpath='{.status.conditions[?(@.type=="Succeeded")].reason}' 2>/dev/null || true
}

taskrun_name() {
  kubectl get taskrun -n "${tekton_namespace}" \
    -l "tekton.dev/pipelineRun=${run_name}" \
    -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true
}

pipeline_pod_name() {
  kubectl get pods -n "${tekton_namespace}" \
    -l "tekton.dev/pipelineRun=${run_name}" \
    -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true
}

step_state() {
  local taskrun="$1"
  local step="$2"
  [[ -n "${taskrun}" ]] || { echo "pending"; return 0; }
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

latest_progress_hint() {
  local pod="$1"
  local container="$2"
  [[ -n "${pod}" && -n "${container}" ]] || return 0
  kubectl logs -n "${tekton_namespace}" "${pod}" -c "${container}" --tail=80 2>/dev/null \
    | awk '
      /^TIMING / { line=$0; next }
      /^#[0-9]+ [0-9.]+ TIMING / { sub(/^#[0-9]+ [0-9.]+ /, ""); line=$0; next }
      /^Skipping frontend image build/ { line=$0; next }
      /^Registry credentials written/ { line=$0; next }
      END { if (line != "") print line }
    ' || true
}

print_progress_snapshot() {
  local taskrun pod current_step="" current_container="" hint="" state=""
  taskrun="$(taskrun_name)"
  pod="$(pipeline_pod_name)"
  echo "progress:"
  for step in fetch-source build-images deploy-previews; do
    state="$(step_state "${taskrun}" "${step}")"
    printf '  - %-16s %s [%s]\n' "${step}" "$(step_label "${step}")" "${state}"
    if [[ "${state}" == "running" ]]; then
      current_step="${step}"
      current_container="step-${step}"
    fi
  done
  if [[ -n "${current_step}" ]]; then
    hint="$(latest_progress_hint "${pod}" "${current_container}")"
    echo "  current: ${current_step}"
    if [[ -n "${hint}" ]]; then
      echo "  latest:  ${hint}"
    fi
  fi
}

pipeline_result() {
  local name="$1"
  local value
  value="$(kubectl get pipelinerun "${run_name}" -n "${tekton_namespace}" \
    -o jsonpath='{range .status.pipelineResults[*]}{.name}{"\t"}{.value}{"\n"}{end}' 2>/dev/null \
    | awk -F '\t' -v n="${name}" '$1 == n {print $2; exit}' || true
  )"
  if [[ -n "${value}" ]]; then
    printf '%s' "${value}"
    return 0
  fi

  kubectl get taskrun -n "${tekton_namespace}" \
    -l "tekton.dev/pipelineRun=${run_name}" \
    -o jsonpath='{range .items[*].status.results[*]}{.name}{"\t"}{.value}{"\n"}{end}' 2>/dev/null \
    | awk -F '\t' -v n="${name}" '$1 == n {print $2; exit}' || true
}

show_logs() {
  local pods
  pods="$(kubectl get pods -n "${tekton_namespace}" \
    -l "tekton.dev/pipelineRun=${run_name}" \
    -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null || true)"
  while IFS= read -r pod; do
    [[ -n "${pod}" ]] || continue
    echo ""
    echo "===== logs: ${pod} ====="
    kubectl logs -n "${tekton_namespace}" "${pod}" --all-containers --tail=500 || true
  done <<< "${pods}"
}

show_success_logs() {
  local pods
  pods="$(kubectl get pods -n "${tekton_namespace}" \
    -l "tekton.dev/pipelineRun=${run_name}" \
    -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null || true)"
  while IFS= read -r pod; do
    [[ -n "${pod}" ]] || continue
    echo ""
    echo "===== summary: ${pod} ====="
    kubectl logs -n "${tekton_namespace}" "${pod}" -c step-deploy-previews --tail=300 \
      | awk '
        /^TIMING / { print; next }
        /^#[0-9]+ [0-9.]+ TIMING / {
          sub(/^#[0-9]+ [0-9.]+ /, "")
          print
          next
        }
        /^Fast preview deployment complete$/ { print; next }
        /^commit\/preview:/ { print; next }
        /^(backend|frontend) service:/ { print; next }
        /^(backend|frontend) workload:/ { print; next }
        /^(backend|frontend) content hash:/ { print; next }
        /^(backend|frontend) url:/ { print; next }
        /^(backend|frontend) deterministic url:/ { print; next }
        /^expires epoch:/ { print; next }
        /^ttl hours:/ { print; next }
        /^frontend mode:/ { print; next }
        /^local command:/ { print; next }
        /^  cd Frontend / { print; next }
      ' || true
  done <<< "${pods}"
}

if [[ "${preview_bootstrap}" == "true" ]]; then
  echo "Ensuring preview buildkitd is applied..."
  kubectl apply -f "${buildkit_manifest}" >/dev/null
  kubectl rollout status "deploy/${buildkitd_deploy}" -n "${buildkitd_namespace}" --timeout=180s >/dev/null

  echo "Ensuring preview env reader RBAC is applied..."
  kubectl apply -f "${env_rbac_manifest}" >/dev/null
else
  echo "Skipping preview bootstrap because PREVIEW_BOOTSTRAP=${preview_bootstrap}"
fi

echo "Ensuring GitHub token secret: ${tekton_namespace}/${github_token_secret}"
ensure_github_token_secret

echo "Creating fast preview PipelineRun: ${run_name}"
render_template | kubectl create --request-timeout=90s -f -

echo ""
echo "Predicted preview URLs:"
echo "backend:  ${api_url}"
if [[ "${frontend_mode}" == "remote" ]]; then
  echo "frontend: ${web_url}"
else
  echo "frontend: local"
  echo "local command: cd Frontend && VITE_PREVIEW_ID=${preview_id} npm run dev"
  echo "check:     概览页左下角应显示 v<package.json> (preview/${preview_id})"
fi
echo ""
echo "Tekton:"
echo "kubectl get pipelinerun ${run_name} -n ${tekton_namespace}"
echo "watch:    bash deploy/preview/preview-status.sh --watch ${run_name}"

if [[ "${wait_for_completion}" != "true" ]]; then
  exit 0
fi

echo ""
echo "Waiting for PipelineRun completion..."
deadline=$((SECONDS + 3600))
last_status=""
last_progress=""
while (( SECONDS < deadline )); do
  status="$(condition_status)"
  reason="$(condition_reason)"
  if [[ "${status}:${reason}" != "${last_status}" ]]; then
    echo "status=${status:-Pending} reason=${reason:-Pending}"
    last_status="${status}:${reason}"
  fi
  progress="$(print_progress_snapshot)"
  if [[ "${progress}" != "${last_progress}" ]]; then
    echo "${progress}"
    last_progress="${progress}"
  fi
  case "${status}" in
    True)
      show_success_logs
      backend_result="$(pipeline_result backend-url)"
      frontend_result="$(pipeline_result frontend-url)"
      echo ""
      echo "Fast preview succeeded:"
      echo "backend:  ${backend_result:-${api_url}}"
      if [[ "${frontend_mode}" == "remote" ]]; then
        echo "frontend: ${frontend_result:-${web_url}}"
      else
        echo "local command: cd Frontend && VITE_PREVIEW_ID=${preview_id} npm run dev"
        echo "check:     概览页左下角应显示 v<package.json> (preview/${preview_id})"
      fi
      exit 0
      ;;
    False)
      show_logs
      die "PipelineRun failed: ${reason:-unknown}"
      ;;
  esac
  sleep 5
done

show_logs
die "timed out waiting for PipelineRun ${run_name}"
