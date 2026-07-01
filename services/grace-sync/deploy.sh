#!/bin/bash
set -euo pipefail

PROJECT_ID="green-valley-442103"
REGION="us-central1"
JOB_NAME="grace-sync"
IMAGE="gcr.io/${PROJECT_ID}/${JOB_NAME}:latest"

# 所有环境变量定义在这里，避免在更新时丢失
ENV_VARS="GRACE_USERNAME=grace-service-dev,\
GRACE_PASSWORD=<REDACTED-GRACE-PASSWORD>,\
GRACE_API_URL=https://dev.cyber-grace.pages.dev/api,\
PIPELINE_TEMPLATE_ID=e223bac8-df8f-4f2a-b79c-e2ce4aaac3ff,\
TARGET_ID=a03ad932-397f-4a3b-a3d8-1cfb13a6dd54,\
SYNC_LOOKBACK_HOURS=1,\
FEISHU_BOT_WEBHOOK=https://open.feishu.cn/open-apis/bot/v2/hook/d611e124-3bde-4688-8dac-93b86edb2c7a,\
DATABREW_URL=https://cyber-databrew-dev.cyberorigin.ai/api,\
DATABREW_TOKEN=dev-token"

echo "🏗️  构建 Docker 镜像..."
docker buildx build --platform linux/amd64 -t "${IMAGE}" --push .
echo "✅ 镜像推送成功"

echo ""
echo "📦 更新 Cloud Run Job..."
# 先清空环境变量，再完整设置（避免留下旧变量）
gcloud run jobs update "${JOB_NAME}" \
  --region="${REGION}" \
  --project="${PROJECT_ID}" \
  --image="${IMAGE}" \
  --clear-env-vars

# 然后设置所有环境变量
gcloud run jobs update "${JOB_NAME}" \
  --region="${REGION}" \
  --project="${PROJECT_ID}" \
  --set-env-vars="${ENV_VARS}"

echo "✅ Cloud Run Job 部署成功"
echo ""
echo "环境变量已设置："
gcloud run jobs describe "${JOB_NAME}" \
  --region="${REGION}" \
  --project="${PROJECT_ID}" \
  --format='table(spec.template.spec.containers[0].env[].name)'
