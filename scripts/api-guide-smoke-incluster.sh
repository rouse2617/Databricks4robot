#!/usr/bin/env bash
# Run docs/review/api-guide.md smoke against the backend Service inside GKE (bypasses IAP).
# Prereq: kubectl context points at the cluster; namespace cyber-databrew-dev exists.
#
# Usage:
#   bash scripts/api-guide-smoke-incluster.sh
# Optional:
#   KUBE_CONTEXT=gke_... NS=cyber-databrew-dev bash scripts/api-guide-smoke-incluster.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CTX="${KUBE_CONTEXT:-}"
NS="${NS:-cyber-databrew-dev}"
K=(kubectl)
[[ -n "$CTX" ]] && K+=(--context="$CTX")
K+=(-n "$NS")

echo "=== sync ConfigMap api-guide-smoke-script ($NS) ==="
"${K[@]}" create configmap api-guide-smoke-script \
	--from-file=smoke.sh="$ROOT/scripts/api-guide-smoke.sh" \
	-o yaml --dry-run=client | "${K[@]}" apply -f -

echo "=== run Job api-guide-smoke ==="
"${K[@]}" delete job api-guide-smoke --ignore-not-found
"${K[@]}" apply -f "$ROOT/deploy/k8s/jobs/api-guide-smoke-job.yaml"

echo "=== wait for Job (success or failure) ==="
for _ in $(seq 1 90); do
	S="$("${K[@]}" get job api-guide-smoke -o jsonpath='{.status.succeeded}' 2>/dev/null || echo 0)"
	F="$("${K[@]}" get job api-guide-smoke -o jsonpath='{.status.failed}' 2>/dev/null || echo 0)"
	if [[ "$S" == "1" ]]; then
		"${K[@]}" logs job/api-guide-smoke
		exit 0
	fi
	if [[ "$F" == "1" ]]; then
		"${K[@]}" logs job/api-guide-smoke || true
		exit 1
	fi
	sleep 2
done
echo "timeout waiting for job"
"${K[@]}" logs job/api-guide-smoke || true
exit 1
