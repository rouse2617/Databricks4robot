import {
	ApartmentOutlined,
	ArrowLeftOutlined,
	BarsOutlined,
} from "@ant-design/icons";
import Ansi from "ansi-to-react";
import {
	Alert,
	Button,
	Descriptions,
	Input,
	Modal,
	message,
	Segmented,
	Space,
	Spin,
	Tag,
	Tooltip,
} from "antd";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import type { WorkflowNodeStatus } from "../api/workflowApi";
import { DurationPanel } from "../components/common/DurationPanel";
import { LinkifiedText } from "../components/common/LinkifiedText";
import { WorkflowNodeDetailPanel } from "../components/pipeline/WorkflowNodeDetailPanel";
import { STATUS_COLORS } from "../lib/constants";
import {
	getAvailableWorkflowOperationConfigs,
	getWorkflowOperationConfigs,
	type WorkflowOperationConfig,
	type WorkflowOperationKey,
} from "../lib/workflow-operations";
import { getWorkflowNodeDisplayText } from "../lib/workflowNodeDisplay";
import { useWorkflowDetail } from "./useWorkflowDetail";
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
}: {
	selectedNode: WorkflowNodeStatus | null;
	loading: boolean;
	logContent: string | null;
	error: string | null;
	search: string;
	onSearch: (value: string) => void;
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
			<h4 style={{ margin: "0 0 10px 0", fontSize: 14 }}>
				{selectedNode
					? `${getWorkflowNodeDisplayText(selectedNode)} 日志`
					: "日志查看器"}
			</h4>
			<Input.Search
				placeholder="日志关键字搜索"
				value={search}
				onChange={(event) => onSearch(event.target.value)}
				allowClear
				style={{ marginBottom: 12 }}
			/>

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

export default function WorkflowDetailPage() {
	const { name } = useParams<{ name: string }>();
	const navigate = useNavigate();
	const [viewMode, setViewMode] = useState<"dag" | "timeline">("dag");
	const [operationLoading, setOperationLoading] =
		useState<WorkflowOperationKey | null>(null);

	const {
		workflow,
		loading,
		loadError,
		selectedNode,
		selectNode,
		loadWorkflow,
		logState,
		setLogSearch,
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
			if (operation.key === "delete" || operation.key === "terminate") {
				Modal.confirm({
					title: `确认${operation.title} ${workflow?.name}?`,
					okText: operation.title,
					okButtonProps: { danger: operation.danger },
					cancelText: "取消",
					onOk: () => executeOperation(operation),
				});
				return;
			}
			executeOperation(operation);
		},
		[executeOperation, workflow?.name],
	);

	const closeNodeDetailPanel = useCallback(() => {
		setShowNodeLogs(false);
		selectNode(null);
	}, [selectNode]);

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
			message.warning("当前工作流状态不可重试");
			return;
		}
		runOperation(retryConfig);
	}, [operations, runOperation, workflow]);

	useEffect(() => {
		if (!selectedNode) {
			setShowNodeLogs(false);
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
					onClick={() => navigate("/workflows")}
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
			<div style={{ flex: 1, display: "flex", minHeight: 0, minWidth: 0 }}>
				{viewMode === "dag" ? (
					<WorkflowDagView
						nodes={workflow.nodes}
						workflowEdges={workflow.edges}
						selectedNodeId={selectedNode?.id ?? null}
						onNodeSelect={selectNode}
						emptyMessage={workflow.message}
					/>
				) : (
					<WorkflowTimelineView
						nodes={workflow.nodes}
						selectedNodeId={selectedNode?.id ?? null}
						onNodeSelect={selectNode}
					/>
				)}
			</div>

			<WorkflowNodeDetailPanel
				node={selectedNode}
				workflow={workflow}
				open={!!selectedNode}
				onClose={closeNodeDetailPanel}
				canRetryWorkflow={canRetryWorkflow}
				onRetryWorkflow={handleRetryWorkflow}
				onShowLogs={handleShowNodeLogs}
			/>

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
			>
				<WorkflowLogPanel
					selectedNode={selectedNode}
					loading={logState.loading}
					logContent={logState.content}
					error={logState.error}
					search={logState.search}
					onSearch={setLogSearch}
				/>
			</Modal>
		</div>
	);
}
