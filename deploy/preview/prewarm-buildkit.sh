#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  deploy/preview/prewarm-buildkit.sh [--ref git-ref] [--frontend true|false] [--wait|--no-wait]

Environment overrides:
  PREVIEW_NAMESPACE           default: cyber-databrew-dev
  PREVIEW_PREWARM_MANIFEST    default: deploy/preview/k8s/preview-buildkit-prewarm.yaml
  PREWARM_REF                 default: dev
  PREWARM_FRONTEND            default: false
  PREWARM_WAIT_TIMEOUT        default: 20m
USAGE
}

die() {
  echo "ERROR: $*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

namespace="${PREVIEW_NAMESPACE:-cyber-databrew-dev}"
prewarm_ref="${PREWARM_REF:-dev}"
prewarm_frontend="${PREWARM_FRONTEND:-false}"
wait_for_completion=true
wait_timeout="${PREWARM_WAIT_TIMEOUT:-20m}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --ref)
      prewarm_ref="${2:-}"
      shift 2
      ;;
    --frontend)
      prewarm_frontend="${2:-}"
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
    *)
      die "unknown argument: $1"
      ;;
  esac
done

[[ -n "${prewarm_ref}" ]] || die "--ref must not be empty"
[[ "${prewarm_ref}" =~ ^[A-Za-z0-9._/-]+$ ]] || die "--ref contains unsupported characters: ${prewarm_ref}"
[[ "${prewarm_frontend}" == "true" || "${prewarm_frontend}" == "false" ]] || die "--frontend must be true or false"

require_cmd kubectl
require_cmd awk

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/../.." && pwd)"
manifest="${PREVIEW_PREWARM_MANIFEST:-${script_dir}/k8s/preview-buildkit-prewarm.yaml}"
[[ -f "${manifest}" ]] || die "missing manifest: ${manifest}"

job_name="preview-buildkit-prewarm-$(date -u +%H%M%S)"

echo "Ensuring BuildKit prewarm CronJob exists..."
kubectl apply -f "${manifest}" >/dev/null

echo "Creating BuildKit prewarm Job: ${namespace}/${job_name}"
kubectl create job "${job_name}" \
  -n "${namespace}" \
  --from=cronjob/preview-buildkit-prewarm \
  --dry-run=client \
  -o yaml \
  | awk -v ref="${prewarm_ref}" -v frontend="${prewarm_frontend}" '
      /^[[:space:]]*- name: PREWARM_REF$/ {
        print
        getline
        sub(/value: .*/, "value: \"" ref "\"")
        print
        next
      }
      /^[[:space:]]*- name: PREWARM_FRONTEND$/ {
        print
        getline
        sub(/value: .*/, "value: \"" frontend "\"")
        print
        next
      }
      { print }
    ' \
  | kubectl apply -f - >/dev/null

echo "Prewarm ref: ${prewarm_ref}"
echo "Prewarm frontend: ${prewarm_frontend}"

if [[ "${wait_for_completion}" != "true" ]]; then
  echo "Job created. Watch with:"
  echo "  kubectl get job ${job_name} -n ${namespace}"
  exit 0
fi

echo "Waiting for Job completion..."
if ! kubectl wait --for=condition=complete "job/${job_name}" -n "${namespace}" --timeout="${wait_timeout}"; then
  echo ""
  echo "===== prewarm logs: ${job_name} ====="
  kubectl logs "job/${job_name}" -n "${namespace}" --all-containers=true --tail=-1 || true
  exit 1
fi

echo ""
echo "===== prewarm logs: ${job_name} ====="
kubectl logs "job/${job_name}" -n "${namespace}" --all-containers=true --tail=-1

echo ""
echo "BuildKit prewarm complete: ${job_name}"
