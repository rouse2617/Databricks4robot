import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { PipelineRun } from "../api/pipelineApi";
import * as runApi from "../api/runApi";
import {
	exportAssetIdsCsv,
	extractAssetIds,
	fetchAllBatchAssetIds,
	fetchAssetIdsForBatches,
} from "./batchJobs";

// Minimal PipelineRun stub — CYB-3800 lib only reads assetIds so we keep the
// fixture narrow instead of hand-stamping every backend field.
function stubRun(overrides: Partial<PipelineRun>): PipelineRun {
	return {
		assetIds: [],
		...overrides,
	} as PipelineRun;
}

describe("extractAssetIds", () => {
	it("returns the union with trimming, dedup, and empty-string drop", () => {
		const runs = [
			stubRun({ assetIds: ["a1", "  a2 ", ""] }),
			stubRun({ assetIds: ["a2", "a3"] }),
			stubRun({ assetIds: undefined as unknown as string[] }),
			stubRun({ assetIds: [] }),
		];
		expect(extractAssetIds(runs)).toEqual(["a1", "a2", "a3"]);
	});

	it("returns an empty array when nothing has assetIds", () => {
		expect(
			extractAssetIds([
				stubRun({ assetIds: [] }),
				stubRun({ assetIds: undefined as unknown as string[] }),
			]),
		).toEqual([]);
	});
});

describe("exportAssetIdsCsv", () => {
	beforeEach(() => {
		// jsdom does not implement URL.createObjectURL / anchor.click; stub
		// them so the download-side-effect can be verified without a real DOM.
		vi.spyOn(URL, "createObjectURL").mockReturnValue(
			"blob:mock" as unknown as string,
		);
		vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
	});

	it("triggers an anchor.click with the expected filename", () => {
		const clickSpy = vi
			.spyOn(HTMLAnchorElement.prototype, "click")
			.mockImplementation(() => {});
		exportAssetIdsCsv(["a1", "a2"], "batch-multi-3");
		expect(clickSpy).toHaveBeenCalledTimes(1);
		const anchor = clickSpy.mock.instances[0] as HTMLAnchorElement;
		expect(anchor.download).toBe("asset-ids-batch-multi-3.csv");
	});
});

// listRunChildren is mocked per-suite below; reset between tests so queued
// mockResolvedValueOnce responses from one test don't leak into the next.
afterEach(() => {
	vi.restoreAllMocks();
});

describe("fetchAllBatchAssetIds", () => {
	it("paginates until total is covered and returns the deduped union", async () => {
		// total counts CHILD RUNS, not asset ids. Simulate 2 runs across 2
		// pages (1 run per page) so the loop has to make a second call.
		const spy = vi
			.spyOn(runApi, "listRunChildren")
			.mockResolvedValueOnce({
				runId: "b1",
				items: [stubRun({ assetIds: ["a1", "a2"] })],
				relations: [],
				summary: {} as never,
				total: 2,
			})
			.mockResolvedValueOnce({
				runId: "b1",
				items: [stubRun({ assetIds: ["a2", "a3"] })],
				relations: [],
				summary: {} as never,
				total: 2,
			});
		const ids = await fetchAllBatchAssetIds("b1");
		expect(ids).toEqual(["a1", "a2", "a3"]);
		expect(spy).toHaveBeenCalledTimes(2);
	});
});

describe("fetchAssetIdsForBatches", () => {
	it("unions asset_ids across multiple batches with dedup", async () => {
		vi.spyOn(runApi, "listRunChildren").mockImplementation(async (runId) => ({
			runId,
			items: [
				stubRun({
					assetIds: runId === "b1" ? ["shared", "a1"] : ["shared", "b1"],
				}),
			],
			relations: [],
			summary: {} as never,
			total: 1,
		}));
		const ids = await fetchAssetIdsForBatches(["b1", "b2"], 2);
		expect(ids.sort()).toEqual(["a1", "b1", "shared"]);
	});
});
