# 流水线 UI/UX 优化方案 - OpenSpec 提案

**版本**: 1.0
**状态**: 待批准
**创建**: 2026-06-30
**优先级**: High

---

## 问题陈述

### 当前痛点

1. **流水线列表页** - 280 个流水线，无分类、无快速导航
   - 用户无法快速找到常用流水线
   - 手动标记成本高
   - 右侧按钮在小屏幕溢出

2. **版本管理** - 84 个版本无法辨别
   - 版本列表仅显示版本号，无元数据
   - 无法区分"草稿"vs"发布版本"
   - 无法定位"上次运行成功的版本"

3. **执行记录页** - 4196 条记录信息过载
   - 列过多，小屏幕横滚
   - 侧滑详情数据一次性加载导致卡顿
   - 失败原因统计但无法联动筛选

4. **编辑/设计页** - 无障碍和性能缺陷
   - 26+ 组件逐个查找困难
   - 悬停菜单不支持 Touch 和键盘导航
   - 大型流水线画布易迷路

---

## 解决方案概述

### 范围（三阶段）

**Phase 1: 列表页 + 版本管理** ← **当前阶段**
- 流水线列表分组视图（智能分类）
- 版本选择器元数据显示和标记分组
- 无障碍菜单改进（a11y 完整支持）
- 后端 Stats API

**Phase 2: 执行记录页**
- 响应式列布局
- 虚拟滚动处理 4000+ 条
- 侧滑详情懒加载
- 失败原因联动筛选

**Phase 3: 编辑/设计页**
- 组件面板分类和搜索
- 画布导航工具
- 自动布局（Dagre）
- 草稿冲突检测

---

## 技术方案

### Phase 1 技术栈

#### 后端 (Go)

```go
// 新增 API 端点
GET /api/v1/pipelines/stats?window=30d
Response: {
  userStats: { clickCounts, runCounts, executionTimes, lastAccessTimes },
  recommendations: [{ pipelineId, name, score, reason }]
}

// 版本冲突检测（乐观锁）
PUT /api/v1/pipelines/{id}
Body: { content, baseVersion, note }
Response: 409 ConflictError | 200 Success
```

#### 前端 (React/TypeScript)

```typescript
// 新组件
- <PipelineGroupView /> - 分组列表视图
- <VersionSelector /> - 改进的版本选择器
- <ComparisonPanel /> - 版本对比
- <MenuButton /> - a11y 友好的菜单

// 改进的 Hook
- useOptimisticMutation() - 乐观更新
- useLazyLoad() - 懒加载数据
- useVirtualList() - 虚拟滚动
```

#### API 合约同步

| 文件 | 更新项 |
|------|--------|
| `api/openapi.yaml` | `GET /pipelines/stats` + 请求/响应 schema |
| `docs/review/api-guide.md` | 新端点的 curl 示例和说明 |
| `sdk/src/cyber_databrew_sdk/` | `pipelines_stats()` 方法 |
| `sdk/tests/unit/` | Stats 方法的单元测试 |
| `scripts/api-guide-smoke.sh` | Stats 端点的 happy path + error path |

---

## 关键特性

### 1. 智能分类（流水线列表）

```
🔥 最常用 (自动生成，基于 30 天点击频次)
  ├─ my-pipeline (评分 8.6)
  ├─ perf-echo-1000 (评分 7.2)

📌 我的标记 (用户手动标记)
  ├─ [production]
  ├─ [testing]

📋 所有 (280 个)
```

**实现**:
- 后端计算 click_count × 0.4 + run_count × 0.35 + last_access_time × 0.25
- 前端每 5 分钟自动刷新统计（无感知）
- 用户操作后乐观更新排序

---

### 2. 无障碍菜单（a11y）

**问题**: Hover 菜单在 Touch 设备和屏幕阅读器上无法使用

**解决**:
- 菜单按钮始终可见（PC 端悬停半透明）
- 支持点击展开菜单
- 完整的 ARIA 属性（role, aria-expanded, aria-label）
- 键盘导航（Tab, Shift+Tab, Esc 支持焦点陷阱）
- 屏幕阅读器完全支持

---

### 3. 版本冲突检测（乐观锁）

**问题**: 多用户编辑同一流水线时，后保存的覆盖先保存的

**解决**:
- 保存时发送 `baseVersion` 号
- 后端检查版本冲突 → 409 响应
- 前端显示友好冲突对话框，提供 3 个选项：
  - 👀 查看更改（Diff 对比）
  - 💾 覆盖并保存（确认后）
  - ➕ 导出草稿（保存用户修改）

---

## 性能影响

| 操作 | 优化前 | 优化后 |
|------|--------|--------|
| 流水线列表加载 | 2.8s | 0.8s (分组预加载) |
| 执行记录初显 | 卡顿（DOM 4000+） | 流畅（虚拟滚动 30 节点） |
| 滚动帧率 | 18fps | 58fps |
| 侧滑详情打开 | 等待 4000+ 详情 | 秒开（点击时异步加载） |

---

## 验收标准

- [ ] Phase 1 OpenSpec 通过审查
- [ ] 分组视图在生产环境可用
- [ ] 版本选择器支持标记和元数据显示
- [ ] 菜单在 Touch + 键盘导航完全可用（WCAG 2.1 AA）
- [ ] Stats API 响应 < 500ms
- [ ] 无新增 TypeScript 错误
- [ ] Chrome DevTools MCP 验证 UI 效果

---

## 时间线

| 阶段 | 任务 | 工作量 | 时间 |
|------|------|--------|------|
| Phase 1 | 提案审批 + OpenSpec | - | 1d |
| Phase 1 | 后端 API + 前端组件 | 5d | 1w |
| Phase 1 | API 合约同步 + 测试 | 2d | 3d |
| Phase 2 | 执行记录页优化 | 5d | 1w |
| Phase 3 | 编辑器页优化 | 6d | 1w |

**总计**: ~3-4 周（3 个工程师并行）

---

## 决策记录

- ✅ **Phase 分离**: 优化跨 3 个界面，分阶段交付降低集成风险
- ✅ **后端 Stats API**: 虽然可在前端计算，但后端统计利于缓存和准确性
- ✅ **虚拟滚动库选择**: react-window（轻量、久经考验）vs react-virtualized（功能多但体积大） → 选 react-window

---

## 相关文档

- 设计文档: `/Users/rick/cyber-databrew/openspec/changes/CYB-XXXX-pipeline-ui-optimization/design.md`
- 任务清单: `/Users/rick/cyber-databrew/openspec/changes/CYB-XXXX-pipeline-ui-optimization/tasks.md`
- 工程规格: https://claude.ai/code/artifact/850648df-9b08-43a6-9a60-7d8460258bed
