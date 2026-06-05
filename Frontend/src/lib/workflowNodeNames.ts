import type { WorkflowDetail, WorkflowNodeStatus } from "../api/workflowApi";
import type { Pipeline } from "../components/pipeline/types";

function normalizeName(value?: string): string | undefined {
	const trimmed = value?.trim();
	return trimmed || undefined;
}

function basename(value?: string): string | undefined {
	const normalized = normalizeName(value);
	if (!normalized) return undefined;
	const dot = normalized.lastIndexOf(".");
	return dot >= 0 ? normalized.slice(dot + 1) : normalized;
}

function addKey(
	map: Map<string, string>,
	key: string | undefined,
	label: string,
) {
	const normalized = normalizeName(key);
	if (!normalized || map.has(normalized)) return;
	map.set(normalized, label);
}

function buildPipelineNodeNameMap(pipeline?: Pipeline): Map<string, string> {
	const map = new Map<string, string>();
	for (const node of pipeline?.nodes ?? []) {
		const nodeId = normalizeName(node.id);
		const componentName = normalizeName(node.component?.name);
		if (!nodeId || !componentName) continue;

		addKey(map, nodeId, componentName);
		addKey(map, `step-${nodeId}`, componentName);
		addKey(map, basename(nodeId), componentName);
	}
	return map;
}

function workflowNodeNameCandidates(node: WorkflowNodeStatus): string[] {
	const candidates = [
		node.id,
		node.name,
		node.displayName,
		node.templateName,
		basename(node.name),
		basename(node.displayName),
		basename(node.templateName),
	].filter((value): value is string => !!normalizeName(value));

	for (const value of [...candidates]) {
		if (value.startsWith("step-")) {
			candidates.push(value.slice("step-".length));
		}
	}

	return Array.from(new Set(candidates));
}

function resolvePipelineNodeDisplayName(
	node: WorkflowNodeStatus,
	pipeline?: Pipeline,
): string | undefined {
	const nameMap = buildPipelineNodeNameMap(pipeline);
	for (const candidate of workflowNodeNameCandidates(node)) {
		const displayName = nameMap.get(candidate);
		if (displayName) return displayName;
	}
	return undefined;
}

export function applyPipelineNodeDisplayNames(
	workflow: WorkflowDetail | null,
	pipeline?: Pipeline,
): WorkflowDetail | null {
	if (!workflow || !pipeline?.nodes?.length) return workflow;
	let changed = false;
	const nodes = workflow.nodes.map((node) => {
		const resolved = resolvePipelineNodeDisplayName(node, pipeline);
		if (!resolved || resolved === node.displayName) return node;
		changed = true;
		return {
			...node,
			displayName: resolved,
			technicalDisplayName:
				node.displayName || node.templateName || basename(node.name) || node.id,
		};
	});
	return changed ? { ...workflow, nodes } : workflow;
}
