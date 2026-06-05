import { request } from "./pipelineClient";

export interface WorkflowSummary {
	name: string;
	status: string;
	nodeCount: number;
	assetCount?: number;
	createdAt: string;
	finishedAt?: string;
	labels?: Record<string, string>;
	estimatedCostUsd?: number | null;
	totalEstimatedCost?: number | null;
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
	technicalDisplayName?: string;
	type?: string;
	templateName?: string;
	phase: string;
	message?: string;
	cluster?: string;
	namespace?: string;
	serviceAccountName?: string;
	podIp?: string;
	children?: string[];
	startedAt?: string;
	finishedAt?: string;
	estimatedDuration?: number;
	progress?: string;
	hostNodeName?: string;
	podName?: string;
	restartCount?: number;
	containers?: WorkflowNodeContainer[];
	podConditions?: WorkflowPodCondition[];
	podEvents?: WorkflowPodEvent[];
	metrics?: WorkflowPodMetrics;
	cost?: WorkflowPodCost;
	estimatedCostUsd?: number;
	debug?: WorkflowPodDebugCapabilities;
	outputs?: {
		parameters?: Array<{ name: string; value?: string }>;
		artifacts?: Array<{ name: string; path?: string }>;
		result?: string;
		exitCode?: number | string;
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

export interface WorkflowNodeContainer {
	name: string;
	image?: string;
	command?: string[];
	args?: string[];
	ready?: boolean;
	restartCount?: number;
	state?: string;
	lastState?: string;
}

export interface WorkflowPodCondition {
	type: string;
	status: string;
	reason?: string;
	message?: string;
	lastTransitionTime?: string;
}

export interface WorkflowPodEvent {
	type: string;
	reason: string;
	message: string;
	count?: number;
	firstTimestamp?: string;
	lastTimestamp?: string;
}

export interface WorkflowPodMetrics {
	cpuCores?: number;
	cpuRequestCores?: number;
	cpuLimitCores?: number;
	memoryBytes?: number;
	memoryRequestBytes?: number;
	memoryLimitBytes?: number;
	gpuCount?: number;
	networkRxBytes?: number;
	networkTxBytes?: number;
	storageBytes?: number;
	sampledAt?: string;
}

export interface WorkflowPodCost {
	totalCostUsd?: number;
	cpuCostUsd?: number;
	memoryCostUsd?: number;
	gpuCostUsd?: number;
	storageCostUsd?: number;
	networkCostUsd?: number;
	window?: string;
	provider?: "opencost" | "custom";
	calculatedAt?: string;
}

export interface WorkflowPodDebugCapabilities {
	execEnabled?: boolean;
	logStreamEnabled?: boolean;
	metricsEnabled?: boolean;
	costEnabled?: boolean;
	reason?: string;
	allowedCommands?: string[];
	maxSessionSeconds?: number;
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

export interface WorkflowLogTruncation {
	bounded: boolean;
	tailLines: number;
	maxTailLines: number;
	tailLinesClamped?: boolean;
	limitBytes: number;
	maxLimitBytes: number;
	limitBytesClamped?: boolean;
	bytesTruncated?: boolean;
	sinceSeconds?: number;
	sinceTime?: string;
}

export interface WorkflowLogPagination {
	available: boolean;
	nextCursor?: string | null;
	reason?: string;
}

export interface WorkflowLogWindow {
	mode: "tail" | "since" | "cursor";
	tailLines?: number;
	limitBytes?: number;
	sinceSeconds?: number;
	sinceTime?: string;
	previous?: boolean;
	timestamps?: boolean;
	scope?: string;
}

export interface WorkflowLogResponse {
	workflowName: string;
	nodeId: string;
	podName: string;
	container: string;
	source: "argo-live";
	logs: string;
	lineCount: number;
	truncated: boolean;
	nextCursor?: string | null;
	truncation: WorkflowLogTruncation;
	pagination?: WorkflowLogPagination;
	window?: WorkflowLogWindow;
}

export interface WorkflowLogQuery {
	tailLines?: number;
	limitBytes?: number;
	cursor?: string;
	container?: string;
	sinceSeconds?: number;
	sinceTime?: string;
	previous?: boolean;
	timestamps?: boolean;
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
	params: WorkflowLogQuery = {},
): Promise<WorkflowLogResponse> {
	const sp = new URLSearchParams();
	sp.set("nodeId", nodeId);
	for (const [key, value] of Object.entries(params)) {
		if (value === undefined || value === null || value === "") continue;
		sp.set(key, String(value));
	}
	return request("GET", `/workflows/${encodeURIComponent(name)}/logs?${sp}`);
}

export function getWorkflowLogStreamUrl(
	name: string,
	nodeId: string,
	params: WorkflowLogQuery = {},
): string {
	const sp = new URLSearchParams();
	sp.set("nodeId", nodeId);
	for (const [key, value] of Object.entries(params)) {
		if (value === undefined || value === null || value === "") continue;
		sp.set(key, String(value));
	}
	return `/api/v1/workflows/${encodeURIComponent(name)}/logs/stream?${sp}`;
}

export interface NodePodDiagnostics {
	cluster?: string;
	namespace: string;
	podName: string;
	podIp?: string;
	serviceAccountName?: string;
	restartCount: number;
	containers: WorkflowNodeContainer[];
	podConditions: WorkflowPodCondition[];
	podEvents: WorkflowPodEvent[];
}

export function getNodePodDiagnostics(
	workflowName: string,
	nodeId: string,
): Promise<NodePodDiagnostics> {
	return request(
		"GET",
		`/workflows/${encodeURIComponent(workflowName)}/nodes/${encodeURIComponent(nodeId)}/pod`,
	);
}

export interface TerminalSession {
	id: string;
	runId?: string;
	workflowName: string;
	nodeId: string;
	podName: string;
	containerName?: string;
	executionTargetId?: string;
	cluster?: string;
	namespace: string;
	command: string;
	status: string;
	attachUrl?: string;
	expiresAt: string;
	createdAt: string;
	attachedAt?: string;
	endedAt?: string;
	errorCode?: string;
	errorMessage?: string;
}

export interface CreateTerminalSessionRequest {
	containerName?: string;
	command: string;
}

export function createTerminalSession(
	workflowName: string,
	nodeId: string,
	body: CreateTerminalSessionRequest,
): Promise<TerminalSession> {
	return request(
		"POST",
		`/workflows/${encodeURIComponent(workflowName)}/nodes/${encodeURIComponent(nodeId)}/terminal-sessions`,
		body,
	);
}

export function getTerminalSession(
	sessionId: string,
): Promise<TerminalSession> {
	return request(
		"GET",
		`/pod-terminal/sessions/${encodeURIComponent(sessionId)}`,
	);
}

export function terminateTerminalSession(
	sessionId: string,
): Promise<TerminalSession> {
	return request(
		"POST",
		`/pod-terminal/sessions/${encodeURIComponent(sessionId)}/terminate`,
		{},
	);
}

export function getTerminalAttachUrl(attachUrl: string): string {
	if (attachUrl.startsWith("ws://") || attachUrl.startsWith("wss://")) {
		return attachUrl;
	}
	const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
	return `${protocol}//${window.location.host}${attachUrl}`;
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
