import type {
	Argument,
	Pipeline,
	PipelineEdgeDef,
	PipelineNodeData,
	PipelineNodeDef,
	Port,
} from "../components/pipeline/types";

/** Coerce API/canvas args (string[] or Argument[]) into transpiler Argument objects. */
export function normalizeComponentArgs(
	args: unknown[] | undefined,
): Argument[] {
	if (!args || args.length === 0) return [];
	return args.map((item, index) => {
		if (typeof item === "string") {
			const value = item.trim();
			return { name: value || `arg${index + 1}`, value };
		}
		if (item && typeof item === "object") {
			const record = item as { name?: string; value?: string; from?: string };
			const value = record.value ?? record.name ?? "";
			const name = record.name?.trim() || value || `arg${index + 1}`;
			return record.from ? { name, value, from: record.from } : { name, value };
		}
		return { name: `arg${index + 1}`, value: String(item) };
	});
}

function envToRecords(
	resources: unknown,
): Array<{ name: string; value: string }> {
	if (!resources || typeof resources !== "object") return [];
	const raw = (resources as { env?: unknown }).env;
	if (!raw) return [];
	if (Array.isArray(raw)) {
		return raw
			.map((item) => {
				if (!item || typeof item !== "object") return null;
				const record = item as { name?: string; value?: string };
				if (!record.name) return null;
				return {
					name: record.name,
					value: record.value || "",
				};
			})
			.filter((item): item is { name: string; value: string } => Boolean(item));
	}
	if (typeof raw === "object") {
		return Object.entries(raw as Record<string, string>).map(
			([name, value]) => ({
				name,
				value: String(value || ""),
			}),
		);
	}
	return [];
}

function envToMap(
	env: Array<{ name: string; value?: string }>,
): Record<string, string> {
	const out: Record<string, string> = {};
	for (const item of env) {
		if (!item.name || item.value === undefined) continue;
		out[item.name] = item.value;
	}
	return out;
}

const DEFAULT_INPUT_PORT = "input";
const DEFAULT_OUTPUT_PORT = "output";

type PipelineCanvasNode = {
	id: string;
	type?: string;
	position: { x: number; y: number };
	data: PipelineNodeData;
};

type PipelineCanvasEdge = {
	id: string;
	source: string;
	target: string;
	sourceHandle?: string | null;
	targetHandle?: string | null;
};

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
	nodes: PipelineCanvasNode[],
	edges: PipelineCanvasEdge[],
	meta: { name: string; version?: string },
): Pipeline {
	return {
		name: meta.name,
		version: meta.version ?? "1",
		nodes: nodes.map((n) => nodeToDef(n)),
		edges: edges.map((e) => edgeToDef(e)),
	};
}

function nodeToDef(n: PipelineCanvasNode): PipelineNodeDef {
	const d = n.data;
	return {
		id: n.id,
		component: {
			name: d.label || "",
			image: d.image || "",
			mode: d.mode || d.type || "container",
			command: d.command || [],
			args: normalizeComponentArgs(d.args as unknown[]),
			...(d.env && d.env.length > 0 ? { env: envToMap(d.env) } : {}),
			...(d.script ? { source: d.script } : {}),
			resources:
				d.cpu || d.memory || d.disk
					? {
							cpu: d.cpu,
							memory: d.memory,
							disk: d.disk,
						}
					: undefined,
		},
		inputs: defaultInputs(),
		outputs: defaultOutputs(),
	};
}

function edgeToDef(e: PipelineCanvasEdge): PipelineEdgeDef {
	return {
		source: formatEdgeEndpoint(e.source, e.sourceHandle, DEFAULT_OUTPUT_PORT),
		target: formatEdgeEndpoint(e.target, e.targetHandle, DEFAULT_INPUT_PORT),
	};
}

/** Restore canvas state from a saved transpiler pipeline JSON. */
export function fromTranspilerPipeline(pipeline: Pipeline): {
	nodes: PipelineCanvasNode[];
	edges: PipelineCanvasEdge[];
} {
	const nodes: PipelineCanvasNode[] = pipeline.nodes.map((pn, i) => ({
		id: pn.id,
		type: "pipelineStep" as const,
		position: { x: 120 + i * 80, y: 100 + i * 60 },
		data: {
			label: pn.component.name,
			image: pn.component.image,
			mode: (pn.component as { mode?: string }).mode || "container",
			script: (pn.component as { source?: string }).source,
			command: pn.component.command || [],
			args: normalizeComponentArgs(pn.component.args as unknown[]),
			env: envToRecords(pn.component.env || pn.component.resources),
			cpu: pn.component.resources?.cpu || "",
			memory: pn.component.resources?.memory || "",
			disk: pn.component.resources?.disk || "",
		},
	}));

	const edges: PipelineCanvasEdge[] = pipeline.edges.map((pe, i) => {
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
