import { apiClient } from "./client";
import {
	type QueryExpr,
	type QueryRequest,
	type QuerySort,
	queryApi,
} from "./query";
import type {
	AlgoEvent,
	Asset,
	AssetEvent,
	AssetProvenance,
	PaginatedResponse,
	PipelineLineage,
} from "./types";

export type { Asset };

// ─── Stats response type ───
export interface AssetStats {
	total_assets: number;
	assets_by_status: Record<string, number>;
	algo_status_summary: Record<string, Record<string, number>>;
	total_deliveries: number;
	recent_assets: Asset[];
}

export interface ListAssetsParams {
	mcap_file_id?: string;
	filter?: string[];
	sort_by?: string;
	page?: number;
	page_size?: number;
}

export interface PreviewSource {
	id: string;
	kind: "live" | "file" | string;
	codec?: string;
	topic?: string;
	url: string;
}

export interface PreviewManifestResponse {
	asset_id: string;
	sources?: PreviewSource[];
	recommended_source_id?: string;
}

/** Direct-MCAP source; `hints.window.*_timestamp_ns` use the MCAP record time base (POSIX ns with `log_time`). */
export interface FoxgloveSourceResponse {
	asset_id: string;
	source_id: string;
	ds: string;
	ds_params: Record<string, string>;
	hints?: Record<string, unknown>;
	expires_at?: string;
}

// ─── CYB-4294: batch duration lookup ────────────────────────────────────────

export type AssetDurationIdType = "auto" | "asset_id" | "grace_video_id";

export interface AssetDurationsRequest {
	ids: string[];
	id_type?: AssetDurationIdType;
	min_duration_ms?: number;
	max_duration_ms?: number;
}

export interface DurationLookupItem {
	input_id: string;
	asset_id: string;
	grace_video_id?: string;
	duration_ms: number;
	duration_sec: number;
	formatted: string;
}

export interface DurationStats {
	matched_count: number;
	missing_count: number;
	filtered_out_count: number;
	total_ms: number;
	mean_ms: number;
	min_ms: number;
	max_ms: number;
	p50_ms: number;
	p90_ms: number;
}

export interface AssetDurationsResponse {
	items: DurationLookupItem[];
	missing_ids: string[];
	filtered_out_ids: string[];
	stats: DurationStats;
}

// ─── CYB-4306: batch cost lookup ────────────────────────────────────────────

export type AssetCostGroupBy = "asset" | "asset_algo";

export interface AssetCostsRequest {
	ids: string[];
	id_type?: AssetDurationIdType;
	start_at: string; // RFC3339
	end_at: string; // RFC3339
	group_by?: AssetCostGroupBy;
}

export interface AssetCostByAlgo {
	algo_key: string;
	cost_usd: number;
	gpu_sec: number;
	cpu_sec: number;
	run_count: number;
}

export interface AssetCostItem {
	input_id: string;
	asset_id: string;
	grace_video_id?: string;
	total_cost_usd: number;
	gpu_sec: number;
	cpu_sec: number;
	gpu_min: number;
	cpu_min: number;
	run_count: number;
	by_algo: AssetCostByAlgo[] | null;
}

export interface AssetCostStats {
	matched_count: number;
	missing_count: number;
	filtered_out_count: number;
	total_cost_usd: number;
	mean_cost_usd: number;
	p50_cost_usd: number;
	p90_cost_usd: number;
	total_gpu_sec: number;
	total_cpu_sec: number;
	total_run_count: number;
}

export interface AssetCostsResponse {
	items: AssetCostItem[];
	missing_ids: string[];
	filtered_out_ids: string[];
	stats: AssetCostStats;
}

// ─── CYB-4305: batch lineage lookup ─────────────────────────────────────────

export type AssetLineageDepth = 1 | "1" | "all";

export interface AssetLineageBatchRequest {
	ids: string[];
	id_type?: AssetDurationIdType;
	// Server accepts `1`, `"1"`, and `"all"`. Default 1.
	depth?: AssetLineageDepth;
}

// LineageBatchItem carries the direct-hop fields on every response.
// upstream_ids / downstream_ids / relation_types are absent when depth=1
// (server does not emit them) and present as arrays when depth=all.
export interface LineageBatchItem {
	input_id: string;
	asset_id: string;
	grace_video_id?: string;
	parent_asset_id: string | null;
	root_asset_id: string | null;
	logical_asset_id: string | null;
	is_current: boolean;
	revision: number;
	upstream_ids?: string[];
	downstream_ids?: string[];
	relation_types?: string[];
}

export interface LineageBatchStats {
	matched_count: number;
	missing_count: number;
	has_parent_count: number;
	is_root_count: number;
	orphan_count: number;
	is_current_count: number;
	// Present when depth=all (server sets an empty map even when no items
	// carry relation types); absent when depth=1.
	relation_type_counts?: Record<string, number>;
}

export interface AssetLineageBatchResponse {
	items: LineageBatchItem[];
	missing_ids: string[];
	// Kept in the envelope for symmetry with the durations / costs shells;
	// lineage has no server-side range filter so this is always [].
	filtered_out_ids: string[];
	stats: LineageBatchStats;
}

export interface AssetMetadataResponse {
	asset_id: string;
	segment_locator: string;
	lifecycle_state: string;
	asset_metadata: Record<string, unknown> | null;
	mcap_metadata: Record<string, unknown> | null;
	grace_video_snapshot: Record<string, unknown> | null;
	storage: {
		gcs: Record<string, unknown> | null;
		aliyun: Record<string, unknown> | null;
	};
	mcap_process_state: Record<string, unknown> | null;
	collection: Record<string, unknown> | null;
	process_info: Record<string, unknown> | null;
	video_info: Record<string, unknown> | null;
}

function parseFilterParam(raw: string): QueryExpr | null {
	const firstColon = raw.indexOf(":");
	if (firstColon === -1) return null;
	const secondColon = raw.indexOf(":", firstColon + 1);
	if (secondColon === -1) return null;
	const field = raw.slice(0, firstColon);
	const op = raw.slice(firstColon + 1, secondColon);
	const rawValue = raw.slice(secondColon + 1);
	if (!field || !op) return null;
	const value = rawValue.includes(",") ? rawValue.split(",") : rawValue;
	return {
		pred: {
			field,
			op,
			value,
		},
	};
}

function buildQuerySort(sortBy?: string): QuerySort[] {
	if (!sortBy) return [{ field: "updated_at", direction: "desc" }];
	if (sortBy.startsWith("-")) {
		return [{ field: sortBy.slice(1), direction: "desc" }];
	}
	return [{ field: sortBy, direction: "asc" }];
}

type AssetEventEnvelope = {
	algo_key?: string;
	prev_status?: string;
	new_status?: string;
	run_id?: string;
	reason?: string;
};

type AssetEventRow = {
	event_id: string;
	event_seq?: number;
	event_type?: string;
	payload_schema_version?: string;
	asset_id: string;
	mcap_file_id?: string;
	event_source?: string;
	request_id?: string;
	publish_state?: string;
	occurred_at?: string;
	created_at: string;
	event_payload?: AssetEventEnvelope & Record<string, unknown>;
};

function toAlgoEvent(row: AssetEventRow): AlgoEvent {
	const payload = row.event_payload ?? {};
	return {
		event_id: row.event_id,
		asset_id: row.asset_id,
		algo_key: payload.algo_key ?? "",
		prev_status: payload.prev_status,
		new_status: payload.new_status ?? "",
		run_id: payload.run_id,
		reason: payload.reason,
		created_at: row.created_at,
	};
}

function toAssetEvent(row: AssetEventRow): AssetEvent {
	return {
		event_id: row.event_id,
		event_seq: row.event_seq ?? 0,
		event_type: row.event_type ?? "",
		payload_schema_version: row.payload_schema_version,
		asset_id: row.asset_id,
		mcap_file_id: row.mcap_file_id,
		event_source: row.event_source,
		request_id: row.request_id,
		publish_state: row.publish_state,
		event_payload: row.event_payload,
		created_at: row.created_at,
		occurred_at: row.occurred_at,
	};
}

type AssetEventStreamCallback = {
	onEvent?: (event: AssetEvent) => void;
	onError?: (error: Event) => void;
	onKeepAlive?: () => void;
	onOpen?: () => void;
};

type StreamableEventSource = {
	event_id?: string;
	event_seq?: number | string;
	event_type?: string;
	asset_id?: string;
	event_payload?: Record<string, unknown>;
	occurred_at?: string;
};

function parseStreamEvent(payload: unknown): AssetEvent | null {
	if (!payload || typeof payload !== "object") return null;
	const obj = payload as StreamableEventSource;
	const eventSeq =
		typeof obj.event_seq === "number"
			? obj.event_seq
			: Number(obj.event_seq ?? 0) || 0;
	const occurredAt =
		typeof obj.occurred_at === "string"
			? obj.occurred_at
			: new Date().toISOString();
	return {
		event_id: obj.event_id ?? `stream-${eventSeq}`,
		event_seq: eventSeq,
		event_type: obj.event_type ?? "unknown",
		payload_schema_version: undefined,
		asset_id: obj.asset_id ?? "",
		mcap_file_id: undefined,
		event_source: undefined,
		request_id: undefined,
		publish_state: undefined,
		event_payload: obj.event_payload,
		created_at: occurredAt,
		occurred_at: occurredAt,
	};
}

function makeEventStreamUrl(path: string): string {
	const base = apiClient.defaults.baseURL ?? "/api/v1";
	return new URL(path, new URL(base, window.location.origin)).toString();
}

function streamAssetEvents(
	url: string,
	callbacks: AssetEventStreamCallback = {},
): () => void {
	const es = new EventSource(url, { withCredentials: true });
	es.addEventListener("open", () => {
		callbacks.onOpen?.();
	});
	es.addEventListener("message", (evt) => {
		if (evt.type === "keepalive" || evt.data?.trim() === "") {
			callbacks.onKeepAlive?.();
			return;
		}
		try {
			const parsed = parseStreamEvent(JSON.parse(evt.data));
			if (parsed) {
				callbacks.onEvent?.(parsed);
			}
		} catch {
			// Ignore malformed SSE payloads.
		}
	});
	es.addEventListener("error", (evt) => {
		callbacks.onError?.(evt);
	});
	return () => es.close();
}

type EventListResponse = {
	items: AssetEvent[];
	next_cursor?: number;
	limit: number;
};

type AssetDeliveryListResponse = {
	items: string[];
	asset_id: string;
	page: number;
	page_size: number;
	next_token?: string;
};

export const assetsApi = {
	list: (params?: ListAssetsParams) => {
		const predicates: QueryExpr[] = [];
		if (params?.mcap_file_id) {
			predicates.push({
				pred: {
					field: "mcap_file_id",
					op: "eq",
					value: params.mcap_file_id,
				},
			});
		}
		for (const filter of params?.filter ?? []) {
			const expr = parseFilterParam(filter);
			if (expr) predicates.push(expr);
		}
		const query: QueryRequest = {
			schema_version: "v1",
			scope: { resource: "assets" },
			where:
				predicates.length === 0
					? undefined
					: predicates.length === 1
						? predicates[0]
						: { and: predicates },
			sort: buildQuerySort(params?.sort_by),
			page: {
				page: params?.page ?? 1,
				page_size: params?.page_size ?? 20,
			},
		};
		return queryApi.run(query).then((data) => ({
			items: data.items,
			total: data.total,
			page: data.page,
			page_size: data.page_size,
		})) as Promise<PaginatedResponse<Asset>>;
	},

	get: (id: string) =>
		apiClient.get<Asset>(`/assets/${id}`).then((r) => r.data),

	// CYB-4011: resolve a Grace video UUID to its DataBrew asset via the
	// backfilled grace_video_id column. Returns the first matching asset, or
	// null if none. Used as a frontend fallback when a Grace UUID is used
	// where a DataBrew asset_id is expected (asset detail redirect; subtask
	// video-duration column).
	resolveByGraceVideoID: (graceVideoID: string) =>
		assetsApi
			.list({ filter: [`grace_video_id:eq:${graceVideoID}`], page_size: 1 })
			.then((r) => r.items[0] ?? null)
			.catch(() => null),

	batchGet: (assetIds: string[]) =>
		apiClient
			.post<{ items: Asset[] }>("/assets:batch_get", { asset_ids: assetIds })
			.then((r) => r.data.items ?? []),

	// CYB-4294: batch duration lookup. Server matches both `asset_id` and
	// `grace_video_id` columns via OR, so ids may be mixed shapes.
	lookupDurations: (req: AssetDurationsRequest) =>
		apiClient
			.post<AssetDurationsResponse>("/assets/durations", req)
			.then((r) => r.data),

	// CYB-4306: batch cost lookup. Server resolves ids via the same OR-on-
	// both-columns index scan, then aggregates leaf-pod costs over the
	// [start_at, end_at] window (required, ≤90 days). Set group_by to
	// "asset_algo" for a per-algo breakdown.
	lookupCosts: (req: AssetCostsRequest) =>
		apiClient
			.post<AssetCostsResponse>("/assets/costs", req)
			.then((r) => r.data),

	// CYB-4305: batch lineage lookup. depth=1 (default) returns direct-hop
	// parent/root/logical/is_current/revision in one SQL scan; depth="all"
	// overlays the ES lineage projection (upstream/downstream/relation).
	lookupLineage: (req: AssetLineageBatchRequest) =>
		apiClient
			.post<AssetLineageBatchResponse>("/assets/lineage-batch", req)
			.then((r) => r.data),

	getPreviewManifest: (id: string) =>
		apiClient
			.get<PreviewManifestResponse>(`/preview/assets/${id}/manifest`)
			.then((r) => r.data),

	getFoxgloveSource: (id: string) =>
		apiClient
			.get<FoxgloveSourceResponse>(`/assets/${id}/foxglove-source`)
			.then((r) => r.data),

	create: (payload: Partial<Asset>) =>
		apiClient.post<Asset>("/assets", payload).then((r) => r.data),

	update: (id: string, payload: Partial<Asset>) =>
		apiClient.patch<Asset>(`/assets/${id}`, payload).then((r) => r.data),

	delete: (id: string) => apiClient.delete(`/assets/${id}`).then((r) => r.data),

	upsertTag: (
		assetId: string,
		body: {
			key: string;
			value: string;
			source_type?: string;
			source_name?: string;
			source_version?: string;
			run_id?: string;
		},
	) =>
		apiClient.post<Asset>(`/assets/${assetId}/tags`, body).then((r) => r.data),

	deleteTag: (assetId: string, key: string, sourceType?: string) =>
		apiClient
			.delete<Asset>(`/assets/${assetId}/tags/${encodeURIComponent(key)}`, {
				params: sourceType ? { source_type: sourceType } : undefined,
			})
			.then((r) => r.data),

	// ─── Delivery association ───
	listDeliveries: (assetId: string, page = 1, pageSize = 20) =>
		apiClient
			.get<AssetDeliveryListResponse>(`/assets/${assetId}/deliveries`, {
				params: { page, page_size: pageSize },
			})
			.then((r) => r.data),

	// ─── Algo lifecycle ───
	startAlgo: (
		assetId: string,
		algoKey: string,
		body: { method: string; run_id?: string },
	) =>
		apiClient
			.post(`/assets/${assetId}/algo/${algoKey}/start`, body)
			.then((r) => r.data),

	finishAlgo: (
		assetId: string,
		algoKey: string,
		body: Record<string, unknown>,
	) =>
		apiClient
			.post(`/assets/${assetId}/algo/${algoKey}/finish`, body)
			.then((r) => r.data),

	resetAlgo: (assetId: string, algoKey: string) =>
		apiClient
			.post(`/assets/${assetId}/algo/${algoKey}/reset`)
			.then((r) => r.data),

	listEvents: (
		assetId: string,
		params?: {
			event_type?: string | string[];
			algo_key?: string;
			cursor?: number;
			start_time?: string;
			end_time?: string;
			limit?: number;
		},
	) =>
		apiClient
			.get<{ items: AssetEventRow[] }>(`/assets/${assetId}/events`, {
				params,
			})
			.then(
				(r) =>
					({
						items: (r.data.items ?? []).map(toAssetEvent),
						next_cursor: (r.data as { next_cursor?: number }).next_cursor,
						limit: (r.data as { limit?: number }).limit ?? params?.limit ?? 50,
					}) as EventListResponse,
			),

	listAlgoEvents: (
		assetId: string,
		algoKey?: string,
		cursor?: number,
		limit = 20,
	) =>
		assetsApi
			.listEvents(assetId, {
				event_type: "algo_*",
				...(algoKey ? { algo_key: algoKey } : {}),
				...(cursor ? { cursor } : {}),
				limit,
			})
			.then((r) => ({
				items: r.items.map((row) => toAlgoEvent(row as AssetEventRow)),
				next_cursor: r.next_cursor,
				limit: r.limit,
			})),

	streamForAsset: (assetId: string, callbacks?: AssetEventStreamCallback) =>
		streamAssetEvents(
			makeEventStreamUrl(
				`/assets/${encodeURIComponent(assetId)}/events/stream`,
			),
			callbacks,
		),

	streamGlobalEvents: (
		callbacks?: AssetEventStreamCallback,
		fallbackAssetId?: string,
	) =>
		fallbackAssetId
			? streamAssetEvents(
					makeEventStreamUrl(
						`/assets/${encodeURIComponent(fallbackAssetId)}/events/stream`,
					),
					callbacks,
				)
			: streamAssetEvents(makeEventStreamUrl("/events/stream"), callbacks),

	listGlobalEvents: (params?: {
		event_type?: string;
		cursor?: number;
		start_time?: string;
		end_time?: string;
		limit?: number;
	}) =>
		apiClient.get<{ items: AssetEventRow[] }>("/events", { params }).then(
			(r) =>
				({
					items: (r.data.items ?? []).map(toAssetEvent),
					next_cursor: (r.data as { next_cursor?: number }).next_cursor,
					limit: (r.data as { limit?: number }).limit ?? params?.limit ?? 100,
				}) as EventListResponse,
		),
	getLineage: (assetId: string) =>
		apiClient
			.get<{
				asset_id: string;
				upstream: Record<string, unknown>;
				downstream: {
					algo_results?: Array<Record<string, unknown>>;
					deliveries?: Array<Record<string, unknown>>;
					eval_results?: Array<Record<string, unknown>>;
				};
			}>(`/assets/${assetId}/lineage`)
			.then((r) => r.data),
	getPipelineLineage: (assetId: string) =>
		apiClient
			.get<PipelineLineage>(`/assets/${assetId}/pipeline-lineage`)
			.then((r) => r.data),
	getProvenance: (assetId: string) =>
		apiClient
			.get<AssetProvenance>(`/assets/${assetId}/provenance`)
			.then((r) => r.data),

	getMetadata: (assetId: string) =>
		apiClient
			.get<AssetMetadataResponse>(`/assets/${assetId}/metadata`)
			.then((r) => r.data),
};
