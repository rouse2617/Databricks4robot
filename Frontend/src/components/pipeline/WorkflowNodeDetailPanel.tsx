import {
	FileTextOutlined,
	ReloadOutlined,
	ThunderboltOutlined,
} from "@ant-design/icons";
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
	Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import dayjs from "dayjs";
import { useMemo } from "react";
import type { WorkflowDetail, WorkflowNodeStatus } from "../../api/workflowApi";
import { STATUS_COLORS } from "../../lib/constants";
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
		return <Empty description="暂无容器" />;
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
				<div style={{ marginBottom: 8, fontWeight: 600 }}>参数</div>
				{parameters.length === 0 ? (
					<Empty description="暂无参数" />
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
				<div style={{ marginBottom: 8, fontWeight: 600 }}>产物</div>
				{artifacts.length === 0 ? (
					<Empty description="暂无产物" />
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
					<Card size="small" title="Result">
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
					<Card size="small" title="ExitCode">
						{typeof node.outputs?.exitCode === "number"
							? node.outputs.exitCode
							: "—"}
					</Card>
				</Col>
			</Row>
			<div>
				<div style={{ marginBottom: 8, fontWeight: 600 }}>Parameters</div>
				{parameters.length === 0 ? (
					<Empty description="暂无参数" />
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
				<div style={{ marginBottom: 8, fontWeight: 600 }}>Artifacts</div>
				{artifacts.length === 0 ? (
					<Empty description="暂无产物" />
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
	const memoizationText = node.memoizationStatus
		? `hit=${node.memoizationStatus.hit} key=${node.memoizationStatus.key} cache=${node.memoizationStatus.cacheName}`
		: "—";

	return (
		<Space direction="vertical" size="middle" style={{ width: "100%" }}>
			<Descriptions size="small" column={1} layout="vertical" bordered>
				<Descriptions.Item label="NAME">
					<Typography.Text copyable={{ text: node.displayName || node.name }}>
						{node.displayName || node.name}
					</Typography.Text>
				</Descriptions.Item>
				<Descriptions.Item label="ID">
					<Typography.Text copyable={{ text: node.id }}>
						{node.id}
					</Typography.Text>
				</Descriptions.Item>
				<Descriptions.Item label="POD NAME">
					{node.podName || "—"}
				</Descriptions.Item>
				<Descriptions.Item label="HOST NODE NAME">
					{node.hostNodeName || "—"}
				</Descriptions.Item>
				<Descriptions.Item label="TYPE">
					{node.type || node.templateName || "—"}
				</Descriptions.Item>
				<Descriptions.Item label="PHASE">
					<Tag color={STATUS_COLORS[node.phase] || "default"}>{node.phase}</Tag>
				</Descriptions.Item>
				<Descriptions.Item label="START TIME">
					{formatRelativeTime(node.startedAt)}
				</Descriptions.Item>
				<Descriptions.Item label="END TIME">
					{formatRelativeTime(node.finishedAt)}
				</Descriptions.Item>
				<Descriptions.Item label="DURATION">
					<DurationPanel
						phase={node.phase}
						startedAt={node.startedAt}
						finishedAt={node.finishedAt}
						progress={node.progress}
					/>
				</Descriptions.Item>
				<Descriptions.Item label="WORKFLOW">{workflow.name}</Descriptions.Item>
				<Descriptions.Item label="PROGRESS">
					{node.progress || workflow.progress || "—"}
				</Descriptions.Item>
				<Descriptions.Item label="MEMOIZATION">
					{memoizationText}
				</Descriptions.Item>
				<Descriptions.Item label="RESOURCES DURATION">
					<pre style={{ margin: 0 }}>
						{formatResourceDurationText(node.resourcesDuration)}
					</pre>
				</Descriptions.Item>
			</Descriptions>
		</Space>
	);
}

export function WorkflowNodeDetailPanel({
	node,
	workflow,
	open,
	onClose,
	onManifest,
	onRetryNode,
	onShowLogs,
	onShowEvents,
}: {
	node: WorkflowNodeStatus | null;
	workflow: WorkflowDetail | null;
	open: boolean;
	onClose: () => void;
	onManifest: () => void;
	onRetryNode: () => void;
	onShowLogs: () => void;
	onShowEvents: () => void;
}) {
	if (!node || !workflow) return null;

	return (
		<Drawer
			title={node.displayName || node.templateName || node.name}
			placement="right"
			width={520}
			open={open}
			onClose={onClose}
			extra={[
				<Button key="manifest" icon={<FileTextOutlined />} onClick={onManifest}>
					MANIFEST
				</Button>,
				<Button key="retry" icon={<ReloadOutlined />} onClick={onRetryNode}>
					RETRY NODE
				</Button>,
				<Button key="logs" icon={<FileTextOutlined />} onClick={onShowLogs}>
					LOGS
				</Button>,
				<Button
					key="events"
					icon={<ThunderboltOutlined />}
					onClick={onShowEvents}
				>
					EVENTS
				</Button>,
			]}
			bodyStyle={{ paddingTop: 8 }}
		>
			<Space direction="vertical" style={{ width: "100%" }}>
				<Tabs
					type="card"
					items={[
						{
							key: "summary",
							label: "SUMMARY",
							children: <SummaryTab node={node} workflow={workflow} />,
						},
						{
							key: "containers",
							label: "CONTAINERS",
							children: <ContainersTab node={node} />,
						},
						{
							key: "inputs-outputs",
							label: "INPUTS/OUTPUTS",
							children: (
								<Space direction="vertical" style={{ width: "100%" }}>
									<InputsTab node={node} />
									<OutputsTab node={node} />
								</Space>
							),
						},
					]}
				/>
			</Space>
		</Drawer>
	);
}
