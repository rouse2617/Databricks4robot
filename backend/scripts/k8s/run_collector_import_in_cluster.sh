#!/usr/bin/env bash
# Apply ConfigMap + Job to run import inside GKE (no kubectl stdout pollution; collector private IP OK).
set -euo pipefail

NS="${NS:-cyber-databrew-dev}"
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
CM_NAME="${CM_NAME:-collector-databrew-import-script}"
JOB_FILE="$(dirname "$0")/collector-databrew-import-job.yaml"

kubectl -n "$NS" create configmap "$CM_NAME" \
  --from-file=import_collector_postgres_to_databrew.py="$ROOT/backend/scripts/import_collector_postgres_to_databrew.py" \
  --from-file=requirements-collector-import.txt="$ROOT/backend/scripts/requirements-collector-import.txt" \
  -o yaml --dry-run=client | kubectl apply -f -

echo "ConfigMap $CM_NAME applied. Ensure Secret collector-databrew-import-env exists in $NS."
echo "Deleting prior Job (if any) so apply is idempotent..."
kubectl -n "$NS" delete job collector-databrew-import --ignore-not-found=true

kubectl apply -f "$JOB_FILE"
echo "Follow logs: kubectl -n $NS logs -f job/collector-databrew-import"
