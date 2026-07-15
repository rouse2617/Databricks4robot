import { describe, expect, it } from "vitest";
import type { WorkflowDagEdge, WorkflowNodeStatus } from "../api/workflowApi";
import {
	buildDagElements,
	countDisplayableWorkflowNodes,
} from "./WorkflowDagView";

describe("buildDagElements", () => {
	const nodes: WorkflowNodeStatus[] = [
		{
			id: "step-1",
			name: "wf.step-step-1",
			displayName: "step-step-1",
			type: "Pod",
			templateName: "step-step-1",
			phase: "Failed",
		},
		{
			id: "step-2",
			name: "wf.step-step-2",
			displayName: "step-step-2",
			templateName: "step-step-2",
			phase: "Omitted",
		},
	];

	it("prefers backend-provided normalized edges", () => {
		const workflowEdges: WorkflowDagEdge[] = [
			{
				id: "e-step-1-step-2",
				source: "step-1",
				target: "step-2",
				kind: "dag",
			},
		];

		const result = buildDagElements(nodes, workflowEdges, null, "");

		expect(result.edges).toHaveLength(1);
		expect(result.edges[0]).toMatchObject({
			source: "step-1",
			target: "step-2",
		});
	});

	it("ignores backend edges with hidden endpoints", () => {
		const workflowEdges: WorkflowDagEdge[] = [
			{
				id: "e-root-step-2",
				source: "root",
				target: "step-2",
				kind: "runtime",
			},
		];

		const result = buildDagElements(nodes, workflowEdges, null, "");

		expect(result.edges).toHaveLength(0);
	});

	it("counts only displayable workflow nodes", () => {
		expect(countDisplayableWorkflowNodes(nodes)).toBe(2);
		expect(
			countDisplayableWorkflowNodes([
				{
					id: "root",
					name: "wf",
					displayName: "wf",
					type: "DAG",
					phase: "Failed",
				},
			]),
		).toBe(0);
	});

	it("excludes the CYB-3058 onExit notify hook node by template name (CYB-3095)", () => {
		const withHook: WorkflowNodeStatus[] = [
			...nodes,
			{
				id: "exit-hook",
				name: "wf.onExit",
				displayName: "wf.onExit",
				type: "Pod",
				templateName: "databrew-exit-notify",
				phase: "Succeeded",
			},
		];
		// The DAG count and rendered nodes must ignore the infra hook so they
		// match the 节点明细 table (which the backend already filters).
		expect(countDisplayableWorkflowNodes(withHook)).toBe(2);
		const result = buildDagElements(withHook, undefined, null, "");
		expect(result.nodes.map((node) => node.id)).not.toContain("exit-hook");
		expect(result.nodes).toHaveLength(2);
	});

	it("excludes the onExit hook by .onExit name suffix when template name differs (CYB-3095)", () => {
		const withHook: WorkflowNodeStatus[] = [
			...nodes,
			{
				id: "exit-hook",
				name: "wf.onExit",
				displayName: "wf.onExit",
				type: "Pod",
				templateName: "some-other-template",
				phase: "Succeeded",
			},
		];
		expect(countDisplayableWorkflowNodes(withHook)).toBe(2);
	});
});
