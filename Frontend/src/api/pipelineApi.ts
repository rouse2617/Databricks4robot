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
	batchRunId?: string;
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

export function listPipelines(): Promise<PipelineTemplate[]> {
	return request<{ items: PipelineTemplate[] }>("GET", "/pipelines").then(
		(r) => r.items,
	);
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

export interface BatchDeployResult {
	batchId: string;
	items: PipelineRun[];
	failed?: Array<{ assetId: string; error: string }>;
}

export function deployTemplate(
	templateId: string,
	assetIds?: string[],
	targetId?: string,
	version?: number,
): Promise<Deployment> {
	return request<Deployment>("POST", `/pipeline-runs/template/${templateId}`, {
		asset_ids: assetIds ?? [],
		target_id: targetId,
		version,
	});
}

export function batchDeployTemplate(
	templateId: string,
	assetIds: string[],
	targetId?: string,
	version?: number,
): Promise<BatchDeployResult> {
	return request<BatchDeployResult>(
		"POST",
		`/pipeline-runs/template/${templateId}/batch`,
		{
			asset_ids: assetIds,
			target_id: targetId,
			version,
		},
	);
}

export function listDeployments(): Promise<Deployment[]> {
	return request<{ items: Deployment[] }>("GET", "/deployments").then(
		(r) => r.items,
	);
}

export function listPipelineRuns(): Promise<PipelineRun[]> {
	return request<{ items: PipelineRun[] }>("GET", "/pipeline-runs").then(
		(r) => r.items,
	);
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
