import {
	CloudServerOutlined,
	DollarOutlined,
	FileTextOutlined,
	LockOutlined,
	ReloadOutlined,
	ThunderboltOutlined,
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
import { useMemo } from "react";
import type {
	WorkflowDetail,
	WorkflowNodeContainer,
	WorkflowNodeStatus,
	WorkflowPodCondition,
	WorkflowPodCost,
	WorkflowPodEvent,
	WorkflowPodMetrics,
} from "../../api/workflowApi";
import { STATUS_COLORS } from "../../lib/constants";
import {
	getWorkflowNodePodName,
	truncateMiddle,
} from "../../lib/workflowNodeDisplay";
import { DurationPanel } from "../common/DurationPanel";

type KeyValue = { name: string; value?: string };
type Artifact = { name: string; path?: string };

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

function PodTab({ node }: { node: WorkflowNodeStatus }) {
	const podName = getWorkflowNodePodName(node);
	const conditionRows = (node.podConditions ?? []).map((condition, index) => ({
		key: `${condition.type}-${index}`,
		...condition,
	}));
	const eventRows = (node.podEvents ?? []).map((event, index) => ({
		key: `${event.reason}-${index}`,
		...event,
	}));

	return (
		<Space direction="vertical" size="middle" style={{ width: "100%" }}>
			<Alert
				type="info"
				showIcon
				message="Pod 诊断接口待接入"
				description="前端已预留 Pod describe、conditions、events、container 状态和 namespace/cluster 字段；后端接 Kubernetes API 后即可填充。"
			/>
			<Descriptions size="small" column={1} layout="vertical" bordered>
				<Descriptions.Item label="Cluster">
					{node.cluster || "—"}
				</Descriptions.Item>
				<Descriptions.Item label="Namespace">
					{node.namespace || "—"}
				</Descriptions.Item>
				<Descriptions.Item label="Pod">
					{podName ? <CopyableEllipsisText text={podName} /> : "—"}
				</Descriptions.Item>
				<Descriptions.Item label="Pod IP">
					{node.podIp || "—"}
				</Descriptions.Item>
				<Descriptions.Item label="Service Account">
					{node.serviceAccountName || "—"}
				</Descriptions.Item>
				<Descriptions.Item label="重启次数">
					{node.restartCount ?? "—"}
				</Descriptions.Item>
			</Descriptions>
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
				message={metrics ? "监控快照" : "监控接口待接入"}
				description={
					metrics
						? `采样时间：${metrics.sampledAt ? formatRelativeTime(metrics.sampledAt) : "—"}`
						: "前端已预留 CPU、内存、GPU、网络、存储指标卡；后端可接 metrics-server、Prometheus 或 OpenCost allocation 数据。"
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
				message={cost ? "计费快照" : "计费接口待接入"}
				description={
					cost
						? `窗口：${cost.window || "—"} · 来源：${cost.provider || "—"}`
						: "前端已预留 OpenCost/custom 成本字段；后端应按 pipeline run、asset、node、cluster、namespace 归因。"
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

function DebugTab({ node }: { node: WorkflowNodeStatus }) {
	const execEnabled = node.debug?.execEnabled === true;
	const commandTemplates = [
		"pwd",
		"ls -lah",
		"env",
		"cat /tmp/outputs/output",
		"df -h",
		"ps aux",
	];

	return (
		<Space direction="vertical" size="middle" style={{ width: "100%" }}>
			<Alert
				type={execEnabled ? "warning" : "info"}
				showIcon
				message={execEnabled ? "调试终端待确认" : "Exec 接口待接入"}
				description={
					node.debug?.reason ||
					"前端已预留终端区域和命令模板；后端需要 WebSocket exec 代理、RBAC、审计、超时和 cluster/namespace 隔离后再启用。"
				}
			/>
			<Space wrap>
				{commandTemplates.map((command) => (
					<Button key={command} size="small" disabled={!execEnabled}>
						{command}
					</Button>
				))}
			</Space>
			<div
				style={{
					height: 220,
					background: "#111827",
					color: "#d1d5db",
					borderRadius: 6,
					padding: 12,
					fontFamily: '"SF Mono", "Fira Code", monospace',
					fontSize: 12,
					display: "flex",
					alignItems: "center",
					justifyContent: "center",
					textAlign: "center",
				}}
			>
				<Space direction="vertical" align="center">
					<LockOutlined style={{ fontSize: 22 }} />
					<span>等待后端 WebSocket exec 能力接入</span>
				</Space>
			</div>
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
						{typeof node.outputs?.exitCode === "number"
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
}: {
	node: WorkflowNodeStatus | null;
	workflow: WorkflowDetail | null;
	open: boolean;
	onClose: () => void;
	onRetryWorkflow?: () => void;
	canRetryWorkflow?: boolean;
	onShowLogs: () => void;
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
				items={[
					{
						key: "summary",
						label: "概览",
						children: <SummaryTab node={node} workflow={workflow} />,
					},
					{
						key: "containers",
						label: "容器",
						children: <ContainersTab node={node} />,
					},
					{
						key: "pod",
						label: (
							<>
								<CloudServerOutlined /> Pod
							</>
						),
						children: <PodTab node={node} />,
					},
					{
						key: "monitoring",
						label: "监控",
						children: <MonitoringTab metrics={node.metrics} />,
					},
					{
						key: "billing",
						label: (
							<>
								<DollarOutlined /> 计费
							</>
						),
						children: <BillingTab cost={node.cost} />,
					},
					{
						key: "debug",
						label: (
							<>
								<ThunderboltOutlined /> 调试
							</>
						),
						children: <DebugTab node={node} />,
					},
					{
						key: "inputs-outputs",
						label: "输入/输出",
						children: (
							<Space direction="vertical" style={{ width: "100%" }}>
								<InputsTab node={node} />
								<OutputsTab node={node} />
							</Space>
						),
					},
				]}
			/>
		</Drawer>
	);
}
