# Assets Discovery Frontend Implementation Plan

> 本文档将 `assets-discovery-ux-spec.md`、`assets-discovery-wireframes.md`、`assets-discovery-component-state-map.md` 转化为可执行的前端实施计划。
>
> 重点覆盖：
>
> - reducer shape
> - action 列表
> - URL schema
> - API contract
> - 组件落地顺序

---

## 1. 实施目标

本阶段不追求一次性实现完整的多模态检索和真实媒体预览，而是要把 `AssetsPage` 先升级为一个**结构正确、状态清晰、可持续扩展**的工作台。

第一阶段的工程目标：

1. 用统一 reducer 管理页面状态
2. 搜索、facet、chips、排序、分页、选择、预览统一进入同一状态模型
3. URL 能完整恢复当前视图
4. Quick View 先做结构化壳子，预留预览占位
5. 详情页顶部加 `Preview Hero placeholder`

中长期扩展目标：

- semantic search
- similar search
- saved views
- preview manifest
- preview media

---

## 2. 范围定义

### 2.1 本阶段包含

- `AssetsPage` 重构为 container + child components
- 页面级 reducer / hook
- URL parse / serialize
- 右侧 quick preview shell
- 详情页 preview hero placeholder
- assets API 适配层扩展

### 2.2 本阶段不包含

- 真正的 semantic / similar search 后端接入
- 真正的视频或图片预览媒体流
- Saved Views 后端持久化
- Facet counts API
- 批量动作真实业务完成

---

## 3. 实施策略

### 3.1 先重构状态，再重构 UI

错误顺序：

1. 先拆很多组件
2. 组件各自管理状态
3. 再尝试统一 URL / query / preview

正确顺序：

1. 先定义 reducer shape
2. 定义 actions
3. 定义 URL schema
4. 定义 selectors
5. 再拆组件

### 3.2 先建立“壳子能力”

以下能力本阶段允许只做壳子：

- `searchMode = semantic`
- `searchMode = similar`
- `Open Full Preview`
- `Find Similar`
- `Preview Hero`

只要状态、按钮位、组件边界先在，后面就能平滑接入。

---

## 4. 目录与文件规划

建议新增目录：

```text
Frontend/src/components/assets/
Frontend/src/components/asset-detail/
Frontend/src/hooks/assets/
Frontend/src/lib/assets/
```

建议文件：

```text
Frontend/src/components/assets/
├── AssetsSearchBar.tsx
├── ActiveFilterChipsRow.tsx
├── AssetsFacetSidebar.tsx
├── AssetsResultsPane.tsx
├── AssetsTableView.tsx
├── AssetsCompactView.tsx
├── AssetQuickPreviewPane.tsx
├── AddFilterPopover.tsx
├── ColumnsConfigPopover.tsx
└── BulkActionBar.tsx

Frontend/src/components/asset-detail/
└── AssetPreviewHero.tsx

Frontend/src/hooks/assets/
├── useAssetsDiscoveryReducer.ts
├── useAssetsQuerySync.ts
├── useAssetsResults.ts
├── useAssetPreview.ts
└── useAssetSelectors.ts

Frontend/src/lib/assets/
├── assetsDiscoveryTypes.ts
├── assetsDiscoveryReducer.ts
├── assetsDiscoveryActions.ts
├── assetsDiscoveryUrl.ts
├── assetsDiscoveryQuery.ts
└── assetsDiscoveryMappers.ts
```

---

## 5. Reducer Shape

### 5.1 顶层状态

建议状态定义如下：

```ts
export type SearchMode = "structured" | "keyword" | "semantic" | "similar";

export type FetchStatus = "idle" | "loading" | "success" | "error";

export type ViewMode = "table" | "compact";

export type SelectionMode =
  | "none"
  | "explicit_rows"
  | "all_filtered_results";

export type PreviewMode =
  | "none"
  | "placeholder"
  | "thumbnail"
  | "sprite"
  | "video_proxy"
  | "frame_gallery"
  | "multi_modal_bundle";

export type PreviewAvailability =
  | "missing"
  | "processing"
  | "ready"
  | "failed";

export interface AssetsDiscoveryState {
  routerState: RouterState;
  searchUiState: SearchUiState;
  queryState: QueryState;
  facetUiState: FacetUiState;
  resultsState: ResultsState;
  selectionState: SelectionState;
  previewState: PreviewState;
  savedViewState: SavedViewState;
  layoutState: LayoutState;
}
```

### 5.2 子状态定义

```ts
export interface RouterState {
  isHydratedFromUrl: boolean;
  lastSerializedQuery: string;
}

export interface SearchUiState {
  draftText: string;
  suggestionsOpen: boolean;
  suggestionsLoading: boolean;
  highlightedSuggestionIndex: number;
  helpPopoverOpen: boolean;
}

export interface QueryToken {
  id: string;
  kind: "text" | "field_filter" | "semantic" | "similar";
  field?: string;
  operator?: string;
  value: string;
  source: "search_bar" | "facet" | "chip" | "saved_view" | "deep_link";
}

export interface FilterChip {
  id: string;
  label: string;
  field: string;
  operator: string;
  value: string;
  source: QueryToken["source"];
}

export interface QueryState {
  searchMode: SearchMode;
  queryText: string;
  parsedTokens: QueryToken[];
  activeFilters: FilterChip[];
  sort: string;
  page: number;
  pageSize: number;
  viewMode: ViewMode;
  selectedColumns: string[];
}

export interface FacetUiState {
  expandedGroups: string[];
  rangeDrafts: Record<string, { min?: number; max?: number }>;
  facetDrawerOpen: boolean;
  addFilterPopoverOpen: boolean;
}

export interface ResultsState {
  items: Asset[];
  total: number;
  fetchStatus: FetchStatus;
  error: string | null;
  lastFetchedQueryKey: string;
  isStale: boolean;
}

export interface SelectionState {
  selectedAssetIds: string[];
  selectionMode: SelectionMode;
  allResultsSelected: boolean;
  selectionCount: number;
}

export interface AssetPreviewSummary {
  assetId: string;
  status: string;
  owner?: string;
  reviewer?: string;
  env?: string;
  scene?: string;
  durationSec?: number;
  algoSummary?: string;
  latestFailureReason?: string;
}

export interface PreviewManifest {
  availability: PreviewAvailability;
  mode: PreviewMode;
  thumbnailUri?: string;
  spriteUri?: string;
  previewVideoUri?: string;
  heroType?: string;
  generatedAt?: string;
  source?: string;
  reason?: string;
}

export interface PreviewState {
  activeAssetId: string | null;
  previewFetchStatus: FetchStatus;
  previewSummary: AssetPreviewSummary | null;
  previewManifest: PreviewManifest | null;
  previewPanelCollapsed: boolean;
  previewMode: PreviewMode;
}

export interface SavedViewItem {
  id: string;
  name: string;
  isSystem: boolean;
  queryStateSnapshot: Pick<
    QueryState,
    "searchMode" | "queryText" | "parsedTokens" | "activeFilters" | "sort" | "viewMode" | "selectedColumns"
  >;
}

export interface SavedViewState {
  currentViewId: string | null;
  currentViewName: string | null;
  views: SavedViewItem[];
  saveDialogOpen: boolean;
  saveStatus: FetchStatus;
}

export interface LayoutState {
  isTablet: boolean;
  isMobile: boolean;
  facetCollapsed: boolean;
  previewCollapsed: boolean;
}
```

### 5.3 初始状态建议

```ts
export const defaultAssetsDiscoveryState: AssetsDiscoveryState = {
  routerState: {
    isHydratedFromUrl: false,
    lastSerializedQuery: "",
  },
  searchUiState: {
    draftText: "",
    suggestionsOpen: false,
    suggestionsLoading: false,
    highlightedSuggestionIndex: -1,
    helpPopoverOpen: false,
  },
  queryState: {
    searchMode: "structured",
    queryText: "",
    parsedTokens: [],
    activeFilters: [],
    sort: "-updated_at",
    page: 1,
    pageSize: 20,
    viewMode: "table",
    selectedColumns: [
      "asset_id",
      "mcap_file_id",
      "env",
      "duration_sec",
      "status",
      "algo_summary",
      "priority_quality",
      "owner",
      "updated_at",
    ],
  },
  facetUiState: {
    expandedGroups: ["basic", "capture", "algorithm", "delivery", "tags"],
    rangeDrafts: {},
    facetDrawerOpen: false,
    addFilterPopoverOpen: false,
  },
  resultsState: {
    items: [],
    total: 0,
    fetchStatus: "idle",
    error: null,
    lastFetchedQueryKey: "",
    isStale: false,
  },
  selectionState: {
    selectedAssetIds: [],
    selectionMode: "none",
    allResultsSelected: false,
    selectionCount: 0,
  },
  previewState: {
    activeAssetId: null,
    previewFetchStatus: "idle",
    previewSummary: null,
    previewManifest: null,
    previewPanelCollapsed: false,
    previewMode: "none",
  },
  savedViewState: {
    currentViewId: "all-assets",
    currentViewName: "全部资产",
    views: [],
    saveDialogOpen: false,
    saveStatus: "idle",
  },
  layoutState: {
    isTablet: false,
    isMobile: false,
    facetCollapsed: false,
    previewCollapsed: false,
  },
};
```

---

## 6. Action 列表

建议 actions 分成 8 组。

### 6.1 Route / Hydration

```ts
type AssetsDiscoveryAction =
  | { type: "HYDRATE_FROM_URL"; payload: Partial<QueryState> }
  | { type: "MARK_URL_HYDRATED" }
  | { type: "SERIALIZE_TO_URL"; payload: { queryString: string } };
```

### 6.2 Search UI

```ts
type AssetsDiscoveryAction =
  | { type: "SET_SEARCH_DRAFT"; payload: { draftText: string } }
  | { type: "OPEN_SUGGESTIONS" }
  | { type: "CLOSE_SUGGESTIONS" }
  | { type: "SET_SUGGESTIONS_LOADING"; payload: { loading: boolean } }
  | { type: "SET_HIGHLIGHTED_SUGGESTION"; payload: { index: number } }
  | { type: "OPEN_SEARCH_HELP" }
  | { type: "CLOSE_SEARCH_HELP" };
```

### 6.3 Query Commit

```ts
type AssetsDiscoveryAction =
  | { type: "COMMIT_QUERY_TEXT"; payload: { queryText: string; searchMode?: SearchMode } }
  | { type: "COMMIT_PARSED_TOKENS"; payload: { tokens: QueryToken[] } }
  | { type: "SET_SEARCH_MODE"; payload: { searchMode: SearchMode } }
  | { type: "ADD_FILTER_CHIP"; payload: { chip: FilterChip } }
  | { type: "REMOVE_FILTER_CHIP"; payload: { chipId: string } }
  | { type: "CLEAR_ALL_FILTERS" }
  | { type: "SET_SORT"; payload: { sort: string } }
  | { type: "SET_PAGE"; payload: { page: number } }
  | { type: "SET_PAGE_SIZE"; payload: { pageSize: number } }
  | { type: "SET_VIEW_MODE"; payload: { viewMode: ViewMode } }
  | { type: "SET_SELECTED_COLUMNS"; payload: { selectedColumns: string[] } };
```

### 6.4 Facet UI

```ts
type AssetsDiscoveryAction =
  | { type: "TOGGLE_FACET_GROUP"; payload: { groupKey: string } }
  | { type: "SET_RANGE_DRAFT"; payload: { key: string; min?: number; max?: number } }
  | { type: "RESET_RANGE_DRAFT"; payload: { key: string } }
  | { type: "OPEN_FACET_DRAWER" }
  | { type: "CLOSE_FACET_DRAWER" }
  | { type: "OPEN_ADD_FILTER_POPOVER" }
  | { type: "CLOSE_ADD_FILTER_POPOVER" };
```

### 6.5 Results Fetch

```ts
type AssetsDiscoveryAction =
  | { type: "REQUEST_RESULTS" }
  | { type: "RECEIVE_RESULTS_SUCCESS"; payload: { items: Asset[]; total: number; queryKey: string } }
  | { type: "RECEIVE_RESULTS_ERROR"; payload: { error: string } }
  | { type: "MARK_RESULTS_STALE" };
```

### 6.6 Selection

```ts
type AssetsDiscoveryAction =
  | { type: "TOGGLE_ROW_SELECTION"; payload: { assetId: string } }
  | { type: "SET_ROW_SELECTION"; payload: { assetIds: string[] } }
  | { type: "SELECT_ALL_FILTERED_RESULTS"; payload: { total: number } }
  | { type: "CLEAR_SELECTION" };
```

### 6.7 Preview

```ts
type AssetsDiscoveryAction =
  | { type: "SET_ACTIVE_PREVIEW_ASSET"; payload: { assetId: string | null } }
  | { type: "REQUEST_PREVIEW" }
  | {
      type: "RECEIVE_PREVIEW_SUCCESS";
      payload: { summary: AssetPreviewSummary; manifest: PreviewManifest };
    }
  | { type: "RECEIVE_PREVIEW_ERROR"; payload: { error: string } }
  | { type: "TOGGLE_PREVIEW_PANEL" }
  | { type: "SET_PREVIEW_MODE"; payload: { previewMode: PreviewMode } };
```

### 6.8 Saved Views

```ts
type AssetsDiscoveryAction =
  | { type: "SET_SYSTEM_VIEWS"; payload: { views: SavedViewItem[] } }
  | { type: "APPLY_SAVED_VIEW"; payload: { view: SavedViewItem } }
  | { type: "OPEN_SAVE_VIEW_DIALOG" }
  | { type: "CLOSE_SAVE_VIEW_DIALOG" }
  | { type: "REQUEST_SAVE_VIEW" }
  | { type: "SAVE_VIEW_SUCCESS"; payload: { view: SavedViewItem } }
  | { type: "SAVE_VIEW_ERROR"; payload: { error: string } };
```

### 6.9 Layout

```ts
type AssetsDiscoveryAction =
  | { type: "SET_VIEWPORT"; payload: { isTablet: boolean; isMobile: boolean } }
  | { type: "SET_FACET_COLLAPSED"; payload: { collapsed: boolean } }
  | { type: "SET_PREVIEW_COLLAPSED"; payload: { collapsed: boolean } };
```

---

## 7. Reducer 规则

### 7.1 所有会改变结果查询的 action 都必须重置页码

以下动作必须自动 `page = 1`：

- `COMMIT_QUERY_TEXT`
- `COMMIT_PARSED_TOKENS`
- `ADD_FILTER_CHIP`
- `REMOVE_FILTER_CHIP`
- `CLEAR_ALL_FILTERS`
- `SET_SORT`
- `SET_SEARCH_MODE`
- `APPLY_SAVED_VIEW`

### 7.2 所有会改变 committed query 的 action 都必须触发结果 stale

通过 reducer 规则统一：

- `resultsState.isStale = true`

然后由 `useAssetsResults` 观察 queryKey 变化后发请求。

### 7.3 选择与预览分离

- 点击表格行：更新 `previewState.activeAssetId`
- 点击 checkbox：更新 `selectionState`

不要把“行点击”和“勾选选择”混成一个动作。

### 7.4 切换 Saved View 时覆盖 committed query，不覆盖 search UI 草稿

行为：

- `queryState` 用 saved view 快照替换
- `searchUiState.draftText` 更新为新的 `queryText`
- `selectionState` 清空
- `previewState` 不保留旧 asset

---

## 8. Selectors

建议在 `useAssetSelectors.ts` 中定义派生选择器。

### 8.1 基础 selectors

```ts
selectQueryKey(state)
selectHasActiveFilters(state)
selectSelectionCount(state)
selectIsAllFilteredSelected(state)
selectCanShowBulkBar(state)
selectCanOpenQuickPreview(state)
selectPreviewAvailability(state)
```

### 8.2 Query / URL selectors

```ts
selectSerializedQueryString(state)
selectFilterParams(state)
selectSearchPlaceholder(state)
selectVisibleColumns(state)
```

### 8.3 Preview selectors

```ts
selectPreviewHeroLabel(state)
selectPreviewStatusText(state)
selectCanFindSimilar(state)
selectCanOpenFullPreview(state)
```

---

## 9. URL Schema

### 9.1 URL 目标

URL 应满足：

- 刷新可恢复
- 回退可恢复
- 分享可复现
- 多模态检索未来可扩展

### 9.2 推荐字段

```text
/assets
  ?mode=structured
  &q=env:warehouse%20algo_status:failed
  &filter=status:eq:approved
  &filter=tag.priority:eq:high
  &sort=-updated_at
  &page=2
  &page_size=20
  &view=table
  &columns=asset_id,mcap_file_id,env,duration_sec,status,algo_summary,owner,updated_at
  &saved_view=warehouse-failed
  &preview=a1b2c3d4
```

### 9.3 字段说明

| 字段 | 必填 | 说明 |
|------|------|------|
| `mode` | 否 | `structured \| keyword \| semantic \| similar` |
| `q` | 否 | 搜索栏 committed query |
| `filter` | 否 | 可重复的结构化过滤条件 |
| `sort` | 否 | 排序，如 `-updated_at` |
| `page` | 否 | 页码 |
| `page_size` | 否 | 页大小 |
| `view` | 否 | `table \| compact` |
| `columns` | 否 | 逗号分隔列集合 |
| `saved_view` | 否 | 当前视图 id |
| `preview` | 否 | 当前右侧预览 asset id |

### 9.4 未来扩展字段

为 semantic / similar 提前保留：

| 字段 | 用途 |
|------|------|
| `semantic_scope` | 语义检索范围 |
| `semantic_threshold` | 语义阈值 |
| `similar_to` | 相似检索种子 |
| `similar_basis` | asset / frame / file |

示例：

```text
/assets?mode=similar&similar_to=a1b2c3d4&similar_basis=asset
```

```text
/assets?mode=semantic&q=warehouse%20夜间失败较多的数据&semantic_scope=assets
```

### 9.5 URL 序列化策略

建议规则：

- 空值不写入 URL
- 默认值不写入 URL
- `filter` 使用重复 key
- `columns` 使用逗号分隔
- `preview` 仅桌面端持久化，移动端不强依赖

---

## 10. API Contract

### 10.1 第一阶段复用现有 API

当前已有：

- `GET /api/v1/assets`
- `GET /api/v1/assets/:id`
- `GET /api/v1/assets/:id/algo-events`

基于现有前端封装：

- [assets.ts](/Users/hrp/cyber/Databricks4robot/Frontend/src/api/assets.ts:1)
- [types.ts](/Users/hrp/cyber/Databricks4robot/Frontend/src/api/types.ts:1)

### 10.2 第一阶段前端适配层建议

建议扩展 `assetsApi`：

```ts
export interface ListAssetsParams {
  q?: string;
  mode?: SearchMode;
  filter?: string[];
  sort_by?: string;
  page?: number;
  page_size?: number;
}
```

说明：

- 即使后端当前不吃 `q` 与 `mode`，前端 API 层也先接受
- 可以先由前端把 `q` 解析成 `filter[]`
- 后端升级后再平滑切换

### 10.3 新增前端接口封装建议

```ts
export interface AssetPreviewSummaryResponse {
  asset_id: string;
  status: string;
  owner?: string;
  reviewer?: string;
  env?: string;
  duration_sec?: number;
  algo_summary?: string;
  latest_failure_reason?: string;
  preview_manifest?: PreviewManifest;
}
```

建议新增：

```ts
assetsApi.getPreviewSummary = (id: string) =>
  assetsApi.get(id).then(mapAssetToPreviewSummary);
```

第一阶段 fallback：

- 直接复用 `GET /assets/:id`
- 在前端 mapper 层构造 `previewSummary` 和 `placeholder manifest`

### 10.4 第一阶段 Preview Manifest fallback 规则

如果当前后端没有 `preview_manifest`：

```ts
function buildPlaceholderPreviewManifest(asset: Asset): PreviewManifest {
  const hasThumbnail = Boolean(asset.files?.thumbnail);
  const hasPreviewVideo = Boolean(asset.files?.preview_mp4);

  if (hasPreviewVideo) {
    return {
      availability: "ready",
      mode: "video_proxy",
      previewVideoUri: asset.files.preview_mp4,
      heroType: "video_proxy",
      source: "asset.files.preview_mp4",
    };
  }

  if (hasThumbnail) {
    return {
      availability: "ready",
      mode: "thumbnail",
      thumbnailUri: asset.files.thumbnail,
      heroType: "thumbnail",
      source: "asset.files.thumbnail",
    };
  }

  return {
    availability: "missing",
    mode: "placeholder",
    reason: "preview not generated",
  };
}
```

### 10.5 第二阶段建议后端新增 API

建议新增：

- `GET /api/v1/assets/:id/preview-summary`
- `GET /api/v1/assets/:id/preview-manifest`

`preview-summary` 负责：

- Quick View 所需摘要
- 低成本、少字段、快返回

`preview-manifest` 负责：

- 预览资源清单
- 未来详情页 Preview Hero 的能力声明

### 10.6 Semantic / Similar API 预案

未来预案：

- `POST /api/v1/search/semantic`
- `POST /api/v1/search/similar`

前端请求体建议：

```ts
interface SemanticSearchRequest {
  query: string;
  filters?: string[];
  page?: number;
  page_size?: number;
}
```

```ts
interface SimilarSearchRequest {
  similar_to: string;
  basis: "asset" | "frame" | "file";
  filters?: string[];
  page?: number;
  page_size?: number;
}
```

结果建议统一回落为 `PaginatedResponse<Asset>`，避免前端 results pane 重写。

---

## 11. URL 与 API 之间的映射

### 11.1 第一阶段映射规则

| URL | 前端内部 | API |
|-----|----------|-----|
| `mode` | `queryState.searchMode` | 第一阶段前端内部使用 |
| `q` | `queryState.queryText` | 解析后转 `filter[]` |
| `filter` | `queryState.activeFilters` | `filter[]` |
| `sort` | `queryState.sort` | `sort_by` |
| `page` | `queryState.page` | `page` |
| `page_size` | `queryState.pageSize` | `page_size` |
| `preview` | `previewState.activeAssetId` | 触发 `getPreviewSummary(id)` |

### 11.2 第一阶段 `q -> filter[]` 策略

因为后端目前没有正式的 query DSL：

- `q` 主要用于 UI 还原
- 请求前把已解析的 token 变成 `filter[]`

示例：

```text
q = "env:warehouse algo_status:failed"
```

解析为：

```text
filter=status:eq:approved
filter=env:eq:warehouse
filter=algo_status:eq:failed
```

这里 `algo_status` 可能在第一阶段仍需前端特殊处理或暂不真实生效，取决于后端支持程度。

---

## 12. 自定义映射函数

建议新增 mapper 文件处理 UI 视图模型。

### 12.1 `Asset -> QuickPreviewSummary`

```ts
mapAssetToPreviewSummary(asset: Asset): AssetPreviewSummary
```

职责：

- 提取结构化摘要字段
- 计算 algo summary 文案
- 选取最近失败原因

### 12.2 `Asset -> TableRowViewModel`

```ts
mapAssetToTableRow(asset: Asset): AssetsTableRowViewModel
```

职责：

- 算法摘要
- 标签摘要
- 列显示兼容

### 12.3 `QueryState -> URLSearchParams`

```ts
serializeAssetsQueryState(state: QueryState): URLSearchParams
```

### 12.4 `URLSearchParams -> Partial<QueryState>`

```ts
parseAssetsQueryStateFromUrl(sp: URLSearchParams): Partial<QueryState>
```

---

## 13. 组件落地顺序

### 13.1 里程碑 1：状态底座

目标：

- 页面先跑在统一 reducer 上

任务：

1. 新建 `assetsDiscoveryTypes.ts`
2. 新建 `assetsDiscoveryActions.ts`
3. 新建 `assetsDiscoveryReducer.ts`
4. 新建 `useAssetsDiscoveryReducer.ts`
5. 新建 `assetsDiscoveryUrl.ts`
6. 在 `AssetsPage.tsx` 接入 reducer，但先不拆组件

完成标志：

- 页面功能与当前相当
- 但状态已迁入 reducer

### 13.2 里程碑 2：查询与 URL 同步

目标：

- 搜索、facet、分页、排序统一写入 URL

任务：

1. `useAssetsQuerySync.ts`
2. `parseAssetsQueryStateFromUrl`
3. `serializeAssetsQueryState`
4. 让 `page` / `sort` / `filters` 都从 queryState 走

完成标志：

- 刷新后页面恢复
- Back/Forward 行为正常

### 13.3 里程碑 3：组件拆分

目标：

- 从巨大的 `AssetsPage.tsx` 拆出稳定组件

任务顺序：

1. `AssetsSearchBar`
2. `ActiveFilterChipsRow`
3. `AssetsFacetSidebar`
4. `AssetsResultsPane`
5. `BulkActionBar`

完成标志：

- `AssetsPage.tsx` 只剩 container 逻辑

### 13.4 里程碑 4：Quick View 壳子

目标：

- 右侧 pane 成为真正的 preview shell

任务：

1. `useAssetPreview.ts`
2. `AssetQuickPreviewPane`
3. `buildPlaceholderPreviewManifest`
4. 支持 URL `preview=...`

完成标志：

- 点击行有右侧预览
- 无预览时也有占位态

### 13.5 里程碑 5：详情页 Preview Hero placeholder

目标：

- `/assets/:id` 顶部结构预留完成

任务：

1. 新建 `AssetPreviewHero.tsx`
2. 在 `AssetDetailPage.tsx` 顶部接入
3. 先走 placeholder manifest

完成标志：

- 详情页第一屏出现预览壳子

### 13.6 里程碑 6：Add Filter / Columns / Saved View 壳子

任务：

1. `AddFilterPopover`
2. `ColumnsConfigPopover`
3. `SaveViewDialog`

完成标志：

- 交互可用或半可用
- 状态结构完整

---

## 14. 每个里程碑的验收点

### 14.1 状态底座验收

- 页面行为不回退
- reducer 能完整驱动结果刷新
- 没有分裂状态

### 14.2 URL 验收

- 刷新可恢复
- 分享链接可恢复
- 浏览器回退正常

### 14.3 Quick View 验收

- 点击行即更新
- 切换行不会残留旧 preview
- placeholder / ready / error 三态明确

### 14.4 Preview Hero 验收

- 顶部布局固定
- 当前没有媒体也能稳定展示
- 后续接 preview manifest 不需要改 page 结构

---

## 15. 当前代码改造风险

### 15.1 `AssetsPage.tsx` 当前逻辑过于耦合

当前问题：

- 搜索、facet、分页、消息提示、表格列定义混在一起
- 局部 state 太多
- `useSearchParams` 仅部分使用

解决：

- 先 reducer
- 再组件拆分

### 15.2 现有后端 filter 能力不完全匹配 UI 目标

例如：

- `algo_status`
- 动态 tag alias
- future semantic / similar

前端第一阶段要接受“有些 UI 先是壳子”的现实。

### 15.3 预览数据当前缺失

这不是阻塞项。

前端应先通过 placeholder manifest 抽象掉这个问题。

---

## 16. 最终建议

如果只给一个落地建议：

**先把 `AssetsPage` 从“页面里堆逻辑”改成“页面级状态机 + 明确组件边界”，再去追求 richer UI。**

因为这页真正复杂的不是样式，而是：

- 搜索和筛选如何统一
- URL 和内存状态如何同步
- 选择和预览如何并存
- 未来多模态检索和资产预览如何不推倒重来

因此推荐下一步执行顺序就是：

1. 写 reducer
2. 接 URL schema
3. 拆组件
4. 上 Quick View shell
5. 上 Preview Hero placeholder
