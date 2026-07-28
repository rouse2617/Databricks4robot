import type {
	DeployConfigSelection,
	PipelineRunNodeProgress,
} from "./pipelineApi";
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
	createdBy?: string;
	createdAt: string;
	updatedAt: string;
	finishedAt?: string;
	// Subtask run span (earliest subtask start / latest subtask finish), returned
	// by the list endpoint. Used to show a real run duration excluding submit/
	// queue/pause waiting, and as a completion-time fallback when finishedAt is unset.
	runStartedAt?: string;
	runFinishedAt?: string;
}

export interface CreateBatchJobRequest {
	name: string;
	templateId: string;
	assetIds: string[];
	targetId?: string;
	templateVersion?: number;
	pilotCount?: number;
	configSelection?: DeployConfigSelection;
	// Per-dispatch Argo priority override. Omit to inherit the target pool's
	// default; -100/0/100 = low/normal/high.
	priority?: number;
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
	retriedCount?: number;
	templateId: string;
	templateVersion?: number;
	skipped: Array<{ itemId: string; reason: string }>;
}

export interface ListBatchJobsParams {
	createdBy?: string;
	status?: string;
	q?: string;
}

export function listBatchJobs(
	params?: ListBatchJobsParams,
): Promise<BatchJob[]> {
	const search = new URLSearchParams();
	if (params?.createdBy) search.set("createdBy", params.createdBy);
	if (params?.status) search.set("status", params.status);
	if (params?.q) search.set("q", params.q);
	const qs = search.toString();
	const path = qs ? `/backfill?${qs}` : "/backfill";
	return request<{ items: BatchJob[] }>("GET", path).then((r) => r.items);
}

export function getBatchJob(id: string): Promise<BatchJob> {
	return request<{ job: BatchJob }>("GET", `/backfill/${id}`).then(
		(r) => r.job,
	);
}

export function createBatchJob(body: CreateBatchJobRequest): Promise<BatchJob> {
	return request<BatchJob>("POST", "/backfill", body);
}

export interface PauseBatchJobOptions {
	stopRunning?: boolean;
}

export interface PauseBatchJobResult {
	status: string;
	stoppedCount?: number;
	stopFailedCount?: number;
}

export function pauseBatchJob(
	id: string,
	options?: PauseBatchJobOptions,
): Promise<PauseBatchJobResult> {
	return request<PauseBatchJobResult>("POST", `/backfill/${id}/pause`, {
		stopRunning: options?.stopRunning ?? false,
	});
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

export function batchJobProgressStatus(
	job: BatchJob,
): "success" | "exception" | "active" | "normal" {
	if (job.status === "completed") {
		return job.failedCount > 0 ? "exception" : "success";
	}
	if (job.status === "failed") return "exception";
	if (job.status === "paused") return "normal";
	return "active";
}

export type BatchJobDisplayStatus =
	| "running"
	| "paused"
	| "completed"
	| "partial_failure"
	| "failed";

/** Minimal item-count shape the batch status is derived from. */
export interface BatchStatusCounts {
	completedCount: number;
	failedCount: number;
	totalCount: number;
	/** Backend lifecycle status, used only for paused / in-flight fallback. */
	status?: string;
}

/**
 * Canonical batch-job display status, derived from item counts rather than the
 * raw backend `status` field. CYB-4012: the backend rollup can report
 * "completed" for a 0-success / all-failed batch, so the list (which trusted
 * the raw status) rendered a green "已完成" for a fully-failed batch — hiding
 * silent failures — while the detail page, which derives from counts, showed
 * "失败". The mapping was also non-monotonic (near-identical ratios read as
 * opposite states). Both views now share this one count-based function.
 *
 * Monotonic in failure rate — once every item is processed:
 *   failed == 0            -> completed        (fully clean)
 *   0 < failed < total     -> partial_failure  (mixed: some worked)
 *   completed == 0         -> failed           (nothing worked)
 * so a higher failure rate can never read as more successful, and only a
 * fully-clean batch is ever "completed".
 */
export function deriveBatchJobStatus(
	job: BatchStatusCounts,
): BatchJobDisplayStatus {
	if (job.status === "paused") return "paused";
	const processed = job.completedCount + job.failedCount;
	// All items processed (matches the detail page's allFinished check).
	if (processed >= job.totalCount) {
		if (job.failedCount === 0) return "completed";
		if (job.completedCount === 0) return "failed";
		return "partial_failure";
	}
	// Not fully processed: honor an explicit terminal failure from the backend;
	// otherwise it is still in flight. Never surface "completed" before every
	// item is processed — that premature-complete is the CYB-4012 mask.
	if (job.status === "failed") return "failed";
	return "running";
}

export interface BackfillItemAttempt {
	runId: string;
	attemptNo: number;
	status: string;
	templateVersion?: number;
	workflowName?: string;
	message?: string;
	nodeProgress?: PipelineRunNodeProgress;
	isCurrent: boolean;
	startedAt?: string;
	finishedAt?: string;
	createdAt: string;
}

export interface BackfillItemAttemptsResult {
	itemId: string;
	assetId: string;
	currentRunId?: string;
	attempts: BackfillItemAttempt[];
}

export interface ValidateBackfillAssetsResult {
	registered: string[];
	unknown: string[];
}

export function getBatchItemAttempts(
	jobId: string,
	params: { itemId?: string; assetId?: string },
): Promise<BackfillItemAttemptsResult> {
	const search = new URLSearchParams();
	if (params.itemId) search.set("itemId", params.itemId);
	if (params.assetId) search.set("assetId", params.assetId);
	return request<BackfillItemAttemptsResult>(
		"GET",
		`/backfill/${jobId}/attempts?${search.toString()}`,
	);
}

export function validateBackfillAssets(
	assetIds: string[],
): Promise<ValidateBackfillAssetsResult> {
	return request<ValidateBackfillAssetsResult>(
		"POST",
		"/backfill/validate-assets",
		{ assetIds },
	);
}
