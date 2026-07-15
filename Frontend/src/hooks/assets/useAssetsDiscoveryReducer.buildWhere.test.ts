// CYB-3385 — buildStructuredQueryWhere pure tests. Kept in its own file so the
// tightly-focused grouping semantics don't share a describe block with the
// broader async reducer stack.
import { describe, expect, it } from "vitest";
import { defaultAssetsDiscoveryState } from "../../lib/assets/assetsDiscoveryTypes";
import { buildStructuredQueryWhere } from "./useAssetsDiscoveryReducer";

function withFilters(
	chips: Array<{ field: string; op: string; value: string | string[] }>,
) {
	return {
		...defaultAssetsDiscoveryState.queryState,
		activeFilters: chips.map((c, i) => ({
			id: `chip-${i}`,
			source: "facet" as const,
			...c,
		})),
	};
}

describe("buildStructuredQueryWhere — CYB-3385 grouping", () => {
	it("returns undefined when no chips", () => {
		expect(buildStructuredQueryWhere(withFilters([]))).toBeUndefined();
	});

	it("keeps a single chip as plain {pred:...} (no needless or wrapper)", () => {
		const out = buildStructuredQueryWhere(
			withFilters([{ field: "asset_type", op: "eq", value: "segment" }]),
		);
		expect(out).toEqual({
			pred: { field: "asset_type", op: "eq", value: "segment" },
		});
	});

	it("groups two same-field same-op chips into an OR (multi-value OR)", () => {
		const out = buildStructuredQueryWhere(
			withFilters([
				{ field: "asset_type", op: "eq", value: "segment" },
				{ field: "asset_type", op: "eq", value: "action" },
			]),
		);
		expect(out).toEqual({
			or: [
				{ pred: { field: "asset_type", op: "eq", value: "segment" } },
				{ pred: { field: "asset_type", op: "eq", value: "action" } },
			],
		});
	});

	it("keeps different-field chips as AND (inter-field intersect)", () => {
		const out = buildStructuredQueryWhere(
			withFilters([
				{ field: "asset_type", op: "eq", value: "segment" },
				{ field: "env", op: "eq", value: "warehouse" },
			]),
		);
		expect(out).toEqual({
			and: [
				{ pred: { field: "asset_type", op: "eq", value: "segment" } },
				{ pred: { field: "env", op: "eq", value: "warehouse" } },
			],
		});
	});

	it("mixes: multi-value OR + single other field → and([or, pred])", () => {
		const out = buildStructuredQueryWhere(
			withFilters([
				{ field: "asset_type", op: "eq", value: "segment" },
				{ field: "asset_type", op: "eq", value: "action" },
				{ field: "env", op: "eq", value: "warehouse" },
			]),
		);
		expect(out).toEqual({
			and: [
				{
					or: [
						{ pred: { field: "asset_type", op: "eq", value: "segment" } },
						{ pred: { field: "asset_type", op: "eq", value: "action" } },
					],
				},
				{ pred: { field: "env", op: "eq", value: "warehouse" } },
			],
		});
	});

	it("splits eq and ne on the same field into separate groups", () => {
		const out = buildStructuredQueryWhere(
			withFilters([
				{ field: "owner", op: "eq", value: "alice" },
				{ field: "owner", op: "eq", value: "bob" },
				{ field: "owner", op: "ne", value: "charlie" },
			]),
		);
		expect(out).toEqual({
			and: [
				{
					or: [
						{ pred: { field: "owner", op: "eq", value: "alice" } },
						{ pred: { field: "owner", op: "eq", value: "bob" } },
					],
				},
				{ pred: { field: "owner", op: "ne", value: "charlie" } },
			],
		});
	});

	it("preserves declaration order across groups", () => {
		const out = buildStructuredQueryWhere(
			withFilters([
				{ field: "b", op: "eq", value: "1" },
				{ field: "a", op: "eq", value: "1" },
				{ field: "b", op: "eq", value: "2" },
			]),
		) as { and: Array<{ or?: unknown; pred?: { field: string } }> };
		expect(out.and[0]).toEqual({
			or: [
				{ pred: { field: "b", op: "eq", value: "1" } },
				{ pred: { field: "b", op: "eq", value: "2" } },
			],
		});
		expect(out.and[1]).toEqual({
			pred: { field: "a", op: "eq", value: "1" },
		});
	});
});
