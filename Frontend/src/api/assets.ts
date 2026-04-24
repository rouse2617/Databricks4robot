import { apiClient } from "./client";

export interface Asset {
  asset_id: string;
  mcap_file_id: string;
  t_start: number;
  t_end: number;
  reviewer: string;
  status: "pending" | "active" | "archived" | "deleted";
  duration_sec: number;
  owner: string;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface AssetList {
  items: Asset[];
  total: number;
  page: number;
  page_size: number;
}

export interface ListAssetsParams {
  mcap_file_id?: string;
  status?: string;
  tag?: string;
  page?: number;
  page_size?: number;
}

export const assetsApi = {
  list: (params?: ListAssetsParams) =>
    apiClient.get<AssetList>("/assets", { params }).then((r) => r.data),

  get: (id: string) =>
    apiClient.get<Asset>(`/assets/${id}`).then((r) => r.data),

  create: (payload: Partial<Asset>) =>
    apiClient.post<Asset>("/assets", payload).then((r) => r.data),

  update: (id: string, payload: Partial<Asset>) =>
    apiClient.patch<Asset>(`/assets/${id}`, payload).then((r) => r.data),

  delete: (id: string) =>
    apiClient.delete(`/assets/${id}`).then((r) => r.data),
};
