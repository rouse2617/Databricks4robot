import { request } from "./pipelineClient";

export type SubscriptionTaskRunStatus =
  | "succeeded"
  | "failed"
  | "empty";

export interface SubscriptionTask {
  id: string;
  name: string;
  enabled: boolean;
  templateId: string;
  templateVersion?: number | null;
  targetId: string;
  projectId: string;
  subscriptionId: string;
  pullIntervalSeconds: number;
  maxMessagesPerPull: number;
  lastRunAt?: string | null;
  lastRunStatus?: SubscriptionTaskRunStatus;
  lastBatchId?: string;
  lastError?: string;
  lastSuccessAt?: string | null;
  createdBy?: string;
  createdAt: string;
  updatedAt: string;
}

export interface SubscriptionTaskCreateRequest {
  name: string;
  enabled?: boolean;
  templateId: string;
  templateVersion?: number;
  targetId: string;
  projectId: string;
  subscriptionId: string;
  pullIntervalSeconds?: number;
  maxMessagesPerPull?: number;
}

export interface SubscriptionTaskListResponse {
  items: SubscriptionTask[];
  total: number;
  page: number;
  pageSize: number;
}

export interface ListSubscriptionTasksOptions {
  q?: string;
  enabled?: boolean;
  page?: number;
  pageSize?: number;
}

const BASE = "/subscription-tasks";

export function listSubscriptionTasks(
  options: ListSubscriptionTasksOptions = {},
): Promise<SubscriptionTaskListResponse> {
  const sp = new URLSearchParams();
  if (options.q) sp.set("q", options.q);
  if (options.enabled !== undefined)
    sp.set("enabled", options.enabled ? "true" : "false");
  if (options.page) sp.set("page", String(options.page));
  if (options.pageSize) sp.set("pageSize", String(options.pageSize));
  const qs = sp.toString();
  return request<SubscriptionTaskListResponse>("GET", qs ? `${BASE}?${qs}` : BASE);
}

export function getSubscriptionTask(id: string): Promise<SubscriptionTask> {
  return request<SubscriptionTask>("GET", `${BASE}/${encodeURIComponent(id)}`);
}

export function createSubscriptionTask(
  body: SubscriptionTaskCreateRequest,
): Promise<SubscriptionTask> {
  return request<SubscriptionTask>("POST", BASE, body);
}

export function updateSubscriptionTask(
  id: string,
  body: SubscriptionTaskCreateRequest,
): Promise<SubscriptionTask> {
  return request<SubscriptionTask>(
    "PUT",
    `${BASE}/${encodeURIComponent(id)}`,
    body,
  );
}

export function deleteSubscriptionTask(id: string): Promise<void> {
  return request<void>("DELETE", `${BASE}/${encodeURIComponent(id)}`);
}

export function pauseSubscriptionTask(id: string): Promise<SubscriptionTask> {
  return request<SubscriptionTask>(
    "POST",
    `${BASE}/${encodeURIComponent(id)}/pause`,
  );
}

export function resumeSubscriptionTask(id: string): Promise<SubscriptionTask> {
  return request<SubscriptionTask>(
    "POST",
    `${BASE}/${encodeURIComponent(id)}/resume`,
  );
}
