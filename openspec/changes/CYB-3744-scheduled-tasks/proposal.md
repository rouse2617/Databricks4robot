# Proposal — CYB-3744 定时任务

## Why

Grace→流水线的自动下发目前是**独立的 Cloud Run Job(`grace-sync` / `grace-sync-prod`)+ Cloud Scheduler 触发**。问题:

- **只有一个人会操作**(要进 GCP 改 Cloud Scheduler / Cloud Run),别人改不了。
- 每次调整(改频率、改数据源、补跑)都要动外部基础设施,和 DataBrew 割裂。
- 出问题(如批量堆积、GPU 卡死)排查要跨 GCP + DataBrew 两套。

目标:把定时自动下发做成 **DataBrew 里的一等公民、自助功能** —— 谁都能在 UI 配置/启停/补跑,不再养一个只有你会碰的外部服务。

## What Changes

### New Capabilities

1. **「定时任务」tab(流水线页)** — 与 设计/流水线/执行记录/组件 平级。列出所有规则(名称、流水线、数据源、触发模式、频率、状态、上次运行),支持 新建/编辑/启停/立即运行。产出的批量仍进「执行记录 → 批量任务」。
2. **可配置数据源(通用 REST 源)** — 一条规则挂一个 `source_type` + `source_config`。v1 做通用 REST-JSON 源:不同端点 / 不同 API 都只是不同配置,UI 里加,无需写代码。凭证只存 Secret Manager 引用。
3. **触发调度(后端循环)** — 复用现有 `grace.Syncer` 循环 + `StartJobReconciler` ticker 模式:到点按规则从数据源取 asset-id → 复用 backfill 建批量。支持 增量(水位线)/ 滚动窗口 / 固定范围 / 指定资产 四种触发模式,及手动「立即运行」。

### Modified Capabilities

- 后端 `grace` 包扩展:除同步视频时长外,新增"按数据源规则取 id → 建批量"的动作。
- 新增规则配置表 + admin API(镜像 `dispatcher_configs` 模式)。

### Retired

- 外部 `grace-sync` / `grace-sync-prod` Cloud Run Job。
- `grace-sync-prod` / `grace-sync-daily` Cloud Scheduler job。
- (迁移完成、in-app 验证通过后执行,见 design.md 迁移与回滚)

## Impact

- **Affected code**:
  - backend: `internal/grace/*`(数据源抽象 + REST 源)、新增 `scheduled task` usecase + 调度循环、`postgres` 新表 repo、`handlers` admin API、`cmd/server/core.go` 接线
  - frontend: 流水线页新增「定时任务」tab + 规则列表/表单
  - migration: 新增 `scheduled_tasks`(规则)表(+ 游标/last_run 字段)
- **New HTTP API**: 规则的 CRUD + 启停 + 立即运行(需走 [API 契约同步](../../../docs/agents/AI-RULES.md):OpenAPI + api-guide + SDK + smoke)
- **Dependencies**: 无新增外部依赖;复用现有 grace client / backfill / ticker / 配置面模式
- **Secrets**: 数据源凭证只用 Secret Manager 引用,不落明文

## Scope

- **In scope**: 定时任务 tab + 通用 REST 源 + 四种触发模式 + 后端调度循环(单飞) + 规则 CRUD API + 退役外部 grace-sync
- **Out of scope**:
  - **去重**(产品明确不做,和现状一致)
  - 非 REST 平台(GraphQL/gRPC/异形鉴权)—— 留 `AssetSource` 接口的 adapter 逃生口,等真实场景再做
  - 需求 2、3(如有)另议

## Success Criteria

- [ ] UI 能新建一条规则(选流水线+资源池+REST 数据源+触发模式),启停、立即运行
- [ ] 增量模式:到点自动从 Grace `/grace/video_steps` 拉新 id 建批量,水位线推进,不漏不重复扫
- [ ] 产出的批量出现在「执行记录 → 批量任务」
- [ ] 多副本后端下规则不双触发(单飞验证)
- [ ] 外部 grace-sync Cloud Run Job + Cloud Scheduler 退役后,同步仍正常(由 in-app 接管)
- [ ] API 契约同步齐全;`go test ./...` + 前端 build/test 通过
