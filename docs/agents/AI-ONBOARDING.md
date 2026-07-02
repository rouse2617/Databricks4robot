# AI 助手参与指引 — cyber-databrew

## 项目概况

cyber-databrew 是一个数据编排平台（Data Orchestration Platform），核心功能是**流水线设计→部署→监控**。用户通过 Web UI 拖拽 DAG 节点构建数据流水线，提交到 Argo Workflows 执行。

## 技术栈

| 层 | 技术 |
|------|------|
| 前端 | React 19 + TypeScript + Vite + Ant Design + XYFlow/ReactFlow |
| 后端 | Go + Gin + PostgreSQL |
| 工作流 | Argo Workflows (K8s CRD) |
| K8s | GKE (生产) + k3d (本地开发) |
| 调度增强 | Koordinator (ElasticQuota) |

## 仓库结构（Frontend/）

```
Frontend/src/
├── api/                  # API 客户端
│   ├── pipelineApi.ts    # Pipeline/部署相关 API
│   ├── runApi.ts         # Run(执行记录) API
│   ├── pipelineConfigs.ts# 配置文件 API
│   └── workflowApi.ts    # Argo workflow API
├── components/
│   ├── pipeline/         # 流水线组件
│   │   ├── DeployPanel.tsx     # 部署弹窗（选择执行目标/配置）
│   │   ├── PoolManager.tsx     # 资源池管理页面
│   │   ├── PipelineNode.tsx    # 画布上的节点组件
│   │   ├── NodeConfigPanel.tsx # 节点配置面板
│   │   └── types.ts           # 核心类型定义
│   ├── common/            # 通用组件
│   └── asset-detail/      # 资产详情
├── features/
│   └── pipeline-designer/ # 流水线设计器(ReactFlow 画布)
│       ├── PipelineDesignerCanvas.tsx  # 主画布
│       └── hooks/         # 自定义 hooks
├── hooks/                 # 通用 hooks
├── lib/                   # 工具函数
├── pages/                 # 页面
│   ├── WorkflowDetailPage.tsx   # Run 详情页（关键！2516 行）
│   ├── RegistryCenterPage.tsx   # 注册中心（含资源池 Tab）
│   ├── PipelinePage.tsx         # 流水线页面
│   ├── WorkflowExecutionList.tsx# 执行记录列表
│   └── workflowLogView.tsx      # 日志查看器
└── styles/
    └── pipeline.css       # 流水线相关样式（2332 行）
```

## 当前开发状态

### 已实现的核心功能

```
流水线设计:  ReactFlow 拖拽 DAG, 节点配置, 保存/部署
Run 详情:    分屏布局(画布上+Tab下), 可拖拽 Resizer, KV 元数据
日志查看器:  暗色终端, 行号, 错误高亮, 前缀折叠
执行记录:    列表分页, 状态过滤, 自动刷新
资源池管理:  注册中心 Tab, 实时 CPU/MEM 进度条, CRUD
批量任务:    20路并行提交, Argo parallelism 控制并发, 一键停止
Config 版本: 版本选择器, mountPath/targetFilename
```

### 正在进行

```
GKE 部署验证:  后端已部署 Cloud Run, 前端待验证
性能调优:      runs API 分页硬上限 200
```

## 关键架构决策

### 1. 执行目标 vs 资源池

用户看到的是「资源池」，底层是 `execution_targets` 表：
```sql
execution_targets(id, name, cluster, namespace, ...)
-- name 显示给用户：'存储池A · 3~6C · 4~8Gi'
-- namespace 指向 K8s namespace
```

部署时选池 = 选 execution_target，Workflow 提交到对应 namespace。

### 2. 实时用量

`GET /api/v1/resource-quotas` 从 K8s ResourceQuota API 实时读取。
前端 PoolManager 和 DeployPanel 通过这个 API 展示 CPU/MEM 进度条。

### 3. 批量处理

```
前端提交:
  POST /api/v1/runs/batch { template_id, asset_ids, max_concurrency }
后端:
  20路并行 → 每个提交 Argo Workflow
  Argo controller parallelism=20 控制并发
  一键停止: POST /runs/batch/:batchId/stop
```

### 4. 资源池 CRUD

```
GET    /api/v1/execution-targets         # 列表
POST   /api/v1/execution-targets         # 创建
PUT    /api/v1/execution-targets/:id     # 更新
DELETE /api/v1/execution-targets/:id     # 删除
```

## 常见开发任务

### 本地启动（k3d + 全栈）

```bash
# 0. 确认在本地 context
kubectl config use-context k3d-cyber-local
kubectl get nodes

# 1. PostgreSQL (port-forward)
kubectl port-forward -n cyber-databrew-dev svc/postgres 5432:5432

# 2. Argo (port-forward)
kubectl port-forward -n cyber-databrew-dev svc/argo-workflows-server 2746:2746

# 3. 后端
cd backend && CGO_ENABLED=0 go build -o /tmp/cyber-backend ./cmd/server/
ARGO_SERVER_URL=http://localhost:2746 DB_HOST=localhost DB_PORT=5432 \
DB_USER=postgres DB_PASSWORD=postgres DB_NAME=cyber_databrew_dev \
ENV=development ARGO_WORKFLOWS_NAMESPACE=cyber-databrew-dev \
LAKEHOUSE_BACKEND=none STORAGE_BACKEND=postgres ARGO_INSECURE_SKIP_TLS=true \
/tmp/cyber-backend

# 4. 前端
cd Frontend && VITE_API_BASE_URL=http://localhost:8080 VITE_DEV_ACCESS_TOKEN=dev-token npm run dev
```

### TypeScript 编译检查

```bash
cd Frontend && ./node_modules/.bin/tsc --noEmit
```

### 提交规则

```bash
# commitlint 检查通过才能提交
git add <files>
git commit -m "type(scope): description

- bullet points of changes

Co-Authored-By: Claude <noreply@anthropic.com>"
git push origin dev --no-verify  # 跳过 pre-commit hooks
```

### Playwright 测试

```bash
node -e "
const { chromium } = require('./node_modules/playwright');
// 测试页面渲染
"
```

## K8s 资源池创建

```bash
# 创建池 (namespace + ResourceQuota + LimitRange)
kubectl create ns pool-xxx
kubectl create quota pool-quota -n pool-xxx --hard=requests.cpu=3,limits.cpu=6,requests.memory=4Gi,limits.memory=8Gi

# 写入 DB（前端才能看到）
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres \
  -d cyber_databrew_dev \
  -c "INSERT INTO execution_targets (...) VALUES (...)"
```

## 关键约束

1. **不要误操作 GKE 集群** — 操作前确认 `kubectl config current-context`
2. **不要未经用户同意 push 代码**
3. **Go 后端用 `CGO_ENABLED=0` 编译**
4. **commitlint 规则**：subject 小写，scope 限 `[backend, frontend, sdk, docs, dagster, deploy, api]`
5. **前端 antd import**：注意样式兼容，antd v5 不要在 v19 React 下用废弃 API
6. **Koordinator** 只在 k3d 本地测试，GKE 上不装

## 环境地址

| 环境 | 前端 | 后端 |
|------|------|------|
| 本地 | http://localhost:5176 | http://localhost:8080 |
| GKE dev | https://cyber-databrew-dev.cyberorigin.ai | Cloud Run |
| Argo | http://localhost:2746 (本地) | in-cluster (GKE) |
| DB | localhost:5432 (port-forward) | Cloud SQL (GKE) |
| 注册中心 | /registry | — |
| Run 详情 | /runs/:runId | — |
| 资源池 | /registry → 资源池 tab | /api/v1/execution-targets |

## 当前遗留问题

- [ ] 垂直时间线样式优化（parse error 待修）
- [ ] 前端 build 后部署到 Cloud Run
- [ ] ResourceQuota API 的 K8s RBAC 在 GKE 上验证
- [ ] Pool namespace auto-provision（当前手动创建）
