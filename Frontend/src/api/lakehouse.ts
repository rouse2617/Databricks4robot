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

export const lakehouseApi = {
  report: () => apiClient.get<LakehouseReport>("/lakehouse/report").then((r) => r.data),
  status: () => apiClient.get<LakehouseStatus>("/lakehouse/status").then((r) => r.data),
  tables: () => apiClient.get<LakehouseItemsResponse<LakehouseTableCount>>("/lakehouse/tables").then((r) => r.data),
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
