# Tasks — 数据产线 MVP

## Context files
See `context-files.md`

## Implementation

### 注意：产线 MVP 与 AlgoRun P1 可并行开发，共享 migration 044（Phase 0 已完成）

### Migration 046
- [ ] [backend] 新增 `production_line_templates` 表（id, name, description, version, status, yaml_definition, created_at, updated_at）
- [ ] [backend] 新增 `production_line_runs` 表（id, template_id, template_version, input_assets, status, triggered_by, started_at, finished_at）
- [ ] [backend] 新增 `production_line_stage_runs` 表（id, run_id, stage_id, stage_type, status, started_at, finished_at, output_assets）
- [ ] [backend] 新增 `quality_check_results` 表（id, stage_run_id, check_type, status, message, details）

### 模板管理

#### Repository 层
- [ ] [backend] `ProductionLineTemplateRepository` — CRUD + 发布/版本管理
- [ ] [backend] `ProductionLineRunRepository` — 运行 CRUD + 状态更新
- [ ] [backend] `ProductionLineStageRunRepository` — 阶段运行 + 状态同步

#### 模板校验器
- [ ] [backend] YAML schema 解析器 — 解析模板 YAML 为结构化 Stage 列表
- [ ] [backend] DAG 检测 — 无环、依赖存在、引用合法
- [ ] [backend] 阶段类型校验 — pipeline / quality_gate / external 类型合法

#### Handler + Usecase
- [ ] [backend] POST /api/v1/production-lines/templates — 创建模板
- [ ] [backend] GET /api/v1/production-lines/templates — 列表
- [ ] [backend] GET /api/v1/production-lines/templates/{id} — 详情
- [ ] [backend] PUT /api/v1/production-lines/templates/{id} — 更新
- [ ] [backend] POST /api/v1/production-lines/templates/{id}:publish — 发布
- [ ] [backend] POST /api/v1/production-lines/runs — 启动运行
- [ ] [backend] GET /api/v1/production-lines/runs — 运行列表
- [ ] [backend] GET /api/v1/production-lines/runs/{id} — 运行详情（含阶段状态）

### Argo Multi-Stage Renderer
- [ ] [backend] 解析产线模板 → 渲染 Argo multi-step DAG Workflow
- [ ] [backend] 注入阶段间资产 manifest 传递（前一阶段输出作为后一阶段输入参数）
- [ ] [backend] 复用现有 transpiler 能力 + pipeline template 引用
- [ ] [backend] quality_gate 阶段支持：执行质量检查脚本并采集结果

### 阶段输出注册
- [ ] [backend] POST /api/v1/production-lines/runs/{id}/stages/{stageId}/outputs
- [ ] [backend] 内部复用 pipeline-assets 注册 + asset_relations 血缘写入

### Quality Gate 基础
- [ ] [backend] schema 检查 — 验证阶段输出字段符合定义
- [ ] [backend] metric_threshold 检查 — 数值指标阈值
- [ ] [backend] file_existence 检查 — GCS 路径存在性

### Argo 状态同步
- [ ] [backend] 定期同步 Argo Workflow 状态 → production_line_stage_runs
- [ ] [backend] 按需查询 API

## API contract sync
- [ ] `api/openapi.yaml` — 所有新端点 + 请求/响应 schema
- [ ] `docs/review/api-guide.md` — curl 示例
- [ ] `scripts/api-guide-smoke.sh` — 烟雾测试

## Frontend UI
- [ ] [Frontend] 产线模板列表页
- [ ] [Frontend] 产线模板详情页 + 编辑
- [ ] [Frontend] 运行表单（选择模板 + 输入资产）
- [ ] [Frontend] 运行详情页 + 阶段 DAG 状态图
- [ ] [Frontend] 质量检查结果展示

## Local verification
- [ ] `make fmt && make vet` in backend/
- [ ] `go test ./...` 全部通过
- [ ] 前端 tsc 无错误

## Deploy verification
- [ ] 构建镜像 + push
- [ ] 部署到 Cloud Run dev
- [ ] 烟雾测试通过
- [ ] 样板产线端到端验证
