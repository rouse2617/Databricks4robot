# Pipeline 修复任务拆分

## 环境确认
当前 dev 环境：`pipelines 404`、`deployments 404`、`workflows 404`、`components 200`
根因：`k8s.NewClient` 在 Cloud Run 上失败 → `pipelineHandler`/`workflowHandler` 为 nil → 路由不注册

---

## 任务 A — 后端：路由注册（P0-1）

**问题**：`routes.go` 中 pipeline/workflow 路由被 `if pipelineHandler != nil` 保护，但 K8s client 在 Cloud Run 上因无 in-cluster kubeconfig 初始化失败。

**修复方案**（二选一）：
- 方案 1（推荐）：让 K8s client 在 Cloud Run 上用 Workload Identity + GKE REST API，不做 in-cluster 假设
- 方案 2（快速）：无论 K8s 是否可用都注册路由，handler 内做 graceful degradation（返回 503 而不是 404）

**文件**：
- `backend/cmd/server/infra.go` — K8s client 初始化
- `backend/internal/k8s/client.go` — K8s client 实现
- `backend/routes/routes.go` — 路由注册（条件判断）

---

## 任务 B — 前端：组件重复（P1-1）

**问题**：`GET /api/v1/components` 返回两条 Pass Through（用户 seed + `sys-pass-through`），侧栏/注册表重复显示。

**文件**：`Frontend/src/pages/PipelinePage.tsx` — `apiToRegistered()`

**改法**：去重逻辑，按 name 过滤或后端 seed 幂等合并。

---

## 任务 C — 前端：镜像 tag 重复（P1-2）

**问题**：`apiToRegistered()` 在 `api.tag` 存在时拼 `` `${api.image}:${api.tag}` ``，若 DB 中 `image` 已含 `:latest` 则显示 `busybox:latest:latest`。

**文件**：`Frontend/src/pages/PipelinePage.tsx`

**改法**：修 `apiToRegistered`，`image` 已含 tag 时不再拼 `:${tag}`；展示层统一 `formatImage(image, tag)`。

---

## 任务 D — 前端：组件端口默认值（P1-3）

**问题**：部分组件 `inputPorts`/`outputPorts` 为空。

**改法**：创建组件表单保证每组件至少 `input` + `output` 默认端口。

---

## 任务 E — 前端：UI 打磨（P2）

| ID | 问题 | 改法 |
|----|------|------|
| P2-1 | 画布节点过大 | `pipeline.css` 中 `.pipeline-node` min-width 从 180px 降到 ~140px |
| P2-2 | 空组件库文案 | "Registry" → "组件 Tab" |
| P2-3 | 画布空态引导 | 增加 onboarding 文案 |
| P2-4 | 部署 Tab 标题 | "运行记录" → "已保存模板 / 部署历史" |
| P2-5 | API 失败静默 | 404 时显示 Alert |
| P2-6 | a11y | Input 补 id + htmlFor |
