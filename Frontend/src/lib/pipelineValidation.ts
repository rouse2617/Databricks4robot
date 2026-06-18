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
		if (args.some((arg) => argText(arg).length > 0)) return true;
		return (component.image ?? "").trim().length > 0;
	}
	if (command.length >= 3 && command[0] === "sh" && command[1] === "-c") {
		return command.slice(2).some((part) => part.length > 0);
	}
	if (command.length === 2 && command[0] === "sh" && command[1] === "-c") {
		return args.some((arg) => argText(arg).length > 0);
	}
	return command.some((part) => part.length > 0);
}

function hasRunnableScript(
	component: Pipeline["nodes"][number]["component"],
): boolean {
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

function canStaticallyVerifyOutputWrite(
	component: Pipeline["nodes"][number]["component"],
) {
	const nodeType = (component.type || "container").trim().toLowerCase();
	if (nodeType === "script") return true;
	const command = (component.command ?? []).map((part) => part.trim());
	return command.length >= 2 && command[0] === "sh" && command[1] === "-c";
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

export function validatePipelineForSave(
	pipeline: Pipeline,
): PipelineValidationResult {
	const errors: string[] = [];
	const warnings: string[] = [];

	if (!pipeline.name?.trim()) {
		errors.push("流水线名称不能为空。");
	}

	if (!Array.isArray(pipeline.nodes)) {
		errors.push("流水线节点结构无效。");
	} else {
		const nodeIds = new Set<string>();
		for (const node of pipeline.nodes) {
			const nodeId = node.id?.trim();
			if (!nodeId) {
				errors.push("流水线存在空节点 ID。");
				continue;
			}
			if (nodeIds.has(nodeId)) {
				errors.push(`节点 ID ${nodeId} 重复。`);
			}
			nodeIds.add(nodeId);
			if (!node.component) {
				errors.push(`节点 ${nodeId} 缺少组件定义。`);
			}
		}

		if (Array.isArray(pipeline.edges)) {
			for (const edge of pipeline.edges) {
				const source = splitRef(edge.source ?? "");
				const target = splitRef(edge.target ?? "");
				if (!source.nodeId || !target.nodeId) {
					errors.push("流水线存在无效连线。");
					continue;
				}
				if (!nodeIds.has(source.nodeId)) {
					errors.push(`连线引用了不存在的源节点 ${source.nodeId}。`);
				}
				if (!nodeIds.has(target.nodeId)) {
					errors.push(`连线引用了不存在的目标节点 ${target.nodeId}。`);
				}
			}
		}
	}

	if (!Array.isArray(pipeline.edges)) {
		errors.push("流水线连线结构无效。");
	}

	try {
		JSON.stringify(pipeline);
	} catch {
		errors.push("流水线 JSON 无法序列化。");
	}

	return { valid: errors.length === 0, errors, warnings };
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
			const message = `输出 ${edge.source} 被 ${edge.target} 消费，但组件脚本没有写入 /tmp/outputs/${source.port}。`;
			if (canStaticallyVerifyOutputWrite(sourceNode.component)) {
				errors.push(message);
			} else {
				warnings.push(
					`${message} 当前组件是镜像内执行逻辑，前端无法静态确认；请确认镜像运行时会生成该文件。`,
				);
			}
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
		} else if (
			nodeType === "container" &&
			!hasRunnableContainerCommand(component)
		) {
			errors.push(`${label} 缺少可执行的 command/args。`);
		}

		for (const output of node.outputs || []) {
			if (!componentWritesOutputPath(node.component, output.name)) {
				if (canStaticallyVerifyOutputWrite(node.component)) {
					warnings.push(
						`${node.id}.${output.name} 未写入 /tmp/outputs/${output.name}；未连接时不会影响运行，连接下游前需要补输出文件。`,
					);
				} else {
					warnings.push(
						`${node.id}.${output.name} 无法静态确认是否写入 /tmp/outputs/${output.name}；如果连接下游，请确保镜像运行时会产出该文件。`,
					);
				}
			}
		}
	}

	return { valid: errors.length === 0, errors, warnings };
}
