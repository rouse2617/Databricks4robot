#!/usr/bin/env bash
# One-time setup for the cyber-databrew Atlas migration Cloud Run Jobs.
#
# Creates one Job per environment (dev, prod). Each Job:
#   - runs arigaio/atlas distroless image built from backend/Dockerfile.migrate
#   - reaches CloudSQL over the existing Serverless VPC connector (cr-central-conn)
#   - reads DB_HOST/USER/PORT/NAME from the same ConfigMap the service uses
#   - reads DB_PASSWORD from the same Secret Manager secret the service uses
#   - default ENTRYPOINT applies pending migrations
#
# Re-run this script only when the Job's shape changes (different VPC connector,
# different service account, different memory/CPU). Image updates are handled by
# the GHA deploy workflow (`gcloud run jobs update --image=...`).
#
# Usage:
#   PROJECT_ID=green-valley-442103 REGION=us-central1 \
#   IMAGE=us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-migrate:init \
#   DB_SECRET=cyber-databrew-dev-postgres-password \
#   DB_HOST=172.27.160.7 DB_PORT=5432 DB_USER=postgres DB_NAME=cyber_databrew_dev \
#   ENV=dev \
#   bash deploy/cloudrun/create-migrate-jobs.sh
#
# Required env per env:
#   ENV          dev | prod
#   PROJECT_ID   GCP project
#   REGION       GCP region
#   IMAGE        initial migrate image tag
#   VPC_CONNECTOR Serverless VPC connector name (default cr-central-conn)
#   DB_SECRET    Secret Manager secret holding the DB password
#   DB_HOST/USER/PORT/NAME  plain env values matching the running service
#   SA           Cloud Run runtime service account (needs run.invoker + cloudsql client)

set -euo pipefail

ENV="${ENV:?ENV must be set (dev|prod)}"
PROJECT_ID="${PROJECT_ID:?PROJECT_ID required}"
REGION="${REGION:?REGION required}"
IMAGE="${IMAGE:?IMAGE required}"
VPC_CONNECTOR="${VPC_CONNECTOR:-cr-central-conn}"
DB_SECRET="${DB_SECRET:?DB_SECRET required (Secret Manager secret name)}"
DB_HOST="${DB_HOST:?DB_HOST required}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-cyber_databrew_${ENV}}"
SA="${SA:-cyber-databrew-cloudrun-${ENV}@${PROJECT_ID}.iam.gserviceaccount.com}"

JOB_NAME="cyber-databrew-migrate-${ENV}"

echo "==> Creating Cloud Run Job $JOB_NAME in $PROJECT_ID/$REGION"
echo "    image:        $IMAGE"
echo "    vpc-connector:$VPC_CONNECTOR"
echo "    sa:           $SA"
echo "    db:           $DB_USER@$DB_HOST:$DB_PORT/$DB_NAME (password from $DB_SECRET)"

gcloud run jobs create "$JOB_NAME" \
  --image="$IMAGE" \
  --region="$REGION" \
  --project="$PROJECT_ID" \
  --service-account="$SA" \
  --vpc-connector="$VPC_CONNECTOR" \
  --vpc-egress=private-ranges-only \
  --set-secrets="DB_PASSWORD=${DB_SECRET}:latest" \
  --set-env-vars="DB_HOST=$DB_HOST,DB_PORT=$DB_PORT,DB_USER=$DB_USER,DB_NAME=$DB_NAME,DB_SSLMODE=disable" \
  --max-retries=0 \
  --task-timeout=600 \
  --memory=512Mi \
  --cpu=1 \
  --no-execute-now

echo
echo "Done. To trigger:"
echo "  gcloud run jobs execute $JOB_NAME --region=$REGION --project=$PROJECT_ID --wait"
echo
echo "First-run baseline against an existing DB (records 000_initial as applied without re-running it):"
echo "  gcloud run jobs execute $JOB_NAME --region=$REGION --project=$PROJECT_ID --wait \\"
echo "    --args=\"--baseline,000_initial.sql\""
