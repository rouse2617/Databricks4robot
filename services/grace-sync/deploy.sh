#!/bin/bash
set -euo pipefail

ENV="${1:-dev}"  # dev or prod
PROJECT_ID="green-valley-442103"
REGION="us-central1"

echo "=== 部署 grace-sync 到 ${ENV} 环境 ==="

if [ "$ENV" = "prod" ]; then
  JOB_NAME="grace-sync-prod"
  IMAGE="gcr.io/${PROJECT_ID}/grace-sync-prod:latest"

  # Prod 配置
  GRACE_USERNAME="grace-service-prod"
  GRACE_API_URL="https://grace.cyberorigin.ai/api"
  GRACE_PASSWORD_SECRET="grace-api-prod"

  DATABREW_URL="https://cyber-databrew.cyberorigin.ai/api/v1"
  DATABREW_TOKEN_SECRET="cyber-databrew-prod-databrew-token"

  PIPELINE_TEMPLATE_ID="9138dc6d-8577-48e3-81d6-c7c2cde70adc"
  TARGET_ID="video-proc-prod"

  FEISHU_BOT_WEBHOOK="https://open.feishu.cn/open-apis/bot/v2/hook/d611e124-3bde-4688-8dac-93b86edb2c7a"

  SYNC_LOOKBACK_HOURS=0
  SYNC_LOOKBACK_MINUTES=30
else
  JOB_NAME="grace-sync"
  IMAGE="gcr.io/${PROJECT_ID}/grace-sync:latest"

  # Dev 配置
  GRACE_USERNAME="grace-service-dev"
  GRACE_API_URL="https://main.cyber-grace.pages.dev/api"
  GRACE_PASSWORD_SECRET="grace-api-dev"

  DATABREW_URL="https://cyber-databrew-dev.cyberorigin.ai/api/v1"
  DATABREW_TOKEN="dev-token"
  DATABREW_TOKEN_SECRET=""

  PIPELINE_TEMPLATE_ID="05c18471-9182-4af6-a373-31e802ea5121"
  TARGET_ID="a03ad932-397f-4a3b-a3d8-1cfb13a6dd54"

  FEISHU_BOT_WEBHOOK="https://open.feishu.cn/open-apis/bot/v2/hook/d611e124-3bde-4688-8dac-93b86edb2c7a"

  SYNC_LOOKBACK_HOURS=0
  SYNC_LOOKBACK_MINUTES=30
fi

ENV_VARS="GRACE_USERNAME=${GRACE_USERNAME},\
GRACE_API_URL=${GRACE_API_URL},\
PIPELINE_TEMPLATE_ID=${PIPELINE_TEMPLATE_ID},\
TARGET_ID=${TARGET_ID},\
SYNC_LOOKBACK_HOURS=${SYNC_LOOKBACK_HOURS},\
SYNC_LOOKBACK_MINUTES=${SYNC_LOOKBACK_MINUTES},\
FEISHU_BOT_WEBHOOK=${FEISHU_BOT_WEBHOOK},\
DATABREW_URL=${DATABREW_URL}"

if [ "$ENV" = "dev" ]; then
  ENV_VARS="${ENV_VARS},DATABREW_TOKEN=${DATABREW_TOKEN}"
fi

echo "🏗️  构建 Docker 镜像..."
docker buildx build --platform linux/amd64 -t "${IMAGE}" --push .
echo "✅ 镜像推送成功"

echo ""
echo "📦 更新 Cloud Run Job..."

# 清理旧的 env vars 后再设置
gcloud run jobs update "${JOB_NAME}" \
  --region="${REGION}" \
  --project="${PROJECT_ID}" \
  --image="${IMAGE}" \
  --clear-env-vars 2>/dev/null || true  # 可能 job 还不存在

# 设置非敏感 env vars
gcloud run jobs update "${JOB_NAME}" \
  --region="${REGION}" \
  --project="${PROJECT_ID}" \
  --image="${IMAGE}" \
  --set-env-vars="${ENV_VARS}"

if [ "$ENV" = "prod" ]; then
  # 敏感 env vars via Secret Manager
  gcloud run jobs update "${JOB_NAME}" \
    --region="${REGION}" \
    --project="${PROJECT_ID}" \
    --set-secrets="GRACE_PASSWORD=${GRACE_PASSWORD_SECRET}:latest,DATABREW_TOKEN=${DATABREW_TOKEN_SECRET}:latest"
else
  # Dev: GRACE_PASSWORD via Secret Manager
  gcloud run jobs update "${JOB_NAME}" \
    --region="${REGION}" \
    --project="${PROJECT_ID}" \
    --set-secrets="GRACE_PASSWORD=${GRACE_PASSWORD_SECRET}:latest"
fi

echo ""
echo "✅ Cloud Run Job 部署成功 (${JOB_NAME})"
echo ""
echo "环境变量："
gcloud run jobs describe "${JOB_NAME}" \
  --region="${REGION}" \
  --project="${PROJECT_ID}" \
  --format='table(spec.template.spec.template.spec.containers[0].env[].name)'

echo ""
echo "=== 部署完成 ==="
echo "运行测试: gcloud run jobs execute ${JOB_NAME} --region=${REGION} --project=${PROJECT_ID}"
