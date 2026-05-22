// ─── Shared API types aligned with backend data model ───

export interface PaginatedResponse<T> {
	items: T[];
	total: number;
	page: number;
	page_size: number;
}

// ─── Asset ───
/** One row of `asset_tags`. Multi-source enabled by CYB-1015. */
export interface AssetTagDetail {
	tag_key: string;
	tag_value: string;
	tag_type?: string;
	source_type: string;
	source_name?: string;
	source_version?: string;
	run_id?: string;
	applied_at?: string;
}

export interface Asset {
	asset_id: string;
	mcap_file_id: string;
	start_timestamp_ns: number;
	end_timestamp_ns: number;
	duration_sec?: number;
	duration_ms?: number;
	reviewer: string;
	status?: string;
	lifecycle_state?: string;
	owner: string;
	type?: string;
	asset_type?: string;
	env?: string;
	task?: string;
	segment_locator?: string;
	last_delivered_at?: string;
	last_delivered_to?: string;
	delivery_count: number;
	algo_results: Record<string, string>;
	tags: Record<string, string>;
	tags_detailed?: AssetTagDetail[];
	files: Record<string, string>;
	thumb_uri?: string;
	storage_uri?: string;
	parent_asset_id?: string;
	root_asset_id?: string;
	lifecycle_meta: Record<string, unknown>;
	retention_tier?: string;
	expire_at?: string | null;
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

export interface AssetEvent {
	event_id: string;
	event_seq: number;
	event_type: string;
	payload_schema_version?: string;
	asset_id: string;
	mcap_file_id?: string;
	event_source?: string;
	request_id?: string;
	publish_state?: string;
	event_payload?: Record<string, unknown>;
	created_at: string;
	occurred_at?: string;
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
