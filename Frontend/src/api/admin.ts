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

	auditSearch: () =>
		apiClient.get<SearchAuditResult>("/admin/search/audit").then((r) => r.data),
};
