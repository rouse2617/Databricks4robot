# CYB-1254 — 将 Pipeline Designer 合入主 Frontend

## 背景

`databrew-pipeline/` 是一个独立的流水线设计器应用，基于 React Flow（@xyflow/react）构建，支持拖拽编排 pipeline 节点、组件注册管理、部署到 Argo Workflow。它之前被移出 PR scope，现已在 `feat/pipeline-integration` 分支上恢复。

当前它作为一个独立的 Vite 应用运行，有自己的路由、布局、样式系统。目标是把它合入主 `Frontend/`，成为主应用的一个页面 /pipeline。

## 目标

1. 将 databrew-pipeline/ui 的 React Flow canvas 作为新页面集成到 Frontend
2. 保留拖拽编排、组件注册、部署面板三个核心功能
3. 复用主应用的 antd 组件和设计系统
4. 保持 React Flow canvas 自身的交互体验（拖拽、连线、配置面板）

## 非目标

- 不改动 transpiler（Go 后端）
- 不替换 databrew-pipeline/argo-ui（Argo 的 fork，独立运行）
- 不涉及后端 API 变更
