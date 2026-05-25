#!/usr/bin/env bash
set -euo pipefail

PROJECT_ID="${PROJECT_ID:-green-valley-442103}"
REGION="${REGION:-us-central1}"
SERVICE_NAME="${SERVICE_NAME:-mcap-preview-dev}"
IMAGE="${IMAGE:-us-central1-docker.pkg.dev/green-valley-442103/video-proc-images/mcap-preview:dev-latest}"

UPSTREAM_BASE_URL="${UPSTREAM_BASE_URL:-https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app}"
GCS_PAGE_SIZE_BYTES="${GCS_PAGE_SIZE_BYTES:-1048576}"
GCS_PAGE_CACHE_BYTES="${GCS_PAGE_CACHE_BYTES:-67108864}"
LOG_LEVEL="${LOG_LEVEL:-info}"
LOG_FORMAT="${LOG_FORMAT:-json}"
DATABREW_TOKEN_PASSTHROUGH="${DATABREW_TOKEN_PASSTHROUGH:-true}"

echo "Deploying ${SERVICE_NAME} to Cloud Run (${REGION}, ${PROJECT_ID})"

gcloud run deploy "${SERVICE_NAME}" \
  --quiet \
  --project "${PROJECT_ID}" \
  --region "${REGION}" \
  --platform managed \
  --image "${IMAGE}" \
  --port 8090 \
  --allow-unauthenticated \
  --min-instances 0 \
  --max-instances 10 \
  --cpu 8 \
  --memory 8Gi \
  --cpu-boost \
  --concurrency 2 \
  --timeout 600 \
  --set-env-vars "UPSTREAM_BASE_URL=${UPSTREAM_BASE_URL},GCS_PAGE_SIZE_BYTES=${GCS_PAGE_SIZE_BYTES},GCS_PAGE_CACHE_BYTES=${GCS_PAGE_CACHE_BYTES},LOG_LEVEL=${LOG_LEVEL},LOG_FORMAT=${LOG_FORMAT},DATABREW_TOKEN_PASSTHROUGH=${DATABREW_TOKEN_PASSTHROUGH}"

echo "Deployment complete."
gcloud run services describe "${SERVICE_NAME}" \
  --project "${PROJECT_ID}" \
  --region "${REGION}" \
  --platform managed \
  --format='value(status.url)'
