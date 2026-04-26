// ─── Assets Discovery Workbench — Type Definitions ───
// All 9 state domains + supporting types for the discovery workbench.
// Validates: Requirements R13

import type { Asset } from "../../api/types";

// ─── Union Types ───

export type SearchMode = "structured" | "keyword" | "semantic" | "similar";

export type FetchStatus = "idle" | "loading" | "success" | "error";

export type ViewMode = "table" | "compact";

export type SelectionMode = "none" | "explicit_rows" | "all_filtered_results";

export type PreviewMode = "none" | "thumbnail" | "video" | "sprite";

export type PreviewAvailability = "missing" | "processing" | "ready" | "failed";

// ─── Token / Chip Interfaces ───

export interface QueryToken {
  id: string;
  field: string;
  op: string;
  value: string | string[];
  source: "search" | "facet" | "add_filter" | "saved_view";
}

/** FilterChip is structurally identical to QueryToken — kept as a named alias for clarity in UI code. */
export interface FilterChip {
  id: string;
  field: string;
  op: string;
  value: string | string[];
  source: "search" | "facet" | "add_filter" | "saved_view";
}

// ─── Preview / Saved-View Supporting Interfaces ───

/** Subset of Asset used in the Quick View preview pane. */
export type AssetPreviewSummary = Asset;

export interface PreviewManifest {
  thumbnailUrl: string | null;
  previewVideoUrl: string | null;
  availability: PreviewAvailability;
  mode: PreviewMode;
}

/** Alias kept for readability — structurally identical to SavedView. */
export interface SavedViewItem {
  id: string;
  name: string;
  builtin: boolean;
  queryState: Partial<QueryState>;
}


// ─── State Domain 1: RouterState ───

export interface RouterState {
  urlHydrated: boolean;
  currentPath: string;
}

// ─── State Domain 2: SearchUiState ───

export interface SearchUiState {
  draftText: string;
  suggestionsOpen: boolean;
  highlightedIndex: number;
  helpOpen: boolean;
}

// ─── State Domain 3: QueryState ───

export interface QueryState {
  searchMode: SearchMode;
  queryText: string;
  activeFilters: FilterChip[];
  sort: string;
  page: number;
  pageSize: number;
  viewMode: ViewMode;
  selectedColumns: string[];
}

// ─── State Domain 4: FacetUiState ───

export interface FacetUiState {
  expandedGroups: string[];
  rangeDrafts: Record<string, { min?: number; max?: number }>;
  dateDrafts: Record<string, { start?: string; end?: string }>;
  addFilterOpen: boolean;
}

// ─── State Domain 5: ResultsState ───

export interface ResultsState {
  items: Asset[];
  total: number;
  totalApprox: boolean;
  fetchStatus: FetchStatus;
  error: string | null;
  isStale: boolean;
}

// ─── State Domain 6: SelectionState ───

export interface SelectionState {
  selectedIds: Set<string>;
  mode: SelectionMode;
}

// ─── State Domain 7: PreviewState ───

export interface PreviewState {
  activeAssetId: string | null;
  fetchStatus: FetchStatus;
  summary: AssetPreviewSummary | null;
  availability: PreviewAvailability;
  collapsed: boolean;
}

// ─── State Domain 8: SavedViewState ───

export interface SavedView {
  id: string;
  name: string;
  builtin: boolean;
  queryState: Partial<QueryState>;
}

export interface SavedViewState {
  currentViewId: string | null;
  views: SavedView[];
  saveDialogOpen: boolean;
}

// ─── State Domain 9: LayoutState ───

export interface LayoutState {
  facetCollapsed: boolean;
  previewCollapsed: boolean;
  columnsPopoverOpen: boolean;
}

// ─── Top-Level Aggregate State ───

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

// ─── Default Columns & Groups ───

export const DEFAULT_COLUMNS: string[] = [
  "asset_id",
  "mcap_file_id",
  "duration",
  "env",
  "status",
  "algo",
  "tags",
  "updated_at",
];

export const DEFAULT_EXPANDED_GROUPS: string[] = [
  "basic",
  "capture",
  "algorithm",
  "delivery",
  "tags",
];

// ─── Default State ───

export const defaultAssetsDiscoveryState: AssetsDiscoveryState = {
  routerState: {
    urlHydrated: false,
    currentPath: "/assets",
  },
  searchUiState: {
    draftText: "",
    suggestionsOpen: false,
    highlightedIndex: -1,
    helpOpen: false,
  },
  queryState: {
    searchMode: "structured",
    queryText: "",
    activeFilters: [],
    sort: "-updated_at",
    page: 1,
    pageSize: 20,
    viewMode: "table",
    selectedColumns: DEFAULT_COLUMNS,
  },
  facetUiState: {
    expandedGroups: DEFAULT_EXPANDED_GROUPS,
    rangeDrafts: {},
    dateDrafts: {},
    addFilterOpen: false,
  },
  resultsState: {
    items: [],
    total: 0,
    totalApprox: false,
    fetchStatus: "idle",
    error: null,
    isStale: true,
  },
  selectionState: {
    selectedIds: new Set(),
    mode: "none",
  },
  previewState: {
    activeAssetId: null,
    fetchStatus: "idle",
    summary: null,
    availability: "missing",
    collapsed: false,
  },
  savedViewState: {
    currentViewId: null,
    views: [],
    saveDialogOpen: false,
  },
  layoutState: {
    facetCollapsed: false,
    previewCollapsed: false,
    columnsPopoverOpen: false,
  },
};
