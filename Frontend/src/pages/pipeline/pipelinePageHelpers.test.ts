import { describe, expect, it } from "vitest";
import type { RegisteredComponent } from "../../components/pipeline/types";
import { createPipelineNode } from "./pipelinePageHelpers";

const sampleComponent: RegisteredComponent = {
	id: "comp-1",
	componentId: "comp-1",
	name: "Sample Step",
	type: "container",
	source: "registry",
	image: "python:3.11",
	command: [],
	args: [],
	env: [],
	inputPorts: [{ name: "input", type: "asset" }],
	outputPorts: [{ name: "output", type: "asset" }],
	cpu: "",
	memory: "",
	disk: "",
	gpu: "",
	computeTier: "",
};

describe("createPipelineNode", () => {
	it("assigns stable node- prefixed ids", () => {
		const node = createPipelineNode(sampleComponent, 10, 20);
		expect(node.id.startsWith("node-")).toBe(true);
	});

	it("stores component and release refs on node data", () => {
		const node = createPipelineNode(
			{ ...sampleComponent, releaseId: "rel-1", releaseLabel: "v2" },
			0,
			0,
		);
		expect(node.data.componentId).toBe("comp-1");
		expect(node.data.releaseId).toBe("rel-1");
		expect(node.data.componentVersionLabel).toBe("v2");
	});
});
