import { request } from "./pipelineClient";

export type SubscriptionTaskRunStatus =
  | "succeeded"
  | "failed"
  | "empty";

export interface PipelineBinding {
  templateId: string;
  templateVersion?: number | null;
  targetId: string;
}

export interface SubscriptionTask {
  id: string;
  name: string;
  enabled: boolean;
  projectId: string;
  subscriptionId: string;
  pullIntervalSeconds: number;
  maxMessagesPerPull: number;
  pipelineBindings: PipelineBinding[];
  lastRunAt?: string | null;
  lastRunStatus?: SubscriptionTaskRunStatus;
  lastBatchIds?: string[];
  lastError?: string;
  lastSuccessAt?: string | null;
  createdBy?: string;
  createdAt: string;
  updatedAt: string;
}

export interface SubscriptionTaskCreateRequest {
  name: string;
  enabled?: boolean;
  projectId: string;
  subscriptionId: string;
  pullIntervalSeconds?: number;
  maxMessagesPerPull?: number;
  pipelineBindings: PipelineBinding[];
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

// ── Dispatch history (CYB-3798) ──────────────────────────────────────────────
// A subscription task fans each message out to one backfill batch per binding.
// Batches carry `created_by = "subscription-task:<taskId>"`, so we reverse-look
// them up on the existing Backfill surface (no dedicated endpoint). The
// convention string is kept here, in one place.

export interface DispatchBatch {
  id: string;
  name: string;
  templateId: string;
  templateVersion?: number;
  status: string;
  totalCount: number;
  completedCount: number;
  failedCount: number;
  createdAt: string;
}

export interface DispatchBatchItem {
  id: string;
  assetId: string;
  status: string;
  pipelineRunId?: string | null;
  workflowName?: string | null;
  errorMessage?: string | null;
  createdAt: string;
}

/** Batches this subscription task has dispatched, newest-first. */
export function listSubscriptionTaskBatches(
  taskId: string,
): Promise<DispatchBatch[]> {
  const createdBy = encodeURIComponent(`subscription-task:${taskId}`);
  return request<{ items: DispatchBatch[] }>(
    "GET",
    `/backfill?createdBy=${createdBy}`,
  ).then((r) => r.items ?? []);
}

/** Per-asset items of one dispatched batch (asset id + status + run link). */
export function listDispatchBatchItems(
  batchId: string,
): Promise<DispatchBatchItem[]> {
  return request<{ items: DispatchBatchItem[] }>(
    "GET",
    `/backfill/${encodeURIComponent(batchId)}/items`,
  ).then((r) => r.items ?? []);
}
