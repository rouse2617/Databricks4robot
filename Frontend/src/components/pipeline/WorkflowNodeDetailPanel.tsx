import {
	CloudServerOutlined,
	FileTextOutlined,
	LockOutlined,
	ReloadOutlined,
} from "@ant-design/icons";
import {
	Alert,
	Button,
	Card,
	Col,
	Descriptions,
	Drawer,
	Empty,
	Row,
	Space,
	Statistic,
	Table,
	Tabs,
	Tag,
	Tooltip,
	Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import dayjs from "dayjs";
import type React from "react";
import { useEffect, useMemo, useReducer, useRef, useState } from "react";
import type {
	WorkflowDetail,
	WorkflowNodeContainer,
	WorkflowNodeStatus,
	WorkflowPodCondition,
	WorkflowPodCost,
	WorkflowPodEvent,
	WorkflowPodMetrics,
} from "../../api/workflowApi";
import {
	createTerminalSession,
	getNodePodDiagnostics,
	getTerminalAttachUrl,
	type NodePodDiagnostics,
	type TerminalSession,
	terminateTerminalSession,
} from "../../api/workflowApi";
import { STATUS_COLORS } from "../../lib/constants";
import {
	getWorkflowNodePodName,
	truncateMiddle,
} from "../../lib/workflowNodeDisplay";
import { DurationPanel } from "../common/DurationPanel";

type KeyValue = { name: string; value?: string };
type Artifact = { name: string; path?: string };
export type WorkflowNodeDetailTabKey = "summary" | "logs" | "runtime" | "io";

const keyValueColumns: ColumnsType<{
	key: string;
	name: string;
	value: string;
}> = [
	{ title: "名称", dataIndex: "name", key: "name" },
	{ title: "值", dataIndex: "value", key: "value" },
];

const artifactColumns: ColumnsType<{
	key: string;
	name: string;
	path: string;
}> = [
	{ title: "名称", dataIndex: "name", key: "name" },
	{
		title: "路径",
		dataIndex: "path",
		key: "path",
		render: (path?: string) => {
			if (!path) return "—";
			if (!path.startsWith("http://") && !path.startsWith("https://")) {
				return <Typography.Text>{path}</Typography.Text>;
			}
			return (
				<a href={path} target="_blank" rel="noreferrer">
					下载
				</a>
			);
		},
	},
];

const containerColumns: ColumnsType<{
	key: string;
	name: string;
	image?: string;
	command?: string[];
	args?: string[];
	ready?: boolean;
	restartCount?: number;
	state?: string;
}> = [
	{ title: "名称", dataIndex: "name", key: "name" },
	{ title: "镜像", dataIndex: "image", key: "image" },
	{
		title: "状态",
		dataIndex: "state",
		key: "state",
		render: (state?: string, record?: { ready?: boolean }) => (
			<Tag color={record?.ready ? "green" : state ? "blue" : "default"}>
				{state || (record?.ready ? "Ready" : "—")}
			</Tag>
		),
	},
	{
		title: "重启",
		dataIndex: "restartCount",
		key: "restartCount",
		render: (value?: number) => value ?? "—",
	},
	{
		title: "命令",
		dataIndex: "command",
		key: "command",
		render: (command?: string[]) => command?.join(" ") || "—",
	},
	{
		title: "参数",
		dataIndex: "args",
		key: "args",
		render: (args?: string[]) => args?.join(" ") || "—",
	},
];

const podConditionColumns: ColumnsType<WorkflowPodCondition & { key: string }> =
	[
		{ title: "类型", dataIndex: "type", key: "type" },
		{
			title: "状态",
			dataIndex: "status",
			key: "status",
			render: (status: string) => (
				<Tag color={status === "True" ? "green" : "default"}>{status}</Tag>
			),
		},
		{ title: "原因", dataIndex: "reason", key: "reason" },
		{
			title: "消息",
			dataIndex: "message",
			key: "message",
			render: (message?: string) => message || "—",
		},
	];

const podEventColumns: ColumnsType<WorkflowPodEvent & { key: string }> = [
	{
		title: "级别",
		dataIndex: "type",
		key: "type",
		render: (type: string) => (
			<Tag color={type === "Warning" ? "orange" : "blue"}>{type}</Tag>
		),
	},
	{ title: "原因", dataIndex: "reason", key: "reason" },
	{
		title: "次数",
		dataIndex: "count",
		key: "count",
		render: (count?: number) => count ?? "—",
	},
	{
		title: "消息",
		dataIndex: "message",
		key: "message",
		render: (message: string) => (
			<Typography.Text
				style={{ maxWidth: 260 }}
				ellipsis={{ tooltip: message }}
			>
				{message}
			</Typography.Text>
		),
	},
];

function formatRelativeTime(value?: string) {
	if (!value || !dayjs(value).isValid()) return "—";
	return `${dayjs(value).format("YYYY-MM-DD HH:mm:ss")} (${dayjs(value).fromNow()})`;
}

function formatResourceDurationText(
	resourcesDuration?: Record<string, number>,
) {
	if (!resourcesDuration) return "—";
	const rows = Object.entries(resourcesDuration);
	if (rows.length === 0) return "—";
	return rows.map(([name, value]) => `${name}: ${value}`).join("\n");
}

function formatKvRows(items?: KeyValue[]) {
	return (items ?? []).map((item, index) => ({
		key: `${item.name || "param"}-${index}`,
		name: item.name,
		value: item.value ?? "",
	}));
}

function formatArtifactRows(items?: Artifact[]) {
	return (items ?? []).map((artifact, index) => ({
		key: `${artifact.name || "artifact"}-${index}`,
		name: artifact.name,
		path: artifact.path || "",
	}));
}

function formatBytes(value?: number) {
	if (typeof value !== "number" || Number.isNaN(value)) return "—";
	const units = ["B", "KiB", "MiB", "GiB", "TiB"];
	let next = value;
	let unitIndex = 0;
	while (next >= 1024 && unitIndex < units.length - 1) {
		next /= 1024;
		unitIndex += 1;
	}
	return `${next.toFixed(next >= 10 || unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
}

function formatCost(value?: number) {
	if (typeof value !== "number" || Number.isNaN(value)) return "—";
	return `$${value.toFixed(value >= 1 ? 2 : 4)}`;
}

function getMetricPercent(used?: number, limit?: number) {
	if (!used || !limit || limit <= 0) return undefined;
	return Math.min(100, Math.round((used / limit) * 100));
}

function CopyableEllipsisText({
	text,
	maxLength = 40,
}: {
	text: string;
	maxLength?: number;
}) {
	const display = truncateMiddle(text, maxLength);
	return (
		<Tooltip title={text}>
			<Typography.Text copyable={{ text }}>{display}</Typography.Text>
		</Tooltip>
	);
}

function ContainersTab({ node }: { node: WorkflowNodeStatus }) {
	const containerData = useMemo(() => {
		const raw: WorkflowNodeContainer[] = node.containers ?? [];
		return raw.map((container, index) => ({
			key: `${container.name || "container"}-${index}`,
			...container,
		}));
	}, [node]);

	if (containerData.length === 0) {
		return (
			<Empty
				description="暂无容器详情"
				styles={{ description: { maxWidth: 360, margin: "0 auto" } }}
			>
				<Typography.Text type="secondary" style={{ fontSize: 12 }}>
					当前 API 未返回容器
					spec；可在「概览」查看节点状态，或通过「日志」排查运行详情。
				</Typography.Text>
			</Empty>
		);
	}

	return (
		<Table
			size="small"
			dataSource={containerData}
			columns={containerColumns}
			pagination={false}
			rowKey="key"
		/>
	);
}

function PodTab({
	node,
	workflowName,
}: {
	node: WorkflowNodeStatus;
	workflowName: string;
}) {
	const podName = getWorkflowNodePodName(node);
	const [podDiag, setPodDiag] = useState<NodePodDiagnostics | null>(null);
	const [podDiagLoading, setPodDiagLoading] = useState(false);
	const [podDiagError, setPodDiagError] = useState<string | null>(null);
	const [refreshTrigger, refreshPodDiagnostics] = useReducer(
		(count) => count + 1,
		0,
	);

	useEffect(() => {
		if (!workflowName || !node.id) return undefined;
		let cancelled = false;
		const requestSeq = refreshTrigger;
		setPodDiagLoading(true);
		setPodDiagError(null);
		getNodePodDiagnostics(workflowName, node.id)
			.then((data) => {
				if (!cancelled && requestSeq === refreshTrigger) setPodDiag(data);
			})
			.catch((err) => {
				if (!cancelled && requestSeq === refreshTrigger) {
					setPodDiag(null);
					setPodDiagError(err instanceof Error ? err.message : String(err));
				}
			})
			.finally(() => {
				if (!cancelled && requestSeq === refreshTrigger)
					setPodDiagLoading(false);
			});
		return () => {
			cancelled = true;
		};
	}, [workflowName, node.id, refreshTrigger]);

	const cluster = podDiag?.cluster ?? node.cluster;
	const namespace = podDiag?.namespace ?? node.namespace;
	const displayPodName = podDiag?.podName ?? podName;
	const podIp = podDiag?.podIp ?? node.podIp;
	const serviceAccountName =
		podDiag?.serviceAccountName ?? node.serviceAccountName;
	const restartCount = podDiag?.restartCount ?? node.restartCount;
	const containers = podDiag?.containers ?? node.containers ?? [];
	const conditions = podDiag?.podConditions ?? node.podConditions ?? [];
	const events = podDiag?.podEvents ?? node.podEvents ?? [];

	const conditionRows = conditions.map((condition, index) => ({
		key: `${condition.type}-${index}`,
		...condition,
	}));
	const eventRows = events.map((event, index) => ({
		key: `${event.reason}-${index}`,
		...event,
	}));

	return (
		<Space direction="vertical" size="middle" style={{ width: "100%" }}>
			{podDiagError ? (
				<Alert
					type="warning"
					showIcon
					message="Pod 诊断数据不可用"
					description="已保留 Argo 节点元数据；请检查 K8s API 连通性、RBAC 权限，或等待 Pod 创建完成。"
					action={
						<Button
							size="small"
							icon={<ReloadOutlined />}
							loading={podDiagLoading}
							onClick={refreshPodDiagnostics}
						>
							重试
						</Button>
					}
				/>
			) : podDiagLoading ? (
				<Alert type="info" showIcon message="正在加载 Pod 诊断数据" />
			) : podDiag ? (
				<Alert type="success" showIcon message="Pod 诊断数据已加载" />
			) : null}
			<Descriptions size="small" column={1} layout="vertical" bordered>
				<Descriptions.Item label="Cluster">{cluster || "—"}</Descriptions.Item>
				<Descriptions.Item label="Namespace">
					{namespace || "—"}
				</Descriptions.Item>
				<Descriptions.Item label="Pod">
					{displayPodName ? (
						<CopyableEllipsisText text={displayPodName} />
					) : (
						"—"
					)}
				</Descriptions.Item>
				<Descriptions.Item label="Pod IP">{podIp || "—"}</Descriptions.Item>
				<Descriptions.Item label="Service Account">
					{serviceAccountName || "—"}
				</Descriptions.Item>
				<Descriptions.Item label="重启次数">
					{restartCount ?? "—"}
				</Descriptions.Item>
			</Descriptions>
			<div>
				<div style={{ marginBottom: 8, fontWeight: 600 }}>Containers</div>
				<ContainersTab node={{ ...node, containers }} />
			</div>
			<div>
				<div style={{ marginBottom: 8, fontWeight: 600 }}>Conditions</div>
				{conditionRows.length === 0 ? (
					<Empty description="暂无 Pod conditions" />
				) : (
					<Table
						size="small"
						dataSource={conditionRows}
						columns={podConditionColumns}
						pagination={false}
						rowKey="key"
					/>
				)}
			</div>
			<div>
				<div style={{ marginBottom: 8, fontWeight: 600 }}>Events</div>
				{eventRows.length === 0 ? (
					<Empty description="暂无 Pod events" />
				) : (
					<Table
						size="small"
						dataSource={eventRows}
						columns={podEventColumns}
						pagination={false}
						rowKey="key"
					/>
				)}
			</div>
		</Space>
	);
}

function MonitoringTab({ metrics }: { metrics?: WorkflowPodMetrics }) {
	const cpuPercent = getMetricPercent(
		metrics?.cpuCores,
		metrics?.cpuLimitCores || metrics?.cpuRequestCores,
	);
	const memoryPercent = getMetricPercent(
		metrics?.memoryBytes,
		metrics?.memoryLimitBytes || metrics?.memoryRequestBytes,
	);

	return (
		<Space direction="vertical" size="middle" style={{ width: "100%" }}>
			<Alert
				type={metrics ? "success" : "info"}
				showIcon
				message={metrics ? "监控快照" : "暂无监控数据"}
				description={
					metrics
						? `采样时间：${metrics.sampledAt ? formatRelativeTime(metrics.sampledAt) : "—"}`
						: "当前运行未返回 CPU、内存、GPU、网络或存储指标。"
				}
			/>
			<Row gutter={[12, 12]}>
				<Col span={12}>
					<Card size="small">
						<Statistic
							title="CPU"
							value={metrics?.cpuCores ?? "—"}
							suffix={typeof metrics?.cpuCores === "number" ? "cores" : ""}
						/>
						<Typography.Text type="secondary">
							{cpuPercent === undefined
								? "request / limit: —"
								: `用量 ${cpuPercent}%`}
						</Typography.Text>
					</Card>
				</Col>
				<Col span={12}>
					<Card size="small">
						<Statistic
							title="Memory"
							value={formatBytes(metrics?.memoryBytes)}
						/>
						<Typography.Text type="secondary">
							{memoryPercent === undefined
								? "request / limit: —"
								: `用量 ${memoryPercent}%`}
						</Typography.Text>
					</Card>
				</Col>
				<Col span={12}>
					<Card size="small">
						<Statistic title="GPU" value={metrics?.gpuCount ?? "—"} />
					</Card>
				</Col>
				<Col span={12}>
					<Card size="small">
						<Statistic
							title="Network"
							value={`${formatBytes(metrics?.networkRxBytes)} / ${formatBytes(metrics?.networkTxBytes)}`}
						/>
						<Typography.Text type="secondary">RX / TX</Typography.Text>
					</Card>
				</Col>
			</Row>
		</Space>
	);
}

function BillingTab({ cost }: { cost?: WorkflowPodCost }) {
	return (
		<Space direction="vertical" size="middle" style={{ width: "100%" }}>
			<Alert
				type={cost ? "success" : "info"}
				showIcon
				message={cost ? "计费快照" : "暂无计费数据"}
				description={
					cost
						? `窗口：${cost.window || "—"} · 来源：${cost.provider || "—"}`
						: "当前运行未返回该步骤的成本拆分。"
				}
			/>
			<Row gutter={[12, 12]}>
				<Col span={12}>
					<Card size="small">
						<Statistic title="总成本" value={formatCost(cost?.totalCostUsd)} />
					</Card>
				</Col>
				<Col span={12}>
					<Card size="small">
						<Statistic title="CPU" value={formatCost(cost?.cpuCostUsd)} />
					</Card>
				</Col>
				<Col span={12}>
					<Card size="small">
						<Statistic title="Memory" value={formatCost(cost?.memoryCostUsd)} />
					</Card>
				</Col>
				<Col span={12}>
					<Card size="small">
						<Statistic title="GPU" value={formatCost(cost?.gpuCostUsd)} />
					</Card>
				</Col>
				<Col span={12}>
					<Card size="small">
						<Statistic
							title="Storage"
							value={formatCost(cost?.storageCostUsd)}
						/>
					</Card>
				</Col>
				<Col span={12}>
					<Card size="small">
						<Statistic
							title="Network"
							value={formatCost(cost?.networkCostUsd)}
						/>
					</Card>
				</Col>
			</Row>
		</Space>
	);
}

type TerminalFrame =
	| { type: "stdout" | "stderr"; data?: string }
	| { type: "status"; status?: string }
	| { type: "exit"; exitCode?: number; reason?: string }
	| { type: "error"; code?: string; message?: string };

function terminalFrameText(frame: TerminalFrame): string {
	switch (frame.type) {
		case "stdout":
		case "stderr":
			return frame.data || "";
		case "status":
			return `\n[status] ${frame.status || "connected"}\n`;
		case "exit":
			return `\n[exit] ${frame.reason || "closed"}${typeof frame.exitCode === "number" ? ` (${frame.exitCode})` : ""}\n`;
		case "error":
			return `\n[error] ${frame.code || "POD_EXEC_ERROR"} ${frame.message || ""}\n`;
		default:
			return "";
	}
}

function DebugTab({
	node,
	workflowName,
}: {
	node: WorkflowNodeStatus;
	workflowName: string;
}) {
	const execEnabled = node.debug?.execEnabled === true;
	const commandTemplates =
		node.debug?.allowedCommands && node.debug.allowedCommands.length > 0
			? node.debug.allowedCommands
			: ["sh", "pwd", "ls", "env"];
	const [selectedCommand, setSelectedCommand] = useState(commandTemplates[0]);
	const [session, setSession] = useState<TerminalSession | null>(null);
	const [starting, setStarting] = useState(false);
	const [terminating, setTerminating] = useState(false);
	const [terminalText, setTerminalText] = useState("");
	const [terminalError, setTerminalError] = useState<string | null>(null);
	const socketRef = useRef<WebSocket | null>(null);

	useEffect(() => {
		if (!commandTemplates.includes(selectedCommand)) {
			setSelectedCommand(commandTemplates[0] || "sh");
		}
	}, [commandTemplates, selectedCommand]);

	useEffect(() => {
		return () => {
			socketRef.current?.close();
			socketRef.current = null;
		};
	}, []);

	const appendTerminalText = (text: string) => {
		setTerminalText((prev) => `${prev}${text}`);
	};

	const startSession = async (command?: string) => {
		const cmd = command ?? selectedCommand;
		if (!execEnabled || !workflowName || !node.id) return;
		if (command) setSelectedCommand(command);
		socketRef.current?.close();
		socketRef.current = null;
		setStarting(true);
		setTerminalError(null);
		setTerminalText(`$ ${cmd}\n`);
		try {
			const created = await createTerminalSession(workflowName, node.id, {
				command: cmd,
			});
			setSession(created);
			if (!created.attachUrl) {
				appendTerminalText("[error] 后端未返回终端连接地址\n");
				return;
			}
			const socket = new WebSocket(getTerminalAttachUrl(created.attachUrl));
			socketRef.current = socket;
			socket.onopen = () => appendTerminalText("[status] connected\n");
			socket.onmessage = (event) => {
				try {
					const frame = JSON.parse(String(event.data)) as TerminalFrame;
					appendTerminalText(terminalFrameText(frame));
				} catch {
					appendTerminalText(String(event.data));
				}
			};
			socket.onerror = () => {
				setTerminalError("终端连接失败，请检查后端 exec 配置或稍后重试。");
			};
			socket.onclose = () => appendTerminalText("[status] disconnected\n");
		} catch (err) {
			const message = err instanceof Error ? err.message : String(err);
			setTerminalError(message);
			appendTerminalText(`[error] ${message}\n`);
		} finally {
			setStarting(false);
		}
	};

	const terminateSession = async () => {
		if (!session) return;
		setTerminating(true);
		try {
			socketRef.current?.close();
			socketRef.current = null;
			const updated = await terminateTerminalSession(session.id);
			setSession(updated);
			appendTerminalText("[status] terminated\n");
		} catch (err) {
			setTerminalError(err instanceof Error ? err.message : String(err));
		} finally {
			setTerminating(false);
		}
	};

	if (!execEnabled) {
		return (
			<Alert
				type="info"
				showIcon
				message="当前执行目标未开启 Pod 终端"
				description="可在 设置 → 执行目标 中开启 Pod 终端 policy"
			/>
		);
	}

	return (
		<Space direction="vertical" size="middle" style={{ width: "100%" }}>
			<Alert
				type="warning"
				showIcon
				message="Pod 终端可用"
				description={node.debug?.reason || ""}
			/>
			{terminalError ? (
				<Alert
					type="error"
					showIcon
					message="终端会话失败"
					description={terminalError}
				/>
			) : null}
			<Space wrap>
				{commandTemplates.map((command) => (
					<Button
						key={command}
						size="small"
						type={selectedCommand === command ? "primary" : "default"}
						disabled={starting}
						onClick={() => startSession(command)}
					>
						{command}
					</Button>
				))}
				<Button
					type="primary"
					size="small"
					loading={starting}
					onClick={() => startSession()}
				>
					打开终端
				</Button>
				<Button
					size="small"
					disabled={!session || session.status === "terminated"}
					loading={terminating}
					onClick={terminateSession}
				>
					终止
				</Button>
			</Space>
			{session ? (
				<Typography.Text type="secondary" style={{ fontSize: 12 }}>
					会话 {session.id.slice(0, 8)} · Pod {session.podName} ·{" "}
					{session.command} · 状态 {session.status}
				</Typography.Text>
			) : null}
			<div
				style={{
					height: 220,
					background: "#111827",
					color: "#d1d5db",
					borderRadius: 6,
					padding: 12,
					fontFamily: '"SF Mono", "Fira Code", monospace',
					fontSize: 12,
					whiteSpace: "pre-wrap",
					overflow: "auto",
				}}
			>
				{terminalText ? (
					terminalText
				) : (
					<Space
						direction="vertical"
						align="center"
						style={{
							width: "100%",
							height: "100%",
							justifyContent: "center",
							textAlign: "center",
						}}
					>
						<LockOutlined style={{ fontSize: 22 }} />
						<span>点击命令按钮打开终端</span>
					</Space>
				)}
			</div>
		</Space>
	);
}

function LogsTab({
	node,
	onShowLogs,
}: {
	node: WorkflowNodeStatus;
	onShowLogs: () => void;
}) {
	return (
		<Space direction="vertical" size="middle" style={{ width: "100%" }}>
			<Alert
				type={
					node.phase === "Failed" || node.phase === "Error" ? "error" : "info"
				}
				showIcon
				message={
					node.phase === "Failed" || node.phase === "Error"
						? "优先查看该步骤日志"
						: "查看该步骤日志"
				}
				description={
					node.message ||
					"日志会按当前步骤过滤展示；大日志会限制渲染尾部内容，后续可接入 tail、分页和流式 follow。"
				}
				action={
					<Button type="primary" size="small" onClick={onShowLogs}>
						打开日志查看器
					</Button>
				}
			/>
			<Descriptions size="small" column={1} layout="vertical" bordered>
				<Descriptions.Item label="步骤">
					{node.displayName || node.name}
				</Descriptions.Item>
				<Descriptions.Item label="Pod">
					{getWorkflowNodePodName(node) || "—"}
				</Descriptions.Item>
				<Descriptions.Item label="状态">
					<Tag color={STATUS_COLORS[node.phase] || "default"}>{node.phase}</Tag>
				</Descriptions.Item>
			</Descriptions>
		</Space>
	);
}

function RuntimeSection({
	title,
	children,
}: {
	title: string;
	children: React.ReactNode;
}) {
	return (
		<section>
			<div style={{ marginBottom: 8, fontWeight: 600 }}>{title}</div>
			{children}
		</section>
	);
}

function RuntimeTab({
	node,
	workflowName,
}: {
	node: WorkflowNodeStatus;
	workflowName: string;
}) {
	return (
		<Space direction="vertical" size="middle" style={{ width: "100%" }}>
			<RuntimeSection title="Pod 与事件">
				<PodTab node={node} workflowName={workflowName} />
			</RuntimeSection>
			<RuntimeSection title="监控">
				<MonitoringTab metrics={node.metrics} />
			</RuntimeSection>
			<RuntimeSection title="计费">
				<BillingTab cost={node.cost} />
			</RuntimeSection>
			<RuntimeSection title="调试">
				<Alert type="info" showIcon message="终端调试" description="终端调试已移至节点卡片。在 DAG 上选择一个节点，即可找到终端入口。" />
			</RuntimeSection>
		</Space>
	);
}

function InputOutputTab({ node }: { node: WorkflowNodeStatus }) {
	return (
		<Space direction="vertical" style={{ width: "100%" }}>
			<InputsTab node={node} />
			<OutputsTab node={node} />
		</Space>
	);
}

function InputsTab({ node }: { node: WorkflowNodeStatus }) {
	const parameters = formatKvRows(node.inputs?.parameters);
	const artifacts = formatArtifactRows(node.inputs?.artifacts);

	return (
		<Space direction="vertical" size="middle" style={{ width: "100%" }}>
			<div>
				<div style={{ marginBottom: 8, fontWeight: 600 }}>输入参数</div>
				{parameters.length === 0 ? (
					<Empty description="暂无输入参数" />
				) : (
					<Table
						size="small"
						dataSource={parameters}
						columns={keyValueColumns}
						pagination={false}
						rowKey="key"
					/>
				)}
			</div>
			<div>
				<div style={{ marginBottom: 8, fontWeight: 600 }}>输入产物</div>
				{artifacts.length === 0 ? (
					<Empty description="暂无输入产物" />
				) : (
					<Table
						size="small"
						dataSource={artifacts}
						columns={artifactColumns}
						pagination={false}
						rowKey="key"
					/>
				)}
			</div>
		</Space>
	);
}

function OutputsTab({ node }: { node: WorkflowNodeStatus }) {
	const parameters = formatKvRows(node.outputs?.parameters);
	const artifacts = formatArtifactRows(node.outputs?.artifacts);

	return (
		<Space direction="vertical" size="middle" style={{ width: "100%" }}>
			<Row gutter={[12, 12]}>
				<Col span={24}>
					<Card size="small" title="结果">
						<Typography.Paragraph>
							<pre
								style={{
									margin: 0,
									whiteSpace: "pre-wrap",
									wordBreak: "break-all",
								}}
							>
								{node.outputs?.result || "—"}
							</pre>
						</Typography.Paragraph>
					</Card>
				</Col>
				<Col span={24}>
					<Card size="small" title="退出码">
						{typeof node.outputs?.exitCode === "number" ||
						typeof node.outputs?.exitCode === "string"
							? node.outputs.exitCode
							: "—"}
					</Card>
				</Col>
			</Row>
			<div>
				<div style={{ marginBottom: 8, fontWeight: 600 }}>输出参数</div>
				{parameters.length === 0 ? (
					<Empty description="暂无输出参数" />
				) : (
					<Table
						size="small"
						dataSource={parameters}
						columns={keyValueColumns}
						pagination={false}
						rowKey="key"
					/>
				)}
			</div>
			<div>
				<div style={{ marginBottom: 8, fontWeight: 600 }}>输出产物</div>
				{artifacts.length === 0 ? (
					<Empty description="暂无输出产物" />
				) : (
					<Table
						size="small"
						dataSource={artifacts}
						columns={artifactColumns}
						pagination={false}
						rowKey="key"
					/>
				)}
			</div>
		</Space>
	);
}

function SummaryTab({
	node,
	workflow,
}: {
	node: WorkflowNodeStatus;
	workflow: WorkflowDetail;
}) {
	const podName = getWorkflowNodePodName(node);
	const memoizationText = node.memoizationStatus
		? `命中=${node.memoizationStatus.hit ? "是" : "否"}，key=${node.memoizationStatus.key}，cache=${node.memoizationStatus.cacheName}`
		: "—";

	return (
		<Space direction="vertical" size="middle" style={{ width: "100%" }}>
			<Descriptions size="small" column={1} layout="vertical" bordered>
				<Descriptions.Item label="名称">
					<Typography.Text copyable={{ text: node.displayName || node.name }}>
						{node.displayName || node.name}
					</Typography.Text>
				</Descriptions.Item>
				<Descriptions.Item label="节点 ID">
					<CopyableEllipsisText text={node.id} />
				</Descriptions.Item>
				<Descriptions.Item label="Pod 名称">
					{podName ? <CopyableEllipsisText text={podName} /> : "—"}
				</Descriptions.Item>
				<Descriptions.Item label="宿主机">
					{node.hostNodeName ? (
						<CopyableEllipsisText text={node.hostNodeName} maxLength={32} />
					) : (
						"—"
					)}
				</Descriptions.Item>
				<Descriptions.Item label="类型">
					{node.type || node.templateName || "—"}
				</Descriptions.Item>
				<Descriptions.Item label="状态">
					<Tag color={STATUS_COLORS[node.phase] || "default"}>{node.phase}</Tag>
				</Descriptions.Item>
				<Descriptions.Item label="开始时间">
					{formatRelativeTime(node.startedAt)}
				</Descriptions.Item>
				<Descriptions.Item label="结束时间">
					{formatRelativeTime(node.finishedAt)}
				</Descriptions.Item>
				<Descriptions.Item label="耗时">
					<DurationPanel
						phase={node.phase}
						startedAt={node.startedAt}
						finishedAt={node.finishedAt}
						progress={node.progress}
					/>
				</Descriptions.Item>
				<Descriptions.Item label="所属工作流">
					<CopyableEllipsisText text={workflow.name} maxLength={36} />
				</Descriptions.Item>
				<Descriptions.Item label="进度">
					{node.progress || workflow.progress || "—"}
				</Descriptions.Item>
				<Descriptions.Item label="Memoization">
					{memoizationText}
				</Descriptions.Item>
				<Descriptions.Item label="资源耗时">
					<pre style={{ margin: 0 }}>
						{formatResourceDurationText(node.resourcesDuration)}
					</pre>
				</Descriptions.Item>
				{node.message ? (
					<Descriptions.Item label="消息">
						<Typography.Paragraph
							style={{ marginBottom: 0, whiteSpace: "pre-wrap" }}
						>
							{node.message}
						</Typography.Paragraph>
					</Descriptions.Item>
				) : null}
			</Descriptions>
		</Space>
	);
}

export function WorkflowNodeDetailPanel({
	node,
	workflow,
	open,
	onClose,
	onRetryWorkflow,
	canRetryWorkflow,
	onShowLogs,
	activeTab = "summary",
	onActiveTabChange,
}: {
	node: WorkflowNodeStatus | null;
	workflow: WorkflowDetail | null;
	open: boolean;
	onClose: () => void;
	onRetryWorkflow?: () => void;
	canRetryWorkflow?: boolean;
	onShowLogs: () => void;
	activeTab?: WorkflowNodeDetailTabKey;
	onActiveTabChange?: (key: WorkflowNodeDetailTabKey) => void;
}) {
	if (!node || !workflow) return null;

	const drawerExtra = [
		<Button
			key="logs"
			type="primary"
			icon={<FileTextOutlined />}
			onClick={onShowLogs}
		>
			日志
		</Button>,
	];
	if (canRetryWorkflow && onRetryWorkflow) {
		drawerExtra.unshift(
			<Button key="retry" icon={<ReloadOutlined />} onClick={onRetryWorkflow}>
				重试工作流
			</Button>,
		);
	}

	return (
		<Drawer
			title={node.displayName || node.templateName || node.name}
			placement="right"
			width={520}
			open={open}
			onClose={onClose}
			extra={drawerExtra}
			styles={{ body: { paddingTop: 8 } }}
		>
			<Tabs
				type="card"
				activeKey={onActiveTabChange ? activeTab : undefined}
				defaultActiveKey={activeTab}
				onChange={(key) => onActiveTabChange?.(key as WorkflowNodeDetailTabKey)}
				items={[
					{
						key: "summary",
						label: "概览",
						children: <SummaryTab node={node} workflow={workflow} />,
					},
					{
						key: "logs",
						label: (
							<>
								<FileTextOutlined /> 日志
							</>
						),
						children: <LogsTab node={node} onShowLogs={onShowLogs} />,
					},
					{
						key: "runtime",
						label: (
							<>
								<CloudServerOutlined /> 运行环境
							</>
						),
						children: <RuntimeTab node={node} workflowName={workflow.name} />,
					},
					{
						key: "io",
						label: "输入/输出",
						children: <InputOutputTab node={node} />,
					},
				]}
			/>
		</Drawer>
	);
}
