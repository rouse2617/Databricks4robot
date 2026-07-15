# CYB-3379 — PoolManager UI 补 templateTolerations / templateNodeSelector 编辑

## Why

`execution_targets.resourceDefaults.{templateTolerations, templateNodeSelector}` 是 DataBrew pipeline pod 调度落地的关键 config。当集群运维改节点污点(如 `environment=dev` → `compute-tier=med / durability=spot`),这两个字段必须同步更新,否则 pod 卡 Pending。

CYB-3378 的调研已经证明:
- 后端 `scheduling.go` 是 config-driven 的完整实现,`GET/POST/PUT/DELETE /api/v1/execution-targets/:id` 全都在
- 前端 `PoolManager`(`RegistryCenterPage` → 「注册中心」→「资源池」tab)**只 wire 了 `listExecutionTargets`(读)**,`resourceDefaults` 完全没暴露在编辑表单里
- 每次 config 漂移都要人工 curl PUT,门槛不合理

## What Changes(纯前端,后端零改动)

- **`Frontend/src/api/pipelineApi.ts`**:补三个 API client 函数
  - `updateExecutionTarget(id, body): Promise<ExecutionTarget>` → PUT
  - `createExecutionTarget(body): Promise<ExecutionTarget>` → POST
  - `deleteExecutionTarget(id): Promise<void>` → DELETE
- **`Frontend/src/components/pipeline/PoolManager.tsx`**:
  - Modal 表单加两个 field editor:
    - `templateTolerations`:list 每项 `key` / `operator (Equal|Exists)` / `value` / `effect (NoSchedule|NoExecute|PreferNoSchedule)`,可增删
    - `templateNodeSelector`:map(key/value)可增删
  - wire 现有的编辑/新建/删除按钮到对应 API client
  - 表单校验:`key` 非空;`operator=Exists` 时允许 `value` 空
- **`Frontend/src/components/pipeline/PoolManager.test.tsx`**:补 Vitest 单测覆盖新增 field + 提交/删除路径

## Impact / 行为变更

- **UI 层**:资源池 tab 里 Modal 从「只看」变成「可增删改」;删除按钮生效(现在可能是空实现)
- **后端**:**零改动**,复用现有 API + 权限 scope(`assets:write` 或等价 admin)
- **数据契约**:与 CYB-3378 用 curl 走的完全同一个字段路径,行为一致
- 无 schema/migration/auth/outbox 变更;不触 off-limits

## Out of scope

- ❌ 通用「资源门槛 → toleration set」智能匹配(承接 CYB-3378 判断:一个真实场景不够)
- ❌ K8s API 自动感知节点污点
- ❌ 后端改动(已就绪)
- ❌ RegistryCenterPage 其他 tab 或 SettingsPage 无关改动

## 验证

- Vitest 单测覆盖:表单 valid submit / 增删 toleration / delete confirmation / error handling
- Chrome DevTools MCP 验收(dev deploy 后):在 UI 上把 video-proc-dev target 的 tolerations 添加一条、删除一条、修改 value,刷新页面确认持久化;跟 CYB-3378 curl PUT 效果一致
