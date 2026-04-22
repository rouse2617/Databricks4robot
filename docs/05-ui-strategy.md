# 05 · Web UI 策略

## 1. 核心思想

> **自研核心资产流 + iframe 嵌入专用工具**。
> 平台只做资产管理的"壳",**MCAP 播放器、标注、Pipeline、BI 全部嵌现成开源工具**。

---

## 2. 核心页面(7 个)

### 2.1 必做页面

| # | 页面 | 用户场景 | 自研 or 嵌入 |
| --- | --- | --- | --- |
| 1 | 首页 / Dashboard | 看整体数据流 / KPI | 自研 |
| 2 | 资产搜索页 | 按 tag / 关键词 / 时间 过滤 | 自研(参考 DataHub) |
| 3 | 资产详情页 | 看一个 asset 的所有 aspect | 自研(Tab 布局,参考 OpenMetadata) |
| 4 | Lineage 图 | 看一个 asset 的上下游 | 自研(React Flow) |
| 5 | QA 工作台 | QA 队列 / 审核 | 自研 + 嵌入 Foxglove |
| 6 | Pipeline / Run | 看 Dagster job 状态 | **iframe 嵌入 Dagster UI** |
| 7 | 分析 / SQL 查询 | 自助分析 | **iframe 嵌入 Superset**(Phase 2) |

### 2.2 资产详情页 Tabs(参考 OpenMetadata)

```
┌──────────────────────────────────────────────────────────┐
│  Asset: a1b2c3d4...                                      │
│  tenant: cyber-grace  |  status: active  |  v2           │
├──────────────────────────────────────────────────────────┤
│ [Overview] [Tags/QA] [Files] [Lineage] [Activity] [Preview MCAP] │
├──────────────────────────────────────────────────────────┤
│                                                          │
│  (Tab 内容,见下)                                        │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

每个 Tab 的内容:

| Tab | 内容 | 数据源 |
| --- | --- | --- |
| **Overview** | 核心字段、时间、创建者、状态 | `cf:core` + `cf:time` |
| **Tags/QA** | Tag 展示 + 编辑、QA 状态 + 审批 | `cf:tag` + `cf:qa` |
| **Files** | 原始 + 派生文件列表,下载 / 预览 | `cf:file` |
| **Lineage** | 上下游图可视化,Impact analysis | `cf:lineage` + 历史 query |
| **Activity** | 事件流(谁 / 什么时候 / 改了什么) | `cf:event` + `asset_events` |
| **Preview MCAP** | 嵌入 Foxglove 播放 | mcap-gateway signed URL → Foxglove App |

---

## 3. 嵌入策略(极重要)

### 3.1 Foxglove App(MCAP 播放)

**方式 A · 外链(首选,零维护)**:

```typescript
function openInFoxglove(assetId: string) {
  const signedUrl = await api.getMcapSignedUrl(assetId);  // 1h TTL
  const url = `https://app.foxglove.dev/?ds=remote-file&url=${encodeURIComponent(signedUrl)}`;
  window.open(url, "_blank");
}
```

**方式 B · 自托管 Lichtblick**(数据安全要求高时):

```bash
# docker-compose.yml
services:
  lichtblick:
    image: ghcr.io/lichtblick-suite/lichtblick:latest
    ports: ["8080:8080"]
```

然后 iframe 嵌入内部域名。

**方式 C · Foxglove SDK**(Phase 2,自研时间轴):

```typescript
import { McapReader } from "@foxglove/mcap";
// 自己写时间轴播放器,调用 SDK 解析
```

> 优先级:**A > B > C**。Phase 0/1 用 A 就够。

### 3.2 Dagster UI(Pipeline 运行)

```typescript
// 资产详情 Activity Tab,显示关联的 Dagster run
const dagsterRunId = asset.cfAlgo.sam2_v3.run_id;
const url = `${DAGSTER_BASE_URL}/runs/${dagsterRunId}`;

// 深链按钮
<a href={url} target="_blank">View in Dagster →</a>
```

如果想深度嵌入,用 iframe,但 Dagster 需要关闭 X-Frame-Options(有风险)。推荐**新标签深链**。

### 3.3 Label Studio / CVAT(标注)

```typescript
// QA 工作台里,"进入标注" 按钮
function startAnnotation(assetId: string) {
  // Label Studio API 创建任务
  const taskUrl = await api.createLabelStudioTask(assetId);
  window.open(taskUrl, "_blank");
}
```

### 3.4 Superset(BI)

```yaml
# Phase 2
superset:
  url: https://superset.grace.internal
  auth: SSO(OIDC)
  datasources:
    - BigQuery(grace.assets)
    - BigQuery(grace.events)
```

iframe 嵌到 "Analytics" 菜单项。

---

## 4. 技术栈

### 4.1 前端

| 层 | 技术 |
| --- | --- |
| 框架 | **React 18** + **TypeScript 5** |
| 构建 | Vite |
| UI 组件库 | **Ant Design 5**(中文友好,组件最全)或 Shadcn/ui(现代) |
| 状态管理 | **Zustand**(轻) + **TanStack Query**(数据获取) |
| 路由 | React Router v6 |
| Lineage 图 | **React Flow** |
| 时间轴 | **visx**(可选,Phase 2 自研播放器才用) |
| 表单 | React Hook Form + Zod |
| i18n | react-i18next(支持中英) |

### 4.2 API 层

```
前端 ──▶ data4cyber-api (BFF,Node.js/Go)
           │
           ├──▶ asset-service (gRPC)
           ├──▶ search-service
           └──▶ mcap-gateway
```

BFF 主要做:
- 跨服务聚合(一个 asset 详情可能要拼 4 个服务)
- 鉴权 / Session
- 缓存(Redis)
- GraphQL gateway(Phase 2)

### 4.3 部署

- GKE(与后端同集群)
- Nginx Ingress(TLS)
- Cloud CDN(前端资源)

---

## 5. 可借鉴的开源 UI 速查表

| 工具 | License | 抄什么 | Repo |
| --- | --- | --- | --- |
| **DataHub** | Apache 2.0 | 搜索页 / Lineage 交互 | [link](https://github.com/datahub-project/datahub) |
| **OpenMetadata** | Apache 2.0 | 详情 Tabs / Activity / Glossary | [link](https://github.com/open-metadata/OpenMetadata) |
| **Amundsen** | Apache 2.0 | 极简搜索首页 | [link](https://github.com/amundsen-io/amundsen) |
| **Dagster UI** | Apache 2.0 | Asset 图 | [link](https://github.com/dagster-io/dagster) |
| **Foxglove App** | 商业 + MPL(Studio 旧版) | MCAP 播放嵌入 | [link](https://app.foxglove.dev) |
| **Lichtblick** | MPL 2.0 | 自托管 MCAP 播放 | [link](https://github.com/lichtblick-suite/lichtblick) |
| **Rerun** | Apache 2.0 / MIT | 3D / 张量可视化 | [link](https://github.com/rerun-io/rerun) |
| **Label Studio** | Apache 2.0 | 标注嵌入 | [link](https://github.com/HumanSignal/label-studio) |
| **CVAT** | MIT | 视频标注嵌入 | [link](https://github.com/cvat-ai/cvat) |
| **Superset** | Apache 2.0 | BI 嵌入 | [link](https://github.com/apache/superset) |
| **LakeFS** | Apache 2.0 | Git-like 数据版本 UI(参考) | [link](https://github.com/treeverse/lakeFS) |

---

## 6. 页面线框(文字版 MVP)

### 6.1 搜索页

```
┌─────────────────────────────────────────────────────────┐
│ data4cyber    [Search: ____________________] [User]     │
├─────────────┬───────────────────────────────────────────┤
│ 🔍 Filters  │  5,231 assets found        Sort: [newest] │
│             │                                           │
│ Tenant      │  ┌─────────────────────────────────────┐  │
│ [x] grace   │  │ a1b2c3d4...  45.5s  Scene.urban     │  │
│ [ ] other   │  │ QA: approved  2026-04-22  alice     │  │
│             │  └─────────────────────────────────────┘  │
│ Tags        │  ┌─────────────────────────────────────┐  │
│ Scene       │  │ e2f3a4b5...  23.1s  Scene.highway   │  │
│  [x] urban  │  │ QA: pending  2026-04-22  bob        │  │
│  [ ] rural  │  └─────────────────────────────────────┘  │
│ Weather     │  ...                                      │
│  [x] rainy  │                                           │
│             │                                           │
│ Duration    │                                           │
│  > 30s [x]  │                                           │
│             │                                           │
│ QA State    │                                           │
│ [x] approved│                                           │
└─────────────┴───────────────────────────────────────────┘
```

### 6.2 详情页

```
┌─────────────────────────────────────────────────────────┐
│ ← Back       Asset: a1b2c3d4...               [Share]   │
│ tenant: grace  |  status: active  |  v2                 │
├─────────────────────────────────────────────────────────┤
│ [Overview] [Tags/QA] [Files] [Lineage] [Activity] [MCAP]│
├─────────────────────────────────────────────────────────┤
│                                                         │
│ Core                                                    │
│   Type: video_segment                                   │
│   Duration: 45.5s  (12.3s → 57.8s)                      │
│   Source MCAP: robot42_log.mcap (1.2GB)  [preview]      │
│                                                         │
│ Tags                                                    │
│   [Scene.urban] [Weather.rainy] [Vehicle.truck]         │
│                                                         │
│ Files                                                   │
│   • raw_mcap (original)       204MB  [stream] [download]│
│   • sam2_v3  v3.1.2           314MB  [stream] [download]│
│   • human_annotation v1       12KB   [open]             │
│                                                         │
│ QA: approved by bob@ on 2026-04-22                      │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### 6.3 Lineage 图

```
┌─────────────────────────────────────────────────────────┐
│ Lineage for a1b2c3d4...                                 │
│                                                         │
│   [robot42.mcap] ──split──▶ [a1b2c3d4] ──sam2_v3──▶ [sam2_mask.mcap]
│                                    │                    │
│                                    ├──annotate──▶ [ann.json]
│                                    │
│                                    └──▶ [dataset train_2026q1]
│                                                         │
│  ○ Expand upstream     ○ Expand downstream              │
└─────────────────────────────────────────────────────────┘
```

---

## 7. 路由结构

```
/                               → Dashboard
/search                         → 搜索页
/assets/:id                     → 详情页
/assets/:id/tags                → 同上,tab=tags
/assets/:id/lineage             → 同上,tab=lineage
/qa                             → QA 工作台
/qa/queue/:queueId              → 特定队列
/glossary                       → 业务术语
/glossary/:classification       → 分类详情
/admin/schemas                  → Schema 管理
/admin/users                    → 用户 / 权限
/dagster                        → iframe to Dagster
/foxglove                       → iframe to Foxglove(Phase 2)
/superset                       → iframe to Superset(Phase 2)
```

---

## 8. 开发路径

### Phase 0(Web UI MVP,2 个月)

- [ ] 搜索页(简版,无分面,只支持 tag 过滤)
- [ ] 详情页(Overview / Files / Activity 三个 Tab)
- [ ] Foxglove 外链播放
- [ ] Dagster 深链
- [ ] 基础登录(OIDC)

### Phase 1(3-4 个月)

- [ ] 分面过滤
- [ ] Lineage 图(React Flow)
- [ ] QA 工作台
- [ ] 嵌入 Label Studio
- [ ] Glossary / Tag 管理 UI

### Phase 2(Phase 2+)

- [ ] 多模态搜索(向量)
- [ ] Superset BI 嵌入
- [ ] 自研时间轴播放器(Foxglove SDK)
- [ ] 移动端适配

---

## 9. 下一步

- 详细 DataHub / OpenMetadata 参考 → [06-borrowed-patterns.md](06-borrowed-patterns.md)
- GCP 产品映射 → [07-tech-stack-gcp.md](07-tech-stack-gcp.md)
- ADR-005(UI 嵌入策略) → [adr/ADR-005-ui-embed-strategy.md](adr/ADR-005-ui-embed-strategy.md)
