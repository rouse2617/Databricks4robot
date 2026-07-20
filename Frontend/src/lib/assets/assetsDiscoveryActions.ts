// ─── Assets Discovery Workbench — Action Type Definitions ───
// Discriminated union covering all 9 action groups for the discovery reducer.
// Validates: Requirements R13

import type { QueryDebugPlan, QueryRequest } from "../../api/query";
import type { Asset } from "../../api/types";
import type {
	FilterChip,
	PreviewManifest,
	QueryState,
	SearchMode,
	ViewMode,
} from "./assetsDiscoveryTypes";

// ─── Action Union ───

export type AssetsDiscoveryAction =
	// ── Route / Hydration (3) ──
	| {
			type: "URL_HYDRATE";
			payload: {
				queryState: Partial<QueryState>;
				previewAssetId: string | null;
				previewSourceId: string | null;
				previewTopic: string | null;
				previewTimeSec?: number | null;
				ds?: string | null;
				dsParams?: Record<string, string> | null;
			};
	  }
	| { type: "MARK_URL_HYDRATED" }

	// ── Search UI (7) ──
	| { type: "SET_SEARCH_DRAFT"; payload: { text: string } }
	| { type: "TOGGLE_SUGGESTIONS"; payload: { open: boolean } }

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
	| {
			type: "FACETS_SUCCESS";
			payload: {
				aggregations?: Record<string, { key: string; doc_count: number }[]>;
				/**
				 * Authoritative total from the dedicated count/facets query
				 * (postgres count fallback when ES is unavailable). Applied even
				 * when no aggregation buckets are returned.
				 */
				total?: number;
			};
	  }
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

	// ── Selection (5) ──
	| { type: "TOGGLE_ROW_SELECTION"; payload: { id: string } }
	// CYB-3231: replace the selection with an explicit id set (used by
	// "select all filtered results" after fetching the matching asset_ids).
	| { type: "SET_SELECTED_IDS"; payload: { ids: string[] } }
	| { type: "CLEAR_SELECTION" }

	// ── Preview (6) ──
	| { type: "SET_ACTIVE_PREVIEW_ASSET"; payload: { assetId: string | null } }
	| { type: "SET_PREVIEW_SOURCE"; payload: { sourceId: string | null } }
	| { type: "SET_PREVIEW_TOPIC"; payload: { topic: string | null } }
	| { type: "SET_PREVIEW_TIME"; payload: { timeSec: number | null } }
	| { type: "PREVIEW_LOADING" }
	| {
			type: "RECEIVE_PREVIEW_SUCCESS";
			payload: { asset: Asset; manifest: PreviewManifest };
	  }
	| { type: "RECEIVE_PREVIEW_ERROR" }
	| { type: "PREVIEW_RETRY" }
	| { type: "PREVIEW_COLLAPSE_TOGGLE" }
	| { type: "PREVIEW_CLEAR" }

	// ── Layout ──
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
