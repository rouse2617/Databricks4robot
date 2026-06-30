# Phase 1 实施任务清单

## 后端任务

### 后端 API 开发

- [ ] `backend/internal/handlers/pipeline/stats.go` - 新增 stats handler
  - [ ] `GetPipelineStats(c *gin.Context)` - 获取用户 30 天统计
  - [ ] 返回 userStats + recommendations (含排分)
  - [ ] 单元测试：happy path + 无数据场景

- [ ] `backend/internal/repository/pipeline_repo.go` - 统计方法
  - [ ] `GetUserPipelineStats(userID, window)` - 从数据库查询点击/运行/访问时间
  - [ ] `CalcRecommendationScore(clickCount, runCount, lastAccessTime)` - 排分算法
  - [ ] 缓存机制（redis，TTL 5 min）

- [ ] `backend/internal/models/pipeline.go` - 新增数据结构
  - [ ] `PipelineStats` struct
  - [ ] `PipelineRecommendation` struct

- [ ] 数据库
  - [ ] 确保 `pipeline_access_logs` 表存在（用于追踪点击和访问时间）
  - [ ] 如需新增字段，创建 migration

### 版本冲突检测

- [ ] `backend/internal/handlers/pipeline/update.go` - 改进 UpdatePipeline handler
  - [ ] 接收 `baseVersion` 参数
  - [ ] 乐观锁检查：baseVersion 是否为最新
  - [ ] 冲突时返回 409 + 最新版本内容
  - [ ] 单元测试：冲突场景

- [ ] `backend/internal/postgres/pipeline_repo.go` - 版本检查
  - [ ] `IsLatestVersion(pipelineID, version)` 方法
  - [ ] `GetLatestVersion(pipelineID)` 方法
  - [ ] `SaveVersion(pipelineID, content, note)` - 保存新版本

### API 文档更新

- [ ] `api/openapi.yaml` - OpenAPI spec 更新
  - [ ] `GET /pipelines/stats` 端点
    - [ ] 请求: query param `window` (default: "30d")
    - [ ] 响应: `PipelineStats` schema
  - [ ] `PUT /pipelines/{id}` 改进
    - [ ] 新增 `baseVersion` 字段
    - [ ] 409 ConflictError schema

- [ ] `docs/review/api-guide.md` - 新增文档
  - [ ] `GET /pipelines/stats` 的 curl 示例
  - [ ] 请求/响应示例
  - [ ] 错误场景说明

### 测试和验证

- [ ] `scripts/api-guide-smoke.sh` - 新增 smoke test
  - [ ] Happy path: 获取用户的 stats，验证排分结果
  - [ ] Error path: window 参数无效，验证 400 响应
  - [ ] 版本冲突场景：baseVersion 过期，验证 409 响应

---

## 前端任务

### 新增/改进组件

- [ ] `Frontend/src/components/PipelineGroupView.tsx` - 新增分组视图组件
  - [ ] 三层分组：最常用、标记、全部
  - [ ] 从后端 stats API 拉取最常用列表
  - [ ] 支持搜索和筛选
  - [ ] 响应式设计（桌面/平板/手机）

- [ ] `Frontend/src/components/VersionSelector.tsx` - 改进版本选择器
  - [ ] 显示版本元数据：修改时间、摘要、状态
  - [ ] 标记分组（我的标记、关键版本、所有版本）
  - [ ] 搜索功能
  - [ ] 版本对比入口

- [ ] `Frontend/src/components/MenuButton.tsx` - 无障碍菜单
  - [ ] 按钮始终可见（PC 端 hover 半透明）
  - [ ] 点击展开菜单（不仅 Hover）
  - [ ] 完整 ARIA 属性（role, aria-expanded, aria-label）
  - [ ] 键盘导航：Tab、Shift+Tab、Esc
  - [ ] 焦点陷阱实现

- [ ] `Frontend/src/components/ComparisonPanel.tsx` - 版本对比面板
  - [ ] 并排显示两个版本的配置
  - [ ] 自动高亮差异项
  - [ ] "仅显示不同" 筛选选项

### Hooks 和 Utilities

- [ ] `Frontend/src/hooks/useOptimisticMutation.ts` - 乐观更新 hook
  - [ ] 支持立刻更新 UI，再异步同步
  - [ ] 失败时回滚
  - [ ] 加载态管理

- [ ] `Frontend/src/hooks/usePipelineStats.ts` - 统计数据 hook
  - [ ] 拉取后端 stats API
  - [ ] 缓存 5 分钟
  - [ ] 定时刷新

- [ ] `Frontend/src/lib/pipelineGrouping.ts` - 分组逻辑
  - [ ] `groupPipelines(list, stats)` - 根据 stats 对流水线分组
  - [ ] `calcRecommendationScore()` - 前端评分副本（可选）

### 页面改进

- [ ] `Frontend/src/pages/PipelineListPage.tsx` - 列表页改进
  - [ ] 集成 `<PipelineGroupView />`
  - [ ] 快速筛选栏改进
  - [ ] 卡片视图支持（可选）

- [ ] `Frontend/src/pages/PipelineEditorPage.tsx` - 编辑器改进
  - [ ] 版本选择器改进
  - [ ] 冲突检测弹框
  - [ ] 冲突时的用户选项（查看更改、覆盖、导出）

### API 客户端

- [ ] `Frontend/src/api/pipelineApi.ts` - API 方法新增
  - [ ] `getPipelineStats(window?: string)`
  - [ ] `updatePipelineWithVersion(id, content, baseVersion, note)`
  - [ ] 冲突响应处理

### 类型定义

- [ ] `Frontend/src/api/pipelineApi.ts` - 类型更新
  - [ ] `PipelineStats` interface
  - [ ] `PipelineRecommendation` interface
  - [ ] `ConflictError` interface

### 测试

- [ ] `Frontend/src/components/__tests__/MenuButton.test.tsx`
  - [ ] 点击展开/关闭
  - [ ] 键盘导航（Tab、Esc）
  - [ ] ARIA 属性检查
  - [ ] Touch 设备模拟

- [ ] `Frontend/src/components/__tests__/VersionSelector.test.tsx`
  - [ ] 显示版本元数据
  - [ ] 分组逻辑
  - [ ] 搜索功能

- [ ] `Frontend/src/pages/__tests__/PipelineListPage.test.tsx`
  - [ ] 分组视图渲染
  - [ ] 最常用列表加载

### 无障碍验证

- [ ] 使用 axe DevTools 扫描无障碍问题 (target: WCAG 2.1 AA)
- [ ] 屏幕阅读器测试（NVDA 或 JAWS）
- [ ] 键盘导航完整性检查

---

## SDK 任务

- [ ] `sdk/src/cyber_databrew_sdk/pipelines.py` - 新增方法
  - [ ] `get_stats(window="30d")` - 获取用户统计
  - [ ] `update_with_version(id, content, base_version, note)` - 带冲突检测的更新

- [ ] `sdk/src/cyber_databrew_sdk/__init__.py` - 导出更新
  - [ ] 确保新方法在公开 API 中

- [ ] `sdk/tests/unit/test_pipelines.py` - 单元测试
  - [ ] `test_get_stats()` - happy path
  - [ ] `test_update_with_version_conflict()` - 冲突场景
  - [ ] 使用 mock HTTP 测试

---

## 验证与部署任务

- [ ] **本地验证**
  - [ ] 后端：`source scripts/dev-backend-env.sh && go test ./...`
  - [ ] 前端：`npm run build && npm run test`
  - [ ] 类型检查：`npm run type-check`（0 errors）

- [ ] **部署前验证** (按 deploy-before-commit.md)
  - [ ] 本地部署脚本：`bash deploy/cloudrun/backend-dev.sh`
  - [ ] 执行 smoke test：`bash scripts/api-guide-smoke.sh`
  - [ ] 前端部署：`bash deploy/cloudrun/frontend-dev.sh`
  - [ ] Chrome DevTools MCP 测试：分组视图、版本选择、菜单交互

- [ ] **部署验证** (按 deploy-verification.md)
  - [ ] 检查后端日志（Cloud Run logs）
  - [ ] 检查前端加载（Network tab，无 5XX）
  - [ ] 手工测试关键流程

- [ ] **PR 检查清单**
  - [ ] 填完 PR template（包含哪些 API 合约文件被更新）
  - [ ] TypeScript 零错误
  - [ ] 所有测试通过
  - [ ] API 合约同步完毕

---

## 任务依赖关系

```
后端 API 开发 ──→ 前端 API 客户端
      ↓
  smoke test ──→ 部署前验证
      ↓
   部署验证 ──→ Chrome DevTools 测试

前端组件开发 ──→ 前端测试 ──→ 集成测试

SDK 更新 ──→ SDK 测试 ──→ 随 PR 提交
```

---

## 优先级和关键路径

**Critical Path** (关键路径，不能延期):
1. 后端 stats API
2. 版本冲突检测（PUT 更新）
3. 前端版本选择器改进
4. API 合约同步

**High Priority**:
5. 菜单 a11y 改进
6. 分组视图

**Nice-to-Have**:
7. 版本对比面板
8. SDK 更新（可 T+1 交付）

---

## 预计工作量

| 任务 | 工程师 | 天数 |
|------|--------|------|
| 后端 API + 测试 | 1 | 2.5d |
| 前端组件 + 测试 | 1 | 3d |
| API 合约同步 | Shared | 1d |
| 部署和验证 | 1 | 1d |
| SDK 更新 | 1 | 0.5d |
| **总计** | 2-3 | **8 天** |

*假设并行工作*
