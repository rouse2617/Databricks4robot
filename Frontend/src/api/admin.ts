import { apiClient } from "./client";

export interface ReindexResult {
	dry_run: boolean;
	total_assets: number;
	indexed: number;
	deleted: number;
	failed: number;
	duration_ms: number;
	errors?: string[];
	elasticsearch_doc_count?: number;
}

export type ReindexJobStatus =
	| "queued"
	| "running"
	| "paused"
	| "succeeded"
	| "failed";

export interface ReindexJob {
	id: string;
	status: ReindexJobStatus;
	dry_run: boolean;
	page_size: number;
	next_page: number;
	stop_requested: boolean;
	total_assets: number;
	assets_scanned: number;
	documents_indexed: number;
	documents_deleted: number;
	failed: number;
	error?: string;
	error_samples?: string[];
	elasticsearch_doc_count?: number;
	progress_pct: number;
	created_at: string;
	updated_at: string;
	started_at?: string;
	finished_at?: string;
}

export interface ReindexJobListResponse {
	items: ReindexJob[];
}

export interface SearchAuditResult {
	pg_assets: number;
	elasticsearch_docs: number;
	missing_in_elasticsearch: number;
	orphan_in_elasticsearch: number;
	consistency: number;
	target: number;
	sample_missing_ids?: string[];
	sample_orphan_ids?: string[];
	duration_ms: number;
}

export const adminApi = {
	reindex: (dryRun: boolean) =>
		apiClient
			.post<ReindexResult>("/admin/search/reindex", {
				dry_run: dryRun,
			})
			.then((r) => r.data),

	createReindexJob: (dryRun: boolean, pageSize = 200) =>
		apiClient
			.post<ReindexJob>("/admin/search/reindex-jobs", {
				dry_run: dryRun,
				page_size: pageSize,
			})
			.then((r) => r.data),

	getReindexJob: (jobID: string) =>
		apiClient
			.get<ReindexJob>(`/admin/search/reindex-jobs/${jobID}`)
			.then((r) => r.data),

	listReindexJobs: (limit = 20) =>
		apiClient
			.get<ReindexJobListResponse>("/admin/search/reindex-jobs", {
				params: { limit },
			})
			.then((r) => r.data.items),

	stopReindexJob: (jobID: string) =>
		apiClient
			.post<ReindexJob>(`/admin/search/reindex-jobs/${jobID}/stop`)
			.then((r) => r.data),

	resumeReindexJob: (jobID: string) =>
		apiClient
			.post<ReindexJob>(`/admin/search/reindex-jobs/${jobID}/resume`)
			.then((r) => r.data),

	auditSearch: () =>
		apiClient.get<SearchAuditResult>("/admin/search/audit").then((r) => r.data),
};
