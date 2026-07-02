import { describe, expect, it } from "vitest";
import type { BatchJob } from "../api/batchJobApi";
import { sortBatchJobsByCreatedDesc } from "../lib/batchJobs";

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
});
