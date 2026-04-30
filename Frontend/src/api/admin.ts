import axios from "axios";

export const ADMIN_TOKEN_KEY = "grace_admin_token";

export function readAdminToken(): string {
  if (typeof window === "undefined") {
    return "";
  }
  return localStorage.getItem(ADMIN_TOKEN_KEY) ?? "";
}

export function writeAdminToken(token: string): void {
  if (typeof window === "undefined") {
    return;
  }
  const trimmed = token.trim();
  if (trimmed === "") {
    localStorage.removeItem(ADMIN_TOKEN_KEY);
    return;
  }
  localStorage.setItem(ADMIN_TOKEN_KEY, trimmed);
}

const adminClient = axios.create({
  baseURL: "/api/v1/admin",
  timeout: 30_000,
});

adminClient.interceptors.request.use((config) => {
  const token = readAdminToken();
  if (token) {
    config.headers["X-Admin-Token"] = token;
  }
  return config;
});

export interface ReindexResult {
  dry_run: boolean;
  total_assets: number;
  indexed: number;
  deleted: number;
  failed: number;
  duration_ms: number;
  errors?: string[];
  elasticsearch_doc_count?: number;
}

export const adminApi = {
  reindex: (dryRun: boolean) =>
    adminClient
      .post<ReindexResult>("/search/reindex", {
        dry_run: dryRun,
      })
      .then((r) => r.data),
};
