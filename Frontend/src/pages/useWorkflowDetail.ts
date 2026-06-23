import { useCallback, useEffect, useMemo, useRef, useState, startTransition } from "react";
import type {
	PipelineRun,
	PipelineRunAssetNode,
	PipelineRunCostSummary,
	PipelineRunEvent,
} from "../api/pipelineApi";
import { ApiError } from "../api/pipelineClient";
import {
	getRun,
	getRunByWorkflowName,
	getRunCostSummary,
	getRunRuntime,
	listRunAssetNodes,
	listRunEvents,
	listRunInputs,
	listRunOutputs,
	type RunInputListResponse,
	type RunOutputListResponse,
	type RunRuntime,
} from "../api/runApi";
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
	| "reconnecting"
	| "ended"
	| "error";

export interface LogOptions {
	container: string;
	previous: boolean;
	tailLines: number;
	timestamps: boolean;
}

interface WorkflowLogState {
	lines: string[];
	loading: boolean;
	error: string | null;
	search: string;
	following: boolean;
	followStatus: WorkflowLogFollowStatus;
	followMessage: string | null;
	response: WorkflowLogResponse | null;
	clientTruncated: boolean;
	options: LogOptions;
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

interface RunMetadataState {
	inputs: RunInputListResponse | null;
	outputs: RunOutputListResponse | null;
	runtime: RunRuntime | null;
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
	runMetadataState: RunMetadataState;
	setLogSearch: (query: string) => void;
	startFollowLogs: () => void;
	stopFollowLogs: () => void;
	downloadLogs: () => void;
}

type WorkflowLookupMode = "workflowName" | "runId";

interface UseWorkflowDetailOptions {
	lookupMode?: WorkflowLookupMode;
}

const EMPTY_LOG_STATE: WorkflowLogState = {
	lines: [],
	loading: false,
	error: null,
	search: "",
	following: false,
	followStatus: "idle",
	followMessage: null,
	response: null,
	clientTruncated: false,
	options: {
		container: "main",
		previous: false,
		tailLines: 200,
		timestamps: false,
	},
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

const EMPTY_RUN_METADATA_STATE: RunMetadataState = {
	inputs: null,
	outputs: null,
	runtime: null,
	loading: false,
	error: null,
};

const ACTIVE_WORKFLOW_STATUSES = new Set(["Running", "Pending"]);
const TERMINAL_WORKFLOW_STATUSES = new Set([
	"Succeeded",
	"Failed",
	"Error",
	"Skipped",
	"Omitted",
]);
const WORKFLOW_POLL_INTERVAL_MS = 8_000;
const LOG_STREAM_FLUSH_INTERVAL_MS = 1000;
const LOG_STREAM_CONNECT_GRACE_MS = 5_000;
const LOG_CLIENT_BUFFER_LINES = 5_000;
const LOG_CLIENT_BUFFER_CHARS = 1_000_000;
const WORKFLOW_FETCH_TIMEOUT_MS = 20_000;
const RUN_DETAIL_TIMEOUT_MS = 20_000;

function withTimeout<T>(
	promise: Promise<T>,
	label: string,
	timeoutMs: number,
): Promise<T> {
	return Promise.race<T>([
		promise,
		new Promise<T>((_, reject) => {
			window.setTimeout(
				() => reject(new Error(`${label} 请求超时（${timeoutMs}ms）`)),
				timeoutMs,
			);
		}),
	]);
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

function isExpectedWorkflowNotFound(err: unknown): boolean {
	return err instanceof ApiError && err.status === 404;
}

function toTimestamp(value?: string): number | null {
	if (!value) return null;
	const time = new Date(value).getTime();
	return Number.isNaN(time) ? null : time;
}

function workflowHasActiveNodesNewerThanRun(
	workflow: WorkflowDetail | null,
	run: PipelineRun | null,
): boolean {
	if (!workflow) return false;
	const activeNodes = workflow.nodes.filter((node) =>
		ACTIVE_WORKFLOW_STATUSES.has(node.phase),
	);
	if (activeNodes.length === 0) return false;
	if (!run || ACTIVE_WORKFLOW_STATUSES.has(run.status)) return true;
	if (!TERMINAL_WORKFLOW_STATUSES.has(run.status)) return true;
	const runTerminalAt = toTimestamp(run.finishedAt ?? run.updatedAt);
	if (!runTerminalAt) return false;
	return activeNodes.some((node) => {
		const nodeStartedAt = toTimestamp(node.startedAt ?? node.finishedAt);
		return nodeStartedAt != null && nodeStartedAt > runTerminalAt;
	});
}

async function resolvePipelineRun(
	lookup: string,
	mode: WorkflowLookupMode,
): Promise<PipelineRun | null> {
	if (mode === "runId") {
		try {
			return await getRun(lookup);
		} catch (err) {
			if (!(err instanceof ApiError && err.status === 404)) {
				throw err;
			}
			return null;
		}
	}
	try {
		return await getRunByWorkflowName(lookup);
	} catch (err) {
		if (!(err instanceof ApiError && err.status === 404)) {
			throw err;
		}
	}
	try {
		return await getRun(lookup);
	} catch (err) {
		if (!(err instanceof ApiError && err.status === 404)) {
			throw err;
		}
	}
	return null;
}

/** Append incoming log lines to an existing lines array with buffer limits. */
export function appendBoundedLogLines(
	current: string[],
	incoming: string[],
): { lines: string[]; truncated: boolean } {
	const cleaned = incoming
		.map((line) => line.replace(/\r$/, ""))
		.filter((line) => line.length > 0);
	if (cleaned.length === 0) {
		return { lines: current, truncated: false };
	}
	let allLines = [...current, ...cleaned];
	let truncated = false;
	if (allLines.length > LOG_CLIENT_BUFFER_LINES) {
		allLines = allLines.slice(-LOG_CLIENT_BUFFER_LINES);
		truncated = true;
	}
	return { lines: allLines, truncated };
}

const MAX_RECONNECT_DELAY_MS = 30_000;
const INITIAL_RECONNECT_DELAY_MS = 1_000;
const MAX_RECONNECT_ATTEMPTS = 5;

function calculateBackoff(attempt: number): number {
	return Math.min(
		INITIAL_RECONNECT_DELAY_MS * Math.pow(2, attempt),
		MAX_RECONNECT_DELAY_MS,
	);
}

export function useWorkflowDetail(
	name?: string,
	options?: UseWorkflowDetailOptions,
): UseWorkflowDetailResult {
	const lookupMode = options?.lookupMode ?? "workflowName";
	const [workflow, setWorkflow] = useState<WorkflowDetail | null>(null);
	const workflowRef = useRef<WorkflowDetail | null>(null);
	const [loading, setLoading] = useState(true);
	const [loadError, setLoadError] = useState<WorkflowLoadError | null>(null);
	const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
	const [logState, setLogState] = useState<WorkflowLogState>(EMPTY_LOG_STATE);
	const followSourceRef = useRef<EventSource | null>(null);
	const logStreamBufferRef = useRef<string[]>([]);
	const logStreamFlushTimerRef = useRef<number | null>(null);
	const logStreamConnectTimerRef = useRef<number | null>(null);
	const reconnectAttemptRef = useRef(0);
	const reconnectTimerRef = useRef<number | null>(null);
	const lastEventIdRef = useRef<string | null>(null);
	const skipNextRunEventsLoadRef = useRef(false);
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
	const [runMetadataState, setRunMetadataState] = useState<RunMetadataState>(
		EMPTY_RUN_METADATA_STATE,
	);
	workflowRef.current = workflow;
	const runtimeWorkflowName = useMemo(
		() =>
			workflow?.name ??
			runEventState.run?.workflowName ??
			(lookupMode === "workflowName" ? name : undefined),
		[lookupMode, name, runEventState.run?.workflowName, workflow?.name],
	);

	const loadRunLedgerData = useCallback(
		async (run: PipelineRun, opts?: { append?: boolean; cursor?: number }) => {
			const cursor = opts?.append ? opts.cursor : undefined;
			setRunMetadataState((current) => ({
				...current,
				loading: true,
				error: null,
			}));

			const [events, assetNodes, costSummary, metadataResults] =
				await Promise.all([
					withTimeout(
						listRunEvents(run.id, {
							limit: 100,
							cursor,
							...runEventFilters,
						}),
						"获取运行事件",
						RUN_DETAIL_TIMEOUT_MS,
					),
					withTimeout(
						listRunAssetNodes(run.id, { limit: 500 }),
						"获取资源节点",
						RUN_DETAIL_TIMEOUT_MS,
					),
					withTimeout(
						getRunCostSummary(run.id),
						"获取成本汇总",
						RUN_DETAIL_TIMEOUT_MS,
					),
					Promise.allSettled([
						withTimeout(
							listRunInputs(run.id),
							"获取运行输入",
							RUN_DETAIL_TIMEOUT_MS,
						),
						withTimeout(
							listRunOutputs(run.id),
							"获取运行输出",
							RUN_DETAIL_TIMEOUT_MS,
						),
						withTimeout(
							getRunRuntime(run.id),
							"获取运行时信息",
							RUN_DETAIL_TIMEOUT_MS,
						),
					] as const),
				]);
			const metadataError = metadataResults
				.filter((result) => result.status === "rejected")
				.map((result) =>
					result.status === "rejected" ? toErrorMessage(result.reason) : "",
				)
				.filter(Boolean)
				.join("; ");
			const inputs =
				metadataResults[0].status === "fulfilled"
					? metadataResults[0].value
					: null;
			const outputs =
				metadataResults[1].status === "fulfilled"
					? metadataResults[1].value
					: null;
			const runtime =
				metadataResults[2].status === "fulfilled"
					? metadataResults[2].value
					: null;
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
			setRunMetadataState({
				inputs,
				outputs,
				runtime,
				loading: false,
				error: metadataError || null,
			});
			return run;
		},
		[runEventFilters],
	);

	const loadRunDetailData = useCallback(
		async (runName: string, opts?: { append?: boolean; cursor?: number }) => {
			const run = await resolvePipelineRun(runName, lookupMode);
			if (!run) {
				setRunEventState({
					run: null,
					items: [],
					loading: false,
					error: "未找到关联的 DataBrew Run",
				});
				setAssetNodeState({
					items: [],
					loading: false,
					error: "未找到关联的 DataBrew Run",
					summary: null,
				});
				setCostSummaryState({
					item: null,
					loading: false,
					error: "未找到关联的 DataBrew Run",
				});
				setRunMetadataState({
					...EMPTY_RUN_METADATA_STATE,
					error: "未找到关联的 DataBrew Run",
				});
				return null;
			}
			return loadRunLedgerData(run, opts);
		},
		[loadRunLedgerData, lookupMode],
	);

	const refreshDetailData = useCallback(() => {
		if (!name) return;
		void loadRunDetailData(name).catch((err) => {
			if (!isExpectedWorkflowNotFound(err)) {
				console.error(err);
			}
		});
	}, [loadRunDetailData, name]);

	const loadWorkflow = useCallback(() => {
		if (!name) return;
		setLoading(true);
		setLoadError(null);
		if (lookupMode === "runId") {
			skipNextRunEventsLoadRef.current = true;
			getRun(name)
				.then((run) => {
					if (!run) {
						setWorkflow(null);
						setLoadError({
							kind: "not_found",
							message: "未找到 DataBrew Run",
						});
						return;
					}

					void loadRunLedgerData(run).catch((err) => {
						if (!isExpectedWorkflowNotFound(err)) {
							console.error(err);
						}
						const message = toErrorMessage(err);
						setRunEventState((current) => ({
							...current,
							loading: false,
							error: message,
						}));
						setAssetNodeState((current) => ({
							...current,
							loading: false,
							error: message,
						}));
						setCostSummaryState((current) => ({
							...current,
							loading: false,
							error: message,
						}));
						setRunMetadataState((current) => ({
							...current,
							loading: false,
							error: message,
						}));
					});

					if (!run.workflowName) {
						setWorkflow(null);
						setLoadError({
							kind: "not_found",
							message: "Run has no runtime workflow reference",
						});
						return;
					}
					return withTimeout(
						getWorkflow(run.workflowName),
						"获取运行详情",
						WORKFLOW_FETCH_TIMEOUT_MS,
					)
						.then((detail) => {
							workflowRef.current = detail;
							setWorkflow(detail);
							setLoadError(null);
							return detail;
						})
						.catch((err) => {
							if (!isExpectedWorkflowNotFound(err)) {
								console.error(err);
							}
							setWorkflow(null);
							setLoadError(toLoadError(err));
							return null;
						});
				})
				.catch((err) => {
					if (!isExpectedWorkflowNotFound(err)) {
						console.error(err);
					}
					setWorkflow(null);
					setLoadError(toLoadError(err));
				})
				.finally(() => {
					setLoading(false);
				});
			return;
		}
		getWorkflow(name)
			.then((detail) => {
				workflowRef.current = detail;
				setWorkflow(detail);
				setLoadError(null);
				refreshDetailData();
			})
			.catch((err) => {
				if (!isExpectedWorkflowNotFound(err)) {
					console.error(err);
				}
				const nextLoadError = toLoadError(err);
				setWorkflow(null);
				setLoadError(nextLoadError);
				if (nextLoadError.kind === "not_found") {
					refreshDetailData();
				}
			})
			.finally(() => setLoading(false));
	}, [loadRunLedgerData, lookupMode, name, refreshDetailData]);

	useEffect(() => {
		loadWorkflow();
	}, [loadWorkflow]);

	const loadRunEvents = useCallback(
		(opts?: { append?: boolean; cursor?: number }) => {
			if (!name) return;
			if (skipNextRunEventsLoadRef.current) {
				skipNextRunEventsLoadRef.current = false;
				return;
			}
			setRunEventState((current) => ({
				...current,
				loading: true,
				error: null,
			}));
			loadRunDetailData(name, opts).catch((err) => {
				if (!isExpectedWorkflowNotFound(err)) {
					console.error(err);
				}
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
				setRunMetadataState((current) => ({
					...current,
					loading: false,
					error: toErrorMessage(err),
				}));
			});
		},
		[loadRunDetailData, name],
	);

	useEffect(() => {
		loadRunEvents();
	}, [loadRunEvents]);

	const setRunEventFilters = useCallback((filters: RunEventFilters) => {
		setRunEventFiltersState(filters);
		setRunEventState((current) => ({ ...current, nextCursor: undefined }));
	}, []);

	const shouldPollWorkflow =
		name != null &&
		workflow != null &&
		(ACTIVE_WORKFLOW_STATUSES.has(
			runEventState.run?.status ?? workflow.status,
		) ||
			workflowHasActiveNodesNewerThanRun(workflow, runEventState.run));
	const shouldPollLedgerOnly =
		name != null &&
		workflow == null &&
		runEventState.run != null &&
		ACTIVE_WORKFLOW_STATUSES.has(runEventState.run.status);

	useEffect(() => {
		if (!runtimeWorkflowName || !shouldPollWorkflow) {
			return;
		}

		const timer = window.setInterval(() => {
			getWorkflow(runtimeWorkflowName)
				.then((detail) => {
					workflowRef.current = detail;
					setWorkflow(detail);
					setLoadError(null);
					refreshDetailData();
				})
				.catch((err) => {
					if (!isExpectedWorkflowNotFound(err)) {
						console.error(err);
					}
				});
		}, WORKFLOW_POLL_INTERVAL_MS);

		return () => window.clearInterval(timer);
	}, [refreshDetailData, runtimeWorkflowName, shouldPollWorkflow]);

	useEffect(() => {
		if (!name || !shouldPollLedgerOnly) {
			return;
		}

		const timer = window.setInterval(() => {
			loadWorkflow();
			loadRunEvents();
		}, WORKFLOW_POLL_INTERVAL_MS);

		return () => window.clearInterval(timer);
	}, [loadRunEvents, loadWorkflow, name, shouldPollLedgerOnly]);

	const loadNodeLogs = useCallback(
		async (nodeId: string, nodePhase?: string) => {
			if (!runtimeWorkflowName) return;
			setLogState((current) => ({
				...current,
				loading: current.lines.length === 0,
				error: null,
			}));
			const shouldTryPrevious =
				nodePhase != null && /failed|error/i.test(nodePhase);
			const applyLogResponse = (
				res: Awaited<ReturnType<typeof getWorkflowLogs>>,
			) => {
				setLogState((current) => ({
					lines: res.logs ? res.logs.split("\n") : [],
					loading: false,
					error: null,
					search: current.search,
					following: current.following,
					followStatus: current.followStatus,
					followMessage: current.followMessage,
					response: res,
					clientTruncated: false,
				}));
			};
			try {
				const res = await getWorkflowLogs(runtimeWorkflowName, nodeId);
				if (!res.logs?.trim() && shouldTryPrevious) {
					try {
						const previous = await getWorkflowLogs(
							runtimeWorkflowName,
							nodeId,
							{
								previous: true,
							},
						);
						if (previous.logs?.trim()) {
							applyLogResponse({
								...previous,
								logs: `[前一容器实例]\n${previous.logs}`,
							});
							return;
						}
					} catch {
						/* fall through to primary response */
					}
				}
				applyLogResponse(res);
			} catch (err) {
				if (shouldTryPrevious) {
					try {
						const previous = await getWorkflowLogs(
							runtimeWorkflowName,
							nodeId,
							{
								previous: true,
							},
						);
						if (previous.logs?.trim()) {
							applyLogResponse({
								...previous,
								logs: `[前一容器实例]\n${previous.logs}`,
							});
							return;
						}
					} catch {
						/* use primary error below */
					}
				}
				setLogState((current) => ({
					lines: [],
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
		[runtimeWorkflowName],
	);

	const clearLogStreamFlushTimer = useCallback(() => {
		if (logStreamFlushTimerRef.current !== null) {
			window.clearTimeout(logStreamFlushTimerRef.current);
			logStreamFlushTimerRef.current = null;
		}
	}, []);

	const clearLogStreamConnectTimer = useCallback(() => {
		if (logStreamConnectTimerRef.current !== null) {
			window.clearTimeout(logStreamConnectTimerRef.current);
			logStreamConnectTimerRef.current = null;
		}
	}, []);

	const flushBufferedLogLines = useCallback(() => {
		clearLogStreamFlushTimer();
		const lines = logStreamBufferRef.current.splice(0);
		if (lines.length === 0) return;
		startTransition(() => {
			setLogState((prev) => {
				const next = appendBoundedLogLines(prev.lines, lines);
				return {
					...prev,
					lines: next.lines,
					clientTruncated: prev.clientTruncated || next.truncated,
					loading: false,
					followStatus: "connected",
					followMessage: "实时日志已连接",
				};
			});
		});
	}, [clearLogStreamFlushTimer]);

	const queueLogLine = useCallback(
		(line: string) => {
			logStreamBufferRef.current.push(line);
			if (logStreamFlushTimerRef.current === null) {
				logStreamFlushTimerRef.current = window.setTimeout(
					flushBufferedLogLines,
					LOG_STREAM_FLUSH_INTERVAL_MS,
				);
			}
		},
		[flushBufferedLogLines],
	);

	const selectNode = useCallback(
		(node: WorkflowNodeStatus | null) => {
			if (!node || !runtimeWorkflowName) {
				clearLogStreamFlushTimer();
				clearLogStreamConnectTimer();
				logStreamBufferRef.current = [];
				setSelectedNodeId(null);
				setLogState((current) => ({
					...EMPTY_LOG_STATE,
					search: current.search,
				}));
				return;
			}
			const nodeChanged = node.id !== selectedNodeId;
			if (nodeChanged) {
				clearLogStreamFlushTimer();
				clearLogStreamConnectTimer();
				logStreamBufferRef.current = [];
			}
			setSelectedNodeId(node.id);
			setLogState((current) => ({
				...current,
				lines: nodeChanged ? [] : current.lines,
				loading: nodeChanged ? false : current.loading,
				error: null,
				response: nodeChanged ? null : current.response,
				clientTruncated: nodeChanged ? false : current.clientTruncated,
			}));
		},
		[
			clearLogStreamConnectTimer,
			clearLogStreamFlushTimer,
			runtimeWorkflowName,
			selectedNodeId,
		],
	);

	const selectedNode = useMemo(() => {
		if (!workflow || !selectedNodeId) return null;
		return workflow.nodes.find((node) => node.id === selectedNodeId) ?? null;
	}, [workflow, selectedNodeId]);

	const stopFollowLogs = useCallback(() => {
		flushBufferedLogLines();
		clearLogStreamConnectTimer();
		if (reconnectTimerRef.current) {
			window.clearTimeout(reconnectTimerRef.current);
			reconnectTimerRef.current = null;
		}
		reconnectAttemptRef.current = 0;
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
	}, [clearLogStreamConnectTimer, flushBufferedLogLines]);

	const startFollowLogs = useCallback(() => {
		if (!runtimeWorkflowName || !selectedNodeId) return;

		stopFollowLogs();
		clearLogStreamFlushTimer();
		clearLogStreamConnectTimer();
		logStreamBufferRef.current = [];

		const params: Record<string, string> = {};
		if (lastEventIdRef.current) {
			params.lastEventId = lastEventIdRef.current;
		}
		const url = getWorkflowLogStreamUrl(runtimeWorkflowName, selectedNodeId, params);
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
			clearLogStreamConnectTimer();
			reconnectAttemptRef.current = 0;
			setLogState((prev) => ({
				...prev,
				following: true,
				followStatus: "connected",
				followMessage: "实时日志已连接",
			}));
		};

		logStreamConnectTimerRef.current = window.setTimeout(() => {
			logStreamConnectTimerRef.current = null;
			if (followSourceRef.current !== source) return;
			setLogState((prev) =>
				prev.followStatus === "connecting"
					? {
							...prev,
							followStatus: "connected",
							followMessage: "实时日志已连接，等待日志事件",
						}
					: prev,
			);
		}, LOG_STREAM_CONNECT_GRACE_MS);

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
					queueLogLine(line);
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
			flushBufferedLogLines();
			clearLogStreamConnectTimer();
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
				flushBufferedLogLines();
				clearLogStreamConnectTimer();
				source.close();
				if (followSourceRef.current === source) {
					followSourceRef.current = null;
				}
				if (reconnectAttemptRef.current < MAX_RECONNECT_ATTEMPTS) {
					const delay = calculateBackoff(reconnectAttemptRef.current);
					reconnectAttemptRef.current += 1;
					setLogState((prev) => ({
						...prev,
						following: true,
						followStatus: "reconnecting",
						followMessage: `重连中... (${reconnectAttemptRef.current}/${MAX_RECONNECT_ATTEMPTS})`,
					}));
					reconnectTimerRef.current = window.setTimeout(() => {
						reconnectTimerRef.current = null;
						startFollowLogs();
					}, delay);
				} else {
					setLogState((prev) => ({
						...prev,
						following: false,
						followStatus: "error",
						followMessage: "实时日志重连失败，请手动连接",
					}));
				}
			};
	}, [
		clearLogStreamConnectTimer,
		clearLogStreamFlushTimer,
		flushBufferedLogLines,
		queueLogLine,
		runtimeWorkflowName,
		selectedNodeId,
		stopFollowLogs,
	]);

	const downloadLogs = useCallback(() => {
		const content = logState.lines.join("\n");
		const selNode = selectedNode;
		if (!logState.lines.length || !runtimeWorkflowName || !selNode) return;
		const nodeName = selNode.displayName || selNode.name || selNode.id;
		const response = logState.response;
		const header = [
			`# workflow: ${runtimeWorkflowName}`,
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
		a.download = `${runtimeWorkflowName}-${nodeName}.log`;
		a.click();
		URL.revokeObjectURL(url);
	}, [logState.lines, logState.response, runtimeWorkflowName, selectedNode]);

	useEffect(() => {
		return () => {
			clearLogStreamFlushTimer();
			clearLogStreamConnectTimer();
			logStreamBufferRef.current = [];
			if (followSourceRef.current) {
				followSourceRef.current.close();
				followSourceRef.current = null;
			}
		};
	}, [clearLogStreamConnectTimer, clearLogStreamFlushTimer]);

	useEffect(() => {
		if (!selectedNodeId || !runtimeWorkflowName) {
			return;
		}
		// When log streaming is active, skip poll-triggered log fetches.
		// The SSE stream provides real-time content; replacing it with an
		// HTTP snapshot causes visible flicker/refresh.
		if (followSourceRef.current) {
			return;
		}
		const phase = workflow?.nodes.find(
			(node) => node.id === selectedNodeId,
		)?.phase;
		void loadNodeLogs(selectedNodeId, phase);
	}, [loadNodeLogs, runtimeWorkflowName, selectedNodeId, workflow?.nodes]);

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
		runMetadataState: { ...runMetadataState },
		setLogSearch: useCallback((query: string) => {
			setLogState((current) => ({ ...current, search: query }));
		}, []),
	};
}
