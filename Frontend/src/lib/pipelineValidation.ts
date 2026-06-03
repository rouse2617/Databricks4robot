import type { Pipeline } from "../components/pipeline/types";

export type PipelineValidationResult = {
	valid: boolean;
	errors: string[];
	warnings: string[];
};

function splitRef(ref: string): { nodeId: string; port: string } {
	const dot = ref.lastIndexOf(".");
	if (dot < 0) return { nodeId: ref, port: "" };
	return { nodeId: ref.slice(0, dot), port: ref.slice(dot + 1) };
}

function safeParamName(value: string) {
	return value.replace(/\./g, "-");
}

function componentWritesOutputPath(
	component: Pipeline["nodes"][number]["component"],
	outputName: string,
) {
	const path = `/tmp/outputs/${outputName}`;
	if (component.source?.includes(path)) return true;
	if (component.command?.some((part) => part.includes(path))) return true;
	if (component.args?.some((arg) => arg.value?.includes(path))) return true;
	return false;
}

function nodeDeclaresOutput(
	node: Pipeline["nodes"][number] | undefined,
	portName: string,
) {
	if (!node) return false;
	const safe = safeParamName(portName);
	return (node.outputs || []).some(
		(port) => port.name === portName || safeParamName(port.name) === safe,
	);
}

export function validatePipelineForRun(
	pipeline: Pipeline,
): PipelineValidationResult {
	const errors: string[] = [];
	const warnings: string[] = [];
	const nodesById = new Map(pipeline.nodes.map((node) => [node.id, node]));
	const targetBindings = new Map<string, string>();

	for (const edge of pipeline.edges || []) {
		const source = splitRef(edge.source);
		const target = splitRef(edge.target);
		if (target.nodeId && target.port) {
			const key = `${target.nodeId}.${safeParamName(target.port)}`;
			const prior = targetBindings.get(key);
			if (prior) {
				errors.push(
					`输入端口 ${key} 已连接 ${prior}，不能再连接 ${edge.source}。请为汇聚节点配置不同输入端口。`,
				);
			} else {
				targetBindings.set(key, edge.source);
			}
		}

		if (!source.nodeId || !source.port) continue;
		const sourceNode = nodesById.get(source.nodeId);
		if (!nodeDeclaresOutput(sourceNode, source.port)) continue;
		if (
			sourceNode &&
			!componentWritesOutputPath(sourceNode.component, source.port)
		) {
			errors.push(
				`输出 ${edge.source} 被 ${edge.target} 消费，但组件脚本没有写入 /tmp/outputs/${source.port}。`,
			);
		}
	}

	for (const node of pipeline.nodes || []) {
		for (const output of node.outputs || []) {
			if (!componentWritesOutputPath(node.component, output.name)) {
				warnings.push(
					`${node.id}.${output.name} 未写入 /tmp/outputs/${output.name}；未连接时不会影响运行，连接下游前需要补输出文件。`,
				);
			}
		}
	}

	return { valid: errors.length === 0, errors, warnings };
}
