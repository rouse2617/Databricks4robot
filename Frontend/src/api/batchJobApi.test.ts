import { describe, expect, it } from "vitest";
import {
	type BatchJobDisplayStatus,
	type BatchStatusCounts,
	deriveBatchJobStatus,
} from "./batchJobApi";

// CYB-4012: the batch list showed a green "已完成" for 0-success / all-failed
// batches (it trusted the raw backend `status`) while the detail page — which
// derives from counts — showed "失败". These pin the single shared count-based
// derivation both views now use: monotonic in failure rate, and only a fully
// clean batch is ever "completed".

describe("deriveBatchJobStatus", () => {
	const cases: Array<{
		name: string;
		job: BatchStatusCounts;
		want: BatchJobDisplayStatus;
	}> = [
		{
			name: "all succeeded -> completed",
			job: { completedCount: 5, failedCount: 0, totalCount: 5, status: "completed" },
			want: "completed",
		},
		{
			name: "all failed -> failed (the CYB-4012 bug: was green 已完成)",
			job: { completedCount: 0, failedCount: 3, totalCount: 3, status: "completed" },
			want: "failed",
		},
		{
			name: "all failed even if backend says running -> failed",
			job: { completedCount: 0, failedCount: 3, totalCount: 3, status: "running" },
			want: "failed",
		},
		{
			name: "mixed, mostly success -> partial_failure",
			job: { completedCount: 426, failedCount: 5, totalCount: 431, status: "failed" },
			want: "partial_failure",
		},
		{
			name: "mixed, mostly failure but one success -> partial_failure",
			job: { completedCount: 1, failedCount: 62, totalCount: 63, status: "completed" },
			want: "partial_failure",
		},
		{
			name: "not fully processed, backend running -> running",
			job: { completedCount: 2, failedCount: 1, totalCount: 10, status: "running" },
			want: "running",
		},
		{
			name: "not fully processed but backend prematurely completed -> running (never mask)",
			job: { completedCount: 2, failedCount: 1, totalCount: 10, status: "completed" },
			want: "running",
		},
		{
			name: "not fully processed but backend terminal-failed -> failed",
			job: { completedCount: 2, failedCount: 1, totalCount: 10, status: "failed" },
			want: "failed",
		},
		{
			name: "paused wins regardless of counts",
			job: { completedCount: 4, failedCount: 0, totalCount: 10, status: "paused" },
			want: "paused",
		},
		{
			name: "empty batch (total 0) -> completed (matches detail allFinished)",
			job: { completedCount: 0, failedCount: 0, totalCount: 0, status: "running" },
			want: "completed",
		},
		{
			name: "boundary: exactly all processed with one failure -> partial_failure",
			job: { completedCount: 1, failedCount: 1, totalCount: 2, status: "completed" },
			want: "partial_failure",
		},
	];

	for (const c of cases) {
		it(c.name, () => {
			expect(deriveBatchJobStatus(c.job)).toBe(c.want);
		});
	}

	it("is monotonic in failure rate on a finished batch: completed -> partial_failure -> failed, never back", () => {
		const total = 10;
		let sawPartial = false;
		let sawFailed = false;
		const seq: BatchJobDisplayStatus[] = [];
		for (let failed = 0; failed <= total; failed++) {
			const status = deriveBatchJobStatus({
				completedCount: total - failed,
				failedCount: failed,
				totalCount: total,
				status: "completed",
			});
			seq.push(status);
			// "completed" may only appear before any failure has been seen.
			if (status === "completed") {
				expect(sawPartial || sawFailed).toBe(false);
			}
			if (status === "partial_failure") sawPartial = true;
			if (status === "failed") sawFailed = true;
		}
		// failure rate 0 -> completed, 0<rate<1 -> partial_failure, rate 1 -> failed.
		expect(seq[0]).toBe("completed");
		expect(seq[total]).toBe("failed");
		expect(seq.slice(1, total).every((s) => s === "partial_failure")).toBe(true);
	});
});
