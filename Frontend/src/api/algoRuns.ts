import { apiClient } from "./client";

export interface AlgoRun {
	run_id: string;
	algo_name: string;
	algo_version: string;
	algo_kind: string;
	triggered_by: string;
	status: string;
	started_at?: string;
	finished_at?: string;
	assets_processed?: number;
	assets_succeeded?: number;
	assets_failed?: number;
	created_at?: string;
}

export const algoRunsApi = {
	get: (runId: string) =>
		apiClient.get<AlgoRun>(`/algo-runs/${runId}`).then((r) => r.data),
};
