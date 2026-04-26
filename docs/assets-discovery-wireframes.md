# Assets Discovery Wireframes

> 这是 `docs/assets-discovery-ux-spec.md` 的配套低保真 wireframe。
>
> 目标不是表达视觉风格，而是把页面结构、信息层级、关键交互位和未来扩展位画清楚。
>
> 对应的组件树与状态图：`docs/assets-discovery-component-state-map.md`
>
> 对应的实施计划：`docs/assets-discovery-frontend-implementation-plan.md`

---

## 1. 说明

本文件聚焦 5 个核心画面：

1. 资产检索页 Desktop 默认态
2. 资产检索页 Desktop 已筛选态
3. 搜索建议层与 Add Filter 弹层
4. 右侧 Quick View 预览态
5. 资产详情页 Preview Hero

补充：

- 平板 / 移动端折叠态
- 无预览态
- 未来多模态检索入口预留

---

## 2. 线框图例

```text
[Button]           可点击按钮
[Field__________]  输入框
( )                单选
[x]                多选/已选
────               分隔 / 区域边界
::::               次级辅助区 / 占位区
< >                下拉 / 切换
```

---

## 3. Screen A: Assets Discovery Desktop 默认态

### 3.1 结构总览

```text
┌────────────────────────────────────────────────────────────────────────────────────────────┐
│ 资产检索                                                                                    │
│ 搜索、过滤、预览并批量操作数据资产                                                           │
│                                                                                            │
│ [ Structured v ] [ Search Asset / MCAP / Owner / Tag / Algo...                  ] [?]     │
│ [ All Assets v ] [ Add Filter ] [ Save View ]                              [ Export ]      │
│                                                                                            │
│ Chips:  无                                                                                 │
├──────────────────────┬───────────────────────────────────────────────────┬─────────────────┤
│ Browse / Facets      │ Results                                           │ Quick View      │
│                      │                                                   │                 │
│ ▼ Basic              │ 12,847 results                                    │ No asset selected
│   [ ] approved       │ Sort: <Updated ↓>   View: <Table>   [Columns]    │                 │
│   [ ] rejected       │                                                   │ Select a row    │
│   [ ] archived       │ ┌───────────────────────────────────────────────┐ │ to preview      │
│                      │ │☐│ Asset ID │ MCAP │ Env │ Dur │ Algo │ ...   │ │                 │
│ ▼ Capture            │ ├───────────────────────────────────────────────┤ │ :::::::::::::   │
│   Env                │ │☐│ a1b2...  │ m1.. │ wh  │ 61s │ 3/1  │ ...   │ │ preview panel   │
│   Scene              │ │☐│ a1b3...  │ m2.. │ ofc │ 12s │ 4/0  │ ...   │ │ reserved        │
│   Duration           │ │☐│ a1b4...  │ m3.. │ out │ 98s │ 2/2  │ ...   │ │                 │
│                      │ │☐│ ...                                       │ │                 │
│ ▼ Algorithm          │ └───────────────────────────────────────────────┘ │                 │
│   Status             │                                                   │                 │
│   Algo Name          │ Pagination: < 1 2 3 ... >                         │                 │
│                      │                                                   │                 │
│ ▼ Delivery           │ No bulk actions                                   │                 │
│   Has delivery       │                                                   │                 │
│   Customer           │                                                   │                 │
│                      │                                                   │                 │
│ ▼ Tags               │                                                   │                 │
│   Priority           │                                                   │                 │
│   Quality            │                                                   │                 │
│   Notes              │                                                   │                 │
└──────────────────────┴───────────────────────────────────────────────────┴─────────────────┘
```

### 3.2 设计意图

- 顶部搜索栏是第一焦点。
- 左侧 facet 是持续可见的缩小范围工具。
- 右侧 Quick View 默认占位，不在空态时消失。
- 结果区不做卡片大图，默认表格。
- 预览区即使当前没有媒体能力，也先固定为 layout 中的一部分。

---

## 4. Screen B: Assets Discovery Desktop 已筛选态

### 4.1 典型筛选结果

```text
┌────────────────────────────────────────────────────────────────────────────────────────────┐
│ 资产检索                                                                                    │
│                                                                                            │
│ [ Structured v ] [ env:warehouse algo_status:failed updated_at>=2026-04-01      ] [?]    │
│ [ Warehouse Failed v ] [ Add Filter ] [ Save View ]                        [ Export ]     │
│                                                                                            │
│ Chips: [env = warehouse ×] [algo_status = failed ×] [updated_at >= 2026-04-01 ×] [Clear]
├──────────────────────┬───────────────────────────────────────────────────┬─────────────────┤
│ Browse / Facets      │ Results                                           │ Quick View      │
│                      │                                                   │                 │
│ ▼ Basic              │ 128 results                                       │ Asset a1b2c3d4  │
│   [x] approved       │ Sort: <Updated ↓>  View: <Table>   [Columns]     │ [approved]      │
│   [ ] rejected       │                                                   │ [priority:high] │
│                      │ Bulk: 12 selected [Create Delivery] [Run Algo]    │                 │
│ ▼ Capture            │                                                   │ [preview hero]  │
│   Env: warehouse     │ ┌───────────────────────────────────────────────┐ │ :::::::::::::   │
│   Scene: any         │ │☑│ Asset ID │ MCAP │ Env │ Dur │ Algo │ ...   │ │ first frame /   │
│   Duration: any      │ ├───────────────────────────────────────────────┤ │ placeholder     │
│                      │ │☑│ a1b2...  │ m1.. │ wh  │ 61s │ 3 ok /1 fail │ │                 │
│ ▼ Algorithm          │ │☑│ a1b5...  │ m8.. │ wh  │ 44s │ 2 ok /2 fail │ │ Preview status  │
│   [x] failed         │ │☐│ a1b9...  │ m9.. │ wh  │ 72s │ 1 ok /1 fail │ │ ready / missing │
│   Algo: any          │ │☑│ ...                                       │ │                 │
│                      │ └───────────────────────────────────────────────┘ │ Algo summary    │
│ ▼ Tags               │                                                   │ hand_tracking ✕ │
│   [x] priority=high  │ [Select all 128 results]                          │ body_tracking ✓ │
│                      │                                                   │ action blocked  │
│                      │                                                   │                 │
│                      │                                                   │ [Open Detail]   │
└──────────────────────┴───────────────────────────────────────────────────┴─────────────────┘
```

### 4.2 关键点

- 已生效条件始终可见，不只留在左侧。
- 批量操作栏升到结果区顶部，不放在视口底部。
- `Select all 128 results` 独立存在，支持跨页批量动作。
- 右侧预览可随着表格选中项切换。

---

## 5. Screen C: 搜索建议层

### 5.1 输入中态

```text
┌─────────────────────────────────────────────────────────────────────┐
│ [ Structured v ] [ wareh                                           ]│
└─────────────────────────────────────────────────────────────────────┘
                         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ Suggestions                                                         │
├─────────────────────────────────────────────────────────────────────┤
│ Fields                                                              │
│   env:warehouse                                                     │
│   scene:warehouse                                                   │
│                                                                     │
│ Values                                                              │
│   warehouse                                                         │
│   priority:high + env:warehouse                                     │
│                                                                     │
│ Recent                                                              │
│   env:warehouse algo_status:failed                                  │
│                                                                     │
│ Future modes                                                        │
│   Semantic: "warehouse 夜间失败较多的数据"                           │
└─────────────────────────────────────────────────────────────────────┘
```

### 5.2 行为说明

- 第一组是字段化建议。
- 第二组是候选值建议。
- 第三组是历史查询。
- 第四组是未来语义检索提示位，第一阶段可以隐藏或灰掉。

---

## 6. Screen D: Add Filter 弹层

### 6.1 基础态

```text
                         ┌───────────────────────────────┐
                         │ Add Filter                    │
                         ├───────────────────────────────┤
                         │ Field      [ env v ]          │
                         │ Operator   [ equals v ]       │
                         │ Value      [ warehouse____ ]  │
                         │                               │
                         │ Quick fields                  │
                         │ [status] [owner] [priority]   │
                         │ [algo_status] [duration]      │
                         │                               │
                         │                 [Cancel] [Add] │
                         └───────────────────────────────┘
```

### 6.2 数值范围态

```text
                         ┌─────────────────────────────────────┐
                         │ Add Filter                          │
                         ├─────────────────────────────────────┤
                         │ Field      [ duration v ]           │
                         │ Operator   [ between v ]            │
                         │ Min        [  30  ] sec             │
                         │ Max        [ 120  ] sec             │
                         │                                     │
                         │                 [Reset] [Cancel] [Add]
                         └─────────────────────────────────────┘
```

### 6.3 设计意图

- 给长尾筛选一个统一入口。
- 避免左侧 facet 把所有字段都摊开导致过长。
- 让业务用户不必写语法也能构造精细条件。

---

## 7. Screen E: Quick View 有预览态

### 7.1 列表右侧预览

```text
┌────────────────────────────────────────────┐
│ Asset a1b2c3d4                             │
│ [approved] [priority:high] [quality:good]  │
├────────────────────────────────────────────┤
│ Preview                                    │
│ ┌────────────────────────────────────────┐ │
│ │                                        │ │
│ │          preview frame / thumb         │ │
│ │                                        │ │
│ └────────────────────────────────────────┘ │
│ Status: ready                              │
│ Type: thumbnail                            │
│ Source: raw_mcap                           │
│ [Open Full Preview]                        │
├────────────────────────────────────────────┤
│ Summary                                    │
│ MCAP        mcap_20260401_0088             │
│ Env         warehouse                      │
│ Scene       indoor                         │
│ Duration    61.2s                          │
│ Owner       alice                          │
│ Reviewer    bob                            │
├────────────────────────────────────────────┤
│ Algo Summary                               │
│ hand_tracking       failed                 │
│ head_tracking       ok                     │
│ body_tracking       ok                     │
│ action_annotation   blocked                │
├────────────────────────────────────────────┤
│ Files                                      │
│ raw_mcap              yes                  │
│ preview_mp4           no                   │
│ thumbnail             yes                  │
├────────────────────────────────────────────┤
│ [Open Detail]                              │
└────────────────────────────────────────────┘
```

### 7.2 无预览态

```text
┌────────────────────────────────────────────┐
│ Asset a1b2c3d4                             │
├────────────────────────────────────────────┤
│ Preview                                    │
│ ┌────────────────────────────────────────┐ │
│ │            Preview unavailable         │ │
│ │                                        │ │
│ │ No thumbnail or preview artifact yet   │ │
│ └────────────────────────────────────────┘ │
│ Status: missing                            │
│ Reason: preview not generated              │
│ [Open Detail] [Request Preview]            │
└────────────────────────────────────────────┘
```

### 7.3 未来相似检索入口

```text
┌────────────────────────────────────────────┐
│ Preview                                    │
│ [ hero frame ]                             │
│ [Open Full Preview] [Find Similar Assets]  │
│ [Find Similar Segments]                    │
└────────────────────────────────────────────┘
```

---

## 8. Screen F: Asset Detail Preview Hero

### 8.1 详情页第一屏

```text
┌────────────────────────────────────────────────────────────────────────────┐
│ ← Back   Asset a1b2c3d4                                    [approved]     │
├───────────────────────────────────────────────┬────────────────────────────┤
│ Preview Hero                                  │ Asset Summary              │
│                                               │                            │
│ ┌───────────────────────────────────────────┐ │ Priority    high           │
│ │                                           │ │ Quality     good           │
│ │          Preview Player / Frame            │ │ Owner       alice          │
│ │                                           │ │ Reviewer    bob            │
│ └───────────────────────────────────────────┘ │ Duration    61.2s          │
│                                               │ Env         warehouse      │
│ [Play] [Timeline] [Overlay] [Keyframes]       │ Scene       indoor         │
│ [Find Similar] [Open Raw File]                │                            │
│                                               │ Algo: 2 ok / 1 fail /1 blk │
│                                               │ Delivery: not delivered    │
├───────────────────────────────────────────────┴────────────────────────────┤
│ [Overview] [Preview] [Algorithms] [Tags] [Deliveries] [Files]             │
├────────────────────────────────────────────────────────────────────────────┤
│ Tab content                                                               │
└────────────────────────────────────────────────────────────────────────────┘
```

### 8.2 说明

- 详情页第一屏必须把“媒体预览”和“结构化摘要”并列。
- `Preview` 不应埋在下方 tab 里作为次级内容。
- 即使当前只有缩略图，也应先占住这个版式。

---

## 9. Screen G: 多模态检索未来态

### 9.1 搜索模式切换

```text
┌────────────────────────────────────────────────────────────────────────────┐
│ [ Structured v ] [ Search Asset / MCAP / Owner / Tag...          ] [?]   │
└────────────────────────────────────────────────────────────────────────────┘

Modes:
  Structured
  Keyword
  Semantic
  Similar
```

### 9.2 Semantic 结果提示

```text
┌────────────────────────────────────────────────────────────────────────────┐
│ [ Semantic v ] [ warehouse 夜间失败较多的数据                    ] [?]   │
│ Result basis: semantic match | Re-ranked by metadata filters              │
│ Chips: [semantic query ×] [env = warehouse ×] [algo_status = failed ×]   │
└────────────────────────────────────────────────────────────────────────────┘
```

### 9.3 Similar 检索链路

```text
Asset Detail
  └─ click [Find Similar]
       └─ open /assets?q=similar_to:asset:a1b2c3&search_mode=similar
             └─ results show "Similarity basis" badge
```

---

## 10. Screen H: Tablet / Mobile

### 10.1 Tablet

```text
┌──────────────────────────────────────────────────────────────┐
│ [ Search______________________________________________ ]    │
│ [Filters] [Save View] [Columns]                             │
│ Chips: [env=warehouse] [failed]                             │
├──────────────────────────────────────────────────────────────┤
│ Results                                                     │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │ a1b2...  warehouse  61s  3/1                           │ │
│ │ a1b5...  warehouse  44s  2/2                           │ │
│ └──────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘

[Facet Drawer from left]
[Quick View Drawer from right]
```

### 10.2 Mobile

```text
┌──────────────────────────────────────────────┐
│ [ Search______________________________ ]     │
│ [Filter] [Sort] [View]                       │
│ Chips: [warehouse] [failed]                  │
├──────────────────────────────────────────────┤
│ Asset cards / compact rows                   │
│                                              │
│ a1b2...                                      │
│ warehouse · 61s · approved                   │
│ 3 ok / 1 fail                                │
│ [Preview] [Detail]                           │
│                                              │
└──────────────────────────────────────────────┘
```

### 10.3 移动端原则

- Facet 不常驻，收抽屉。
- Quick View 不使用右侧栏，改底部抽屉或直接详情页。
- `Preview` 按钮仍保留，以适配未来预览能力。

---

## 11. 关键交互流

### 11.1 从列表到预览

```text
Row click
  -> selected row
  -> right quick view updates
  -> optional lazy fetch preview summary
```

### 11.2 从预览到详情

```text
Quick View
  -> [Open Detail]
  -> /assets/:id
  -> Preview Hero at top
```

### 11.3 从预览到相似检索

```text
Quick View / Detail Preview
  -> [Find Similar]
  -> open assets discovery in Similar mode
```

---

## 12. 第一阶段实现所需最小 UI 壳子

即使当前没有多模态检索和真实预览，也建议先把这些壳子做出来：

- 搜索栏支持 `mode + q`
- filter chips 行
- 右侧 Quick View 容器
- 详情页 Preview Hero 占位区
- 预览状态字段：`ready / processing / missing / failed`
- `Open Full Preview` 和 `Find Similar` 按钮位，可先 disabled

这样后续接入：

- 缩略图
- preview mp4
- embedding 检索
- 相似检索

时不会再重构主布局。

---

## 13. 与 Spec 的对应关系

- 信息架构与搜索策略：见 [assets-discovery-ux-spec.md](/Users/hrp/cyber/Databricks4robot/docs/assets-discovery-ux-spec.md:210)
- 多模态搜索模式：见 [9.7 多模态检索的搜索模式预埋](/Users/hrp/cyber/Databricks4robot/docs/assets-discovery-ux-spec.md:417)
- 预览能力分层：见 [13.5 预览能力分层](/Users/hrp/cyber/Databricks4robot/docs/assets-discovery-ux-spec.md:756)
- 详情页 Preview Hero：见 [13.7 从结果页到详情页的预览路径](/Users/hrp/cyber/Databricks4robot/docs/assets-discovery-ux-spec.md:825)
