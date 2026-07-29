# Proposal — CYB-4450: 替换算法处理 tab 为运行记录 tab

## Why

资产详情页的「算法处理」tab 展示的是 CF 时代的 `algo_results` legacy 数据（`parseAlgoResults`），"算法"概念已不再维护——业务已切换到流水线模板。用户需要一个从「资产 → 哪些流水线处理过」的视角，替代现有的算法视图。

## What Changes

### Modified Capabilities
- **asset-detail-tabs**:「算法处理」tab 整体移除，替换为「运行记录」tab

### New Capabilities
- **asset-pipeline-history**: `GET /api/v1/assets/:id/pipeline-history` — 返回该资产被流水线处理过的 run 记录，按模板分组、按时间倒序

## Impact
- **Affected code**:
  - `Frontend/src/pages/AssetDetailPage.tsx` — 删除「算法处理」tab + `parseAlgoResults` + `algoEvents` 状态 + `loadAlgoEvents`；既有 `RunsTab` 改名为 `运行记录`
  - `Frontend/src/components/asset-detail/AlgoTab.tsx` — 整文件删除（不再引用）
  - `api/openapi.yaml` + `docs/review/api-guide.md` — 无需变更

- **New APIs**: 无（沿用既有的 `GET /api/v1/assets/:id/runs`，CYB-4297）
- **Dependencies**: 无

## Scope
- **In scope**:
  - 删除「算法处理」tab + 所有关联前端代码
  - 新增后端 endpoint + 前端 tab「运行记录」
  - 返回字段：pipelineName, runId, status, createdAt, startedAt, finishedAt
  - API 合约同步

- **Out of scope**:
  - 镜像信息（image/image_tag/image_digest）— `pipeline_runs` 没有这些列，需单独跟踪数据源
  - Step 粒度展开 — 第一版只做 run 级
  - 删除后端 `startAlgo` / `listAlgoEvents` handler（保留接口不删，避免 SDK 断连）

## Success Criteria
- [ ] 「算法处理」tab 不再出现在资产详情页
- [ ] 「运行记录」tab 按模板名分组、每组下按时间倒序列出每次 run
- [ ] run_id 可点击跳转到运行详情
- [ ] 状态以 Tag 渲染，耗时 = `finishedAt - startedAt`
- [ ] 无流水处理记录的资产显示空态
