# Round 2 — 全部任务拆解

## 任务全景

```
P0: 组件生态 ──────────────────── P1: 扒 Argo UI ──── P1: pro-flow ──── P2: 资产绑定 ──── P3: 模板
                                   │
 组件注册中心 ← 独立              OperationsMap ← 依赖 Phase 0 (已完成)
 组件搜索分类 ← 独立              filters     ← 依赖 Phase 0
 画布参数面板 ← 依赖注册中心       events/duration ← 纯 utils

依赖: 画布参数面板必须等注册中心出了才能联调。
其他全部无依赖，全并行。
```

## Task Breakdown

### Track A — 组件生态 (P0, ~3天)

#### A1: 组件注册页面
- 新建 `Frontend/src/pages/ComponentsPage.tsx`
- 路由 `/components`
- 表格展示已注册组件（名称、镜像、类型、描述）
- 新增/编辑弹窗（Formik/Yup form：name, image, type[container/script/resource/suspend], command[], args[], env[], description）
- 后端: `backend/internal/handler/pipeline_component/` — 已有 CRUD handler 但需要确认 API 是否完整
- 联动左侧菜单「组件管理」

#### A2: 组件搜索分类
- 组件面板加 `Input.Search` 过滤
- 按 type 分组显示（Container / Script / 自定义）
- 拖动到画布时自动填充模板参数

#### A3: 节点配置面板（画布参数编辑）
- 双击画布节点 → 弹出编辑 modal
- 字段: 命令、参数、环境变量、资源限制（CPU/内存）
- 根据节点类型展示不同表单（Pass Through 显示 command, Python Script 显示 source）
- 参考 visual-argo-workflows 的 `components/modals/template/Edit.tsx`

### Track B — 扒 Argo UI 组件 (P1, ~3天)

#### B1: WorkflowOperationsMap
- 从 `shared/workflow-operations-map.ts` 提取 7 个操作定义
- 适配成我们前端的 Ant Design Button 配置
- 挂在 workflow detail 页面 header 和 list row 上
- 后端 API 已就绪（Phase 0）

#### B2: WorkflowFilters（丰富过滤）
- Argo 有 6 个过滤维度：namespace, phase, labels, name, createdAfter, finishedBefore
- 我们已有 phase filter（status），加上：name search（A4 已完成）、labels、date range
- 用 Ant Design `Checkbox.Group` + `DatePicker.RangePicker` + `Input` + `Tag`
- URL 参数序列化（Argo 的做法）

#### B3: 其他 Argo utils
- `DurationPanel` — 持续时间格式化
- `Phase`/`PhaseIcon` — 状态图标
- `WorkflowLabels` — 标签显示组件
- `LinkifiedText` — URL 自动链接
- `PodName` 解析

### Track C — @ant-design/pro-flow (P1, ~1天)

#### C1: 安装集成
- `npm install @ant-design/pro-flow`
- 替换或增强现有 React Flow 画布
- 配置 5 种选择态样式

#### C2: 快捷键 + 右键菜单
- Cmd+Z 撤销 / Cmd+Shift+Z 重做
- Cmd+A 全选 / Cmd+C+V 复制粘贴
- 右键菜单：复制、删除、配置

#### C3: 内嵌属性面板
- 右侧 Drawer 属性面板
- 节点选中时自动展开

### Track D — 资产↔流水线绑定 (P2, ~2天)

#### D1: 资产详情页"创建流水线"
- 资产详情页加按钮「以此资产创建流水线」
- 跳转画布页并预填 `asset_ids`
- 后端已有 asset_ids 支持

#### D2: 流水线节点关联资产
- 节点侧面板显示关联资产信息（名称、类型、大小）
- 运行中显示资产处理进度

#### D3: 完成回跳
- 流水线完成后，从运行页可以跳回关联资产详情页

### Track E — 模板版本系统 (P3, ~2天)

#### E1: 模板列表页
- 新建 `/templates` 页面
- 列表展示已保存模板（名称、版本、节点数、更新时间）
- 从模板直接部署

#### E2: 版本对比
- 选择两个版本 → diff 展示（JSON diff）
- 回滚到指定版本

## Dependencies

```
A1 注册中心  ──→  A3 画布配置面板  ──→  (需要注册中心数据)
A1 注册中心  ──→  A2 搜索分类     ──→  (需要注册中心数据)
B1 OperationsMap ──→ Phase 0 API (已完成)
C1 pro-flow  ──→  独立，可随时装
D1 资产绑定  ──→  独立，资产详情页已有
E1 模板列表  ──→  后端模板数据已有

无依赖的可以全部并行：
  A1 ✗ A2(等A1) → 不能并行
  A1 ✗ C1 → 并行 ✅
  A1 ✗ B1 → 并行 ✅
  A1 ✗ B2 → 并行 ✅
  A1 ✗ D1 → 并行 ✅
  A1 ✗ E1 → 并行 ✅
  C1 ✗ B1 ✗ B2 ✗ D1 ✗ E1 → 全并行 ✅
```

## 并行策略

| Wave | Worktree | 任务 | 依赖 | 预估 |
|------|----------|------|------|------|
| **Wave 1** | `comp-registry` | A1 组件注册页面（后端+前端CRUD） | 无 | ~3h |
| | `argo-ops-map` | B1 OperationsMap + B3 utils | Phase 0 ✅ | ~2h |
| | `pro-flow-integrate` | C1 安装集成 + C2 快捷键 | 无 | ~2h |
| **Wave 2** | `comp-search-config` | A2 搜索 + A3 配置面板 | 等 A1 | ~2h |
| | `workflow-filters` | B2 丰富过滤 | Phase 0 ✅ | ~2h |
| | `asset-pipeline` | D1-D3 资产绑定 | 无 | ~2h |
| **Wave 3** | `template-version` | E1 模板列表 + E2 版本对比 | 无 | ~2h |
