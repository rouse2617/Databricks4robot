#!/usr/bin/env bash
# Deploy the bronze-incremental Cloud Run Job + Cloud Scheduler trigger.
#
# Usage:
#   bash deploy/cloudrun/bronze-incremental/deploy.sh
#
# Env overrides:
#   PROJECT, REGION, JOB_NAME, SCHEDULE, GSA, IMAGE
#
# Workflow:
#   1. docker build --platform=linux/amd64 (per repo rule: NEVER Cloud Build)
#   2. docker push to Artifact Registry
#   3. gcloud run jobs deploy
#   4. gcloud scheduler jobs create http (first time) or update
#
# Requires:
#   - Local Docker daemon (Apple Silicon Macs: --platform=linux/amd64 forced)
#   - gcloud auth configure-docker us-central1-docker.pkg.dev (once)

set -euo pipefail

PROJECT="${PROJECT:-green-valley-442103}"
REGION="${REGION:-us-central1}"
JOB_NAME="${JOB_NAME:-bronze-incremental}"
SCHEDULER_NAME="${SCHEDULER_NAME:-bronze-incremental-trigger}"
SCHEDULE="${SCHEDULE:-0 * * * *}"
GSA="${GSA:-cyber-databrew-dev@${PROJECT}.iam.gserviceaccount.com}"
IMAGE_REPO="${IMAGE_REPO:-${REGION}-docker.pkg.dev/${PROJECT}/rick-cyber-databrew-images/${JOB_NAME}}"
IMAGE_TAG="${IMAGE_TAG:-dev-latest}"
IMAGE="${IMAGE:-${IMAGE_REPO}:${IMAGE_TAG}}"

# PG conn (mirrors backend Cloud Run env — private IP via VPC connector).
DB_HOST="${DB_HOST:-172.27.160.7}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-cyber_databrew_dev}"
DB_PASSWORD_SECRET="${DB_PASSWORD_SECRET:-cyber-databrew-dev-postgres-password}"
VPC_CONNECTOR="${VPC_CONNECTOR:-cr-central-conn}"
VPC_EGRESS="${VPC_EGRESS:-private-ranges-only}"

# Script root (where Dockerfile lives).
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "${SCRIPT_DIR}"

echo "=== 1/4 docker build (linux/amd64) ==="
docker build --platform=linux/amd64 -t "${IMAGE}" .

echo "=== 2/4 docker push ==="
docker push "${IMAGE}"

echo "=== 3/4 gcloud run jobs deploy ==="
gcloud run jobs deploy "${JOB_NAME}" \
  --project="${PROJECT}" \
  --region="${REGION}" \
  --image="${IMAGE}" \
  --service-account="${GSA}" \
  --vpc-connector="${VPC_CONNECTOR}" \
  --vpc-egress="${VPC_EGRESS}" \
  --task-timeout=5m \
  --max-retries=1 \
  --cpu=1 \
  --memory=1Gi \
  --set-env-vars="DB_HOST=${DB_HOST},DB_PORT=${DB_PORT},DB_USER=${DB_USER},DB_NAME=${DB_NAME},GCP_PROJECT=${PROJECT},BQ_DATASET=${BQ_DATASET:-lakehouse_bronze},BQ_TABLE=${BQ_TABLE:-bronze_asset_events},SILVER_QUALITY_ENABLED=${SILVER_QUALITY_ENABLED:-true},SILVER_QUALITY_TABLE=${SILVER_QUALITY_TABLE:-silver_asset_quality_current},SILVER_QUALITY_MODE=${SILVER_QUALITY_MODE:-incremental},SILVER_DIRTY_ID_BATCH=${SILVER_DIRTY_ID_BATCH:-1000},BQ_SILVER_DATASET=${BQ_SILVER_DATASET:-${BQ_DATASET:-lakehouse_bronze}},BQ_SILVER_TABLE=${BQ_SILVER_TABLE:-silver_asset_quality_current},CURSOR_SOURCE=${CURSOR_SOURCE:-checkpoint},CURSOR_SCAN_FALLBACK=${CURSOR_SCAN_FALLBACK:-false},BATCH_SIZE=${BATCH_SIZE:-2000},SILVER_BATCH_SIZE=${SILVER_BATCH_SIZE:-2000}" \
  --set-secrets="DB_PASSWORD=${DB_PASSWORD_SECRET}:latest"

echo "=== 4/4 gcloud scheduler ==="
JOB_URI="https://${REGION}-run.googleapis.com/apis/run.googleapis.com/v1/namespaces/${PROJECT}/jobs/${JOB_NAME}:run"

# Grant Scheduler-invoking SA the run.invoker on this job (idempotent).
gcloud run jobs add-iam-policy-binding "${JOB_NAME}" \
  --project="${PROJECT}" \
  --region="${REGION}" \
  --member="serviceAccount:${GSA}" \
  --role=roles/run.invoker

# Create or update scheduler job.
if gcloud scheduler jobs describe "${SCHEDULER_NAME}" --location="${REGION}" --project="${PROJECT}" >/dev/null 2>&1; then
  echo "scheduler job exists; updating schedule"
  gcloud scheduler jobs update http "${SCHEDULER_NAME}" \
    --project="${PROJECT}" \
    --location="${REGION}" \
    --schedule="${SCHEDULE}" \
    --uri="${JOB_URI}" \
    --http-method=POST \
    --oauth-service-account-email="${GSA}"
else
  echo "creating scheduler job"
  gcloud scheduler jobs create http "${SCHEDULER_NAME}" \
    --project="${PROJECT}" \
    --location="${REGION}" \
    --schedule="${SCHEDULE}" \
    --uri="${JOB_URI}" \
    --http-method=POST \
    --oauth-service-account-email="${GSA}"
fi

echo
echo "=== DONE ==="
echo "Job:        ${JOB_NAME}"
echo "Scheduler:  ${SCHEDULER_NAME} (${SCHEDULE})"
echo "Image:      ${IMAGE}"
echo
echo "Manual trigger to verify (will start immediately):"
echo "  gcloud run jobs execute ${JOB_NAME} --region=${REGION} --project=${PROJECT}"
echo
echo "Watch logs:"
echo "  gcloud run jobs executions list --job=${JOB_NAME} --region=${REGION} --project=${PROJECT}"
echo "  gcloud logging read 'resource.type=\"cloud_run_job\" AND resource.labels.job_name=\"${JOB_NAME}\"' --limit=50 --format=value(textPayload) --project=${PROJECT}"
