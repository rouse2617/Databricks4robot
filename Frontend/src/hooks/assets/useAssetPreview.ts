// ─── useAssetPreview — Preview fetch hook ───
// Thin wrapper that encapsulates the preview-fetch side-effect.
// The actual preview fetch logic already lives inside useAssetsDiscoveryReducer's
// useEffect. This hook re-exports the buildPlaceholderPreviewManifest helper and
// provides a standalone hook for contexts that need preview fetching outside the
// full discovery reducer (e.g. detail page).
// Validates: Requirements R7

import { useEffect, useRef } from "react";
import { assetsApi } from "../../api/assets";
import type { Asset } from "../../api/types";
import type {
  PreviewManifest,
  PreviewAvailability,
  PreviewMode,
} from "../../lib/assets/assetsDiscoveryTypes";
import type { AssetsDiscoveryAction } from "../../lib/assets/assetsDiscoveryActions";

// ─── Helpers ───

/**
 * Build a placeholder PreviewManifest from an asset's files map.
 * Checks for thumbnail and preview_mp4 keys to determine availability.
 */
export function buildPlaceholderPreviewManifest(asset: Asset): PreviewManifest {
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

/**
 * Standalone preview-fetch hook.
 *
 * Accepts `activeAssetId` and a `dispatch` function. When `activeAssetId`
 * changes to a non-null value, fetches the full asset via `assetsApi.get(id)`,
 * builds a placeholder manifest, and dispatches RECEIVE_PREVIEW_SUCCESS.
 *
 * Short-circuits if the same asset is already loaded (tracked via ref).
 *
 * NOTE: The main discovery page uses the preview effect built into
 * useAssetsDiscoveryReducer. This hook is useful for standalone contexts
 * or can be called from the reducer hook as a delegate.
 */
export function useAssetPreview(
  activeAssetId: string | null,
  dispatch: React.Dispatch<AssetsDiscoveryAction>,
): void {
  const prevIdRef = useRef<string | null>(null);

  useEffect(() => {
    // Short-circuit if same asset is already loaded
    if (activeAssetId === prevIdRef.current) return;
    prevIdRef.current = activeAssetId;

    // Nothing to fetch
    if (!activeAssetId) return;

    let cancelled = false;

    const fetchPreview = async () => {
      dispatch({ type: "PREVIEW_LOADING" });

      try {
        const asset = await assetsApi.get(activeAssetId);

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
  }, [activeAssetId, dispatch]);
}
