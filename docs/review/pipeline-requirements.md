# Pipeline 智算平台 — 产品需求

> 版本：2026-05-27 v1  
> 状态：初稿  
> 基准：当前 feat/pipeline-integration 分支代码 + 已上线 dev 环境

---

## 1. 产品定位

Pipeline 是 **Cyber Databrew 智算平台** 的核心编排层。用户把处理逻辑写成 Docker 镜像，在前端拖拽组合成 DAG（有向无环图），平台自动转换为 Argo Workflow 并调度到 K8s 执行。

一句话：**写镜像、拖 DAG、跑起来、看结果。**

```
┌─────────────────────────────────────────────────────┐
│                    Pipeline 平台                       │
│                                                      │
│  用户 Docker 镜像 → 组件注册 → 拖拽编排 →            │
│                      Transpiler → Argo Workflow →     │
│                      K8s 执行 → 资产产出               │
│                                                      │
│  上下游：资产管理（输入/输出）+ 事件流（审计/追溯）    │
└─────────────────────────────────────────────────────┘
```

---

## 2. 用户角色

| 角色 | 是谁 | 核心诉求 |
|------|------|---------|
| **算法工程师** | 写处理算法的同学 | 把镜像变成 pipeline 组件，编排处理流程，关注执行结果和质量 |
| **数据工程师** | 搭数据管线的同学 | 构建可复用的处理链路，管理多版本 pipeline template |
| **业务/运营** | 用数据的人 | 按需选择 pipeline 处理指定资产，看结果 |
| **Admin** | 平台维护者 | 管理组件注册表，监控执行状态，处理失败 |

---

## 3. 功能需求

### F1: 组件注册表（Component Registry）

用户把 Docker 镜像注册为可拖拽的组件。

**用户故事**：
> 作为算法工程师，我写好一个处理镜像 `my-processor:v1` 推送到 registry 后，能在前端注册为组件，声明它的输入输出、资源需求、环境变量。之后在 Pipeline Designer 里就能拖出来用了。

**需求**：

| # | 需求 | 优先级 | 备注 |
|---|------|--------|------|
| F1.1 | 注册组件：image + tag + 名称 + 描述 | P0 | ✅ CYB-1264 |
| F1.2 | 声明输入端口（输入源类型：asset / param / file） | P0 | ✅ CYB-1264 |
| F1.3 | 声明输出端口（输出的产物类型） | P0 | ✅ CYB-1264 |
| F1.4 | 声明资源需求（CPU / 内存 / GPU / 临时存储） | P0 | ✅ CYB-1264 |
| F1.5 | 声明环境变量（可透传 pipeline 级别的参数） | P1 | ✅ CYB-1264 |
| F1.6 | 组件版本管理（迭代不影响已有 pipeline） | P1 | 待加 |
| F1.7 | 组件搜索/筛选 | P1 | 待加 |
| F1.8 | 组件来源标注（system / custom / marketplace） | P2 | 默认 custom，system 待加 |

**API 设计参考**：

```
POST   /api/v1/components              注册组件
GET    /api/v1/components              列出组件
GET    /api/v1/components/:id          组件详情
PUT    /api/v1/components/:id          更新组件
DELETE /api/v1/components/:id          删除组件
POST   /api/v1/components/:id/versions 发布新版本
```

---

### F2: Pipeline Designer（拖拽编排）

可视化 DAG 编辑器，用户拖拽组件、连线、配参数。

**用户故事**：
> 作为数据工程师，我想在画布上拖拽组件、连线定义数据处理流程，不用写 YAML。每个节点我可以配置输入参数，保存为 template 后续复用或部署运行。

**需求**：

| # | 需求 | 优先级 | 备注 |
|---|------|--------|------|
| F2.1 | 组件面板展示可用组件列表 | P0 | ✅ 已实现 |
| F2.2 | 拖拽组件到画布（React Flow） | P0 | ✅ 已实现 |
| F2.3 | 组件间连线定义数据依赖 | P0 | ✅ 已实现 |
| F2.4 | 节点参数配置面板（点击节点编辑） | P0 | ✅ 已实现（基础版） |
| F2.5 | 保存 pipeline 为 template + 加载已有 template | P0 | ✅ 已实现 |
| F2.6 | 清空画布（含确认弹窗） | P0 | ✅ 已实现 |
| F2.7 | 画布缩放/适配/网格 | P0 | ✅ React Flow Controls + Background |
| F2.8 | 删除节点/边 | P0 | ✅ 已实现 |
| F2.9 | 资产选择绑定（运行前选择要处理的 asset） | P1 | 当前缺，见 F4 |
| F2.10 | 节点参数模板默认值 | P1 | 减少每次运行的配置量 |
| F2.11 | Pipeline 自动布局（DAG 层次化排列） | P2 | 避免手动调整位置 |
| F2.12 | Pipeline 版本对比（diff） | P2 | 知道改了什么 |
| F2.13 | 组件组/子图（嵌套 DAG） | P2 | 大型 pipeline 结构化 |
| F2.14 | 拖拽上传文件作为输入 | P3 | CV 场景常见 |

---

### F3: 执行引擎（Transpiler → Argo → K8s）

把前端 DAG 转换为 Argo Workflow YAML，提交到 K8s 执行。

**用户故事**：
> 作为算法工程师，我编排好 pipeline 后点"运行"，平台帮我生成 Argo Workflow 并调度到 K8s 集群，每个 step 就是我写的镜像。我不需要懂 K8s 或 Argo。

**需求**：

| # | 需求 | 优先级 | 备注 |
|---|------|--------|------|
| F3.1 | Pipeline → Argo Workflow YAML 转换 | P0 | ✅ Transpiler 已实现 |
| F3.2 | 参数传递（节点间 input/output 映射） | P0 | ✅ 已实现 |
| F3.3 | 资源限制（CPU/Mem/GPU 写入 YAML） | P0 | ✅ 已实现 |
| F3.4 | 提交 Workflow 到 K8s | P0 | ✅ 已实现 |
| F3.5 | 保存 deployment 记录到 PG | P0 | ✅ pipeline_deployments 表 |
| F3.6 | 运行结果上报（成功/失败/节点状态） | P0 | ✅ 已实现（同步 Argo 状态） |
| F3.7 | 重试策略配置（次数/backoff） | P1 | ✅ CYB-1262 |
| F3.8 | 超时配置（每个 step / 整个 workflow） | P1 | ✅ CYB-1262 |
| F3.9 | 并行度控制（DAG 中并行执行的最大节点数） | P1 | 防止打爆集群 |
| F3.10 | 从 template 部署（无需重新拖拽） | P0 | ✅ 已实现 |
| F3.11 | 跨命名空间/集群执行 | P3 | 多租户场景 |

**Transpiler 当前能力**（`backend/internal/transpiler/`）：

| 能力 | 状态 |
|------|------|
| DAG template 生成 | ✅ |
| input/output 参数映射 | ✅ |
| 资源限制 | ✅ |
| 环境变量 | ✅ |
| 依赖解析 | ✅ |
| retryStrategy | ✅ CYB-1262 |
| activeDeadlineSeconds | ✅ CYB-1262 |
| parallelism 限制 | ⚪ 待加 |
| volume 挂载 | ⚪ 待加 |

---

### F4: Asset 集成（输入选择 → 产出注册 → 血缘追溯）

Pipeline 和资产管理的核心结合点。

**用户故事**：
> 作为业务运营，我想选中一批 asset 扔进 pipeline 处理，处理完的产出自动注册为新 asset并记录血缘（这批产出是从哪些输入来的）。
> 作为算法工程师，我想 pipeline 跑完后能追溯：这个产出 asset 来自哪个 pipeline 的哪个 step、用了什么版本的镜像。

**需求**：

| # | 需求 | 优先级 | 备注 |
|---|------|--------|------|
| F4.1 | pipeline 运行前选择输入 asset(s) | P1 | ✅ CYB-1263 |
| F4.2 | 运行参数透传（asset path、metadata 传给容器） | P1 | env / parameter 注入 |
| F4.3 | 处理结果自动注册新 asset（容器回调 API） | P1 | 容器 finish 后 POST /assets |
| F4.4 | pipeline 执行记录写入 asset_events | P2 | 事件流可追溯 |
| F4.5 | asset ↔ pipeline run 血缘查询 | P2 | 从 asset 反查来自哪次 run |
| F4.6 | 批量 asset → pipeline（一次 run 处理 N 个 asset） | P2 | 类似 backfill |
| F4.7 | pipeline 产出自动关联输入 asset 血缘 | P2 | asset_relations 写入 |

**接口设计参考**：

```
# 运行 pipeline 时选 asset
POST /api/v1/pipelines/:id/deploy
Body: {
  "asset_ids": ["V4ftKjqX", "kK9qIAqG"],
  "params": { "threshold": 0.5 }
}

# pipeline 产出查询
GET /api/v1/assets/:id/pipeline-lineage
→ 返回: 哪次 run、哪个 step、什么镜像产生的

# 按 pipeline 查产出
GET /api/v1/deployments/:id/outputs
→ 返回: 这次 run 产出了哪些 asset
```

---

### F5: 运行监控（Workflow 可视化）

**用户故事**：
> 作为算法工程师，pipeline 跑起来后我想看到实时进度——哪些 step 跑完了、哪些还在跑、失败了怎么看日志。不需要切到 argo-ui。

**需求**：

| # | 需求 | 优先级 | 备注 |
|---|------|--------|------|
| F5.1 | Workflow 列表页（所有 run + 状态筛选） | P0 | ✅ WorkflowListPage |
| F5.2 | Workflow DAG 可视化（节点颜色按状态区分） | P0 | ✅ WorkflowDetailPage (React Flow) |
| F5.3 | 节点详情（状态/时间/消息） | P0 | ✅ 右侧面板 |
| F5.4 | Step 日志查看 | P1 | argo-ui 已有；考虑内嵌 |
| F5.5 | 手动重试失败 step | P1 | Argo 原生支持 retry |
| F5.6 | 终止运行中的 workflow | P1 | Argo 支持停止 |
| F5.7 | 运行时间线 / Gantt 图 | P2 | 看各 step 耗时分布 |
| F5.8 | 资源使用量展示（CPU/Mem 实际使用 vs request） | P2 | 辅助调参 |
| F5.9 | argo-ui 内嵌集成 | P1 | ✅ nginx 已配 /argo/ 路径 |

---

### F6: Backfill（批量回放）

**用户故事**：
> 作为数据工程师，我升级了处理镜像或加了新算法，要对一批历史资产重新跑 pipeline。我希望能筛选出一批资产、选一个 pipeline template、一键启动批量回放，能看到进度。

**需求**：

| # | 需求 | 优先级 | 备注 |
|---|------|--------|------|
| F6.1 | 创建 backfill job（选 template + 筛选条件） | P1 | ✅ CYB-1267 |
| F6.2 | Backfill 进度（总数/已完成/失败） | P1 | ✅ CYB-1267 |
| F6.3 | 暂停/继续 backfill | P1 | ✅ CYB-1267 |
| F6.4 | Backfill 失败重试 | P1 | ✅ CYB-1267 |
| F6.5 | Backfill 历史查询 | P2 | ✅ CYB-1267 (GET /backfill + GET /backfill/:id) |
| F6.6 | 自动 backfill（新版本发布后自动触发） | P3 | 类似 Dagster 的 auto-materialize |

**设计参考**：

```
backfill_jobs 表:
  id, name, template_id, filter_json (筛选条件),
  total_count, completed_count, failed_count,
  status (running/paused/completed/failed), created_at

backfill_items 表:
  id, job_id, asset_id, status, workflow_name,
  error_message, started_at, finished_at

POST   /api/v1/backfill                   创建 backfill
GET    /api/v1/backfill                   列出 backfill
GET    /api/v1/backfill/:id               backfill 详情（含进度）
POST   /api/v1/backfill/:id/pause        暂停
POST   /api/v1/backfill/:id/resume       继续
POST   /api/v1/backfill/:id/retry-failed 重试失败的
```

---

### F7: Pipeline Template 管理

**用户故事**：
> 作为数据工程师，我搭好一条处理链路后存为 template，团队其他人可以直接用，不用重新拖。后续迭代 template 能发布新版本，已有 run 不受影响。

**需求**：

| # | 需求 | 优先级 | 备注 |
|---|------|--------|------|
| F7.1 | 保存当前画布为 template | P0 | ✅ 已实现 |
| F7.2 | 加载已有 template 到画布 | P0 | ✅ 已实现 |
| F7.3 | 列出所有 template | P0 | ✅ 已实现 |
| F7.4 | 删除 template | P0 | ✅ 已实现 |
| F7.5 | Template 版本管理 | P1 | ✅ CYB-1265 |
| F7.6 | Template 分享（跨用户） | P2 | 配合多租户 |
| F7.7 | Template 标签/分类 | P2 | 方便检索 |
| F7.8 | Template 从部署记录快速保存 | P2 | 跑成功了改一改另存 |

---

## 4. 数据模型

### 4.1 Pipeline 结构（transpiler 内部模型）

```go
type Pipeline struct {
    Name    string
    Version string
    Nodes   []Node
    Edges   []Edge
}

type Node struct {
    ID        string
    Component Component     // 镜像 + 参数
    Inputs    []Port
    Outputs   []Port
}

type Component struct {
    Image   string            // registry.example.com/my-processor:v1
    Command []string
    Args    []string
    Env     []EnvVar
    Resources ResourceRequirements
}

type Edge struct {
    Source      string        // "node-a.output-port"
    Target      string        // "node-b.input-port"
}
```

### 4.2 PG 持久化

```
pipeline_templates:
  id TEXT PK, name TEXT, pipeline JSONB (完整 DAG),
  node_count INT, created_at, updated_at

pipeline_deployments:
  id TEXT PK, template_id FK → pipeline_templates,
  pipeline_name TEXT, workflow_name TEXT,
  status TEXT, node_count INT,
  manifest TEXT (Argo YAML),
  pipeline_json JSONB (快照),
  created_at, finished_at

backfill_jobs:           ← P1 新增
  id UUID PK, name TEXT, template_id FK,
  filter_json JSONB, total_count INT,
  completed_count INT, failed_count INT,
  status TEXT, created_at, updated_at

backfill_items:          ← P1 新增
  id UUID PK, job_id FK → backfill_jobs,
  asset_id TEXT, status TEXT,
  workflow_name TEXT, error_message TEXT,
  started_at, finished_at
```

---

## 5. 分阶段实施路线

### Phase 0（当前状态 ✅）

| 功能 | 状态 |
|------|------|
| 组件侧边栏 + 拖拽到画布 | ✅ |
| 节点连线 + 参数编辑 | ✅ |
| 保存/加载/删除 template | ✅ |
| Transpiler 基础（DAG + 参数 + 资源） | ✅ |
| 部署（从画布 / 从 template） | ✅ |
| Deployment 列表/详情/删除 | ✅ |
| Workflow DAG 可视化 | ✅ |
| argo-ui 内嵌（nginx /argo/） | ✅ |

### Phase 1（接下来）

| 功能 | 预计工作量 | 依赖 |
|------|-----------|------|
| F1 组件注册表 API + 前端管理页 | 后端 ~150 行 + 前端 ~200 行 | 无 | ✅ CYB-1264 |
| F4.1 运行前选 asset | 前端 ~80 行 + transpiler ~50 行 | 无 | ✅ CYB-1263 |
| F3.7 retryStrategy 配到 transpiler | transpiler ~30 行 | 无 | ✅ CYB-1262 |
| F3.8 超时配置 | transpiler ~20 行 | 无 | ✅ CYB-1262 |
| F5.4/F5.5/F5.6 Step 操作（日志/重试/终止） | handler ~100 行 + 前端 ~150 行 | Argo API |
| F6 Backfill 全套（表 + API + 前端进度） | ~400 行 | F4.1 | ✅ CYB-1267 |
| F7.5 Template 版本管理 | ~80 行 | 无 | ✅ CYB-1265 |

**Phase 1 总估：~3-4 周**

### Phase 2（规模化和体验）

| 功能 | 预计 |
|------|------|
| F4.2~F4.7 Asset 深集成（血缘 + 事件 + 批量） | ~300 行 |
| F2.11 自动布局 | 前端 ~100 行 |
| F2.12 Pipeline diff | 前端 ~120 行 |
| F2.13 子图/嵌套 DAG | transpiler ~150 行 |
| F5.7/F5.8 时间线 + 资源监控 | ~200 行 |
| F7.6/F7.7 Template 协作 | ~80 行 |
| F3.9 并行度控制 | transpiler ~30 行 |

**Phase 2 总估：~4-5 周**

---

## 6. 架构集成点

```
Frontend (React Flow)
    │  DAG JSON
    ▼
Backend /api/v1/pipelines/* (Go)
    │
    ├──→ Transpiler → Argo Workflow YAML
    │       │
    │       ├──→ K8s API (argo clientset) → submit Workflow
    │       │         │
    │       │         └──→ Argo Workflow Controller
    │       │                  │
    │       │           ┌──────┴──────┐
    │       │           ▼             ▼
    │       │       Pod (step1)   Pod (step2)  ...
    │       │       └── 用户 Docker 镜像
    │       │
    │       └──→ pipeline_deployments (PG)
    │
    ├──→ Workflow 状态同步 (定时/事件)
    │
    └──→ Asset 集成:
         - 输入: 选 asset → 参数透传
         - 输出: 容器回调 API → 注册新 asset → asset_events
         - 血缘: asset_relations 记录 input↔output
```

---

## 7. 约束 & 边缘情况

| 约束 | 说明 |
|------|------|
| **镜像必须用户自建** | 平台不提供 processing SDK，用户写任何语言的镜像都行 |
| **容器无状态** | 所有持久化通过 API 回写，容器内不保存状态 |
| **K8s 资源有限** | 需要 parallelism 控制，防止打爆集群 |
| **Argo 不存大量历史** | 历史走 PG（pipeline_deployments），etcd 只放活跃 WF |
| **Template 不可变引用** | 部署时快照 pipeline_json，不因 template 更新而改变已部署的 run |
| **幂等部署** | 同一 pipeline + 同一参数重复部署不创建重复 WF |
| **输入不存在** | 选 asset 时校验 asset 是否存在 + lifecycle 状态是否允许处理 |

---

## 8. Open Questions

| 问题 | 状态 |
|------|------|
| 多租户隔离后 pipeline 资源归属？ | 待定（P2 多租户时确定） |
| 组件镜像是否支持 GPU 资源？ | 可以，transpiler 支持 resources.limits |
| Pipeline 中间结果存储在哪？ | 容器自己写 GCS / 挂 PVC |
| 产出的 asset 如何和输入关联血缘？ | asset_relations 表待扩充类型 |
| Step 日志是否需要在平台内嵌？ | argo-ui 已有；先集成，看用户反馈再决定是否自建 |
| 是否需要 notification（run 完成通知飞书/邮件）？ | 待定（可加在背书中） |
