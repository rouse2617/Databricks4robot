import { describe, expect, it } from "vitest";
import {
	buildContentModel,
	detectCommonLinePrefix,
	isErrorLogLine,
	normalizeLogContent,
	prepareVisibleLogContent,
	stripLinePrefixes,
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

	it("detects a dominant repeated pod prefix across lines", () => {
		const lines = [
			"wf-step-1234567890-abc: starting",
			"wf-step-1234567890-abc: processing",
			"wf-step-1234567890-abc: done",
		];
		expect(detectCommonLinePrefix(lines)).toBe("wf-step-1234567890-abc:");
	});

	it("ignores short or non-dominant first words", () => {
		expect(detectCommonLinePrefix(["ok one", "ok two", "different line"])).toBe(
			"",
		);
	});

	it("strips the detected prefix from every line", () => {
		const prefix = "wf-step-1234567890-abc:";
		const lines = [`${prefix} starting`, `${prefix} done`, "no-prefix line"];
		expect(stripLinePrefixes(lines, prefix)).toEqual([
			"starting",
			"done",
			"no-prefix line",
		]);
	});

	it("flags error/failure lines and ignores ordinary lines", () => {
		expect(isErrorLogLine("PermissionDenied: 403 Secret Manager API")).toBe(
			true,
		);
		expect(isErrorLogLine("process exited with exit status 1")).toBe(true);
		expect(isErrorLogLine("step completed successfully")).toBe(false);
	});

	it("collects error line indexes when building the content model", () => {
		const model = buildContentModel(
			["starting", "ERROR failed to connect", "retrying", "fatal: boom"],
			"",
			false,
		);
		expect(model.errorLineIndexes).toEqual([1, 3]);
		expect(model.lines[1].isError).toBe(true);
		expect(model.lines[0].isError).toBe(false);
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
