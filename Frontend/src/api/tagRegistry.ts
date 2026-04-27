import { apiClient } from "./client";

export interface TagRegistryItem {
  key: string;
  type: "enum" | "string";
  values?: string[];
  max_length?: number;
}

export interface TagRegistryResponse {
  items: TagRegistryItem[];
}

export const tagRegistryApi = {
  list: () =>
    apiClient
      .get<TagRegistryResponse>("/tag-registry")
      .then((r) => r.data.items),
};
