import type { WorkflowNodeStatus } from "../../../api/workflowApi";

/** Read-only runtime view aligned with Argo `status.nodes` fields. */
export interface ArgoNodeRuntimeView {
	argoNodeId: string;
	name: string;
	displayName: string;
	templateName?: string;
	type?: string;
	phase: string;
	message?: string;
	startedAt?: string;
	finishedAt?: string;
	progress?: string;
	podName?: string;
	hostNodeName?: string;
	children?: string[];
}

export function toArgoNodeRuntimeView(
	node: WorkflowNodeStatus,
): ArgoNodeRuntimeView {
	return {
		argoNodeId: node.id,
		name: node.name,
		displayName: node.displayName,
		templateName: node.templateName,
		type: node.type,
		phase: node.phase,
		message: node.message,
		startedAt: node.startedAt,
		finishedAt: node.finishedAt,
		progress: node.progress,
		podName: node.podName,
		hostNodeName: node.hostNodeName,
		children: node.children,
	};
}

export type { WorkflowNodeStatus };
