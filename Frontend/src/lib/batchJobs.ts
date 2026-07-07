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

function isTerminalBatchJob(job: BatchJob): boolean {
	return job.status === "completed" || job.status === "failed";
}

/**
 * Completion timestamp for display: the job's own finishedAt, or the latest
 * subtask finish (runFinishedAt) as a fallback for terminal jobs whose
 * finished_at was never stamped (legacy / async-finalized batches). Undefined
 * for non-terminal jobs (so running/paused show "—").
 */
export function batchJobCompletionAt(job: BatchJob): string | undefined {
	return (
		job.finishedAt ?? (isTerminalBatchJob(job) ? job.runFinishedAt : undefined)
	);
}

/**
 * Real run duration in seconds — the subtask run span (first subtask start →
 * last subtask finish), which excludes submit/queue/pause waiting. This is NOT
 * finished_at - created_at (wall-clock incl. queueing). Null when the batch is
 * not terminal, has no run span, or the span is negative.
 */
export function batchJobRunDurationSeconds(job: BatchJob): number | null {
	if (!isTerminalBatchJob(job) || !job.runStartedAt || !job.runFinishedAt) {
		return null;
	}
	const secs = dayjs(job.runFinishedAt).diff(dayjs(job.runStartedAt), "second");
	return secs >= 0 ? secs : null;
}

export function formatDurationSeconds(secs: number): string {
	if (secs < 60) return `${secs}s`;
	const m = Math.floor(secs / 60);
	const s = secs % 60;
	return `${m}m ${s}s`;
}
