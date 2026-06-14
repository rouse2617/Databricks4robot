import type { Pipeline } from "../components/pipeline/types";
import { request } from "./pipelineClient";

export interface PipelineTemplate {
	id: string;
	name: string;
	version: number;
	versionCount?: number;
	activeVersion?: number;
	scope?: string;
	owner?: string;
	pipeline: Pipeline;
	nodeCount: number;
	createdAt: string;
	updatedAt?: string;
}

export interface Deployment {
	id: string;
	templateId?: string;
	templateVersion?: number;
	pipelineName: string;
	workflowName: string;
	status: string;
	nodeCount: number;
	scope?: string;
	owner?: string;
	assetIds?: string[];
	assetCount?: number;
	executionTarget?: ExecutionTarget;
	createdAt: string;
	finishedAt?: string;
	manifest?: string;
	pipelineJSON?: Pipeline;
}

export interface PipelineRun extends Deployment {
	executionTargetId?: string;
	targetSnapshot?: Record<string, unknown>;
	argoNamespace?: string;
	argoWorkflowUid?: string;
	message?: string;
	noAssetRun?: boolean;
	startedAt?: string;
	totalEstimatedCost?: number | null;
	batchJobId?: string;
}

export interface PipelineRunListResponse {
	items: PipelineRun[];
	total: number;
	page?: number;
	pageSize?: number;
}

export interface PipelineRunEvent {
	id: string;
	runId: string;
	workflowName?: string;
	eventType: string;
	subjectType: "run" | "workflow" | "node" | "pod" | string;
	subjectId: string;
	status?: string;
	message?: string;
	reason?: string;
	payload?: Record<string, unknown>;
	sequence: number;
	occurredAt: string;
	observedAt: string;
	createdAt: string;
}

export interface PipelineRunEventsResponse {
	items: PipelineRunEvent[];
	nextCursor?: number;
	total: number;
}

export interface PipelineRunWatcherState {
	id: string;
	lastSyncedAt?: string;
	lastScanStartedAt?: string;
	lastScanFinishedAt?: string;
	lastSuccessAt?: string;
	lastErrorAt?: string;
	activeScanLimit: number;
	lastSyncedRunCount: number;
	consecutiveFailures: number;
	totalScans: number;
	totalErrors: number;
	scanLagSeconds?: number;
	lastError?: string;
	healthy: boolean;
	stale: boolean;
	updatedAt?: string;
}

export interface PipelineRunAssetNode {
	id: string;
	runId: string;
	assetId: string;
	pipelineNodeId: string;
	argoNodeId?: string;
	displayName?: string;
	status?: string;
	message?: string;
	podName?: string;
	logRef?: string;
	estimatedCostUsd?: number;
	costSource: string;
	startedAt?: string;
	finishedAt?: string;
	updatedAt: string;
}

export interface PipelineRunAssetNodeListResponse {
	items: PipelineRunAssetNode[];
	nextCursor?: string;
	total: number;
	summary: {
		assetCount: number;
		nodeCount: number;
		statuses: Record<string, number>;
		totalEstimatedCostUsd?: number;
		costSource: string;
	};
}

export interface PipelineRunCostSummary {
	runId: string;
	totalEstimatedCostUsd?: number;
	costSource: string;
	nodeSummaries: Array<{
		nodeId: string;
		displayName?: string;
		status?: string;
		podCount: number;
		estimatedCostUsd?: number;
		costSource: string;
		durationSeconds?: number;
	}>;
	assetNodeSummaries: Array<{
		assetId: string;
		nodeId: string;
		displayName?: string;
		status?: string;
		estimatedCostUsd?: number;
		costSource: string;
	}>;
	generatedAt: string;
}

export interface ExecutionTarget {
	id: string;
	name: string;
	cluster: string;
	namespace: string;
	argoServerConfigured: boolean;
	status: "available" | "unavailable";
	isDefault: boolean;
	description?: string;
}

export function previewDeploy(
	pipeline: Pipeline,
): Promise<{ manifest: string }> {
	return request<{ manifest?: string }>("POST", "/deploy?dryRun=true", {
		pipeline,
	}).then((resp) => ({ manifest: resp.manifest || "" }));
}

export function listPipelines(
	params: ListPipelinesParams = {},
): Promise<ListPipelinesResponse> {
	const page = params.page ?? 1;
	const pageSize = params.pageSize ?? 20;
	const qs = new URLSearchParams();
	if (params.page) qs.set("page", String(params.page));
	if (params.pageSize) qs.set("page_size", String(params.pageSize));
	if (params.q) qs.set("q", params.q);
	if (params.scope) qs.set("scope", params.scope);
	if (params.sort) qs.set("sort", params.sort);
	const query = qs.toString();
	return request<
		Partial<ListPipelinesResponse> & {
			items?: PipelineTemplate[];
			page_size?: number;
		}
	>("GET", `/pipelines${query ? `?${query}` : ""}`).then((raw) => {
		const items = raw.items ?? [];
		const resolvedPageSize = raw.pageSize ?? raw.page_size ?? pageSize;
		const hasServerTotal = typeof raw.total === "number";
		const total = hasServerTotal ? raw.total : items.length;
		let normalizedItems = items;
		if (!hasServerTotal && items.length > resolvedPageSize) {
			const start = (page - 1) * resolvedPageSize;
			normalizedItems = items.slice(start, start + resolvedPageSize);
		}
		return {
			items: normalizedItems,
			total,
			page: raw.page ?? page,
			pageSize: resolvedPageSize,
		};
	});
}

export interface ListPipelinesParams {
	page?: number;
	pageSize?: number;
	q?: string;
	scope?: string;
	sort?: "updated_at_desc" | "name_asc" | "name_desc" | "created_at_desc";
}

export interface ListPipelinesResponse {
	items: PipelineTemplate[];
	total: number;
	page: number;
	pageSize: number;
}

export interface PipelineDiffNode {
	id: string;
	component?: Record<string, unknown>;
}

export interface PipelineDiffEdge {
	source: string;
	target: string;
}

export interface PipelineDiff {
	added_nodes: PipelineDiffNode[];
	removed_nodes: PipelineDiffNode[];
	modified_nodes: PipelineDiffNode[];
	added_edges: PipelineDiffEdge[];
	removed_edges: PipelineDiffEdge[];
}

export function getPipelineDiff(
	id1: string,
	id2: string,
): Promise<PipelineDiff> {
	return request<PipelineDiff>("GET", `/pipelines/${id1}/diff/${id2}`);
}

export function getPipeline(id: string): Promise<PipelineTemplate> {
	return request<PipelineTemplate>("GET", `/pipelines/${id}`);
}

export function listPipelineVersions(
	templateId: string,
): Promise<PipelineTemplate[]> {
	return request<{ items: PipelineTemplate[] }>(
		"GET",
		`/pipelines/${templateId}/versions`,
	).then((r) => r.items);
}

export function savePipeline(
	name: string,
	pipeline: Pipeline,
): Promise<PipelineTemplate> {
	return request<PipelineTemplate>("POST", "/pipelines", { name, pipeline });
}

export function deletePipeline(id: string): Promise<void> {
	return request<void>("DELETE", `/pipelines/${id}`);
}

export function promotePipeline(id: string): Promise<PipelineTemplate> {
	return request<PipelineTemplate>("POST", `/pipelines/${id}/promote`);
}

export function deployTemplate(
	templateId: string,
	assetIds?: string[],
	targetId?: string,
	version?: number,
): Promise<DeployTemplateResponse> {
	return request<DeployTemplateResponse>(
		"POST",
		`/pipeline-runs/template/${templateId}`,
		{
			asset_ids: assetIds ?? [],
			target_id: targetId,
			version,
		},
	);
}

export type DeployTemplateResponse =
	| Deployment
	| { items: Deployment[]; total?: number };

export function normalizeDeployResults(
	result: DeployTemplateResponse,
): Deployment[] {
	if (
		result &&
		typeof result === "object" &&
		"items" in result &&
		Array.isArray(result.items)
	) {
		return result.items;
	}
	return [result as Deployment];
}

export function listDeployments(): Promise<Deployment[]> {
	return request<{ items: Deployment[] }>("GET", "/deployments").then(
		(r) => r.items,
	);
}

export function getPipelineRunByWorkflowName(
	workflowName: string,
): Promise<PipelineRun> {
	return request<PipelineRun>(
		"GET",
		`/pipeline-runs/by-workflow/${encodeURIComponent(workflowName)}`,
	);
}

export function listPipelineRuns(options?: {
	view?: "summary" | "full";
	excludeBatch?: boolean;
	batchJobId?: string;
	status?: string;
	page?: number;
	pageSize?: number;
}): Promise<PipelineRunListResponse> {
	const view = options?.view ?? "full";
	const search = new URLSearchParams();
	if (view === "summary") search.set("view", "summary");
	if (options?.excludeBatch) search.set("excludeBatch", "true");
	if (options?.batchJobId) search.set("batchJobId", options.batchJobId);
	if (options?.status) search.set("status", options.status);
	if (options?.page) search.set("page", String(options.page));
	if (options?.pageSize) search.set("pageSize", String(options.pageSize));
	const suffix = search.toString() ? `?${search.toString()}` : "";
	return request<PipelineRunListResponse>("GET", `/pipeline-runs${suffix}`);
}

export function listPipelineRunEvents(
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
	const suffix = search.toString() ? `?${search.toString()}` : "";
	return request<PipelineRunEventsResponse>(
		"GET",
		`/pipeline-runs/${runId}/events${suffix}`,
	);
}

export function getPipelineRunWatcherStatus(): Promise<PipelineRunWatcherState> {
	return request<PipelineRunWatcherState>(
		"GET",
		"/pipeline-runs/watcher/status",
	);
}

export function listPipelineRunAssetNodes(
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
	const suffix = search.toString() ? `?${search.toString()}` : "";
	return request<PipelineRunAssetNodeListResponse>(
		"GET",
		`/pipeline-runs/${runId}/asset-nodes${suffix}`,
	);
}

export function getPipelineRunCostSummary(
	runId: string,
): Promise<PipelineRunCostSummary> {
	return request<PipelineRunCostSummary>(
		"GET",
		`/pipeline-runs/${runId}/cost-summary`,
	);
}

export function listExecutionTargets(): Promise<ExecutionTarget[]> {
	return request<{ items: ExecutionTarget[] }>(
		"GET",
		"/execution-targets",
	).then((r) => r.items);
}

export function deleteDeployment(id: string): Promise<void> {
	return request<void>("DELETE", `/deployments/${id}`);
}

export function retryDeployment(id: string): Promise<Deployment> {
	return request<Deployment>("POST", `/deployments/${id}/retry`, {});
}
