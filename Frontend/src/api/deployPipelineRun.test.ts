import { beforeEach, describe, expect, it, vi } from "vitest";

const mockCreateBatchJob = vi.fn();
const mockDeployTemplate = vi.fn();

vi.mock("./batchJobApi", () => ({
	createBatchJob: (...args: unknown[]) => mockCreateBatchJob(...args),
}));

vi.mock("./pipelineApi", () => ({
	deployTemplate: (...args: unknown[]) => mockDeployTemplate(...args),
	normalizeDeployResults: (result: unknown) =>
		Array.isArray((result as { items?: unknown[] })?.items)
			? (result as { items: unknown[] }).items
			: [result],
}));

import { deployPipelineForAssets } from "./deployPipelineRun";

describe("deployPipelineForAssets", () => {
	beforeEach(() => {
		mockCreateBatchJob.mockReset();
		mockDeployTemplate.mockReset();
	});

	it("creates batch job when at least two assets are selected", async () => {
		mockCreateBatchJob.mockResolvedValue({
			id: "job-1",
			totalCount: 2,
		});

		const result = await deployPipelineForAssets("tpl-1", ["a1", "a2"], {
			batchName: "demo-batch",
		});

		expect(result.mode).toBe("batch");
		expect(mockCreateBatchJob).toHaveBeenCalledWith({
			name: "demo-batch",
			templateId: "tpl-1",
			assetIds: ["a1", "a2"],
		});
		expect(mockDeployTemplate).not.toHaveBeenCalled();
	});

	it("deploys single run for one asset", async () => {
		mockDeployTemplate.mockResolvedValue({ id: "run-1" });

		const result = await deployPipelineForAssets("tpl-1", ["a1"]);

		expect(result.mode).toBe("single");
		expect(mockDeployTemplate).toHaveBeenCalledWith(
			"tpl-1",
			["a1"],
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
	});
});
