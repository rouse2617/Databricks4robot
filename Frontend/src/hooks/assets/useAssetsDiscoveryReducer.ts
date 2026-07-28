// ─── Assets Discovery Workbench — Reducer Hook ───
// Wraps useReducer with side-effects for results fetching and preview loading.
// Validates: Requirements R13

import { useEffect, useMemo, useReducer, useRef } from "react";
import { assetsApi } from "../../api/assets";
import {
	type QueryExpr,
	type QueryFacetBucket,
	type QueryRequest,
	type QuerySort,
	queryApi,
} from "../../api/query";
import type { Asset } from "../../api/types";
import { extractApiErrorMessage } from "../../lib/apiError";
import type { AssetsDiscoveryAction } from "../../lib/assets/assetsDiscoveryActions";
import { assetsDiscoveryReducer } from "../../lib/assets/assetsDiscoveryReducer";
import type { AssetsDiscoveryState } from "../../lib/assets/assetsDiscoveryTypes";
import { defaultAssetsDiscoveryState } from "../../lib/assets/assetsDiscoveryTypes";
import { buildPreviewManifestFromSources } from "./useAssetPreview";

// ─── Helpers ───

/**
 * Derive a stable query key string from the effective Query IR request, so the
 * results effect can detect meaningful changes without deep-comparing objects.
 */
function stableStringify(value: unknown): string {
	const seen = new WeakSet<object>();

	const walk = (v: unknown): unknown => {
		if (v === null) return null;
		if (Array.isArray(v)) return v.map(walk);
		if (typeof v !== "object") return v;

		const obj = v as Record<string, unknown>;
		if (seen.has(obj)) return "[Circular]";
		seen.add(obj);

		const out: Record<string, unknown> = {};
		for (const key of Object.keys(obj).sort()) {
			const child = obj[key];
			if (child === undefined) continue;
			out[key] = walk(child);
		}
		return out;
	};

	return JSON.stringify(walk(value));
}

// CYB-3385: multi-value facet semantics — same (field, op) group is OR
// (inter-value), different groups are AND (inter-field). Fixes the bug where
// selecting two asset_type values (segment + action) produced
// `type=segment AND type=action` → 0 hits. Keeps single-chip queries as a
// plain {pred:...} to avoid a needless {or:[single]} wrapper on the wire.
// Exported so pure tests can exercise the grouping without spinning up the
// full hook + queryApi mock stack.
export function buildStructuredQueryWhere(
	queryState: AssetsDiscoveryState["queryState"],
): QueryExpr | undefined {
	if (queryState.activeFilters.length === 0) {
		return undefined;
	}
	const grouped = new Map<string, QueryExpr[]>();
	const order: string[] = [];
	for (const chip of queryState.activeFilters) {
		const key = `${chip.field}\x00${chip.op}`;
		const pred: QueryExpr = {
			pred: { field: chip.field, op: chip.op, value: chip.value },
		};
		const bucket = grouped.get(key);
		if (bucket) {
			bucket.push(pred);
		} else {
			grouped.set(key, [pred]);
			order.push(key);
		}
	}
	const groupExprs = order.map((key) => {
		const preds = grouped.get(key) ?? [];
		return preds.length === 1 ? preds[0] : { or: preds };
	});
	if (groupExprs.length === 1) {
		return groupExprs[0];
	}
	return { and: groupExprs };
}

function buildStructuredQuerySort(sort: string): QuerySort[] {
	if (!sort) {
		return [];
	}
	if (sort.startsWith("-")) {
		return [{ field: sort.slice(1), direction: "desc" }];
	}
	return [{ field: sort, direction: "asc" }];
}

const ASSET_DISCOVERY_FACETS: NonNullable<QueryRequest["facets"]> = [
	{ field: "lifecycle_state", size: 20 },
	{ field: "asset_type", size: 20 },
	{ field: "owner", size: 20 },
	{ field: "mcap.vendor_id", size: 20 },
	{ field: "mcap.scene_id", size: 20 },
	{ field: "tag.priority", size: 20 },
	{ field: "tag.quality", size: 20 },
	// CYB-3297 Phase B: env is free-form data (metadata.env); request it as a
	// facet so the sidebar can list the real values instead of a hardcoded set.
	{ field: "env", size: 50 },
	// CYB-3715: 4 flatten mirror columns exposed as top-level asset fields.
	// Text columns are facet-able (low cardinality — camera model families,
	// data source enum, source_platform enum). UUID fields (device_id /
	// collector_id / scene_id) remain filter-only, no facet request.
	{ field: "camera_model", size: 20 },
	{ field: "data_source", size: 20 },
	{ field: "collection_method", size: 20 },
	{ field: "source_platform", size: 20 },
];

function buildStructuredQueryWhereExpr(
	queryState: AssetsDiscoveryState["queryState"],
): QueryExpr | undefined {
	const where = buildStructuredQueryWhere(queryState);
	const trimmedQuery = queryState.queryText.trim();
	const isExactAssetId = /^[A-Za-z0-9]{8}$/.test(trimmedQuery);
	const keywordExpr =
		queryState.searchMode === "keyword" && trimmedQuery.length > 0
			? {
					pred: isExactAssetId
						? {
								field: "asset_id",
								op: "eq",
								value: trimmedQuery,
							}
						: {
								field: "_fulltext",
								op: "ilike",
								value: trimmedQuery,
							},
				}
			: null;

	if (keywordExpr && where) {
		return { and: [keywordExpr, where] };
	}
	return keywordExpr ?? where;
}

/** List request: no facets so the backend skips ES aggregation on first paint. */
function buildListQueryRequest(
	queryState: AssetsDiscoveryState["queryState"],
): QueryRequest {
	return {
		schema_version: "v1",
		mode: queryState.searchMode,
		scope: { resource: "assets" },
		select: { fields: queryState.selectedColumns },
		where: buildStructuredQueryWhereExpr(queryState),
		sort: buildStructuredQuerySort(queryState.sort),
		page: {
			page: queryState.page,
			page_size: queryState.pageSize,
			offset: (queryState.page - 1) * queryState.pageSize,
			limit: queryState.pageSize,
		},
		debug: { explain: true },
	};
}

/**
 * CYB-3231: cap for "select all filtered results" — we resolve matching asset_ids
 * client-side, so bound how many we pull to avoid unbounded id lists.
 */
export const SELECT_ALL_MAX = 1000;

/**
 * Build a query that returns just the asset_ids matching the current filters
 * (same where/sort as the list), one page up to `limit`. Used to populate the
 * selection for "select all filtered results".
 */
export function buildSelectAllIdsQueryRequest(
	queryState: AssetsDiscoveryState["queryState"],
	limit: number,
): QueryRequest {
	return {
		schema_version: "v1",
		mode: queryState.searchMode,
		scope: { resource: "assets" },
		select: { fields: ["asset_id"] },
		where: buildStructuredQueryWhereExpr(queryState),
		sort: buildStructuredQuerySort(queryState.sort),
		page: { page: 1, page_size: limit, offset: 0, limit },
	};
}

type FacetFilterSlice = Pick<
	AssetsDiscoveryState["queryState"],
	"searchMode" | "queryText" | "activeFilters" | "sort"
>;

/** Facet request: same filters as list but minimal page; keyed without page. */
function buildFacetsQueryRequest(filters: FacetFilterSlice): QueryRequest {
	const queryState = {
		...filters,
		page: 1,
		pageSize: 1,
		selectedColumns: ["asset_id"],
		viewMode: "table" as const,
	};
	return {
		schema_version: "v1",
		mode: filters.searchMode,
		scope: { resource: "assets" },
		select: { fields: ["asset_id"] },
		where: buildStructuredQueryWhereExpr(queryState),
		sort: buildStructuredQuerySort(filters.sort),
		page: { page: 1, page_size: 1, offset: 0, limit: 1 },
		facets: ASSET_DISCOVERY_FACETS,
	};
}

function mapFacetFieldToAggregationKey(field: string): string {
	switch (field) {
		case "lifecycle_state":
			return "lifecycle_state_agg";
		case "asset_type":
			return "asset_type_agg";
		case "owner":
			return "owner_agg";
		case "mcap.vendor_id":
			return "vendor_agg";
		case "mcap.scene_id":
			return "scene_agg";
		case "tag.priority":
			return "priority_agg";
		case "tag.quality":
			return "quality_agg";
		case "env":
			return "env_agg";
		default:
			return field;
	}
}

function mapFacetsToAggregations(
	facets?: Record<string, QueryFacetBucket[]>,
): Record<string, { key: string; doc_count: number }[]> | undefined {
	if (!facets) return undefined;
	const out: Record<string, { key: string; doc_count: number }[]> = {};
	for (const [field, buckets] of Object.entries(facets)) {
		out[mapFacetFieldToAggregationKey(field)] = (buckets ?? []).map(
			(bucket) => ({
				key: bucket.value,
				doc_count: bucket.count,
			}),
		);
	}
	return out;
}

// ─── Hook ───

export function useAssetsDiscoveryReducer(): [
	AssetsDiscoveryState,
	React.Dispatch<AssetsDiscoveryAction>,
] {
	const [state, dispatch] = useReducer(
		assetsDiscoveryReducer,
		defaultAssetsDiscoveryState,
	);

	const listQueryRequest = useMemo(
		() => buildListQueryRequest(state.queryState),
		[state.queryState],
	);
	const facetFilters: FacetFilterSlice = useMemo(
		() => ({
			searchMode: state.queryState.searchMode,
			queryText: state.queryState.queryText,
			activeFilters: state.queryState.activeFilters,
			sort: state.queryState.sort,
		}),
		[
			state.queryState.searchMode,
			state.queryState.queryText,
			state.queryState.activeFilters,
			state.queryState.sort,
		],
	);
	const facetsQueryRequest = useMemo(
		() => buildFacetsQueryRequest(facetFilters),
		[facetFilters],
	);

	const listRequestKey = useMemo(
		() => stableStringify(listQueryRequest),
		[listQueryRequest],
	);
	const facetsRequestKey = useMemo(
		() => stableStringify(facetsQueryRequest),
		[facetsQueryRequest],
	);
	const currentListKey = listRequestKey;
	const listKeyRef = useRef(currentListKey);
	/** Last facets filter key we successfully loaded (excludes page). */
	const facetsLoadedKeyRef = useRef<string | null>(null);
	/**
	 * Authoritative total for the current filters, sourced from the dedicated
	 * count/facets query. The list query's `total` is unreliable when ES is
	 * unavailable (it can be smaller than the rows actually returned), so we
	 * cache the count here and reuse it across page changes.
	 */
	const facetsTotalRef = useRef<number | null>(null);
	const facetsRequestKeyRef = useRef(facetsRequestKey);
	facetsRequestKeyRef.current = facetsRequestKey;
	const inFlightRef = useRef(false);
	const isResultsStale = state.resultsState.isStale;
	const isURLHydrated = state.routerState.urlHydrated;
	const activePreviewAssetID = state.previewState.activeAssetId;
	const pageSize = state.queryState.pageSize;

	// ── Results fetch effect ──
	// Fires when resultsState.isStale becomes true OR when the derived query key
	// changes. Depending on `state.queryState` directly would re-run on every
	// dispatch because reducers always produce new references.
	useEffect(() => {
		if (!isResultsStale || !isURLHydrated) return;

		// Deduplicate: if a fetch is already in-flight, skip.
		if (inFlightRef.current) return;
		inFlightRef.current = true;

		listKeyRef.current = currentListKey;

		let cancelled = false;

		const fetchResults = async () => {
			dispatch({ type: "RESULTS_LOADING" });

			try {
				const data = await queryApi.run(listQueryRequest);
				const items: Asset[] = data.items ?? [];
				// The list query's `total` is unreliable when ES is unavailable
				// (it can be smaller than the rows actually returned). Prefer the
				// authoritative count from the facets query when it has loaded for
				// the current filters, and always floor by the rows on this page so
				// the UI never shows `total < rows`.
				const cachedAuthoritativeTotal =
					facetsLoadedKeyRef.current === facetsRequestKey
						? facetsTotalRef.current
						: null;
				const total: number = Math.max(
					cachedAuthoritativeTotal ?? data.total ?? 0,
					items.length,
				);
				if (cancelled || listKeyRef.current !== currentListKey) {
					inFlightRef.current = false;
					return;
				}
				const warnings = [...(data.warnings ?? [])];
				dispatch({
					type: "RESULTS_SUCCESS",
					payload: {
						items,
						total,
						totalApprox:
							cachedAuthoritativeTotal != null
								? false
								: total > 0 && items.length === pageSize,
						debugPlan: data.debug_plan,
						warnings,
					},
				});
				// Avoid stale right preview when current query returns no rows.
				if (total === 0 && activePreviewAssetID) {
					dispatch({ type: "PREVIEW_CLEAR" });
				}
				inFlightRef.current = false;
				return;
			} catch (err) {
				if (cancelled || listKeyRef.current !== currentListKey) {
					inFlightRef.current = false;
					return;
				}
				dispatch({
					type: "RESULTS_ERROR",
					payload: {
						error: extractApiErrorMessage(err, "查询资产失败"),
					},
				});
				inFlightRef.current = false;
			}
		};

		fetchResults();

		return () => {
			cancelled = true;
			inFlightRef.current = false;
		};
	}, [
		activePreviewAssetID,
		currentListKey,
		facetsRequestKey,
		isResultsStale,
		isURLHydrated,
		pageSize,
		listQueryRequest,
	]);

	// ── Facets fetch effect ──
	// Decoupled from the list fetch on purpose (CYB-3301). Previously the facets
	// request was chained after the list request in the effect above; the list's
	// RESULTS_SUCCESS flips isStale true→false, which re-runs that effect and
	// fires its cleanup (cancelled=true) — cancelling the still-in-flight facets
	// fetch before it could dispatch FACETS_SUCCESS. Net effect: the facets
	// request was sent and answered, but its aggregations were discarded, so no
	// facet ever showed counts. Keying this effect on the filter key only (not
	// isStale) avoids that race.
	useEffect(() => {
		if (!isURLHydrated) return;
		if (facetsLoadedKeyRef.current === facetsRequestKey) return;

		let cancelled = false;
		(async () => {
			try {
				const facetData = await queryApi.run(facetsQueryRequest);
				if (cancelled || facetsRequestKeyRef.current !== facetsRequestKey) {
					return;
				}
				// The count/facets query returns the authoritative total (postgres
				// count fallback when ES is down). Apply it even when no aggregation
				// buckets come back (facets === null).
				const authoritativeTotal =
					typeof facetData.total === "number" ? facetData.total : null;
				const aggregations = mapFacetsToAggregations(facetData.facets);
				if (authoritativeTotal == null && !aggregations) {
					return;
				}
				if (authoritativeTotal != null) {
					facetsTotalRef.current = authoritativeTotal;
				}
				facetsLoadedKeyRef.current = facetsRequestKey;
				dispatch({
					type: "FACETS_SUCCESS",
					payload: {
						...(aggregations ? { aggregations } : {}),
						...(authoritativeTotal != null
							? { total: authoritativeTotal }
							: {}),
					},
				});
			} catch {
				// Facet sidebar keeps previous counts; the list is unaffected.
			}
		})();

		return () => {
			cancelled = true;
		};
	}, [isURLHydrated, facetsRequestKey, facetsQueryRequest]);

	// ── Preview fetch effect ──
	// Fires when active preview asset or retry nonce changes.
	const prevPreviewFetchKeyRef = useRef<string>("");

	useEffect(() => {
		const activeId = state.previewState.activeAssetId;
		const fetchKey = `${activeId ?? ""}:${state.previewState.retryNonce}`;

		// Skip if unchanged
		if (fetchKey === prevPreviewFetchKeyRef.current) return;
		prevPreviewFetchKeyRef.current = fetchKey;

		// Nothing to fetch
		if (!activeId) return;

		let cancelled = false;

		const fetchPreview = async () => {
			dispatch({ type: "PREVIEW_LOADING" });

			try {
				const [asset, foxgloveSource] = await Promise.all([
					assetsApi.get(activeId),
					assetsApi.getFoxgloveSource(activeId).catch(() => null),
				]);

				if (cancelled) return;

				const manifest = buildPreviewManifestFromSources(
					asset,
					null,
					foxgloveSource,
					{
						previewSourceId: state.previewState.activeSourceId ?? undefined,
						previewTopic: state.previewState.activeTopic ?? undefined,
					},
				);

				dispatch({
					type: "RECEIVE_PREVIEW_SUCCESS",
					payload: { asset, manifest },
				});
			} catch {
				if (cancelled) return;
				dispatch({ type: "RECEIVE_PREVIEW_ERROR" });
			}
		};

		fetchPreview();

		return () => {
			cancelled = true;
		};
	}, [
		state.previewState.activeAssetId,
		state.previewState.activeSourceId,
		state.previewState.activeTopic,
		state.previewState.retryNonce,
	]);

	return [state, dispatch];
}
