// CYB-4303: dashboard API client.
//
// Mirrors backend/internal/models/dashboard.go DurationDistribution shape
// exactly. hi_ms is nullable — the top bucket returns null so callers know
// it's open-ended.

import { apiClient } from "./client";

export interface DurationBucket {
	label: string;
	lo_ms: number;
	hi_ms: number | null;
	count: number;
	total_ms: number;
}

export interface DurationDistributionResponse {
	asset_type: string | null;
	buckets: DurationBucket[];
	total_assets: number;
	total_ms: number;
	mean_ms: number;
	min_ms: number;
	max_ms: number;
	p50_ms: number;
	p90_ms: number;
}

export const dashboardApi = {
	// assetType is optional. Empty/undefined → server returns the fleet-wide
	// aggregation across every asset_type.
	durationDistribution: (
		assetType?: string,
	): Promise<DurationDistributionResponse> => {
		const trimmed = (assetType ?? "").trim();
		return apiClient
			.get<DurationDistributionResponse>("/dashboard/duration-distribution", {
				params: trimmed ? { asset_type: trimmed } : undefined,
			})
			.then((r) => r.data);
	},
};
