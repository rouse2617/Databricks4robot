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

	it("maps generated node ids to Argo step-node ids", () => {
		const lookup = buildPipelineNodeLabelLookup({
			name: "my-pipeline",
			nodes: [
				{
					id: "node-8ed5c569-7632-45c4-9ed4-fe5afbb0204b",
					component: {
						name: "hand-detect-yolov26m",
						image: "example.com/hand-detect:latest",
					},
				},
			],
		});
		const node: WorkflowNodeStatus = {
			id: "wf-asset-1652292909",
			name: "wf-asset.step-node-8ed5c569-7632-45c4-9ed4-fe5afbb0204b",
			displayName: "step-node-8ed5c569-7632-45c4-9ed4-fe5afbb0204b",
			templateName: "step-node-8ed5c569-7632-45c4-9ed4-fe5afbb0204b",
			type: "Pod",
			phase: "Pending",
		};

		expect(getWorkflowNodeDisplayText(node, lookup)).toBe(
			"hand-detect-yolov26m",
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
