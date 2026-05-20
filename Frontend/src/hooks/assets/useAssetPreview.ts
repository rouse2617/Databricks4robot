// ─── useAssetPreview — Preview fetch hook ───
// `fetchPreviewBundle` is the single entry point: it loads asset + foxglove
// source + mcap-preview manifest in parallel and assembles the UI-facing
// `PreviewManifest`. Both the discovery reducer and the detail page call it
// directly; `useAssetPreview` is a thin standalone hook for callers that just
// want "load preview for assetId" with React lifecycle handling.
// Validates: Requirements R7

import { useEffect, useRef } from "react";
import type {
	FoxgloveSourceResponse,
	PreviewManifestResponse,
} from "../../api/assets";
import { assetsApi } from "../../api/assets";
import type { Asset } from "../../api/types";
import type { AssetsDiscoveryAction } from "../../lib/assets/assetsDiscoveryActions";
import type {
	PreviewAvailability,
	PreviewManifest,
	PreviewMode,
	PreviewSourceOption,
} from "../../lib/assets/assetsDiscoveryTypes";
import { buildPreviewErrorMessage } from "../../lib/assets/previewErrors";
import {
	activePreviewSource,
	buildSegmentPreviewUrl,
	enrichSegmentPreviewUrl,
	formatPreviewCodecLabel,
	isHevcPreviewCodec,
	parseWindowNsFromHints,
} from "../../lib/assets/previewSegmentUrl";

// ─── Helpers ───

export interface PreviewManifestOptions {
	// Topic override; mostly used by legacy callers — modern flows pass
	// `previewSourceId` and let manifest resolution pick the topic.
	previewTopic?: string;
	// Requested manifest source id; falls back to recommended/first when
	// missing or invalid (see `pickPreviewSourceId`).
	previewSourceId?: string;
}

export { buildPreviewErrorMessage };

/**
 * Reads `hints.window.start_timestamp_ns` / `end_timestamp_ns` (seconds for UI).
 */
function parseWindowSecFromHints(foxgloveSource?: FoxgloveSourceResponse | null): {
	windowStartSec: number | null;
	windowEndSec: number | null;
} {
	const { startNs, endNs } = parseWindowNsFromHints(foxgloveSource);
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

/** Resolved active source id from a built preview manifest. */
export function resolvedPreviewSourceId(
	manifest: PreviewManifest,
): string | null {
	return (
		manifest.activeSourceId ??
		manifest.recommendedSourceId ??
		manifest.sources?.[0]?.id ??
		null
	);
}

/** Pick a manifest source id that exists, else recommended, else first. */
export function pickPreviewSourceId(
	sourceOptions: PreviewSourceOption[],
	previewManifest: PreviewManifestResponse | null | undefined,
	requestedId?: string | null,
): string | null {
	const validIds = new Set(sourceOptions.map((s) => s.id));
	const requested = requestedId?.trim() || null;
	if (requested && validIds.has(requested)) {
		return requested;
	}
	const recommended = previewManifest?.recommended_source_id ?? null;
	if (recommended && validIds.has(recommended)) {
		return recommended;
	}
	return sourceOptions[0]?.id ?? null;
}

function mapManifestSources(
	pm: PreviewManifestResponse | null | undefined,
): PreviewSourceOption[] {
	return (pm?.sources ?? []).map((s) => {
		const codecLabel = formatPreviewCodecLabel(s.codec);
		const label = s.topic
			? codecLabel
				? `视频 (${codecLabel}): ${s.topic}`
				: `视频: ${s.topic}`
			: s.id;
		return {
			id: s.id,
			label,
			kind: s.kind,
			codec: s.codec,
			topic: s.topic,
		};
	});
}

export function buildPreviewManifestFromSources(
	asset: Asset,
	previewManifest: PreviewManifestResponse | null | undefined,
	foxgloveSource?: FoxgloveSourceResponse | null,
	opts?: PreviewManifestOptions,
): PreviewManifest {
	const fallback = buildPlaceholderPreviewManifest(asset, opts);
	if (foxgloveSource?.ds_params?.url) {
		const sourceParams = foxgloveSource.ds_params ?? {};
		const { windowStartSec, windowEndSec } =
			parseWindowSecFromHints(foxgloveSource);
		const { startNs, endNs } = parseWindowNsFromHints(foxgloveSource);
		const sourceOptions = mapManifestSources(previewManifest);
		const selectedSourceId = pickPreviewSourceId(
			sourceOptions,
			previewManifest,
			opts?.previewSourceId,
		);
		const selectedManifestSource =
			(previewManifest?.sources ?? []).find((s) => s.id === selectedSourceId) ??
			null;
		const topic =
			opts?.previewTopic ??
			selectedManifestSource?.topic ??
			undefined;
		const baseVideoUrl =
			selectedManifestSource?.url ??
			buildSegmentPreviewUrl(asset.asset_id, { topic, startNs, endNs });
		const dsParams = {
			...sourceParams,
			asset_id: asset.asset_id,
		};
		return {
			...fallback,
			availability: "ready",
			mode: "mcap",
			previewVideoUrl: enrichSegmentPreviewUrl(baseVideoUrl, {
				topic,
				startNs,
				endNs,
			}),
			mcapUrl: sourceParams.url,
			sourceId: foxgloveSource.source_id,
			ds: foxgloveSource.ds,
			dsParams,
			windowStartSec,
			windowEndSec,
			sources: sourceOptions,
			recommendedSourceId: previewManifest?.recommended_source_id ?? undefined,
			activeSourceId: selectedSourceId,
		};
	}
	return fallback;
}

/** Fire-and-forget HEVC transcode prewarm when the active source needs it. */
export async function maybePrewarmPreview(
	assetId: string,
	manifest: PreviewManifest,
): Promise<void> {
	const active = activePreviewSource(manifest);
	if (!isHevcPreviewCodec(active?.codec)) {
		return;
	}
	await assetsApi
		.prewarmPreview(assetId, active?.topic ? { topic: active.topic } : undefined)
		.catch(() => {
			// Best-effort; playback still works without prewarm.
		});
}

export async function fetchPreviewBundle(
	assetId: string,
	opts?: PreviewManifestOptions,
): Promise<{
	asset: Asset;
	foxgloveSource: FoxgloveSourceResponse | null;
	previewManifest: PreviewManifestResponse | null;
	manifest: PreviewManifest;
}> {
	const [asset, foxgloveSource, previewManifest] = await Promise.all([
		assetsApi.get(assetId),
		assetsApi.getFoxgloveSource(assetId).catch(() => null),
		assetsApi.getPreviewManifest(assetId).catch(() => null),
	]);
	const manifest = buildPreviewManifestFromSources(
		asset,
		previewManifest,
		foxgloveSource,
		opts,
	);
	if (manifest.mode === "mcap") {
		await maybePrewarmPreview(assetId, manifest);
	}
	return { asset, foxgloveSource, previewManifest, manifest };
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
	opts?: PreviewManifestOptions,
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
				const { asset, manifest } = await fetchPreviewBundle(
					activeAssetId,
					opts,
				);

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
	}, [activeAssetId, dispatch, opts?.previewSourceId, opts?.previewTopic]);
}
