import type {
	Pipeline,
	PipelineEdgeDef,
	PipelineNodeDef,
	PipelineNodeRuntimeConfig,
	PipelineNodeRuntimeSecretMount,
	PipelineNodeRuntimeStorageMount,
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
	isDependencyEdge,
} from "./edge-format";
import {
	defaultInputPorts,
	defaultOutputPorts,
	normalizePorts,
} from "./port-normalizer";

function envToList(
	env: Array<{ name: string; value?: string }>,
): Array<{ name: string; value: string }> {
	const out: Array<{ name: string; value: string }> = [];
	for (const item of env) {
		if (!item?.name) continue;
		out.push({ name: item.name, value: item.value ?? "" });
	}
	return out;
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

function nodeToDef(n: PipelineCanvasNode): PipelineNodeDef {
	const d = n.data;
	const runtimeConfig = normalizeRuntimeConfig(d.runtimeConfig);
	const runtimeSecrets = normalizeRuntimeSecrets(d.runtimeSecrets);
	const storageMounts = normalizeStorageMounts(d.storageMounts);
	const env = envToList(d.env ?? []);
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
			...(env.length > 0 ? { env } : {}),
			...(d.componentId ? { componentId: d.componentId } : {}),
			...(d.releaseId ? { releaseId: d.releaseId } : {}),
			...(d.componentVersionLabel
				? { componentVersionLabel: d.componentVersionLabel }
				: {}),
			resources:
				d.cpu || d.memory || d.disk || d.gpu || d.computeTier
					? {
							cpu: d.cpu,
							memory: d.memory,
							disk: d.disk,
							gpu: d.gpu,
							computeTier: d.computeTier,
							type: d.type || "container",
							source: d.source || "custom",
						}
					: undefined,
		},
		inputs: normalizePorts(d.inputPorts, defaultInputPorts),
		outputs: normalizePorts(d.outputPorts, defaultOutputPorts),
		...(runtimeConfig ? { runtimeConfig } : {}),
		...(runtimeSecrets.length > 0 ? { runtimeSecrets } : {}),
		...(storageMounts.length > 0 ? { storageMounts } : {}),
	};
}

function edgeToDef(e: PipelineCanvasEdge): PipelineEdgeDef {
	if (isDependencyEdge(e)) {
		return {
			source: e.source,
			target: e.target,
		};
	}
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
