import type { Pipeline } from "../components/pipeline/types";
import type {
	DeployConfigSelection,
	DeployTemplateResponse,
	PipelineRun,
	PipelineRunAssetNodeListResponse,
	PipelineRunCostSummary,
	PipelineRunEventsResponse,
	PipelineRunListResponse,
	PipelineRunNode,
	PipelineRunWatcherState,
} from "./pipelineApi";
import { request } from "./pipelineClient";

export type {
	PipelineRun as Run,
	PipelineRunAssetNodeListResponse as RunAssetNodeListResponse,
	PipelineRunCostSummary as RunCostSummary,
	PipelineRunEvent as RunEvent,
	PipelineRunEventsResponse as RunEventsResponse,
	PipelineRunListResponse as RunListResponse,
	PipelineRunNode as RunNode,
	PipelineRunWatcherState as RunWatcherState,
} from "./pipelineApi";

export interface RunInput {
	id: string;
	runId: string;
	nodeId?: string;
	type: "asset" | "runtime_target" | "config" | "parameter" | string;
	refId?: string;
	refVersion?: string;
	fileName?: string;
	mountPath?: string;
	targetFilename?: string;
	contentHash?: string;
	projectionKey?: string;
	source?: string;
	snapshot?: Record<string, unknown>;
}

export interface RunInputListResponse {
	runId: string;
	items: RunInput[];
	total: number;
}

export interface RunOutput {
	id: string;
	runId: string;
	nodeId?: string;
	type: "node_outputs" | "logs" | "metrics" | string;
	refId?: string;
	uri?: string;
	snapshot?: Record<string, unknown>;
}

export interface RunOutputListResponse {
	runId: string;
	items: RunOutput[];
	total: number;
}

export interface RunChildrenResponse {
	runId: string;
	items: PipelineRun[];
	relations: RunRelation[];
	summary: RunChildSummary;
	total: number;
}

export interface RunRelation {
	id: string;
	parentRunId: string;
	childRunId: string;
	relationType: "batch_child" | string;
	source?: string;
	assetId?: string;
}

export interface RunChildSummary {
	total: number;
	statuses: Record<string, number>;
	aggregateStatus: string;
	activeCount: number;
	terminalCount: number;
	succeededCount: number;
	failedCount: number;
	cancelledCount: number;
	pendingCount: number;
	runningCount: number;
	suspendedCount: number;
	hasFailures: boolean;
	hasBlocking: boolean;
	topFailureReasons?: RunBlockingReason[];
}

export interface RunBlockingReason {
	reason: string;
	message?: string;
	count?: number;
	exampleRunId?: string;
	exampleAssetId?: string;
	source?: string;
}

export interface RunRuntime {
	runId: string;
	runtime: {
		runtimeType: "argo" | string;
		workflowName?: string;
		namespace?: string;
		uid?: string;
		status?: string;
		message?: string;
		executionTargetId?: string;
		targetSnapshot?: Record<string, unknown>;
		debugUrl?: string;
	};
}

export interface ListRunsOptions {
	view?: "summary" | "full";
	excludeBatch?: boolean;
	batchJobId?: string;
	status?: string;
	pipelineNodeId?: string;
	nodeStatus?: string;
	page?: number;
	pageSize?: number;
}

function buildRunListSearch(options?: ListRunsOptions) {
	const view = options?.view ?? "full";
	const search = new URLSearchParams();
	if (view === "summary") search.set("view", "summary");
	if (options?.excludeBatch) search.set("excludeBatch", "true");
	if (options?.batchJobId) search.set("batchJobId", options.batchJobId);
	if (options?.status) search.set("status", options.status);
	if (options?.pipelineNodeId) {
		search.set("pipelineNodeId", options.pipelineNodeId);
	}
	if (options?.nodeStatus) search.set("nodeStatus", options.nodeStatus);
	if (options?.page) search.set("page", String(options.page));
	if (options?.pageSize) search.set("pageSize", String(options.pageSize));
	return search.toString();
}

function withSearch(path: string, search: URLSearchParams | string) {
	const suffix = search.toString();
	return `${path}${suffix ? `?${suffix}` : ""}`;
}

export function createRunByTemplate(
	templateId: string,
	assetIds?: string[],
	targetId?: string,
	version?: number,
	configSelection?: DeployConfigSelection,
): Promise<DeployTemplateResponse> {
	return request<DeployTemplateResponse>(
		"POST",
		`/runs/template/${encodeURIComponent(templateId)}`,
		{
			asset_ids: assetIds ?? [],
			target_id: targetId,
			version,
			configSelection,
		},
	);
}

export function createRun(
	pipeline: Pipeline,
	name?: string,
	assetIds?: string[],
	targetId?: string,
	configSelection?: DeployConfigSelection,
): Promise<PipelineRun> {
	return request<PipelineRun>("POST", "/runs", {
		pipeline,
		name,
		asset_ids: assetIds ?? [],
		target_id: targetId,
		configSelection,
	});
}

export function listRuns(
	options?: ListRunsOptions,
): Promise<PipelineRunListResponse> {
	return request<PipelineRunListResponse>(
		"GET",
		withSearch("/runs", buildRunListSearch(options)),
	);
}

export function getRun(id: string): Promise<PipelineRun> {
	return request<PipelineRun>("GET", `/runs/${encodeURIComponent(id)}`);
}

export function getRunByWorkflowName(
	workflowName: string,
): Promise<PipelineRun> {
	return request<PipelineRun>(
		"GET",
		`/runs/by-workflow/${encodeURIComponent(workflowName)}`,
	);
}

export function deleteRun(id: string): Promise<void> {
	return request<void>("DELETE", `/runs/${encodeURIComponent(id)}`);
}

export function listRunEvents(
	runId: string,
	params?: {
		limit?: number;
		cursor?: number;
		subjectType?: string;
		eventType?: string;
		status?: string;
		q?: string;
		from?: string;
		to?: string;
	},
): Promise<PipelineRunEventsResponse> {
	const search = new URLSearchParams();
	if (params?.limit) search.set("limit", String(params.limit));
	if (params?.cursor) search.set("cursor", String(params.cursor));
	if (params?.subjectType) search.set("subjectType", params.subjectType);
	if (params?.eventType) search.set("eventType", params.eventType);
	if (params?.status) search.set("status", params.status);
	if (params?.q) search.set("q", params.q);
	if (params?.from) search.set("from", params.from);
	if (params?.to) search.set("to", params.to);
	return request<PipelineRunEventsResponse>(
		"GET",
		withSearch(`/runs/${encodeURIComponent(runId)}/events`, search),
	);
}

export function listRunNodes(runId: string): Promise<{
	items: PipelineRunNode[];
	total: number;
}> {
	return request<{ items: PipelineRunNode[]; total: number }>(
		"GET",
		`/runs/${encodeURIComponent(runId)}/nodes`,
	);
}

export function listRunAssetNodes(
	runId: string,
	params?: {
		limit?: number;
		cursor?: string;
		assetId?: string;
		nodeId?: string;
		status?: string;
		orderBy?: "cost" | "duration" | "status";
	},
): Promise<PipelineRunAssetNodeListResponse> {
	const search = new URLSearchParams();
	if (params?.limit) search.set("limit", String(params.limit));
	if (params?.cursor) search.set("cursor", params.cursor);
	if (params?.assetId) search.set("assetId", params.assetId);
	if (params?.nodeId) search.set("nodeId", params.nodeId);
	if (params?.status) search.set("status", params.status);
	if (params?.orderBy) search.set("orderBy", params.orderBy);
	return request<PipelineRunAssetNodeListResponse>(
		"GET",
		withSearch(`/runs/${encodeURIComponent(runId)}/asset-nodes`, search),
	);
}

export function getRunCostSummary(
	runId: string,
): Promise<PipelineRunCostSummary> {
	return request<PipelineRunCostSummary>(
		"GET",
		`/runs/${encodeURIComponent(runId)}/cost-summary`,
	);
}

export function listRunInputs(runId: string): Promise<RunInputListResponse> {
	return request<RunInputListResponse>(
		"GET",
		`/runs/${encodeURIComponent(runId)}/inputs`,
	);
}

export function listRunOutputs(runId: string): Promise<RunOutputListResponse> {
	return request<RunOutputListResponse>(
		"GET",
		`/runs/${encodeURIComponent(runId)}/outputs`,
	);
}

export function listRunChildren(runId: string): Promise<RunChildrenResponse> {
	return request<RunChildrenResponse>(
		"GET",
		`/runs/${encodeURIComponent(runId)}/children`,
	);
}

export function getRunRuntime(runId: string): Promise<RunRuntime> {
	return request<RunRuntime>(
		"GET",
		`/runs/${encodeURIComponent(runId)}/runtime`,
	);
}

export function getRunWatcherStatus(): Promise<PipelineRunWatcherState> {
	return request<PipelineRunWatcherState>("GET", "/runs/watcher/status");
}

export function retryRun(id: string): Promise<PipelineRun> {
	return request<PipelineRun>("POST", `/runs/${encodeURIComponent(id)}/retry`);
}

export function resubmitRun(id: string): Promise<PipelineRun> {
	return request<PipelineRun>(
		"POST",
		`/runs/${encodeURIComponent(id)}/resubmit`,
	);
}

export function rerunRun(id: string): Promise<PipelineRun> {
	return request<PipelineRun>("POST", `/runs/${encodeURIComponent(id)}/rerun`);
}

export function stopRun(id: string): Promise<{ message: string }> {
	return request<{ message: string }>(
		"POST",
		`/runs/${encodeURIComponent(id)}/stop`,
	);
}

export function suspendRun(id: string): Promise<{ message: string }> {
	return request<{ message: string }>(
		"POST",
		`/runs/${encodeURIComponent(id)}/suspend`,
	);
}

export function resumeRun(id: string): Promise<{ message: string }> {
	return request<{ message: string }>(
		"POST",
		`/runs/${encodeURIComponent(id)}/resume`,
	);
}

export function terminateRun(id: string): Promise<{ message: string }> {
	return request<{ message: string }>(
		"POST",
		`/runs/${encodeURIComponent(id)}/terminate`,
	);
}
