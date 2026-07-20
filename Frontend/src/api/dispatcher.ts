import { apiClient } from "./client";

// CYB-3679 — dispatcher online tuning.

export interface DispatcherConfig {
	cluster_id: string;
	max_concurrency: number;
	submit_batch: number;
	rate_per_sec: number;
	paused: boolean;
	updated_by: string;
	updated_at: string;
}

export interface DispatcherClusterStatus {
	cluster_id: string;
	config: DispatcherConfig;
	has_row: boolean;
	effective_concurrency: number;
	suppression_reason?: "" | "paused" | "aimd_backoff";
}

export interface SaveDispatcherConfigRequest {
	max_concurrency: number;
	submit_batch: number;
	rate_per_sec: number;
	paused: boolean;
}

export const dispatcherApi = {
	list: () =>
		apiClient
			.get("/dispatcher/clusters")
			.then((r) => (r.data?.clusters ?? []) as DispatcherClusterStatus[]),
	save: (cluster: string, req: SaveDispatcherConfigRequest) =>
		apiClient
			.put(`/dispatcher/clusters/${encodeURIComponent(cluster)}`, req)
			.then((r) => r.data),
	remove: (cluster: string) =>
		apiClient
			.delete(`/dispatcher/clusters/${encodeURIComponent(cluster)}`)
			.then((r) => r.data),
};
