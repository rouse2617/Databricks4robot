import { describe, expect, it } from "vitest";
import {
	createPipelineNodeId,
	maxLegacyStepCounter,
	parseLegacyStepCounter,
} from "./node-id";

describe("createPipelineNodeId", () => {
	it("returns ids prefixed with node-", () => {
		expect(createPipelineNodeId().startsWith("node-")).toBe(true);
	});

	it("generates unique ids", () => {
		const ids = new Set(
			Array.from({ length: 20 }, () => createPipelineNodeId()),
		);
		expect(ids.size).toBe(20);
	});
});

describe("parseLegacyStepCounter", () => {
	it("parses step-N suffix", () => {
		expect(parseLegacyStepCounter("step-3")).toBe(3);
	});

	it("returns null for non-legacy ids", () => {
		expect(parseLegacyStepCounter("node-abc")).toBeNull();
		expect(parseLegacyStepCounter("")).toBeNull();
	});
});

describe("maxLegacyStepCounter", () => {
	it("returns highest legacy counter", () => {
		expect(maxLegacyStepCounter(["step-1", "node-x", "step-9", "step-4"])).toBe(
			9,
		);
	});

	it("returns 0 when no legacy ids", () => {
		expect(maxLegacyStepCounter(["node-a", "node-b"])).toBe(0);
	});
});
