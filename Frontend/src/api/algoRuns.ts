import { apiClient } from "./client";
import type { PaginatedResponse } from "./types";

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
	external_url?: string;
}

export interface ListAlgoRunsParams {
	page?: number;
	page_size?: number;
	algo_name?: string;
	status?: string;
	started_after?: string;
	started_before?: string;
}

export const algoRunsApi = {
	list: (params?: ListAlgoRunsParams) => {
		const sp = new URLSearchParams();
		if (params?.page) sp.set("page", String(params.page));
		if (params?.page_size) sp.set("page_size", String(params.page_size));
		if (params?.algo_name) sp.set("algo_name", params.algo_name);
		if (params?.status) sp.set("status", params.status);
		if (params?.started_after) sp.set("started_after", params.started_after);
		if (params?.started_before) sp.set("started_before", params.started_before);
		return apiClient
			.get<PaginatedResponse<AlgoRun>>(`/algo-runs?${sp.toString()}`)
			.then((r) => r.data);
	},

	get: (runId: string) =>
		apiClient.get<AlgoRun>(`/algo-runs/${runId}`).then((r) => r.data),
};
