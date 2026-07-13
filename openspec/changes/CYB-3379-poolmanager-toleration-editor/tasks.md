# Tasks — CYB-3379

## API client(pipelineApi.ts)

- [ ] 添加 `updateExecutionTarget(id: string, body: Partial<ExecutionTarget>): Promise<ExecutionTarget>` → `PUT /api/v1/execution-targets/:id`
- [ ] 添加 `createExecutionTarget(body: Partial<ExecutionTarget>): Promise<ExecutionTarget>` → `POST /api/v1/execution-targets`
- [ ] 添加 `deleteExecutionTarget(id: string): Promise<void>` → `DELETE /api/v1/execution-targets/:id`
- [ ] `ExecutionTarget` interface 若缺 `resourceDefaults.templateTolerations` / `templateNodeSelector` 类型,补全

## Form editor(PoolManager.tsx)

- [ ] Modal 表单加 `templateTolerations` list editor(antd `Form.List`),每项 4 field:
  - `key`(string,必填)
  - `operator`(Select:`Equal` / `Exists`,默认 `Equal`)
  - `value`(string,`Exists` 时可为空)
  - `effect`(Select:`NoSchedule` / `NoExecute` / `PreferNoSchedule`,默认 `NoSchedule`)
- [ ] Modal 表单加 `templateNodeSelector` map editor(antd `Form.List` 或类似),每项 `key` / `value`
- [ ] 表单校验规则:`key` trim 后非空;`operator=Exists` 时 `value` 可空,`Equal` 时 `value` 必填
- [ ] wire「新建」按钮 → 调 `createExecutionTarget`;「编辑」→ `updateExecutionTarget`;「删除」→ `deleteExecutionTarget`(带确认对话框)
- [ ] 提交成功后刷新列表(重新 `listExecutionTargets` 或 optimistic update)
- [ ] Error handling:HTTP 4xx/5xx 显示 antd `message.error` + 表单不关闭

## 测试

- [ ] `PoolManager.test.tsx` 补:
  - 打开新建 Modal,填 name/namespace/toleration 一条,提交 → 断言 `createExecutionTarget` 被调用,body 正确
  - 打开编辑 Modal(预填现有 target),改 toleration value,提交 → 断言 `updateExecutionTarget` 被调用
  - 点删除 + 确认 → 断言 `deleteExecutionTarget` 被调用
  - `operator=Exists` 时 value 空提交也成功;`operator=Equal` 且 value 空时表单校验拒绝
- [ ] `cd Frontend && npm run lint`
- [ ] `cd Frontend && npm run test -- --run` PoolManager.test
- [ ] `cd Frontend && npm run build`(Tier L)

## Deploy + 验证(前端)

- [ ] merge 到 dev 后 CI 自动 build/push CloudFlare Worker frontend
- [ ] Chrome DevTools MCP 打开 dev 前端 `/registry`,进「资源池」tab
- [ ] 编辑 video-proc-dev target,添加一条 toleration,保存 → 后端 GET 验证持久化
- [ ] 与 CYB-3378 用 curl PUT 修的结果对比一致

## PR

- [ ] `gh pr create --base dev`,Linear ID = CYB-3379,贴 lint/test/build 输出 + Chrome MCP 截图
- [ ] **等用户 review 后由用户合并**
