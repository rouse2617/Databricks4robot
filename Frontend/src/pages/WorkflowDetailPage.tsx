import {
	ApartmentOutlined,
	ArrowLeftOutlined,
	BarsOutlined,
} from "@ant-design/icons";
import Ansi from "ansi-to-react";
import {
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
import { PHASE_COLORS, STATUS_COLORS } from "../lib/constants";
import {
	getWorkflowOperationConfigs,
	type WorkflowOperationConfig,
	type WorkflowOperationKey,
} from "../lib/workflow-operations";
import { useWorkflowDetail } from "./useWorkflowDetail";
import { WorkflowDagView } from "./WorkflowDagView";

function getNodeDisplayText(node: WorkflowNodeStatus): string {
	return node.displayName || node.templateName || node.name;
}

function TimelineView({ nodes }: { nodes: WorkflowNodeStatus[] }) {
	const withTime = nodes.filter((n) => n.startedAt);
	if (withTime.length === 0) {
		return (
			<div style={{ padding: 40, textAlign: "center", color: "#6b7280" }}>
				暂无节点时间数据
			</div>
		);
	}

	const times = withTime.map((n) => ({
		start: new Date(n.startedAt as string).getTime(),
		end: n.finishedAt ? new Date(n.finishedAt).getTime() : Date.now(),
		id: n.id,
	}));
	const globalStart = Math.min(...times.map((t) => t.start));
	const globalEnd = Math.max(...times.map((t) => t.end));
	const range = globalEnd - globalStart || 1;

	const rowH = 36;
	const headerH = 30;
	const labelW = 180;
	const padL = 16;

	return (
		<div style={{ overflow: "auto", height: "100%", padding: 16 }}>
			<div style={{ minWidth: 600 }}>
				<div
					style={{
						position: "relative",
						height: headerH,
						marginLeft: labelW + padL,
					}}
				>
					{[0, 0.25, 0.5, 0.75, 1].map((pct) => (
						<div
							key={pct}
							style={{
								position: "absolute",
								left: `${pct * 100}%`,
								top: 0,
								fontSize: 11,
								color: "#6b7280",
								transform: "translateX(-50%)",
							}}
						>
							{new Date(globalStart + range * pct).toLocaleTimeString()}
						</div>
					))}
					<div
						style={{
							position: "absolute",
							top: 16,
							left: 0,
							right: 0,
							height: 1,
							background: "#e5e7eb",
						}}
					/>
				</div>
				{times.map((item) => {
					const node = nodes.find((itemNode) => itemNode.id === item.id);
					if (!node) return null;
					const left = ((item.start - globalStart) / range) * 100;
					const width = ((item.end - item.start) / range) * 100;
					return (
						<div
							key={node.id}
							style={{ display: "flex", alignItems: "center", height: rowH }}
						>
							<div
								style={{
									width: labelW,
									fontSize: 12,
									textAlign: "right",
									paddingRight: padL,
									overflow: "hidden",
									textOverflow: "ellipsis",
									whiteSpace: "nowrap",
									color: "#374151",
								}}
								title={node.displayName || node.name}
							>
								{node.displayName || node.name}
							</div>
							<div style={{ flex: 1, position: "relative" }}>
								<div
									style={{
										position: "absolute",
										left: `${left}%`,
										width: `${Math.max(width, 1)}%`,
										top: 4,
										height: rowH - 8,
										borderRadius: 4,
										background: PHASE_COLORS[node.phase] || "#9ca3af",
										opacity: 0.85,
										display: "flex",
										alignItems: "center",
										paddingLeft: 6,
										fontSize: 11,
										color: "#fff",
										overflow: "hidden",
									}}
								>
									{Math.round((item.end - item.start) / 1000)}s
								</div>
							</div>
						</div>
					);
				})}
			</div>
		</div>
	);
}

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
	const logElement =
		logContent === null ? null : buildHighlightedLogNodes(logContent, search);

	useEffect(() => {
		if (
			selectedNode &&
			!loading &&
			!error &&
			logContent !== null &&
			logBodyRef.current
		) {
			logBodyRef.current.scrollTop = logBodyRef.current.scrollHeight;
		}
	}, [selectedNode, loading, error, logContent]);

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
					? `${getNodeDisplayText(selectedNode)} 日志`
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
					点击 DAG 节点查看该节点日志
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
				<div
					ref={logBodyRef}
					style={{
						flex: 1,
						fontSize: 11,
						fontFamily: '"SF Mono", "Fira Code", monospace',
						whiteSpace: "pre-wrap",
						wordBreak: "break-all",
						overflow: "auto",
						background: "#f8f9fa",
						padding: 12,
						borderRadius: 6,
						border: "1px solid #e5e7eb",
						minHeight: 0,
					}}
				>
					{logElement}
				</div>
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

	const handleManifest = useCallback(() => {
		if (!selectedNode) return;
		message.info({
			content: "MANIFEST 功能暂未接入，当前展示为占位信息",
		});
	}, [selectedNode]);

	const handleRetryNode = useCallback(() => {
		if (!selectedNode || !workflow) return;
		const retryConfig = operations.find(
			(operation) => operation.key === "retry",
		);
		if (!retryConfig) {
			message.warning("未找到可用重试动作");
			return;
		}
		if (retryConfig.disabled) {
			message.warning(`当前工作流状态不可执行${retryConfig.title}`);
			return;
		}
		runOperation(retryConfig);
	}, [operations, runOperation, selectedNode, workflow]);

	const handleShowEvents = useCallback(() => {
		if (!selectedNode) return;
		window.open("/events", "_blank", "noopener");
	}, [selectedNode]);

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

	if (!workflow) {
		return <div style={{ padding: 24 }}>未找到工作流</div>;
	}

	return (
		<div
			style={{
				height: "calc(100vh - 64px)",
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
						{operations.map((operation) => (
							<Tooltip
								key={operation.key}
								title={operation.disabled ? "当前状态不可用" : operation.title}
							>
								<Button
									size="small"
									icon={operation.icon}
									danger={operation.danger}
									disabled={operation.disabled}
									loading={operationLoading === operation.key}
									onClick={() => runOperation(operation)}
								>
									{operation.title}
								</Button>
							</Tooltip>
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
			<div style={{ flex: 1, display: "flex", minHeight: 0 }}>
				{viewMode === "dag" ? (
					<WorkflowDagView
						nodes={workflow.nodes}
						workflowEdges={workflow.edges}
						selectedNodeId={selectedNode?.id ?? null}
						onNodeSelect={selectNode}
					/>
				) : (
					<TimelineView nodes={workflow.nodes} />
				)}
			</div>

			<WorkflowNodeDetailPanel
				node={selectedNode}
				workflow={workflow}
				open={!!selectedNode}
				onClose={closeNodeDetailPanel}
				onManifest={handleManifest}
				onRetryNode={handleRetryNode}
				onShowLogs={handleShowNodeLogs}
				onShowEvents={handleShowEvents}
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
