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
	iceberg_max_seq?: number;
}

export interface SyncStatusResponse {
	available: boolean;
	source?: "realtime" | "sync_reconciliation";
	message?: string;
	data?: SyncStatusData;
}

export interface BronzeSyncProgressResponse {
	outbox_published_max_seq: number;
	bronze_max_event_seq: number;
	bronze_lag_events: number;
	bronze_last_ingested_at?: string | null;
	bronze_stale_seconds: number;
	checked_at: string;
}

export interface LakehouseOverviewResponse {
	bronze_event_rows?: number;
	bronze_rows?: number;
	silver_asset_rows?: number;
	silver_rows?: number;
	gold_latest_date?: string | null;
	gold_row_count?: number;
	asset_total?: number;
	today_new_assets?: number;
	yesterday_new_assets?: number;
	week_new_assets?: number;
	day7_avg_new_assets?: number;
	data_lag_hours?: number;
	[key: string]: unknown;
}

export interface LakehouseAssetGrowthItem {
	event_date?: string;
	new_assets?: number;
	cumulative_assets?: number;
	[key: string]: unknown;
}

export interface LakehouseAssetGrowthResponse {
	days?: number;
	items?: LakehouseAssetGrowthItem[];
	[key: string]: unknown;
}

export interface LakehouseEventDailyItem {
	date?: string;
	event_date?: string;
	event_type?: string;
	asset_count?: number;
	count?: number;
	[key: string]: unknown;
}

export interface LakehouseEventDailyResponse {
	days?: number;
	items?: LakehouseEventDailyItem[];
	[key: string]: unknown;
}

export interface LakehouseEventTypeShareItem {
	event_type?: string;
	asset_count?: number;
	count?: number;
	ratio?: number;
	[key: string]: unknown;
}

export interface LakehouseEventTypeShareResponse {
	date?: string;
	items?: LakehouseEventTypeShareItem[];
	[key: string]: unknown;
}

export interface FailureClusterItem {
	failure_mode: string;
	algo_name: string;
	affected_assets: number;
	ratio: number;
}

export interface FailureClustersResponse {
	days: number;
	items: FailureClusterItem[];
}

export const lakehouseApi = {
	report: () =>
		apiClient.get<LakehouseReport>("/lakehouse/report").then((r) => r.data),
	status: () =>
		apiClient.get<LakehouseStatus>("/lakehouse/status").then((r) => r.data),
	tables: () =>
		apiClient
			.get<LakehouseItemsResponse<LakehouseTableCount>>("/lakehouse/tables")
			.then((r) => r.data),
	syncStatus: () =>
		apiClient
			.get<SyncStatusResponse>("/lakehouse/sync-status")
			.then((r) => r.data),
	syncProgress: () =>
		apiClient
			.get<BronzeSyncProgressResponse>("/lakehouse/sync-progress")
			.then((r) => r.data),
	overview: () =>
		apiClient
			.get<LakehouseOverviewResponse>("/lakehouse/overview")
			.then((r) => r.data),
	eventDaily: (days = 14) =>
		apiClient
			.get<LakehouseEventDailyResponse>("/lakehouse/event-daily", {
				params: { days },
			})
			.then((r) => r.data),
	assetGrowth: (days = 30) =>
		apiClient
			.get<LakehouseAssetGrowthResponse>("/lakehouse/asset-growth", {
				params: { days },
			})
			.then((r) => r.data),
	eventTypeShare: (date: string) =>
		apiClient
			.get<LakehouseEventTypeShareResponse>("/lakehouse/event-type-share", {
				params: { date },
			})
			.then((r) => r.data),
	trainingAssets: (snapshotId = "mvp_hand_tracking_quality_v1") =>
		apiClient
			.get<LakehouseItemsResponse>("/lakehouse/training-assets", {
				params: { snapshot_id: snapshotId },
			})
			.then((r) => r.data),
	recomputeCandidates: (
		algoKey = "hand_tracking@1.2.0",
		targetVersion = "next",
	) =>
		apiClient
			.get<LakehouseItemsResponse>("/lakehouse/recompute-candidates", {
				params: { algo_key: algoKey, target_version: targetVersion },
			})
			.then((r) => r.data),
	tagTimeline: (tagKey = "quality") =>
		apiClient
			.get<LakehouseItemsResponse>("/lakehouse/tag-timeline", {
				params: { tag_key: tagKey },
			})
			.then((r) => r.data),
	qualityDistribution: (window = "30d") =>
		apiClient
			.get<LakehouseItemsResponse>("/lakehouse/quality-distribution", {
				params: { window },
			})
			.then((r) => r.data),
	customerReplay: (customerId = "urn:grace:customer:A") =>
		apiClient
			.get<LakehouseItemsResponse>("/lakehouse/customer-replay", {
				params: { customer_id: customerId },
			})
			.then((r) => r.data),
	failureClusters: (days = 7) =>
		apiClient
			.get<FailureClustersResponse>("/lakehouse/failure-clusters", {
				params: { days },
			})
			.then((r) => r.data),
};
