#!/usr/bin/env bash
# Build PR Feishu message and post to A 群 (FEISHU_ROUTE_BRANCH=main).
# Env: ACTION, MERGED, REPO, PR_NUMBER, PR_TITLE, PR_URL, HEAD_BRANCH, BASE_BRANCH, ACTOR
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

if [ "${ACTION:-}" = "closed" ] && [ "${MERGED:-}" != "true" ]; then
  echo "SKIP: PR closed without merge."
  exit 0
fi

if [ "${ACTION:-}" = "closed" ] && [ "${MERGED:-}" = "true" ]; then
  TITLE="【PR 已合并到 main】"
  HINT="代码已进入 main；merge 不会自动发布 prod。

需要上 prod 时，请在本 PR 页面评论（整行、精确匹配）：
  /deploy-cloudrun-prod-backend
  /deploy-cloudrun-prod-frontend
（backend / frontend 可各评一条；Tekton 会构建并部署 Cloud Run prod）

说明见仓库 .tekton/README.md"
  DETAIL="$(printf 'PR：#%s\n目标分支：%s\n标题：%s\n合并人：%s\n' \
    "${PR_NUMBER}" "${BASE_BRANCH}" "${PR_TITLE}" "${ACTOR}")"
elif [ "${ACTION:-}" = "synchronize" ]; then
  TITLE="【PR 有新提交】"
  HINT="GitHub Actions 会自动跑 pre-commit、test-integration、commitlint 等检查；若失败会另发【CI 失败】到本群。"
  DETAIL="$(printf 'PR：#%s\n分支：%s → %s\n标题：%s\n推送人：%s\n' \
    "${PR_NUMBER}" "${HEAD_BRANCH}" "${BASE_BRANCH}" "${PR_TITLE}" "${ACTOR}")"
else
  TITLE="【PR 已打开】"
  HINT="请 Review；合并前需 CI 通过。"
  DETAIL="$(printf 'PR：#%s\n分支：%s → %s\n标题：%s\n操作人：%s\n' \
    "${PR_NUMBER}" "${HEAD_BRANCH}" "${BASE_BRANCH}" "${PR_TITLE}" "${ACTOR}")"
fi

export FEISHU_ROUTE_BRANCH="main"
export FEISHU_MESSAGE="$(printf '%s\n\n仓库：%s\n%s\n\n说明：%s\n\n链接：%s\n' \
  "${TITLE}" "${REPO}" "${DETAIL}" "${HINT}" "${PR_URL}")"
bash "${ROOT}/.github/scripts/feishu-notify.sh"
