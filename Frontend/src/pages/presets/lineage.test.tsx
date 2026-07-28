// CYB-4305: unit tests for the lineage preset.
//
// Focus is the pure helpers — histogram (skipped for depth=1, categorical
// for depth=all), column-set switching, filename token, extraStats mapping,
// and CSV row shape. Full render-path coverage sits on
// AssetDurationLookup.test.tsx (shell is preset-agnostic).

import { beforeEach, describe, expect, it, vi } from "vitest";
import type {
	AssetLineageBatchResponse,
	LineageBatchItem,
} from "../../api/assets";
import type { BatchLookupResponse } from "../../components/batch-lookup/types";
import {
	buildColumns,
	bucketRelationTypes,
	exportLineageCsv,
	type LineageFilters,
	lineagePreset,
} from "./lineage";

describe("buildColumns", () => {
	it("emits 7 cols at depth=1", () => {
		const cols = buildColumns(1);
		expect(cols.length).toBe(7);
		expect(cols.map((c) => c.key)).toEqual([
			"input_id",
			"asset_id",
			"grace_video_id",
			"parent_asset_id",
			"root_asset_id",
			"is_current",
			"revision",
		]);
	});

	it("emits 10 cols at depth=all (adds upstream/downstream/relation)", () => {
		const cols = buildColumns("all");
		expect(cols.length).toBe(10);
		expect(cols.map((c) => c.key)).toEqual([
			"input_id",
			"asset_id",
			"grace_video_id",
			"parent_asset_id",
			"root_asset_id",
			"is_current",
			"revision",
			"upstream_count",
			"downstream_count",
			"relation_types",
		]);
	});
});

describe("bucketRelationTypes", () => {
	const items: LineageBatchItem[] = [
		mkItem({ input_id: "a", relation_types: ["derive", "split"] }),
		mkItem({ input_id: "b", relation_types: ["derive"] }),
		mkItem({ input_id: "c", relation_types: ["merge", "derive"] }),
	];

	it("returns undefined when depth=1 (skips the histogram card)", () => {
		expect(bucketRelationTypes(items, { depth: 1 })).toBeUndefined();
	});

	it("returns undefined when filters are missing", () => {
		expect(bucketRelationTypes(items, undefined)).toBeUndefined();
	});

	it("aggregates per-item relation_types when depth=all", () => {
		const buckets = bucketRelationTypes(items, { depth: "all" });
		expect(buckets).toBeDefined();
		// Alphabetical order.
		expect(buckets?.map((b) => b.label)).toEqual(["derive", "merge", "split"]);
		expect(buckets?.map((b) => b.count)).toEqual([3, 1, 1]);
	});

	it("returns an empty bucket list when depth=all but no items carry relation_types", () => {
		const buckets = bucketRelationTypes(
			[mkItem({ input_id: "a", relation_types: [] })],
			{ depth: "all" },
		);
		expect(buckets).toEqual([]);
	});
});

describe("preset.columns callback", () => {
	it("switches with filters.depth", () => {
		expect(typeof lineagePreset.columns).toBe("function");
		if (typeof lineagePreset.columns !== "function") return;
		expect(lineagePreset.columns({ depth: 1 }).length).toBe(7);
		expect(lineagePreset.columns({ depth: "all" }).length).toBe(10);
	});
});

describe("preset.extraStats", () => {
	it("emits 5 cards from the stats bag", () => {
		const res: BatchLookupResponse<LineageBatchItem> = {
			items: [],
			missing_ids: [],
			filtered_out_ids: [],
			stats: {
				matched_count: 10,
				missing_count: 2,
				has_parent_count: 8,
				is_root_count: 2,
				orphan_count: 1,
				is_current_count: 7,
			},
		};
		const stats = lineagePreset.extraStats?.(res) ?? [];
		expect(stats.length).toBe(5);
		expect(stats.map((s) => s.label)).toEqual([
			"已匹配",
			"有 parent 的",
			"是 root 的",
			"孤儿",
			"当前版本",
		]);
		expect(stats.map((s) => s.value)).toEqual([10, 8, 2, 1, 7]);
	});

	it("defaults missing counters to 0", () => {
		const res: BatchLookupResponse<LineageBatchItem> = {
			items: [],
			missing_ids: [],
			filtered_out_ids: [],
			stats: {},
		};
		const stats = lineagePreset.extraStats?.(res) ?? [];
		expect(stats.every((s) => s.value === 0)).toBe(true);
	});
});

describe("preset.csv.filename", () => {
	it("encodes depth in the filename token", () => {
		expect(lineagePreset.csv.filename(42, { depth: 1 })).toBe(
			"asset-lineage-depth1-42.csv",
		);
		expect(lineagePreset.csv.filename(42, { depth: "all" })).toBe(
			"asset-lineage-depthAll-42.csv",
		);
	});

	it("defaults to depth1 when filters are missing", () => {
		expect(lineagePreset.csv.filename(0)).toBe("asset-lineage-depth1-0.csv");
	});
});

describe("preset.csv.row", () => {
	it("emits 11 columns and joins multi-value fields with '|'", () => {
		const row = lineagePreset.csv.row(
			mkItem({
				input_id: "aaaaaaaa",
				asset_id: "aaaaaaaa",
				grace_video_id: "019eda00-0000-0000-0000-000000000001",
				parent_asset_id: "parent01",
				root_asset_id: "root0001",
				logical_asset_id: "logical1",
				is_current: true,
				revision: 3,
				upstream_ids: ["p1", "p2"],
				downstream_ids: ["c1"],
				relation_types: ["derive", "split"],
			}),
		);
		expect(row).toEqual([
			"aaaaaaaa",
			"aaaaaaaa",
			"019eda00-0000-0000-0000-000000000001",
			"parent01",
			"root0001",
			"logical1",
			"true",
			3,
			"p1|p2",
			"c1",
			"derive|split",
		]);
	});

	it("empties null pointer fields cleanly", () => {
		const row = lineagePreset.csv.row(
			mkItem({
				input_id: "a",
				asset_id: "a",
				parent_asset_id: null,
				root_asset_id: null,
				logical_asset_id: null,
			}),
		);
		expect(row[3]).toBe(""); // parent
		expect(row[4]).toBe(""); // root
		expect(row[5]).toBe(""); // logical
	});
});

describe("exportLineageCsv", () => {
	beforeEach(() => {
		vi.restoreAllMocks();
		vi.spyOn(URL, "createObjectURL").mockReturnValue(
			"blob:mock" as unknown as string,
		);
		vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
	});

	it("filename embeds depth and matched count", () => {
		const clickSpy = vi
			.spyOn(HTMLAnchorElement.prototype, "click")
			.mockImplementation(() => {});
		exportLineageCsv(
			[mkItem({ input_id: "aaaaaaaa", asset_id: "aaaaaaaa" })],
			1,
			{ depth: "all" } as LineageFilters,
		);
		expect(clickSpy).toHaveBeenCalledTimes(1);
		const anchor = clickSpy.mock.instances[0] as HTMLAnchorElement;
		expect(anchor.download).toBe("asset-lineage-depthAll-1.csv");
	});
});

describe("fetch adapter (toResponseEnvelope indirectly)", () => {
	it("preserves items, missing_ids, and copies the 6 scalar counters", async () => {
		// mock the api call at the module boundary via vi.mock
		const mockRaw: AssetLineageBatchResponse = {
			items: [
				mkItem({ input_id: "a", asset_id: "a" }),
				mkItem({ input_id: "b", asset_id: "b" }),
			],
			missing_ids: ["zzzzzzzz"],
			filtered_out_ids: [],
			stats: {
				matched_count: 2,
				missing_count: 1,
				has_parent_count: 1,
				is_root_count: 1,
				orphan_count: 0,
				is_current_count: 2,
			},
		};
		const module_ = await import("../../api/assets");
		const spy = vi.spyOn(module_.assetsApi, "lookupLineage").mockResolvedValue(mockRaw);
		try {
			const resp = await lineagePreset.fetch({
				ids: ["a", "b", "zzzzzzzz"],
				id_type: "auto",
				filters: { depth: 1 },
			});
			expect(resp.items.length).toBe(2);
			expect(resp.missing_ids).toEqual(["zzzzzzzz"]);
			expect(resp.stats.matched_count).toBe(2);
			expect(resp.stats.is_root_count).toBe(1);
		} finally {
			spy.mockRestore();
		}
	});
});

// mkItem is a tiny factory used by these tests to construct a LineageBatchItem
// with sensible defaults. All fields default to null / [] so tests only
// mention the property they care about.
function mkItem(overrides: Partial<LineageBatchItem>): LineageBatchItem {
	return {
		input_id: "test",
		asset_id: "test",
		grace_video_id: undefined,
		parent_asset_id: null,
		root_asset_id: null,
		logical_asset_id: null,
		is_current: false,
		revision: 0,
		...overrides,
	};
}
