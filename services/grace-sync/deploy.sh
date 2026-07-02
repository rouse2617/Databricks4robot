#!/bin/bash
set -euo pipefail

PROJECT_ID="green-valley-442103"
REGION="us-central1"
JOB_NAME="grace-sync"
IMAGE="gcr.io/${PROJECT_ID}/${JOB_NAME}:latest"

# GCP Secret Manager references for sensitive values.
# The Cloud Run Job's runtime SA must have roles/secretmanager.secretAccessor
# on these secrets (e.g. `secretmanager.secretAccessor` for the deploy SA,
# and the runtime SA used to execute the job).
GRACE_API_SECRET="grace-api-dev"   # JSON: {"AUTH_USERNAME": "...", "AUTH_PASSWORD": "..."}
DATABREW_TOKEN_SECRET="databrew-dev-token"   # plain token, only mounted if you have a GCP secret for it

# Non-sensitive env vars (no secret material here).
ENV_VARS="GRACE_USERNAME=grace-service-dev,\
GRACE_API_URL=https://dev.cyber-grace.pages.dev/api,\
PIPELINE_TEMPLATE_ID=e223bac8-df8f-4f2a-b79c-e2ce4aaac3ff,\
TARGET_ID=a03ad932-397f-4a3b-a3d8-1cfb13a6dd54,\
SYNC_LOOKBACK_HOURS=0,\
SYNC_LOOKBACK_MINUTES=30,\
FEISHU_BOT_WEBHOOK=https://open.feishu.cn/open-apis/bot/v2/hook/d611e124-3bde-4688-8dac-93b86edb2c7a,\
DATABREW_URL=https://cyber-databrew-dev.cyberorigin.ai/api,\
DATABREW_TOKEN=dev-token"

# Sensitive env vars mapped to Secret Manager: the runtime value is sourced
# from the latest version of each named secret.
SECRET_ENVS="GRACE_PASSWORD=${GRACE_API_SECRET}:AUTH_PASSWORD:latest"

echo "🏗️  构建 Docker 镜像..."
docker buildx build --platform linux/amd64 -t "${IMAGE}" --push .
echo "✅ 镜像推送成功"

echo ""
echo "📦 更新 Cloud Run Job..."
# Reset env first so deleted vars don't linger.
gcloud run jobs update "${JOB_NAME}" \
  --region="${REGION}" \
  --project="${PROJECT_ID}" \
  --image="${IMAGE}" \
  --clear-env-vars

# Plain (non-sensitive) env vars.
gcloud run jobs update "${JOB_NAME}" \
  --region="${REGION}" \
  --project="${PROJECT_ID}" \
  --set-env-vars="${ENV_VARS}"

# Sensitive env vars via Secret Manager.
gcloud run jobs update "${JOB_NAME}" \
  --region="${REGION}" \
  --project="${PROJECT_ID}" \
  --set-secrets="${SECRET_ENVS}"

echo "✅ Cloud Run Job 部署成功"
echo ""
echo "环境变量已设置："
gcloud run jobs describe "${JOB_NAME}" \
  --region="${REGION}" \
  --project="${PROJECT_ID}" \
  --format='table(spec.template.spec.containers[0].env[].name)'
