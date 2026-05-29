import { FileTextOutlined, ReloadOutlined } from "@ant-design/icons";
import {
	Button,
	Card,
	Col,
	Descriptions,
	Drawer,
	Empty,
	Row,
	Space,
	Table,
	Tabs,
	Tag,
	Tooltip,
	Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import dayjs from "dayjs";
import { useMemo } from "react";
import type { WorkflowDetail, WorkflowNodeStatus } from "../../api/workflowApi";
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
}> = [
	{ title: "名称", dataIndex: "name", key: "name" },
	{ title: "镜像", dataIndex: "image", key: "image" },
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
	type ContainerItem = {
		name: string;
		image?: string;
		command?: string[];
		args?: string[];
	};
	const containerData = useMemo(() => {
		const raw = (node as { containers?: ContainerItem[] }).containers ?? [];
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
