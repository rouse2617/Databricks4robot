# Task: Create Linear Issue — Argo UI 集成方案

## 背景
我们目前有自定义前端（React + Ant Design），实现了 Workflow 编排、列表、详情页。但之前发现详情页缺失日志查看器、节点详情面板、参数展示等核心功能。

通过对比 Argo Workflows UI（http://localhost:2746/workflows/sandbox-project-a），发现其详情页功能完整（DAG 图、节点面板、日志 streaming、参数展示），且已生产验证。

Rick 提出：**不应重写详情页，应拿 Argo UI 当底座，只替换 Submit New Workflow 页面为我们的拖拽画布。**

需要你创建 Linear Issue 记录这个分析。

## Issue 内容

### Title
`[RFE] 复用 Argo UI 底座，替换 Submit 入口为拖拽画布`

### Team
用 cyber-databrew 的项目团队（搜索 "cyber" 或 "databrew" 相关的 team key）

### Labels
- enhancement
- frontend
- architecture

### Description

用以下内容创建 Issue：

```markdown
## 问题
当前前端详情页缺失核心功能：日志查看器（API 返回空）、节点详情面板（点节点无反应）、参数展示、YAML 查看。

## 分析
通过对比 Argo Workflows UI（[Argo Workflows](http://localhost:2746/workflows/sandbox-project-a)），发现：
- Argo UI 的 workflow 详情页功能完整（日志 streaming + pod/container 切换 + 正则过滤、节点点击弹面板展示输入/输出参数、YAML 查看/下载）
- Argo UI 是 Apache 2.0 开源，React 前端，社区成熟
- 我们的核心增量是**编排体验**——拖拽画布 + 自定义组件库

## 建议方案
拿 Argo UI 当底座，只改 Submit New Workflow 入口：

```
Argo UI ─┬─ Workflow 列表页 ── 复用 ✅
          ├─ Workflow 详情页（DAG图、日志、参数、YAML） ── 复用 ✅
          └─ Submit New Workflow ── 替换为拖拽画布 + 组件库
```

这样：
- 日志查看器 ✅ 现成的 streaming + pod/container 切换 + 正则过滤
- 节点详情面板 ✅ 现成的输入输出参数展示
- YAML 查看/下载 ✅ 现成的
- 我们只需要做：drag & drop 画布 → 生成 pipeline JSON → 调用 Argo 的 submit API

## 待确认
1. 集成方式：iframe 嵌入 vs monorepo fork 融合 vs 其他？
2. 是否需要在当前前端和 Argo UI 之间做认证/路由融合
3. 是否需要保留当前前端中的一些定制功能（如 SSE 事件流、部署预览等）

## 工作量估计
- 原来方案（重写详情页）：2-3 周
- 新方案（集成 Argo UI + 编排页面）：3-5 天
```

### Priority
3 (Medium)

## Linear API
如果需要手动调用，API Key 在环境变量 LINEAR_API_KEY 中。如果已经配置了 Linear MCP，用 MCP 更简单。
