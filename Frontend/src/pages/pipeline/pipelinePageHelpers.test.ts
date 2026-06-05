import { describe, expect, it } from "vitest";
import { getAutoAddNodeScreenPosition } from "./pipelinePageHelpers";

describe("pipeline page helpers", () => {
	it("spaces auto-added nodes horizontally before wrapping", () => {
		const bounds = { left: 500, top: 100, width: 1200 };

		const first = getAutoAddNodeScreenPosition(0, bounds);
		const second = getAutoAddNodeScreenPosition(1, bounds);

		expect(first).toEqual({ x: 680, y: 240 });
		expect(second.x - first.x).toBeGreaterThanOrEqual(300);
		expect(second.y).toBe(first.y);
	});

	it("wraps auto-added nodes when canvas width is narrow", () => {
		const bounds = { left: 500, top: 100, width: 360 };

		const first = getAutoAddNodeScreenPosition(0, bounds);
		const second = getAutoAddNodeScreenPosition(1, bounds);

		expect(second.x).toBe(first.x);
		expect(second.y - first.y).toBeGreaterThanOrEqual(160);
	});
});
