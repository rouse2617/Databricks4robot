// ─── Assets Discovery Workbench — Reducer Hook ───
// Wraps useReducer with side-effects for results fetching and preview loading.
// Validates: Requirements R13

import { useEffect, useMemo, useReducer, useRef } from "react";
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
import { fetchPreviewBundle } from "./useAssetPreview";

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

function buildStructuredQueryWhere(
	queryState: AssetsDiscoveryState["queryState"],
): QueryExpr | undefined {
	const predicates: QueryExpr[] = queryState.activeFilters.map((chip) => ({
		pred: {
			field: chip.field,
			op: chip.op,
			value: chip.value,
		},
	}));
	if (predicates.length === 0) {
		return undefined;
	}
	if (predicates.length === 1) {
		return predicates[0];
	}
	return { and: predicates };
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

function assetMatchesAlgoStatusFilter(
	asset: Asset,
	statuses: string[],
): boolean {
	if (statuses.length === 0) return true;
	const algoResults = asset.algo_results ?? {};
	const seenStatuses = new Set<string>();
	for (const [key, value] of Object.entries(algoResults)) {
		if (!key.endsWith(":status")) continue;
		seenStatuses.add(String(value));
	}
	if (seenStatuses.size === 0) {
		seenStatuses.add("pending");
	}
	return statuses.some((status) => seenStatuses.has(status));
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
	const facetsRequestKeyRef = useRef(facetsRequestKey);
	facetsRequestKeyRef.current = facetsRequestKey;
	const inFlightRef = useRef(false);
	const isResultsStale = state.resultsState.isStale;
	const isURLHydrated = state.routerState.urlHydrated;
	const activePreviewAssetID = state.previewState.activeAssetId;
	const pageSize = state.queryState.pageSize;
	const activeAlgoStatusFilters = useMemo(
		() =>
			state.queryState.activeFilters
				.filter((chip) => chip.field === "algo_status" && chip.op === "eq")
				.map((chip) => String(chip.value)),
		[state.queryState.activeFilters],
	);

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
		const facetsKeyAtStart = facetsRequestKey;
		const shouldRefreshFacets = facetsLoadedKeyRef.current !== facetsKeyAtStart;

		const fetchResults = async () => {
			dispatch({ type: "RESULTS_LOADING" });

			try {
				const data = await queryApi.run(listQueryRequest);
				const fetchedItems: Asset[] = data.items ?? [];
				const hasAlgoStatusFallback = activeAlgoStatusFilters.length > 0;
				const items = hasAlgoStatusFallback
					? fetchedItems.filter((asset) =>
							assetMatchesAlgoStatusFilter(asset, activeAlgoStatusFilters),
						)
					: fetchedItems;
				// total always uses backend count; frontend fallback only affects current page items.
				const total: number = data.total ?? 0;
				if (cancelled || listKeyRef.current !== currentListKey) {
					inFlightRef.current = false;
					return;
				}
				const warnings = [...(data.warnings ?? [])];
				if (hasAlgoStatusFallback) {
					warnings.push(
						"⚠️ algo_status 筛选仅在当前页生效：列表已过滤，但总数和分页仍为全量数据。如需精确结果请在算法页复核，或清除 algo_status 筛选。",
					);
				}
				dispatch({
					type: "RESULTS_SUCCESS",
					payload: {
						items,
						total,
						totalApprox: total > 0 && items.length === pageSize,
						debugPlan: data.debug_plan,
						warnings,
					},
				});
				// Avoid stale right preview when current query returns no rows.
				if (total === 0 && activePreviewAssetID) {
					dispatch({ type: "PREVIEW_CLEAR" });
				}
				inFlightRef.current = false;

				if (!shouldRefreshFacets || cancelled) {
					return;
				}

				try {
					const facetData = await queryApi.run(facetsQueryRequest);
					if (
						cancelled ||
						listKeyRef.current !== currentListKey ||
						facetsKeyAtStart !== facetsRequestKeyRef.current
					) {
						return;
					}
					if (facetsLoadedKeyRef.current === facetsKeyAtStart) {
						return;
					}
					const aggregations = mapFacetsToAggregations(facetData.facets);
					if (!aggregations) {
						return;
					}
					facetsLoadedKeyRef.current = facetsKeyAtStart;
					dispatch({
						type: "FACETS_SUCCESS",
						payload: { aggregations },
					});
				} catch {
					// Facet sidebar can keep previous counts; list is already shown.
				}
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
		facetsQueryRequest,
		facetsRequestKey,
		isResultsStale,
		isURLHydrated,
		pageSize,
		activeAlgoStatusFilters,
		listQueryRequest,
	]);

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
				const { asset, manifest } = await fetchPreviewBundle(activeId, {
					previewSourceId: state.previewState.activeSourceId ?? undefined,
					previewTopic: state.previewState.activeTopic ?? undefined,
				});

				if (cancelled) return;

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
