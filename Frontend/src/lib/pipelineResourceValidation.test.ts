import { describe, expect, it } from "vitest";
import { validateResourceQuantity } from "./pipelineResourceValidation";

describe("pipeline resource validation", () => {
	it("requires units for memory and disk", () => {
		expect(validateResourceQuantity("memory", "1")).toContain("内存");
		expect(validateResourceQuantity("disk", "1")).toContain("磁盘");
		expect(validateResourceQuantity("memory", "512Mi")).toBeNull();
		expect(validateResourceQuantity("disk", "20Gi")).toBeNull();
	});

	it("requires cpu memory and disk quantities to be positive", () => {
		expect(validateResourceQuantity("cpu", "0")).toContain("CPU");
		expect(validateResourceQuantity("memory", "0Gi")).toContain("内存");
		expect(validateResourceQuantity("disk", "0Gi")).toContain("磁盘");
		expect(validateResourceQuantity("cpu", "-1")).toContain("CPU");
	});

	it("allows cpu cores and millicores while rejecting malformed values", () => {
		expect(validateResourceQuantity("cpu", "1")).toBeNull();
		expect(validateResourceQuantity("cpu", "500m")).toBeNull();
		expect(validateResourceQuantity("cpu", "abc")).toContain("CPU");
	});

	it("requires gpu to be a non-negative integer", () => {
		expect(validateResourceQuantity("gpu", "0")).toBeNull();
		expect(validateResourceQuantity("gpu", "1")).toBeNull();
		expect(validateResourceQuantity("gpu", "0.5")).toContain("GPU");
	});
});
