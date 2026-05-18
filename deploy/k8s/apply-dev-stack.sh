#!/usr/bin/env bash
# Apply full dev stack in one namespace:
#   1) in-cluster dependencies (Postgres + Elasticsearch)
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

kubectl apply -n "${NS}" -k "${SCRIPT_DIR}/overlays/dev"
echo "Applied dev stack to namespace ${NS}."
