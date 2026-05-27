# Proposal — 资产模型扩展 Phase 1

## Why
当前 asset_type 有限（raw_mcap/segment/clip 等），不支持 Data+AI 链路中的 Dataset 和 AnnotationResult 类型。需要扩展资产模型以承载 AI 数据资产的统一管理。

## What Changes
- Extend `asset_relations.relation_type` CHECK 约束，加入 AI 资产关系
- 引入 code-based `asset_type_schemas` registry
- 新增 `dataset`、`annotation_result` asset_type 及专属元数据
- 写入校验从硬编码 switch 演进到 registry-driven validation
- ES mapping 增加 `dataset.*`、`annotation_result.*` projection
- 新增 `GET /api/v1/asset-types/{type}/schema` 端点（返回 JSON Schema）

## Scope
- **In scope**: 类型注册表、新类型定义、校验变更、ES mapping、schema 端点
- **Out of scope**: 专属投影表（Phase 3）、前端动态表单、存量数据回填

## Reference Design
`docs/review/asset-model-expansion.md` (§ 6 Phase 1, § 8 落地顺序 1-4)
