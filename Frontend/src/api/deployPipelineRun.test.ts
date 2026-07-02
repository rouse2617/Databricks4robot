import { beforeEach, describe, expect, it, vi } from "vitest";

const mockCreateBatchJob = vi.fn();
const mockCreateRunByTemplate = vi.fn();

vi.mock("./batchJobApi", () => ({
	createBatchJob: (...args: unknown[]) => mockCreateBatchJob(...args),
}));

vi.mock("./runApi", () => ({
	createRunByTemplate: (...args: unknown[]) => mockCreateRunByTemplate(...args),
}));

vi.mock("./pipelineApi", () => ({
	normalizeDeployResults: (result: unknown) =>
		Array.isArray((result as { items?: unknown[] })?.items)
			? (result as { items: unknown[] }).items
			: [result],
}));

import { deployPipelineForAssets } from "./deployPipelineRun";

describe("deployPipelineForAssets", () => {
	beforeEach(() => {
		mockCreateBatchJob.mockReset();
		mockCreateRunByTemplate.mockReset();
	});

	it("creates batch job when at least two assets are selected", async () => {
		mockCreateBatchJob.mockResolvedValue({
			id: "job-1",
			totalCount: 2,
		});

		const result = await deployPipelineForAssets("tpl-1", ["a1", "a2"], {
			batchName: "demo-batch",
			targetId: "video-proc-dev",
		});

		expect(result.mode).toBe("batch");
		expect(mockCreateBatchJob).toHaveBeenCalledWith({
			name: "demo-batch",
			templateId: "tpl-1",
			assetIds: ["a1", "a2"],
			targetId: "video-proc-dev",
		});
		expect(mockCreateRunByTemplate).not.toHaveBeenCalled();
	});

	it("deploys single run for one asset", async () => {
		mockCreateRunByTemplate.mockResolvedValue({ id: "run-1" });

		const result = await deployPipelineForAssets("tpl-1", ["a1"]);

		expect(result.mode).toBe("single");
		expect(mockCreateRunByTemplate).toHaveBeenCalledWith(
			"tpl-1",
			["a1"],
			undefined,
			undefined,
			undefined,
		);
	});

	it("rejects batches above the asset limit", async () => {
		const assetIds = Array.from(
			{ length: 10_001 },
			(_, index) => `asset-${index}`,
		);
		await expect(deployPipelineForAssets("tpl-1", assetIds)).rejects.toThrow(
			/最多支持 10,000 个资产/,
		);
		expect(mockCreateBatchJob).not.toHaveBeenCalled();
		expect(mockCreateRunByTemplate).not.toHaveBeenCalled();
	});
});
