import { apiClient } from "./client";

export interface ReindexResult {
  dry_run: boolean;
  total_assets: number;
  indexed: number;
  failed: number;
  duration_ms: number;
  errors?: string[];
}

export const adminApi = {
  reindex: (dryRun: boolean) =>
    apiClient
      .post<ReindexResult>("/admin/search/reindex", null, {
        params: { dry_run: dryRun },
      })
      .then((r) => r.data),
};
