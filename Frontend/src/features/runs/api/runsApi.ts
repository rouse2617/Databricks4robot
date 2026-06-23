import { request } from "../../../api/pipelineClient";
import type { PipelineRun } from "../../../api/pipelineApi";

export type RunType = "pipeline" | "component_build" | "rag_build";

export interface RunRuntimeRef {
	type: string;
	namespace: string;
	resourceName: string;
	uid?: string;
}

export interface RunActions {
	canRetry?: boolean;
	canTerminate?: boolean;
	canSuspend?: boolean;
	canResume?: boolean;
	canResubmit?: boolean;
	canStop?: boolean;
}

export interface DatabrewRun {
	id: string;
	type: RunType;
	name: string;
	status: string;
	statusLabel?: string;
	runtime: RunRuntimeRef;
	owner?: string;
	createdBy?: string;
	message?: string;
	summary?: Record<string, unknown>;
	actions?: RunActions;
	createdAt: string;
	startedAt?: string;
	finishedAt?: string;
	updatedAt?: string;
}

export interface RunListResponse {
	items: DatabrewRun[];
	total: number;
}

export function listRuns(params?: {
	type?: RunType;
	status?: string;
	page?: number;
	pageSize?: number;
}): Promise<RunListResponse> {
	const search = new URLSearchParams();
	if (params?.type) search.set("type", params.type);
	if (params?.status) search.set("status", params.status);
	if (params?.page) search.set("page", String(params.page));
	if (params?.pageSize) search.set("pageSize", String(params.pageSize));
	const suffix = search.toString() ? `?${search.toString()}` : "";
	return request<RunListResponse>("GET", `/runs${suffix}`);
}

export function getRun(runId: string): Promise<DatabrewRun> {
	return request<DatabrewRun>("GET", `/runs/${encodeURIComponent(runId)}`);
}

export function getRunByWorkflowName(workflowName: string): Promise<DatabrewRun> {
	return request<DatabrewRun>(
		"GET",
		`/runs/by-workflow/${encodeURIComponent(workflowName)}`,
	);
}

export function getRunRuntime(runId: string): Promise<Record<string, unknown>> {
	return request<Record<string, unknown>>(
		"GET",
		`/runs/${encodeURIComponent(runId)}/runtime`,
	);
}

export function getRunArtifacts(runId: string): Promise<{ items: Record<string, unknown>[] }> {
	return request<{ items: Record<string, unknown>[] }>(
		"GET",
		`/runs/${encodeURIComponent(runId)}/artifacts`,
	);
}

export function retryRun(runId: string): Promise<PipelineRun> {
	return request<PipelineRun>("POST", `/runs/${encodeURIComponent(runId)}/retry`, {});
}

export function stopRun(runId: string): Promise<{ message: string }> {
	return request<{ message: string }>("POST", `/runs/${encodeURIComponent(runId)}/stop`, {});
}

export function suspendRun(runId: string): Promise<{ message: string }> {
	return request<{ message: string }>(
		"POST",
		`/runs/${encodeURIComponent(runId)}/suspend`,
		{},
	);
}

export function resumeRun(runId: string): Promise<{ message: string }> {
	return request<{ message: string }>(
		"POST",
		`/runs/${encodeURIComponent(runId)}/resume`,
		{},
	);
}

export function resubmitRun(runId: string): Promise<{ message: string }> {
	return request<{ message: string }>(
		"POST",
		`/runs/${encodeURIComponent(runId)}/resubmit`,
		{},
	);
}

export function terminateRun(runId: string): Promise<{ message: string }> {
	return request<{ message: string }>(
		"POST",
		`/runs/${encodeURIComponent(runId)}/terminate`,
		{},
	);
}
