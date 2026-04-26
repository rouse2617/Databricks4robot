// ─── Shared API types aligned with backend data model ───

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}

// ─── Asset ───
export interface Asset {
  asset_id: string;
  mcap_file_id: string;
  start_timestamp_ns: number;
  end_timestamp_ns: number;
  duration_sec: number;
  reviewer: string;
  status: string;
  owner: string;
  type?: string;
  env?: string;
  task?: string;
  last_delivered_at?: string;
  last_delivered_to?: string;
  delivery_count: number;
  algo_results: Record<string, string>;
  tags: Record<string, string>;
  files: Record<string, string>;
  lifecycle_meta: Record<string, any>;
  created_at: string;
  updated_at: string;
  version: number;
}

// ─── MCAP File ───
export interface McapFile {
  mcap_file_id: string;
  gcs_path: string;
  size_bytes: number;
  raw_hash_md5: string;
  ingest_state: string;
  start_timestamp_ns?: number;
  end_timestamp_ns?: number;
  channel_count?: number;
  chunk_count?: number;
  owner: string;
  process_state?: Record<string, string>;
  created_at: string;
  updated_at: string;
  version: number;
}

// ─── Delivery ───
export interface Delivery {
  delivery_id: string;
  customer_id: string;
  status: string;
  delivered_at?: string;
  manifest_uri?: string;
  contract_id?: string;
  note?: string;
  asset_count: number;
  owner: string;
  created_at: string;
  updated_at: string;
  version: number;
}

// ─── Delivery Item ───
export interface DeliveryItem {
  delivery_id: string;
  asset_id: string;
  created_at: string;
}

// ─── Algo Event ───
export interface AlgoEvent {
  event_id: string;
  asset_id: string;
  algo_key: string;
  prev_status?: string;
  new_status: string;
  run_id?: string;
  reason?: string;
  created_at: string;
}

// ─── Algo status helpers ───
export type AlgoStatus = "blocked" | "pending" | "running" | "ok" | "failed";

export interface AlgoInfo {
  algoKey: string;
  name: string;
  version: string;
  status: AlgoStatus;
  startedAt?: string;
  finishedAt?: string;
  method?: string;
  runId?: string;
  outputUri?: string;
  reason?: string;
}

// ─── Dashboard stats ───
export interface PlatformStats {
  totalAssets: number;
  totalMcapFiles: number;
  totalDeliveries: number;
  assetsByStatus: Record<string, number>;
  algoStatusSummary: Record<string, Record<AlgoStatus, number>>;
  recentActivity: ActivityItem[];
}

export interface ActivityItem {
  type: "asset_created" | "algo_finished" | "delivery_committed";
  id: string;
  description: string;
  timestamp: string;
}
