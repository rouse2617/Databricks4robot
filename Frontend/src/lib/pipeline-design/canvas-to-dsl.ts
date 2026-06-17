import type {
	Pipeline,
	PipelineEdgeDef,
	PipelineNodeDef,
} from "../../components/pipeline/types";
import type {
	PipelineCanvasEdge,
	PipelineCanvasNode,
} from "../../features/pipeline-designer/model/canvas-model";
import {
	normalizeComponentArgs,
	normalizeShellCommandArgs,
} from "./args-normalizer";
import {
	DEFAULT_INPUT_PORT,
	DEFAULT_OUTPUT_PORT,
	formatEdgeEndpoint,
} from "./edge-format";
import {
	defaultInputPorts,
	defaultOutputPorts,
	normalizePorts,
} from "./port-normalizer";

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

function nodeToDef(n: PipelineCanvasNode): PipelineNodeDef {
	const d = n.data;
	return {
		id: n.id,
		component: {
			name: d.label || "",
			image: d.image || "",
			type: d.type || "container",
			source: d.source || "custom",
			command: d.command || [],
			args: normalizeShellCommandArgs(
				d.command || [],
				normalizeComponentArgs(d.args as unknown[]),
			),
			...(d.componentId ? { componentId: d.componentId } : {}),
			...(d.releaseId ? { releaseId: d.releaseId } : {}),
			...(d.componentVersionLabel
				? { componentVersionLabel: d.componentVersionLabel }
				: {}),
			resources:
				d.cpu ||
				d.memory ||
				d.disk ||
				d.gpu ||
				d.computeTier ||
				(d.env && d.env.length > 0)
					? {
							cpu: d.cpu,
							memory: d.memory,
							disk: d.disk,
							gpu: d.gpu,
							computeTier: d.computeTier,
							type: d.type || "container",
							source: d.source || "custom",
							env: d.env ? envToMap(d.env) : undefined,
						}
					: undefined,
		},
		inputs: normalizePorts(d.inputPorts, defaultInputPorts),
		outputs: normalizePorts(d.outputPorts, defaultOutputPorts),
	};
}

function edgeToDef(e: PipelineCanvasEdge): PipelineEdgeDef {
	return {
		source: formatEdgeEndpoint(e.source, e.sourceHandle, DEFAULT_OUTPUT_PORT),
		target: formatEdgeEndpoint(e.target, e.targetHandle, DEFAULT_INPUT_PORT),
	};
}

export function canvasToDesignDSL(
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

/** @deprecated use canvasToDesignDSL */
export const toTranspilerPipeline = canvasToDesignDSL;
