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

export function useWorkflowDetail(name?: string): UseWorkflowDetailResult {
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

	const loadWorkflow = useCallback(() => {
		if (!name) return;
		setLoading(true);
		setLoadError(null);
		getWorkflow(name)
			.then((detail) => {
				setWorkflow(detail);
				setLoadError(null);
			})
			.catch((err) => {
				console.error(err);
				setWorkflow(null);
				setLoadError(toLoadError(err));
			})
			.finally(() => setLoading(false));
	}, [name]);

	useEffect(() => {
		loadWorkflow();
	}, [loadWorkflow]);

	const loadRunEvents = useCallback(
		(opts?: { append?: boolean; cursor?: number }) => {
			if (!name) return;
			setRunEventState((current) => ({
				...current,
				loading: true,
				error: null,
			}));
			listPipelineRuns()
				.then((response) => {
					const runs = response.items ?? [];
					const run = runs.find((item) => item.workflowName === name) ?? null;
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
		[name, runEventFilters],
	);

	useEffect(() => {
		loadRunEvents();
	}, [loadRunEvents]);

	const setRunEventFilters = useCallback((filters: RunEventFilters) => {
		setRunEventFiltersState(filters);
		setRunEventState((current) => ({ ...current, nextCursor: undefined }));
	}, []);

	useEffect(() => {
		if (!name || !workflow || !ACTIVE_WORKFLOW_STATUSES.has(workflow.status)) {
			return;
		}

		const timer = window.setInterval(() => {
			getWorkflow(name)
				.then((detail) => {
					setWorkflow(detail);
					setLoadError(null);
				})
				.catch((err) => {
					console.error(err);
				});
		}, WORKFLOW_POLL_INTERVAL_MS);

		return () => window.clearInterval(timer);
	}, [name, workflow?.status, workflow]);

	const loadNodeLogs = useCallback(
		async (nodeId: string) => {
			if (!name) return;
			setLogState((current) => ({
				...current,
				loading: true,
				error: null,
			}));
			try {
				const res = await getWorkflowLogs(name, nodeId);
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
		[name],
	);

	const selectNode = useCallback(
		(node: WorkflowNodeStatus | null) => {
			if (!node || !name) {
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
		[name, selectedNodeId],
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
		if (!name || !selectedNodeId) return;

		stopFollowLogs();

		const url = getWorkflowLogStreamUrl(name, selectedNodeId);
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
	}, [name, selectedNodeId, stopFollowLogs]);

	const downloadLogs = useCallback(() => {
		const content = logState.content;
		const selNode = selectedNode;
		if (!content || !name || !selNode) return;
		const nodeName = selNode.displayName || selNode.name || selNode.id;
		const response = logState.response;
		const header = [
			`# workflow: ${name}`,
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
		a.download = `${name}-${nodeName}.log`;
		a.click();
		URL.revokeObjectURL(url);
	}, [logState.content, logState.response, name, selectedNode]);

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

	return {
		workflow,
		loading,
		loadError,
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
