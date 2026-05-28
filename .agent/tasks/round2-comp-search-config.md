# comp-search-config — 组件搜索 + 节点配置面板

## 参考源码（必须读）
- `/Users/rick/src/reference/visual-argo-workflows/services/ui/src/components/modals/template/Edit.tsx` — 节点编辑弹窗（分 Tab: General/Container/Script/Resource/Suspend）
- `/Users/rick/src/reference/visual-argo-workflows/services/ui/src/components/modals/template/Container.tsx` — 容器配置表单字段
- `/Users/rick/src/reference/visual-argo-workflows/services/ui/src/components/modals/template/Script.tsx` — 脚本配置表单字段
- `/Users/rick/src/reference/visual-argo-workflows/services/ui/src/components/modals/template/form-utils.ts` — 表单验证 + 初始值

## 任务

### 1. 组件搜索分类（搜索+分组）
- 在画布左侧组件面板（`ComponentManager.tsx`）加 `Input.Search` 按名称过滤
- 按 `type` 分组展示（Container / Script / 自定义），用 collapse 折叠
- 现有数据源：`GET /api/v1/pipeline-components`

### 2. 节点配置面板（双击弹窗）
- 双击画布节点 → 弹出 modal 编辑（参考 Edit.tsx 的 Tab 模式）
- 字段：命令（tags mode）、参数（tags mode）、环境变量（Form.List KEY/VALUE）、资源限制（CPU/内存）
- 根据节点类型展示不同表单（参考 Container.tsx / Script.tsx）
- 保存后提交到工作流合约（pipelineContract）

## 约束
- Ant Design 组件
- lint/build/test 通过
- 提交到 round2/comp-search-config 分支
