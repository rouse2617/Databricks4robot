import type { Pipeline } from "../components/pipeline/types";
import { normalizeComponentArgs } from "./pipelineContract";

export type PipelineValidationResult = {
	valid: boolean;
	errors: string[];
	warnings: string[];
};

function argText(value: { name?: string; value?: string } | undefined): string {
	return (value?.value ?? value?.name ?? "").trim();
}

function hasRunnableContainerCommand(
	component: Pipeline["nodes"][number]["component"],
): boolean {
	const command = (component.command ?? []).map((part) => part.trim());
	const args = normalizeComponentArgs(component.args ?? []);
	if (command.length === 0) {
		return args.some((arg) => argText(arg).length > 0);
	}
	if (command.length >= 3 && command[0] === "sh" && command[1] === "-c") {
		return command.slice(2).some((part) => part.length > 0);
	}
	if (command.length === 2 && command[0] === "sh" && command[1] === "-c") {
		return args.some((arg) => argText(arg).length > 0);
	}
	return command.some((part) => part.length > 0);
}

function hasRunnableScript(component: Pipeline["nodes"][number]["component"]): boolean {
	const source = component.source?.trim() ?? "";
	if (source.length > 0) {
		return true;
	}
	return normalizeComponentArgs(component.args ?? []).some(
		(arg) => argText(arg).length > 0,
	);
}

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
		const component = node.component;
		const label = component.name?.trim() || node.id;
		const nodeType = (component.type || "container").trim().toLowerCase();
		if (nodeType === "resource" || nodeType === "suspend") {
			continue;
		}
		if (nodeType === "script") {
			if (!hasRunnableScript(component)) {
				errors.push(`${label} 缺少可执行的脚本内容。`);
			}
		} else if (nodeType === "container" && !hasRunnableContainerCommand(component)) {
			errors.push(`${label} 缺少可执行的 command/args。`);
		}

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
