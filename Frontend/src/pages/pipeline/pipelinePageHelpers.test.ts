import { describe, expect, it } from "vitest";
import type { RegisteredComponent } from "../../components/pipeline/types";
import {
	createPipelineNode,
	dedupeComponents,
	defaultDeployWorkflowName,
} from "./pipelinePageHelpers";

function releaseComponent(
	overrides: Partial<RegisteredComponent> & {
		componentId: string;
		releaseId: string;
	},
): RegisteredComponent {
	return {
		id: overrides.releaseId,
		componentId: overrides.componentId,
		releaseId: overrides.releaseId,
		name: overrides.name ?? "smoke",
		image: "busybox:latest",
		command: ["sh", "-c", "echo ok"],
		args: [],
		cpu: "",
		memory: "",
		disk: "",
		source: "component-release",
		releaseLabel: overrides.releaseLabel ?? "v1",
		...overrides,
	};
}

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

describe("dedupeComponents", () => {
	it("keeps one release per component id", () => {
		const comps = dedupeComponents([
			releaseComponent({
				componentId: "comp-1",
				releaseId: "rel-1",
				releaseLabel: "v1",
			}),
			releaseComponent({
				componentId: "comp-1",
				releaseId: "rel-2",
				releaseLabel: "v2",
			}),
		]);

		expect(comps).toHaveLength(1);
		expect(comps[0]?.releaseId).toBe("rel-2");
	});
});

describe("defaultDeployWorkflowName", () => {
	it("uses pipeline timestamp prefix", () => {
		expect(defaultDeployWorkflowName(1_700_000_000_000)).toBe(
			"pipeline-1700000000000",
		);
	});
});

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
