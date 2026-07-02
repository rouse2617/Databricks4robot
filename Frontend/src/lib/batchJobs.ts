import dayjs from "dayjs";
import type { BatchJob } from "../api/batchJobApi";

export function batchJobCreatedAtMs(job: BatchJob): number {
	const value = dayjs(job.createdAt).valueOf();
	return Number.isFinite(value) ? value : 0;
}

export function sortBatchJobsByCreatedDesc(jobs: BatchJob[]): BatchJob[] {
	return [...jobs].sort(
		(a, b) => batchJobCreatedAtMs(b) - batchJobCreatedAtMs(a),
	);
}
