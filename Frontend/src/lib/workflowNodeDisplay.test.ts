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

	it("maps readable step-<component> template names (CYB-3076)", () => {
		const lookup = buildPipelineNodeLabelLookup({
			name: "my-pipeline",
			nodes: [
				{
					id: "node-8242f006-694e-4c17-945c-fdc485c50ff1",
					component: {
						name: "head-track-pycuvslam",
						image: "example.com/head-track:latest",
					},
				},
			],
		});

		const readable: WorkflowNodeStatus = {
			id: "wf.step-head-track-pycuvslam",
			name: "run-id.step-head-track-pycuvslam",
			displayName: "step-head-track-pycuvslam",
			templateName: "step-head-track-pycuvslam",
			type: "Pod",
			phase: "Running",
		};
		expect(getWorkflowNodeDisplayText(readable, lookup)).toBe(
			"head-track-pycuvslam",
		);

		// Duplicate-component disambiguation form (step-<slug>-<uuid8>).
		const withSuffix: WorkflowNodeStatus = {
			...readable,
			displayName: "step-head-track-pycuvslam-8242f006",
			templateName: "step-head-track-pycuvslam-8242f006",
		};
		expect(getWorkflowNodeDisplayText(withSuffix, lookup)).toBe(
			"head-track-pycuvslam",
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
