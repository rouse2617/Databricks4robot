import { request } from "./pipelineClient";

export interface BatchJob {
	id: string;
	name: string;
	templateId: string;
	templateVersion?: number;
	filterJson?: Record<string, unknown>;
	totalCount: number;
	completedCount: number;
	failedCount: number;
	pilotCount?: number;
	pilotPhase?: "none" | "running" | "review" | "done" | string;
	status: "running" | "paused" | "completed" | "failed" | string;
	createdAt: string;
	updatedAt: string;
}

export interface CreateBatchJobRequest {
	name: string;
	templateId: string;
	assetIds: string[];
	templateVersion?: number;
	pilotCount?: number;
}

export interface BatchNodeSummary {
	batchJobId: string;
	templateId: string;
	templateVersion?: number;
	subtasks: {
		total: number;
		completed: number;
		failed: number;
		running: number;
		pending: number;
		paused: boolean;
	};
	nodes: BatchNodeSummaryNode[];
	dataCoverage: {
		runsWithNodeRows: number;
		runsTotal: number;
		complete: boolean;
	};
	generatedAt: string;
}

export interface BatchNodeSummaryNode {
	pipelineNodeId: string;
	displayName: string;
	dagOrder: number;
	counts: Record<string, number>;
	attempted: number;
	failureRate: number;
}

export interface BatchNodeFailureItem {
	backfillItemId: string;
	assetId: string;
	runId: string;
	workflowName: string;
	pipelineNodeId: string;
	displayName: string;
	status: string;
	message?: string;
	startedAt?: string;
	finishedAt?: string;
}

export interface BatchNodeFailureList {
	items: BatchNodeFailureItem[];
	total: number;
	page: number;
	pageSize: number;
}

export interface RerunBatchJobRequest {
	scope:
		| "failed"
		| "pending"
		| "incomplete"
		| "completed"
		| "custom"
		| "node_failed";
	templateId?: string;
	templateVersion?: number;
	itemIds?: string[];
	assetIds?: string[];
	pipelineNodeId?: string;
	dryRun?: boolean;
}

export interface RerunBatchJobResult {
	status: string;
	dryRun: boolean;
	matchedCount: number;
	templateId: string;
	templateVersion?: number;
	skipped: Array<{ itemId: string; reason: string }>;
}

export function listBatchJobs(): Promise<BatchJob[]> {
	return request<{ items: BatchJob[] }>("GET", "/backfill").then(
		(r) => r.items,
	);
}

export function getBatchJob(id: string): Promise<BatchJob> {
	return request<{ job: BatchJob }>("GET", `/backfill/${id}`).then(
		(r) => r.job,
	);
}

export function createBatchJob(body: CreateBatchJobRequest): Promise<BatchJob> {
	return request<BatchJob>("POST", "/backfill", body);
}

export function pauseBatchJob(id: string): Promise<void> {
	return request<{ status: string }>("POST", `/backfill/${id}/pause`, {}).then(
		() => undefined,
	);
}

export function resumeBatchJob(id: string): Promise<void> {
	return request<{ status: string }>("POST", `/backfill/${id}/resume`, {}).then(
		() => undefined,
	);
}

export function retryFailedBatchItems(id: string): Promise<void> {
	return request<{ status: string }>(
		"POST",
		`/backfill/${id}/retry-failed`,
		{},
	).then(() => undefined);
}

export function getBatchNodeSummary(id: string): Promise<BatchNodeSummary> {
	return request<BatchNodeSummary>("GET", `/backfill/${id}/node-summary`);
}

export function listBatchNodeFailures(
	id: string,
	params: {
		pipelineNodeId: string;
		status?: string;
		page?: number;
		pageSize?: number;
		q?: string;
	},
): Promise<BatchNodeFailureList> {
	const search = new URLSearchParams();
	search.set("pipelineNodeId", params.pipelineNodeId);
	if (params.status) search.set("status", params.status);
	if (params.page) search.set("page", String(params.page));
	if (params.pageSize) search.set("pageSize", String(params.pageSize));
	if (params.q) search.set("q", params.q);
	return request<BatchNodeFailureList>(
		"GET",
		`/backfill/${id}/node-failures?${search.toString()}`,
	);
}

export function rerunBatchJob(
	id: string,
	body: RerunBatchJobRequest,
): Promise<RerunBatchJobResult> {
	return request<RerunBatchJobResult>("POST", `/backfill/${id}/rerun`, body);
}

export function continueFullBatchJob(id: string): Promise<void> {
	return request<{ status: string }>(
		"POST",
		`/backfill/${id}/continue-full`,
		{},
	).then(() => undefined);
}

export interface UploadBackfillResultRequest {
	assetId: string;
	reportId: string;
	version: string;
	manifest?: Record<string, unknown>;
	result?: Record<string, unknown>;
}

export interface UploadBackfillResultResponse {
	assetId: string;
	reportId: string;
	version: string;
	algoKey: string;
	status: string;
}

export function uploadBackfillResult(
	body: UploadBackfillResultRequest,
): Promise<UploadBackfillResultResponse> {
	return request<UploadBackfillResultResponse>(
		"POST",
		"/backfill/results",
		body,
	);
}

export function batchJobProgress(job: BatchJob): number {
	if (!job.totalCount) return 0;
	return Math.round(
		((job.completedCount + job.failedCount) / job.totalCount) * 100,
	);
}
