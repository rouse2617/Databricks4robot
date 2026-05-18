#!/usr/bin/env bash
# Apply ConfigMap + Deployment + Service (requires Secret in target namespace first).
#
#   K8S_NAMESPACE=cyber-databrew-dev ./deploy/k8s/apply-base.sh
#   K8S_NAMESPACE=cyber-databrew-prod ./deploy/k8s/apply-base.sh
#   DRY_RUN=1 ...   # client-side dry-run only
#   SKIP_SECRET_CHECK=1 ...  # do not fail if secret missing (e.g. you will create it in the same pipeline)

set -euo pipefail

NS="${K8S_NAMESPACE:-cyber-databrew-dev}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="${SCRIPT_DIR}/base"

if [[ "${SKIP_SECRET_CHECK:-0}" != "1" ]]; then
	if ! kubectl get secret cyber-databrew-secrets -n "${NS}" &>/dev/null; then
		cat >&2 <<EOF
Missing Secret cyber-databrew-secrets in namespace ${NS}.

1) Copy the template and fill real values (do not commit the copy):
   cp ${BASE_DIR}/secret.example.yaml ${BASE_DIR}/secret.local.yaml
2) Apply:
   kubectl apply -n ${NS} -f ${BASE_DIR}/secret.local.yaml

Or use kubectl create secret generic ... --from-literal=... (see comments in secret.example.yaml).
EOF
		exit 1
	fi
fi

kubectl kustomize "${BASE_DIR}" >/dev/null
# Avoid "${arr[@]}" with set -u on Bash 3.2/macOS — empty array triggers "unbound variable".
if [[ "${DRY_RUN:-0}" == "1" ]]; then
	kubectl apply -n "${NS}" --dry-run=client -k "${BASE_DIR}"
else
	kubectl apply -n "${NS}" -k "${BASE_DIR}"
fi
echo "Applied kustomize base to namespace ${NS}."
