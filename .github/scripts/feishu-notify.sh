#!/usr/bin/env bash
# Post a text message to Feishu (branch-routed webhooks).
# Env:
#   FEISHU_ROUTE_BRANCH — branch name used to pick webhook (e.g. main vs feature)
#   FEISHU_MESSAGE      — message body (plain text)
#   FEISHU_BOT_WEBHOOK_MAIN / FEISHU_BOT_WEBHOOK_BRANCH / FEISHU_BOT_WEBHOOK (from caller)
set -euo pipefail

branch="${FEISHU_ROUTE_BRANCH:-}"
message="${FEISHU_MESSAGE:-}"

if [ -z "${message}" ]; then
  echo "SKIP: FEISHU_MESSAGE is empty."
  exit 0
fi

if [ "${branch}" = "main" ]; then
  webhook="${FEISHU_BOT_WEBHOOK_MAIN:-}"
  target="main"
else
  webhook="${FEISHU_BOT_WEBHOOK_BRANCH:-}"
  target="branch"
fi

if [ -z "${webhook}" ] && [ -n "${FEISHU_BOT_WEBHOOK:-}" ]; then
  echo "WARN: branch-specific webhook unset; using FEISHU_BOT_WEBHOOK."
  webhook="${FEISHU_BOT_WEBHOOK}"
fi

if [ -z "${webhook}" ]; then
  echo "SKIP: no Feishu webhook for target=${target} (branch=${branch})."
  exit 0
fi

payload="$(jq -nc --arg text "${message}" '{msg_type:"text", content:{text:$text}}')"

curl --fail --show-error --silent \
  -X POST \
  -H "Content-Type: application/json" \
  -d "${payload}" \
  "${webhook}"
