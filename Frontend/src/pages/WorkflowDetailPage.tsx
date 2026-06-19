import {
	ApartmentOutlined,
	ArrowLeftOutlined,
	BarsOutlined,
	CopyOutlined,
	ReloadOutlined,
} from "@ant-design/icons";
import Ansi from "ansi-to-react";
import {
	Alert,
	App,
	Button,
	Card,
	Descriptions,
	Input,
	Modal,
	Segmented,
	Select,
	Space,
	Spin,
	Table,
	Tag,
	Tooltip,
	Typography,
} from "antd";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useLocation, useNavigate, useParams } from "react-router-dom";
import type {
	PipelineRun,
	PipelineRunAssetNode,
	PipelineRunEvent,
	PipelineRunNode,
} from "../api/pipelineApi";
import {
	deleteRun,
	resubmitRun,
	resumeRun,
	retryRun,
	stopRun,
	suspendRun,
	terminateRun,
} from "../api/runApi";
import type {
	WorkflowDetail,
	WorkflowLogResponse,
	WorkflowNodeStatus,
} from "../api/workflowApi";
import { DurationPanel } from "../components/common/DurationPanel";
import { LinkifiedText } from "../components/common/LinkifiedText";
import type { Pipeline, PipelineNodeDef } from "../components/pipeline/types";
import {
	WorkflowNodeDetailPanel,
	type WorkflowNodeDetailTabKey,
} from "../components/pipeline/WorkflowNodeDetailPanel";
import { isCanonicalAssetId } from "../lib/assetId";
import { STATUS_COLORS } from "../lib/constants";
import { resolveWorkflowDetailBackTarget } from "../lib/pipelineNavigation";
import { formatWorkflowPhaseLabel } from "../lib/statusLabels";
import {
	getAvailableWorkflowOperationConfigs,
	getWorkflowOperationConfigs,
	getWorkflowOperationConfirmText,
	runWorkflowRetryWithFeedback,
	type WorkflowOperationConfig,
	type WorkflowOperationKey,
} from "../lib/workflow-operations";
import { buildPipelineNodeLabelLookup } from "../lib/workflowNodeDisplay";
import {
	useWorkflowDetail,
	type WorkflowLogFollowStatus,
} from "./useWorkflowDetail";
import type { WorkflowDagNodeAction } from "./WorkflowDagNode";
import {
	countDisplayableWorkflowNodes,
	WorkflowDagView,
} from "./WorkflowDagView";
import { WorkflowTimelineView } from "./WorkflowTimelineView";
import {
	LOG_MAX_RENDER_LINES,
	prepareVisibleLogContent,
} from "./workflowLogView";
import "../styles/pipeline.css";

function buildHighlightedLogNodes(logContent: string, keyword: string) {
	const normalized = keyword.trim();
	if (!normalized) {
		return [<Ansi key="raw-log">{logContent}</Ansi>];
	}

	const escaped = normalized.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
	const pattern = new RegExp(`(${escaped})`, "gi");
	const parts = logContent.split(pattern);
	let offset = 0;
	let segmentIndex = 0;
	return parts.flatMap((part) => {
		const key = `log-chunk-${offset}-${offset + part.length}`;
		offset += part.length;
		const isMatch = segmentIndex % 2 === 1;
		segmentIndex += 1;
		if (!part) return [];
		if (isMatch) {
			return (
				<mark
					key={key}
					style={{ background: "#fef08a", padding: 0, borderRadius: 0 }}
				>
					<Ansi>{part}</Ansi>
				</mark>
			);
		}
		return <Ansi key={key}>{part}</Ansi>;
	});
}

const LOG_VIRTUAL_ROW_HEIGHT = 18;
const LOG_VIRTUAL_OVERSCAN_ROWS = 12;

function VirtualLogContent({
	content,
	search,
	scrollTop,
	viewportHeight,
}: {
	content: string;
	search: string;
	scrollTop: number;
	viewportHeight: number;
}) {
	const lines = useMemo(() => {
		const raw = content.endsWith("\n") ? content.slice(0, -1) : content;
		return raw === "" ? [] : raw.split("\n");
	}, [content]);
	const rowCount = lines.length;
	const visibleCount = Math.max(
		1,
		Math.ceil(viewportHeight / LOG_VIRTUAL_ROW_HEIGHT) +
			LOG_VIRTUAL_OVERSCAN_ROWS * 2,
	);
	const start = Math.max(
		0,
		Math.floor(scrollTop / LOG_VIRTUAL_ROW_HEIGHT) - LOG_VIRTUAL_OVERSCAN_ROWS,
	);
	const end = Math.min(rowCount, start + visibleCount);
	const visibleRows = [];
	for (let absoluteIndex = start; absoluteIndex < end; absoluteIndex += 1) {
		const line = lines[absoluteIndex] ?? "";
		visibleRows.push({
			key: `${absoluteIndex}-${line.slice(0, 24)}`,
			line,
		});
	}

	return (
		<div
			style={{
				height: rowCount * LOG_VIRTUAL_ROW_HEIGHT,
				minHeight: "100%",
				position: "relative",
			}}
		>
			<div
				style={{
					position: "absolute",
					top: start * LOG_VIRTUAL_ROW_HEIGHT,
					left: 0,
					right: 0,
				}}
			>
				{visibleRows.map((row) => (
					<div
						key={row.key}
						style={{
							height: LOG_VIRTUAL_ROW_HEIGHT,
							lineHeight: `${LOG_VIRTUAL_ROW_HEIGHT}px`,
							whiteSpace: "pre",
						}}
					>
						{row.line ? buildHighlightedLogNodes(row.line, search) : "\u00a0"}
					</div>
				))}
			</div>
		</div>
	);
}

const ACTIVE_NODE_PHASES = new Set(["Running", "Pending"]);
const TERMINAL_NODE_PHASES = new Set([
	"Succeeded",
	"Failed",
	"Error",
	"Skipped",
	"Omitted",
]);

function toTime(value?: string): number | null {
	if (!value) return null;
	const time = new Date(value).getTime();
	return Number.isNaN(time) ? null : time;
}

function addNodeLookupKey(
	map: Map<string, PipelineRunNode>,
	key: unknown,
	node: PipelineRunNode,
) {
	if (typeof key !== "string") return;
	const normalized = key.trim();
	if (!normalized || map.has(normalized)) return;
	map.set(normalized, node);
}

function buildRunNodeLookup(runNodes: PipelineRunNode[] = []) {
	const lookup = new Map<string, PipelineRunNode>();
	for (const node of runNodes) {
		addNodeLookupKey(lookup, node.argoNodeId, node);
		addNodeLookupKey(lookup, node.pipelineNodeId, node);
		addNodeLookupKey(lookup, node.argoNodeName, node);
		addNodeLookupKey(lookup, node.displayName, node);
		addNodeLookupKey(lookup, node.templateName, node);
	}
	return lookup;
}

function addPipelineNodeLookupKey(
	map: Map<string, PipelineNodeDef>,
	key: unknown,
	node: PipelineNodeDef,
) {
	if (typeof key !== "string") return;
	const normalized = key.trim();
	if (!normalized || map.has(normalized)) return;
	map.set(normalized, node);
	if (normalized.startsWith("step-")) {
		const stripped = normalized.replace(/^step-/, "");
		addPipelineNodeLookupKey(map, stripped, node);
	} else {
		addPipelineNodeLookupKey(map, `step-${normalized}`, node);
	}
}

function buildPipelineNodeLookup(pipeline?: Pipeline | null) {
	const lookup = new Map<string, PipelineNodeDef>();
	for (const node of pipeline?.nodes ?? []) {
		addPipelineNodeLookupKey(lookup, node.id, node);
		addPipelineNodeLookupKey(lookup, node.component?.name, node);
	}
	return lookup;
}

function workflowNodePipelineLookupKeys(node: WorkflowNodeStatus): string[] {
	const keys = [
		node.id,
		node.name,
		node.name?.split(".").pop(),
		node.displayName,
		node.templateName,
	];
	const out: string[] = [];
	const seen = new Set<string>();
	for (const rawKey of keys) {
		const key = rawKey?.trim();
		if (!key || seen.has(key)) continue;
		seen.add(key);
		out.push(key);
	}
	return out;
}

function findRunNodeSnapshot(
	lookup: Map<string, PipelineRunNode>,
	node: WorkflowNodeStatus,
): PipelineRunNode | null {
	return (
		lookup.get(node.id) ??
		lookup.get(node.name) ??
		lookup.get(node.displayName) ??
		(node.templateName ? lookup.get(node.templateName) : undefined) ??
		null
	);
}

export function findPipelineNodeForWorkflowNode(
	pipeline: Pipeline | null | undefined,
	run: PipelineRun | null | undefined,
	node: WorkflowNodeStatus | null | undefined,
): PipelineNodeDef | null {
	if (!pipeline || !node) return null;
	const pipelineNodeLookup = buildPipelineNodeLookup(pipeline);
	const runNodeLookup = buildRunNodeLookup(run?.nodes);
	const runNode = findRunNodeSnapshot(runNodeLookup, node);
	if (runNode?.pipelineNodeId) {
		const pipelineNode = pipelineNodeLookup.get(runNode.pipelineNodeId);
		if (pipelineNode) return pipelineNode;
	}
	for (const key of workflowNodePipelineLookupKeys(node)) {
		const pipelineNode = pipelineNodeLookup.get(key);
		if (pipelineNode) return pipelineNode;
	}
	return null;
}

function getWorkflowNodeSnapshotTime(
	workflow: WorkflowDetail,
	node: WorkflowNodeStatus,
): number | null {
	return (
		toTime(node.finishedAt) ??
		toTime(node.startedAt) ??
		toTime(workflow.finishedAt) ??
		toTime(workflow.createdAt)
	);
}

function getRunNodeSnapshotTime(
	run: PipelineRun,
	node: PipelineRunNode,
): number | null {
	return (
		toTime(node.finishedAt) ??
		toTime(node.updatedAt) ??
		toTime(node.startedAt) ??
		toTime(run.finishedAt) ??
		toTime(run.updatedAt) ??
		toTime(run.createdAt)
	);
}

function shouldUseRunNodeSnapshot(
	workflow: WorkflowDetail,
	workflowNode: WorkflowNodeStatus,
	run: PipelineRun,
	runNode: PipelineRunNode,
): boolean {
	if (!runNode.phase) return false;
	const workflowActive = ACTIVE_NODE_PHASES.has(workflowNode.phase);
	const runNodeTerminal = TERMINAL_NODE_PHASES.has(runNode.phase);
	const runTerminal = TERMINAL_NODE_PHASES.has(run.status);
	const workflowNodeAt = getWorkflowNodeSnapshotTime(workflow, workflowNode);
	const runNodeAt = getRunNodeSnapshotTime(run, runNode);

	if (workflowActive && runNodeTerminal && workflowNodeAt && runNodeAt) {
		return runNodeAt >= workflowNodeAt;
	}
	if (runNodeTerminal || runTerminal) return true;
	if (workflowNodeAt && runNodeAt) return runNodeAt >= workflowNodeAt;
	return runNode.phase !== workflowNode.phase || Boolean(runNode.message);
}

function isWorkflowControlNode(
	workflow: WorkflowDetail,
	node: WorkflowNodeStatus,
) {
	const type = (node.type || "").toLowerCase();
	return (
		type === "dag" ||
		type === "steps" ||
		type === "stepgroup" ||
		node.id === workflow.name ||
		node.name === workflow.name ||
		node.displayName === workflow.name
	);
}

function shouldUseRunTerminalForControlNode(
	workflow: WorkflowDetail,
	node: WorkflowNodeStatus,
	run: PipelineRun,
) {
	if (!TERMINAL_NODE_PHASES.has(run.status)) return false;
	if (!ACTIVE_NODE_PHASES.has(node.phase)) return false;
	if (!isWorkflowControlNode(workflow, node)) return false;
	const runAt = toTime(run.finishedAt) ?? toTime(run.updatedAt);
	const nodeAt = getWorkflowNodeSnapshotTime(workflow, node);
	if (runAt && nodeAt && nodeAt > runAt) return false;
	return true;
}

function shouldUseRunTerminalFallbackForUnmappedNode(
	workflow: WorkflowDetail,
	node: WorkflowNodeStatus,
	run: PipelineRun,
) {
	if (!TERMINAL_NODE_PHASES.has(run.status)) return false;
	if (!ACTIVE_NODE_PHASES.has(node.phase)) return false;
	const runAt = toTime(run.finishedAt) ?? toTime(run.updatedAt);
	const nodeAt = getWorkflowNodeSnapshotTime(workflow, node);
	if (runAt && nodeAt && nodeAt > runAt) return false;
	return true;
}

function mergeRunNodeSnapshot(
	node: WorkflowNodeStatus,
	runNode: PipelineRunNode,
): WorkflowNodeStatus {
	return {
		...node,
		phase: runNode.phase || node.phase,
		message: runNode.message || node.message,
		podName: runNode.podName || node.podName,
		hostNodeName: runNode.hostNodeName || node.hostNodeName,
		children: runNode.children?.length ? runNode.children : node.children,
		startedAt: runNode.startedAt || node.startedAt,
		finishedAt: runNode.finishedAt || node.finishedAt,
		estimatedCostUsd: runNode.estimatedCostUsd ?? node.estimatedCostUsd,
		inputs: (runNode.inputs as WorkflowNodeStatus["inputs"]) ?? node.inputs,
		outputs: (runNode.outputs as WorkflowNodeStatus["outputs"]) ?? node.outputs,
		resourcesDuration: runNode.resourcesDuration ?? node.resourcesDuration,
	};
}

function buildDisplayWorkflowNodes(
	workflow: WorkflowDetail,
	run?: PipelineRun | null,
): WorkflowNodeStatus[] {
	if (!run) return workflow.nodes;
	const runNodeLookup = buildRunNodeLookup(run.nodes);
	return workflow.nodes.map((node) => {
		const runNode = findRunNodeSnapshot(runNodeLookup, node);
		if (runNode && shouldUseRunNodeSnapshot(workflow, node, run, runNode)) {
			return mergeRunNodeSnapshot(node, runNode);
		}
		if (shouldUseRunTerminalForControlNode(workflow, node, run)) {
			return {
				...node,
				phase: run.status,
				message: run.message || node.message,
				finishedAt: run.finishedAt || node.finishedAt,
			};
		}
		if (shouldUseRunTerminalFallbackForUnmappedNode(workflow, node, run)) {
			return {
				...node,
				phase: run.status,
				message: run.message || node.message,
				finishedAt: run.finishedAt || node.finishedAt,
			};
		}
		return node;
	});
}

function WorkflowLogPanel({
	selectedNode,
	loading,
	logContent,
	error,
	search,
	following,
	followStatus,
	followMessage,
	logResponse,
	clientTruncated,
	onSearch,
	onFollow,
	onStop,
	onDownload,
}: {
	selectedNode: WorkflowNodeStatus | null;
	loading: boolean;
	logContent: string | null;
	error: string | null;
	search: string;
	following: boolean;
	followStatus: WorkflowLogFollowStatus;
	followMessage: string | null;
	logResponse: WorkflowLogResponse | null;
	clientTruncated: boolean;
	onSearch: (value: string) => void;
	onFollow: () => void;
	onStop: () => void;
	onDownload: () => void;
}) {
	const { message: messageApi } = App.useApp();
	const logBodyRef = useRef<HTMLDivElement | null>(null);
	const [logViewport, setLogViewport] = useState({
		scrollTop: 0,
		viewportHeight: 360,
	});
	const visibleLog = useMemo(
		() =>
			logContent === null
				? null
				: prepareVisibleLogContent(logContent, selectedNode),
		[logContent, selectedNode],
	);
	const followStatusMeta: Record<
		WorkflowLogFollowStatus,
		{ color: string; label: string }
	> = {
		idle: { color: "default", label: "未连接" },
		connecting: { color: "processing", label: "连接中" },
		connected: { color: "green", label: "实时中" },
		ended: { color: "blue", label: "已结束" },
		error: { color: "red", label: "已断开" },
	};
	const followMeta = followStatusMeta[followStatus];
	const paginationUnavailable =
		logResponse?.pagination && logResponse.pagination.available === false;
	const syncLogViewport = useCallback(() => {
		const el = logBodyRef.current;
		if (!el) return;
		setLogViewport({
			scrollTop: el.scrollTop,
			viewportHeight: el.clientHeight || 360,
		});
	}, []);

	useEffect(() => {
		if (
			selectedNode &&
			!loading &&
			!error &&
			visibleLog !== null &&
			logBodyRef.current
		) {
			logBodyRef.current.scrollTop = logBodyRef.current.scrollHeight;
			syncLogViewport();
		}
	}, [selectedNode, loading, error, visibleLog, syncLogViewport]);

	useEffect(() => {
		syncLogViewport();
	}, [syncLogViewport]);

	return (
		<div
			style={{
				height: "100%",
				display: "flex",
				flexDirection: "column",
				padding: 16,
			}}
		>
			<div
				style={{
					display: "flex",
					gap: 8,
					alignItems: "center",
					marginBottom: 12,
				}}
			>
				<Input.Search
					id="workflow-log-search"
					name="workflow-log-search"
					placeholder="日志关键字搜索"
					value={search}
					onChange={(event) => onSearch(event.target.value)}
					allowClear
					style={{ flex: 1 }}
				/>
				{following ? (
					<Button type="primary" disabled={!selectedNode} onClick={onStop}>
						停止实时日志
					</Button>
				) : (
					<Button
						disabled={!selectedNode}
						loading={followStatus === "connecting"}
						onClick={onFollow}
					>
						实时日志
					</Button>
				)}
				<Button disabled={!selectedNode || !logContent} onClick={onDownload}>
					下载当前窗口
				</Button>
			</div>
			<div
				style={{
					display: "flex",
					flexWrap: "wrap",
					gap: 8,
					alignItems: "center",
					marginBottom: 8,
				}}
			>
				<Tag color={followMeta.color}>实时状态：{followMeta.label}</Tag>
				{logResponse ? (
					<>
						<Tag color="blue">
							tail {logResponse.truncation.tailLines.toLocaleString()} 行
						</Tag>
						<Tag color="purple">
							上限 {logResponse.truncation.limitBytes.toLocaleString()} bytes
						</Tag>
						<Tag color="default">
							来源 {logResponse.source} /{" "}
							{logResponse.window?.scope ?? "bounded"}
						</Tag>
					</>
				) : null}
				{followMessage ? (
					<Typography.Text type="secondary" style={{ fontSize: 12 }}>
						{followMessage}
					</Typography.Text>
				) : null}
			</div>

			{!selectedNode ? (
				<div style={{ color: "#9ca3af", fontSize: 13 }}>
					点击 DAG 或时间线节点查看该节点日志
				</div>
			) : loading ? (
				<div style={{ textAlign: "center", padding: 40 }}>
					<Spin />
				</div>
			) : error ? (
				<div
					style={{ color: "#dc2626", fontSize: 13 }}
				>{`获取日志失败：${error}`}</div>
			) : logContent === null ? (
				<div style={{ color: "#9ca3af", fontSize: 13 }}>暂无日志</div>
			) : (
				<>
					{paginationUnavailable && (
						<Alert
							type="info"
							showIcon
							style={{ marginBottom: 8 }}
							message="当前日志源不支持稳定历史分页"
							description={
								logResponse?.pagination?.reason ||
								"实时 Argo 日志只提供 tail/since/follow 窗口；完整历史归档需要后续接入持久化日志。"
							}
						/>
					)}
					{visibleLog?.truncated && (
						<Alert
							type="warning"
							showIcon
							style={{ marginBottom: 8 }}
							message={`日志较大，当前仅显示尾部 ${visibleLog.content.length.toLocaleString()} 字符 / ${Math.min(visibleLog.totalLines, LOG_MAX_RENDER_LINES).toLocaleString()} 行。`}
							description="完整大日志需要后端 tail、分页或流式接口支持；当前视图会限制渲染量以避免浏览器卡顿。"
						/>
					)}
					{clientTruncated && (
						<Alert
							type="warning"
							showIcon
							style={{ marginBottom: 8 }}
							message="实时日志已限制浏览器内存缓冲"
							description="为了避免页面卡顿，前端仅保留最近一段实时日志。需要完整日志时请下载当前窗口或使用后续归档能力。"
						/>
					)}
					<div
						style={{
							display: "flex",
							alignItems: "center",
							justifyContent: "space-between",
							gap: 12,
							marginBottom: 8,
							fontSize: 12,
							color: "#64748b",
						}}
					>
						<span>
							显示 {visibleLog?.content.length.toLocaleString()} 字符 /{" "}
							{visibleLog?.totalLines.toLocaleString()} 行
							{logResponse?.truncated ? "，服务端已按字节上限截断" : ""}
							{search.trim() ? `，搜索：${search.trim()}` : ""}
						</span>
						<Button
							size="small"
							icon={<CopyOutlined />}
							onClick={async () => {
								if (!visibleLog) return;
								await navigator.clipboard.writeText(visibleLog.content);
								messageApi.success?.("已复制当前可见日志");
							}}
						>
							复制可见日志
						</Button>
					</div>
					<div
						ref={logBodyRef}
						onScroll={syncLogViewport}
						style={{
							flex: 1,
							fontSize: 11,
							fontFamily: '"SF Mono", "Fira Code", monospace',
							overflow: "auto",
							background: "#f8f9fa",
							padding: 12,
							borderRadius: 6,
							border: "1px solid #e5e7eb",
							minHeight: 0,
							lineHeight: 1.55,
						}}
					>
						{visibleLog ? (
							<VirtualLogContent
								content={visibleLog.content}
								search={search}
								scrollTop={logViewport.scrollTop}
								viewportHeight={logViewport.viewportHeight}
							/>
						) : null}
					</div>
				</>
			)}
			{selectedNode && (
				<div style={{ marginTop: 12 }}>
					<Descriptions column={1} size="small" colon={false}>
						<Descriptions.Item label="节点ID">
							{selectedNode.id}
						</Descriptions.Item>
						<Descriptions.Item label="类型">
							{selectedNode.type || "-"}
						</Descriptions.Item>
						<Descriptions.Item label="状态">
							<Tag color={STATUS_COLORS[selectedNode.phase] || "default"}>
								{selectedNode.phase}
							</Tag>
						</Descriptions.Item>
						{selectedNode.message && (
							<Descriptions.Item label="消息">
								<LinkifiedText text={selectedNode.message} />
							</Descriptions.Item>
						)}
					</Descriptions>
				</div>
			)}
			{!selectedNode && <div style={{ marginTop: "auto" }} />}
		</div>
	);
}

function getWorkflowLabel(
	labels: Record<string, string> | undefined,
	key: string,
) {
	return (
		labels?.[key] ||
		labels?.[`cyberorigin.ai/${key}`] ||
		labels?.[`databrew/${key}`]
	);
}

const RUN_EVENT_LABELS: Record<string, string> = {
	run_submitted: "提交",
	run_scheduled: "已进入调度",
	workflow_created: "Workflow 已创建",
	workflow_observed: "发现 Workflow",
	workflow_phase_changed: "Workflow 状态",
	node_started: "节点开始",
	node_succeeded: "节点成功",
	node_failed: "节点失败",
	node_error: "节点错误",
	pod_created: "Pod 创建",
	pod_phase_changed: "Pod 状态",
	run_completed: "运行完成",
	run_failed: "运行失败",
	run_retry_requested: "请求重试",
	run_resubmitted: "重新提交",
	run_stop_requested: "请求停止",
	run_delete_requested: "请求删除",
	run_deleted: "删除完成",
	run_delete_failed: "删除失败",
};

function formatEventTime(value: string) {
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) {
		return value || "-";
	}
	return date.toLocaleString();
}

function eventTagColor(event: PipelineRunEvent) {
	const status = event.status || event.eventType;
	if (/failed|error/i.test(status)) return "red";
	if (/succeeded|completed/i.test(status)) return "green";
	if (/running|started|submitted|created/i.test(status)) return "blue";
	if (/stop|delete|retry|resubmit/i.test(status)) return "orange";
	return "default";
}

function formatRunEventError(error: string) {
	if (error.includes("未找到关联的 DataBrew Run")) {
		return "暂无 DataBrew 运行事件。该工作流可能由 Argo 外部提交，仍可查看 DAG、Pod 和日志。";
	}
	return error;
}

function shortEventSubject(event: PipelineRunEvent) {
	if (event.subjectType === "run") return "运行";
	if (event.subjectType === "workflow") return "Workflow";
	if (event.subjectType === "node") return "节点";
	if (event.subjectType === "pod") return "Pod";
	return event.subjectType || "对象";
}

function WorkflowSummaryCards({
	workflow,
	runEventState,
	costSummaryState,
}: {
	workflow: NonNullable<ReturnType<typeof useWorkflowDetail>["workflow"]>;
	runEventState: ReturnType<typeof useWorkflowDetail>["runEventState"];
	costSummaryState: ReturnType<typeof useWorkflowDetail>["costSummaryState"];
}) {
	const navigate = useNavigate();
	const templateName = workflow.labels
		? getWorkflowLabel(workflow.labels, "template-name") ||
			getWorkflowLabel(workflow.labels, "pipeline-name")
		: undefined;
	const templateVersion = workflow.labels
		? Number(getWorkflowLabel(workflow.labels, "template-version"))
		: undefined;
	const assetIds = workflow.labels
		? getWorkflowLabel(workflow.labels, "asset-ids") ||
			getWorkflowLabel(workflow.labels, "asset_ids")
		: undefined;
	const assetIdList = assetIds
		? assetIds
				.split(",")
				.map((item) => item.trim())
				.filter(Boolean)
		: [];
	const run = runEventState.run;
	const resolvedTemplateName = templateName || run?.pipelineName || undefined;
	const resolvedTemplateVersion =
		templateVersion && Number.isFinite(templateVersion)
			? templateVersion
			: run?.templateVersion;
	const resolvedAssetIdList =
		assetIdList.length > 0
			? assetIdList
			: (run?.assetIds?.filter(Boolean) ?? []);
	const resolvedAssetCount = run?.noAssetRun
		? 0
		: resolvedAssetIdList.length || run?.assetCount || 0;
	const visibleResolvedAssetIds = resolvedAssetIdList.slice(0, 4);
	const hiddenResolvedAssetCount = Math.max(
		resolvedAssetIdList.length - visibleResolvedAssetIds.length,
		0,
	);

	const cards = [
		{
			label: "模板",
			value: resolvedTemplateName ? (
				run?.templateId ? (
					<Button
						type="link"
						size="small"
						style={{ padding: 0, height: "auto" }}
						onClick={() => {
							const params = new URLSearchParams({
								templateId: run?.templateId ?? "",
								tab: "design",
							});
							if (run?.scope === "prod") {
								params.set("readonly", "1");
							}
							navigate(`/pipeline?${params.toString()}`);
						}}
					>
						{resolvedTemplateName}
						{resolvedTemplateVersion ? ` v${resolvedTemplateVersion}` : ""}
					</Button>
				) : (
					<>
						{resolvedTemplateName}
						{resolvedTemplateVersion ? ` v${resolvedTemplateVersion}` : ""}
					</>
				)
			) : (
				"—"
			),
			extra: undefined,
		},
		{
			label: "资产",
			value:
				run?.noAssetRun && resolvedAssetCount === 0 ? (
					"无资产运行"
				) : resolvedAssetCount > 0 ? (
					<div style={{ display: "flex", gap: 4, flexWrap: "wrap" }}>
						{visibleResolvedAssetIds.map((assetId) => {
							const tag = (
								<Tag
									key={assetId}
									color="blue"
									className="workflow-summary-asset-tag"
									style={{ marginInlineEnd: 0 }}
									title={assetId}
								>
									{assetId}
								</Tag>
							);
							return isCanonicalAssetId(assetId) ? (
								<Link
									key={assetId}
									to={`/assets/${encodeURIComponent(assetId)}`}
								>
									{tag}
								</Link>
							) : (
								<span key={assetId}>{tag}</span>
							);
						})}
						{hiddenResolvedAssetCount > 0 ? (
							<Tag>+{hiddenResolvedAssetCount}</Tag>
						) : null}
					</div>
				) : (
					"无资产"
				),
		},
		{
			label: "节点",
			value: countDisplayableWorkflowNodes(workflow.nodes),
		},
		{
			label: "耗时",
			value: (
				<DurationPanel
					phase={workflow.status}
					startedAt={workflow.createdAt}
					finishedAt={workflow.finishedAt}
				/>
			),
		},
		{
			label: "事件",
			value: runEventState.items.length,
		},
		{
			label: "成本",
			value:
				costSummaryState.item?.totalEstimatedCostUsd != null
					? `~$${costSummaryState.item.totalEstimatedCostUsd?.toFixed(4)}`
					: "—",
		},
	];

	return (
		<div className="workflow-summary-cards">
			{cards.map((card) => (
				<div key={card.label} className="workflow-summary-card">
					<div className="workflow-summary-card__label">{card.label}</div>
					<div className="workflow-summary-card__value">
						{card.value}
						{card.extra ? (
							<Tag color="blue" style={{ marginInlineStart: 4, fontSize: 10 }}>
								{card.extra}
							</Tag>
						) : null}
					</div>
				</div>
			))}
		</div>
	);
}

function WorkflowRunContextPanel({
	runEventState,
	runEventFilters,
	onFilterEvents,
	onRefreshEvents,
	onLoadMoreEvents,
	onSelectNodeEvent,
}: {
	runEventState: ReturnType<typeof useWorkflowDetail>["runEventState"];
	runEventFilters: ReturnType<typeof useWorkflowDetail>["runEventFilters"];
	onFilterEvents: (
		filters: ReturnType<typeof useWorkflowDetail>["runEventFilters"],
	) => void;
	onRefreshEvents: () => void;
	onLoadMoreEvents: () => void;
	onSelectNodeEvent: (event: PipelineRunEvent) => void;
}) {
	const [searchDraft, setSearchDraft] = useState(runEventFilters.q ?? "");
	const latestEvents = runEventState.items.slice(-5);

	useEffect(() => {
		setSearchDraft(runEventFilters.q ?? "");
	}, [runEventFilters.q]);

	return (
		<div
			style={{
				margin: "0 24px 12px",
				border: "1px solid #e5e7eb",
				borderRadius: 8,
				background: "#fff",
				overflow: "hidden",
				padding: 10,
			}}
		>
			<div
				style={{
					display: "flex",
					alignItems: "center",
					gap: 8,
					marginBottom: 8,
					flexWrap: "wrap",
				}}
			>
				<Typography.Text strong style={{ fontSize: 13 }}>
					事件时间线
				</Typography.Text>
				<Tag>{runEventState.items.length}</Tag>
				<Select
					size="small"
					placeholder="事件类型"
					allowClear
					style={{ width: 130 }}
					value={runEventFilters.eventType}
					onChange={(val) =>
						onFilterEvents({ ...runEventFilters, eventType: val })
					}
					options={Object.entries(RUN_EVENT_LABELS).map(([key, label]) => ({
						label,
						value: key,
					}))}
				/>
				<Select
					size="small"
					placeholder="状态"
					allowClear
					style={{ width: 100 }}
					value={runEventFilters.status}
					onChange={(val) =>
						onFilterEvents({ ...runEventFilters, status: val })
					}
					options={["Succeeded", "Failed", "Error", "Running"].map((s) => ({
						label: formatWorkflowPhaseLabel(s),
						value: s,
					}))}
				/>
				<Input.Search
					size="small"
					placeholder="搜索消息/ID"
					allowClear
					style={{ width: 180 }}
					value={searchDraft}
					onChange={(event) => setSearchDraft(event.target.value)}
					onSearch={(value) =>
						onFilterEvents({ ...runEventFilters, q: value || undefined })
					}
					onClear={() => onFilterEvents({ ...runEventFilters, q: undefined })}
				/>
				<Button
					size="small"
					icon={<ReloadOutlined />}
					loading={runEventState.loading}
					onClick={onRefreshEvents}
				>
					刷新
				</Button>
			</div>
			{runEventState.error ? (
				<Alert
					type="info"
					showIcon
					message="事件暂不可用"
					description={`这是历史工作流或外部提交的工作流，暂时没有 DataBrew 运行事件。${formatRunEventError(runEventState.error)}`}
					style={{ marginBottom: 8 }}
				/>
			) : null}
			{runEventState.loading &&
			latestEvents.length === 0 &&
			!runEventState.error ? (
				<Spin size="small" />
			) : latestEvents.length === 0 && !runEventState.error ? (
				<Typography.Text type="secondary">暂无运行事件</Typography.Text>
			) : (
				<div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
					{latestEvents.map((event) => (
						<button
							key={event.id}
							type="button"
							onClick={() => onSelectNodeEvent(event)}
							disabled={event.subjectType !== "node"}
							className={
								event.subjectType === "node"
									? "workflow-run-event"
									: "workflow-run-event workflow-run-event--static"
							}
						>
							<div style={{ display: "flex", alignItems: "center", gap: 6 }}>
								<Tag color={eventTagColor(event)} style={{ fontSize: 10 }}>
									{RUN_EVENT_LABELS[event.eventType] || event.eventType}
								</Tag>
								<Typography.Text strong style={{ fontSize: 12 }}>
									{shortEventSubject(event)}
								</Typography.Text>
								<Typography.Text type="secondary" style={{ fontSize: 11 }}>
									{formatEventTime(event.occurredAt)}
								</Typography.Text>
							</div>
							{event.message ? (
								<Typography.Text
									type="secondary"
									style={{ display: "block", marginTop: 2, fontSize: 11 }}
									ellipsis={{ tooltip: event.message }}
								>
									{event.message}
								</Typography.Text>
							) : null}
						</button>
					))}
				</div>
			)}
			{runEventState.nextCursor ? (
				<Button
					size="small"
					type="link"
					onClick={onLoadMoreEvents}
					style={{ marginTop: 6 }}
				>
					加载更多事件
				</Button>
			) : null}
		</div>
	);
}

function ExpiredWorkflowLedgerView({
	name,
	runEventState,
	onBack,
	onRefreshEvents,
}: {
	name?: string;
	runEventState: ReturnType<typeof useWorkflowDetail>["runEventState"];
	onBack: () => void;
	onRefreshEvents: () => void;
}) {
	const latestEvents = runEventState.items.slice(-10);
	const isPendingLedger = runEventState.run?.status === "Pending";

	return (
		<div style={{ padding: 24 }}>
			<Space direction="vertical" size={16} style={{ width: "100%" }}>
				<Space>
					<Button icon={<ArrowLeftOutlined />} onClick={onBack}>
						返回
					</Button>
					<Typography.Title level={4} style={{ margin: 0 }}>
						{name || runEventState.run?.workflowName || "运行详情"}
					</Typography.Title>
					{runEventState.run?.status ? (
						<Tag color={STATUS_COLORS[runEventState.run.status] || "default"}>
							{runEventState.run.status}
						</Tag>
					) : null}
				</Space>
				<Alert
					type={isPendingLedger ? "info" : "warning"}
					showIcon
					message={
						isPendingLedger
							? "子任务尚未提交到 Argo"
							: "Argo 工作流已不可用，正在展示历史记录"
					}
					description={
						isPendingLedger
							? "该批量子任务仍在排队或等待重试，DAG、Pod 实时状态和日志暂不可用；页面会随提交进度自动更新。"
							: "Workflow 可能已被 Argo TTL 清理，DAG、Pod 实时状态和实时日志暂不可用；提交记录、状态变化和节点事件仍会在此保留。"
					}
					action={
						<Button size="small" onClick={onRefreshEvents}>
							刷新事件
						</Button>
					}
				/>
				<Card
					title="执行记录"
					extra={
						runEventState.run ? (
							<Space size={8}>
								<Typography.Text type="secondary">
									ID {runEventState.run.id}
								</Typography.Text>
								{runEventState.run.templateVersion ? (
									<Tag>模板 v{runEventState.run.templateVersion}</Tag>
								) : null}
							</Space>
						) : null
					}
				>
					{runEventState.error ? (
						<Alert
							type="info"
							showIcon
							message="执行记录暂不可用"
							description={runEventState.error}
						/>
					) : runEventState.loading &&
						latestEvents.length === 0 &&
						!runEventState.error ? (
						<Spin size="small" />
					) : latestEvents.length === 0 && !runEventState.error ? (
						<Space direction="vertical" size={8}>
							<Typography.Text type="secondary">暂无运行事件</Typography.Text>
							{runEventState.run?.message ? (
								<Typography.Text type="secondary">
									{runEventState.run.message}
								</Typography.Text>
							) : null}
						</Space>
					) : (
						<Space direction="vertical" size={8} style={{ width: "100%" }}>
							{latestEvents.map((event) => (
								<div
									key={event.id}
									style={{
										border: "1px solid #e5e7eb",
										borderRadius: 6,
										padding: "8px 10px",
										background: "#fff",
									}}
								>
									<Space size={8} wrap>
										<Tag color={eventTagColor(event)}>
											{RUN_EVENT_LABELS[event.eventType] || event.eventType}
										</Tag>
										<Typography.Text strong>
											{shortEventSubject(event)}
										</Typography.Text>
										<Typography.Text type="secondary">
											{formatEventTime(event.occurredAt)}
										</Typography.Text>
									</Space>
									<Typography.Text
										type="secondary"
										style={{ display: "block", marginTop: 4 }}
									>
										{event.message || event.status || event.subjectId}
									</Typography.Text>
								</div>
							))}
						</Space>
					)}
				</Card>
			</Space>
		</div>
	);
}

function formatCost(value?: number | null) {
	if (typeof value !== "number") return "未生成快照";
	if (value < 0.01) return `$${value.toFixed(4)}`;
	return `$${value.toFixed(2)}`;
}

function formatCostSource(value?: string) {
	if (value === "estimated_resource_duration") return "估算";
	if (value === "not_available") return "暂无计费配置";
	return "未生成";
}

function formatDurationSeconds(startedAt?: string, finishedAt?: string) {
	if (!startedAt || !finishedAt) return "-";
	const start = new Date(startedAt).getTime();
	const end = new Date(finishedAt).getTime();
	if (Number.isNaN(start) || Number.isNaN(end) || end < start) return "-";
	const seconds = Math.round((end - start) / 1000);
	if (seconds < 60) return `${seconds}s`;
	const minutes = Math.floor(seconds / 60);
	return `${minutes}m ${seconds % 60}s`;
}

function getPipelineLabelFromLookup(
	pipelineNodeLabels: Map<string, string>,
	key?: string,
) {
	const normalized = key?.trim();
	if (!normalized) return null;
	const exact = pipelineNodeLabels.get(normalized);
	if (exact) return exact;
	const shortKey = normalized.split(".").pop()?.trim();
	if (!shortKey || shortKey === normalized) return null;
	return pipelineNodeLabels.get(shortKey) ?? null;
}

function getAssetNodeDisplayInfo(
	row: PipelineRunAssetNode,
	pipelineNodeLabels: Map<string, string>,
): { primary: string; secondary: string | null } {
	if (row.pipelineNodeId === "dag") {
		return { primary: "运行汇总", secondary: null };
	}
	const primary =
		getPipelineLabelFromLookup(pipelineNodeLabels, row.pipelineNodeId) ??
		getPipelineLabelFromLookup(pipelineNodeLabels, row.displayName) ??
		getPipelineLabelFromLookup(pipelineNodeLabels, row.argoNodeId) ??
		row.displayName ??
		row.pipelineNodeId ??
		"-";
	const secondaryCandidates = [
		row.displayName,
		row.pipelineNodeId,
		row.argoNodeId,
	];
	const secondary =
		secondaryCandidates
			.map((value) => value?.trim())
			.find((value) => value && value !== primary) ?? null;
	return { primary, secondary };
}

function assetNodeMessageSummary(message?: string): string | null {
	const normalized = message?.replace(/\s+/g, " ").trim();
	if (!normalized) return null;
	const maxLength = 96;
	if (normalized.length <= maxLength) return normalized;
	return `${normalized.slice(0, maxLength - 1)}…`;
}

function WorkflowAssetNodePanel({
	assetNodeState,
	costSummaryState,
	workflowNodeCount,
	pipelineNodeLabels,
	onSelectAssetNode,
}: {
	assetNodeState: ReturnType<typeof useWorkflowDetail>["assetNodeState"];
	costSummaryState: ReturnType<typeof useWorkflowDetail>["costSummaryState"];
	workflowNodeCount: number;
	pipelineNodeLabels: Map<string, string>;
	onSelectAssetNode: (
		row: PipelineRunAssetNode,
		action: WorkflowDagNodeAction,
	) => void;
}) {
	const summary = assetNodeState.summary;
	const noAssetOnly =
		assetNodeState.items.length > 0 &&
		assetNodeState.items.every((row) => row.assetId === "no-asset");
	const displayAssetCount = noAssetOnly
		? "无资产运行"
		: (summary?.assetCount ?? 0);
	const totalEstimatedCost =
		costSummaryState.item?.totalEstimatedCostUsd ??
		summary?.totalEstimatedCostUsd;
	const costSource = costSummaryState.item?.costSource ?? summary?.costSource;
	const expectedNodeCount = workflowNodeCount || summary?.nodeCount || 0;
	const syncedNodeCount = costSummaryState.item?.nodeSummaries?.length ?? 0;
	const syncedAssetNodeCount =
		costSummaryState.item?.assetNodeSummaries?.length ?? 0;
	const hasCostRows =
		syncedNodeCount > 0 ||
		syncedAssetNodeCount > 0 ||
		assetNodeState.items.length > 0;
	const costUnavailable =
		!costSummaryState.loading &&
		costSource === "not_available" &&
		hasCostRows &&
		totalEstimatedCost == null;
	const costSyncPartial =
		!costUnavailable &&
		expectedNodeCount > 0 &&
		syncedNodeCount > 0 &&
		syncedNodeCount < expectedNodeCount;
	const costSyncPending =
		costSummaryState.loading ||
		(expectedNodeCount > 0 && !hasCostRows && totalEstimatedCost == null);
	const showCostSummary = !costUnavailable;
	return (
		<div
			style={{
				margin: "0 24px 12px",
				border: "1px solid #e5e7eb",
				borderRadius: 8,
				background: "#fff",
				overflow: "hidden",
			}}
		>
			<div
				style={{
					display: "flex",
					alignItems: "center",
					justifyContent: "space-between",
					gap: 12,
					padding: "8px 10px",
					borderBottom: "1px solid #e5e7eb",
				}}
			>
				<div>
					<Typography.Text strong>节点明细</Typography.Text>
					<Typography.Text
						type="secondary"
						style={{ display: "block", fontSize: 12 }}
					>
						按资产和步骤定位状态、日志、Pod 与成本快照。
					</Typography.Text>
				</div>
				<Space wrap size={12}>
					<Typography.Text type="secondary">
						资产 {displayAssetCount}
					</Typography.Text>
					<Typography.Text type="secondary">
						节点 {expectedNodeCount}
					</Typography.Text>
					{showCostSummary ? (
						<>
							<Typography.Text type="secondary">
								总成本{" "}
								{costSyncPending ? "同步中" : formatCost(totalEstimatedCost)}
							</Typography.Text>
							<Tag
								color={costSyncPending || costSyncPartial ? "orange" : "blue"}
							>
								{costSyncPending
									? "同步中"
									: costSyncPartial
										? `部分同步 ${syncedNodeCount}/${expectedNodeCount}`
										: formatCostSource(costSource)}
							</Tag>
						</>
					) : null}
				</Space>
			</div>
			{costSyncPending || costSyncPartial ? (
				<Alert
					type="info"
					showIcon
					message="成本快照仍在同步"
					description="Argo 节点状态会先返回，DataBrew 成本汇总可能延迟几秒；刷新后会补齐节点耗时与估算成本。"
					style={{ margin: "8px 10px 0" }}
				/>
			) : null}
			<Table<PipelineRunAssetNode>
				size="small"
				rowKey="id"
				loading={assetNodeState.loading}
				dataSource={assetNodeState.items}
				pagination={
					assetNodeState.items.length > 10
						? { pageSize: 10, size: "small", showSizeChanger: false }
						: false
				}
				locale={{
					emptyText: assetNodeState.error || "暂无资产节点明细",
				}}
				columns={[
					{
						title: "资产",
						dataIndex: "assetId",
						width: 170,
						render: (value: string) => (
							<Tag color={value === "no-asset" ? "default" : "blue"}>
								{value === "no-asset" ? "无资产" : value}
							</Tag>
						),
					},
					{
						title: "节点",
						dataIndex: "displayName",
						render: (_, row) => {
							const display = getAssetNodeDisplayInfo(row, pipelineNodeLabels);
							return (
								<Space direction="vertical" size={0} style={{ maxWidth: 360 }}>
									<Typography.Text
										strong
										ellipsis={{ tooltip: display.primary }}
									>
										{display.primary}
									</Typography.Text>
									{display.secondary ? (
										<Typography.Text
											type="secondary"
											copyable={{ text: display.secondary }}
											ellipsis={{ tooltip: display.secondary }}
											style={{ fontSize: 12, maxWidth: 340 }}
										>
											{display.secondary}
										</Typography.Text>
									) : null}
								</Space>
							);
						},
					},
					{
						title: "状态",
						dataIndex: "status",
						width: 240,
						render: (value: string, row) => {
							const messageSummary = assetNodeMessageSummary(row.message);
							return (
								<Space direction="vertical" size={0} style={{ maxWidth: 220 }}>
									<Tag
										color={eventTagColor({
											status: value,
											eventType: value,
										} as PipelineRunEvent)}
									>
										{value || "-"}
									</Tag>
									{messageSummary ? (
										<Tooltip title={row.message}>
											<Typography.Text
												type="secondary"
												style={{ fontSize: 12, maxWidth: 220 }}
												ellipsis
											>
												{messageSummary}
											</Typography.Text>
										</Tooltip>
									) : null}
								</Space>
							);
						},
					},
					{
						title: "耗时",
						width: 100,
						render: (_, row) =>
							formatDurationSeconds(row.startedAt, row.finishedAt),
					},
					{
						title: "估算成本",
						width: 120,
						render: (_, row) => (
							<Tooltip
								title={
									typeof row.estimatedCostUsd === "number"
										? "估算成本，非 GCP Billing 最终账单"
										: "该节点未生成成本快照"
								}
							>
								<span>{formatCost(row.estimatedCostUsd)}</span>
							</Tooltip>
						),
					},
					{
						title: "操作",
						width: 190,
						render: (_, row) => {
							const actions = [
								<Button
									key="summary"
									size="small"
									type="link"
									onClick={() => onSelectAssetNode(row, "summary")}
								>
									节点
								</Button>,
							];
							if (row.logRef) {
								actions.push(
									<Button
										key="logs"
										size="small"
										type="link"
										onClick={() => onSelectAssetNode(row, "logs")}
									>
										日志
									</Button>,
								);
							}
							if (row.podName) {
								actions.push(
									<Button
										key="pod"
										size="small"
										type="link"
										onClick={() => onSelectAssetNode(row, "runtime")}
									>
										Pod
									</Button>,
								);
							}
							return <Space size={6}>{actions}</Space>;
						},
					},
				]}
			/>
		</div>
	);
}

export default function WorkflowDetailPage({
	legacyRoute = false,
}: {
	legacyRoute?: boolean;
}) {
	const { message: messageApi } = App.useApp();
	const { name, runId } = useParams<{ name?: string; runId?: string }>();
	const detailName = name ?? runId;
	const navigate = useNavigate();
	const location = useLocation();
	const backTarget = useMemo(
		() => resolveWorkflowDetailBackTarget(location.state),
		[location.state],
	);
	const backLabel = (location.state as { fromBatchJobId?: string } | null)
		?.fromBatchJobId
		? "返回批次详情"
		: "返回";
	const [viewMode, setViewMode] = useState<"dag" | "timeline">("dag");
	const [operationLoading, setOperationLoading] =
		useState<WorkflowOperationKey | null>(null);
	const [confirmOperation, setConfirmOperation] =
		useState<WorkflowOperationConfig | null>(null);
	const [nodeDetailTab, setNodeDetailTab] =
		useState<WorkflowNodeDetailTabKey>("summary");
	const [nodePanelOpen, setNodePanelOpen] = useState(false);

	const {
		workflow,
		loading,
		loadError,
		selectedNode,
		selectNode,
		loadWorkflow,
		logState,
		runEventState,
		runEventFilters,
		setRunEventFilters,
		loadRunEvents,
		assetNodeState,
		costSummaryState,
		setLogSearch,
		startFollowLogs,
		stopFollowLogs,
		downloadLogs,
	} = useWorkflowDetail(
		detailName,
		runId ? { lookupMode: "runId" } : undefined,
	);
	const pipelineNodeLabels = useMemo(
		() => buildPipelineNodeLabelLookup(runEventState.run?.pipelineJSON),
		[runEventState.run?.pipelineJSON],
	);
	const displayNodes = useMemo(
		() =>
			workflow ? buildDisplayWorkflowNodes(workflow, runEventState.run) : [],
		[workflow, runEventState.run],
	);
	const displayWorkflow = useMemo(() => {
		if (!workflow) {
			return workflow;
		}
		if (!runEventState.run) {
			return {
				...workflow,
				nodes: displayNodes,
			};
		}
		return {
			...workflow,
			nodes: displayNodes,
			status: runEventState.run.status || workflow.status,
			message: runEventState.run.message || workflow.message,
			finishedAt: runEventState.run.finishedAt || workflow.finishedAt,
		};
	}, [displayNodes, workflow, runEventState.run]);
	const [showNodeLogs, setShowNodeLogs] = useState(false);
	const operations = useMemo(
		() => (displayWorkflow ? getWorkflowOperationConfigs(displayWorkflow) : []),
		[displayWorkflow],
	);
	const availableOperations = useMemo(
		() =>
			displayWorkflow
				? getAvailableWorkflowOperationConfigs(displayWorkflow)
				: [],
		[displayWorkflow],
	);
	const runProductOperation = useCallback(
		async (runId: string, key: WorkflowOperationKey) => {
			switch (key) {
				case "delete":
					await deleteRun(runId);
					return;
				case "retry":
					await retryRun(runId);
					return;
				case "resubmit":
					await resubmitRun(runId);
					return;
				case "stop":
					await stopRun(runId);
					return;
				case "suspend":
					await suspendRun(runId);
					return;
				case "resume":
					await resumeRun(runId);
					return;
				case "terminate":
					await terminateRun(runId);
					return;
			}
		},
		[],
	);
	const displaySelectedNode = useMemo(() => {
		if (!selectedNode || !displayWorkflow) return selectedNode;
		return (
			displayWorkflow.nodes.find((node) => node.id === selectedNode.id) ??
			selectedNode
		);
	}, [displayWorkflow, selectedNode]);
	const displaySelectedPipelineNode = useMemo(
		() =>
			findPipelineNodeForWorkflowNode(
				runEventState.run?.pipelineJSON,
				runEventState.run,
				displaySelectedNode,
			),
		[displaySelectedNode, runEventState.run],
	);
	const canRetryFailedNode = useMemo(() => {
		const retryOp = operations.find(
			(operation) => operation.key === "retry" && !operation.disabled,
		);
		if (!retryOp || !displaySelectedNode) return false;
		return displaySelectedNode.phase === "Failed";
	}, [displaySelectedNode, operations]);

	const executeOperation = useCallback(
		async (operation: WorkflowOperationConfig) => {
			if (!displayWorkflow || operation.disabled) return;
			const runId = runEventState.run?.id;
			setOperationLoading(operation.key);
			try {
				if (operation.key === "delete" && runId) {
					await deleteRun(runId);
					messageApi.success?.("执行记录删除已提交");
					navigate("/pipeline?tab=executions");
					return;
				} else if (operation.key === "retry") {
					const outcome = await runWorkflowRetryWithFeedback(
						displayWorkflow,
						() =>
							runId
								? runProductOperation(runId, "retry").then(() => ({
										message: "run retry submitted",
									}))
								: operation.run(),
					);
					if (outcome === "no_progress") {
						messageApi.warning?.(
							"重试已提交，但执行状态未变化。若曾手动停止，请使用「重提交」。",
						);
					} else {
						messageApi.success?.("重试已提交");
					}
				} else {
					if (runId) {
						await runProductOperation(runId, operation.key);
					} else {
						await operation.run();
					}
					messageApi.success?.(
						operation.key === "delete"
							? "工作流删除已提交"
							: `${operation.title}已提交`,
					);
				}
				if (operation.key === "delete") {
					navigate("/pipeline?tab=executions");
					return;
				}
				if (operation.key === "resubmit") {
					navigate("/pipeline?tab=executions");
					return;
				}
				loadWorkflow();
			} catch (err) {
				messageApi.error?.(`${operation.title}失败: ${String(err)}`);
			} finally {
				setOperationLoading(null);
			}
		},
		[
			displayWorkflow,
			loadWorkflow,
			messageApi,
			navigate,
			runEventState.run?.id,
			runProductOperation,
		],
	);

	const runOperation = useCallback(
		(operation: WorkflowOperationConfig) => {
			if (
				operation.key === "delete" ||
				operation.key === "terminate" ||
				operation.key === "resubmit" ||
				operation.key === "retry"
			) {
				setConfirmOperation(operation);
				return;
			}
			executeOperation(operation);
		},
		[executeOperation],
	);

	const closeNodeDetailPanel = useCallback(() => {
		setNodePanelOpen(false);
		setShowNodeLogs(false);
		selectNode(null);
	}, [selectNode]);

	const handleSelectNode = useCallback(
		(node: WorkflowNodeStatus | null) => {
			setNodeDetailTab("summary");
			setNodePanelOpen(!!node);
			selectNode(node);
		},
		[selectNode],
	);

	const handleNodeAction = useCallback(
		(node: WorkflowNodeStatus, action: WorkflowDagNodeAction) => {
			const tabByAction: Record<
				WorkflowDagNodeAction,
				WorkflowNodeDetailTabKey
			> = {
				summary: "summary",
				logs: "logs",
				runtime: "runtime",
				io: "io",
				terminal: "runtime",
			};
			setNodeDetailTab(tabByAction[action]);
			selectNode(node);
			if (action === "logs") {
				setNodePanelOpen(false);
				setShowNodeLogs(true);
				return;
			}
			setNodePanelOpen(true);
		},
		[selectNode],
	);

	const handleSelectEventNode = useCallback(
		(event: PipelineRunEvent) => {
			if (!displayWorkflow || event.subjectType !== "node") {
				return;
			}
			const node = displayWorkflow.nodes.find(
				(item) => item.id === event.subjectId,
			);
			if (!node) {
				messageApi.warning?.("事件关联的节点不在当前 DAG 中");
				return;
			}
			handleSelectNode(node);
		},
		[displayWorkflow, handleSelectNode, messageApi],
	);

	const handleFilterEvents = useCallback(
		(filters: ReturnType<typeof useWorkflowDetail>["runEventFilters"]) => {
			setRunEventFilters(filters);
		},
		[setRunEventFilters],
	);

	const handleLoadMoreEvents = useCallback(() => {
		if (!runEventState.nextCursor) return;
		loadRunEvents({ append: true, cursor: runEventState.nextCursor });
	}, [loadRunEvents, runEventState.nextCursor]);

	const handleRefreshEvents = useCallback(() => {
		loadWorkflow();
		loadRunEvents();
	}, [loadRunEvents, loadWorkflow]);

	const handleSelectAssetNode = useCallback(
		(row: PipelineRunAssetNode, action: WorkflowDagNodeAction) => {
			if (!displayWorkflow) return;
			const node = displayWorkflow.nodes.find(
				(item) =>
					item.id === row.argoNodeId ||
					item.id === row.pipelineNodeId ||
					item.displayName === row.displayName ||
					item.name === row.displayName,
			);
			if (!node) {
				messageApi.warning?.("资产节点关联的 DAG 节点暂不可见");
				return;
			}
			handleNodeAction(node, action);
		},
		[displayWorkflow, handleNodeAction, messageApi],
	);

	const handleShowNodeLogs = useCallback(() => {
		if (!displaySelectedNode) {
			return;
		}
		setShowNodeLogs(true);
	}, [displaySelectedNode]);

	const handleCloseNodeLogs = useCallback(() => {
		setShowNodeLogs(false);
	}, []);

	const handleRetryWorkflow = useCallback(() => {
		if (!displayWorkflow) return;
		const retryConfig = operations.find(
			(operation) => operation.key === "retry",
		);
		if (!retryConfig || retryConfig.disabled) {
			messageApi.warning?.("当前工作流状态不可重试");
			return;
		}
		runOperation(retryConfig);
	}, [displayWorkflow, messageApi, operations, runOperation]);

	useEffect(() => {
		if (legacyRoute && detailName) {
			navigate(`/pipeline/executions/${encodeURIComponent(detailName)}`, {
				replace: true,
			});
		}
	}, [legacyRoute, detailName, navigate]);

	useEffect(() => {
		if (!selectedNode) {
			setShowNodeLogs(false);
			setNodePanelOpen(false);
		}
	}, [selectedNode]);

	if (loading) {
		return (
			<div
				style={{
					display: "flex",
					justifyContent: "center",
					padding: 80,
				}}
			>
				<Spin size="large" />
			</div>
		);
	}

	if (!loading && loadError?.kind === "not_found") {
		if (runEventState.run || runEventState.items.length > 0) {
			return (
				<ExpiredWorkflowLedgerView
					name={detailName}
					runEventState={runEventState}
					onBack={() => navigate(backTarget)}
					onRefreshEvents={loadRunEvents}
				/>
			);
		}
		return (
			<div style={{ padding: 24 }}>
				<Alert
					type="warning"
					showIcon
					message="未找到工作流"
					description={loadError.message}
					action={
						<Space>
							<Button size="small" onClick={() => navigate(backTarget)}>
								返回
							</Button>
							<Button size="small" onClick={loadWorkflow}>
								重试
							</Button>
						</Space>
					}
				/>
			</div>
		);
	}

	if (!loading && loadError?.kind === "error") {
		return (
			<div style={{ padding: 24 }}>
				<Alert
					type="error"
					showIcon
					message="加载工作流失败"
					description={loadError.message}
					action={
						<Button size="small" onClick={loadWorkflow}>
							重试
						</Button>
					}
				/>
			</div>
		);
	}

	if (!workflow || !displayWorkflow) {
		return null;
	}

	const displayableNodeCount = countDisplayableWorkflowNodes(
		displayWorkflow.nodes,
	);
	const graphHeight =
		displayableNodeCount <= 1 ? 320 : displayableNodeCount <= 5 ? 420 : 500;

	return (
		<div
			style={{
				height: "calc(100vh - 49px)",
				display: "flex",
				flexDirection: "column",
				width: "100%",
			}}
		>
			<div className="workflow-detail-header">
				<Button
					icon={<ArrowLeftOutlined />}
					size="small"
					onClick={() => navigate(backTarget)}
				>
					{backLabel}
				</Button>
				<h3 className="workflow-detail-title" title={workflow.name}>
					{workflow.name}
				</h3>
				<Tag color={STATUS_COLORS[displayWorkflow.status] || "default"}>
					{formatWorkflowPhaseLabel(displayWorkflow.status)}
				</Tag>
				{runEventState.run || runEventState.items.length > 0 ? (
					<Tag color="green">DataBrew 运行</Tag>
				) : (
					<Tag color="orange">外部 Workflow</Tag>
				)}
				{displayWorkflow.message ? (
					<Tooltip title={displayWorkflow.message}>
						<span
							style={{
								color: "#dc2626",
								fontSize: 12,
								maxWidth: 280,
								overflow: "hidden",
								textOverflow: "ellipsis",
								whiteSpace: "nowrap",
							}}
						>
							{displayWorkflow.message}
						</span>
					</Tooltip>
				) : null}
				<DurationPanel
					phase={displayWorkflow.status}
					startedAt={workflow.createdAt}
					finishedAt={displayWorkflow.finishedAt}
					progress={workflow.progress}
				/>
				{costSummaryState.item &&
				typeof costSummaryState.item.totalEstimatedCostUsd === "number" ? (
					<Tooltip title="估算成本，非 GCP Billing 最终账单">
						<Tag color="blue">
							~${costSummaryState.item.totalEstimatedCostUsd?.toFixed(4)}
						</Tag>
					</Tooltip>
				) : null}
				<div className="workflow-detail-actions">
					<Space size={4} wrap>
						{availableOperations.map((operation) => (
							<Button
								key={operation.key}
								size="small"
								icon={operation.icon}
								danger={operation.danger}
								loading={operationLoading === operation.key}
								onClick={() => runOperation(operation)}
							>
								{operation.title}
							</Button>
						))}
					</Space>
					<Segmented
						value={viewMode}
						onChange={(val) => setViewMode(val as "dag" | "timeline")}
						options={[
							{
								label: (
									<>
										<ApartmentOutlined /> DAG
									</>
								),
								value: "dag",
							},
							{
								label: (
									<>
										<BarsOutlined /> 时间线
									</>
								),
								value: "timeline",
							},
						]}
					/>
				</div>
			</div>
			<WorkflowSummaryCards
				workflow={displayWorkflow}
				runEventState={runEventState}
				costSummaryState={costSummaryState}
			/>
			<div
				style={{
					flex: 1,
					minHeight: 0,
					minWidth: 0,
					overflowY: "auto",
					background: "#f8fafc",
					paddingTop: 12,
				}}
			>
				<div
					className="workflow-detail-graph-shell"
					style={{
						height: viewMode === "timeline" ? "auto" : graphHeight,
						minHeight: viewMode === "timeline" ? 280 : graphHeight,
						overflow: viewMode === "timeline" ? "visible" : "hidden",
					}}
				>
					{viewMode === "dag" ? (
						<WorkflowDagView
							nodes={displayWorkflow.nodes}
							workflowEdges={displayWorkflow.edges}
							selectedNodeId={displaySelectedNode?.id ?? null}
							onNodeSelect={handleSelectNode}
							onNodeAction={handleNodeAction}
							emptyMessage={displayWorkflow.message}
							workflowStatus={displayWorkflow.status}
							pipelineLabels={pipelineNodeLabels}
						/>
					) : (
						<WorkflowTimelineView
							nodes={displayWorkflow.nodes}
							selectedNodeId={displaySelectedNode?.id ?? null}
							onNodeSelect={handleSelectNode}
							pipelineLabels={pipelineNodeLabels}
						/>
					)}
				</div>
				{runEventState.run || runEventState.items.length > 0 ? (
					<WorkflowAssetNodePanel
						assetNodeState={assetNodeState}
						costSummaryState={costSummaryState}
						workflowNodeCount={displayableNodeCount}
						pipelineNodeLabels={pipelineNodeLabels}
						onSelectAssetNode={handleSelectAssetNode}
					/>
				) : (
					<div style={{ padding: "12px 24px" }}>
						<Alert
							type="info"
							showIcon
							message="暂无资产节点明细"
							description="此工作流不是由 DataBrew 部署，没有资产绑定信息。仍可使用 DAG、Pod 日志和终端调试。"
						/>
					</div>
				)}
				<WorkflowRunContextPanel
					runEventState={runEventState}
					runEventFilters={runEventFilters}
					onFilterEvents={handleFilterEvents}
					onRefreshEvents={handleRefreshEvents}
					onLoadMoreEvents={handleLoadMoreEvents}
					onSelectNodeEvent={handleSelectEventNode}
				/>
			</div>

			<WorkflowNodeDetailPanel
				node={displaySelectedNode}
				workflow={displayWorkflow}
				pipelineNode={displaySelectedPipelineNode}
				open={nodePanelOpen}
				onClose={closeNodeDetailPanel}
				canRetryWorkflow={canRetryFailedNode}
				onRetryWorkflow={handleRetryWorkflow}
				onShowLogs={handleShowNodeLogs}
				activeTab={nodeDetailTab}
				onActiveTabChange={setNodeDetailTab}
			/>

			<Modal
				open={!!confirmOperation}
				title={
					confirmOperation
						? `确认${confirmOperation.title} ${workflow.name}?`
						: ""
				}
				okText={confirmOperation?.title}
				cancelText="取消"
				okButtonProps={{ danger: confirmOperation?.danger }}
				onOk={async () => {
					if (!confirmOperation) return;
					const operation = confirmOperation;
					setConfirmOperation(null);
					await executeOperation(operation);
				}}
				onCancel={() => setConfirmOperation(null)}
			>
				{confirmOperation
					? getWorkflowOperationConfirmText(confirmOperation.key)
					: null}
			</Modal>

			<Modal
				open={showNodeLogs}
				title={
					displaySelectedNode
						? `${displaySelectedNode.displayName || displaySelectedNode.name} 日志`
						: "日志"
				}
				width="80%"
				onCancel={handleCloseNodeLogs}
				footer={null}
				style={{ top: 32 }}
				styles={{ body: { height: "calc(100vh - 180px)", padding: 0 } }}
			>
				<WorkflowLogPanel
					selectedNode={displaySelectedNode}
					loading={logState.loading}
					logContent={logState.content}
					error={logState.error}
					search={logState.search}
					following={logState.following}
					followStatus={logState.followStatus}
					followMessage={logState.followMessage}
					logResponse={logState.response}
					clientTruncated={logState.clientTruncated}
					onSearch={setLogSearch}
					onFollow={startFollowLogs}
					onStop={stopFollowLogs}
					onDownload={downloadLogs}
				/>
			</Modal>
		</div>
	);
}
