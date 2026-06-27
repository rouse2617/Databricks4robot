import type { Edge } from "@xyflow/react";

export const DEFAULT_INPUT_PORT = "input";
export const DEFAULT_OUTPUT_PORT = "output";
export const PIPELINE_EDGE_KIND_DEPENDENCY = "dependency";
/** Data edges (asset transfer) — solid line with animated flow */
export const DATA_EDGE_STYLE = {
	stroke: "#3b82f6",
	strokeWidth: 2,
	animated: false,
};
/** Dependency edges (control flow) — dashed line */
export const DEPENDENCY_EDGE_STYLE = {
	stroke: "#64748b",
	strokeWidth: 1.5,
	strokeDasharray: "6 4",
};

export type PipelineEdgeKind = "data" | typeof PIPELINE_EDGE_KIND_DEPENDENCY;
export type PipelineEdgeData = {
	kind?: PipelineEdgeKind;
};

export function isDependencyEdge(edge: Pick<Edge, "data">): boolean {
	return (
		(edge.data as PipelineEdgeData | undefined)?.kind ===
		PIPELINE_EDGE_KIND_DEPENDENCY
	);
}

export function dependencyEdgeData(data?: Edge["data"]) {
	return {
		...(data ?? {}),
		kind: PIPELINE_EDGE_KIND_DEPENDENCY,
	};
}

/** Format canvas edge endpoints for the transpiler (node-id.port-name). */
export function formatEdgeEndpoint(
	nodeId: string,
	handle: string | null | undefined,
	defaultPort: string,
): string {
	if (nodeId.includes(".")) {
		return nodeId;
	}
	const port = (handle && handle.length > 0 ? handle : defaultPort).replace(
		/^\./,
		"",
	);
	return `${nodeId}.${port}`;
}

export function splitRef(ref: string): { nodeId: string; port?: string } {
	const dot = ref.lastIndexOf(".");
	if (dot < 0) {
		return { nodeId: ref };
	}
	return { nodeId: ref.slice(0, dot), port: ref.slice(dot + 1) };
}
