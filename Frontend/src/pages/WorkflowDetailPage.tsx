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
import { Link, useNavigate, useParams } from "react-router-dom";
import type {
	PipelineRunAssetNode,
	PipelineRunEvent,
} from "../api/pipelineApi";
import type {
	WorkflowLogResponse,
	WorkflowNodeStatus,
} from "../api/workflowApi";
import { DurationPanel } from "../components/common/DurationPanel";
import { LinkifiedText } from "../components/common/LinkifiedText";
import {
	WorkflowNodeDetailPanel,
	type WorkflowNodeDetailTabKey,
} from "../components/pipeline/WorkflowNodeDetailPanel";
import { STATUS_COLORS } from "../lib/constants";
import {
	getAvailableWorkflowOperationConfigs,
	getWorkflowOperationConfigs,
	type WorkflowOperationConfig,
	type WorkflowOperationKey,
} from "../lib/workflow-operations";
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
	const visibleLog = useMemo(
		() =>
			logContent === null
				? null
				: prepareVisibleLogContent(logContent, selectedNode),
		[logContent, selectedNode],
	);
	const logElement =
		visibleLog === null
			? null
			: buildHighlightedLogNodes(visibleLog.content, search);
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

	useEffect(() => {
		if (
			selectedNode &&
			!loading &&
			!error &&
			visibleLog !== null &&
			logBodyRef.current
		) {
			logBodyRef.current.scrollTop = logBodyRef.current.scrollHeight;
		}
	}, [selectedNode, loading, error, visibleLog]);

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
								messageApi.success("已复制当前可见日志");
							}}
						>
							复制可见日志
						</Button>
					</div>
					<div
						ref={logBodyRef}
						style={{
							flex: 1,
							fontSize: 11,
							fontFamily: '"SF Mono", "Fira Code", monospace',
							whiteSpace: "pre-wrap",
							wordBreak: "break-word",
							overflow: "auto",
							background: "#f8f9fa",
							padding: 12,
							borderRadius: 6,
							border: "1px solid #e5e7eb",
							minHeight: 0,
							lineHeight: 1.55,
						}}
					>
						{logElement}
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
	if (error.includes("未找到关联的 DataBrew pipeline run")) {
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
	const assetCount = assetIdList.length;
	const visibleAssetIds = assetIdList.slice(0, 4);
	const hiddenAssetCount = Math.max(
		assetIdList.length - visibleAssetIds.length,
		0,
	);

	const cards = [
		{
			label: "模板",
			value: templateName || "—",
			extra: templateVersion ? `v${templateVersion}` : undefined,
		},
		{
			label: "资产",
			value:
				assetCount > 0 ? (
					<div style={{ display: "flex", gap: 4, flexWrap: "wrap" }}>
						{visibleAssetIds.map((assetId) => (
							<Link key={assetId} to={`/assets/${encodeURIComponent(assetId)}`}>
								<Tag color="blue" style={{ marginInlineEnd: 0 }}>
									{assetId}
								</Tag>
							</Link>
						))}
						{hiddenAssetCount > 0 ? <Tag>+{hiddenAssetCount}</Tag> : null}
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
		<div
			style={{
				display: "flex",
				gap: 12,
				padding: "8px 24px",
				borderBottom: "1px solid #e5e7eb",
				background: "#fff",
				flexWrap: "wrap",
			}}
		>
			{cards.map((card) => (
				<div
					key={card.label}
					style={{
						flex: "1 1 100px",
						minWidth: 80,
						padding: "6px 10px",
						background: "#f8fafc",
						borderRadius: 6,
						border: "1px solid #e5e7eb",
					}}
				>
					<div style={{ fontSize: 11, color: "#94a3b8", marginBottom: 2 }}>
						{card.label}
					</div>
					<div style={{ fontSize: 13, fontWeight: 600, color: "#0f172a" }}>
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
	const latestEvents = runEventState.items.slice(-5);

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
						label: s,
						value: s,
					}))}
				/>
				<Input.Search
					size="small"
					placeholder="搜索消息/ID"
					allowClear
					style={{ width: 180 }}
					value={runEventFilters.q}
					onChange={(event) =>
						onFilterEvents({ ...runEventFilters, q: event.target.value })
					}
					onSearch={() => onRefreshEvents()}
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

	return (
		<div style={{ padding: 24 }}>
			<Space direction="vertical" size={16} style={{ width: "100%" }}>
				<Space>
					<Button icon={<ArrowLeftOutlined />} onClick={onBack}>
						返回
					</Button>
					<Typography.Title level={4} style={{ margin: 0 }}>
						{name || "运行详情"}
					</Typography.Title>
					{runEventState.run?.status ? (
						<Tag color={STATUS_COLORS[runEventState.run.status] || "default"}>
							{runEventState.run.status}
						</Tag>
					) : null}
				</Space>
				<Alert
					type="warning"
					showIcon
					message="Argo 工作流已不可用，正在展示 DataBrew 历史"
					description="Workflow 可能已被 Argo TTL 清理，DAG、Pod 实时状态和实时日志暂不可用；提交记录、状态变化和节点事件会继续保留在 DataBrew 运行账本中。"
					action={
						<Button size="small" onClick={onRefreshEvents}>
							刷新事件
						</Button>
					}
				/>
				<Card
					title="运行账本"
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
							message="运行账本暂不可用"
							description={runEventState.error}
						/>
					) : runEventState.loading &&
						latestEvents.length === 0 &&
						!runEventState.error ? (
						<Spin size="small" />
					) : latestEvents.length === 0 && !runEventState.error ? (
						<Typography.Text type="secondary">暂无运行事件</Typography.Text>
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

function WorkflowAssetNodePanel({
	assetNodeState,
	costSummaryState,
	workflowNodeCount,
	onSelectAssetNode,
}: {
	assetNodeState: ReturnType<typeof useWorkflowDetail>["assetNodeState"];
	costSummaryState: ReturnType<typeof useWorkflowDetail>["costSummaryState"];
	workflowNodeCount: number;
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
					<Typography.Text type="secondary">
						总成本{" "}
						{costSyncPending
							? "同步中"
							: costUnavailable
								? "暂无估算"
								: formatCost(totalEstimatedCost)}
					</Typography.Text>
					<Tag
						color={
							costSyncPending || costSyncPartial
								? "orange"
								: costUnavailable
									? "default"
									: "blue"
						}
					>
						{costSyncPending
							? "同步中"
							: costSyncPartial
								? `部分同步 ${syncedNodeCount}/${expectedNodeCount}`
								: formatCostSource(costSource)}
					</Tag>
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
			) : costUnavailable ? (
				<Alert
					type="warning"
					showIcon
					message="暂无估算成本"
					description="后端未加载计费配置，或当前资源组合没有价格映射；节点状态、日志和 Pod 诊断不受影响。"
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
						render: (_, row) =>
							row.pipelineNodeId === "dag"
								? "运行汇总"
								: row.displayName || row.pipelineNodeId,
					},
					{
						title: "状态",
						dataIndex: "status",
						width: 120,
						render: (value: string) => (
							<Tag
								color={eventTagColor({
									status: value,
									eventType: value,
								} as PipelineRunEvent)}
							>
								{value || "-"}
							</Tag>
						),
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
	const { name } = useParams<{ name: string }>();
	const navigate = useNavigate();
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
	} = useWorkflowDetail(name);
	const [showNodeLogs, setShowNodeLogs] = useState(false);
	const operations = useMemo(
		() => (workflow ? getWorkflowOperationConfigs(workflow) : []),
		[workflow],
	);
	const availableOperations = useMemo(
		() => (workflow ? getAvailableWorkflowOperationConfigs(workflow) : []),
		[workflow],
	);
	const canRetryWorkflow = useMemo(
		() => operations.some((op) => op.key === "retry" && !op.disabled),
		[operations],
	);

	const executeOperation = useCallback(
		async (operation: WorkflowOperationConfig) => {
			if (!workflow || operation.disabled) return;
			setOperationLoading(operation.key);
			try {
				await operation.run();
				messageApi.success(`${operation.title}已提交`);
				if (operation.key === "delete") {
					navigate("/workflows");
					return;
				}
				loadWorkflow();
			} catch (err) {
				messageApi.error(`${operation.title}失败: ${String(err)}`);
			} finally {
				setOperationLoading(null);
			}
		},
		[loadWorkflow, messageApi, navigate, workflow],
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
			if (!workflow || event.subjectType !== "node") {
				return;
			}
			const node = workflow.nodes.find((item) => item.id === event.subjectId);
			if (!node) {
				messageApi.warning("事件关联的节点不在当前 DAG 中");
				return;
			}
			handleSelectNode(node);
		},
		[handleSelectNode, messageApi, workflow],
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

	const handleSelectAssetNode = useCallback(
		(row: PipelineRunAssetNode, action: WorkflowDagNodeAction) => {
			if (!workflow) return;
			const node = workflow.nodes.find(
				(item) =>
					item.id === row.argoNodeId ||
					item.id === row.pipelineNodeId ||
					item.displayName === row.displayName ||
					item.name === row.displayName,
			);
			if (!node) {
				messageApi.warning("资产节点关联的 DAG 节点暂不可见");
				return;
			}
			handleNodeAction(node, action);
		},
		[handleNodeAction, messageApi, workflow],
	);

	const handleShowNodeLogs = useCallback(() => {
		if (!selectedNode) {
			return;
		}
		setShowNodeLogs(true);
	}, [selectedNode]);

	const handleCloseNodeLogs = useCallback(() => {
		setShowNodeLogs(false);
	}, []);

	const handleRetryWorkflow = useCallback(() => {
		if (!workflow) return;
		const retryConfig = operations.find(
			(operation) => operation.key === "retry",
		);
		if (!retryConfig || retryConfig.disabled) {
			messageApi.warning("当前工作流状态不可重试");
			return;
		}
		runOperation(retryConfig);
	}, [messageApi, operations, runOperation, workflow]);

	useEffect(() => {
		if (legacyRoute && name) {
			navigate(`/pipeline/executions/${encodeURIComponent(name)}`, {
				replace: true,
			});
		}
	}, [legacyRoute, name, navigate]);

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
					name={name}
					runEventState={runEventState}
					onBack={() => navigate("/pipeline?tab=executions")}
					onRefreshEvents={loadRunEvents}
				/>
			);
		}
		return <div style={{ padding: 24 }}>未找到工作流</div>;
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

	if (!workflow) {
		return null;
	}

	const displayableNodeCount = countDisplayableWorkflowNodes(workflow.nodes);
	const graphHeight =
		displayableNodeCount <= 1 ? 320 : displayableNodeCount <= 5 ? 420 : 500;

	return (
		<div
			style={{
				height: "calc(100vh - 49px)",
				display: "flex",
				flexDirection: "column",
				maxWidth: 1400,
				margin: "0 auto",
			}}
		>
			<div
				style={{
					padding: "8px 24px",
					borderBottom: "1px solid #e5e7eb",
					display: "flex",
					alignItems: "center",
					gap: 12,
					flexWrap: "wrap",
				}}
			>
				<Button
					icon={<ArrowLeftOutlined />}
					size="small"
					onClick={() => navigate("/pipeline?tab=executions")}
				>
					返回
				</Button>
				<h3 style={{ margin: 0, fontSize: 15 }}>{workflow.name}</h3>
				<Tag color={STATUS_COLORS[workflow.status] || "default"}>
					{workflow.status}
				</Tag>
				{runEventState.run || runEventState.items.length > 0 ? (
					<Tag color="green">DataBrew 运行</Tag>
				) : (
					<Tag color="orange">外部 Workflow</Tag>
				)}
				{workflow.message ? (
					<Tooltip title={workflow.message}>
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
							{workflow.message}
						</span>
					</Tooltip>
				) : null}
				<DurationPanel
					phase={workflow.status}
					startedAt={workflow.createdAt}
					finishedAt={workflow.finishedAt}
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
				<div
					style={{
						marginLeft: "auto",
						display: "flex",
						alignItems: "center",
						gap: 8,
					}}
				>
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
				workflow={workflow}
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
					style={{
						height: graphHeight,
						margin: "0 24px 12px",
						border: "1px solid #dbe3ee",
						borderRadius: 8,
						overflow: "hidden",
						background: "#eef2f6",
					}}
				>
					{viewMode === "dag" ? (
						<WorkflowDagView
							nodes={workflow.nodes}
							workflowEdges={workflow.edges}
							selectedNodeId={selectedNode?.id ?? null}
							onNodeSelect={handleSelectNode}
							onNodeAction={handleNodeAction}
							emptyMessage={workflow.message}
							workflowStatus={workflow.status}
						/>
					) : (
						<WorkflowTimelineView
							nodes={workflow.nodes}
							selectedNodeId={selectedNode?.id ?? null}
							onNodeSelect={handleSelectNode}
						/>
					)}
				</div>
				{runEventState.run || runEventState.items.length > 0 ? (
					<WorkflowAssetNodePanel
						assetNodeState={assetNodeState}
						costSummaryState={costSummaryState}
						workflowNodeCount={displayableNodeCount}
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
					onRefreshEvents={loadRunEvents}
					onLoadMoreEvents={handleLoadMoreEvents}
					onSelectNodeEvent={handleSelectEventNode}
				/>
			</div>

			<WorkflowNodeDetailPanel
				node={selectedNode}
				workflow={workflow}
				open={nodePanelOpen}
				onClose={closeNodeDetailPanel}
				canRetryWorkflow={canRetryWorkflow}
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
				{confirmOperation?.key === "resubmit" ||
				confirmOperation?.key === "retry" ? (
					<p>将基于当前工作流再次提交执行。</p>
				) : null}
			</Modal>

			<Modal
				open={showNodeLogs}
				title={
					selectedNode
						? `${selectedNode.displayName || selectedNode.name} 日志`
						: "日志"
				}
				width="80%"
				onCancel={handleCloseNodeLogs}
				footer={null}
				style={{ top: 32 }}
				styles={{ body: { height: "calc(100vh - 180px)", padding: 0 } }}
			>
				<WorkflowLogPanel
					selectedNode={selectedNode}
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
