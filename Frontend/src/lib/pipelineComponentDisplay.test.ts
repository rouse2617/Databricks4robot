import { describe, expect, it } from "vitest";
import {
	formatCommitDisplay,
	normalizeGitCommit,
} from "./pipelineComponentDisplay";

describe("normalizeGitCommit", () => {
	it("strips zero padding from short SHAs stored as 40-char hex", () => {
		expect(
			normalizeGitCommit("b68162f0000000000000000000000000000000000"),
		).toBe("b68162f");
	});

	it("keeps full-length SHAs unchanged", () => {
		const full = "a86a258f91fad9c64a0123456789abcdef01234567";
		expect(normalizeGitCommit(full)).toBe(full);
	});

	it("keeps non-hex values unchanged", () => {
		expect(normalizeGitCommit("main-abc123")).toBe("main-abc123");
	});
});

describe("formatCommitDisplay", () => {
	it("shows short commits without ellipsis", () => {
		expect(formatCommitDisplay("b68162f")).toBe("b68162f");
	});
});
