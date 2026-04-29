# data4cyber 前端设计参考

> 综合 5 个竞品/开源项目的 UI 模式，提炼出适用于 data4cyber 的设计决策。
> 原始分析已归档，本文档为最终合并版。

---

## 一、调研来源

| 平台 | 定位 | 我们借鉴了什么 |
|------|------|---------------|
| EmbodiFlow (io-ai.tech) | 具身智能数据标注与管理 SaaS | 批量操作浮出栏、导出任务队列、字典管理、按设备聚合 |
| RoboxStudio (BAAI 智源) | 具身智能一站式平台 | 概览页 6 KPI + Tab 维度切换 + 排行榜 + 双饼图 |
| Dagster | 数据编排平台 (开源) | Command Center 首页、Asset Health 指示器、Saved Selection |
| OpenMetadata | 数据资产目录 (开源) | Faceted Search、Entity Detail 多 Tab、ReactFlow 血缘图 |
| Airflow | 工作流编排 (开源) | Grid View 状态矩阵（行=任务 列=时间 色块=状态） |

---

## 二、data4cyber 页面设计决策

### 概览页（Dashboard）

来源：RoboxStudio 的 KPI 卡片 + Dagster 的 Command Center

```
┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
│ 资产总量  │ │ MCAP文件 │ │ 算法成功率│ │ 本月交付  │
│ 12,847   │ │ 342      │ │ 89.2%    │ │ 23 批次  │
│ ↑12% 本周│ │ ↑5 本周  │ │ ↓2%      │ │ 1.2万条  │
└──────────┘ └──────────┘ └──────────┘ └──────────┘

[资产维度] [算法维度] [交付维度]    ← Tab 切换（学 RoboxStudio）

┌─────────────────────┐  ┌─────────────────────┐
│ 算法状态分布 (色块)   │  │ ⚠ 需要关注 (失败表) │
└─────────────────────┘  └─────────────────────┘

┌─────────────────────────────────────────────┐
│ 最近更新的资产 (表格，点击跳转详情)           │
└─────────────────────────────────────────────┘
```

设计决策：
- 4 个 KPI 而非 RoboxStudio 的 6 个——我们维度更少但每个更重要
- 每个 KPI 带趋势箭头（↑↓），比纯数字更有信息量
- "需要关注"表格直接可点击跳转——Dagster Command Center 的核心理念
- 算法状态分布是我们独有的（EmbodiFlow 和 RoboxStudio 都没有算法处理概念）

### 资产管理页（Assets）

来源：OpenMetadata 的 Facet 面板 + EmbodiFlow 的批量操作浮出栏

```
┌────────────┬─────────────────────────────────────────┐
│ Facet 面板  │  资产管理 (12,847)         [搜索] [刷新] │
│            │                                         │
│ ▼ QA 状态   │  ☐ │ ID │ MCAP │ 时长 │ 算法状态 │ Tags │
│ ☑ approved │  ☑ │ .. │ ..   │ 5.0s │ sam2:🟢  │ A   │
│ ☐ rejected │  ☐ │ .. │ ..   │ 12s  │ sam2:🔴  │ B   │
│            │                                         │
│ ▼ 环境      │  ── 已选 1 条 ──────────────────────── │
│ ▼ 算法状态   │  [创建交付] [触发算法] [批量Tag] [删除]  │
│ ▼ 时长范围   │                                         │
└────────────┴─────────────────────────────────────────┘
```

设计决策：
- 左侧 Facet 面板是核心差异化——EmbodiFlow 只有简单下拉筛选
- "算法状态"作为 Facet 维度——两个竞品都没有
- 表格里的"算法状态"列用多行色块展示每个算法状态，信息密度高
- 底部浮出操作栏学 EmbodiFlow——选中后出现批量操作按钮
- URL query string 驱动筛选状态，可分享可书签

### 资产详情页（Asset Detail）

来源：OpenMetadata 的 Entity Detail 多 Tab

```
← 返回   Asset a1b2c3d4...   [approved]

[概览] [算法处理] [标签] [交付历史] [文件]
```

5 个 Tab：
- 概览：Descriptions 展示所有 cf_meta 字段
- 算法处理：每个算法的状态/耗时/产物表格 + 重置按钮 + 状态变更时间线
- 标签：cf_tag 展示与编辑
- 交付历史：该 asset 交付给过哪些客户（对接 v_asset_delivery_history）
- 文件：cf_files 文件引用注册表

设计决策：
- 算法处理 Tab 是核心——展示状态机 + 依赖关系 + 可操作（重置/重试）
- 时间线组件展示状态变更历史（对接 algo-events API）

### 算法处理页（Phase 2）

来源：Airflow Grid View

```
            │ sam2@1.2 │ tracker@2.0 │ deblur@1.0 │ action@1.0│
────────────┼──────────┼─────────────┼────────────┼───────────│
a1b2c3d4... │   🟢     │    🔴       │    ⏳      │   🔒      │
c3d4e5f6... │   🟢     │    🟢       │    🟢      │   🟡      │

汇总: 🟢 45%  🔴 12%  🟡 8%  ⏳ 25%  🔒 10%
```

设计决策：
- 矩阵视图一眼看全局（Airflow 的 Grid View 模式）
- `blocked` 状态（🔒）展示依赖关系——所有竞品都没有这个
- 点击色块弹出 Popover 直接操作（重试/重置），不跳转页面

### 交付管理页（Phase 2）

来源：EmbodiFlow Export 的任务队列模式

设计决策：
- 交付 = 异步任务：创建 → 后台处理 → 完成后查看
- 创建交付用 Step 向导（选资产 → 选客户 → 确认 → 提交）
- 交付详情展示完整状态流转：pending → delivered → accepted/rejected → recalled
- 这是 EmbodiFlow 做不到的——它的"导出"是一次性操作，没有签收/拒收/召回

### MCAP 文件页（Phase 2）

来源：EmbodiFlow Robots 按型号聚合 + RoboxStudio 设备排行

设计决策：
- 默认按 device_id / camera_model 聚合展示
- 点击展开某设备的文件列表
- 顶部 KPI：文件总数、设备数、总大小、总时长

---

## 三、与竞品的差异化

| 能力 | EmbodiFlow | RoboxStudio | data4cyber |
|------|-----------|-------------|------------|
| 概览 Dashboard | 3 KPI + 质量饼图 | 6 KPI + 排行榜 | 4 KPI + 算法状态 + 失败告警 |
| 数据筛选 | 简单下拉 | 顶部筛选条 | Facet 面板 + algo 状态筛选 |
| 算法处理监控 | 无 | 无 | 矩阵视图 + 依赖图 + blocked |
| 交付生命周期 | 导出（一次性） | 无 | 完整状态机（签收/拒收/召回） |
| 标注界面 | 多面板视频 | 有 | 不需要（grace 系统完成） |
| 设备管理 | 按型号聚合 | 设备管理 | 按设备聚合 MCAP |
| 模型训练 | 私有部署 | 有 | Phase 2+（Dagster 编排） |

---

## 四、设计系统

通过 ui-ux-pro-max skill 生成，风格：SaaS Data-Dense Dashboard。

| 参数 | 值 | 说明 |
|------|-----|------|
| Primary | `#2563EB` | Trust Blue，SaaS 标准 |
| Sidebar | `#0F172A` | Slate 900，深色专业感 |
| Background | `#F8FAFC` | 浅灰，减少视觉疲劳 |
| Text | `#1E293B` | 高对比度正文 |
| Success | `#16A34A` | ok / approved |
| Error | `#DC2626` | failed / rejected |
| Warning | `#D97706` | running |
| 正文字体 | Inter | SaaS 行业标准 |
| 数据字体 | Fira Code | 等宽，表格数字对齐 |
| 信息密度 | 8/10 | 紧凑表格，最大化数据可见性 |
| 动效强度 | 4/10 | hover 200ms 过渡，无花哨动画 |
| 卡片圆角 | 8px | 现代但不过度圆润 |
| 阴影层级 | 3 级 | elevation-1/2/3，卡片 → hover → 弹窗 |

状态色块全局一致规则：
- 🟢 `ok` / `approved` / `success` → `#16A34A`
- 🔴 `failed` / `rejected` / `error` → `#DC2626`
- 🟡 `running` / `in-progress` → `#D97706`
- ⏳ `pending` → 灰色 `#94A3B8`
- 🔒 `blocked` → 灰色 + LockOutlined 图标

---

## 五、后端 API 需求

### 已有接口（可直接使用）

- `GET /api/v1/assets?filter=...&sort_by=...&page=...` — 资产列表（Facet 筛选）
- `GET /api/v1/assets/:id` — 资产详情（含 cf_algo / cf_tag）
- `PATCH /api/v1/assets/:id` — 更新标签/状态
- `POST /api/v1/assets/:id/algo/:key/start|finish|reset` — 算法生命周期
- `GET /api/v1/assets/:id/algo-events` — 状态变更事件
- `POST /api/v1/deliveries` — 创建交付（幂等）
- `GET /api/v1/deliveries/:id` — 交付详情
- `GET /api/v1/mcap-files` — MCAP 文件列表

### 需要新建的接口

| 优先级 | 接口 | 用途 |
|--------|------|------|
| P0 | `GET /algo-registry` | 算法矩阵列头 |
| P0 | `GET /deliveries` (全量列表) | 交付管理页 |
| P1 | `GET /stats/overview` | 概览页 KPI |
| P1 | `GET /stats/algo-distribution` | 概览页算法状态分布 |
| P2 | `GET /stats/daily-trend` | 概览页趋势图 |
| P2 | `GET /mcap-files/by-device` | MCAP 按设备聚合 |
| P2 | `GET /deliveries/:id/items` | 交付详情资产列表 |

---

## 六、实施路线

| 阶段 | 页面 | 状态 |
|------|------|------|
| Phase 1 (已完成) | AppLayout 侧栏 8 导航 | ✅ |
| Phase 1 (已完成) | 概览页（KPI + 算法状态 + 失败告警） | ✅ |
| Phase 1 (已完成) | 资产管理（Facet 面板 + 批量操作栏） | ✅ |
| Phase 1 (已完成) | 资产详情（5 Tab：概览/算法/标签/交付/文件） | ✅ |
| Phase 2 | 算法处理矩阵视图 | 占位 |
| Phase 2 | 交付管理（列表 + 创建向导 + 状态流转） | 占位 |
| Phase 2 | MCAP 文件（按设备聚合） | 占位 |
| Phase 2 | 数据分析（趋势图 / 饼图） | 占位 |
| Phase 2 | 标签字典（对接 tag_registry.yaml） | 占位 |
