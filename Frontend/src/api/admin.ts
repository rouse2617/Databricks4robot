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

export const adminApi = {
  reindex: (dryRun: boolean) =>
    apiClient
      .post<ReindexResult>("/admin/search/reindex", {
        dry_run: dryRun,
      })
      .then((r) => r.data),
};
