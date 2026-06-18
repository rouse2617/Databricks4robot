import type {
	Pipeline,
	PipelineNodeRuntimeConfig,
} from "../../components/pipeline/types";
import type {
	PipelineCanvasEdge,
	PipelineCanvasNode,
} from "../../features/pipeline-designer/model/canvas-model";
import { normalizeComponentArgs } from "./args-normalizer";
import { splitRef } from "./edge-format";
import {
	defaultInputPorts,
	defaultOutputPorts,
	normalizePorts,
} from "./port-normalizer";

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

function readComponentRef(component: Record<string, unknown>) {
	return {
		componentId:
			typeof component.componentId === "string"
				? component.componentId
				: undefined,
		releaseId:
			typeof component.releaseId === "string" ? component.releaseId : undefined,
		componentVersionLabel:
			typeof component.componentVersionLabel === "string"
				? component.componentVersionLabel
				: undefined,
	};
}

function normalizeRuntimeConfig(
	config: PipelineNodeRuntimeConfig | undefined,
): PipelineNodeRuntimeConfig | undefined {
	if (!config?.configId || !config.version) return undefined;
	const mountPath = config.mountPath?.trim();
	const targetFilename = config.targetFilename?.trim();
	if (!mountPath || !targetFilename) return undefined;
	return {
		mode: "saved",
		configId: config.configId,
		version: config.version,
		...(config.fileName?.trim() ? { fileName: config.fileName.trim() } : {}),
		mountPath,
		targetFilename,
		...(config.displayName?.trim()
			? { displayName: config.displayName.trim() }
			: {}),
	};
}

/** Restore canvas state from a saved pipeline JSON. */
export function designDSLToCanvas(pipeline: Pipeline): {
	nodes: PipelineCanvasNode[];
	edges: PipelineCanvasEdge[];
} {
	const nodes: PipelineCanvasNode[] = pipeline.nodes.map((pn, i) => {
		const component = pn.component as unknown as Record<string, unknown>;
		const refs = readComponentRef(component);
		return {
			id: pn.id,
			type: "pipelineStep" as const,
			position: { x: 120 + i * 80, y: 100 + i * 60 },
			data: {
				label: pn.component.name,
				image: pn.component.image,
				type: (pn.component as { type?: string }).type || "container",
				source: (pn.component as { source?: string }).source || "custom",
				command: pn.component.command || [],
				args: normalizeComponentArgs(pn.component.args as unknown[]),
				env: envToRecords(pn.component.resources),
				cpu: pn.component.resources?.cpu || "",
				memory: pn.component.resources?.memory || "",
				disk: pn.component.resources?.disk || "",
				gpu: pn.component.resources?.gpu || "",
				computeTier: pn.component.resources?.computeTier || "",
				inputPorts: normalizePorts(pn.inputs, defaultInputPorts),
				outputPorts: normalizePorts(pn.outputs, defaultOutputPorts),
				runtimeConfig: normalizeRuntimeConfig(pn.runtimeConfig),
				...refs,
			},
		};
	});

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

/** @deprecated use designDSLToCanvas */
export const fromTranspilerPipeline = designDSLToCanvas;
