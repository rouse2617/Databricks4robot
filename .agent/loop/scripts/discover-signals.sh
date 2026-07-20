#!/bin/bash
# discover-signals.sh — Ralph DISCOVER 阶段的实时信号采集器
# 输出:5 类真实事件源,每类最多 5 行,给 subagent 判断有无待处理事项
set -u
BACKEND="${BACKEND:-https://cyber-databrew-backend-dev-234851712830.us-central1.run.app}"
TOKEN="${DATABREW_TOKEN:-dev-token}"
REPO="${REPO:-CyberOrigin2077/cyber-databrew}"

echo "=== 1. Open PRs requesting me / needing review ==="
gh pr list --repo "$REPO" --state open --search "review-requested:@me -author:@me" --json number,title,updatedAt,isDraft,author --jq '.[:5][] | select(.author.login!="rouse2617") | "PR #\(.number) [\(.updatedAt)] draft=\(.isDraft) \(.title)"' 2>&1 | head -5

echo ""
echo "=== 2. Recent failed CI runs (last 24h) ==="
gh run list --repo "$REPO" --status failure --limit 5 --json databaseId,workflowName,createdAt,headBranch --jq '.[] | "run \(.databaseId) [\(.createdAt)] \(.workflowName) on \(.headBranch)"' 2>&1 | head -5

echo ""
echo "=== 3. My open PRs stalled (>1h since update) ==="
gh pr list --repo "$REPO" --author "@me" --state open --json number,title,updatedAt,statusCheckRollup --jq '.[] | select((.updatedAt | fromdateiso8601) < (now - 3600)) | "PR #\(.number) stalled since \(.updatedAt) \(.title)"' 2>&1 | head -5

echo ""
echo "=== 4. Cloud Run backend errors (last 30m,仅 ERROR 严重级) ==="
gcloud logging read "resource.type=cloud_run_revision resource.labels.service_name=cyber-databrew-backend-dev severity=ERROR" --limit=5 --freshness=30m --format='value(timestamp,jsonPayload.msg,jsonPayload.err)' 2>&1 | head -10

echo ""
echo "=== 5. Recent stranded/loop-like log patterns ==="
gcloud logging read 'resource.type=cloud_run_revision resource.labels.service_name=cyber-databrew-backend-dev ("stranded" OR "already-exists but run still has no uid" OR "retry next cycle")' --limit=3 --freshness=15m --format='value(timestamp,jsonPayload.msg,jsonPayload.jobID)' 2>&1 | head -6

echo ""
echo "=== signals done ==="
