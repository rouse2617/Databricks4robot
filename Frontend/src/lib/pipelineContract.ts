import type { Edge, Node } from "@xyflow/react";
import type {
	Pipeline,
	PipelineEdgeDef,
	PipelineNodeData,
	PipelineNodeDef,
	Port,
} from "../components/pipeline/types";

const DEFAULT_INPUT_PORT = "input";
const DEFAULT_OUTPUT_PORT = "output";

const defaultInputs = (): Port[] => [
	{ name: DEFAULT_INPUT_PORT, type: "string" },
];
const defaultOutputs = (): Port[] => [
	{ name: DEFAULT_OUTPUT_PORT, type: "string" },
];

/** Format React Flow edge endpoints for the transpiler (node-id.port-name). */
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

export function toTranspilerPipeline(
	nodes: Node<PipelineNodeData>[],
	edges: Edge[],
	meta: { name: string; version?: string },
): Pipeline {
	return {
		name: meta.name,
		version: meta.version ?? "1",
		nodes: nodes.map((n) => nodeToDef(n)),
		edges: edges.map((e) => edgeToDef(e)),
	};
}

function nodeToDef(n: Node<PipelineNodeData>): PipelineNodeDef {
	const d = n.data;
	return {
		id: n.id,
		component: {
			name: d.label || "",
			image: d.image || "",
			command: d.command || [],
			args: d.args || [],
			resources:
				d.cpu || d.memory || d.disk
					? { cpu: d.cpu, memory: d.memory, disk: d.disk }
					: undefined,
		},
		inputs: defaultInputs(),
		outputs: defaultOutputs(),
	};
}

function edgeToDef(e: Edge): PipelineEdgeDef {
	return {
		source: formatEdgeEndpoint(e.source, e.sourceHandle, DEFAULT_OUTPUT_PORT),
		target: formatEdgeEndpoint(e.target, e.targetHandle, DEFAULT_INPUT_PORT),
	};
}

/** Restore canvas state from a saved transpiler pipeline JSON. */
export function fromTranspilerPipeline(pipeline: Pipeline): {
	nodes: Node<PipelineNodeData>[];
	edges: Edge[];
} {
	const nodes: Node<PipelineNodeData>[] = pipeline.nodes.map((pn, i) => ({
		id: pn.id,
		type: "pipelineStep" as const,
		position: { x: 120 + i * 80, y: 100 + i * 60 },
		data: {
			label: pn.component.name,
			image: pn.component.image,
			command: pn.component.command || [],
			args: pn.component.args || [],
			cpu: pn.component.resources?.cpu || "",
			memory: pn.component.resources?.memory || "",
			disk: pn.component.resources?.disk || "",
		},
	}));

	const edges: Edge[] = pipeline.edges.map((pe, i) => {
		const source = splitRef(pe.source);
		const target = splitRef(pe.target);
		return {
			id: `e-${i}`,
			source: source.nodeId,
			target: target.nodeId,
			...(source.port ? { sourceHandle: source.port } : {}),
			...(target.port ? { targetHandle: target.port } : {}),
		};
	});

	return { nodes, edges };
}

function splitRef(ref: string): { nodeId: string; port?: string } {
	const dot = ref.lastIndexOf(".");
	if (dot < 0) {
		return { nodeId: ref };
	}
	return { nodeId: ref.slice(0, dot), port: ref.slice(dot + 1) };
}
