#!/usr/bin/env bash
# One-shot diagnostics for cyber-databrew backend in a namespace.
# Usage: K8S_NAMESPACE=cyber-databrew-dev ./deploy/k8s/diagnose.sh
set -euo pipefail

NS="${K8S_NAMESPACE:-cyber-databrew-dev}"
APP_LABEL="app.kubernetes.io/name=cyber-databrew"
DEPLOY="cyber-databrew-backend"
CONTAINER="api"

echo "=== context ==="
kubectl config current-context 2>&1 || true
echo
echo "=== namespace ${NS} ==="
kubectl get ns "${NS}" 2>&1 || { echo "namespace missing? kubectl apply -f deploy/k8s/namespaces.yaml"; exit 1; }
echo
echo "=== deployment ==="
kubectl -n "${NS}" get deploy "${DEPLOY}" -o wide 2>&1 || true
echo
echo "=== pods ==="
kubectl -n "${NS}" get pods -l "${APP_LABEL}" -o wide 2>&1 || true
echo
echo "=== events (recent) ==="
kubectl -n "${NS}" get events --sort-by='.lastTimestamp' 2>&1 | tail -25
echo
POD="$(kubectl -n "${NS}" get pods -l "${APP_LABEL}" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
if [[ -n "${POD}" ]]; then
	echo
	echo "=== describe pod ${POD} (status/conditions) ==="
	kubectl -n "${NS}" describe pod "${POD}" 2>&1 | tail -60
	echo
	echo "=== logs ${POD} / ${CONTAINER} (last 120 lines) ==="
	kubectl -n "${NS}" logs "${POD}" -c "${CONTAINER}" --tail=120 2>&1 || \
		kubectl -n "${NS}" logs "${POD}" --all-containers --tail=120 2>&1 || true
else
	echo
	echo "No pod found for label ${APP_LABEL}. Check deployment and image pull secrets."
fi
echo
echo "=== image in deployment spec ==="
kubectl -n "${NS}" get deploy "${DEPLOY}" -o jsonpath='{.spec.template.spec.containers[0].image}{"\n"}' 2>&1 || true
