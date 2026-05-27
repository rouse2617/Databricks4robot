# Design — Pipeline Integration

## 架构

```
Frontend/src/
├── pages/
│   └── PipelinePage.tsx          # 主页面，3 个 tab 视图
├── components/pipeline/
│   ├── PipelineCanvas.tsx        # React Flow canvas（核心交互）
│   ├── PipelineNode.tsx          # 自定义 canvas node
│   ├── ComponentPalette.tsx      # 拖拽组件面板
│   ├── NodeConfigPanel.tsx       # 选中节点的配置面板
│   ├── ComponentManager.tsx      # 组件注册管理
│   └── DeployPanel.tsx           # 部署历史 / 模板管理
├── api/
│   └── pipelineApi.ts            # pipeline API 客户端
└── styles/
    └── pipeline.css              # React Flow & 管道画布样式
```

## 集成方式

### 路由

在 `App.tsx` 添加 `/pipeline` 路由，懒加载 `PipelinePage`。

### 布局

PipelinePage 使用 antd Tabs 组件替代原来的 tab nav，在内容区展示：
- **Pipeline** — 三栏布局：Palette | Canvas | ConfigPanel
- **Registry** — ComponentManager 组件
- **Deploy** — 部署历史与模板管理

### 依赖

新增 `@xyflow/react` 到 Frontend/package.json。

### 菜单

AppLayout.tsx 中 `/pipeline` 改为内部导航（`navigate("/pipeline")`），移除外部 URL 跳转。

## 样式策略

- React Flow canvas 的节点样式保持原有自定义 CSS（`pipeline.css`）
- 按钮、表单、Modal 等改用 antd 组件
- 复用主应用的 design-tokens（色板、字体、间距）

## 状态

- PipelinePage 内部 useState/useCallback 管理本地状态
- Component Registry 暂存 localStorage（与原有行为一致）
- 部署列表通过 API 获取

## 兼容性

- 不影响现有页面和路由
- `/pipeline` 在原 AppLayout 中已有菜单项，只需改导航方式
- 原有 standalone 的 `databrew-pipeline/ui/` 目录保留在 repo 中（不删除）
