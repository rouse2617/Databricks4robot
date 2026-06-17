import { describe, expect, it } from "vitest";
import type { WorkflowNodeStatus } from "../api/workflowApi";
import {
	buildPipelineNodeLabelLookup,
	getWorkflowNodeDisplayText,
} from "./workflowNodeDisplay";

describe("workflowNodeDisplay", () => {
	it("maps Argo template names to pipeline component labels", () => {
		const lookup = buildPipelineNodeLabelLookup({
			name: "my-pipeline",
			nodes: [
				{
					id: "step-1",
					component: {
						name: "cyb1823-ui-terminal-sleep-1148",
						image: "alpine:3.20",
					},
				},
			],
		});
		const node: WorkflowNodeStatus = {
			id: "wf.step-step-1",
			name: "wf.step-step-1",
			displayName: "step-step-1",
			templateName: "step-step-1",
			type: "Pod",
			phase: "Running",
		};
		expect(getWorkflowNodeDisplayText(node, lookup)).toBe(
			"cyb1823-ui-terminal-sleep-1148",
		);
	});

	it("falls back to Argo display name without pipeline labels", () => {
		const node: WorkflowNodeStatus = {
			id: "wf.step-1",
			name: "wf.step-1",
			displayName: "step-1",
			templateName: "step-1",
			type: "Pod",
			phase: "Running",
		};
		expect(getWorkflowNodeDisplayText(node)).toBe("step-1");
	});
});
