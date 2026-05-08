import { apiClient } from "./client";

export interface EvalResultItem {
	eval_result_id: string;
	asset_id: string;
	eval_name: string;
	eval_version: string;
	status: string;
	result_payload?: Record<string, unknown>;
	run_id?: string;
	source_type?: string;
	source_name?: string;
	created_at: string;
}

export interface AssetMetricItem {
	asset_id: string;
	target_type: string;
	target_id: string;
	metric_key: string;
	metric_type: string;
	metric_value?: number | null;
	metric_value_int?: number | null;
	metric_value_text?: string | null;
	metric_value_bool?: boolean | null;
	eval_name: string;
	eval_version: string;
	source_type?: string;
	source_name?: string;
	updated_at: string;
}

export interface MetricsSearchFilter {
	metric_key: string;
	op: "eq" | "gt" | "gte" | "lt" | "lte";
	value: number;
}

export interface MetricsSearchResponse {
	asset_ids: string[];
	total: number;
	page: number;
	page_size: number;
}

export const evalApi = {
	listEvalResults: (assetId: string, limit = 20) =>
		apiClient
			.get<{ items: EvalResultItem[]; total: number }>(
				`/assets/${assetId}/eval-results`,
				{ params: { limit } },
			)
			.then((r) => r.data),

	listMetrics: (assetId: string) =>
		apiClient
			.get<{ items: AssetMetricItem[]; asset_id: string }>(
				`/assets/${assetId}/metrics`,
			)
			.then((r) => r.data),

	searchByMetrics: (
		filters: MetricsSearchFilter[],
		lifecycleState?: string,
		page = 1,
		pageSize = 20,
	) =>
		apiClient
			.post<MetricsSearchResponse>("/metrics:search", {
				filters: {
					lifecycle_state: lifecycleState ?? "",
					metrics: filters,
				},
				page,
				page_size: pageSize,
			})
			.then((r) => r.data),
};
