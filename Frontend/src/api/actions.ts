import { apiClient } from "./client";

export type ActionSourceType = "human" | "algo" | "rule" | "system";

export interface Action {
	action_id: string;
	asset_id: string;
	start_ns: number;
	end_ns: number;
	action_index?: number | null;
	primary_label?: string;
	labels: string[];
	description?: string;
	attrs: Record<string, unknown>;
	source_type: ActionSourceType;
	source_name?: string;
	source_version?: string;
	run_id?: string;
	confidence?: number | null;
	external_id?: string;
	tenant_id?: string | null;
	project_id?: string | null;
	is_deleted: boolean;
	version: number;
	created_at: string;
	updated_at: string;
}

export interface ActionListResponse {
	items: Action[];
	asset_id: string;
	total: number;
}

export interface ActionCreateInput {
	start_ns: number;
	end_ns: number;
	action_index?: number;
	primary_label?: string;
	labels?: string[];
	description?: string;
	attrs?: Record<string, unknown>;
	source_type?: ActionSourceType;
	source_name?: string;
	source_version?: string;
	run_id?: string;
	confidence?: number;
	external_id?: string;
}

export interface ActionListParams {
	/** Point-in-time ns; returns actions whose interval covers it. */
	at?: number;
	/** Lower bound (ns) of an overlap window. */
	from?: number;
	/** Upper bound (ns, exclusive) of an overlap window. */
	to?: number;
	/** Match primary_label OR membership in labels[]. */
	label?: string;
	limit?: number;
}

export const actionsApi = {
	list: (assetId: string, params: ActionListParams = {}) =>
		apiClient
			.get<ActionListResponse>(`/assets/${assetId}/action-annotations`, { params })
			.then((r) => r.data),

	create: (assetId: string, payload: ActionCreateInput) =>
		apiClient
			.post<Action>(`/assets/${assetId}/action-annotations`, payload)
			.then((r) => r.data),
};
