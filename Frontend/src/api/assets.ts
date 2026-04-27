import { apiClient } from "./client";
import type { Asset, AlgoEvent, PaginatedResponse } from "./types";

export type { Asset };

export interface ListAssetsParams {
  mcap_file_id?: string;
  status?: string;
  filter?: string[];
  sort_by?: string;
  page?: number;
  page_size?: number;
}

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

  // ─── Delivery association ───
  listDeliveries: (assetId: string) =>
    apiClient
      .get<{ delivery_ids: string[] }>(`/assets/${assetId}/deliveries`)
      .then((r) => r.data.delivery_ids ?? []),

  // ─── Algo lifecycle ───
  startAlgo: (assetId: string, algoKey: string, body: { method: string; run_id?: string }) =>
    apiClient.post(`/assets/${assetId}/algo/${algoKey}/start`, body).then((r) => r.data),

  finishAlgo: (assetId: string, algoKey: string, body: Record<string, unknown>) =>
    apiClient.post(`/assets/${assetId}/algo/${algoKey}/finish`, body).then((r) => r.data),

  resetAlgo: (assetId: string, algoKey: string) =>
    apiClient.post(`/assets/${assetId}/algo/${algoKey}/reset`).then((r) => r.data),

  listAlgoEvents: (assetId: string, algoKey?: string) =>
    apiClient
      .get<{ items: AlgoEvent[] }>(`/assets/${assetId}/algo-events`, {
        params: algoKey ? { algo_key: algoKey } : undefined,
      })
      .then((r) => r.data.items),
};
