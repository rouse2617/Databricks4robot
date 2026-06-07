import { describe, expect, it } from "vitest";
import { getShortBuildRef } from "./appVersion";

describe("getShortBuildRef", () => {
	it("keeps short refs unchanged", () => {
		expect(getShortBuildRef("a714077")).toBe("a714077");
	});

	it("shortens long commit refs", () => {
		expect(getShortBuildRef("buildref1234567890abcdef")).toBe("buildre");
	});

	it("preserves dirty suffixes", () => {
		expect(getShortBuildRef("buildref1234567890abcdef-dirty")).toBe(
			"buildre-dirty",
		);
	});

	it("keeps fallback refs readable", () => {
		expect(getShortBuildRef("local")).toBe("local");
		expect(getShortBuildRef("")).toBe("unknown");
	});
});
