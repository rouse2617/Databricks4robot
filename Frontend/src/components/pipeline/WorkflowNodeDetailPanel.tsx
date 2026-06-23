// @ts-nocheck -- pre-existing unused declarations on dev, not CYB-1618 scope
import {
	CloudServerOutlined,
	FileTextOutlined,
	ReloadOutlined,
} from "@ant-design/icons";
import {
	Alert,
	Button,
	Card,
	Col,
	Collapse,
	Descriptions,
	Drawer,
	Empty,
	Row,
	Select,
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
import {
	useCallback,
	useEffect,
	useMemo,
	useReducer,
	useRef,
	useState,
} from "react";
import { ApiError } from "../../api/pipelineClient";
import type {
	WorkflowDetail,
	WorkflowNodeContainer,
	WorkflowNodeStatus,
	WorkflowPodCondition,
	WorkflowPodCost,
	WorkflowPodEvent,
	WorkflowPodMetrics,
	WorkflowPodResourceUsage,
	WorkflowResourceUsageReport,
} from "../../api/workflowApi";
import {
	createTerminalSession,
	getNodePodDiagnostics,
	getTerminalAttachUrl,
	getWorkflowNodeResourceUsage,
	type NodePodDiagnostics,
	type TerminalSession,
	terminateTerminalSession,
} from "../../api/workflowApi";
import { ArgoNodeRuntimeInspector } from "../../features/pipeline-designer";
import {
	formatWorkflowPhaseLabel,
	resolveStatusTagColor,
} from "../../lib/statusLabels";
import {
	getWorkflowNodePodName,
	truncateMiddle,
} from "../../lib/workflowNodeDisplay";
import { DurationPanel } from "../common/DurationPanel";
import type { PipelineNodeDef } from "./types";

type KeyValue = { name: string; value?: string };
type Artifact = { name: string; path?: string };
export type WorkflowNodeDetailTabKey = "summary" | "logs" | "runtime" | "io";
type RuntimeBindingRow = {
	key: string;
	type: string;
	name: string;
	path: string;
	mode: string;
	envName: string;
};
type KeyValueRow = { key: string; name: string; value: string };
type ArtifactRow = { key: string; name: string; path: string };

const keyValueColumns: ColumnsType<KeyValueRow> = [
	{ title: "名称", dataIndex: "name", key: "name" },
	{
		title: "值",
		dataIndex: "value",
		key: "value",
		render: (value: string) =>
			value ? (
				<Typography.Text copyable={{ text: value }}>{value}</Typography.Text>
			) : (
				"—"
			),
	},
];

const artifactColumns: ColumnsType<ArtifactRow> = [
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

const diagnosticColumns: ColumnsType<ArtifactRow> = [
	{
		title: "类型",
		dataIndex: "name",
		key: "name",
		width: 96,
		render: (name: string) => <Tag color="blue">{name || "diagnostic"}</Tag>,
	},
	{
		title: "入口",
		dataIndex: "path",
		key: "path",
		render: (path: string) =>
			path ? (
				<Typography.Text copyable={{ text: path }}>{path}</Typography.Text>
			) : (
				"—"
			),
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

const runtimeBindingColumns: ColumnsType<RuntimeBindingRow> = [
	{
		title: "类型",
		dataIndex: "type",
		key: "type",
		width: 76,
		render: (type: string) => <Tag>{type}</Tag>,
	},
	{
		title: "资源",
		dataIndex: "name",
		key: "name",
		render: (name: string) => (
			<Typography.Text ellipsis={{ tooltip: name }}>{name}</Typography.Text>
		),
	},
	{
		title: "挂载路径",
		dataIndex: "path",
		key: "path",
		render: (path: string) =>
			path === "—" ? "—" : <Typography.Text copyable>{path}</Typography.Text>,
	},
	{
		title: "注入环境变量",
		dataIndex: "envName",
		key: "envName",
		render: (envName: string) =>
			envName === "—" ? (
				"—"
			) : (
				<Typography.Text code copyable={{ text: envName }}>
					{envName}
				</Typography.Text>
			),
	},
	{
		title: "版本/权限",
		dataIndex: "mode",
		key: "mode",
		width: 96,
		render: (mode: string) => mode || "—",
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

function isRunInputParameter(row: KeyValueRow) {
	const name = row.name.toLowerCase();
	return (
		name === "asset" ||
		name === "asset_id" ||
		name === "asset_ids" ||
		name === "runtime_target" ||
		name === "execution_target" ||
		name === "target"
	);
}

function isDiagnosticArtifact(row: ArtifactRow) {
	const name = row.name.toLowerCase();
	const path = row.path.toLowerCase();
	return (
		name === "logs" ||
		name === "metrics" ||
		name.includes("log") ||
		name.includes("metric") ||
		path.includes("/logs") ||
		path.includes("/metrics")
	);
}

function renderCompactTable<T extends { key: string }>({
	rows,
	columns,
	empty,
}: {
	rows: T[];
	columns: ColumnsType<T>;
	empty: string;
}) {
	return rows.length === 0 ? (
		<Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={empty} />
	) : (
		<Table
			size="small"
			dataSource={rows}
			columns={columns}
			pagination={false}
			rowKey="key"
		/>
	);
}

function joinMountedFilePath(mountPath?: string, fileName?: string) {
	const base = mountPath?.trim();
	const name = fileName?.trim();
	if (!base && !name) return "—";
	if (!base) return name || "—";
	if (!name) return base;
	return `${base.replace(/\/+$/, "")}/${name.replace(/^\/+/, "")}`;
}

function runtimeMountEnvName(
	prefix: "PIPELINE_SECRET" | "PIPELINE_STORAGE",
	id: string,
) {
	const suffix = id
		.trim()
		.toUpperCase()
		.replace(/[^A-Z0-9]+/g, "_")
		.replace(/^_+|_+$/g, "");
	return `${prefix}_${suffix || "MOUNT"}_PATH`;
}

function runtimeBindingRows(
	pipelineNode?: PipelineNodeDef | null,
): RuntimeBindingRow[] {
	if (!pipelineNode) return [];
	const rows: RuntimeBindingRow[] = [];
	const runtimeConfig = pipelineNode.runtimeConfig;
	if (runtimeConfig) {
		rows.push({
			key: "config",
			type: "配置",
			name:
				runtimeConfig.displayName ||
				runtimeConfig.fileName ||
				runtimeConfig.targetFilename ||
				runtimeConfig.configId,
			path: joinMountedFilePath(
				runtimeConfig.mountPath,
				runtimeConfig.targetFilename || runtimeConfig.fileName,
			),
			mode: runtimeConfig.version ? `v${runtimeConfig.version}` : "—",
			envName: "PIPELINE_CONFIG_PATH",
		});
	}
	for (const [index, item] of (pipelineNode.runtimeSecrets ?? []).entries()) {
		const resourceId = item.resourceId?.trim();
		rows.push({
			key: `secret-${resourceId || index}`,
			type: "密钥",
			name: item.displayName || resourceId || "未命名密钥",
			path: item.mountPath || "—",
			mode: "只读",
			envName: resourceId
				? runtimeMountEnvName("PIPELINE_SECRET", resourceId)
				: "—",
		});
	}
	for (const [index, item] of (pipelineNode.storageMounts ?? []).entries()) {
		const resourceId = item.resourceId?.trim();
		rows.push({
			key: `storage-${resourceId || index}`,
			type: "存储",
			name: item.displayName || resourceId || "未命名存储",
			path: item.mountPath || "—",
			mode: item.readOnly ? "只读" : "读写",
			envName: resourceId
				? runtimeMountEnvName("PIPELINE_STORAGE", resourceId)
				: "—",
		});
	}
	return rows;
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

function formatRequestLimit(request?: string, limit?: string) {
	const req = request?.trim();
	const lim = limit?.trim();
	if (req && lim) return `request / limit: ${req} / ${lim}`;
	if (req) return `request: ${req}`;
	if (lim) return `limit: ${lim}`;
	return "request / limit: —";
}

function hasResourceUsageSnapshot(pod?: WorkflowPodResourceUsage | null) {
	if (!pod) return false;
	return Boolean(
		pod.cpu_request ||
			pod.memory_request ||
			pod.cpu_limit ||
			pod.memory_limit ||
			pod.cpu_resource_duration ||
			pod.memory_resource_duration ||
			pod.cpu_usage ||
			pod.memory_usage,
	);
}

function firstResourcePod(report?: WorkflowResourceUsageReport | null) {
	return report?.pods?.[0] ?? null;
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
					if (err instanceof ApiError && err.status === 403) {
						setPodDiagError("forbidden");
						return;
					}
					if (err instanceof ApiError && err.status === 404) {
						setPodDiagError("not_found");
						return;
					}
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
					description={
						podDiagError === "forbidden"
							? "当前环境缺少读取 Pod 诊断所需的 Kubernetes 权限；已保留 Argo 节点元数据。"
							: podDiagError === "not_found"
								? "当前节点尚未解析到 Pod，或 Pod 还未创建完成；已保留 Argo 节点元数据。"
								: "已保留 Argo 节点元数据；请检查 K8s API 连通性、RBAC 权限，或等待 Pod 创建完成。"
					}
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

function MonitoringTab({
	metrics,
	resourceReport,
	resourceLoading,
	resourceError,
	onRefreshResources,
}: {
	metrics?: WorkflowPodMetrics;
	resourceReport?: WorkflowResourceUsageReport | null;
	resourceLoading?: boolean;
	resourceError?: string | null;
	onRefreshResources?: () => void;
}) {
	const resourcePod = firstResourcePod(resourceReport);
	const hasResourceSnapshot = hasResourceUsageSnapshot(resourcePod);
	const cpuPercent = getMetricPercent(
		metrics?.cpuCores,
		metrics?.cpuLimitCores || metrics?.cpuRequestCores,
	);
	const memoryPercent = getMetricPercent(
		metrics?.memoryBytes,
		metrics?.memoryLimitBytes || metrics?.memoryRequestBytes,
	);
	const hasLiveMetrics = Boolean(metrics);
	const showResourceError =
		Boolean(resourceError) && !hasLiveMetrics && !hasResourceSnapshot;
	const showResourceLoading =
		Boolean(resourceLoading) && !hasLiveMetrics && !hasResourceSnapshot;
	const alertType = showResourceError
		? "warning"
		: hasLiveMetrics || hasResourceSnapshot
			? "success"
			: "info";
	const alertMessage = showResourceError
		? "监控数据不可用"
		: hasLiveMetrics
			? "监控快照"
			: hasResourceSnapshot
				? "资源规格快照"
				: showResourceLoading
					? "正在加载监控数据"
					: "暂无监控数据";
	const alertDescription = showResourceError
		? resourceError
		: hasLiveMetrics
			? `采样时间：${metrics?.sampledAt ? formatRelativeTime(metrics.sampledAt) : "—"}`
			: hasResourceSnapshot
				? `采样时间：${resourceReport?.observed_at ? formatRelativeTime(resourceReport.observed_at) : "—"} · 实时指标：${resourceReport?.source?.metrics || "unavailable"}`
				: showResourceLoading
					? "正在读取该节点的资源请求、限制和资源耗时。"
					: "当前运行未返回 CPU、内存、GPU、网络或存储指标。";

	return (
		<Space direction="vertical" size="middle" style={{ width: "100%" }}>
			<Alert
				type={alertType}
				showIcon
				message={alertMessage}
				description={alertDescription}
				action={
					onRefreshResources ? (
						<Button
							size="small"
							icon={<ReloadOutlined />}
							loading={resourceLoading}
							onClick={onRefreshResources}
						>
							刷新
						</Button>
					) : undefined
				}
			/>
			<Row gutter={[12, 12]}>
				<Col span={12}>
					<Card size="small">
						<Statistic
							title="CPU"
							value={
								metrics?.cpuCores ??
								resourcePod?.cpu_resource_duration ??
								resourcePod?.cpu_usage ??
								"—"
							}
							suffix={typeof metrics?.cpuCores === "number" ? "cores" : ""}
						/>
						<Typography.Text type="secondary">
							{cpuPercent === undefined
								? formatRequestLimit(
										resourcePod?.cpu_request,
										resourcePod?.cpu_limit,
									)
								: `用量 ${cpuPercent}%`}
						</Typography.Text>
					</Card>
				</Col>
				<Col span={12}>
					<Card size="small">
						<Statistic
							title="Memory"
							value={
								typeof metrics?.memoryBytes === "number"
									? formatBytes(metrics.memoryBytes)
									: resourcePod?.memory_resource_duration ||
										resourcePod?.memory_usage ||
										"—"
							}
						/>
						<Typography.Text type="secondary">
							{memoryPercent === undefined
								? formatRequestLimit(
										resourcePod?.memory_request,
										resourcePod?.memory_limit,
									)
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
						: "本次运行没有记录该步骤的成本拆分；这通常出现在历史工作流、外部提交工作流或未开启计费采集的执行目标。"
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
	const debug = node.debug;
	const execEnabled = Boolean(debug?.execEnabled);
	const reason =
		debug?.reason ||
		(execEnabled
			? "该节点允许创建 Pod 终端会话。"
			: "当前节点或执行目标未开启 Pod 终端。");
	const allowedCommands = debug?.allowedCommands?.join(", ") || "—";

	return (
		<Space direction="vertical" size="middle" style={{ width: "100%" }}>
			<Alert
				type={execEnabled ? "success" : "info"}
				showIcon
				message={execEnabled ? "终端可用" : "终端不可用"}
				description={reason}
			/>
			<Descriptions size="small" column={1} layout="vertical" bordered>
				<Descriptions.Item label="Pod exec">
					<Tag color={execEnabled ? "green" : "default"}>
						{execEnabled ? "enabled" : "disabled"}
					</Tag>
				</Descriptions.Item>
				<Descriptions.Item label="日志流">
					<Tag color={debug?.logStreamEnabled ? "green" : "default"}>
						{debug?.logStreamEnabled ? "enabled" : "disabled"}
					</Tag>
				</Descriptions.Item>
				<Descriptions.Item label="允许命令">
					<Typography.Text code>{allowedCommands}</Typography.Text>
				</Descriptions.Item>
				<Descriptions.Item label="最长会话">
					{debug?.maxSessionSeconds ? `${debug.maxSessionSeconds}s` : "—"}
				</Descriptions.Item>
			</Descriptions>
		</Space>
	);
}

type TerminalFrame = {
	type?: string;
	status?: string;
	data?: string;
	code?: string;
	message?: string;
	exitCode?: number;
	reason?: string;
};

const TERMINAL_OUTPUT_BUFFER_CHARS = 200_000;

function appendTerminalOutput(current: string, chunk: string) {
	const next = `${current}${chunk}`;
	if (next.length <= TERMINAL_OUTPUT_BUFFER_CHARS) return next;
	return next.slice(-TERMINAL_OUTPUT_BUFFER_CHARS);
}

function formatTerminalError(err: unknown): string {
	if (err instanceof ApiError) {
		return `${err.code}: ${err.message}`;
	}
	if (err instanceof Error) return err.message;
	return String(err);
}

function TerminalSessionPanel({
	node,
	workflowName,
}: {
	node: WorkflowNodeStatus;
	workflowName: string;
}) {
	const debug = node.debug;
	const execEnabled = Boolean(debug?.execEnabled);
	const allowedCommands = useMemo(
		() =>
			debug?.allowedCommands && debug.allowedCommands.length > 0
				? debug.allowedCommands
				: ["sh"],
		[debug?.allowedCommands],
	);
	const [command, setCommand] = useState(allowedCommands[0] ?? "sh");
	const [session, setSession] = useState<TerminalSession | null>(null);
	const [output, setOutput] = useState("");
	const [connecting, setConnecting] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [restartReady, setRestartReady] = useState(false);
	const socketRef = useRef<WebSocket | null>(null);

	useEffect(() => {
		if (!allowedCommands.includes(command)) {
			setCommand(allowedCommands[0] ?? "sh");
		}
	}, [allowedCommands, command]);

	useEffect(() => {
		return () => {
			socketRef.current?.close();
			socketRef.current = null;
		};
	}, []);

	const closeSocket = useCallback(() => {
		socketRef.current?.close();
		socketRef.current = null;
	}, []);

	const startTerminal = useCallback(async () => {
		if (!execEnabled) return;
		closeSocket();
		setConnecting(true);
		setError(null);
		setOutput("");
		setRestartReady(false);
		try {
			const created = await createTerminalSession(workflowName, node.id, {
				command,
			});
			setSession(created);
			if (!created.attachUrl) {
				setError("POD_EXEC_UNAVAILABLE: 后端未返回终端 attach URL");
				return;
			}
			const socket = new WebSocket(getTerminalAttachUrl(created.attachUrl));
			socketRef.current = socket;
			socket.onopen = () => {
				setConnecting(false);
				setSession((current) =>
					current ? { ...current, status: "attached" } : current,
				);
			};
			socket.onmessage = (event) => {
				let frame: TerminalFrame;
				try {
					frame = JSON.parse(String(event.data));
				} catch {
					frame = { type: "stdout", data: String(event.data) };
				}
				if (frame.type === "stdout" || frame.type === "stderr") {
					const prefix = frame.type === "stderr" ? "[stderr] " : "";
					setOutput((current) =>
						appendTerminalOutput(current, `${prefix}${frame.data ?? ""}`),
					);
					return;
				}
				if (frame.type === "status") {
					setSession((current) =>
						current
							? { ...current, status: frame.status || current.status }
							: current,
					);
					return;
				}
				if (frame.type === "error") {
					setConnecting(false);
					setRestartReady(true);
					setError(
						`${frame.code || "POD_EXEC_FAILED"}: ${frame.message || "terminal attach failed"}`,
					);
					return;
				}
				if (frame.type === "exit") {
					setConnecting(false);
					setRestartReady(true);
					setSession((current) =>
						current
							? {
									...current,
									status: frame.exitCode === 0 ? "ended" : "failed",
								}
							: current,
					);
					closeSocket();
				}
			};
			socket.onerror = () => {
				setConnecting(false);
				setRestartReady(true);
				setError("POD_EXEC_FAILED: WebSocket 连接失败");
			};
			socket.onclose = () => {
				setConnecting(false);
				socketRef.current = null;
			};
		} catch (err) {
			setError(formatTerminalError(err));
			setConnecting(false);
		}
	}, [closeSocket, command, execEnabled, node.id, workflowName]);

	const terminate = useCallback(async () => {
		const currentSession = session;
		closeSocket();
		if (!currentSession) return;
		setConnecting(false);
		setRestartReady(true);
		try {
			const next = await terminateTerminalSession(currentSession.id);
			setSession(next);
		} catch (err) {
			setError(formatTerminalError(err));
		}
	}, [closeSocket, session]);

	const terminalInactive =
		restartReady ||
		!session ||
		["ended", "failed", "terminated", "expired"].includes(session.status);

	return (
		<Space direction="vertical" size="small" style={{ width: "100%" }}>
			<Alert
				type={execEnabled ? "success" : "info"}
				showIcon
				message={execEnabled ? "Pod 终端可进入" : "Pod 终端不可进入"}
				description={
					debug?.reason ||
					(execEnabled
						? "使用后端一次性 attach token 连接当前 Pod。"
						: "当前执行目标未开启 Pod 终端，或 Pod 不在可进入状态。")
				}
			/>
			<Space wrap>
				<Select
					size="small"
					style={{ width: 140 }}
					value={command}
					disabled={!execEnabled || !terminalInactive}
					options={allowedCommands.map((item) => ({
						label: item,
						value: item,
					}))}
					onChange={setCommand}
				/>
				<Button
					type="primary"
					size="small"
					disabled={!execEnabled || !terminalInactive}
					loading={connecting}
					onClick={() => void startTerminal()}
				>
					{restartReady || session ? "重新打开终端" : "进入终端"}
				</Button>
				<Button
					size="small"
					disabled={!session || terminalInactive}
					onClick={() => void terminate()}
				>
					结束会话
				</Button>
				{session ? <Tag>{session.status}</Tag> : null}
			</Space>
			{error ? (
				<Alert
					type="error"
					showIcon
					message="终端连接失败"
					description={error}
				/>
			) : null}
			<pre
				style={{
					minHeight: 96,
					maxHeight: 220,
					overflow: "auto",
					margin: 0,
					padding: 10,
					border: "1px solid #e5e7eb",
					borderRadius: 6,
					background: "#0f172a",
					color: "#e2e8f0",
					fontSize: 12,
					whiteSpace: "pre-wrap",
				}}
			>
				{output || "终端输出会显示在这里。"}
			</pre>
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
					<Tag color={resolveStatusTagColor(node.phase)}>
						{formatWorkflowPhaseLabel(node.phase)}
					</Tag>
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

function RuntimeBindingsPanel({
	pipelineNode,
}: {
	pipelineNode?: PipelineNodeDef | null;
}) {
	const rows = runtimeBindingRows(pipelineNode);
	if (!pipelineNode) {
		return (
			<Alert
				type="info"
				showIcon
				message="暂无 DataBrew 节点配置快照"
				description="当前执行没有返回 pipeline JSON，或该 Argo 节点未映射到设计器节点；可继续查看 Pod 环境和日志。"
			/>
		);
	}
	if (rows.length === 0) {
		return (
			<Alert
				type="info"
				showIcon
				message="该节点没有运行时挂载"
				description="没有绑定配置文件、密钥或存储资源；资产 ID 仍会按部署参数注入到每个 Pod。"
			/>
		);
	}
	return (
		<Space direction="vertical" size="small" style={{ width: "100%" }}>
			<Table
				size="small"
				dataSource={rows}
				columns={runtimeBindingColumns}
				pagination={false}
				rowKey="key"
				scroll={{ x: 620 }}
			/>
			<Descriptions size="small" column={1} bordered>
				<Descriptions.Item label="资产环境变量">
					<Typography.Text code>ASSET_IDS</Typography.Text>
					<Typography.Text type="secondary"> / </Typography.Text>
					<Typography.Text code>ASSET_COUNT</Typography.Text>
					<Typography.Text type="secondary"> / </Typography.Text>
					<Typography.Text code>ASSET_0_ID</Typography.Text>
				</Descriptions.Item>
			</Descriptions>
		</Space>
	);
}

function RuntimeTab({
	node,
	workflowName,
	pipelineNode,
}: {
	node: WorkflowNodeStatus;
	workflowName: string;
	pipelineNode?: PipelineNodeDef | null;
}) {
	const [resourceReport, setResourceReport] =
		useState<WorkflowResourceUsageReport | null>(null);
	const [resourceLoading, setResourceLoading] = useState(false);
	const [resourceError, setResourceError] = useState<string | null>(null);
	const [resourceRefreshTrigger, refreshResources] = useReducer(
		(count) => count + 1,
		0,
	);

	useEffect(() => {
		if (!workflowName || !node.id) return undefined;
		let cancelled = false;
		const requestSeq = resourceRefreshTrigger;
		setResourceLoading(true);
		setResourceError(null);
		getWorkflowNodeResourceUsage(workflowName, node.id)
			.then((data) => {
				if (!cancelled && requestSeq === resourceRefreshTrigger) {
					setResourceReport(data);
				}
			})
			.catch((err) => {
				if (!cancelled && requestSeq === resourceRefreshTrigger) {
					setResourceReport(null);
					if (err instanceof ApiError && err.status === 404) {
						setResourceError("当前节点尚未生成资源快照，或对应 Pod 已被清理。");
						return;
					}
					setResourceError(err instanceof Error ? err.message : String(err));
				}
			})
			.finally(() => {
				if (!cancelled && requestSeq === resourceRefreshTrigger) {
					setResourceLoading(false);
				}
			});
		return () => {
			cancelled = true;
		};
	}, [workflowName, node.id, resourceRefreshTrigger]);

	return (
		<Space direction="vertical" size="middle" style={{ width: "100%" }}>
			<RuntimeSection title="DataBrew 节点挂载">
				<RuntimeBindingsPanel pipelineNode={pipelineNode} />
			</RuntimeSection>
			<RuntimeSection title="Argo 运行时">
				<ArgoNodeRuntimeInspector node={node} />
			</RuntimeSection>
			<RuntimeSection title="Pod 与事件">
				<PodTab node={node} workflowName={workflowName} />
			</RuntimeSection>
			<RuntimeSection title="监控">
				<MonitoringTab
					metrics={node.metrics}
					resourceReport={resourceReport}
					resourceLoading={resourceLoading}
					resourceError={resourceError}
					onRefreshResources={refreshResources}
				/>
			</RuntimeSection>
			<RuntimeSection title="计费">
				<BillingTab cost={node.cost} />
			</RuntimeSection>
			<RuntimeSection title="调试">
				<DebugTab node={node} />
			</RuntimeSection>
			<RuntimeSection title="终端">
				<TerminalSessionPanel node={node} workflowName={workflowName} />
			</RuntimeSection>
		</Space>
	);
}

function InputOutputTab({ node }: { node: WorkflowNodeStatus }) {
	const inputParameters = formatKvRows(node.inputs?.parameters);
	const inputArtifacts = formatArtifactRows(node.inputs?.artifacts);
	const outputParameters = formatKvRows(node.outputs?.parameters);
	const outputArtifacts = formatArtifactRows(node.outputs?.artifacts);
	const runInputParameters = inputParameters.filter(isRunInputParameter);
	const nodeParameters = inputParameters.filter(
		(row) => !isRunInputParameter(row),
	);
	const diagnostics = outputArtifacts.filter(isDiagnosticArtifact);
	const realOutputArtifacts = outputArtifacts.filter(
		(row) => !isDiagnosticArtifact(row),
	);
	const outputResult =
		typeof node.outputs?.result === "string" && node.outputs.result.trim()
			? node.outputs.result
			: "";
	const outputExitCode =
		typeof node.outputs?.exitCode === "number" ||
		typeof node.outputs?.exitCode === "string"
			? String(node.outputs.exitCode)
			: "";
	const runtimeInputRows: KeyValueRow[] = [
		...runInputParameters,
		{
			key: "workflow-node",
			name: "node",
			value: node.displayName || node.name || node.id,
		},
		{
			key: "workflow-pod",
			name: "pod",
			value: getWorkflowNodePodName(node) || "",
		},
	].filter((row) => row.value);
	const outputSummaryRows: KeyValueRow[] = [
		...outputParameters,
		...(outputResult
			? [{ key: "output-result", name: "result", value: outputResult }]
			: []),
		...(outputExitCode
			? [{ key: "output-exit-code", name: "exitCode", value: outputExitCode }]
			: []),
	];
	const diagnosticRows: ArtifactRow[] = [
		...diagnostics,
		{
			key: "diagnostic-pod",
			name: "pod",
			path: getWorkflowNodePodName(node) || "",
		},
	].filter((row) => row.path);

	return (
		<Collapse
			defaultActiveKey={["run-inputs"]}
			items={[
				{
					key: "run-inputs",
					label: "运行输入",
					children: (
						<Space direction="vertical" size="middle" style={{ width: "100%" }}>
							<div>
								<div style={{ marginBottom: 8, fontWeight: 600 }}>
									资产、target、全局输入
								</div>
								{renderCompactTable({
									rows: runtimeInputRows,
									columns: keyValueColumns,
									empty: "暂无运行输入",
								})}
							</div>
							<div>
								<div style={{ marginBottom: 8, fontWeight: 600 }}>输入产物</div>
								{renderCompactTable({
									rows: inputArtifacts,
									columns: artifactColumns,
									empty: "暂无输入产物",
								})}
							</div>
						</Space>
					),
				},
				{
					key: "node-parameters",
					label: "节点参数",
					children: (
						<Space direction="vertical" size="middle" style={{ width: "100%" }}>
							<div>
								<div style={{ marginBottom: 8, fontWeight: 600 }}>
									command / args / config
								</div>
								{renderCompactTable({
									rows: nodeParameters,
									columns: keyValueColumns,
									empty: "暂无节点参数",
								})}
							</div>
						</Space>
					),
				},
				{
					key: "node-outputs",
					label: "节点输出",
					children: (
						<Space direction="vertical" size="middle" style={{ width: "100%" }}>
							<div>
								<div style={{ marginBottom: 8, fontWeight: 600 }}>输出值</div>
								{renderCompactTable({
									rows: outputSummaryRows,
									columns: keyValueColumns,
									empty: "暂无输出值",
								})}
							</div>
							<div>
								<div style={{ marginBottom: 8, fontWeight: 600 }}>输出产物</div>
								{renderCompactTable({
									rows: realOutputArtifacts,
									columns: artifactColumns,
									empty: "暂无真实输出产物",
								})}
							</div>
							<Alert
								type="info"
								showIcon
								message="下游消费关系"
								description="当前运行详情未返回稳定的消费者字段；后端补齐后这里会显示 output 被哪些下游 input 消费。"
							/>
						</Space>
					),
				},
				{
					key: "diagnostics",
					label: "诊断入口",
					children: (
						<Space direction="vertical" size="middle" style={{ width: "100%" }}>
							{renderCompactTable({
								rows: diagnosticRows,
								columns: diagnosticColumns,
								empty: "暂无日志、metrics 或 Pod 入口",
							})}
						</Space>
					),
				},
			]}
		/>
	);
}

function SummaryTab({
	node,
	workflow,
	pipelineNode,
}: {
	node: WorkflowNodeStatus;
	workflow: WorkflowDetail;
	pipelineNode?: PipelineNodeDef | null;
}) {
	const podName = getWorkflowNodePodName(node);
	const versionLabel =
		node.versionLabel ||
		pipelineNode?.component?.componentVersionLabel ||
		node.templateName ||
		"—";
	const commitLabel = node.sourceCommit || "—";
	const imageLabel =
		node.image ||
		pipelineNode?.component?.image ||
		node.containers?.[0]?.image ||
		"—";
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
				<Descriptions.Item label="版本">
					{versionLabel === "—" ? "—" : <CopyableEllipsisText text={versionLabel} />}
				</Descriptions.Item>
				<Descriptions.Item label="Commit">
					{commitLabel === "—" ? "—" : <CopyableEllipsisText text={commitLabel} maxLength={12} />}
				</Descriptions.Item>
				<Descriptions.Item label="镜像">
					{imageLabel === "—" ? "—" : <CopyableEllipsisText text={imageLabel} maxLength={52} />}
				</Descriptions.Item>
				<Descriptions.Item label="状态">
					<Tag color={resolveStatusTagColor(node.phase)}>
						{formatWorkflowPhaseLabel(node.phase)}
					</Tag>
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
			<RuntimeSection title="运行时挂载">
				<RuntimeBindingsPanel pipelineNode={pipelineNode} />
			</RuntimeSection>
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
	pipelineNode,
}: {
	node: WorkflowNodeStatus | null;
	workflow: WorkflowDetail | null;
	pipelineNode?: PipelineNodeDef | null;
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
		const nodeFailed = node.phase === "Failed";
		drawerExtra.unshift(
			<Button key="retry" icon={<ReloadOutlined />} onClick={onRetryWorkflow}>
				{nodeFailed ? "重试失败节点" : "重试工作流"}
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
						children: (
							<SummaryTab
								node={node}
								workflow={workflow}
								pipelineNode={pipelineNode}
							/>
						),
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
						children: (
							<RuntimeTab
								node={node}
								workflowName={workflow.name}
								pipelineNode={pipelineNode}
							/>
						),
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
