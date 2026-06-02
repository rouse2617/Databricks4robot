import {
	ApartmentOutlined,
	ArrowLeftOutlined,
	BarsOutlined,
	CopyOutlined,
} from "@ant-design/icons";
import Ansi from "ansi-to-react";
import {
	Alert,
	Button,
	Card,
	Descriptions,
	Input,
	Modal,
	message,
	Segmented,
	Space,
	Spin,
	Tag,
	Tooltip,
	Typography,
} from "antd";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import type { WorkflowNodeStatus } from "../api/workflowApi";
import { DurationPanel } from "../components/common/DurationPanel";
import { LinkifiedText } from "../components/common/LinkifiedText";
import {
	WorkflowNodeDetailPanel,
	type WorkflowNodeDetailTabKey,
} from "../components/pipeline/WorkflowNodeDetailPanel";
import { WorkflowNodeSummaryTable } from "../components/pipeline/WorkflowNodeSummaryTable";
import { STATUS_COLORS } from "../lib/constants";
import {
	getAvailableWorkflowOperationConfigs,
	getWorkflowOperationConfigs,
	type WorkflowOperationConfig,
	type WorkflowOperationKey,
} from "../lib/workflow-operations";
import { useWorkflowDetail } from "./useWorkflowDetail";
import type { WorkflowDagNodeAction } from "./WorkflowDagNode";
import { WorkflowDagView } from "./WorkflowDagView";
import { WorkflowTimelineView } from "./WorkflowTimelineView";
import {
	LOG_MAX_RENDER_LINES,
	prepareVisibleLogContent,
} from "./workflowLogView";

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
	onSearch,
	following,
	onFollow,
	onStop,
	onDownload,
}: {
	selectedNode: WorkflowNodeStatus | null;
	loading: boolean;
	logContent: string | null;
	error: string | null;
	search: string;
	onSearch: (value: string) => void;
	following: boolean;
	onFollow: () => void;
	onStop: () => void;
	onDownload: () => void;
}) {
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
					marginBottom: 12,
					display: "flex",
					gap: 8,
					alignItems: "center",
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
				{!following ? (
					<Button size="small" type="primary" onClick={onFollow}>
						实时日志
					</Button>
				) : (
					<Button size="small" danger onClick={onStop}>
						停止
					</Button>
				)}
				<Button size="small" onClick={onDownload}>
					下载
				</Button>
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
					{visibleLog?.truncated && (
						<Alert
							type="warning"
							showIcon
							style={{ marginBottom: 8 }}
							message={`日志较大，当前仅显示尾部 ${visibleLog.content.length.toLocaleString()} 字符 / ${Math.min(visibleLog.totalLines, LOG_MAX_RENDER_LINES).toLocaleString()} 行。`}
							description="完整大日志需要后端 tail、分页或流式接口支持；当前视图会限制渲染量以避免浏览器卡顿。"
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
							{search.trim() ? `，搜索：${search.trim()}` : ""}
						</span>
						<Button
							size="small"
							icon={<CopyOutlined />}
							onClick={async () => {
								if (!visibleLog) return;
								await navigator.clipboard.writeText(visibleLog.content);
								message.success("已复制当前可见日志");
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

function WorkflowRunContextPanel({
	workflow,
}: {
	workflow: NonNullable<ReturnType<typeof useWorkflowDetail>["workflow"]>;
}) {
	const templateName =
		getWorkflowLabel(workflow.labels, "pipeline-template") ||
		getWorkflowLabel(workflow.labels, "template") ||
		"后端待接入";
	const assetIds =
		getWorkflowLabel(workflow.labels, "asset-ids") ||
		getWorkflowLabel(workflow.labels, "assets") ||
		"后端待接入";

	return (
		<div
			style={{
				display: "grid",
				gridTemplateColumns: "repeat(auto-fit, minmax(260px, 1fr))",
				gap: 12,
				padding: "12px 16px",
				borderBottom: "1px solid #e5e7eb",
				background: "#fff",
			}}
		>
			<Card size="small" title="Pipeline Template">
				<Typography.Text>{templateName}</Typography.Text>
			</Card>
			<Card size="small" title="关联资产">
				<Typography.Text>{assetIds}</Typography.Text>
			</Card>
			<Card size="small" title="运行事件">
				<Typography.Text type="secondary">
					run_events 接口待接入，后续展示提交、调度、Pod
					创建、节点状态变更和重试事件。
				</Typography.Text>
			</Card>
		</div>
	);
}

function WorkflowAssetNodePanel() {
	return (
		<div
			style={{
				margin: "0 16px 12px",
				padding: "8px 12px",
				border: "1px dashed #cbd5e1",
				borderRadius: 8,
				background: "#f8fafc",
				color: "#64748b",
				fontSize: 12,
				display: "flex",
				alignItems: "center",
				justifyContent: "space-between",
				gap: 12,
			}}
		>
			<strong style={{ color: "#334155" }}>资产 × 节点明细</strong>
			<span>后端接入后展示每个资产在每个步骤的状态、日志和资源信息。</span>
		</div>
	);
}

export default function WorkflowDetailPage({
	legacyRoute = false,
}: {
	legacyRoute?: boolean;
}) {
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
				message.success(`${operation.title}已提交`);
				if (operation.key === "delete") {
					navigate("/workflows");
					return;
				}
				loadWorkflow();
			} catch (err) {
				message.error(`${operation.title}失败: ${String(err)}`);
			} finally {
				setOperationLoading(null);
			}
		},
		[loadWorkflow, navigate, workflow],
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

	const handleInspectNodeFromSummary = useCallback(
		(node: WorkflowNodeStatus) => {
			setNodeDetailTab("summary");
			selectNode(node);
			setShowNodeLogs(false);
			setNodePanelOpen(true);
		},
		[selectNode],
	);

	const handleOpenNodeLogsFromSummary = useCallback(
		(node: WorkflowNodeStatus) => {
			setNodeDetailTab("logs");
			selectNode(node);
			setNodePanelOpen(false);
			setShowNodeLogs(true);
		},
		[selectNode],
	);

	const handleOpenNodeRuntimeFromSummary = useCallback(
		(node: WorkflowNodeStatus) => {
			setNodeDetailTab("runtime");
			selectNode(node);
			setShowNodeLogs(false);
			setNodePanelOpen(true);
		},
		[selectNode],
	);

	const handleShowNodeLogs = useCallback(() => {
		if (!selectedNode) {
			return;
		}
		setShowNodeLogs(true);
	}, [selectedNode]);

	const handleCloseNodeLogs = useCallback(() => {
		setShowNodeLogs(false);
		stopFollowLogs();
	}, [stopFollowLogs]);

	const handleRetryWorkflow = useCallback(() => {
		if (!workflow) return;
		const retryConfig = operations.find(
			(operation) => operation.key === "retry",
		);
		if (!retryConfig || retryConfig.disabled) {
			message.warning("当前工作流状态不可重试");
			return;
		}
		runOperation(retryConfig);
	}, [operations, runOperation, workflow]);

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

	return (
		<div
			style={{
				height: "calc(100vh - 49px)",
				display: "flex",
				flexDirection: "column",
			}}
		>
			<div
				style={{
					padding: "12px 24px",
					borderBottom: "1px solid #e5e7eb",
					display: "flex",
					alignItems: "center",
					gap: 12,
				}}
			>
				<Button
					icon={<ArrowLeftOutlined />}
					onClick={() => navigate("/pipeline?tab=executions")}
				>
					返回
				</Button>
				<h3 style={{ margin: 0 }}>{workflow.name}</h3>
				<Tag color={STATUS_COLORS[workflow.status] || "default"}>
					{workflow.status}
				</Tag>
				{workflow.message ? (
					<Tooltip title={workflow.message}>
						<span
							style={{
								color: "#dc2626",
								fontSize: 12,
								maxWidth: 420,
								overflow: "hidden",
								textOverflow: "ellipsis",
								whiteSpace: "nowrap",
							}}
						>
							{workflow.message}
						</span>
					</Tooltip>
				) : null}
				<span style={{ color: "#6b7280", fontSize: 12 }}>
					创建: {new Date(workflow.createdAt).toLocaleString()}
					{workflow.finishedAt &&
						` | 完成: ${new Date(workflow.finishedAt).toLocaleString()}`}
					{" | 耗时: "}
					<DurationPanel
						phase={workflow.status}
						startedAt={workflow.createdAt}
						finishedAt={workflow.finishedAt}
						progress={workflow.progress}
					/>
				</span>
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
			<WorkflowRunContextPanel workflow={workflow} />
			<WorkflowNodeSummaryTable
				nodes={workflow.nodes}
				selectedNodeId={selectedNode?.id ?? null}
				onInspectNode={handleInspectNodeFromSummary}
				onOpenLogs={handleOpenNodeLogsFromSummary}
				onOpenRuntime={handleOpenNodeRuntimeFromSummary}
			/>
			<div style={{ flex: 1, display: "flex", minHeight: 0, minWidth: 0 }}>
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
			<WorkflowAssetNodePanel />

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
					onFollow={startFollowLogs}
					onStop={stopFollowLogs}
					onDownload={downloadLogs}
					onSearch={setLogSearch}
				/>
			</Modal>
		</div>
	);
}
