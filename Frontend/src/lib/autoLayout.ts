import type { Edge, Node } from "@xyflow/react";
import dagre from "dagre";

const NODE_WIDTH = 180;
const NODE_HEIGHT = 100;
const RANK_SEP = 120;
const NODE_SEP = 60;

/**
 * Compute auto-layout positions using Dagre.
 * Returns a map of nodeId → { x, y }.
 * Falls back to original positions on failure.
 */
export function computeDagreLayout(
	nodes: Node[],
	edges: Edge[],
	direction: "LR" | "TB" = "LR",
): Record<string, { x: number; y: number }> {
	try {
		const g = new dagre.graphlib.Graph();
		g.setDefaultEdgeLabel(() => ({}));
		g.setGraph({ rankdir: direction, ranksep: RANK_SEP, nodesep: NODE_SEP });

		for (const node of nodes) {
			g.setNode(node.id, { width: NODE_WIDTH, height: NODE_HEIGHT });
		}
		for (const edge of edges) {
			g.setEdge(edge.source, edge.target);
		}

		dagre.layout(g);

		const layout: Record<string, { x: number; y: number }> = {};
		for (const node of nodes) {
			const dagreNode = g.node(node.id);
			if (dagreNode) {
				layout[node.id] = {
					x: dagreNode.x - NODE_WIDTH / 2,
					y: dagreNode.y - NODE_HEIGHT / 2,
				};
			}
		}
		return layout;
	} catch {
		// Fallback: keep original positions
		const layout: Record<string, { x: number; y: number }> = {};
		for (const node of nodes) {
			layout[node.id] = node.position;
		}
		return layout;
	}
}
