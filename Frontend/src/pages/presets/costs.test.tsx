// CYB-4306: unit tests for the costs preset.
//
// Focuses on the pure helpers — histogram bucketing, filename token, csv
// export, column-set switching, and filter validation. The full paste/query
// flow through BatchAssetLookup is already covered by
// AssetDurationLookup.test.tsx; the shell is preset-agnostic so we do not
// re-test the render path here.

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { AssetCostItem } from "../../api/assets";
import {
	bucketCostItems,
	buildColumns,
	costsPreset,
	exportCostsCsv,
	formatUsd,
	validateCostsFilters,
} from "./costs";

describe("bucketCostItems", () => {
	it("groups items into the fixed 5 cost buckets", () => {
		const items: AssetCostItem[] = [
			mkItem({ input_id: "a", total_cost_usd: 0 }), // <$0.10
			mkItem({ input_id: "b", total_cost_usd: 0.05 }), // <$0.10
			mkItem({ input_id: "c", total_cost_usd: 0.5 }), // $0.10-$1
			mkItem({ input_id: "d", total_cost_usd: 5 }), // $1-$10
			mkItem({ input_id: "e", total_cost_usd: 50 }), // $10-$100
			mkItem({ input_id: "f", total_cost_usd: 500 }), // $100+
			mkItem({ input_id: "g", total_cost_usd: 0.099 }), // boundary — still <$0.10
			mkItem({ input_id: "h", total_cost_usd: 0.1 }), // boundary — first $0.10-$1
		];
		const buckets = bucketCostItems(items);
		expect(buckets.map((b) => b.count)).toEqual([3, 2, 1, 1, 1]);
	});
});

describe("formatUsd", () => {
	it("uses 4 decimals below $10 and 2 decimals otherwise", () => {
		expect(formatUsd(0.0001)).toBe("$0.0001");
		expect(formatUsd(0.1234)).toBe("$0.1234");
		expect(formatUsd(1.23)).toBe("$1.2300");
		expect(formatUsd(10)).toBe("$10.00");
		expect(formatUsd(1234.5)).toBe("$1234.50");
		expect(formatUsd(undefined)).toBe("-");
		expect(formatUsd(Number.NaN)).toBe("-");
	});
});

describe("validateCostsFilters", () => {
	it("requires both start and end", () => {
		expect(
			validateCostsFilters({
				start_at: undefined,
				end_at: undefined,
				group_by: "asset",
			}),
		).toBe("开始时间和结束时间必填");
		expect(
			validateCostsFilters({
				start_at: "2026-07-21T00:00:00Z",
				end_at: undefined,
				group_by: "asset",
			}),
		).toBe("开始时间和结束时间必填");
	});

	it("rejects end <= start", () => {
		expect(
			validateCostsFilters({
				start_at: "2026-07-28T00:00:00Z",
				end_at: "2026-07-21T00:00:00Z",
				group_by: "asset",
			}),
		).toBe("结束时间必须晚于开始时间");
		// Equal → still error (must be strictly after).
		expect(
			validateCostsFilters({
				start_at: "2026-07-21T00:00:00Z",
				end_at: "2026-07-21T00:00:00Z",
				group_by: "asset",
			}),
		).toBe("结束时间必须晚于开始时间");
	});

	it("rejects windows > 90 days", () => {
		expect(
			validateCostsFilters({
				start_at: "2026-01-01T00:00:00Z",
				end_at: "2026-05-01T00:00:00Z", // ~120 days
				group_by: "asset",
			}),
		).toBe("时间窗不能超过 90 天");
	});

	it("returns null for a valid 7-day window", () => {
		expect(
			validateCostsFilters({
				start_at: "2026-07-21T00:00:00Z",
				end_at: "2026-07-28T00:00:00Z",
				group_by: "asset",
			}),
		).toBeNull();
	});
});

describe("buildColumns", () => {
	it("emits 7 columns in asset mode", () => {
		const cols = buildColumns("asset");
		expect(cols.length).toBe(7);
		expect(cols.map((c) => c.key)).toEqual([
			"input_id",
			"asset_id",
			"grace_video_id",
			"total_cost_usd",
			"gpu_min",
			"cpu_min",
			"run_count",
		]);
	});

	it("emits 8 columns in asset_algo mode (algo_key column added)", () => {
		const cols = buildColumns("asset_algo");
		expect(cols.length).toBe(8);
		expect(cols.map((c) => c.key)).toEqual([
			"input_id",
			"asset_id",
			"grace_video_id",
			"algo_key",
			"total_cost_usd",
			"gpu_min",
			"cpu_min",
			"run_count",
		]);
	});
});

describe("preset.columns", () => {
	it("is a function whose column set switches with group_by", () => {
		expect(typeof costsPreset.columns).toBe("function");
		if (typeof costsPreset.columns !== "function") return;
		const assetCols = costsPreset.columns({
			start_at: undefined,
			end_at: undefined,
			group_by: "asset",
		});
		const algoCols = costsPreset.columns({
			start_at: undefined,
			end_at: undefined,
			group_by: "asset_algo",
		});
		expect(assetCols.length).toBe(7);
		expect(algoCols.length).toBe(8);
	});
});

describe("exportCostsCsv", () => {
	beforeEach(() => {
		// Restore any spies leaked from the previous test — otherwise the
		// HTMLAnchorElement.prototype.click spy from the first test still
		// records instances into a fresh spy variable.
		vi.restoreAllMocks();
		vi.spyOn(URL, "createObjectURL").mockReturnValue(
			"blob:mock" as unknown as string,
		);
		vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
	});

	it("filename embeds the start/end date and matched-count", () => {
		const clickSpy = vi
			.spyOn(HTMLAnchorElement.prototype, "click")
			.mockImplementation(() => {});
		exportCostsCsv(
			[
				mkItem({
					input_id: "aaaaaaaa",
					total_cost_usd: 1.25,
					gpu_sec: 60,
					cpu_sec: 30,
					run_count: 2,
				}),
			],
			1,
			{
				start_at: "2026-07-21T00:00:00Z",
				end_at: "2026-07-28T00:00:00Z",
				group_by: "asset",
			},
		);
		expect(clickSpy).toHaveBeenCalledTimes(1);
		const anchor = clickSpy.mock.instances[0] as HTMLAnchorElement;
		expect(anchor.download).toBe("asset-costs-2026-07-21-2026-07-28-1.csv");
	});

	it("falls back to 'unknown' tokens when filters are missing", () => {
		const clickSpy = vi
			.spyOn(HTMLAnchorElement.prototype, "click")
			.mockImplementation(() => {});
		exportCostsCsv([], 0, undefined);
		const anchor = clickSpy.mock.instances[0] as HTMLAnchorElement;
		expect(anchor.download).toBe("asset-costs-unknown-unknown-0.csv");
	});
});

// mkItem is a tiny factory used by these tests to construct an AssetCostItem
// with sensible defaults. All fields default to zero so tests only mention
// the property they care about.
function mkItem(overrides: Partial<AssetCostItem>): AssetCostItem {
	return {
		input_id: "test",
		asset_id: "test",
		grace_video_id: undefined,
		total_cost_usd: 0,
		gpu_sec: 0,
		cpu_sec: 0,
		gpu_min: 0,
		cpu_min: 0,
		run_count: 0,
		by_algo: null,
		...overrides,
	};
}
