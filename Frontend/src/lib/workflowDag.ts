import type { Edge as RFEdge } from "@xyflow/react";
import type { WorkflowNodeStatus } from "../api/workflowApi";

/** Build React Flow edges from Argo workflow node status (uses `children`, then dotted names). */
export function buildWorkflowFlowEdges(
	nodes: WorkflowNodeStatus[],
): RFEdge[] {
	const byId = new Map(nodes.map((n) => [n.id, n]));
	const edges: RFEdge[] = [];
	const seen = new Set<string>();

	const addEdge = (source: string, target: string) => {
		if (!byId.has(source) || !byId.has(target)) return;
		const id = `e-${source}-${target}`;
		if (seen.has(id)) return;
		seen.add(id);
		edges.push({
			id,
			source,
			target,
			style: { stroke: "#94a3b8" },
		});
	};

	for (const n of nodes) {
		for (const childId of n.children ?? []) {
			addEdge(n.id, childId);
		}
	}

	for (const n of nodes) {
		const parts = n.name.split(".");
		if (parts.length > 1) {
			const parentName = parts.slice(0, -1).join(".");
			const parent = nodes.find((x) => x.name === parentName);
			if (parent) addEdge(parent.id, n.id);
		}
	}

	return edges;
}
