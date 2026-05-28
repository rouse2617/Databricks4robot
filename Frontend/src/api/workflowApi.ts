import { request } from "./pipelineClient";

export interface WorkflowSummary {
	name: string;
	status: string;
	nodeCount: number;
	createdAt: string;
	finishedAt?: string;
}

export interface WorkflowNodeStatus {
	id: string;
	name: string;
	displayName: string;
	phase: string;
	message?: string;
	startedAt?: string;
	finishedAt?: string;
}

export interface WorkflowDetail {
	name: string;
	status: string;
	message?: string;
	nodes: WorkflowNodeStatus[];
	createdAt: string;
	finishedAt?: string;
}

export function listWorkflows(): Promise<{ items: WorkflowSummary[] }> {
	return request("GET", "/workflows");
}

export function getWorkflow(name: string): Promise<WorkflowDetail> {
	return request("GET", `/workflows/${encodeURIComponent(name)}`);
}

export function getWorkflowLogs(
	name: string,
	nodeId: string,
): Promise<{ logs: string }> {
	return request(
		"GET",
		`/workflows/${encodeURIComponent(name)}/logs?nodeId=${encodeURIComponent(nodeId)}`,
	);
}
