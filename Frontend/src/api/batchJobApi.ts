import { request } from "./pipelineClient";

export interface BatchJob {
	id: string;
	name: string;
	templateId: string;
	filterJson?: Record<string, unknown>;
	totalCount: number;
	completedCount: number;
	failedCount: number;
	status: "running" | "paused" | "completed" | "failed" | string;
	createdAt: string;
	updatedAt: string;
}

export interface CreateBatchJobRequest {
	name: string;
	templateId: string;
	assetIds: string[];
}

export function listBatchJobs(): Promise<BatchJob[]> {
	return request<{ items: BatchJob[] }>("GET", "/backfill").then((r) => r.items);
}

export function getBatchJob(id: string): Promise<BatchJob> {
	return request<{ job: BatchJob }>("GET", `/backfill/${id}`).then((r) => r.job);
}

export function createBatchJob(
	body: CreateBatchJobRequest,
): Promise<BatchJob> {
	return request<BatchJob>("POST", "/backfill", body);
}

export function pauseBatchJob(id: string): Promise<void> {
	return request<{ status: string }>("POST", `/backfill/${id}/pause`, {});
}

export function resumeBatchJob(id: string): Promise<void> {
	return request<{ status: string }>("POST", `/backfill/${id}/resume`, {});
}

export function retryFailedBatchItems(id: string): Promise<void> {
	return request<{ status: string }>(
		"POST",
		`/backfill/${id}/retry-failed`,
		{},
	);
}

export function batchJobProgress(job: BatchJob): number {
	if (!job.totalCount) return 0;
	return Math.round(
		((job.completedCount + job.failedCount) / job.totalCount) * 100,
	);
}
