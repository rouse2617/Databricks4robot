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

	it("marks generated graph elements as read-only for keyboard and accessibility semantics", () => {
		const workflowEdges: WorkflowDagEdge[] = [
			{
				id: "e-step-1-step-2",
				source: "step-1",
				target: "step-2",
				kind: "dag",
			},
		];

		const result = buildDagElements(nodes, workflowEdges, null, "");

		expect(result.nodes).toHaveLength(2);
		expect(result.nodes.every((node) => node.draggable === false)).toBe(true);
		expect(result.nodes.every((node) => node.connectable === false)).toBe(true);
		expect(result.nodes.every((node) => node.focusable === false)).toBe(true);
		expect(result.nodes.every((node) => node.deletable === false)).toBe(true);
		expect(result.edges).toHaveLength(1);
		expect(result.edges.every((edge) => edge.focusable === false)).toBe(true);
		expect(result.edges.every((edge) => edge.selectable === false)).toBe(true);
		expect(result.edges.every((edge) => edge.deletable === false)).toBe(true);
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

	it("centers fan-in join nodes over their direct predecessors", () => {
		const complexNodes = [
			"qa-seed-root",
			"qa-left-a",
			"qa-left-b",
			"qa-right-a",
			"qa-right-b",
			"qa-mid-a",
			"qa-join-abc",
			"qa-final-sink",
		].map((id): WorkflowNodeStatus => {
			return {
				id,
				name: `wf.${id}`,
				displayName: id,
				type: "Pod",
				templateName: id,
				phase: "Succeeded",
			};
		});
		const complexEdges: WorkflowDagEdge[] = [
			{
				id: "seed-left-a",
				source: "qa-seed-root",
				target: "qa-left-a",
				kind: "dag",
			},
			{
				id: "left-a-left-b",
				source: "qa-left-a",
				target: "qa-left-b",
				kind: "dag",
			},
			{
				id: "seed-right-a",
				source: "qa-seed-root",
				target: "qa-right-a",
				kind: "dag",
			},
			{
				id: "right-a-right-b",
				source: "qa-right-a",
				target: "qa-right-b",
				kind: "dag",
			},
			{
				id: "seed-mid-a",
				source: "qa-seed-root",
				target: "qa-mid-a",
				kind: "dag",
			},
			{
				id: "left-b-join",
				source: "qa-left-b",
				target: "qa-join-abc",
				kind: "dag",
			},
			{
				id: "right-b-join",
				source: "qa-right-b",
				target: "qa-join-abc",
				kind: "dag",
			},
			{
				id: "mid-a-join",
				source: "qa-mid-a",
				target: "qa-join-abc",
				kind: "dag",
			},
			{
				id: "join-final",
				source: "qa-join-abc",
				target: "qa-final-sink",
				kind: "dag",
			},
		];

		const result = buildDagElements(complexNodes, complexEdges, null, "");
		const nodeById = new Map(result.nodes.map((node) => [node.id, node]));
		const centerY = (id: string) => {
			const node = nodeById.get(id);
			if (!node) throw new Error(`missing node ${id}`);
			return node.position.y + 56;
		};
		const expectedJoinCenter =
			(centerY("qa-left-b") + centerY("qa-right-b") + centerY("qa-mid-a")) / 3;

		expect(centerY("qa-join-abc")).toBeCloseTo(expectedJoinCenter, 5);
		expect(
			result.edges
				.filter((edge) => edge.target === "qa-join-abc")
				.every((edge) => edge.className === "workflow-dag-view__edge--fanin"),
		).toBe(true);
		expect(
			result.edges.find((edge) => edge.target === "qa-final-sink")?.className,
		).toBeUndefined();
	});
});
