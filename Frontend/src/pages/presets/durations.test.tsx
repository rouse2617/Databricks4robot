// CYB-4333: unit tests for the durations preset.
//
// Lifted from the deleted `AssetDurationLookup.test.tsx` so the pure helpers
// (parseIdBlob, bucketItems, exportDurationsCsv) keep their unit coverage
// once the durations shim page went away. Mirrors the layout of
// `costs.test.tsx` — pure helpers only, no shell mounting; the full
// paste/query render path lives in `BatchLookupPage.test.tsx`.

import { beforeEach, describe, expect, it, vi } from "vitest";
import { parseIdBlob } from "../../components/batch-lookup/types";
import { bucketItems, exportDurationsCsv } from "./durations";

describe("parseIdBlob", () => {
	it("splits on whitespace and commas, trims, dedupes, and reports duplicates", () => {
		const result = parseIdBlob("a1, a2\n a3\n\na1 a4,,a2");
		expect(result.ids).toEqual(["a1", "a2", "a3", "a4"]);
		expect(result.duplicates).toBe(2); // a1 and a2 each show up twice
	});
	it("returns empty on empty input", () => {
		expect(parseIdBlob("")).toEqual({ ids: [], duplicates: 0 });
		expect(parseIdBlob("   \n  ")).toEqual({ ids: [], duplicates: 0 });
	});
});

describe("bucketItems", () => {
	it("groups items into the fixed 5 buckets", () => {
		const items = [
			{
				input_id: "a",
				asset_id: "a",
				duration_ms: 0,
				duration_sec: 0,
				formatted: "0s",
			},
			{
				input_id: "b",
				asset_id: "b",
				duration_ms: 30_000,
				duration_sec: 30,
				formatted: "30s",
			}, // <1min
			{
				input_id: "c",
				asset_id: "c",
				duration_ms: 300_000,
				duration_sec: 300,
				formatted: "5m 0s",
			}, // 1-10min
			{
				input_id: "d",
				asset_id: "d",
				duration_ms: 1_200_000,
				duration_sec: 1200,
				formatted: "20m 0s",
			}, // 10-30min
			{
				input_id: "e",
				asset_id: "e",
				duration_ms: 2_700_000,
				duration_sec: 2700,
				formatted: "45m 0s",
			}, // 30-60min
			{
				input_id: "f",
				asset_id: "f",
				duration_ms: 7_200_000,
				duration_sec: 7200,
				formatted: "2h 0m",
			}, // 60min+
		];
		const buckets = bucketItems(items);
		expect(buckets.map((b) => b.count)).toEqual([2, 1, 1, 1, 1]);
	});
});

describe("exportDurationsCsv", () => {
	beforeEach(() => {
		vi.restoreAllMocks();
		vi.spyOn(URL, "createObjectURL").mockReturnValue(
			"blob:mock" as unknown as string,
		);
		vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
	});

	it("triggers an anchor.click with the expected filename", () => {
		const clickSpy = vi
			.spyOn(HTMLAnchorElement.prototype, "click")
			.mockImplementation(() => {});
		exportDurationsCsv(
			[
				{
					input_id: "aaaaaaaa",
					asset_id: "aaaaaaaa",
					duration_ms: 1_000,
					duration_sec: 1,
					formatted: "1s",
				},
			],
			1,
		);
		expect(clickSpy).toHaveBeenCalledTimes(1);
		const anchor = clickSpy.mock.instances[0] as HTMLAnchorElement;
		expect(anchor.download).toBe("asset-durations-1.csv");
	});
});
