import { describe, expect, it } from "vitest";
import {
	normalizeLogContent,
	prepareVisibleLogContent,
} from "./workflowLogView";

describe("workflowLogView", () => {
	it("normalizes carriage returns and keeps existing line boundaries", () => {
		expect(normalizeLogContent("a\r\nb\rc", null)).toBe("a\nb\nc");
	});

	it("splits concatenated pod-prefixed argo logs into readable lines", () => {
		const content = "pod-abc message 1pod-abc message 2pod-abc message 3";

		expect(normalizeLogContent(content, { podName: "pod-abc" })).toBe(
			"pod-abc message 1\npod-abc message 2\npod-abc message 3",
		);
	});

	it("renders only the tail of very long logs by character and line limits", () => {
		const lines = Array.from({ length: 10 }, (_, index) => `line-${index + 1}`);
		const visible = prepareVisibleLogContent(lines.join("\n"), null, 30, 3);

		expect(visible.truncated).toBe(true);
		expect(visible.totalLines).toBe(10);
		expect(visible.hiddenChars).toBeGreaterThan(0);
		expect(visible.hiddenLines).toBeGreaterThan(0);
		expect(visible.content).toBe("line-8\nline-9\nline-10");
	});
});
