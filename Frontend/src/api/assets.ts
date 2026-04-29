import { apiClient } from "./client";
import type { Asset, AlgoEvent, AssetEvent, PaginatedResponse } from "./types";

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
  status?: string;
  filter?: string[];
  sort_by?: string;
  page?: number;
  page_size?: number;
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

type EventListResponse = {
  items: AssetEvent[];
  next_cursor?: number;
  limit: number;
};

type AssetDeliveryListResponse = {
  items: string[];
  total: number;
  page: number;
  page_size: number;
  next_token?: string;
};

export const assetsApi = {
  list: (params?: ListAssetsParams) => {
    // Build URLSearchParams manually to ensure filter[] is sent as repeated params
    const sp = new URLSearchParams();
    if (params?.page) sp.set("page", String(params.page));
    if (params?.page_size) sp.set("page_size", String(params.page_size));
    if (params?.sort_by) sp.set("sort_by", params.sort_by);
    if (params?.mcap_file_id) sp.set("mcap_file_id", params.mcap_file_id);
    if (params?.filter) {
      for (const f of params.filter) {
        sp.append("filter", f);
      }
    }
    return apiClient
      .get<PaginatedResponse<Asset>>(`/assets?${sp.toString()}`)
      .then((r) => r.data);
  },

  get: (id: string) =>
    apiClient.get<Asset>(`/assets/${id}`).then((r) => r.data),

  create: (payload: Partial<Asset>) =>
    apiClient.post<Asset>("/assets", payload).then((r) => r.data),

  update: (id: string, payload: Partial<Asset>) =>
    apiClient.patch<Asset>(`/assets/${id}`, payload).then((r) => r.data),

  delete: (id: string) =>
    apiClient.delete(`/assets/${id}`).then((r) => r.data),

  upsertTag: (assetId: string, body: { key: string; value: string }) =>
    apiClient.post<Asset>(`/assets/${assetId}/tags`, body).then((r) => r.data),

  deleteTag: (assetId: string, key: string) =>
    apiClient.delete<Asset>(`/assets/${assetId}/tags/${encodeURIComponent(key)}`).then((r) => r.data),

  // ─── Delivery association ───
  listDeliveries: (assetId: string, page = 1, pageSize = 20) =>
    apiClient
      .get<AssetDeliveryListResponse>(`/assets/${assetId}/deliveries`, {
        params: { page, page_size: pageSize },
      })
      .then((r) => r.data),

  // ─── Algo lifecycle ───
  startAlgo: (assetId: string, algoKey: string, body: { method: string; run_id?: string }) =>
    apiClient.post(`/assets/${assetId}/algo/${algoKey}/start`, body).then((r) => r.data),

  finishAlgo: (assetId: string, algoKey: string, body: Record<string, unknown>) =>
    apiClient.post(`/assets/${assetId}/algo/${algoKey}/finish`, body).then((r) => r.data),

  resetAlgo: (assetId: string, algoKey: string) =>
    apiClient.post(`/assets/${assetId}/algo/${algoKey}/reset`).then((r) => r.data),

  listEvents: (
    assetId: string,
    params?: {
      event_type?: string | string[];
      algo_key?: string;
      cursor?: number;
      limit?: number;
    },
  ) =>
    apiClient
      .get<{ items: AssetEventRow[] }>(`/assets/${assetId}/events`, {
        params,
      })
      .then((r) => ({
        items: (r.data.items ?? []).map(toAssetEvent),
        next_cursor: (r.data as { next_cursor?: number }).next_cursor,
        limit: (r.data as { limit?: number }).limit ?? params?.limit ?? 50,
      }) as EventListResponse),

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

};
