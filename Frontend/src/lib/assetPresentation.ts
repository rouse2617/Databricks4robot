import type { Asset } from "../api/types";

type AssetLike = Pick<
	Asset,
	| "duration_ms"
	| "duration_sec"
	| "lifecycle_state"
	| "status"
	| "asset_type"
	| "type"
>;

export function getDurationMs(asset: AssetLike): number | null {
	if (
		typeof asset.duration_ms === "number" &&
		!Number.isNaN(asset.duration_ms)
	) {
		return asset.duration_ms;
	}
	if (
		typeof asset.duration_sec === "number" &&
		!Number.isNaN(asset.duration_sec)
	) {
		return Math.round(asset.duration_sec * 1000);
	}
	return null;
}

export function formatDurationSeconds(asset: AssetLike, digits = 1): string {
	const durationMs = getDurationMs(asset);
	if (durationMs == null) return "—";
	return `${(durationMs / 1000).toFixed(digits)}s`;
}

export function getLifecycleState(asset: AssetLike): string {
	return asset.lifecycle_state || asset.status || "";
}

export function getAssetType(asset: AssetLike): string {
	return asset.asset_type || asset.type || "";
}

export function getAssetStateColor(asset: AssetLike): string {
	const state = getLifecycleState(asset);
	switch (state) {
		case "ready":
		case "approved":
			return "success";
		case "processing":
		case "running":
		case "delivered":
		case "pending":
			return "processing";
		case "rejected":
		case "failed":
			return "error";
		case "superseded":
			return "warning";
		default:
			return "default";
	}
}
