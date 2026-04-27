// ─── Assets Discovery Workbench — Reducer Hook ───
// Wraps useReducer with side-effects for results fetching and preview loading.
// Validates: Requirements R13

import { useReducer, useEffect, useRef } from "react";
import { assetsDiscoveryReducer } from "../../lib/assets/assetsDiscoveryReducer";
import type {
  AssetsDiscoveryState,
  PreviewManifest,
  PreviewAvailability,
  PreviewMode,
} from "../../lib/assets/assetsDiscoveryTypes";
import { defaultAssetsDiscoveryState } from "../../lib/assets/assetsDiscoveryTypes";
import type { AssetsDiscoveryAction } from "../../lib/assets/assetsDiscoveryActions";
import { assetsApi } from "../../api/assets";
import { searchApi } from "../../api/search";
import type { Asset } from "../../api/types";

// ─── Helpers ───

/**
 * Derive a stable query key string from queryState so the results effect
 * can detect meaningful changes without deep-comparing objects.
 */
function deriveQueryKey(state: AssetsDiscoveryState): string {
  const q = state.queryState;
  const filterKeys = q.activeFilters
    .map((f) => `${f.field}:${f.op}:${Array.isArray(f.value) ? f.value.join(",") : f.value}`)
    .sort()
    .join("|");
  return `${q.searchMode}::${q.queryText}::${q.sort}::${q.page}::${q.pageSize}::${filterKeys}`;
}

/**
 * Convert activeFilters into the `filter` string array expected by assetsApi.list().
 * Each FilterChip becomes `field:op:value`.
 */
function buildFilterParams(state: AssetsDiscoveryState): string[] {
  return state.queryState.activeFilters.map((chip) => {
    const val = Array.isArray(chip.value)
      ? `[${chip.value.map((v) => `"${v}"`).join(",")}]`
      : chip.value;
    return `${chip.field}:${chip.op}:${val}`;
  });
}

/**
 * Build a placeholder PreviewManifest from an asset's files map.
 * Checks for thumbnail and preview_mp4 keys to determine availability.
 */
function buildPlaceholderPreviewManifest(asset: Asset): PreviewManifest {
  const hasThumbnail = !!(asset.files && asset.files.thumbnail);
  const hasVideo = !!(asset.files && asset.files.preview_mp4);

  let availability: PreviewAvailability = "missing";
  let mode: PreviewMode = "none";

  if (hasVideo) {
    availability = "ready";
    mode = "video";
  } else if (hasThumbnail) {
    availability = "ready";
    mode = "thumbnail";
  }

  return {
    thumbnailUrl: hasThumbnail ? asset.files.thumbnail : null,
    previewVideoUrl: hasVideo ? asset.files.preview_mp4 : null,
    availability,
    mode,
  };
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
  const queryKeyRef = useRef(deriveQueryKey(state));

  // ── Results fetch effect ──
  // Fires when resultsState.isStale becomes true.
  useEffect(() => {
    if (!state.resultsState.isStale || !state.routerState.urlHydrated) return;

    const currentKey = deriveQueryKey(state);
    queryKeyRef.current = currentKey;

    let cancelled = false;

    const fetchResults = async () => {
      dispatch({ type: "RESULTS_LOADING" });

      const filters = buildFilterParams(state);

      try {
        let items: Asset[];
        let total: number;

        if (state.queryState.searchMode === "keyword") {
          // Keyword mode → OpenSearch via /search/assets
          const data = await searchApi.searchAssets({
            q: state.queryState.queryText || undefined,
            filter: filters.length > 0 ? filters : undefined,
            page: state.queryState.page,
            page_size: state.queryState.pageSize,
          });
          items = (data.items ?? []) as unknown as Asset[];
          total = data.total ?? 0;

          // Discard if a newer query has been issued
          if (cancelled || queryKeyRef.current !== currentKey) return;

          dispatch({
            type: "RESULTS_SUCCESS",
            payload: {
              items,
              total,
              totalApprox:
                total > 0 &&
                items.length === state.queryState.pageSize,
              aggregations: data.aggregations,
            },
          });
          return;
        } else {
          // Structured mode → Postgres via /assets
          const data = await assetsApi.list({
            filter: filters.length > 0 ? filters : undefined,
            sort_by: state.queryState.sort,
            page: state.queryState.page,
            page_size: state.queryState.pageSize,
          });
          items = data.items ?? [];
          total = data.total ?? 0;
        }

        // Discard if a newer query has been issued
        if (cancelled || queryKeyRef.current !== currentKey) return;

        dispatch({
          type: "RESULTS_SUCCESS",
          payload: {
            items,
            total,
            totalApprox:
              total > 0 &&
              items.length === state.queryState.pageSize,
          },
        });
      } catch (err) {
        if (cancelled || queryKeyRef.current !== currentKey) return;

        // Auto-fallback: if keyword mode (OpenSearch) fails, retry with Postgres
        if (state.queryState.searchMode === "keyword") {
          try {
            const fallbackData = await assetsApi.list({
              filter: filters.length > 0 ? filters : undefined,
              sort_by: state.queryState.sort,
              page: state.queryState.page,
              page_size: state.queryState.pageSize,
            });
            if (cancelled || queryKeyRef.current !== currentKey) return;
            const fallbackItems = fallbackData.items ?? [];
            const fallbackTotal = fallbackData.total ?? 0;
            dispatch({
              type: "RESULTS_SUCCESS",
              payload: {
                items: fallbackItems,
                total: fallbackTotal,
                totalApprox:
                  fallbackTotal > 0 &&
                  fallbackItems.length === state.queryState.pageSize,
              },
            });
            // Show degradation warning via error field
            dispatch({
              type: "RESULTS_ERROR",
              payload: {
                error: "OpenSearch 不可用，已降级到 Postgres 查询",
              },
            });
            return;
          } catch {
            // Fallback also failed, show original error
          }
        }

        dispatch({
          type: "RESULTS_ERROR",
          payload: {
            error: err instanceof Error ? err.message : "Failed to fetch assets",
          },
        });
      }
    };

    fetchResults();

    return () => {
      cancelled = true;
    };
  }, [state.resultsState.isStale, state.queryState, state.routerState.urlHydrated]);

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
