import { describe, expect, it } from "vitest";
import type { WorkflowDagEdge, WorkflowNodeStatus } from "../api/workflowApi";
import { buildDagElements } from "./WorkflowDagView";

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

		const result = buildDagElements(nodes, workflowEdges, null);

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

		const result = buildDagElements(nodes, workflowEdges, null);

		expect(result.edges).toHaveLength(0);
	});
});
