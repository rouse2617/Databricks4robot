import { apiClient } from "./client";
import type { Asset } from "./types";

export interface SearchAssetsParams {
	q?: string;
	mode?: "keyword" | "semantic" | "similar";
	filter?: string[];
	page?: number;
	page_size?: number;
}

export interface SearchAggBucket {
	key: string;
	doc_count: number;
}

export interface SearchAlgoHit {
	name?: string;
	version?: string;
	status?: string;
	started_at?: string;
	finished_at?: string;
	method?: string;
	run_id?: string;
	output_uri?: string;
	reason?: string;
	result_tag?: string;
	result_score?: number | string;
}

export interface SearchTagHit {
	key?: string;
	value?: string | number | boolean | null;
}

export interface SearchAssetHit {
	asset_id?: string;
	mcap_file_id?: string;
	segment_locator?: string;
	asset_type?: string;
	lifecycle_state?: string;
	status?: string;
	version?: number;
	retention_tier?: string;
	owner?: string;
	reviewer?: string;
	start_timestamp_ns?: number;
	end_timestamp_ns?: number;
	duration_ms?: number;
	delivery_count?: number;
	expire_at?: string | null;
	created_at?: string;
	updated_at?: string;
	metadata?: Record<string, unknown>;
	tags_flat?: Record<string, unknown>;
	tags?: SearchTagHit[];
	algos?: SearchAlgoHit[];
	_highlight?: Record<string, string[]>;
}

export interface SearchAssetResult extends Asset {
	_highlight?: Record<string, string[]>;
}

export interface SearchAssetsResponse {
	items: SearchAssetResult[];
	total: number;
	page: number;
	page_size: number;
	aggregations?: Record<string, SearchAggBucket[]>;
}

/** GET /search/sync-status — how ES index is kept in sync (CDC vs dev reconciler). */
export interface SearchSyncStatusResponse {
	elasticsearch_ok: boolean;
	outbox_relay_enabled: boolean;
	outbox_es_subscriber_enabled: boolean;
	search_index_mode:
		| "unavailable"
		| "outbox_es_subscriber"
		| "local_reconcile"
		| "manual";
	env?: string;
	admin_search_enabled?: boolean;
}

/** GET /search/sync-progress — runtime PG→ES sync progress snapshot. */
export interface SearchSyncProgressResponse {
	postgres_assets_total: number;
	elasticsearch_docs_total: number;
	pg_es_gap: number;
	pg_es_sync_ratio: number;
	/** All `publish_state=pending` rows (includes safety-lag buffer). */
	outbox_pending_events: number;
	/** Pending rows old enough for the relay to claim now. */
	outbox_pending_claimable: number;
	/** Rows in `processing` (claimed, in-flight to ES). */
	outbox_processing_events: number;
	/** Configured relay safety lag in seconds (OUTBOX_RELAY_SAFETY_LAG_SEC). */
	outbox_relay_safety_lag_sec: number;
	oldest_pending_age_sec: number;
	/** MAX(event_seq) across asset_events (PG side watermark). */
	pg_max_event_seq?: number;
	/** MAX(event_seq) where publish_state='published' (handed off to MQ). */
	outbox_published_max_seq?: number;
	/** PG -> MQ backlog in event_seq space. */
	seq_lag?: number;
	/** MIN(applied_seq) across ES checkpoint shards (conservative watermark). */
	es_applied_min_seq?: number;
	/** MQ -> ES backlog in event_seq space. */
	consumer_lag?: number;
	checked_at: string;
}

function normalizeTagValue(value: unknown): string {
	if (typeof value === "string") return value;
	if (typeof value === "number" || typeof value === "boolean")
		return String(value);
	return "";
}

function buildTagsMap(hit: SearchAssetHit): Record<string, string> {
	const tags: Record<string, string> = {};

	for (const [key, value] of Object.entries(hit.tags_flat ?? {})) {
		const normalized = normalizeTagValue(value);
		if (normalized) {
			tags[key] = normalized;
		}
	}

	if (Object.keys(tags).length > 0) {
		return tags;
	}

	for (const entry of hit.tags ?? []) {
		if (!entry.key) continue;
		const normalized = normalizeTagValue(entry.value);
		if (normalized) {
			tags[entry.key] = normalized;
		}
	}

	return tags;
}

function buildAlgoResults(hit: SearchAssetHit): Record<string, string> {
	const algoResults: Record<string, string> = {};

	for (const algo of hit.algos ?? []) {
		if (!algo.name) continue;
		const algoKey = algo.version ? `${algo.name}@${algo.version}` : algo.name;

		if (algo.status) algoResults[`${algoKey}:status`] = algo.status;
		if (algo.started_at) algoResults[`${algoKey}:started_at`] = algo.started_at;
		if (algo.finished_at)
			algoResults[`${algoKey}:finished_at`] = algo.finished_at;
		if (algo.method) algoResults[`${algoKey}:method`] = algo.method;
		if (algo.run_id) algoResults[`${algoKey}:run_id`] = algo.run_id;
		if (algo.output_uri) algoResults[`${algoKey}:output_uri`] = algo.output_uri;
		if (algo.reason) algoResults[`${algoKey}:reason`] = algo.reason;
		if (algo.result_tag) algoResults[`${algoKey}:result_tag`] = algo.result_tag;
		if (algo.result_score !== undefined) {
			algoResults[`${algoKey}:result_score`] = String(algo.result_score);
		}
	}

	return algoResults;
}

export function normalizeSearchHitToAsset(
	hit: SearchAssetHit,
): SearchAssetResult {
	const lifecycleMeta = hit.metadata ?? {};
	const tags = buildTagsMap(hit);
	const durationMs =
		typeof hit.duration_ms === "number" ? hit.duration_ms : undefined;
	const derivedEnv =
		typeof lifecycleMeta.scene === "string"
			? lifecycleMeta.scene
			: typeof tags.scene === "string"
				? tags.scene
				: undefined;

	return {
		asset_id: hit.asset_id ?? "",
		mcap_file_id: hit.mcap_file_id ?? "",
		segment_locator: hit.segment_locator,
		start_timestamp_ns: hit.start_timestamp_ns ?? 0,
		end_timestamp_ns: hit.end_timestamp_ns ?? 0,
		duration_sec: durationMs !== undefined ? durationMs / 1000 : undefined,
		duration_ms: durationMs,
		reviewer: hit.reviewer ?? "",
		status: hit.status,
		lifecycle_state: hit.lifecycle_state ?? hit.status,
		owner: hit.owner ?? "",
		type: hit.asset_type,
		asset_type: hit.asset_type,
		env: derivedEnv,
		delivery_count: hit.delivery_count ?? 0,
		algo_results: buildAlgoResults(hit),
		tags,
		files: {},
		lifecycle_meta: lifecycleMeta,
		retention_tier:
			hit.retention_tier ??
			(typeof lifecycleMeta.retention_tier === "string"
				? lifecycleMeta.retention_tier
				: undefined),
		expire_at: hit.expire_at ?? null,
		created_at: hit.created_at ?? "",
		updated_at: hit.updated_at ?? "",
		version: hit.version ?? 0,
		_highlight: hit._highlight,
	};
}

export const searchApi = {
	fetchSyncStatus: () =>
		apiClient
			.get<SearchSyncStatusResponse>("/search/sync-status")
			.then((r) => r.data),
	fetchSyncProgress: () =>
		apiClient
			.get<SearchSyncProgressResponse>("/search/sync-progress")
			.then((r) => r.data),

	searchAssets: (params?: SearchAssetsParams) => {
		const sp = new URLSearchParams();
		if (params?.q) sp.set("q", params.q);
		if (params?.mode) sp.set("mode", params.mode);
		if (params?.page) sp.set("page", String(params.page));
		if (params?.page_size) sp.set("page_size", String(params.page_size));
		if (params?.filter) {
			for (const f of params.filter) {
				sp.append("filter", f);
			}
		}
		return apiClient
			.get<Omit<SearchAssetsResponse, "items"> & { items: SearchAssetHit[] }>(
				`/search/assets?${sp.toString()}`,
			)
			.then((r) => ({
				...r.data,
				items: (r.data.items ?? []).map(normalizeSearchHitToAsset),
			}));
	},
};
