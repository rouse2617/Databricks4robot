import { describe, expect, it } from "vitest";
import type { WorkflowNodeStatus } from "../api/workflowApi";
import { buildWorkflowFlowEdges } from "./workflowDag";

describe("buildWorkflowFlowEdges", () => {
	it("uses Argo children field (hash-suffixed pod ids)", () => {
		const nodes: WorkflowNodeStatus[] = [
			{
				id: "wf-root",
				name: "wf-root",
				displayName: "wf-root",
				phase: "Succeeded",
				children: ["wf-root-3221465865"],
			},
			{
				id: "wf-root-3221465865",
				name: "wf-root.step-node-1-test",
				displayName: "step-node-1-test",
				phase: "Succeeded",
			},
		];
		const edges = buildWorkflowFlowEdges(nodes);
		expect(edges).toHaveLength(1);
		expect(edges[0]).toMatchObject({
			source: "wf-root",
			target: "wf-root-3221465865",
		});
	});

	it("falls back to dotted node names when children missing", () => {
		const nodes: WorkflowNodeStatus[] = [
			{
				id: "a",
				name: "wf",
				displayName: "wf",
				phase: "Succeeded",
			},
			{
				id: "b",
				name: "wf.step-1",
				displayName: "step-1",
				phase: "Succeeded",
			},
		];
		const edges = buildWorkflowFlowEdges(nodes);
		expect(edges).toHaveLength(1);
		expect(edges[0]?.source).toBe("a");
		expect(edges[0]?.target).toBe("b");
	});
});
