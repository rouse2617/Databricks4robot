import { apiClient } from "./client";

export interface LakehouseTableCount {
  table_name: string;
  row_count: number;
}

export interface LakehouseQuestionResult {
  question: string;
  answer: string;
  [key: string]: unknown;
}

export interface LakehouseReport {
  generated_at: string;
  sync_mode: string;
  sync_note: string;
  tables: LakehouseTableCount[];
  questions: Record<string, LakehouseQuestionResult>;
}

export interface LakehouseStatus {
  enabled: boolean;
  healthy: boolean;
  catalog?: string;
  schema?: string;
  error?: string;
}

export interface LakehouseItemsResponse<T = Record<string, unknown>> {
  items: T[];
  [key: string]: unknown;
}

export interface SyncStatusData {
  dagster_run_id: string;
  checked_at: string;
  pg_total_count: number;
  iceberg_total_count: number;
  count_diff_pct: number;
  pg_status_dist: Record<string, number>;
  iceberg_status_dist: Record<string, number>;
  status_diff: Record<string, { pg: number; iceberg: number; diff: number }>;
  is_alert: boolean;
}

export interface SyncStatusResponse {
  available: boolean;
  message?: string;
  data?: SyncStatusData;
}

export const lakehouseApi = {
  report: () => apiClient.get<LakehouseReport>("/lakehouse/report").then((r) => r.data),
  status: () => apiClient.get<LakehouseStatus>("/lakehouse/status").then((r) => r.data),
  tables: () => apiClient.get<LakehouseItemsResponse<LakehouseTableCount>>("/lakehouse/tables").then((r) => r.data),
  syncStatus: () => apiClient.get<SyncStatusResponse>("/lakehouse/sync-status").then((r) => r.data),
  trainingAssets: (snapshotId = "mvp_hand_tracking_quality_v1") =>
    apiClient
      .get<LakehouseItemsResponse>("/lakehouse/training-assets", { params: { snapshot_id: snapshotId } })
      .then((r) => r.data),
  recomputeCandidates: (algoKey = "hand_tracking@1.2.0", targetVersion = "next") =>
    apiClient
      .get<LakehouseItemsResponse>("/lakehouse/recompute-candidates", {
        params: { algo_key: algoKey, target_version: targetVersion },
      })
      .then((r) => r.data),
  tagTimeline: (tagKey = "quality") =>
    apiClient
      .get<LakehouseItemsResponse>("/lakehouse/tag-timeline", { params: { tag_key: tagKey } })
      .then((r) => r.data),
  qualityDistribution: (window = "30d") =>
    apiClient
      .get<LakehouseItemsResponse>("/lakehouse/quality-distribution", { params: { window } })
      .then((r) => r.data),
  customerReplay: (customerId = "urn:grace:customer:A") =>
    apiClient
      .get<LakehouseItemsResponse>("/lakehouse/customer-replay", { params: { customer_id: customerId } })
      .then((r) => r.data),
};
