#!/usr/bin/env bash
# Apply full dev stack in one namespace:
#   1) in-cluster dependencies (Postgres/ES/Redpanda/Connect + CDC connector)
#   2) backend base manifests with dev config patch
#
# Usage:
#   K8S_NAMESPACE=cyber-databrew-dev ./deploy/k8s/apply-dev-stack.sh
#
set -euo pipefail

NS="${K8S_NAMESPACE:-cyber-databrew-dev}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

kubectl apply -n "${NS}" -k "${SCRIPT_DIR}/dev-deps"

kubectl -n "${NS}" rollout status statefulset/postgres --timeout=300s
kubectl -n "${NS}" rollout status statefulset/elasticsearch --timeout=300s
kubectl -n "${NS}" rollout status deployment/redpanda --timeout=300s
kubectl -n "${NS}" rollout status deployment/connect --timeout=300s
if ! kubectl -n "${NS}" wait --for=condition=complete job/cdc-connect-init --timeout=300s; then
	echo "ERROR: cdc-connect-init did not complete in namespace ${NS}." >&2
	echo "Inspect connector bootstrap logs with:" >&2
	echo "  kubectl -n ${NS} logs job/cdc-connect-init" >&2
	if [[ "${ALLOW_CDC_INIT_FAILURE:-0}" != "1" ]]; then
		echo "Set ALLOW_CDC_INIT_FAILURE=1 to continue anyway." >&2
		exit 1
	fi
	echo "Continuing because ALLOW_CDC_INIT_FAILURE=1." >&2
fi

kubectl apply -n "${NS}" -k "${SCRIPT_DIR}/dev"
echo "Applied dev stack to namespace ${NS}."
