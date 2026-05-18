// ─── useAssetPreview — Preview fetch hook ───
// Thin wrapper that encapsulates the preview-fetch side-effect.
// The actual preview fetch logic already lives inside useAssetsDiscoveryReducer's
// useEffect. This hook re-exports the buildPlaceholderPreviewManifest helper and
// provides a standalone hook for contexts that need preview fetching outside the
// full discovery reducer (e.g. detail page).
// Validates: Requirements R7

import { useEffect, useRef } from "react";
import type { FoxgloveSourceResponse } from "../../api/assets";
import { assetsApi } from "../../api/assets";
import type { Asset } from "../../api/types";
import type { AssetsDiscoveryAction } from "../../lib/assets/assetsDiscoveryActions";
import type {
	PreviewAvailability,
	PreviewManifest,
	PreviewMode,
	PreviewSourceOption,
} from "../../lib/assets/assetsDiscoveryTypes";

// ─── Helpers ───

export interface PreviewManifestOptions {
	previewTopic?: string; // kept for compatibility with existing callers
	previewSourceId?: string; // kept for compatibility with existing callers
}

function readCookieValue(name: string): string | null {
	if (typeof document === "undefined" || !document.cookie) {
		return null;
	}
	for (const part of document.cookie.split(";")) {
		const [k, ...rest] = part.trim().split("=");
		if (k !== name) continue;
		const raw = rest.join("=");
		if (!raw) return null;
		try {
			return decodeURIComponent(raw);
		} catch {
			return raw;
		}
	}
	return null;
}

function buildSegmentPreviewUrl(
	assetId: string,
	opts?: PreviewManifestOptions,
): string {
	const path = `/api/v1/preview/assets/${encodeURIComponent(assetId)}/segment.mp4`;
	const q = new URLSearchParams();
	if (opts?.previewTopic) {
		q.set("topic", opts.previewTopic);
	}
	const graceToken = readCookieValue("grace_session");
	if (graceToken) {
		// <video> cannot set X-Grace-Token; backend supports query fallback.
		q.set("grace_token", graceToken);
	}
	const qs = q.toString();
	return qs ? `${path}?${qs}` : path;
}

/**
 * Reads `hints.window.start_timestamp_ns` / `end_timestamp_ns`.
 * Contract: nanoseconds MUST use the **same epoch as MCAP record `log_time`**
 * (POSIX time from epoch is usual). Segment-relative timestamps will break overlap
 * with indexed chunk spans in the embedded player until data is migrated.
 */
function parseWindowSecFromHints(
	foxgloveSource?: FoxgloveSourceResponse | null,
): { windowStartSec: number | null; windowEndSec: number | null } {
	const hints = foxgloveSource?.hints;
	if (!hints || typeof hints !== "object") {
		return { windowStartSec: null, windowEndSec: null };
	}
	const windowValue = (hints as Record<string, unknown>).window;
	if (!windowValue || typeof windowValue !== "object") {
		return { windowStartSec: null, windowEndSec: null };
	}
	const win = windowValue as Record<string, unknown>;
	const startNsRaw = win.start_timestamp_ns;
	const endNsRaw = win.end_timestamp_ns;
	const parseNs = (v: unknown): number | null => {
		if (typeof v === "number" && Number.isFinite(v)) return v;
		if (typeof v === "string" && v.trim().length > 0) {
			const n = Number(v);
			if (Number.isFinite(n)) return n;
		}
		return null;
	};
	const startNs = parseNs(startNsRaw);
	const endNs = parseNs(endNsRaw);
	return {
		windowStartSec: startNs != null ? startNs / 1_000_000_000 : null,
		windowEndSec: endNs != null ? endNs / 1_000_000_000 : null,
	};
}

/**
 * Build a placeholder PreviewManifest from asset metadata.
 * In direct-MCAP architecture we only expose thumbnail/none fallback here.
 */
export function buildPlaceholderPreviewManifest(
	asset: Asset,
	_opts?: PreviewManifestOptions,
): PreviewManifest {
	const hasThumbnail = !!asset.files?.thumbnail;
	const availability: PreviewAvailability = hasThumbnail ? "ready" : "missing";
	const mode: PreviewMode = hasThumbnail ? "thumbnail" : "none";

	return {
		thumbnailUrl: hasThumbnail ? asset.files.thumbnail : null,
		previewVideoUrl: null,
		availability,
		mode,
	};
}

export function buildPreviewManifestFromSources(
	asset: Asset,
	previewManifest: unknown,
	foxgloveSource?: FoxgloveSourceResponse | null,
	opts?: PreviewManifestOptions,
): PreviewManifest {
	const fallback = buildPlaceholderPreviewManifest(asset, opts);
	if (foxgloveSource?.ds_params?.url) {
		const sourceParams = foxgloveSource.ds_params ?? {};
		const { windowStartSec, windowEndSec } =
			parseWindowSecFromHints(foxgloveSource);
		const pm = (previewManifest ?? {}) as {
			sources?: Array<{
				id: string;
				kind: string;
				codec?: string;
				topic?: string;
				url?: string;
			}>;
			recommended_source_id?: string;
		};
		const sourceOptions: PreviewSourceOption[] = (pm.sources ?? []).map(
			(s) => ({
				id: s.id,
				label: s.topic ? `视频: ${s.topic}` : s.id,
				kind: s.kind,
				codec: s.codec,
				topic: s.topic,
			}),
		);
		const selectedSourceId =
			opts?.previewSourceId ??
			pm.recommended_source_id ??
			sourceOptions[0]?.id ??
			null;
		const selectedSource =
			(pm.sources ?? []).find((s) => s.id === selectedSourceId) ?? null;
		const dsParams = {
			...sourceParams,
			asset_id: asset.asset_id,
		};
		return {
			...fallback,
			availability: "ready",
			mode: "mcap",
			previewVideoUrl:
				selectedSource?.url ?? buildSegmentPreviewUrl(asset.asset_id, opts),
			mcapUrl: sourceParams.url,
			sourceId: foxgloveSource.source_id,
			ds: foxgloveSource.ds,
			dsParams,
			windowStartSec,
			windowEndSec,
			sources: sourceOptions,
			recommendedSourceId: pm.recommended_source_id ?? undefined,
			activeSourceId: selectedSourceId,
		};
	}
	return fallback;
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

				const foxgloveSource = await assetsApi
					.getFoxgloveSource(activeAssetId)
					.catch(() => null);
				const manifest = buildPreviewManifestFromSources(
					asset,
					null,
					foxgloveSource,
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
	}, [activeAssetId, dispatch]);
}
