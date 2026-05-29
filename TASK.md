# 合并流水线三个页面为一个 Tab 页面

## 背景

当前流水线模块有三个独立路由和三个独立页面：

| 路由 | 文件 | 内容 |
|------|------|------|
| `/components` | `ComponentListPage.tsx` | 步骤组件 CRUD 管理 |
| `/pipeline` | `PipelinePage.tsx` | 流水线画布设计 |
| `/workflows` | `WorkflowListPage.tsx` | 流水线执行记录列表 |

用户要求把这三个合并到 `/pipeline` 一个页面，用 Ant Design Tabs 切换。

## 技术栈

- React 19 + TypeScript
- Ant Design 5 (Tabs, Table, Form, etc.)
- React Router 6
- @ant-design/pro-flow (画布)

## 具体任务

### 1. PipelinePage.tsx — 重构为 Tab 容器

文件路径: `Frontend/src/pages/PipelinePage.tsx`

将现有的 PipelineCanvas（canvas+组件面板+部署面板）包装在一个 Tabs 组件中，作为"设计"Tab 的内容。

Tab 结构：
```
[设计] [执行记录] [组件]
```

**"设计" Tab** — 保留现有全量功能：
- PipelineCanvas (flow editor, component palette, node config, deploy panel)
- 保持 full-bleed 布局（无 padding）
- 保留 `view` state（pipeline | deploy 切换）

**"执行记录" Tab** — 嵌入 WorkflowListPage 内容：
- 执行记录表格 + 状态统计 + 筛选
- 保留操作按钮（重新提交、重试、终止、删除）
- 查看按钮仍然跳转到 `/workflows/:name`
- 给适当的内边距（padding: 24px）

**"组件" Tab** — 嵌入 ComponentListPage 内容：
- 组件管理表格 + CRUD 弹窗
- 搜索、新建、编辑、删除
- 给适当的内边距（padding: 24px）

Tab 切换通过 URL search params 同步：
- `/pipeline` → 默认显示"设计"Tab
- `/pipeline?tab=executions` → "执行记录"Tab
- `/pipeline?tab=components` → "组件"Tab

实现方式：将 ComponentListPage 和 WorkflowListPage 的表格内容提取为独立组件（`ComponentManager` 和 `WorkflowExecutionList`），在 PipelinePage 中 import 并在 Tabs 中使用。

### 2. App.tsx — 调整路由

文件路径: `Frontend/src/App.tsx`

- 删除 `/components` 路由（重定向到 `/pipeline?tab=components`）
- 删除 `/workflows` 路由（重定向到 `/pipeline?tab=executions`）
- **保留** `/workflows/:name` 路由（WorkflowDetailPage 详情页）

### 3. AppLayout.tsx — 合并侧边栏菜单

文件路径: `Frontend/src/components/AppLayout.tsx`

- 删除侧边栏中的"步骤组件"(`/components`) 和"流水线执行记录"(`/workflows`) 菜单项
- 将"流水线设计"(`/pipeline`) 改为"流水线"
- 更新 `resolveSelectedKey` 函数，移除 `/components` 和 `/workflows` 的映射
- 更新 `isFullBleedPage` 函数，仅 `/pipeline` 仍然 full-bleed（其余 tab 内容已有自己的 padding）

### 4. 提交

```bash
git add -A && git commit -m "feat(frontend): merge pipeline pages into tabs" --no-verify
```

## 硬约束

1. **保留 WorkflowDetailPage** — `/workflows/:name` 路由必须继续工作，不受影响
2. **保留 PipelineCanvas 所有功能** — 节点拖拽、组件面板、部署弹窗、导入/导出 JSON、模板保存等，一个都不能少
3. **保留 WorkflowListPage 和 ComponentListPage 的完整功能** — 筛选、分页、CRUD 操作、操作按钮都要正常
4. **Tab 切换保留状态** — 切换 Tab 后再切回来，之前的筛选/搜索条件不应该重置（用 React key 或 state 管理）
5. **不要破坏 TypeScript 编译** — 改完后跑 `npx tsc --noEmit` 确认无新错误

## 验证

```bash
cd /Users/rick/cyber-databrew && git -C /tmp/worktree-pages-merge diff --name-only
cd /tmp/worktree-pages-merge/Frontend && npx tsc --noEmit 2>&1 | head -20
```

## 参考文件（必读）

- `/tmp/worktree-pages-merge/Frontend/src/pages/PipelinePage.tsx` — 主页面，需要重构
- `/tmp/worktree-pages-merge/Frontend/src/pages/ComponentListPage.tsx` — 组件管理，提取内容
- `/tmp/worktree-pages-merge/Frontend/src/pages/WorkflowListPage.tsx` — 执行记录，提取内容
- `/tmp/worktree-pages-merge/Frontend/src/App.tsx` — 路由配置
- `/tmp/worktree-pages-merge/Frontend/src/components/AppLayout.tsx` — 侧边栏
