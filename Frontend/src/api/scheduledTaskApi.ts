// Scheduled Tasks API client (CYB-3744).
//
// Shape mirrors `api/openapi.yaml` (paths + schemas under the ScheduledTasks
// tag). Keep the naming symmetric with the backend DTOs so the shared UI form
// components can bind directly. Credential fields are ONLY `secretRef` (env
// var name); the value itself never travels through the frontend.
import { request } from "./pipelineClient";

// ─── Types (mirror openapi.yaml ScheduledTask*) ─────────────────

export type ScheduledTaskTriggerMode =
	| "incremental"
	| "rolling"
	| "range"
	| "ids";
export type ScheduledTaskSourceType = "rest";
export type ScheduledTaskRunStatus =
	| "succeeded"
	| "failed"
	| "empty"
	| "skipped";
export type ScheduledTaskAuthType = "basic" | "bearer" | "header_key" | "none";

export interface ScheduledTaskAuthConfig {
	type: ScheduledTaskAuthType;
	/** basic-only; NOT a secret. */
	username?: string;
	/** Env-var name whose value holds the credential. Never send raw creds. */
	secret_ref?: string;
	/** header_key auth only. */
	header_name?: string;
}

export interface ScheduledTaskQueryConfig {
	static?: Record<string, string[]>;
	filter_param?: string;
	time_field?: string;
	time_format?: "rfc3339" | "unix";
	time_query_start?: string;
	time_query_end?: string;
}

export interface ScheduledTaskPagingConfig {
	mode?: "page_size" | "none";
	page_param?: string;
	size_param?: string;
	page_size?: number;
	total_path?: string;
	data_path?: string;
}

export interface ScheduledTaskRestSource {
	base_url: string;
	path: string;
	method?: "GET";
	auth?: ScheduledTaskAuthConfig;
	query?: ScheduledTaskQueryConfig;
	paging?: ScheduledTaskPagingConfig;
	id_path: string;
	headers?: Record<string, string>;
	user_agent?: string;
	timeout_seconds?: number;
}

export interface ScheduledTaskTriggerConfig {
	intervalSeconds?: number;
	lookbackSeconds?: number;
	initialLookbackSeconds?: number;
	from?: string;
	to?: string;
	ids?: string[];
	notifyOnEmpty?: boolean;
	stuckThresholdSeconds?: number;
}

export interface ScheduledTask {
	id: string;
	name: string;
	enabled: boolean;
	templateId: string;
	templateVersion?: number | null;
	targetId: string;
	scheduling?: Record<string, unknown>;
	sourceType: ScheduledTaskSourceType;
	sourceConfig?: ScheduledTaskRestSource;
	triggerMode: ScheduledTaskTriggerMode;
	triggerConfig?: ScheduledTaskTriggerConfig;
	cursor?: string;
	lastRunAt?: string | null;
	lastRunStatus?: ScheduledTaskRunStatus;
	lastBatchId?: string;
	lastError?: string;
	lastSuccessAt?: string | null;
	runNowRequestedAt?: string | null;
	createdBy?: string;
	createdAt: string;
	updatedAt: string;
}

export interface ScheduledTaskCreateRequest {
	name: string;
	enabled?: boolean;
	templateId: string;
	templateVersion?: number;
	targetId: string;
	scheduling?: Record<string, unknown>;
	sourceType: ScheduledTaskSourceType;
	sourceConfig?: ScheduledTaskRestSource;
	triggerMode: ScheduledTaskTriggerMode;
	triggerConfig?: ScheduledTaskTriggerConfig;
}

export interface ScheduledTaskListResponse {
	items: ScheduledTask[];
	total: number;
	page: number;
	pageSize: number;
}

export interface ListScheduledTasksOptions {
	q?: string;
	enabled?: boolean;
	sourceType?: ScheduledTaskSourceType;
	page?: number;
	pageSize?: number;
}

// ─── Endpoints ─────────────────────────────────────────────────

const BASE = "/scheduled-tasks";

export function listScheduledTasks(
	options: ListScheduledTasksOptions = {},
): Promise<ScheduledTaskListResponse> {
	const sp = new URLSearchParams();
	if (options.q) sp.set("q", options.q);
	if (options.enabled !== undefined)
		sp.set("enabled", options.enabled ? "true" : "false");
	if (options.sourceType) sp.set("sourceType", options.sourceType);
	if (options.page) sp.set("page", String(options.page));
	if (options.pageSize) sp.set("pageSize", String(options.pageSize));
	const qs = sp.toString();
	return request<ScheduledTaskListResponse>("GET", qs ? `${BASE}?${qs}` : BASE);
}

export function getScheduledTask(id: string): Promise<ScheduledTask> {
	return request<ScheduledTask>("GET", `${BASE}/${encodeURIComponent(id)}`);
}

export function createScheduledTask(
	body: ScheduledTaskCreateRequest,
): Promise<ScheduledTask> {
	return request<ScheduledTask>("POST", BASE, body);
}

export function updateScheduledTask(
	id: string,
	body: ScheduledTaskCreateRequest,
): Promise<ScheduledTask> {
	return request<ScheduledTask>(
		"PUT",
		`${BASE}/${encodeURIComponent(id)}`,
		body,
	);
}

export function deleteScheduledTask(id: string): Promise<void> {
	return request<void>("DELETE", `${BASE}/${encodeURIComponent(id)}`);
}

export function pauseScheduledTask(id: string): Promise<ScheduledTask> {
	return request<ScheduledTask>(
		"POST",
		`${BASE}/${encodeURIComponent(id)}/pause`,
	);
}

export function resumeScheduledTask(id: string): Promise<ScheduledTask> {
	return request<ScheduledTask>(
		"POST",
		`${BASE}/${encodeURIComponent(id)}/resume`,
	);
}

export function runScheduledTaskNow(id: string): Promise<ScheduledTask> {
	return request<ScheduledTask>(
		"POST",
		`${BASE}/${encodeURIComponent(id)}/run-now`,
	);
}
