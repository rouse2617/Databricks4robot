import { request } from "./pipelineClient";

export interface WorkflowSummary {
	name: string;
	status: string;
	nodeCount: number;
	createdAt: string;
	finishedAt?: string;
	labels?: Record<string, string>;
}

export interface ListWorkflowsParams {
	status?: string;
	name?: string;
	label?: string[];
	createdAfter?: string;
	finishedBefore?: string;
}

export interface WorkflowNodeStatus {
	id: string;
	name: string;
	displayName: string;
	type?: string;
	templateName?: string;
	phase: string;
	message?: string;
	children?: string[];
	startedAt?: string;
	finishedAt?: string;
	estimatedDuration?: number;
	progress?: string;
	hostNodeName?: string;
	podName?: string;
	outputs?: {
		parameters?: Array<{ name: string; value?: string }>;
		artifacts?: Array<{ name: string; path?: string }>;
		result?: string;
		exitCode?: number;
	};
	inputs?: {
		parameters?: Array<{ name: string; value?: string }>;
		artifacts?: Array<{ name: string; path?: string }>;
	};
	resourcesDuration?: Record<string, number>;
	memoizationStatus?: {
		hit: boolean;
		key: string;
		cacheName: string;
	};
}

export type WorkflowDagEdgeKind = "runtime" | "dag" | "fallback";

export interface WorkflowDagEdge {
	id: string;
	source: string;
	target: string;
	kind: WorkflowDagEdgeKind;
}

export interface WorkflowDetail {
	name: string;
	status: string;
	message?: string;
	nodes: WorkflowNodeStatus[];
	edges?: WorkflowDagEdge[];
	createdAt: string;
	finishedAt?: string;
	labels?: Record<string, string>;
	estimatedDuration?: number;
	progress?: string;
}

export interface WorkflowOperationResponse {
	message: string;
}

export function listWorkflows(
	params: ListWorkflowsParams = {},
): Promise<{ items: WorkflowSummary[] }> {
	const sp = new URLSearchParams();
	if (params.status) sp.set("status", params.status);
	if (params.name) sp.set("name", params.name);
	if (params.label && params.label.length > 0) {
		for (const label of params.label) {
			sp.append("label", label);
		}
	}
	if (params.createdAfter) sp.set("createdAfter", params.createdAfter);
	if (params.finishedBefore) sp.set("finishedBefore", params.finishedBefore);

	const query = sp.toString();
	return request("GET", query ? `/workflows?${query}` : "/workflows");
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

function postWorkflowOperation(
	name: string,
	operation: string,
): Promise<WorkflowOperationResponse> {
	return request(
		"POST",
		`/workflows/${encodeURIComponent(name)}/${operation}`,
		{},
	);
}

export function retryWorkflow(
	name: string,
): Promise<WorkflowOperationResponse> {
	return postWorkflowOperation(name, "retry");
}

export function resubmitWorkflow(
	name: string,
): Promise<WorkflowOperationResponse> {
	return postWorkflowOperation(name, "resubmit");
}

export function suspendWorkflow(
	name: string,
): Promise<WorkflowOperationResponse> {
	return postWorkflowOperation(name, "suspend");
}

export function stopWorkflow(name: string): Promise<WorkflowOperationResponse> {
	return postWorkflowOperation(name, "stop");
}

export function resumeWorkflow(
	name: string,
): Promise<WorkflowOperationResponse> {
	return postWorkflowOperation(name, "resume");
}

export function terminateWorkflow(
	name: string,
): Promise<WorkflowOperationResponse> {
	return postWorkflowOperation(name, "terminate");
}

export function deleteWorkflow(
	name: string,
): Promise<WorkflowOperationResponse> {
	return request("DELETE", `/workflows/${encodeURIComponent(name)}`);
}
