// ─── Assets Discovery Workbench — Action Type Definitions ───
// Discriminated union covering all 9 action groups for the discovery reducer.
// Validates: Requirements R13

import type { Asset } from "../../api/types";
import type {
	FilterChip,
	PreviewManifest,
	QueryState,
	SavedView,
	SearchMode,
	SelectionMode,
	ViewMode,
} from "./assetsDiscoveryTypes";
import type { QueryDebugPlan, QueryRequest } from "../../api/query";

// ─── Action Union ───

export type AssetsDiscoveryAction =
	// ── Route / Hydration (3) ──
	| {
			type: "URL_HYDRATE";
			payload: {
				queryState: Partial<QueryState>;
				previewAssetId: string | null;
			};
	  }
	| { type: "MARK_URL_HYDRATED" }
	| { type: "SET_CURRENT_PATH"; payload: { path: string } }

	// ── Search UI (7) ──
	| { type: "SET_SEARCH_DRAFT"; payload: { text: string } }
	| { type: "SEARCH_DRAFT_CHANGE"; payload: { text: string } }
	| { type: "SEARCH_COMMIT"; payload: { tokens: FilterChip[] } }
	| { type: "SEARCH_MODE_CHANGE"; payload: { mode: SearchMode } }
	| { type: "TOGGLE_SUGGESTIONS"; payload: { open: boolean } }
	| { type: "SET_HIGHLIGHTED_INDEX"; payload: { index: number } }
	| { type: "TOGGLE_HELP"; payload: { open: boolean } }

	// ── Query Commit (11) ──
	| {
			type: "COMMIT_QUERY_TEXT";
			payload: { text: string; tokens: FilterChip[] };
	  }
	| { type: "ADD_FILTER_CHIP"; payload: { chip: FilterChip } }
	| { type: "REMOVE_FILTER_CHIP"; payload: { id: string } }
	| { type: "CLEAR_ALL_FILTERS" }
	| { type: "SET_SORT"; payload: { sort: string } }
	| { type: "SET_PAGE"; payload: { page: number } }
	| { type: "SET_PAGE_SIZE"; payload: { pageSize: number } }
	| { type: "SET_VIEW_MODE"; payload: { mode: ViewMode } }
	| { type: "SET_SELECTED_COLUMNS"; payload: { columns: string[] } }
	| { type: "SET_SEARCH_MODE"; payload: { mode: SearchMode } }
	| { type: "APPLY_SAVED_VIEW"; payload: { view: SavedView } }

	// ── Facet UI (7) ──
	| { type: "FACET_TOGGLE"; payload: { field: string; value: string } }
	| {
			type: "FACET_RANGE_DRAFT";
			payload: { field: string; min?: number; max?: number };
	  }
	| {
			type: "FACET_RANGE_APPLY";
			payload: { field: string; min?: number; max?: number };
	  }
	| {
			type: "FACET_DATE_DRAFT";
			payload: { field: string; start?: string; end?: string };
	  }
	| {
			type: "FACET_DATE_APPLY";
			payload: { field: string; start?: string; end?: string };
	  }
	| { type: "FACET_GROUP_TOGGLE"; payload: { group: string } }
	| { type: "TOGGLE_ADD_FILTER"; payload: { open: boolean } }

	// ── Results Fetch (4) ──
	| { type: "RESULTS_LOADING" }
	| {
			type: "RESULTS_SUCCESS";
			payload: {
				items: Asset[];
				total: number;
				totalApprox: boolean;
				aggregations?: Record<string, { key: string; doc_count: number }[]>;
				debugPlan?: QueryDebugPlan;
				warnings?: string[];
			};
	  }
	| { type: "RESULTS_ERROR"; payload: { error: string } }
	| { type: "MARK_RESULTS_FRESH" }
	| { type: "VALIDATION_LOADING" }
	| {
			type: "VALIDATION_SUCCESS";
			payload: {
				valid: boolean;
				normalizedQuery: QueryRequest | null;
				warnings: string[];
				fieldCapabilities: Array<{ field: string; engines: string[] }>;
				debugPlan: QueryDebugPlan | null;
			};
	  }
	| { type: "VALIDATION_ERROR"; payload: { error: string } }
	| { type: "TOGGLE_INSPECTOR"; payload: { open: boolean } }

	// ── Selection (4) ──
	| { type: "TOGGLE_ROW_SELECTION"; payload: { id: string } }
	| { type: "SELECT_ALL_FILTERED" }
	| { type: "CLEAR_SELECTION" }
	| { type: "SET_SELECTION_MODE"; payload: { mode: SelectionMode } }

	// ── Preview (6) ──
	| { type: "SET_ACTIVE_PREVIEW_ASSET"; payload: { assetId: string | null } }
	| { type: "PREVIEW_LOADING" }
	| {
			type: "RECEIVE_PREVIEW_SUCCESS";
			payload: { asset: Asset; manifest: PreviewManifest };
	  }
	| { type: "RECEIVE_PREVIEW_ERROR" }
	| { type: "PREVIEW_COLLAPSE_TOGGLE" }
	| { type: "PREVIEW_CLEAR" }

	// ── Saved Views (7) ──
	| { type: "SAVED_VIEW_SELECT"; payload: { viewId: string } }
	| { type: "SAVED_VIEW_SAVE"; payload: { name: string } }
	| { type: "SAVED_VIEW_DELETE"; payload: { viewId: string } }
	| { type: "SAVED_VIEW_RENAME"; payload: { viewId: string; name: string } }
	| { type: "SAVED_VIEW_LOAD"; payload: { views: SavedView[] } }
	| { type: "TOGGLE_SAVE_DIALOG"; payload: { open: boolean } }
	| { type: "SET_CURRENT_VIEW"; payload: { viewId: string | null } }

	// ── Layout (3) ──
	| { type: "FACET_COLLAPSE_TOGGLE" }
	| {
			type: "LAYOUT_RESPONSIVE";
			payload: { isTablet: boolean; isMobile: boolean };
	  }
	| { type: "TOGGLE_COLUMNS_POPOVER"; payload: { open: boolean } };

// ─── Action Creator Helpers ───

/**
 * Create a FilterChip with a unique id.
 * The id is composed of field + op + timestamp + random suffix to avoid collisions.
 */
export function createFilterChip(
	field: string,
	op: string,
	value: string | string[],
	source: FilterChip["source"],
): FilterChip {
	const id = `${field}_${op}_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`;
	return { id, field, op, value, source };
}

/**
 * Create a QueryToken (FilterChip with source "search") for search-bar parsed tokens.
 */
export function createQueryToken(
	field: string,
	op: string,
	value: string | string[],
): FilterChip {
	return createFilterChip(field, op, value, "search");
}
