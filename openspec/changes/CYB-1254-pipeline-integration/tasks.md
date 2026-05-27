# Tasks — CYB-1254

## 实施步骤

- [ ] 1. 安装 @xyflow/react 到 Frontend
- [ ] 2. 创建 `api/pipelineApi.ts`（集成 pipeline API 客户端）
- [ ] 3. 创建 `components/pipeline/` 组件目录结构
  - [ ] 3a. PipelineNode.tsx（自定义 React Flow node）
  - [ ] 3b. ComponentPalette.tsx（拖拽面板）
  - [ ] 3c. NodeConfigPanel.tsx（节点配置）
  - [ ] 3d. ComponentManager.tsx（组件注册管理）
  - [ ] 3e. DeployPanel.tsx（部署列表）
- [ ] 4. 创建 `PipelinePage.tsx`（主页面，整合 3 个 tab）
- [ ] 5. 创建 `styles/pipeline.css`（canvas 样式）
- [ ] 6. 修改 `App.tsx`（添加 /pipeline 路由）
- [ ] 7. 修改 `AppLayout.tsx`（内部导航，非外部链接）
- [ ] 8. 验证：`npm run build` 通过
- [ ] 9. 前端 dev deploy → MCP 验证
- [ ] 10. PR
