import type { WorkflowNodeStatus } from "../api/workflowApi";

export function truncateMiddle(text: string, maxLength: number): string {
	if (maxLength < 4 || text.length <= maxLength) {
		return text;
	}
	const edge = Math.floor((maxLength - 3) / 2);
	return `${text.slice(0, edge)}...${text.slice(text.length - edge)}`;
}

export function getWorkflowNodeDisplayText(node: WorkflowNodeStatus): string {
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
