import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
	getPipelineRunCostSummary,
	listPipelineRunAssetNodes,
	listPipelineRunEvents,
	listPipelineRuns,
	type PipelineRun,
	type PipelineRunAssetNode,
	type PipelineRunCostSummary,
	type PipelineRunEvent,
} from "../api/pipelineApi";
import { ApiError } from "../api/pipelineClient";
import {
	getWorkflow,
	getWorkflowLogStreamUrl,
	getWorkflowLogs,
	type WorkflowDetail,
	type WorkflowLogResponse,
	type WorkflowNodeStatus,
} from "../api/workflowApi";

export type WorkflowLogFollowStatus =
	| "idle"
	| "connecting"
	| "connected"
	| "ended"
	| "error";

interface WorkflowLogState {
	content: string | null;
	loading: boolean;
	error: string | null;
	search: string;
	following: boolean;
	followStatus: WorkflowLogFollowStatus;
	followMessage: string | null;
	response: WorkflowLogResponse | null;
	clientTruncated: boolean;
}

interface RunEventState {
	run: PipelineRun | null;
	items: PipelineRunEvent[];
	nextCursor?: number;
	loading: boolean;
	error: string | null;
}

interface RunEventFilters {
	eventType?: string;
	status?: string;
	subjectType?: string;
	q?: string;
}

interface WorkflowRunLookup {
	workflowName: string;
	runId?: string;
	normalizedName?: string;
	runIdSource?: "name" | "run-id" | "fallback";
}

const FAILED_NODE_PHASES = new Set(["Failed", "Error"]);
const ACTIVE_NODE_PHASES = new Set([
	"Running",
	"Pending",
	"Initializing",
	"PendingCreate",
	"PodInitializing",
	"ContainerCreating",
	"Unknown",
]);
const TERMINAL_NODE_PHASES = new Set(["Succeeded", "Skipped", "Omitted"]);
const TERMINAL_WORKFLOW_PHASES = new Set([
	"succeeded",
	"failed",
	"error",
	"skipped",
	"omitted",
]);

function normalizeWorkflowStatus(status: string): string {
	return status.toLowerCase().trim();
}

function buildStatusSyncWarning(workflow: WorkflowDetail): string | undefined {
	const workflowStatus = normalizeWorkflowStatus(workflow.status);
	const failedNodes = workflow.nodes.filter((node) =>
		FAILED_NODE_PHASES.has(node.phase),
	);
	const activeNodes = workflow.nodes.filter((node) =>
		ACTIVE_NODE_PHASES.has(node.phase),
	);
	const terminalNodes = workflow.nodes.filter((node) =>
		TERMINAL_NODE_PHASES.has(node.phase),
	);

	if (
		workflowStatus === "running" &&
		failedNodes.length > 0 &&
		activeNodes.length === 0
	) {
		const firstFailedNode = failedNodes[0];
		const reason = firstFailedNode?.message
			? `，先失败节点: ${firstFailedNode.displayName || firstFailedNode.id}，原因: ${firstFailedNode.message}`
			: "";
		return `工作流状态仍为 Running，但检测到 ${failedNodes.length} 个失败/错误节点${reason}。请确认是否已发生状态回写延迟。`;
	}

	if (TERMINAL_WORKFLOW_PHASES.has(workflowStatus) && activeNodes.length > 0) {
		return `工作流状态已为 ${workflow.status}，但仍有 ${activeNodes.length} 个节点显示运行中，建议稍后刷新并检查节点消息。`;
	}

	if (
		workflowStatus === "running" &&
		terminalNodes.length === workflow.nodes.length
	) {
		return "工作流状态仍为 Running，但所有节点已为终态，可能存在状态同步延迟。";
	}

	return undefined;
}

const workflowLookupCacheKey = (lookup: WorkflowRunLookup | null): string => {
	if (!lookup) return "";
	const runId = normalizeInput(lookup.runId) || "";
	const name = normalizeInput(lookup.workflowName) || "";
	return `${name}::${runId}`;
};

function uniqueLookupCandidates(values: Array<string | undefined>): string[] {
	const seen = new Set<string>();
	const result: string[] = [];
	for (const value of values) {
		const normalized = normalizeInput(value);
		if (!normalized || seen.has(normalized)) continue;
		seen.add(normalized);
		result.push(normalized);
	}
	return result;
}

function pickWorkflowLookupName(
	runs: PipelineRun[],
	name?: string,
	runId?: string,
): WorkflowRunLookup | null {
	const normalizedName = normalizeInput(name);
	const normalizedRunId = normalizeInput(runId);
	const fallbackRunId = normalizedRunId || "";
	if (!normalizedName && !normalizedRunId) {
		return null;
	}

	const exactById = runs.find((run) => run.id === normalizedRunId);
	if (exactById) {
		return {
			workflowName:
				exactById.workflowName || exactById.pipelineName || fallbackRunId,
			runId: exactById.id,
			normalizedName,
			runIdSource: "run-id",
		};
	}

	const matchedByWorkflowName = runs.find(
		(run) =>
			run.workflowName === normalizedRunId ||
			run.pipelineName === normalizedRunId ||
			run.id === normalizedRunId,
	);
	if (matchedByWorkflowName) {
		return {
			workflowName:
				matchedByWorkflowName.workflowName ||
				matchedByWorkflowName.pipelineName ||
				fallbackRunId,
			runId: matchedByWorkflowName.id,
			normalizedName,
			runIdSource: "run-id",
		};
	}

	if (normalizedName) {
		const fallbackByName = runs.find(
			(run) =>
				run.workflowName === normalizedName ||
				run.pipelineName === normalizedName ||
				run.id === normalizedName,
		);
		if (fallbackByName) {
			return {
				workflowName:
					fallbackByName.workflowName ||
					fallbackByName.pipelineName ||
					normalizedName,
				runId: fallbackByName.id,
				normalizedName,
				runIdSource: "name",
			};
		}
	}

	return {
		workflowName: normalizedName || fallbackRunId,
		normalizedName: normalizedName || fallbackRunId,
		runId: normalizedRunId,
		runIdSource: "fallback",
	};
}

interface AssetNodeState {
	items: PipelineRunAssetNode[];
	loading: boolean;
	error: string | null;
	summary: {
		assetCount: number;
		nodeCount: number;
		statuses: Record<string, number>;
		totalEstimatedCostUsd?: number;
		costSource: string;
	} | null;
}

interface CostSummaryState {
	item: PipelineRunCostSummary | null;
	loading: boolean;
	error: string | null;
}

export type WorkflowLoadErrorKind = "not_found" | "error";

export interface WorkflowLoadError {
	kind: WorkflowLoadErrorKind;
	message: string;
}

interface UseWorkflowDetailResult {
	workflow: WorkflowDetail | null;
	loading: boolean;
	loadError: WorkflowLoadError | null;
	statusSyncWarning?: string;
	selectedNode: WorkflowNodeStatus | null;
	selectNode: (node: WorkflowNodeStatus | null) => void;
	loadWorkflow: () => void;
	logState: WorkflowLogState;
	runEventState: RunEventState;
	runEventFilters: RunEventFilters;
	setRunEventFilters: (filters: RunEventFilters) => void;
	loadRunEvents: (opts?: { append?: boolean; cursor?: number }) => void;
	assetNodeState: AssetNodeState;
	costSummaryState: CostSummaryState;
	setLogSearch: (query: string) => void;
	startFollowLogs: () => void;
	stopFollowLogs: () => void;
	downloadLogs: () => void;
}

const EMPTY_LOG_STATE: WorkflowLogState = {
	content: null,
	loading: false,
	error: null,
	search: "",
	following: false,
	followStatus: "idle",
	followMessage: null,
	response: null,
	clientTruncated: false,
};

const EMPTY_RUN_EVENT_STATE: RunEventState = {
	run: null,
	items: [],
	loading: false,
	error: null,
};

const EMPTY_ASSET_NODE_STATE: AssetNodeState = {
	items: [],
	loading: false,
	error: null,
	summary: null,
};

const EMPTY_COST_SUMMARY_STATE: CostSummaryState = {
	item: null,
	loading: false,
	error: null,
};

const ACTIVE_WORKFLOW_STATUSES = new Set(["Running", "Pending"]);
const WORKFLOW_POLL_INTERVAL_MS = 8_000;
const LOG_CLIENT_BUFFER_CHARS = 1_000_000;

function normalizeInput(value?: string): string | undefined {
	return value?.trim() || undefined;
}

function buildWorkflowRunLookup(
	name?: string,
	runId?: string,
	runs: PipelineRun[] = [],
): WorkflowRunLookup | null {
	return pickWorkflowLookupName(runs, name, runId);
}

function findRunByLookup(
	runs: PipelineRun[],
	lookup: WorkflowRunLookup | null,
): PipelineRun | null {
	if (!lookup?.workflowName) return null;
	const exactById = lookup.runId
		? runs.find((run) => run.id === lookup.runId)
		: null;
	if (exactById) return exactById;

	const byWorkflowOrPipeline = runs.find(
		(run) =>
			run.workflowName === lookup.workflowName ||
			run.pipelineName === lookup.workflowName,
	);
	if (byWorkflowOrPipeline) return byWorkflowOrPipeline;

	return runs.find((run) => run.id === lookup.workflowName) ?? null;
}

function toErrorMessage(err: unknown): string {
	if (err instanceof Error) {
		return err.message;
	}
	return String(err);
}

function toLoadError(err: unknown): WorkflowLoadError {
	if (err instanceof ApiError && err.status === 404) {
		return { kind: "not_found", message: err.message };
	}
	return { kind: "error", message: toErrorMessage(err) };
}

function appendBoundedLogContent(
	current: string | null,
	line: string,
): { content: string; truncated: boolean } {
	const prefix = current && !current.endsWith("\n") ? "\n" : "";
	const next = `${current ?? ""}${prefix}${line}\n`;
	if (next.length <= LOG_CLIENT_BUFFER_CHARS) {
		return { content: next, truncated: false };
	}
	return {
		content: next.slice(-LOG_CLIENT_BUFFER_CHARS),
		truncated: true,
	};
}

export function useWorkflowDetail(
	name?: string,
	runId?: string,
): UseWorkflowDetailResult {
	const [workflow, setWorkflow] = useState<WorkflowDetail | null>(null);
	const [loading, setLoading] = useState(true);
	const [loadError, setLoadError] = useState<WorkflowLoadError | null>(null);
	const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
	const [logState, setLogState] = useState<WorkflowLogState>(EMPTY_LOG_STATE);
	const followSourceRef = useRef<EventSource | null>(null);
	const [runEventState, setRunEventState] = useState<RunEventState>(
		EMPTY_RUN_EVENT_STATE,
	);
	const [runEventFilters, setRunEventFiltersState] = useState<RunEventFilters>(
		{},
	);
	const [assetNodeState, setAssetNodeState] = useState<AssetNodeState>(
		EMPTY_ASSET_NODE_STATE,
	);
	const [costSummaryState, setCostSummaryState] = useState<CostSummaryState>(
		EMPTY_COST_SUMMARY_STATE,
	);
	const [workflowLookup, setWorkflowLookup] =
		useState<WorkflowRunLookup | null>(
			name ? { workflowName: name, runId: normalizeInput(runId) } : null,
		);
	const [lookupReady, setLookupReady] = useState(false);

	const resolveWorkflowLookup = useCallback(async () => {
		try {
			const runs = await listPipelineRuns();
			const lookup = buildWorkflowRunLookup(name, runId, runs);
			setWorkflowLookup(lookup);
			return lookup;
		} catch (err) {
			console.error(err);
			const fallbackName = normalizeInput(name) ?? normalizeInput(runId);
			const fallback = fallbackName ? { workflowName: fallbackName } : null;
			setWorkflowLookup(fallback);
			return fallback;
		} finally {
			setLookupReady(true);
		}
	}, [name, runId]);

	useEffect(() => {
		setLookupReady(false);
		setWorkflowLookup((prev) => {
			const next = name
				? { workflowName: name, runId: normalizeInput(runId) }
				: null;
			const nextCacheKey = workflowLookupCacheKey(next);
			const prevCacheKey = workflowLookupCacheKey(prev);
			return next && nextCacheKey === prevCacheKey ? prev : next;
		});
		void resolveWorkflowLookup();
	}, [name, runId, resolveWorkflowLookup]);

	const fetchWorkflowByCandidates = useCallback(
		async (candidates: string[]): Promise<WorkflowDetail | null> => {
			let lastError: unknown;
			for (const candidate of candidates) {
				try {
					return await getWorkflow(candidate);
				} catch (err) {
					lastError = err;
					if (err instanceof ApiError && err.status === 404) {
						continue;
					}
					throw err;
				}
			}
			if (lastError) throw lastError;
			throw new ApiError(404, "WORKFLOW_NOT_FOUND", "workflow not found");
		},
		[],
	);

	const loadWorkflow = useCallback(async () => {
		if (!lookupReady || !workflowLookup?.workflowName) return;
		setLoading(true);
		setLoadError(null);
		try {
			const candidates = uniqueLookupCandidates([
				workflowLookup.workflowName,
				workflowLookup.runId,
				name,
				workflowLookup.normalizedName,
			]);
			const detail = await fetchWorkflowByCandidates(candidates);
			setWorkflow(detail);
			setLoadError(null);
		} catch (err) {
			console.error(err);
			setWorkflow(null);
			setLoadError(toLoadError(err));
		} finally {
			setLoading(false);
		}
	}, [fetchWorkflowByCandidates, lookupReady, name, workflowLookup]);

	useEffect(() => {
		if (!lookupReady) return;
		loadWorkflow();
	}, [loadWorkflow, lookupReady]);

	const loadRunEvents = useCallback(
		(opts?: { append?: boolean; cursor?: number }) => {
			if (!lookupReady || !workflowLookup?.workflowName) return;
			setRunEventState((current) => ({
				...current,
				loading: true,
				error: null,
			}));
			listPipelineRuns()
				.then((runs) => {
					const run = findRunByLookup(runs, workflowLookup);
					if (!run) {
						setRunEventState({
							run: null,
							items: [],
							loading: false,
							error: "未找到关联的 DataBrew pipeline run",
						});
						return;
					}
					const cursor = opts?.append ? opts.cursor : undefined;
					return Promise.all([
						listPipelineRunEvents(run.id, {
							limit: 100,
							cursor,
							...runEventFilters,
						}),
						listPipelineRunAssetNodes(run.id, { limit: 500 }),
						getPipelineRunCostSummary(run.id),
					]).then(([events, assetNodes, costSummary]) => {
						setRunEventState((current) => ({
							run,
							items: opts?.append
								? [...current.items, ...(events.items ?? [])]
								: (events.items ?? []),
							nextCursor: events.nextCursor,
							loading: false,
							error: null,
						}));
						setAssetNodeState({
							items: assetNodes.items ?? [],
							loading: false,
							error: null,
							summary: assetNodes.summary ?? null,
						});
						setCostSummaryState({
							item: costSummary,
							loading: false,
							error: null,
						});
					});
				})
				.catch((err) => {
					console.error(err);
					setRunEventState((current) => ({
						...current,
						loading: false,
						error: toErrorMessage(err),
					}));
					setAssetNodeState((current) => ({
						...current,
						loading: false,
						error: toErrorMessage(err),
					}));
					setCostSummaryState((current) => ({
						...current,
						loading: false,
						error: toErrorMessage(err),
					}));
				});
		},
		[workflowLookup, lookupReady, runEventFilters],
	);

	useEffect(() => {
		loadRunEvents();
	}, [loadRunEvents]);

	const setRunEventFilters = useCallback((filters: RunEventFilters) => {
		setRunEventFiltersState(filters);
		setRunEventState((current) => ({ ...current, nextCursor: undefined }));
	}, []);

	useEffect(() => {
		if (
			!lookupReady ||
			!workflowLookup?.workflowName ||
			!workflow ||
			!ACTIVE_WORKFLOW_STATUSES.has(workflow.status)
		) {
			return;
		}

		const timer = window.setInterval(() => {
			getWorkflow(workflowLookup.workflowName)
				.then((detail) => {
					setWorkflow(detail);
					setLoadError(null);
				})
				.catch((err) => {
					console.error(err);
				});
		}, WORKFLOW_POLL_INTERVAL_MS);

		return () => window.clearInterval(timer);
	}, [lookupReady, workflowLookup?.workflowName, workflow?.status, workflow]);

	const loadNodeLogs = useCallback(
		async (nodeId: string) => {
			if (!workflowLookup?.workflowName) return;
			const workflowName = workflowLookup.workflowName;
			setLogState((current) => ({
				...current,
				loading: true,
				error: null,
			}));
			try {
				const res = await getWorkflowLogs(workflowName, nodeId);
				setLogState((current) => ({
					content: res.logs || "",
					loading: false,
					error: null,
					search: current.search,
					following: current.following,
					followStatus: current.followStatus,
					followMessage: current.followMessage,
					response: res,
					clientTruncated: false,
				}));
			} catch (err) {
				setLogState((current) => ({
					content: null,
					loading: false,
					error: toErrorMessage(err),
					search: current.search,
					following: current.following,
					followStatus: current.followStatus,
					followMessage: current.followMessage,
					response: null,
					clientTruncated: false,
				}));
			}
		},
		[workflowLookup?.workflowName],
	);

	const selectNode = useCallback(
		(node: WorkflowNodeStatus | null) => {
			if (!node || !workflowLookup?.workflowName) {
				setSelectedNodeId(null);
				setLogState((current) => ({
					...EMPTY_LOG_STATE,
					search: current.search,
				}));
				return;
			}
			setSelectedNodeId(node.id);
			if (node.id !== selectedNodeId) {
				setLogState((current) => ({
					...current,
					content: "",
					loading: false,
					error: null,
					response: null,
					clientTruncated: false,
				}));
			}
		},
		[workflowLookup?.workflowName, selectedNodeId],
	);

	const selectedNode = useMemo(() => {
		if (!workflow || !selectedNodeId) return null;
		return workflow.nodes.find((node) => node.id === selectedNodeId) ?? null;
	}, [workflow, selectedNodeId]);

	const stopFollowLogs = useCallback(() => {
		if (followSourceRef.current) {
			followSourceRef.current.close();
			followSourceRef.current = null;
		}
		setLogState((prev) => ({
			...prev,
			following: false,
			followStatus: "idle",
			followMessage: "实时日志已停止",
		}));
	}, []);

	const startFollowLogs = useCallback(() => {
		if (!workflowLookup?.workflowName || !selectedNodeId) return;

		stopFollowLogs();

		const url = getWorkflowLogStreamUrl(
			workflowLookup.workflowName,
			selectedNodeId,
		);
		const source = new EventSource(url);
		followSourceRef.current = source;

		setLogState((prev) => ({
			...prev,
			following: true,
			followStatus: "connecting",
			followMessage: "正在连接实时日志",
			error: null,
		}));

		source.onopen = () => {
			setLogState((prev) => ({
				...prev,
				following: true,
				followStatus: "connected",
				followMessage: "实时日志已连接",
			}));
		};

		source.addEventListener("log", (event: MessageEvent) => {
			try {
				const data = JSON.parse(event.data);
				const line =
					typeof data.line === "string"
						? data.line
						: typeof data.content === "string"
							? data.content
							: "";
				if (line) {
					setLogState((prev) => ({
						...prev,
						...(() => {
							const next = appendBoundedLogContent(prev.content, line);
							return {
								content: next.content,
								clientTruncated: prev.clientTruncated || next.truncated,
							};
						})(),
						loading: false,
						followStatus: "connected",
						followMessage: "实时日志已连接",
					}));
				}
			} catch {
				// ignore malformed events
			}
		});

		source.addEventListener("end", (event: MessageEvent) => {
			let reason = "实时日志已结束";
			try {
				const data = JSON.parse(event.data);
				if (typeof data.reason === "string" && data.reason) {
					reason =
						data.reason === "limit-bytes"
							? "已达到本次实时日志字节上限"
							: "实时日志已结束";
				}
			} catch {
				// ignore malformed events
			}
			source.close();
			if (followSourceRef.current === source) {
				followSourceRef.current = null;
			}
			setLogState((prev) => ({
				...prev,
				following: false,
				followStatus: "ended",
				followMessage: reason,
			}));
		});

		source.onerror = () => {
			source.close();
			if (followSourceRef.current === source) {
				followSourceRef.current = null;
			}
			setLogState((prev) => ({
				...prev,
				following: false,
				followStatus: "error",
				followMessage: "实时日志连接已断开",
			}));
		};
	}, [workflowLookup?.workflowName, selectedNodeId, stopFollowLogs]);

	const downloadLogs = useCallback(() => {
		const content = logState.content;
		const selNode = selectedNode;
		if (!content || !workflowLookup?.workflowName || !selNode) return;
		const nodeName = selNode.displayName || selNode.name || selNode.id;
		const response = logState.response;
		const header = [
			`# workflow: ${workflowLookup.workflowName}`,
			`# node: ${nodeName}`,
			`# container: ${response?.container ?? "main"}`,
			`# scope: ${response?.window?.scope ?? "loaded-log-window"}`,
			`# tailLines: ${response?.truncation?.tailLines ?? "unknown"}`,
			`# limitBytes: ${response?.truncation?.limitBytes ?? "unknown"}`,
			`# generatedAt: ${new Date().toISOString()}`,
			"",
		].join("\n");
		const blob = new Blob([header, content], { type: "text/plain" });
		const url = URL.createObjectURL(blob);
		const a = document.createElement("a");
		a.href = url;
		a.download = `${workflowLookup.workflowName}-${nodeName}.log`;
		a.click();
		URL.revokeObjectURL(url);
	}, [
		logState.content,
		logState.response,
		selectedNode,
		workflowLookup?.workflowName,
	]);

	useEffect(() => {
		return () => {
			if (followSourceRef.current) {
				followSourceRef.current.close();
				followSourceRef.current = null;
			}
		};
	}, []);

	useEffect(() => {
		if (!selectedNodeId || !name) {
			return;
		}
		void loadNodeLogs(selectedNodeId);
	}, [loadNodeLogs, name, selectedNodeId]);

	useEffect(() => {
		if (!workflow || !selectedNodeId) return;
		const selectedNodeExists = workflow.nodes.some(
			(node) => node.id === selectedNodeId,
		);
		if (!selectedNodeExists) {
			setSelectedNodeId(null);
			setLogState((current) => ({
				...EMPTY_LOG_STATE,
				search: current.search,
			}));
		}
	}, [selectedNodeId, workflow]);
	const statusSyncWarning = useMemo(
		() => (workflow ? buildStatusSyncWarning(workflow) : undefined),
		[workflow],
	);

	return {
		workflow,
		loading,
		loadError,
		statusSyncWarning,
		selectedNode,
		loadWorkflow,
		selectNode,
		startFollowLogs,
		stopFollowLogs,
		downloadLogs,
		logState: {
			...logState,
		},
		runEventState: {
			...runEventState,
			items: [...runEventState.items],
		},
		runEventFilters: { ...runEventFilters },
		setRunEventFilters,
		loadRunEvents,
		assetNodeState: {
			...assetNodeState,
			items: [...assetNodeState.items],
		},
		costSummaryState: { ...costSummaryState },
		setLogSearch: useCallback((query: string) => {
			setLogState((current) => ({ ...current, search: query }));
		}, []),
	};
}
