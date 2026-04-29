# Frontend — data4cyber 数据平台

面向机器人 MCAP 视频数据的资产管理、算法处理监控、交付管理平台前端。

## Tech Stack

- React 19 + TypeScript
- Vite (构建 + HMR)
- Ant Design 5 (组件库)
- Tailwind CSS (工具类样式)
- dayjs (日期处理)

## 快速启动

```bash
npm install
npm run dev        # http://localhost:5173
```

构建生产版本：

```bash
npm run build      # 输出到 dist/
npm run preview    # 预览生产构建
```

## 环境配置

```bash
cp .env.example .env
```

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `VITE_API_BASE_URL` | 后端 API 地址 | `http://localhost:8080` |

## 页面结构

```
/login              → 登录页（Token 认证，Phase 0）
/dashboard          → 概览（KPI 卡片 + 算法状态分布 + 失败告警 + 最近更新）
/assets             → 资产管理（左侧 Facet 筛选面板 + 表格 + 批量操作栏）
/assets/:id         → 资产详情（多 Tab：概览 / 算法处理 / 标签 / 交付历史 / 文件）
/mcap-files         → MCAP 文件管理（Phase 2：按设备聚合视图）
/algo               → 算法处理（Phase 2：矩阵视图 行=Asset 列=算法 色块=状态）
/deliveries         → 交付管理（Phase 2：交付列表 + 创建向导 + 状态流转）
/analytics          → 数据分析（Phase 2：趋势图 / 饼图 / 排行榜）
/tags               → 标签字典（Phase 2：对接 tag_registry.yaml）
/settings           → 设置（当前会话信息）
```

## 目录结构

```
src/
├── api/                  # API 客户端
│   ├── client.ts         #   axios 实例，自动注入 X-Grace-Token
│   ├── assets.ts         #   资产 CRUD + 算法生命周期接口
│   └── types.ts          #   TypeScript 类型定义（对齐后端 model）
├── components/
│   └── AppLayout.tsx     # 全局布局：深色侧栏 + 内容区
├── hooks/
│   └── useAuth.ts        # Token 认证 hook
├── pages/
│   ├── DashboardPage.tsx     # 概览页
│   ├── AssetsPage.tsx        # 资产管理（Facet + 批量操作）
│   ├── AssetDetailPage.tsx   # 资产详情（5 Tab）
│   ├── McapFilesPage.tsx     # MCAP 文件（占位）
│   ├── AlgoProcessingPage.tsx# 算法处理（占位）
│   ├── DeliveriesPage.tsx    # 交付管理（占位）
│   ├── AnalyticsPage.tsx     # 数据分析（占位）
│   ├── TagDictionaryPage.tsx # 标签字典（占位）
│   ├── SettingsPage.tsx      # 设置
│   └── LoginPage.tsx         # 登录
├── App.tsx               # 路由定义（所有页面 lazy-load）
├── main.tsx              # 入口（Ant Design ConfigProvider + 中文 locale）
└── index.css             # 全局样式（SaaS 设计系统 + Ant Design 覆写）
```

## 设计系统

基于 ui-ux-pro-max skill 生成，风格为 SaaS Data-Dense Dashboard：

| 参数 | 值 |
|------|-----|
| Primary | `#2563EB` (Trust Blue) |
| Background | `#F8FAFC` |
| Sidebar | `#0F172A` (Slate 900) |
| Text | `#1E293B` |
| Success | `#16A34A` |
| Error | `#DC2626` |
| Warning | `#D97706` |
| 字体 | Inter (正文) + Fira Code (数据/代码) |
| 信息密度 | 8/10（紧凑表格，最大化数据可见性） |
| 动效强度 | 4/10（hover 过渡 200ms，无滚动动画） |

状态色块全局一致：
- `ok` / `approved` → 绿色
- `failed` / `rejected` → 红色
- `running` → 黄色
- `pending` → 灰色
- `blocked` → 灰色 + 锁图标

## 后端 API 依赖

前端对接 `/api/v1` 下的接口，当前已对接：

| 接口 | 前端使用位置 |
|------|-------------|
| `GET /assets?filter=...&sort_by=...` | AssetsPage (Facet 筛选) |
| `GET /assets/:id` | AssetDetailPage (概览 + 算法 Tab) |
| `PATCH /assets/:id` | AssetDetailPage (标签编辑) |
| `DELETE /assets/:id` | AssetsPage (批量删除) |
| `POST /assets/:id/algo/:key/reset` | AssetDetailPage (算法重置) |
| `GET /assets/:id/algo-events` | AssetDetailPage (状态变更时间线) |

尚需后端新建的接口（Phase 2）：

| 接口 | 用途 |
|------|------|
| `GET /stats/overview` | 概览页 KPI |
| `GET /stats/algo-distribution` | 概览页算法状态饼图 |
| `GET /algo-registry` | 算法处理矩阵列头 |
| `GET /deliveries` (全量列表) | 交付管理页 |
| `GET /mcap-files/by-device` | MCAP 按设备聚合 |

历史设计文档已归档到 `docs/archive/frontend/`，详见 `docs/archive/frontend/assets-discovery-ux-spec.md`、`docs/archive/frontend/assets-discovery-wireframes.md`、`docs/archive/frontend/assets-discovery-component-state-map.md`、`docs/archive/frontend/assets-discovery-frontend-implementation-plan.md` 与 `docs/archive/frontend/frontend-design-reference.md`。

## 设计参考

前端设计综合参考了 EmbodiFlow、RoboxStudio、Dagster、OpenMetadata、Airflow 五个平台，归档参考见 `docs/archive/frontend/frontend-design-reference.md`。
