# 批次二实施路线

> 版本：2026-05-28  
> 状态：评估稿  
> 范围：批次二三个子项的范围概览、依赖关系、实施顺序、工作量估算、风险点、worktree 拆分方案

---

## 1. 批次二范围概览

### 1.1 AlgoRun MVP Phase 1 — 训练产出自动资产化 + 输入血缘

**目标：** 让外部训练系统通过 API 接入资产闭环，训练产出自动注册为资产版本，输入到输出自动建血缘边。

**设计文档：** `docs/review/algorn-enhancement.md` §9 Phase 1

**交付项：**

| 编号 | 项 | 说明 |
|------|----|------|
| A1 | 新增表 `algo_run_inputs` | 结构化输入资产角色（训练集、评估集、配置等） |
| A2 | 新增表 `algo_run_outputs` | run-output 映射表，幂等/列表/对比基础 |
| A3 | 扩展 `asset_relations.relation_type` | 增加 `trained_from`、`evaluated_on`、`validated_on`、`configured_by`、`fine_tuned_from`、`features_from` |
| A4 | `asset_relations` 加 `metadata JSONB` | 承载训练参数、指标、role |
| A5 | `POST /api/v1/algo-runs/{run_id}/outputs:register` | 注册训练产出为 `ml_model` / `dataset` 资产 revision |
| A6 | `GET /api/v1/algo-runs/{run_id}/outputs` | 查询 run 产物 |
| A7 | 资产版本复用 | 复用现有 `logical_assets` + `assets` 版本逻辑 |
| A8 | 自动写输入到输出的血缘边 | 注册成功时写 `asset_relations` |

**暂缓（Phase 2/3）：** Pipeline 训练节点 UI、run 对比、ES 投影 run、模型发布治理

### 1.2 资产模型 Phase 2 — ML Model + EvaluationReport

**目标：** 在 Phase 1（dataset/annotation_result）基础上扩展 `ml_model` 和 `evaluation_report` 类型及相关关系。

**设计文档：** `docs/review/asset-model-expansion.md` §6 Phase 2

**交付项：**

| 编号 | 项 | 说明 |
|------|----|------|
| B1 | `asset_type_schemas` registry | 类型 schema 注册表或等价代码 registry |
| B2 | `ml_model` 类型支持 | framework、architecture、metrics、quantization、artifact_uri |
| B3 | `evaluation_report` 类型支持 | model/dataset 引用、metrics、evaluated_at、tool、report_uri |
| B4 | relation_type 扩展 | `tested_on`、`evaluates`、`compares_to`、`calibrated_from`、`generated_by` |
| B5 | ES typed projection | `ml_model.*` / `evaluation_report.*` 命名空间 |
| B6 | API 别名端点（可选） | `POST /api/v1/ml-models`、`POST /api/v1/datasets` 薄封装 |

**注意：** B4 的 relation_type 扩展与 A3 重复，合并为一次 migration。

### 1.3 数据产线 MVP — 多阶段产线编排

**目标：** 从"单段 workflow"升级为"多阶段产线"，跑通采集→转换→清洗→注册闭环。

**设计文档：** `docs/review/data-production-line.md` §12 MVP

**交付项：**

| 编号 | 项 | 说明 |
|------|----|------|
| C1 | 新增表 `production_line_templates`、`runs`、`stage_runs`、`quality_check_results` | 先用 JSONB 快照 |
| C2 | ProductionLineTemplate YAML schema + 校验器 | DAG 无环、依赖存在、引用合法 |
| C3 | Backend：模板 CRUD、发布、版本 | draft/published 流程 |
| C4 | Backend：产线启动（解析输入→创建 run→渲染 Argo DAG→提交 workflow） | 复用现有 transpiler |
| C5 | Backend：阶段输出注册 API | 内部复用 pipeline-assets 注册 + `asset_relations` |
| C6 | Backend：基础 quality gate | schema、metric_threshold、file_existence |
| C7 | Backend：Argo 状态同步 | phase → stage_run status 映射 |
| C8 | Frontend：产线模板列表/详情/运行表单/运行详情 | 阶段 DAG 状态图 + 质量结果展示 |
| C9 | 样板产线 | 路采数据入库链路 |

---

## 2. 子项依赖关系

```text
Asset Model Phase 2 (B)
       │
       ├── provides: asset_type_schemas, ml_model/evaluation_report types, ES projection
       │
       ▼
AlgoRun Phase 1 (A)
       │
       ├── depends on: ml_model/dataset asset types (B2), expanded relation_types (B4+A3)
       ├── shared migration with B: relation_type CHECK + asset_relations.metadata
       │
       ▼
Data Production Line MVP (C)
       │
       ├── depends on: asset versioning infrastructure (existing), relation_type expansion (A3/B4)
       ├── depends on: pipeline-output registration (existing, pipeline-assets)
       └── can use: algo_run_outputs registration pattern (A5) as reference
```

**关键依赖结论：**

- **B(Asset Model P2) 是 A 的前置条件：** AlgoRun 产物注册需要 `ml_model` / `dataset` 作为目标 asset_type。
- **A(AlgoRun P1) 和 C(产线) 可部分并行：** 共享 `asset_relations` 扩展，但产线自身的完整 CRUD + Argo renderer + Frontend 与 AlgoRun 的产物注册无强依赖。
- **relation_type 扩展是共享 block：** 必须最先完成，否则 A 和 C 都无法写血缘边。

---

## 3. 建议实施顺序

### Phase 0：基础层（Week 1-2）

优先完成 Asset Model Phase 2 的核心，为 AlgoRun 打好基础。

| 步骤 | 工作项 | 输出 |
|------|--------|------|
| 0.1 | migration：扩展 `asset_relations.relation_type`（一次性加入所有 AI 关系：`trained_from`、`evaluated_on`、`validated_on`、`configured_by`、`fine_tuned_from`、`features_from`、`tested_on`、`evaluates`、`compares_to`、`calibrated_from`、`generated_by`） | 迁移文件 044 |
| 0.2 | migration：`asset_relations` 加 `metadata JSONB` + 索引 | 迁移文件 044（同一文件） |
| 0.3 | migration：扩展 `assets.chk_mcap_file_required` 允许 `ml_model` / `evaluation_report` | 迁移文件 044（同一文件） |
| 0.4 | `asset_type_schemas` registry（代码 registry 或表）注册 `dataset`（已有）、`annotation_result`（已有）、`ml_model`、`evaluation_report` | registry |
| 0.5 | ES mapping 增加 `ml_model.*`、`evaluation_report.*` projection | ES builder |

### Phase 1：AlgoRun 产出注册（Week 2-4）

| 步骤 | 工作项 | 输出 |
|------|--------|------|
| 1.1 | migration：`algo_run_inputs`、`algo_run_outputs` 表 | 迁移文件 045 |
| 1.2 | `algorun_artifact` handler/usecase/repository | `POST /outputs:register`、`GET /outputs` |
| 1.3 | 资产版本复用逻辑（`ml_model`/`dataset` revision 创建） | asset usecase 扩展 |
| 1.4 | 注册时写 `asset_relations`（`trained_from`、`evaluated_on`、`configured_by`） | lineage 事务 |
| 1.5 | `GET /algo-runs/{run_id}/outputs` 实现 | 列表查询 |
| 1.6 | API contract sync：OpenAPI、api-guide | 文档 |
| 1.7 | 验收测试：创建→启动→完成 run → 注册产出 → 查 lineage | 自动化测试 |

### Phase 2：生产产线 MVP（Week 3-7）

| 步骤 | 工作项 | 输出 |
|------|--------|------|
| 2.1 | migration：`production_line_templates`、`runs`、`stage_runs`、`quality_check_results` | 迁移文件 046 |
| 2.2 | 模板校验器（YAML schema + DAG 检测） | validator 模块 |
| 2.3 | 模板 CRUD handler/usecase/repository | `POST/GET/LIST /production-lines/templates` |
| 2.4 | 产线启动渲染器（复用 transpiler 生成 Argo DAG） | renderer 模块 |
| 2.5 | 阶段输出注册 API | `POST /production-lines/runs/{id}/stages/{stageId}/outputs` |
| 2.6 | 基础 quality gate 实现 | schema / metric_threshold / file_existence |
| 2.7 | Argo 状态同步器 | 定期同步 + 按需查询 |
| 2.8 | Frontend：模板列表/详情/运行入口 | UI |
| 2.9 | Frontend：运行详情 + 阶段 DAG 图 + 质量展示 | UI |
| 2.10 | 样板产线（路采数据入库） | 端到端验证 |

### Phase 3：AlgoRun 增强 + Pipeline 训练节点（Week 7-9）

| 步骤 | 工作项 | 输出 |
|------|--------|------|
| 3.1 | AlgoRun rerun `POST /algo-runs/{run_id}:rerun` | run 重跑 |
| 3.2 | AlgoRun compare `GET /algo-runs:compare` | 双 run 对比 |
| 3.3 | 列表扩展 `logical_asset_id`、`parent_run_id`、`pipeline_name` 过滤 | 查询增强 |
| 3.4 | Pipeline TrainingStep transpiler 扩展 | 训练节点转译 |
| 3.5 | Pipeline TrainingStep 前端组件 | 设计器训练节点 |

---

## 4. 工作量估算

> 人日估算基于文档中的预估，按实际代码量调整。

| 子项 | 模块 | 人日 | 说明 |
|------|------|-----:|------|
| **基础层** | | | |
| | migration 044（relation_type + metadata + asset_type CHECK） | 1 | 单文件，但需确保现有数据兼容 |
| | `asset_type_schemas` registry | 2 | 代码 registry + JSON Schema |
| | ES typed projection | 2 | mapping + builder |
| | 小计 | **5** | |
| **AlgoRun P1** | | | |
| | migration 045（algo_run_inputs + outputs） | 1 | 两张表 |
| | `algorun_artifact` handler/usecase/repo | 5 | 注册 + 列表 + 幂等 |
| | 资产版本复用 | 3 | ml_model/dataset revision 创建 |
| | 血缘写入 | 2 | 事务内建 `asset_relations` |
| | API 文档 + 测试 | 2 | |
| | 小计 | **13** | |
| **资产模型 P2** | | | |
| | ml_model/evaluation_report 校验逻辑 | 2 | 类型专属校验 |
| | 关系约束（trained_from/evaluates 等配对） | 1 | |
| | API 别名端点（可选） | 2 | 薄封装 |
| | 小计 | **5**（基本）~ **7**（含别名） | |
| **数据产线 MVP** | | | |
| | migration 046（4 张表） | 2 | production_line 系列表 |
| | 模板 CRUD + 校验器 | 5 | 含 YAML schema + DAG |
| | Argo multi-stage renderer | 5 | 复用现有 transpiler |
| | 阶段输出注册 | 4 | 复用 pipeline-assets |
| | Quality gate 基础 | 4 | 3 种规则 |
| | 状态同步 | 2 | Argo → stage_run |
| | Frontend 模板/运行 | 6 | 列表+详情+运行表单 |
| | Frontend DAG 图 + 质量 | 4 | |
| | QA + 样板产线 | 3 | |
| | 小计 | **35** | |
| **AlgoRun P2（暂缓）** | | | |
| | rerun / compare / 列表增强 | 5 | 暂缓 |
| | Pipeline 训练节点 | 6 | 暂缓 |
| | 小计 | **11** | |

**批次二核心（基础层 + AlgoRun P1 + 资产模型 P2 + 产线 MVP）：约 58 人日**

若分 2 人并行开发（一人 AlgoRun + 资产模型，一人产线），约 **5-6 周**。

若单人力串行，约 **8-10 周**。

---

## 5. 风险点

| 风险 | 影响 | 缓解 |
|------|------|------|
| **ES 不可用对搜索的影响（已暴露）** | 产线阶段输出资产后，搜索不可见，用户感知延迟 | 详情页回源 PG；ES 作为最终一致查询层；增加 sync-status 可见性 |
| **relation_type migration 破坏现有数据** | 新增 CHECK 约束可能因现有非法数据失败 | 单独 migration，先 dev 验证，分两步：先加新类型，再更新约束 |
| **AlgoRun 注册事务过大** | 单个事务包含 INSERT assets + UPDATE logical_assets + INSERT asset_relations + INSERT algo_run_outputs + INSERT asset_events，长事务风险 | 确保 PG 事务超时配置合理；考虑拆分事件为异步 |
| **产线 Argo renderer 复杂度** | 从 Pipeline single-DAG 升级到 multi-stage DAG，模板语法和参数传递容易出错 | MVP 限制阶段类型（先支持 pipeline + quality_gate）；先用嵌入式而非提交式 |
| **产线前端改动大** | 6+4 人日的前端工作量，涉及画布、DAG、质量展示 | 后端先出 API + 可测试，前端可分两次迭代 |
| **ml_model 类型 schema 不稳定** | ML 框架、指标格式快速变化，硬编码校验会频繁改动 | 使用 JSON Schema registry + soft validation（warning 而非 error） |
| **并发 revision 分配冲突** | 多个 run 同时注册同一 logical_asset 产生 version 冲突 | `SELECT FOR UPDATE` 串行化；冲突返回 409 重试 |
| **重试导致重复资产版本** | 失败重跑的 stage 可能产生冗余 revision | 默认重试覆盖未发布输出；幂等键 `(run_id, artifact_name)` |

---

## 6. 建议 Worktree 拆分方案

### Worktree A：Asset Model Phase 2（基础层）

```text
分支：feat/asset-model-p2
范围：Phase 0 全部
时间：~1 周
输出：
  - migration 044
  - asset_type_schemas registry
  - ml_model/evaluation_report 校验
  - ES typed projection
依赖：无（可最先开始）
```

### Worktree B：AlgoRun MVP Phase 1

```text
分支：feat/algorun-p1
范围：Phase 1 全部 + Phase 3 的 rerun/compare
时间：~2-3 周
前置：Worktree A（至少 migration 044 完成）
共享：迁移文件编号需协调（045）
依赖：
  - relation_type 扩展（来自 A）
  - ml_model 类型校验（来自 A）
输出：
  - migration 045
  - algo_run_inputs/outputs 表
  - outputs:register / outputs:list API
  - 资产版本复用（ml_model/dataset）
  - 血缘自动写入
  - rerun / compare（暂缓，可后加）
```

### Worktree C：数据产线 MVP

```text
分支：feat/production-line-mvp
范围：Phase 2 全部
时间：~4 周
前置：Worktree A 的 migration 044 + AssetTypeRegistry
并行：可与 Worktree B 并行（共享 relation_type 扩展，但业务逻辑独立）
输出：
  - migration 046
  - production_line 系列表
  - 模板 CRUD + 校验
  - Argo multi-stage renderer
  - 阶段输出注册
  - Quality gate 基础
  - Frontend 模板/运行/DAG
  - 样板产线
```

### 执行顺序图

```text
Week 1    Week 2    Week 3    Week 4    Week 5    Week 6    Week 7
┌────────┐
│  A     │
│ (基础层)│
└───┬────┘
    │     ┌───────────────┐
    └────▶│  B (AlgoRun)  │
          │               │          ┌──────────────────────────────┐
          │               │          │                              │
          └───────────────┘          │  D (AlgoRun P2 — 暂缓)       │
                                     │  rerun/compare/training node │
          ┌──────────────────────────┤                              │
          │  C (产线 MVP)           │                              │
          │                          └──────────────────────────────┘
          │
          └──────────────────────────────────────────┘
```

---

## 7. 总结建议

1. **先从基础层开始：** 扩展 `asset_relations.relation_type` 和 `asset_type_schemas` registry 是批次二所有子项的共同前提。建议 Week 1 专注完成迁移文件 044 和 registry。

2. **AlgoRun P1 和产线 MVP 可并行：** 两者业务逻辑隔离（AlgoRun 聚焦 run-product 映射，产线聚焦 multi-stage 编排），共享 relation_type 扩展即可。建议两人分别跟 B 和 C。

3. **产线 MVP 前端是瓶颈：** 35 人日中 10 人日在前端（6+4）。如果前端资源有限，建议后端先全部出 API + 用命令行/smoke 验证，前端分两次迭代。

4. **暂缓项明确放开：** AlgoRun rerun/compare、Pipeline 训练节点、模型发布治理、ES 投影 run 放在 Phase 2 处理。不影响批次二的 MVP 验收标准。

5. **验收标准：**
   - AlgoRun P1：训练完成后调用 `outputs:register`，PG 中可查到新资产 revision + 血缘边；`GET /outputs` 返回产物列表
   - 资产模型 P2：可创建 `ml_model` / `evaluation_report` 类型的资产，ES 可搜索到类型专属字段
   - 产线 MVP：可保存发布产线模板 → 选择输入资产启动 → Argo multi-stage DAG → 阶段输出注册 → lineage 可查
