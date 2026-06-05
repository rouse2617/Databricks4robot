import { describe, expect, it } from "vitest";
import type { WorkflowDetail } from "../api/workflowApi";
import type { Pipeline } from "../components/pipeline/types";
import { applyPipelineNodeDisplayNames } from "./workflowNodeNames";

const pipeline: Pipeline = {
	name: "component-name-pipeline",
	nodes: [
		{
			id: "step-1",
			component: {
				name: "cloudrun-e2e-test",
				image: "alpine:latest",
			},
		},
	],
	edges: [],
};

function workflowWithNode(
	overrides: Partial<WorkflowDetail["nodes"][number]>,
): WorkflowDetail {
	return {
		name: "wf",
		status: "Running",
		createdAt: "2026-06-05T00:00:00Z",
		nodes: [
			{
				id: "step-1",
				name: "wf.step-step-1",
				displayName: "step-step-1",
				templateName: "step-step-1",
				type: "Pod",
				phase: "Pending",
				...overrides,
			},
		],
	};
}

describe("applyPipelineNodeDisplayNames", () => {
	it("uses the pipeline component name for argo step-prefixed nodes", () => {
		const result = applyPipelineNodeDisplayNames(
			workflowWithNode({}),
			pipeline,
		);

		expect(result?.nodes[0].displayName).toBe("cloudrun-e2e-test");
		expect(result?.nodes[0].technicalDisplayName).toBe("step-step-1");
		expect(result?.nodes[0].id).toBe("step-1");
	});

	it("matches static dag task ids generated from pipeline node ids", () => {
		const result = applyPipelineNodeDisplayNames(
			workflowWithNode({
				id: "step-step-1",
				name: "step-step-1",
			}),
			pipeline,
		);

		expect(result?.nodes[0].displayName).toBe("cloudrun-e2e-test");
		expect(result?.nodes[0].technicalDisplayName).toBe("step-step-1");
		expect(result?.nodes[0].id).toBe("step-step-1");
	});

	it("falls back without mutating nodes when pipeline metadata is unavailable", () => {
		const workflow = workflowWithNode({});
		const result = applyPipelineNodeDisplayNames(workflow, undefined);

		expect(result).toBe(workflow);
		expect(result?.nodes[0].displayName).toBe("step-step-1");
	});
});
