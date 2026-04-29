# Assets Discovery UX Spec

> 适用范围：`/assets` 资产检索与发现页，以及与其强相关的右侧预览、详情页跳转、Saved Views、URL 状态同步。
>
> 设计基底：`OpenMetadata` 的页面骨架 + `Labelbox` 的过滤交互 + `Kibana` 的高级搜索栏。
>
> 配套低保真线框：`assets-discovery-wireframes.md`
>
> 配套组件树与状态图：`assets-discovery-component-state-map.md`
>
> 对应前端实施计划：`assets-discovery-frontend-implementation-plan.md`

---

## 1. 文档目的

本规范解决的问题不是“把资产列表做漂亮”，而是把 `AssetsPage` 从一个普通 CRUD 列表升级成一个真正的**资产发现工作台**。

目标：

- 用户可以通过 `Asset ID / MCAP ID / Owner / Tag / Algo 状态 / 时间范围` 快速定位资产
- 用户可以在**不记语法**的情况下，通过 UI 过滤逐步收敛结果
- 熟练用户可以使用简洁但可组合的高级搜索语法，加速复杂检索
- 用户可以在结果页直接完成 80% 的判断，不必频繁跳转详情页
- 当前视图可以被 `复制 / 分享 / 保存 / 恢复`
- 当前设计需要为未来的**多模态检索**提前铺路，而不是把搜索能力锁死在字符串过滤上
- 每个资产未来进入详情页后，应有明确的**预览入口与预览区**，即使第一阶段只展示占位或静态缩略信息

不在本规范内的内容：

- 资产详情页的深度内容布局
- 算法处理矩阵页
- MCAP 文件聚合页
- 交付创建向导

---

## 2. 设计背景

当前 `/assets` 页已经有基础结构：

- 左侧 Facet 面板
- 中间表格
- 顶部搜索框
- 底部批量操作栏

但它还存在几个根本问题：

1. 搜索框语义不明确。UI 写的是“搜索...”，但底层实际上只在拼一个 tag 过滤条件。
2. 筛选能力真假混杂。部分 facet 只是 UI，占位但不真正参与查询。
3. 结果页缺少“轻量详情”。用户要判断一个资产是否值得操作，必须进入详情页。
4. 当前筛选条件不够可见。用户不容易理解“现在为什么只剩这些结果”。
5. 搜索、筛选、排序、分页、批量操作没有形成统一的状态模型。

因此这页需要从“表格页”转向“发现页”。

同时还要避免两个中长期风险：

1. 当前只围绕 metadata filter 设计，后续一旦接入 embedding / 相似搜索 / 文本描述检索，UI 容易推倒重来。
2. 当前资产点击后只有结构化详情，没有媒体预览设计位，后续补视频/图片/时间轴会破坏现有布局。

---

## 3. 产品定位

`AssetsPage` 的定位应当是：

**data4cyber 的主检索入口与资产操作中枢。**

它承担的核心任务不是展示数据，而是支持这四种高频工作：

- 定位资产：通过 ID、标签、环境、算法状态找到目标数据
- 比较资产：在结果列表中比较候选集合
- 初步判断：通过右侧预览做出“看 / 交付 / 重跑算法 / 标注”的决策
- 批量操作：在筛选结果之上执行交付、批量打标、批量算法操作

它还需要承担一个隐性职责：

- 为未来的**多模态资产发现**保留统一入口，让“结构化过滤”“关键词搜索”“语义检索”“相似资产召回”“预览驱动探索”最终可以收敛在同一工作台中

---

## 4. 用户角色与典型任务

### 4.1 用户角色

#### 数据运营 / PM

- 关注：资产规模、质量、标签完备度、交付准备情况
- 常见查询：
  - `warehouse` 场景近 7 天新增
  - `priority=high` 且 `未交付`
  - `quality=poor` 需要复核

#### 算法工程师

- 关注：算法失败、阻塞链路、产物缺失、时长异常
- 常见查询：
  - `hand_tracking failed`
  - `action_annotation blocked`
  - `duration > 120s` 且 `env=outdoor`

#### 交付 / 客户成功

- 关注：客户范围、可交付状态、最近交付历史、覆盖率
- 常见查询：
  - 某客户已交付的资产
  - 还未交付但已完成全部关键算法的资产
  - 最近 30 天所有高优先级可交付资产

#### 标注 / QA

- 关注：审核状态、标签缺失、备注检索
- 常见查询：
  - `reviewer=alice`
  - `status=rejected`
  - 备注中包含某个特殊问题描述

### 4.2 典型工作流

#### 工作流 A：快速找一个确定资产

1. 用户打开 `/assets`
2. 在搜索栏输入 `asset:a1b2c3`
3. 结果区立刻收敛到单条记录
4. 右侧预览显示算法、标签、文件、交付情况
5. 用户点击 `Open Full Detail`

#### 工作流 B：通过 UI 收缩到一个可操作集合

1. 用户先点左侧 `环境=warehouse`
2. 再点 `算法状态=failed`
3. 再点 `priority=high`
4. 结果区顶部出现 3 个 filter chips
5. 用户选中 12 条资产，执行“批量触发算法”

#### 工作流 C：高级查询

1. 用户输入 `env:warehouse algo_status:failed updated_at>=2026-04-01`
2. 系统把这串查询解析为结构化 chips
3. 用户再在左侧勾选 `reviewer=alice`
4. URL 自动更新，可复制给其他人

---

## 5. 设计原则

### 5.1 渐进增强

- 新手先用 UI facet，不要求记语法
- 熟练用户可以输入字段化查询
- 两套交互必须共用一套状态，不允许互相打架

### 5.2 结果可解释

- 当前所有筛选必须可见
- 每个筛选都能单独移除
- 用户必须知道当前结果是如何被收窄的

### 5.3 搜索是入口，筛选是过程，预览是决策支点

- 顶部搜索栏负责“进入目标范围”
- 左侧 facet 负责“逐步缩小”
- 右侧 preview 负责“减少跳页成本”

### 5.4 信息密度高，但不让用户迷路

- 中心区域默认用表格，不用大卡片流
- 将复杂信息压缩成摘要，但保证可展开
- 算法状态不要平铺 20 个 tag，把“摘要 + hover 明细”结合起来

### 5.5 URL 是真实状态源之一

- 页码、排序、搜索、facet、列显示、view 都应能序列化到 URL
- 用户刷新后应回到同一视图

---

## 6. 参考产品拆解

### 6.1 OpenMetadata：页面骨架

借鉴点：

- 顶部全局搜索
- 左侧 Faceted Search
- 中心结果区
- 右侧 Quick Preview / Summary

不照搬的点：

- 不采用过重的数据目录话术
- 不做过深的 entity taxonomy
- 不在第一版塞 lineage / glossary / domain 体系

### 6.2 Labelbox：过滤交互

借鉴点：

- `Add Filter` 心智
- 条件构建器
- filter chips
- 保存视图 / slice
- 结构化 metadata 过滤

不照搬的点：

- 不做强依赖 modal 的 filter builder
- 不把所有交互都藏进次级弹层

### 6.3 Kibana：高级搜索栏

借鉴点：

- query bar + pills 共存
- 字段建议 / 自动补全
- 支持表达式但不强迫表达式
- URL 可分享

不照搬的点：

- 不使用过于偏日志系统的 DSL
- 不暴露底层存储字段名给普通用户

---

## 7. 页面信息架构

### 7.1 桌面端总布局

```text
┌────────────────────────────────────────────────────────────────────────────┐
│ Assets Discovery                                                           │
│ [ Search assets, mcap, owner, tags...                         ] [Save View]│
│ [ All Assets v ]  [ Add Filter ]  [ Table ] [ Compact ]  [ Export ]        │
│ Chips: env:warehouse  algo_status:failed  tag.priority:high  updated:7d    │
├────────────────┬─────────────────────────────────────────────┬──────────────┤
│ Browse / Facet │ Results                                     │ Quick View   │
│                │                                             │              │
│ Basic          │ 123 results                                 │ Asset ID     │
│ Capture        │ Sort: Updated ↓   Columns   Bulk actions    │ Status       │
│ Algorithm      │ ------------------------------------------  │ MCAP         │
│ Delivery       │ asset_id | mcap | env | duration | algo...  │ Algo summary │
│ Tags           │ ...                                         │ Tags         │
│ Saved Views    │                                             │ Files        │
│                │ Pagination                                  │ Deliveries   │
│                │                                             │ Timeline     │
└────────────────┴─────────────────────────────────────────────┴──────────────┘
```

### 7.2 移动端 / 窄屏

移动端不保留三栏。

模式改为：

- 顶部搜索栏固定
- Facet 收到抽屉
- 结果区全宽
- 点击资产后进入底部抽屉预览或单独详情页

优先级：

- 桌面端优先
- 平板端保留 Facet 抽屉 + 右侧预览可折叠
- 手机端只保留搜索、筛选、结果、基础预览

---

## 8. 顶部区域设计

### 8.1 页面标题区

建议保留但简化：

- 主标题：`资产检索`
- 副标题：`搜索、过滤、预览并批量操作数据资产`

不建议在这里再放一堆 KPI。KPI 属于 Dashboard，不属于发现页。

### 8.2 搜索栏

搜索栏应为主视觉入口，宽度足够，支持自动补全。

建议结构：

- 左侧搜索图标
- 中间输入区域
- 右侧快捷帮助入口 `?`
- 回车提交
- 输入期间显示建议下拉

Placeholder 建议：

`搜索 Asset、MCAP、Owner、Tag，或输入 env:warehouse algo_status:failed`

不要再使用笼统的 `搜索...`。

### 8.3 Saved View 下拉

位于搜索栏附近，优先级高于“导出”。

默认内置视图：

- `全部资产`
- `最近更新`
- `高优先级`
- `算法失败`
- `待交付`
- `我审核的`

后续支持用户自定义保存。

### 8.4 Add Filter 入口

这是 Labelbox 风格的重要入口。

它不是替代左侧 facet，而是提供：

- 快速添加一个临时条件
- 对长尾字段做过滤
- 对时间范围 / 数值范围做更精细设置

点击后打开轻量浮层，支持：

- 选择字段
- 选择操作符
- 输入值
- 立即添加为 chip

---

## 9. 搜索栏语法设计

### 9.1 设计原则

- 只支持少量业务语法
- 使用业务字段名，不暴露 `cf_tag`、`cf_algo` 这类底层命名
- 可从自由文本自动升格为结构化条件

### 9.2 第一版支持字段

```text
asset:
mcap:
owner:
reviewer:
status:
env:
scene:
task:
batch:
tag.<key>:
algo:
algo_status:
has:
duration
created_at
updated_at
sort:
```

### 9.3 语法示例

```text
asset:a1b2c3
mcap:m_20260401_0088
owner:alice
reviewer:bob
status:approved
env:warehouse
scene:indoor
tag.priority:high
tag.notes:"night run"
algo:hand_tracking@1.2.0
algo_status:failed
duration>60
updated_at>=2026-04-01
has:delivery
sort:-updated_at
```

### 9.4 自由文本行为

当用户输入没有字段前缀的内容时：

- 若格式明显像 `asset_id`，优先按 `asset` 建议
- 若格式明显像 `mcap_file_id`，优先按 `mcap` 建议
- 否则默认走“跨高价值字段的模糊搜索”

跨字段模糊搜索建议第一版仅覆盖：

- `asset_id`
- `mcap_file_id`
- `owner`
- `reviewer`
- `tag.notes`
- `tag.batch`
- `tag.task`

不要在第一版宣称“全字段全文检索”。

### 9.5 自动补全

输入时提供三类建议：

- 字段建议：`owner:`、`env:`、`algo_status:`
- 值建议：`warehouse`、`approved`、`failed`
- 历史建议：用户最近用过的查询

### 9.6 解析结果的可视化

用户按下回车后，不应只保留原始字符串。

系统应将解析结果转成 chips：

- `env:warehouse`
- `algo_status:failed`
- `updated_at>=2026-04-01`

保留原始字符串只适合调试，不适合业务用户。

### 9.7 多模态检索的搜索模式预埋

当前版本虽然不实现完整多模态检索，但搜索栏的交互和状态模型必须允许未来扩展为多模式。

建议预留 4 类搜索模式：

#### A. Structured

面向今天的能力：

- `asset:`
- `mcap:`
- `owner:`
- `tag.*`
- `algo_status:`

这是第一阶段默认模式。

#### B. Keyword

面向跨字段关键字搜索：

- 在备注、任务名、批次、owner、reviewer 等字段做模糊匹配
- 仍然是“可解释”的文本检索，不依赖 embedding

建议在 UI 中和 Structured 共用一个输入框，不单独切模式。

#### C. Semantic

面向未来的文本语义检索：

- 用户输入一句自然语言，例如：
  - `抓取动作失败较多的 warehouse 夜间数据`
  - `室外场景里人体遮挡严重的片段`
- 后端通过 embedding / reranking 返回候选资产

为避免后续重做，建议现在就允许 query state 记录：

- `search_mode=semantic`
- `query_text`
- `semantic_scope`
- `semantic_threshold`

即使第一阶段先不启用 UI。

#### D. Similarity

面向未来的“找相似资产”：

- `similar_to:asset_id`
- `similar_to:file_id`
- `similar_to:preview_frame_id`

这个能力未来非常适合从预览区反向触发：

- 在详情页点“找相似资产”
- 在预览帧上点“查找相似片段”

因此 `AssetsPage` 的状态模型不应假设所有查询都来自文本框。

### 9.8 多模态搜索栏的未来形态

建议最终形态不是多个分散入口，而是**一个统一搜索栏 + 搜索模式辅助控件**：

```text
[ Structured v ] [ Search assets, tags, scenes, or describe what you want... ]
```

未来可扩展为：

- `Structured`
- `Keyword`
- `Semantic`
- `Similar`

第一阶段只露出 `Structured / Keyword`，其余模式保留状态与扩展位，不做入口污染。

---

## 10. Facet 面板设计

### 10.1 Facet 分组

左侧 facet 建议分 5 组。

#### A. 基础信息

- `status`
- `owner`
- `reviewer`
- `asset_id` 快速前缀匹配
- `mcap_file_id` 快速前缀匹配

#### B. 采集属性

- `env`
- `scene`
- `task`
- `batch`
- `duration`
- `created_at`
- `updated_at`

#### C. 算法属性

- `algo_status`
- `algo_key`
- `has_failed_algo`
- `has_blocked_algo`
- `has_missing_output`

#### D. 交付属性

- `has_delivery`
- `delivery_count`
- `last_delivered_at`
- `customer_id`

#### E. 标签属性

- `priority`
- `quality`
- `notes`
- 动态扩展 tag keys

### 10.2 交互规则

- 每组可折叠
- 每组顶部显示已命中项数量
- 选中条件后立刻生效
- 数值范围与时间范围使用 Apply / Reset，避免拖动中频繁查询

### 10.3 Facet 类型

不同字段使用不同组件：

- 枚举字段：Checkbox Group
- 高基数字段：Searchable Select
- 时间字段：Date Range Picker
- 数值字段：Min/Max 输入
- 布尔条件：Toggle / single checkbox

### 10.4 Facet 结果计数

第二阶段建议支持每个 facet 值显示命中计数，例如：

- `approved (128)`
- `failed (23)`

第一阶段可先不做，避免后端复杂度过高。

---

## 11. Filter Chips 设计

### 11.1 位置

chips 位于搜索栏和结果表格之间，永远可见。

### 11.2 内容

每个 chip 需要包含：

- 字段名
- 操作符
- 值
- 删除按钮

示例：

- `env = warehouse`
- `algo_status = failed`
- `duration > 60s`
- `tag.priority = high`

### 11.3 行为

- 点击 `x` 删除单个条件
- 支持 `Clear all`
- 支持在 chip 上 hover 看完整原始表达式
- 支持 pin/favorite 到 Saved View

### 11.4 来源透明

chip 需要保留来源信息：

- 来自搜索栏
- 来自左侧 facet
- 来自点击表格值

这不必默认展示，但在调试和事件埋点里应保留。

---

## 12. 结果区设计

### 12.1 默认模式

默认使用表格。

原因：

- 资产列表是高密度对比场景
- 需要批量勾选
- 需要看多列并排
- 需要排序和固定列

### 12.2 视图切换

支持两种结果视图：

- `Table`
- `Compact`

`Compact` 仍然基于列表，不建议改成 Pinterest 式卡片。

### 12.3 默认列

建议默认展示：

- `Asset ID`
- `MCAP`
- `Env / Scene`
- `Duration`
- `Status`
- `Algo Summary`
- `Priority / Quality`
- `Owner`
- `Updated At`

### 12.4 Algo Summary 设计

不要在表格中平铺所有算法。

建议摘要形式：

- `3 ok / 1 failed / 2 pending`
- 若存在失败，则突出显示失败算法名

Hover 后弹出完整矩阵：

```text
hand_tracking@1.2.0   failed
head_tracking@1.0.0   ok
body_tracking@1.0.0   ok
action_annotation     blocked
```

### 12.5 Tag 展示

只展示高价值 tag：

- `priority`
- `quality`
- 另加 1 到 2 个摘要 tag

不要直接把所有 tags 铺满整列。

### 12.6 排序

第一阶段排序字段：

- `updated_at`
- `created_at`
- `duration`
- `delivery_count`

默认：

- `updated_at DESC`

### 12.7 列配置

支持 `Columns` 菜单：

- 显示/隐藏列
- 恢复默认列

这对不同角色很有用。

---

## 13. Quick View 预览面板

### 13.1 目标

右侧预览的目标是减少跳页，不是替代详情页。

### 13.2 展示内容

建议分为 5 个块：

#### A. Header

- `Asset ID`
- `status`
- `priority / quality`
- 主操作按钮：`Open Full Detail`

#### B. 基础信息

- `mcap_file_id`
- `env`
- `scene`
- `task`
- `duration`
- `owner`
- `reviewer`

#### C. 算法摘要

- 成功/失败/阻塞计数
- 最近失败原因
- 是否存在缺失产物

#### D. 文件与交付

- `raw_mcap`
- 关键算法产物
- 最近交付时间
- 最近交付客户

#### E. 最近事件

- 最近 5 条 algo events / lifecycle events

### 13.3 交互

- 点击表格某行，右侧同步预览
- 上下键可切换选中行，预览跟随
- 新开页按钮进入 `/assets/:id`
- 面板可收起

### 13.4 性能策略

- 列表数据返回摘要即可
- 右侧预览首次打开时再请求更完整信息
- 预览数据可按 `asset_id` 做短时缓存

### 13.5 预览能力分层

预览不是单一能力，而应分层设计。

#### Layer 0：无预览

适用于当前系统还没有生成任何 preview artifact 的情况。

UI 仍然要给出稳定结构：

- `Preview unavailable`
- 原因：未生成 / 处理中 / 无可预览文件
- CTA：`查看详情` 或 `请求生成预览`

这能避免以后接入预览时重做整块布局。

#### Layer 1：静态预览

第一批可以支持：

- 缩略图
- 首帧图
- 关键帧拼图
- 文件摘要信息

这是最容易落地的最小预览版本。

#### Layer 2：轻量媒体预览

未来支持：

- MP4 / HLS 预览
- GIF 片段
- 低帧率采样回放
- 带时间轴的关键帧跳转

这层已经足以让用户在不下载原始 MCAP 的情况下完成大多数判断。

#### Layer 3：模态增强预览

未来更完整的能力：

- 多路视角切换
- 传感器 overlay
- 算法结果 overlay
- 标签 overlay
- 时间轴事件标记

这层适合在详情页中实现，不建议直接塞进列表页 Quick View。

### 13.6 预览区应展示什么

无论列表页 Quick View 还是详情页 Preview Hero，都建议统一一套预览信息结构：

- `preview status`
- `preview source`
- `preview type`
- `preview generated_at`
- `preview duration`
- `preview controls`

其中 `preview type` 可以是：

- `thumbnail`
- `sprite`
- `video_proxy`
- `frame_gallery`
- `multi_modal_bundle`

### 13.7 从结果页到详情页的预览路径

建议把预览体验拆成两层：

#### 结果页 Quick View

目标：

- 快速判断，不做深度消费

建议内容：

- 一张主缩略图 / 首帧
- 简短媒体摘要
- 最近失败算法
- `Open Full Preview` 按钮

#### 详情页 Preview Hero

目标：

- 成为每个资产详情页的第一屏核心区域

建议未来在 `/assets/:id` 顶部预留：

- 左侧：媒体预览
- 右侧：核心 metadata + 标签 + 算法摘要

可参考结构：

```text
┌───────────────────────────────┬────────────────────────────┐
│ Preview Player / Frame Panel  │ Asset Summary              │
│                               │ status / priority / owner  │
│ [play] [timeline] [overlay]   │ tags / algo summary        │
│                               │ deliveries / files         │
└───────────────────────────────┴────────────────────────────┘
```

即使当前还没有视频预览，也应在详情页结构设计里把这个区域作为保留位。

---

## 14. 批量操作栏

### 14.1 触发方式

当用户勾选一条及以上资产时，结果区顶部或底部出现批量操作栏。

建议放在结果区顶部，减少滚动依赖。

### 14.2 第一阶段支持的按钮

- `创建交付`
- `触发算法`
- `批量打 Tag`
- `导出 ID`
- `取消选择`

删除按钮应谨慎，不建议在第一版默认突出。

### 14.3 选择模型

建议支持两层选择：

- 仅当前页已选
- 选择全部筛选结果

这在大批量交付中非常关键。

### 14.4 风险提示

当筛选结果很多且用户执行批量操作时，明确提示：

- 当前页 20 条
- 当前筛选结果 1,284 条

防止误操作。

---

## 15. Saved Views 设计

### 15.1 Saved View 内容

一个 Saved View 至少保存：

- 搜索查询
- facet 条件
- 排序
- 列配置
- 结果视图模式

### 15.2 内置视图

- `全部资产`
- `最近更新`
- `高优先级`
- `算法失败`
- `待交付`
- `我的审核`

### 15.3 用户自定义视图

交互：

1. 用户配置好当前视图
2. 点击 `Save View`
3. 输入名称
4. 可选“设为默认”

### 15.4 分享

Saved View 与 URL 应兼容：

- 可以直接复制当前 URL
- 也可以分享已命名视图

---

## 16. URL 状态模型

建议统一使用 query string 作为可恢复状态载体。

示例：

```text
/assets?q=env:warehouse%20algo_status:failed&filter=status:eq:approved&filter=tag.priority:eq:high&sort=-updated_at&page=2&view=table
```

建议编码字段：

- `q`
- `filter` 可重复
- `sort`
- `page`
- `view`
- `columns`
- `saved_view`

原则：

- 用户刷新不丢状态
- 用户返回上一页能回到之前列表态
- URL 可直接粘贴给别人

---

## 17. 空态、异常态、无结果态

### 17.1 初始空态

适用于没有任何资产时：

- 说明平台还没有数据
- 提供 CTA：`上传 MCAP` 或 `查看导入流程`

### 17.2 无结果态

当筛选后没有结果：

- 显示当前生效条件摘要
- 提供按钮：
  - `清空筛选`
  - `回到全部资产`
  - `保存此查询`

### 17.3 加载态

- 搜索栏触发查询时显示轻量 loading
- 表格 skeleton 不要过重
- 右侧 preview 单独 loading，不阻塞主列表

### 17.4 错误态

区分两类：

- 查询失败：网络或后端错误
- 查询为空：正常业务结果

不要把“空结果”伪装成“失败”。

---

## 18. 可访问性与键盘交互

### 18.1 键盘

- `/` 聚焦搜索框
- `Enter` 提交搜索
- `Esc` 关闭建议下拉
- `↑ / ↓` 在建议项和结果行间导航
- `Space` 勾选当前行

### 18.2 对比度

- tag 状态颜色必须满足可读性
- 不依赖颜色单独传达状态
- `blocked` 必须配合图标或文案

### 18.3 屏幕阅读器

- 搜索栏要有明确 label
- filter chips 删除按钮要有 aria label
- 表格行选中态需要语义化反馈

---

## 19. 视觉设计语言

### 19.1 视觉基调

保留当前 `data4cyber` 已建立的 SaaS 密集信息风格：

- 深色侧栏
- 浅色内容背景
- 紧凑表格
- 低噪音阴影
- 少量强调色

### 19.2 强调层级

- 搜索栏：一级视觉焦点
- filter chips：二级焦点
- 表格行 hover：三级反馈
- 右侧 preview：信息承接区，不抢主区域注意力

### 19.3 状态色

统一沿用现有状态色：

- `ok / approved` 绿色
- `failed / rejected` 红色
- `running` 黄色
- `pending` 灰色
- `blocked` 灰色 + 锁

### 19.4 动效

- 输入建议浮层：100-150ms
- 表格 hover：150-200ms
- 右侧 preview 展开：180-220ms

不要做大面积花哨动画。

---

## 20. 前后端字段模型建议

### 20.1 不要暴露底层字段名

UI 层字段建议使用业务命名：

- `tag.notes`
- `algo_status`
- `has_delivery`

不要直接让普通用户看到：

- `cf_tag.notes`
- `cf_algo.hand_tracking@1.2.0.status`

### 20.2 后端 alias 映射

后端过滤层建议新增 alias：

- `tag.<key>` -> `cf_tag.<key>`
- `file.<key>` -> `cf_files.<key>`
- `algo.<key>.status` 或 `algo_status`
- `has:delivery` -> 业务派生条件

### 20.3 搜索字段白名单

后端应维护一个“允许 UI 搜索的字段字典”，避免前端拼任意路径。

第一阶段至少包括：

- `asset_id`
- `mcap_file_id`
- `owner`
- `reviewer`
- `status`
- `env`
- `scene`
- `task`
- `batch`
- `tag.priority`
- `tag.quality`
- `tag.notes`

### 20.4 多模态检索的底层字段预埋

如果未来希望接入多模态检索，建议现在就在资产模型或旁路索引层预留这些概念，而不是等 UI 做完再回头补。

建议至少抽象出以下实体：

- `preview_manifest`
- `preview_assets`
- `embedding_refs`
- `modality_summary`
- `search_index_status`

建议含义：

#### preview_manifest

记录一个资产当前有哪些可展示的 preview 资源。

示例：

```json
{
  "status": "ready",
  "hero_type": "video_proxy",
  "default_uri": "gs://.../preview.mp4",
  "thumbnail_uri": "gs://.../thumb.jpg",
  "sprite_uri": "gs://.../sprite.jpg",
  "frame_index_uri": "gs://.../frames.json"
}
```

#### preview_assets

不是业务上的“资产”，而是可展示资源集合，例如：

- thumbnail
- poster
- preview_mp4
- waveform
- gallery

#### embedding_refs

记录该资产有哪些 embedding 可用于检索。

示例：

```json
{
  "text_summary_v1": "emb://asset/a1/text_summary_v1",
  "keyframe_clip_v1": "emb://asset/a1/keyframe_clip_v1",
  "multimodal_scene_v1": "emb://asset/a1/multimodal_scene_v1"
}
```

#### modality_summary

记录该资产当前具备哪些模态内容与可用性：

- image frames
- video proxy
- text summary
- algo overlay
- sensor traces

#### search_index_status

记录：

- 是否已完成 keyword 索引
- 是否已完成 semantic embedding
- 是否允许 similar search

这样 UI 才能决定显示哪些检索能力，而不是盲目展示灰色按钮。

### 20.5 预览与检索的关系

预览不只是“展示”，它未来还会成为检索入口。

需要提前考虑的触发动作：

- 以当前资产为种子查相似资产
- 以当前关键帧查相似片段
- 以当前标签 / overlay 上下文查相关资产
- 从预览里截取时间窗口发起局部检索

因此预览区的组件设计不应是静态图片组件，而应抽象为可附加交互事件的预览容器。

---

## 21. 组件拆分建议

建议将 `AssetsPage` 拆为如下组件：

```text
AssetsPage
├── AssetsSearchBar
├── AssetsSavedViewBar
├── AssetsFilterChips
├── AssetsFacetPanel
│   ├── BasicFacetGroup
│   ├── CaptureFacetGroup
│   ├── AlgoFacetGroup
│   ├── DeliveryFacetGroup
│   └── TagFacetGroup
├── AssetsResultsToolbar
├── AssetsTable
├── AssetsBulkActionBar
└── AssetQuickPreviewDrawer
```

状态建议集中管理：

- `queryState`
- `facetState`
- `tableState`
- `selectionState`
- `previewState`

不要把每块状态散落在多个不相关 hook 里。

---

## 22. 埋点与评估指标

为验证这套设计是否真的提升效率，建议埋点以下指标：

- 搜索提交次数
- facet 使用次数
- query bar vs facet 的使用比例
- 平均从打开页面到打开目标资产详情的时间
- 无结果查询占比
- Saved View 使用率
- 批量操作触发率
- 从列表直接完成操作的比例

目标不是“用户点得更多”，而是：

- 更快找到资产
- 更少误筛选
- 更少无意义跳页

---

## 23. 分阶段实施建议

### Phase 1：基础可用

- 修正现有搜索字段和 filter 命名
- 补齐真正生效的 facet
- 引入 filter chips
- URL 同步
- 优化结果表格列定义

### Phase 2：发现体验成型

- 加 `Saved Views`
- 加 `Add Filter`
- 加右侧 Quick Preview
- 加列配置与结果模式切换

### Phase 3：高级检索

- 引入字段化 query bar
- 自动补全
- 自由文本到结构化条件的转换
- 跨字段模糊搜索

### Phase 4：高级资产运营

- 选择全部筛选结果
- 分享视图
- 推荐查询
- 个人最近使用视图

### Phase 5：多模态检索与预览增强

- 详情页 Preview Hero
- 静态预览资源接入
- 预览状态与 `preview_manifest`
- 语义检索状态模型接入
- `find similar` 交互入口
- embedding / semantic result 的结果标记与解释

---

## 24. 推荐落地结论

如果只能总结成一句话：

**把 `/assets` 做成“资产发现工作台”，而不是“带几个筛选框的表格页”。**

具体落地上：

- 布局抄 `OpenMetadata`
- 过滤交互学 `Labelbox`
- 高级搜索栏学 `Kibana`
- 业务字段和查询语法必须由 data4cyber 自己定义

第一阶段最值得优先投入的不是动画，不是美化，而是：

1. 统一搜索/筛选字段字典
2. 让所有条件都真正可见、可删、可分享
3. 增加右侧轻量预览，减少详情页跳转成本
4. 搜索状态模型不要只服务字符串过滤，要能容纳 semantic / similar search
5. 每个资产详情页都要预留预览位，哪怕第一阶段先显示 `Preview unavailable`

这几件做对了，整个资产检索体验就会从“能用”提升到“顺手”，并且不会在以后做多模态检索和资产预览时推倒重来。
