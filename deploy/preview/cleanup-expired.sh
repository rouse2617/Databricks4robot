#!/usr/bin/env bash
set -euo pipefail

PROJECT_ID="${PROJECT_ID:-green-valley-442103}"
REGION="${REGION:-us-central1}"
PREVIEW_NAMESPACE="${PREVIEW_NAMESPACE:-cyber-databrew-dev}"
NOW_EPOCH="${NOW_EPOCH:-$(date -u +%s)}"
DRY_RUN="${DRY_RUN:-false}"
PREVIEW_CLEANUP_TARGET="${PREVIEW_CLEANUP_TARGET:-k8s}"

cleanup_k8s() {
  kubectl get deployments,services \
    -n "${PREVIEW_NAMESPACE}" \
    -l preview=true \
    -o jsonpath='{range .items[*]}{.kind}{"\t"}{.metadata.name}{"\t"}{.metadata.labels.expires-epoch}{"\n"}{end}' |
  while IFS=$'\t' read -r kind name expires_epoch; do
    [[ -n "${kind}" ]] || continue
    [[ -n "${name}" ]] || continue
    [[ -n "${expires_epoch}" ]] || continue
    [[ "${expires_epoch}" =~ ^[0-9]+$ ]] || continue

    if (( expires_epoch > NOW_EPOCH )); then
      continue
    fi

    echo "expired preview: ${kind}/${name} namespace=${PREVIEW_NAMESPACE} expires_epoch=${expires_epoch}"
    if [[ "${DRY_RUN}" == "true" ]]; then
      continue
    fi

    kubectl delete "${kind}/${name}" -n "${PREVIEW_NAMESPACE}"
  done

  kubectl get deployments \
    -n "${PREVIEW_NAMESPACE}" \
    -l preview-content=true \
    -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.labels.app\.kubernetes\.io/component}{"\t"}{.metadata.labels.preview-content-hash}{"\n"}{end}' |
  while IFS=$'\t' read -r name component content_hash; do
    [[ -n "${name}" ]] || continue
    [[ -n "${component}" ]] || continue
    [[ -n "${content_hash}" ]] || continue

    active_aliases="$(kubectl get services \
      -n "${PREVIEW_NAMESPACE}" \
      -l "preview=true,app.kubernetes.io/component=${component},preview-content-hash=${content_hash}" \
      -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null || true)"
    if [[ -n "${active_aliases}" ]]; then
      continue
    fi

    echo "orphan preview content: Deployment/${name} namespace=${PREVIEW_NAMESPACE} content_hash=${content_hash}"
    if [[ "${DRY_RUN}" == "true" ]]; then
      continue
    fi

    kubectl delete "deployment/${name}" -n "${PREVIEW_NAMESPACE}"
  done
}

cleanup_cloudrun() {
  gcloud run services list \
    --project "${PROJECT_ID}" \
    --region "${REGION}" \
    --platform managed \
    --filter='metadata.labels.preview=true' \
    --format='value(metadata.name,metadata.labels.expires-epoch)' |
  while IFS=$'\t' read -r service expires_epoch; do
    [[ -n "${service}" ]] || continue
    [[ -n "${expires_epoch}" ]] || continue
    [[ "${expires_epoch}" =~ ^[0-9]+$ ]] || continue

    if (( expires_epoch > NOW_EPOCH )); then
      continue
    fi

    echo "expired preview: ${service} expires_epoch=${expires_epoch}"
    if [[ "${DRY_RUN}" == "true" ]]; then
      continue
    fi

    gcloud run services delete "${service}" \
      --project "${PROJECT_ID}" \
      --region "${REGION}" \
      --quiet
  done
}

case "${PREVIEW_CLEANUP_TARGET}" in
  k8s)
    cleanup_k8s
    ;;
  cloudrun)
    cleanup_cloudrun
    ;;
  *)
    echo "ERROR: PREVIEW_CLEANUP_TARGET must be k8s or cloudrun" >&2
    exit 2
    ;;
esac
