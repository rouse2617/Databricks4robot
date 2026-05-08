// ─── Assets Discovery Workbench — Reducer Hook ───
// Wraps useReducer with side-effects for results fetching and preview loading.
// Validates: Requirements R13

import { useEffect, useMemo, useReducer, useRef } from "react";
import { assetsApi } from "../../api/assets";
import {
	queryApi,
	type QueryExpr,
	type QueryFacetBucket,
	type QueryRequest,
	type QuerySort,
} from "../../api/query";
import type { Asset } from "../../api/types";
import { extractApiErrorMessage } from "../../lib/apiError";
import type { AssetsDiscoveryAction } from "../../lib/assets/assetsDiscoveryActions";
import { assetsDiscoveryReducer } from "../../lib/assets/assetsDiscoveryReducer";
import type { AssetsDiscoveryState } from "../../lib/assets/assetsDiscoveryTypes";
import { defaultAssetsDiscoveryState } from "../../lib/assets/assetsDiscoveryTypes";
import { buildPlaceholderPreviewManifest } from "./useAssetPreview";

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

function buildStructuredQueryWhere(state: AssetsDiscoveryState): QueryExpr | undefined {
	const predicates: QueryExpr[] = state.queryState.activeFilters
		.map((chip) => ({
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

function buildStructuredQueryRequest(state: AssetsDiscoveryState): QueryRequest {
	const where = buildStructuredQueryWhere(state);
	const keywordExpr =
		state.queryState.searchMode === "keyword" &&
		state.queryState.queryText.trim().length > 0
			? {
					pred: {
						field: "_fulltext",
						op: "ilike",
						value: state.queryState.queryText.trim(),
					},
				}
			: null;

	return {
		schema_version: "v1",
		mode: state.queryState.searchMode,
		scope: { resource: "assets" },
		select: { fields: state.queryState.selectedColumns },
		where:
			keywordExpr && where
				? { and: [keywordExpr, where] }
				: keywordExpr ?? where,
		sort: buildStructuredQuerySort(state.queryState.sort),
		page: {
			// Send both paging styles for compatibility while moving to offset/limit.
			page: state.queryState.page,
			page_size: state.queryState.pageSize,
			offset: (state.queryState.page - 1) * state.queryState.pageSize,
			limit: state.queryState.pageSize,
		},
		facets: [
			{ field: "lifecycle_state", size: 20 },
			{ field: "asset_type", size: 20 },
			{ field: "owner", size: 20 },
			{ field: "mcap.vendor_id", size: 20 },
			{ field: "mcap.scene_id", size: 20 },
			{ field: "tag.priority", size: 20 },
			{ field: "tag.quality", size: 20 },
		],
		debug: { explain: true },
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
		out[mapFacetFieldToAggregationKey(field)] = (buckets ?? []).map((bucket) => ({
			key: bucket.value,
			doc_count: bucket.count,
		}));
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

	// Track the latest query key to discard stale responses
	const requestKey = useMemo(
		() => stableStringify(buildStructuredQueryRequest(state)),
		[
			state.queryState.activeFilters,
			state.queryState.page,
			state.queryState.pageSize,
			state.queryState.queryText,
			state.queryState.searchMode,
			state.queryState.selectedColumns,
			state.queryState.sort,
		],
	);
	const currentKey = requestKey;
	const queryKeyRef = useRef(currentKey);

	// ── Results fetch effect ──
	// Fires when resultsState.isStale becomes true OR when the derived query key
	// changes. Depending on `state.queryState` directly would re-run on every
	// dispatch because reducers always produce new references.
	useEffect(() => {
		if (!state.resultsState.isStale || !state.routerState.urlHydrated) return;

		queryKeyRef.current = currentKey;

		let cancelled = false;

		const fetchResults = async () => {
			dispatch({ type: "RESULTS_LOADING" });

			try {
				let items: Asset[];
				let total: number;
				const request = buildStructuredQueryRequest(state);
				const data = await queryApi.run(request);
				items = data.items ?? [];
				total = data.total ?? 0;
				if (cancelled || queryKeyRef.current !== currentKey) return;
				dispatch({
					type: "RESULTS_SUCCESS",
					payload: {
						items,
						total,
						totalApprox:
							total > 0 && items.length === state.queryState.pageSize,
						aggregations: mapFacetsToAggregations(data.facets),
						debugPlan: data.debug_plan,
						warnings: data.warnings,
					},
				});
				// Avoid stale right preview when current query returns no rows.
				if (total === 0 && state.previewState.activeAssetId) {
					dispatch({ type: "PREVIEW_CLEAR" });
				}
				return;
			} catch (err) {
				if (cancelled || queryKeyRef.current !== currentKey) return;
				dispatch({
					type: "RESULTS_ERROR",
					payload: {
						error: extractApiErrorMessage(err, "查询资产失败"),
					},
				});
			}
		};

		fetchResults();

		return () => {
			cancelled = true;
		};
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [
		state.resultsState.isStale,
		state.routerState.urlHydrated,
		currentKey,
	]);

	// ── Preview fetch effect ──
	// Fires when previewState.activeAssetId changes to a non-null value.
	const prevPreviewIdRef = useRef<string | null>(null);

	useEffect(() => {
		const activeId = state.previewState.activeAssetId;

		// Skip if unchanged
		if (activeId === prevPreviewIdRef.current) return;
		prevPreviewIdRef.current = activeId;

		// Nothing to fetch
		if (!activeId) return;

		let cancelled = false;

		const fetchPreview = async () => {
			dispatch({ type: "PREVIEW_LOADING" });

			try {
				const asset = await assetsApi.get(activeId);

				if (cancelled) return;

				const manifest = buildPlaceholderPreviewManifest(asset);

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
	}, [state.previewState.activeAssetId]);

	return [state, dispatch];
}
