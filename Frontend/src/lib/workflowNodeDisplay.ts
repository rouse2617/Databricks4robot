import type { WorkflowNodeStatus } from "../api/workflowApi";
import type { Pipeline } from "../components/pipeline/types";

export function truncateMiddle(text: string, maxLength: number): string {
	if (maxLength < 4 || text.length <= maxLength) {
		return text;
	}
	const edge = Math.floor((maxLength - 3) / 2);
	return `${text.slice(0, edge)}...${text.slice(text.length - edge)}`;
}

export function buildPipelineNodeLabelLookup(
	pipeline?: Pipeline | null,
): Map<string, string> {
	const lookup = new Map<string, string>();
	if (!pipeline?.nodes?.length) {
		return lookup;
	}
	for (const node of pipeline.nodes) {
		const label = node.component?.name?.trim() || node.id;
		const nodeId = node.id.trim();
		if (!nodeId) continue;
		lookup.set(nodeId, label);
		lookup.set(`step-${nodeId}`, label);
		const stripped = nodeId.replace(/^step-/, "");
		if (stripped !== nodeId) {
			lookup.set(`step-${stripped}`, label);
		}
	}
	return lookup;
}

function workflowNodeLookupKeys(node: WorkflowNodeStatus): string[] {
	const keys = [
		node.templateName?.trim(),
		node.displayName?.trim(),
		node.name?.split(".").pop()?.trim(),
	];
	const out: string[] = [];
	const seen = new Set<string>();
	for (const key of keys) {
		if (!key || seen.has(key)) continue;
		seen.add(key);
		out.push(key);
	}
	return out;
}

export function getWorkflowNodeDisplayText(
	node: WorkflowNodeStatus,
	pipelineLabels?: Map<string, string> | null,
): string {
	if (pipelineLabels && pipelineLabels.size > 0) {
		for (const key of workflowNodeLookupKeys(node)) {
			const label = pipelineLabels.get(key);
			if (label) {
				return label;
			}
		}
	}
	return node.displayName || node.templateName || node.name;
}

export function getWorkflowNodePodName(node: WorkflowNodeStatus): string {
	if (node.podName?.trim()) {
		return node.podName.trim();
	}
	const type = (node.type ?? "").toLowerCase();
	if (type === "pod" && node.id) {
		return node.id;
	}
	return node.name || node.id || "—";
}
