# Proposal — 数据产线 MVP

## Why
当前 Pipeline 只支持单段 workflow，无法编排采集→转换→清洗→注册的多阶段产线。数据生产链路的阶段间依赖、质量门禁、资产传递都需手动处理。

## What Changes

### New Capabilities
- 产线模板 CRUD：定义多阶段 DAG、阶段类型、质量规则
- 产线运行启动：解析输入资产 → 渲染 Argo multi-stage DAG → 提交 workflow
- 阶段输出自动注册为资产版本 + 血缘追溯
- 基础 quality gate：schema 校验、metric threshold、file existence

### New Tables
| 表 | 职责 |
|----|------|
| `production_line_templates` | 产线模板定义（YAML schema + JSONB 快照） |
| `production_line_runs` | 产线运行记录 |
| `production_line_stage_runs` | 阶段运行状态 |
| `quality_check_results` | 质量检查结果 |

### Modified Capabilities
- Argo transpiler：从单段 DAG 升级到 multi-stage DAG
- 资产注册：复用 pipeline-assets 注册逻辑

## Impact
- **Affected code**: `backend/internal/handlers/`, `backend/internal/usecase/`, `backend/internal/repository/`
- **New APIs**: 模板 CRUD + 运行 API + 阶段输出注册 + 质量结果查询
- **New modules**: production_line handler, usecase, repository, renderer, quality_gate
- **Frontend**: 模板列表/详情/运行表单/运行详情 + DAG 图 + 质量展示

## Scope
- **In scope**: 后端全部 API + 前端模板/运行/详情 UI
- **Out of scope**: 标注平台集成、训练集 manifest、分段重试、产线级 lineage API 聚合

## Success Criteria
- [ ] 可保存并发布产线模板
- [ ] 选择输入资产启动产线，Argo 中出现 multi-step DAG
- [ ] 至少 3 个阶段串联执行
- [ ] 阶段产出自动注册为资产版本
- [ ] asset_relations 可追溯输入→输出
- [ ] 质量门禁失败时产线阻断
