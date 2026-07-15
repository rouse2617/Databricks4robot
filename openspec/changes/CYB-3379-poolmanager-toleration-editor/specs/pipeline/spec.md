# pipeline spec delta — CYB-3379

## MODIFIED — execution-target 编辑 UI

- **Given** 已登录用户在前端「注册中心」→「资源池」tab
- **When** 用户点「编辑」按钮打开某个 execution target 的编辑 Modal
- **Then** Modal 表单必须显示当前 target 的 `resourceDefaults.templateTolerations` 和 `templateNodeSelector` 完整列表,可增删条目、修改每一项的 key/value/operator/effect

- **Given** 用户在 Modal 表单里保存合法数据(所有 toleration 的 key 非空、`operator=Equal` 时 value 非空)
- **When** 用户点「保存」
- **Then** 前端必须调 `PUT /api/v1/execution-targets/:id`,body 含完整 target(含改后 `templateTolerations` / `templateNodeSelector`);成功 200 后 Modal 关闭并刷新列表

- **Given** 用户填的 toleration 里某项 `operator=Equal` 且 `value` 为空
- **When** 提交表单
- **Then** 表单校验拒绝,显示错误"value 必填(operator=Equal 时)",Modal 不关闭

- **Given** 用户点表格里某 target 的「删除」按钮
- **When** 弹出的确认对话框点「确定」
- **Then** 前端调 `DELETE /api/v1/execution-targets/:id`,成功后从列表移除

- **Given** target `isDefault=true`
- **Then** 删除按钮禁用或隐藏(避免删掉默认 target 导致所有 pipeline 崩)
