// ─── Assets Discovery Workbench — Type Definitions ───
// All 9 state domains + supporting types for the discovery workbench.
// Validates: Requirements R13

import type { QueryDebugPlan, QueryRequest } from "../../api/query";
import type { Asset } from "../../api/types";

// ─── Union Types ───

export type SearchMode = "structured" | "keyword" | "semantic" | "similar";

export type FetchStatus = "idle" | "loading" | "success" | "error";

export type ViewMode = "table" | "card" | "compact";

export type SelectionMode = "none" | "explicit_rows" | "all_filtered_results";

export type PreviewMode =
	| "none"
	| "thumbnail"
	| "video"
	| "live-stream"
	| "mcap"
	| "sprite";

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

export interface PreviewSourceOption {
	id: string;
	label: string;
	kind: "live" | "file" | string;
	codec?: string;
	topic?: string;
}

export interface PreviewManifest {
	thumbnailUrl: string | null;
	previewVideoUrl: string | null;
	mcapUrl?: string | null;
	sourceId?: string | null;
	ds?: string | null;
	dsParams?: Record<string, string>;
	windowStartSec?: number | null;
	windowEndSec?: number | null;
	availability: PreviewAvailability;
	mode: PreviewMode;
	sources?: PreviewSourceOption[];
	recommendedSourceId?: string;
	activeSourceId?: string | null;
}

// ─── State Domain 1: RouterState ───

export interface RouterState {
	urlHydrated: boolean;
}

// ─── State Domain 2: SearchUiState ───

export interface SearchUiState {
	draftText: string;
	suggestionsOpen: boolean;
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
	aggregations?: Record<string, { key: string; doc_count: number }[]>;
	debugPlan?: QueryDebugPlan;
	warnings?: string[];
}

// ─── State Domain 6: SelectionState ───

export interface SelectionState {
	selectedIds: Set<string>;
	mode: SelectionMode;
}

// ─── State Domain 7: PreviewState ───

export interface PreviewState {
	activeAssetId: string | null;
	activeSourceId: string | null;
	activeTopic: string | null;
	activeTimeSec: number | null;
	ds: string | null;
	dsParams: Record<string, string> | null;
	fetchStatus: FetchStatus;
	summary: AssetPreviewSummary | null;
	manifest: PreviewManifest | null;
	availability: PreviewAvailability;
	retryNonce: number;
	collapsed: boolean;
}

export interface ValidationState {
	fetchStatus: FetchStatus;
	valid: boolean;
	error: string | null;
	normalizedQuery: QueryRequest | null;
	warnings: string[];
	fieldCapabilities: Array<{ field: string; engines: string[] }>;
	debugPlan: QueryDebugPlan | null;
	inspectorOpen: boolean;
}

// ─── State Domain 9: LayoutState ───

export interface LayoutState {
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
	layoutState: LayoutState;
	validationState: ValidationState;
}

// ─── Default Columns & Groups ───

export const DEFAULT_COLUMNS: string[] = [
	"asset_id",
	"asset_type",
	"lifecycle_state",
	"owner",
	"duration",
	"updated_at",
];

export const DEFAULT_EXPANDED_GROUPS: string[] = ["algorithm"];

// ─── Default State ───

export const defaultAssetsDiscoveryState: AssetsDiscoveryState = {
	routerState: {
		urlHydrated: false,
	},
	searchUiState: {
		draftText: "",
		suggestionsOpen: false,
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
		debugPlan: undefined,
		warnings: [],
	},
	selectionState: {
		selectedIds: new Set(),
		mode: "none",
	},
	previewState: {
		activeAssetId: null,
		activeSourceId: null,
		activeTopic: null,
		activeTimeSec: null,
		ds: null,
		dsParams: null,
		// Default collapsed: hand the horizontal space back to the results table.
		// Users open the panel on demand via the toggle button.
		fetchStatus: "idle",
		summary: null,
		manifest: null,
		availability: "missing",
		retryNonce: 0,
		collapsed: true,
	},
	layoutState: {
		columnsPopoverOpen: false,
	},
	validationState: {
		fetchStatus: "idle",
		valid: false,
		error: null,
		normalizedQuery: null,
		warnings: [],
		fieldCapabilities: [],
		debugPlan: null,
		inspectorOpen: false,
	},
};
