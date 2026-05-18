import { describe, expect, it } from "vitest";
import { isCanonicalAssetId } from "./assetId";

describe("isCanonicalAssetId", () => {
	it("accepts exactly 8 alphanumeric characters", () => {
		expect(isCanonicalAssetId("7VBGimAO")).toBe(true);
		expect(isCanonicalAssetId("  Ab12Cd34  ")).toBe(true);
	});

	it("rejects empty, wrong length, and non-alphanumeric values", () => {
		expect(isCanonicalAssetId("")).toBe(false);
		expect(isCanonicalAssetId("1234567")).toBe(false);
		expect(isCanonicalAssetId("123456789")).toBe(false);
		expect(isCanonicalAssetId("abcd-123")).toBe(false);
	});
});
