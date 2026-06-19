import dagre from "dagre";
import type {
	Pipeline,
	PipelineNodeRuntimeConfig,
	PipelineNodeRuntimeSecretMount,
	PipelineNodeRuntimeStorageMount,
} from "../../components/pipeline/types";
import type {
	PipelineCanvasEdge,
	PipelineCanvasNode,
} from "../../features/pipeline-designer/model/canvas-model";
import { normalizeComponentArgs } from "./args-normalizer";
import {
	DEPENDENCY_EDGE_STYLE,
	dependencyEdgeData,
	splitRef,
} from "./edge-format";
import {
	defaultInputPorts,
	defaultOutputPorts,
	normalizePorts,
} from "./port-normalizer";

const CANVAS_NODE_LAYOUT_WIDTH = 220;
const CANVAS_NODE_LAYOUT_HEIGHT = 150;
const CANVAS_NODE_RANK_GAP = 112;
const CANVAS_NODE_SIBLING_GAP = 72;
const CANVAS_LAYOUT_MARGIN_X = 120;
const CANVAS_LAYOUT_MARGIN_Y = 100;

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

function normalizeRuntimeSecrets(
	items: PipelineNodeRuntimeSecretMount[] | undefined,
): PipelineNodeRuntimeSecretMount[] {
	return (items || [])
		.map((item) => {
			const resourceId = item.resourceId?.trim();
			if (!resourceId) return null;
			return {
				resourceId,
				...(item.mountPath?.trim() ? { mountPath: item.mountPath.trim() } : {}),
				...(item.displayName?.trim()
					? { displayName: item.displayName.trim() }
					: {}),
			};
		})
		.filter((item): item is PipelineNodeRuntimeSecretMount => Boolean(item));
}

function normalizeStorageMounts(
	items: PipelineNodeRuntimeStorageMount[] | undefined,
): PipelineNodeRuntimeStorageMount[] {
	return (items || [])
		.map((item) => {
			const resourceId = item.resourceId?.trim();
			if (!resourceId) return null;
			return {
				resourceId,
				...(item.mountPath?.trim() ? { mountPath: item.mountPath.trim() } : {}),
				...(typeof item.readOnly === "boolean"
					? { readOnly: item.readOnly }
					: {}),
				...(item.displayName?.trim()
					? { displayName: item.displayName.trim() }
					: {}),
			};
		})
		.filter((item): item is PipelineNodeRuntimeStorageMount => Boolean(item));
}

function fallbackPosition(index: number) {
	return {
		x:
			CANVAS_LAYOUT_MARGIN_X +
			index * (CANVAS_NODE_LAYOUT_WIDTH + CANVAS_NODE_RANK_GAP),
		y: CANVAS_LAYOUT_MARGIN_Y,
	};
}

function buildNodePositions(
	pipeline: Pipeline,
): Map<string, { x: number; y: number }> {
	const graph = new dagre.graphlib.Graph();
	graph.setGraph({
		rankdir: "LR",
		ranksep: CANVAS_NODE_RANK_GAP,
		nodesep: CANVAS_NODE_SIBLING_GAP,
		marginx: CANVAS_LAYOUT_MARGIN_X,
		marginy: CANVAS_LAYOUT_MARGIN_Y,
	});
	graph.setDefaultEdgeLabel(() => ({}));

	const nodeIds = new Set<string>();
	for (const node of pipeline.nodes) {
		nodeIds.add(node.id);
		graph.setNode(node.id, {
			width: CANVAS_NODE_LAYOUT_WIDTH,
			height: CANVAS_NODE_LAYOUT_HEIGHT,
		});
	}

	for (const edge of pipeline.edges) {
		const source = splitRef(edge.source).nodeId;
		const target = splitRef(edge.target).nodeId;
		if (nodeIds.has(source) && nodeIds.has(target)) {
			graph.setEdge(source, target);
		}
	}

	dagre.layout(graph);

	const positions = new Map<string, { x: number; y: number }>();
	pipeline.nodes.forEach((node, index) => {
		const layoutNode = graph.node(node.id);
		if (
			layoutNode &&
			typeof layoutNode.x === "number" &&
			typeof layoutNode.y === "number"
		) {
			positions.set(node.id, {
				x: Math.round(layoutNode.x - CANVAS_NODE_LAYOUT_WIDTH / 2),
				y: Math.round(layoutNode.y - CANVAS_NODE_LAYOUT_HEIGHT / 2),
			});
			return;
		}
		positions.set(node.id, fallbackPosition(index));
	});
	return positions;
}

/** Restore canvas state from a saved pipeline JSON. */
export function designDSLToCanvas(pipeline: Pipeline): {
	nodes: PipelineCanvasNode[];
	edges: PipelineCanvasEdge[];
} {
	const positions = buildNodePositions(pipeline);
	const nodes: PipelineCanvasNode[] = pipeline.nodes.map((pn, i) => {
		const component = pn.component as unknown as Record<string, unknown>;
		const refs = readComponentRef(component);
		return {
			id: pn.id,
			type: "pipelineStep" as const,
			position: positions.get(pn.id) ?? fallbackPosition(i),
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
				runtimeSecrets: normalizeRuntimeSecrets(pn.runtimeSecrets),
				storageMounts: normalizeStorageMounts(pn.storageMounts),
				...refs,
			},
		};
	});

	const edges: PipelineCanvasEdge[] = pipeline.edges.map((pe, i) => {
		const source = splitRef(pe.source);
		const target = splitRef(pe.target);
		const isDependency = !source.port && !target.port;
		return {
			id: `e-${i}`,
			source: source.nodeId,
			target: target.nodeId,
			...(source.port ? { sourceHandle: source.port } : {}),
			...(target.port ? { targetHandle: target.port } : {}),
			...(isDependency
				? {
						animated: true,
						style: DEPENDENCY_EDGE_STYLE,
						data: dependencyEdgeData(),
					}
				: {}),
		};
	});

	return { nodes, edges };
}

/** @deprecated use designDSLToCanvas */
export const fromTranspilerPipeline = designDSLToCanvas;
