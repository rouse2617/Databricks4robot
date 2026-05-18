# Assets Discovery Component Tree And State Map

> 本文档是 `assets-discovery-ux-spec.md` 与 `assets-discovery-wireframes.md` 的工程化展开。
>
> 目标：把低保真 wireframe 进一步细化成可实现的组件树、状态域、事件流和页面状态图。
>
> 下一步实施计划：`assets-discovery-frontend-implementation-plan.md`

---

## 1. 文档目标

本文件解决的问题是：

- 线框图已经确定，但组件边界还不清晰
- 搜索、筛选、排序、选择、预览、URL 同步之间的状态归属还未定
- 多模态检索和资产预览未来要接入，但现在需要先确定扩展位

因此本文档输出三类结果：

1. 组件树
2. 状态域与状态拥有者
3. 页面状态图与关键事件流

---

## 2. 设计边界

本文档聚焦两个页面：

- `/assets`
- `/assets/:id` 中与 `Preview Hero` 强相关的顶部区域

不覆盖：

- Dashboard
- 交付创建向导
- 算法矩阵页
- MCAP 聚合页

---

## 3. 顶层工程原则

### 3.1 页面状态集中，展示组件尽量无状态

`AssetsPage` 不是简单表格页，而是工作台。

因此以下状态不应散落在多个组件内部：

- 当前查询
- facet 条件
- 列表排序
- 分页
- 当前选择
- 当前预览资产
- URL 同步状态
- Saved View

建议由页面级容器统一持有，再下发给展示组件。

### 3.2 区分 draft state 和 committed state

最容易混乱的是“用户正在输入”和“已经触发查询”。

要明确区分：

- `searchDraft`
- `queryState`

例如：

- 用户在输入框里键入 `wareh`
- 这只是 draft，不应立刻改变结果查询
- 当用户按回车、选中 suggestion 或点击 chip 时，才更新 committed query

### 3.3 区分数据状态和视图状态

数据状态示例：

- assets 结果
- preview summary
- facet counts

视图状态示例：

- facet 面板是否展开
- quick view 是否收起
- columns 菜单是否打开
- add filter popover 是否打开

二者不能混为一类。

### 3.4 路由是可恢复状态源，而不是唯一内存状态源

推荐策略：

- URL 保存“可恢复”状态
- 本地内存保存“瞬时交互”状态

可恢复状态：

- search mode
- committed query
- filters
- sort
- page
- view mode
- selected columns
- saved view id

瞬时状态：

- search suggestion open
- hovered row
- preview loading shimmer
- add filter dialog open

---

## 4. Desktop 组件树

### 4.1 `/assets` 页面完整组件树

```text
AssetsRoutePage
└── AssetsDiscoveryPage
    ├── AssetsPageHeader
    │   ├── AssetsTitleBlock
    │   ├── AssetsSearchBar
    │   │   ├── SearchModeSwitch
    │   │   ├── SearchInput
    │   │   ├── SearchHelpButton
    │   │   └── SearchSuggestionsPopover
    │   │       ├── SuggestionSectionFields
    │   │       ├── SuggestionSectionValues
    │   │       ├── SuggestionSectionRecent
    │   │       └── SuggestionSectionFutureModes
    │   ├── SavedViewSelector
    │   ├── AddFilterButton
    │   ├── SaveViewButton
    │   └── ExportButton
    ├── ActiveFilterChipsRow
    │   ├── FilterChipList
    │   └── ClearAllFiltersButton
    ├── AssetsWorkspaceLayout
    │   ├── AssetsFacetSidebar
    │   │   ├── FacetGroupBasic
    │   │   ├── FacetGroupCapture
    │   │   ├── FacetGroupAlgorithm
    │   │   ├── FacetGroupDelivery
    │   │   └── FacetGroupTags
    │   ├── AssetsResultsPane
    │   │   ├── ResultsToolbar
    │   │   │   ├── ResultsCount
    │   │   │   ├── SortSelector
    │   │   │   ├── ViewModeSwitch
    │   │   │   ├── ColumnsButton
    │   │   │   └── SelectAllResultsAction
    │   │   ├── BulkActionBar
    │   │   │   ├── BulkSelectionSummary
    │   │   │   ├── CreateDeliveryButton
    │   │   │   ├── RunAlgoButton
    │   │   │   ├── BatchTagButton
    │   │   │   ├── ExportIdsButton
    │   │   │   └── ClearSelectionButton
    │   │   ├── AssetsResultsView
    │   │   │   ├── AssetsTableView
    │   │   │   │   ├── AssetsTable
    │   │   │   │   ├── AssetsTableRow
    │   │   │   │   ├── AlgoSummaryCell
    │   │   │   │   ├── TagSummaryCell
    │   │   │   │   └── SelectionCheckboxCell
    │   │   │   └── AssetsCompactView
    │   │   ├── ResultsEmptyState
    │   │   ├── ResultsErrorState
    │   │   └── ResultsPagination
    │   └── AssetQuickPreviewPane
    │       ├── QuickPreviewEmptyState
    │       ├── QuickPreviewHeader
    │       ├── QuickPreviewMediaBlock
    │       ├── QuickPreviewSummaryBlock
    │       ├── QuickPreviewAlgoBlock
    │       ├── QuickPreviewFilesBlock
    │       ├── QuickPreviewEventsBlock
    │       └── QuickPreviewActions
    ├── AddFilterPopover
    │   ├── FilterFieldSelector
    │   ├── FilterOperatorSelector
    │   ├── FilterValueInput
    │   └── FilterQuickTemplates
    ├── ColumnsConfigPopover
    └── SaveViewDialog
```

### 4.2 `/assets/:id` Preview Hero 组件树

```text
AssetDetailRoutePage
└── AssetDetailPage
    ├── AssetDetailHeader
    ├── AssetPreviewHero
    │   ├── AssetPreviewMediaPanel
    │   │   ├── PreviewCanvasOrPlayer
    │   │   ├── PreviewToolbar
    │   │   │   ├── PlayButton
    │   │   │   ├── TimelineButton
    │   │   │   ├── OverlayButton
    │   │   │   ├── KeyframesButton
    │   │   │   └── FindSimilarButton
    │   │   └── PreviewStatusBanner
    │   └── AssetPreviewSummaryPanel
    │       ├── AssetMetaSummary
    │       ├── AssetTagSummary
    │       ├── AssetAlgoSummary
    │       ├── AssetDeliverySummary
    │       └── AssetFileSummary
    └── AssetDetailTabs
```

---

## 5. 组件分层模型

建议按 4 层组织组件。

### 5.1 Route Layer

负责：

- 路由参数
- URL query parse / serialize
- 页面级数据协调

组件：

- `AssetsRoutePage`
- `AssetDetailRoutePage`

### 5.2 Container Layer

负责：

- 状态持有
- API 调用
- 状态派发
- 组合子组件

组件：

- `AssetsDiscoveryPage`
- `AssetDetailPage`

### 5.3 Section Layer

负责：

- 页面分区逻辑
- 组合多个展示组件
- 派发局部事件

组件：

- `AssetsPageHeader`
- `AssetsFacetSidebar`
- `AssetsResultsPane`
- `AssetQuickPreviewPane`
- `AssetPreviewHero`

### 5.4 Presentational Layer

负责：

- UI 呈现
- 单一控件逻辑
- 尽量无业务副作用

组件：

- `SearchInput`
- `FilterChip`
- `SortSelector`
- `AssetsTable`
- `QuickPreviewMediaBlock`

---

## 6. 页面状态域总览

### 6.1 `/assets` 状态切片

建议页面状态切成 9 个 domain：

```text
AssetsDiscoveryState
├── routerState
├── searchUiState
├── queryState
├── facetUiState
├── resultsState
├── selectionState
├── previewState
├── savedViewState
└── layoutState
```

### 6.2 每个状态域职责

#### routerState

职责：

- 从 URL 读取可恢复状态
- 将 committed 状态同步回 URL

包含：

- `pathname`
- `queryString`
- `restoredAt`
- `isHydratedFromUrl`

#### searchUiState

职责：

- 管理搜索栏输入中的瞬时交互

包含：

- `draftText`
- `suggestionsOpen`
- `highlightedSuggestionIndex`
- `suggestionsLoading`
- `helpPopoverOpen`

#### queryState

职责：

- 管理真正参与结果查询的 committed 搜索条件

包含：

- `searchMode`
- `queryText`
- `parsedTokens`
- `activeFilters`
- `sort`
- `page`
- `pageSize`
- `viewMode`
- `selectedColumns`

#### facetUiState

职责：

- 管理左侧 facet 面板自身的展开/折叠与临时输入

包含：

- `expandedGroups`
- `rangeDrafts`
- `facetDrawerOpen`
- `addFilterPopoverOpen`

#### resultsState

职责：

- 管理结果集、加载态、错误态

包含：

- `items`
- `total`
- `fetchStatus`
- `error`
- `lastFetchedQueryKey`
- `isStale`

#### selectionState

职责：

- 管理当前页勾选和跨页选择逻辑

包含：

- `selectedAssetIds`
- `selectionMode`
- `allResultsSelected`
- `selectionCount`

#### previewState

职责：

- 管理当前右侧预览资产及其媒体/摘要状态

包含：

- `activeAssetId`
- `previewFetchStatus`
- `previewSummary`
- `previewManifest`
- `previewPanelCollapsed`
- `previewMode`

#### savedViewState

职责：

- 管理系统内置视图和用户自定义视图

包含：

- `currentViewId`
- `currentViewName`
- `views`
- `saveDialogOpen`
- `saveStatus`

#### layoutState

职责：

- 管理当前 viewport 与 pane 布局

包含：

- `viewport`
- `isTablet`
- `isMobile`
- `facetCollapsed`
- `previewCollapsed`

---

## 7. 详细状态结构建议

### 7.1 Search Mode

```ts
type SearchMode = "structured" | "keyword" | "semantic" | "similar";
```

说明：

- 第一阶段 UI 只露出 `structured` 与 `keyword`
- 但状态定义现在就保留 `semantic` 与 `similar`

### 7.2 Fetch Status

```ts
type FetchStatus = "idle" | "loading" | "success" | "error";
```

适用：

- `resultsState.fetchStatus`
- `previewState.previewFetchStatus`
- `savedViewState.saveStatus`

### 7.3 Preview Mode

```ts
type PreviewMode =
  | "none"
  | "placeholder"
  | "thumbnail"
  | "sprite"
  | "video_proxy"
  | "frame_gallery"
  | "multi_modal_bundle";
```

### 7.4 Selection Mode

```ts
type SelectionMode =
  | "none"
  | "explicit_rows"
  | "all_filtered_results";
```

### 7.5 Preview Availability

```ts
type PreviewAvailability =
  | "missing"
  | "processing"
  | "ready"
  | "failed";
```

---

## 8. 核心状态拥有者

### 8.1 组件与状态拥有关系

| 组件 | 持有状态 | 不应持有 |
|------|----------|----------|
| `AssetsRoutePage` | URL hydration 结果 | 结果列表本体 |
| `AssetsDiscoveryPage` | 页面级聚合状态 | 纯 UI hover |
| `AssetsSearchBar` | 输入框光标、局部 open/close | committed query |
| `AssetsFacetSidebar` | facet 组折叠、本地 range draft | 最终结果列表 |
| `AssetsResultsPane` | 无，接收 props 渲染 | 搜索草稿 |
| `AssetQuickPreviewPane` | 局部 tabs / local collapse | 结果查询条件 |
| `AssetPreviewHero` | 局部 player UI | 全局 search mode |

### 8.2 为什么 `AssetsResultsPane` 不应自己持有结果查询条件

因为以下触发都可能改变结果：

- 搜索栏提交
- chip 删除
- facet 点击
- saved view 切换
- `Find Similar` deep link

如果结果 pane 自己维护一份 query，会很快失控。

---

## 9. 页面状态图

### 9.1 `/assets` 总状态图

```text
┌─────────┐
│ booting │
└────┬────┘
     │ hydrate from URL / defaults
     v
┌──────────────┐
│ query_ready  │
└────┬─────────┘
     │ fetch results
     v
┌──────────────┐
│ loading_list │
└────┬────┬────┘
     │    │
     │    └───────────────┐
     v                    v
┌──────────────┐    ┌────────────┐
│ list_ready   │    │ list_error │
└────┬────┬────┘    └─────┬──────┘
     │    │               │ retry
     │    │               v
     │    └──────────> loading_list
     │
     ├─ row selected ──> preview_loading
     │
     ├─ query changed ─> loading_list
     │
     └─ no result ─────> list_empty
```

### 9.2 搜索栏状态图

```text
idle
  -> focus
typing
  -> request suggestions
suggesting
  -> select suggestion
commit_query
  -> update queryState
  -> loading_list

typing
  -> blur / escape
idle
```

### 9.3 结果预览状态图

```text
preview_none
  -> row click
preview_loading
  -> manifest missing
preview_placeholder
  -> manifest ready
preview_ready
  -> row click another
preview_loading
  -> fetch error
preview_error
```

### 9.4 选择状态图

```text
none_selected
  -> checkbox row
rows_selected
  -> select all filtered
all_filtered_selected
  -> clear selection
none_selected
```

### 9.5 Saved View 状态图

```text
view_idle
  -> choose saved view
apply_saved_view
  -> update queryState
loading_list

view_idle
  -> click save
save_dialog_open
  -> submit
save_pending
  -> success
view_idle
```

---

## 10. 关键事件流

### 10.1 搜索建议选中

```text
User selects suggestion
  -> AssetsSearchBar emits onCommitQuery(token)
  -> AssetsDiscoveryPage updates queryState
  -> queryState resets page=1
  -> routerState serializes URL
  -> resultsState fetch starts
```

### 10.2 点击 facet

```text
User toggles facet item
  -> AssetsFacetSidebar emits onFacetChange(nextFilters)
  -> AssetsDiscoveryPage updates queryState.activeFilters
  -> chips row updates
  -> URL updates
  -> results refetch
```

### 10.3 点击表格行

```text
User clicks row
  -> selection does not change by default
  -> previewState.activeAssetId = row.asset_id
  -> preview fetch starts
  -> right pane renders loading
  -> preview summary arrives
  -> pane renders media/status/summary
```

### 10.4 点击 `Select all filtered results`

```text
User clicks select all filtered
  -> selectionState.selectionMode = all_filtered_results
  -> selectionState.allResultsSelected = true
  -> bulk action bar updates summary
```

### 10.5 从详情页触发相似检索

```text
User clicks [Find Similar]
  -> navigate /assets?q=similar_to:asset:<id>&search_mode=similar
  -> AssetsRoutePage hydrates URL
  -> queryState.searchMode = similar
  -> results fetch starts
  -> results page renders similarity badge
```

---

## 11. 页面状态图与 API 调用关系

### 11.1 `/assets` 首屏所需最小 API

第一阶段：

- `GET /assets`
- `GET /assets/:id` 或 `GET /assets/:id/preview-summary`（未来）

第二阶段：

- `GET /assets/facet-counts`
- `GET /saved-views`
- `POST /saved-views`

未来多模态：

- `POST /search/semantic`
- `POST /search/similar`
- `GET /assets/:id/preview-manifest`

### 11.2 建议的查询键模型

```text
resultsQueryKey =
  searchMode +
  queryText +
  serializedActiveFilters +
  sort +
  page +
  pageSize +
  viewMode
```

```text
previewQueryKey =
  activeAssetId +
  previewMode
```

---

## 12. 组件职责明细

### 12.1 `AssetsSearchBar`

职责：

- 显示 mode switch
- 维护 draft input
- 请求 suggestions
- 提交 committed query

输入：

- `searchMode`
- `draftText`
- `committedQueryText`
- `suggestions`
- `loading`

输出事件：

- `onDraftChange`
- `onCommitQuery`
- `onModeChange`
- `onOpenHelp`

### 12.2 `AssetsFacetSidebar`

职责：

- 展示 facet groups
- 处理 group 折叠
- 处理 range draft
- 发出条件修改事件

输入：

- `activeFilters`
- `expandedGroups`
- `rangeDrafts`

输出事件：

- `onToggleFacet`
- `onApplyRangeFacet`
- `onResetFacetGroup`

### 12.3 `AssetsResultsPane`

职责：

- 纯粹展示当前结果和 toolbar
- 不拥有查询源状态

输入：

- `items`
- `total`
- `fetchStatus`
- `sort`
- `viewMode`
- `selectedAssetIds`

输出事件：

- `onRowClick`
- `onToggleRowSelection`
- `onChangePage`
- `onChangeSort`
- `onChangeColumns`

### 12.4 `AssetQuickPreviewPane`

职责：

- 展示当前选中资产的预览摘要
- 根据 preview manifest 切换占位态/静态态/媒体态

输入：

- `activeAssetId`
- `previewFetchStatus`
- `previewManifest`
- `previewSummary`

输出事件：

- `onOpenDetail`
- `onOpenFullPreview`
- `onFindSimilar`

### 12.5 `AssetPreviewHero`

职责：

- 成为资产详情页顶部的预览中枢
- 承接未来多模态预览与相似检索

输入：

- `asset`
- `previewManifest`
- `previewAvailability`

输出事件：

- `onPlay`
- `onOpenTimeline`
- `onToggleOverlay`
- `onFindSimilar`

---

## 13. Tablet / Mobile 组件差异

### 13.1 Tablet

保留相同状态模型，但视图结构变化：

- Facet 变成左抽屉
- Quick Preview 变成右抽屉
- 结果区仍为主区

### 13.2 Mobile

建议不做三栏状态分裂，改为：

- `facetDrawerState`
- `previewBottomSheetState`

但不要重写 queryState / selectionState / previewState 的核心模型。

也就是说：

- 布局可变
- 业务状态模型统一

---

## 14. 第一阶段实现建议

### 14.1 先落页面级 reducer / store 结构

建议先实现一个页面级 reducer，而不是先拆一堆组件。

原因：

- 这页的难点在状态协同，不在单个 UI 控件
- reducer 先定，组件拆分才不会反复回滚

### 14.2 第一阶段最低组件集

建议按以下顺序落地：

1. `AssetsDiscoveryPageState`
2. `AssetsSearchBar`
3. `ActiveFilterChipsRow`
4. `AssetsFacetSidebar`
5. `AssetsResultsPane`
6. `AssetQuickPreviewPane`
7. `AssetPreviewHero` placeholder

### 14.3 第一阶段可先占位的能力

- `semantic` mode 可只保留状态，不开放入口
- `similar` mode 可只接受 deep link，不给普通按钮
- `Open Full Preview` 按钮可先 disabled
- `Find Similar` 按钮可先 disabled

---

## 15. 与现有代码的映射建议

当前文件：

- [AssetsPage.tsx](/Users/hrp/cyber/cyber-databrew/Frontend/src/pages/AssetsPage.tsx:71)
- [AssetDetailPage.tsx](/Users/hrp/cyber/cyber-databrew/Frontend/src/pages/AssetDetailPage.tsx:56)

建议演进为：

```text
Frontend/src/pages/AssetsPage.tsx
  -> 只保留 route/container 角色

Frontend/src/components/assets/
  ├── AssetsSearchBar.tsx
  ├── ActiveFilterChipsRow.tsx
  ├── AssetsFacetSidebar.tsx
  ├── AssetsResultsPane.tsx
  ├── AssetQuickPreviewPane.tsx
  ├── AddFilterPopover.tsx
  └── ColumnsConfigPopover.tsx

Frontend/src/components/asset-detail/
  └── AssetPreviewHero.tsx

Frontend/src/hooks/assets/
  ├── useAssetsDiscoveryState.ts
  ├── useAssetsQuerySync.ts
  ├── useAssetPreview.ts
  └── useSavedViews.ts
```

---

## 16. 与文档体系的关系

- 交互与信息架构：见 [assets-discovery-ux-spec.md](/Users/rick/cyber-databrew/docs/archive/frontend/assets-discovery-ux-spec.md:1)
- 低保真页面线框：见 [assets-discovery-wireframes.md](/Users/rick/cyber-databrew/docs/archive/frontend/assets-discovery-wireframes.md:1)

如果要继续往实现推进，下一份文档最合适的是：

- `assets-discovery-frontend-implementation-plan.md`

它应进一步细化：

- reducer shape
- action list
- URL schema
- API contract
- 组件落地顺序
