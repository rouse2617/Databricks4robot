import { describe, expect, it } from "vitest";
import type { BatchJob } from "../api/batchJobApi";
import {
	batchJobCompletionAt,
	batchJobRunDurationSeconds,
	formatDurationSeconds,
	sortBatchJobsByCreatedDesc,
} from "../lib/batchJobs";

function batchJob(overrides: Partial<BatchJob>): BatchJob {
	return {
		id: "batch-id",
		name: "batch",
		templateId: "template-id",
		totalCount: 1,
		completedCount: 0,
		failedCount: 0,
		status: "running",
		createdAt: "2026-06-19T00:00:00Z",
		updatedAt: "2026-06-19T00:00:00Z",
		...overrides,
	};
}

describe("BatchJobList", () => {
	it("sorts batch jobs by creation time newest first", () => {
		const sorted = sortBatchJobsByCreatedDesc([
			batchJob({ id: "old", createdAt: "2026-06-19T17:55:31+08:00" }),
			batchJob({ id: "new", createdAt: "2026-06-19T18:02:04+08:00" }),
		]);

		expect(sorted.map((job) => job.id)).toEqual(["new", "old"]);
	});

	describe("completion time (CYB-3108)", () => {
		it("uses the job's own finishedAt when present", () => {
			const job = batchJob({
				status: "completed",
				finishedAt: "2026-07-07T10:00:00Z",
				runFinishedAt: "2026-07-07T09:59:00Z",
			});
			expect(batchJobCompletionAt(job)).toBe("2026-07-07T10:00:00Z");
		});

		it("falls back to latest subtask finish when finishedAt is unstamped", () => {
			const job = batchJob({
				status: "failed",
				finishedAt: undefined,
				runFinishedAt: "2026-07-07T09:59:00Z",
			});
			expect(batchJobCompletionAt(job)).toBe("2026-07-07T09:59:00Z");
		});

		it("returns undefined for a running job even if subtasks have finished", () => {
			const job = batchJob({
				status: "running",
				finishedAt: undefined,
				runFinishedAt: "2026-07-07T09:59:00Z",
			});
			expect(batchJobCompletionAt(job)).toBeUndefined();
		});
	});

	describe("real run duration (CYB-3108)", () => {
		it("spans first subtask start to last subtask finish (excludes queue/pause)", () => {
			// Created long before it actually ran: wall-clock would be ~1h, real run 5m.
			const job = batchJob({
				status: "completed",
				createdAt: "2026-07-07T09:00:00Z",
				runStartedAt: "2026-07-07T09:55:00Z",
				runFinishedAt: "2026-07-07T10:00:00Z",
			});
			expect(batchJobRunDurationSeconds(job)).toBe(300);
			expect(formatDurationSeconds(300)).toBe("5m 0s");
		});

		it("is null while running or without a run span", () => {
			expect(
				batchJobRunDurationSeconds(
					batchJob({
						status: "running",
						runStartedAt: "2026-07-07T09:55:00Z",
						runFinishedAt: "2026-07-07T10:00:00Z",
					}),
				),
			).toBeNull();
			expect(
				batchJobRunDurationSeconds(
					batchJob({ status: "failed", runStartedAt: undefined }),
				),
			).toBeNull();
		});
	});
});
