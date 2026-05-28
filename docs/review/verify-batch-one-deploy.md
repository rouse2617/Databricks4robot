# 批次一部署验证报告

## 1. 概述

- **批次范围**：资产模型扩展 Phase 1 + 统一检索 P1
- **分支**：`feat/pipeline-integration`
- **最新 commit**：`1b4ed42` (`fix(backend): handle nullable mcap_file_id in scan functions`)
- **日期**：2026-05-28

## 2. 部署信息

- **服务**：`cyber-databrew-backend-dev` (Cloud Run)
- **活跃 revision**：`cyber-databrew-backend-dev-fix-all-scan` (100% 流量)
- **端点**：`https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app`

## 3. 实现内容

| 模块 | 状态 |
|------|------|
| `dataset` / `annotation_result` 类型 schema 注册表 | ✅ |
| `GET /api/v1/asset-types/{type}/schema` 端点 | ✅ |
| ES typed projections (dataset.*, annotation_result.*) | ✅ |
| asset_relations CHECK 约束增加 annotated_from, materialized_from | ✅ |
| 统一检索 lineage filter 扩展 | ✅ |
| OpenAPI 同步 | ✅ |

## 4. 发现并修复的 Bug

### 4.1 ES errgroup 传播导致 PG 查询被取消

**现象**：`POST /api/v1/queries/run` 返回 HTTP 500，错误 `postgres AssetRepo.ListWithFilters count: context canceled`

**根因**：`executeCompiledRun` 使用 `errgroup.WithContext(ctx)` 并行执行 PG 和 ES goroutine。ES 不可用时（dev 环境未配置 ES），ES goroutine 返回 error → errgroup 取消共享 context → PG 的 `COUNT(*)` 被 PostgreSQL 立即取消。

**修复**（`backend/internal/handlers/query/handler.go`）：ES goroutine 改为 `return nil` 而非 `return err`，错误信息存储在 `esErr` 变量中，后续 fallback 代码已正确处理此情况。

**文件**：`backend/internal/handlers/query/handler.go:301`

### 4.2 NULL mcap_file_id scan 失败

**现象**：dataset 类型资产（无 mcap_file_id）在多个查询接口返回 HTTP 500，错误 `cannot scan NULL into *string`

**根因**：PostgreSQL 查询扫描 `mcap_file_id` 列时直接 scan 进 `models.Asset.McapFileID(string)` 类型字段，当列为 NULL 时报错。旧数据都有 mcap_file_id，新增的 dataset 类型资产没有此字段。

**修复**：3 个扫描函数改为先 scan 进 `*string` 局部变量，非 NULL 再赋值给 struct 字段：
- `scanOneAsset` — 影响 `GET /assets/:id`、`GET /assets/:id/provenance`、`GET /assets/:id/events/stream`
- `ListWithFilters` — 影响 `POST /api/v1/queries/run`
- `ListByLogicalAssetID` — 影响资产血缘查询

**文件**：`backend/internal/postgres/repos.go`

## 5. 烟雾测试结果

| 指标 | 修复前 | 修复后 |
|------|--------|--------|
| 通过数 | 24 | **34** |
| 失败数 | 5（1真+4skip） | **1**（仅 /healthz 404） |

**详细对比**：

| 测试项 | 修复前 | 修复后 |
|--------|--------|--------|
| asset type schema dataset | ❌ 404 | ✅ 200 |
| queries/run | ❌ 500 | ✅ 200 |
| asset by id | ❌ 500 | ✅ 200 |
| asset provenance | ❌ 500 | ✅ 200 |
| asset events stream SSE | ❌ 500 | ✅ 200 |
| GET /healthz | ❌ 404 | ❌ 404（与批次无关，Cloud Run 无此端点） |

## 6. 前端 E2E 测试结果

由 Claude Code 在 2026-05-28 通过 Chrome 浏览器自动测试。

| 测试项 | 结果 | 备注 |
|--------|------|------|
| 页面加载 | ✅ | SPA 正常，侧边栏导航完整 |
| 全文搜索 | ✅ | 搜索 `dataset` 返回 350K+ 结果 |
| Dataset 资产搜索 | ✅ | DS 前缀资产可见 |
| 资产详情查看 | ✅ | 弹窗展示完整字段 |
| 快速筛选 | ✅ | Ready 按钮过滤至 297K 条 |
| 高级筛选 | ✅ | 多字段叠加，标签可清除 |
| Semantic 搜索 | ⚠️ 降级 | ES 索引不可用，降级为 DB 查询 |
| 搜索模式切换 | ✅ | 4 种模式可选 |

截图保存在桌面：`e2e-asset-list.png`、`e2e-search-modes.png`、`e2e-semantic-search-degraded.png`

## 7. 待完成

- ~~批次二（AlgoRun MVP + 数据产线 MVP）~~ — 等待启动指令

## 8. 结论

批次一（资产模型扩展 Phase 1 + 统一检索 P1）**全部完成并通过验证**：

- ✅ 代码实现已合入 `feat/pipeline-integration`
- ✅ 构建/部署到 Cloud Run dev 成功
- ✅ API 端点全部验证通过
- ✅ 烟雾测试 34/35 通过（唯一失败与环境相关）
- ✅ 修复了 2 个生产级 bug（errgroup 传播 + NULL scan）
- ✅ 前端 E2E 测试通过（Semantic 降级属已知环境限制）
