import { apiClient } from "./client";

export interface AlgoRegistryItem {
  key: string;
  name: string;
  version: string;
  depends_on: string[];
}

export interface AlgoRegistryResponse {
  items: AlgoRegistryItem[];
}

export const algoRegistryApi = {
  list: () =>
    apiClient
      .get<AlgoRegistryResponse>("/algo-registry")
      .then((r) => r.data.items),
};
