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

	it("infers and splits repeated log prefixes when pod name is unavailable", () => {
		const content = "worker-01 message 1worker-01 message 2worker-01 message 3";

		expect(normalizeLogContent(content, null)).toBe(
			"worker-01 message 1\nworker-01 message 2\nworker-01 message 3",
		);
	});

	it("falls back to inferred prefixes when selected pod name differs from argo log prefix", () => {
		const content =
			"wf-step-123456 firstwf-step-123456 secondwf-step-123456 third";

		expect(normalizeLogContent(content, { podName: "wf.step" })).toBe(
			"wf-step-123456 first\nwf-step-123456 second\nwf-step-123456 third",
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
