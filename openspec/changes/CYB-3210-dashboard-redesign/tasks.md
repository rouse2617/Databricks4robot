# 任务清单 — CYB-3210

## 任务 A：移除"视频预览"菜单项

- [ ] **修改 AppLayout.tsx**
  - 删除 line 43：`{ key: "/preview", icon: <VideoCameraOutlined />, label: "视频预览" },`
  - 删除 line 78 左右：路由判断 `if (pathname.startsWith("/preview"))`
  - 删除 line 91 左右：`pathname.startsWith("/preview")` 相关逻辑
  - 文件：`Frontend/src/components/AppLayout.tsx`

- [ ] **验证**
  - 导航栏不显示"视频预览"
  - 访问 `/preview` 显示 404 或重定向到 `/dashboard`

---

## 任务 B：Dashboard 页面重构（CYB-3210）

### Phase 1：KPI 卡片重构

- [ ] **修改 DashboardPage.tsx**
  - 找到 KPI 卡片的 Row/Col 布局
  - 改成网格化：`display: grid; gridTemplateColumns: repeat(auto-fit, minmax(240px, 1fr));`
  - 调整 KPI 卡片高度：`minHeight: 140px`（增加 hit target）
  - 优化间距：`gap: 16px`
  - 文件：`Frontend/src/pages/DashboardPage.tsx`

- [ ] **验证**
  - 1024px：KPI 卡片折行成 1-2 列
  - 1600px：KPI 卡片 3-4 列显示
  - 卡片点击区域更大

### Phase 2：图表容器 max-width

- [ ] **添加 max-width 容器**
  - 在 DashboardPage 的最外层添加 `maxWidth: 1600px, margin: "0 auto"`
  - 确保图表和 KPI 都受 max-width 限制
  - 文件：`Frontend/src/pages/DashboardPage.tsx`

- [ ] **验证**
  - 2560px 窗口：Dashboard 内容宽度仍是 1600px，两侧留白

### Phase 3：加载状态优化（可选，后续）

- [ ] 为 KPI 卡片添加骨架屏
- [ ] 图表加载时显示 Skeleton

---

## 集成与测试

- [ ] **全屏幕宽度测试**
  - 1024px（窄屏）
  - 1600px（标准）
  - 2560px（带鱼屏）
  - 验证布局自适应

- [ ] **功能测试**
  - 导航栏确实删除了"视频预览"
  - Dashboard 能正常加载数据
  - 无控制台错误

---

## 提交流程

- [ ] **Task A 提交**
  - Message: `refactor(frontend): remove video preview menu item`

- [ ] **Task B Phase 1-2 提交**
  - Message: `feat(frontend): cyb-3210 dashboard page redesign (phases 1-2)`
  - 包含 KPI 卡片重构 + max-width 优化

- [ ] **创建 PR**
  - Base: `dev`
  - Title: `refactor(frontend): remove video preview & cyb-3210 dashboard redesign`
  - 同时包含两个改动

---

## 时间估计

- Task A（菜单移除）：5-10 分钟
- Phase 1（卡片重构）：15-20 分钟
- Phase 2（max-width）：10-15 分钟
- 测试：10-15 分钟
- **总计：40-60 分钟**
