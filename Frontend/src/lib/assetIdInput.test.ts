import { describe, expect, it } from "vitest";
import { mergeAssetIds, parseAssetIdInput } from "./assetIdInput";

describe("parseAssetIdInput", () => {
	it("splits comma and whitespace separated ids", () => {
		expect(parseAssetIdInput("a1, a2\na3")).toEqual(["a1", "a2", "a3"]);
	});

	it("ignores empty segments", () => {
		expect(parseAssetIdInput("  a1 , , a2  ")).toEqual(["a1", "a2"]);
	});
});

describe("mergeAssetIds", () => {
	it("deduplicates while preserving order", () => {
		expect(mergeAssetIds(["a1"], ["a2", "a1", "a3"])).toEqual([
			"a1",
			"a2",
			"a3",
		]);
	});
});
