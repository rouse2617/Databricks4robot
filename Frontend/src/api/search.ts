import { apiClient } from "./client";

export interface SearchAssetsParams {
  q?: string;
  filter?: string[];
  page?: number;
  page_size?: number;
}

export interface SearchAggBucket {
  key: string;
  doc_count: number;
}

export interface SearchAssetsResponse {
  items: (Record<string, unknown> & { _highlight?: Record<string, string[]> })[];
  total: number;
  page: number;
  page_size: number;
  aggregations?: Record<string, SearchAggBucket[]>;
}

export const searchApi = {
  searchAssets: (params?: SearchAssetsParams) => {
    const sp = new URLSearchParams();
    if (params?.q) sp.set("q", params.q);
    if (params?.page) sp.set("page", String(params.page));
    if (params?.page_size) sp.set("page_size", String(params.page_size));
    if (params?.filter) {
      for (const f of params.filter) {
        sp.append("filter", f);
      }
    }
    return apiClient
      .get<SearchAssetsResponse>(`/search/assets?${sp.toString()}`)
      .then((r) => r.data);
  },
};
