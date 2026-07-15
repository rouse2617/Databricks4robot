# CYB-3298 — deploy-dev 后端构建陈旧二进制修复

## Why
实测确认（2026-07-10）：`deploy-dev` 报成功、VM 构建了正确 commit（日志 `code ready: 9fcc053`）、Cloud Run 新建 revision 且 100% 流量、`consumer_lag:0`，但运行的后端**不含已合并的 builder 改动**（CYB-3297-C 的 mcap.* 字段）。PATCH 触发 asset_updated（seq 递增并被 ES 应用）后 doc 仍无 `mcap.device_id` → **运行的是旧二进制**。

根因：VM 上常驻的 warm buildx builder（`deploy-dev.yml:115-118` 复用 `databrew-builder`，warm 时只 `--cache-to` 不 `--cache-from`）复用了陈旧的构建层，产出的镜像不含新代码。且构建从不传 `COMMIT` build-arg → `/version` 恒为 `commit: unknown`，无法核对运行版本。

## What Changes
仅改 `.github/workflows/deploy-dev.yml` 后端构建步骤：
1. `--no-cache-filter=builder`：强制 `builder` 阶段（COPY 源码 + go build）每次重跑，**保证编译最新源码**（GOCACHE 挂载仍在，增量编译，代价可接受）。
2. `--build-arg COMMIT=$COMMIT_SHA`（+ `VERSION`、`BUILD_TIME`）：写入 `/version`，让每次部署**可核对运行 commit**；同时使 go build 层输入随 commit 变化，双保险。

## Impact
- Affected code: `.github/workflows/deploy-dev.yml`（backend build step 仅）
- 无 schema / 应用代码改动。
- 代价：dev 每次后端部署会重编译（+ 重新 `go mod download`），慢约 1–2 分钟；换取正确性与可验证性。
- ⚠️ 改部署基建（outward-facing）。用户已明确指示「通过 cicd 部署」。prod（deploy-prod.yml）同类风险，本 PR 不含，另立跟进。

## 验证
- 合并本 PR 本身会触发一次 deploy-dev（路径含该 workflow）。
- 部署后：`/version` 显示真实 commit（非 unknown）；对某 segment 触发一次 asset 事件后，其 ES doc 出现 `mcap.device_id`（即 CYB-3297-C 生效）。
