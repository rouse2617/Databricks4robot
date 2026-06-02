import {
	ClockCircleOutlined,
	CloudServerOutlined,
	DollarOutlined,
	FileTextOutlined,
	InfoCircleOutlined,
} from "@ant-design/icons";
import { Button, Space, Table, Tag, Tooltip, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import type { WorkflowNodeStatus } from "../../api/workflowApi";
import { STATUS_COLORS } from "../../lib/constants";
import {
	getWorkflowNodeDisplayText,
	getWorkflowNodePodName,
} from "../../lib/workflowNodeDisplay";

interface WorkflowNodeSummaryRow {
	key: string;
	node: WorkflowNodeStatus;
	name: string;
	status: string;
	podCount: number;
	durationSeconds: number | null;
	estimatedCostUsd: number | null;
	startedAt?: string;
	finishedAt?: string;
	podName?: string;
}

interface WorkflowNodeSummaryTableProps {
	nodes: WorkflowNodeStatus[];
	selectedNodeId?: string | null;
	onInspectNode: (node: WorkflowNodeStatus) => void;
	onOpenLogs: (node: WorkflowNodeStatus) => void;
	onOpenRuntime: (node: WorkflowNodeStatus) => void;
}

function parseDurationSeconds(node: WorkflowNodeStatus): number | null {
	if (node.startedAt) {
		const start = new Date(node.startedAt).getTime();
		const isRunning = node.phase === "Running";
		const end = node.finishedAt ? new Date(node.finishedAt).getTime() : null;
		if (
			end != null &&
			Number.isFinite(start) &&
			Number.isFinite(end) &&
			end >= start
		) {
			return Math.floor((end - start) / 1000);
		}
		if (isRunning && Number.isFinite(start)) {
			return Math.max(0, Math.floor((Date.now() - start) / 1000));
		}
	}
	if (
		typeof node.estimatedDuration === "number" &&
		Number.isFinite(node.estimatedDuration) &&
		node.estimatedDuration >= 0
	) {
		return Math.floor(node.estimatedDuration);
	}
	return null;
}

function formatDurationSeconds(seconds: number | null): string {
	if (seconds == null) return "-";
	const hours = Math.floor(seconds / 3600);
	const minutes = Math.floor((seconds % 3600) / 60);
	const rest = seconds % 60;
	if (hours > 0) return `${hours}h ${minutes}m ${rest}s`;
	if (minutes > 0) return `${minutes}m ${rest}s`;
	return `${rest}s`;
}

function formatCost(value: number | null): string {
	if (value == null) return "—";
	if (value < 0.01) return `$${value.toFixed(4)}`;
	return `$${value.toFixed(2)}`;
}

function formatDateTime(value?: string): string {
	if (!value) return "-";
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) return "-";
	return date.toLocaleString();
}

function isSummaryNode(node: WorkflowNodeStatus): boolean {
	const type = (node.type ?? "").toLowerCase();
	if (type === "pod" || type === "template") return true;
	if (node.podName) return true;
	if (node.startedAt || node.finishedAt) return true;
	return ["Succeeded", "Failed", "Error", "Running"].includes(node.phase);
}

export function buildWorkflowNodeSummaryRows(
	nodes: WorkflowNodeStatus[],
): WorkflowNodeSummaryRow[] {
	return nodes.filter(isSummaryNode).map((node) => {
		const cost =
			typeof node.estimatedCostUsd === "number"
				? node.estimatedCostUsd
				: typeof node.cost?.totalCostUsd === "number"
					? node.cost.totalCostUsd
					: null;
		const podName = node.podName || getWorkflowNodePodName(node);
		const nodeHasPod =
			!!node.podName || (node.type ?? "").toLowerCase() === "pod";
		return {
			key: node.id,
			node,
			name: getWorkflowNodeDisplayText(node),
			status: node.phase || "-",
			podCount: nodeHasPod ? 1 : 0,
			durationSeconds: parseDurationSeconds(node),
			estimatedCostUsd: cost,
			startedAt: node.startedAt,
			finishedAt: node.finishedAt,
			podName,
		};
	});
}

export function WorkflowNodeSummaryTable({
	nodes,
	selectedNodeId,
	onInspectNode,
	onOpenLogs,
	onOpenRuntime,
}: WorkflowNodeSummaryTableProps) {
	const rows = buildWorkflowNodeSummaryRows(nodes);
	const totalDurationSeconds = rows.reduce(
		(sum, row) => sum + (row.durationSeconds ?? 0),
		0,
	);
	const totalCost = rows.reduce((sum, row) => {
		if (row.estimatedCostUsd == null) return sum;
		return sum + row.estimatedCostUsd;
	}, 0);
	const hasCost = rows.some((row) => row.estimatedCostUsd != null);
	const podCount = rows.reduce((sum, row) => sum + row.podCount, 0);

	const columns: ColumnsType<WorkflowNodeSummaryRow> = [
		{
			title: "节点",
			dataIndex: "name",
			key: "name",
			width: 260,
			render: (_, row) => (
				<Space direction="vertical" size={0}>
					<Typography.Text strong>{row.name}</Typography.Text>
					<Typography.Text type="secondary" style={{ fontSize: 12 }}>
						{row.node.templateName || row.node.name}
					</Typography.Text>
				</Space>
			),
		},
		{
			title: "状态",
			dataIndex: "status",
			key: "status",
			width: 110,
			render: (status: string) => (
				<Tag color={STATUS_COLORS[status] || "default"}>{status}</Tag>
			),
		},
		{
			title: "Pod",
			dataIndex: "podCount",
			key: "podCount",
			width: 90,
			align: "right",
			sorter: (a, b) => a.podCount - b.podCount,
			render: (count: number, row) =>
				count > 0 ? (
					<Tooltip title={row.podName}>
						<Typography.Text>{count}</Typography.Text>
					</Tooltip>
				) : (
					<Typography.Text type="secondary">-</Typography.Text>
				),
		},
		{
			title: "总耗时",
			dataIndex: "durationSeconds",
			key: "durationSeconds",
			width: 120,
			align: "right",
			sorter: (a, b) => (a.durationSeconds ?? -1) - (b.durationSeconds ?? -1),
			defaultSortOrder: "descend",
			render: (value: number | null) => formatDurationSeconds(value),
		},
		{
			title: "估算成本",
			dataIndex: "estimatedCostUsd",
			key: "estimatedCostUsd",
			width: 130,
			align: "right",
			sorter: (a, b) => (a.estimatedCostUsd ?? -1) - (b.estimatedCostUsd ?? -1),
			render: (value: number | null) => (
				<Tooltip
					title={value == null ? "本节点暂无成本数据" : "估算成本，非最终账单"}
				>
					<Typography.Text type={value == null ? "secondary" : undefined}>
						{formatCost(value)}
					</Typography.Text>
				</Tooltip>
			),
		},
		{
			title: "开始",
			dataIndex: "startedAt",
			key: "startedAt",
			width: 170,
			render: (value?: string) => formatDateTime(value),
		},
		{
			title: "结束",
			dataIndex: "finishedAt",
			key: "finishedAt",
			width: 170,
			render: (value?: string) => formatDateTime(value),
		},
		{
			title: "操作",
			key: "actions",
			width: 180,
			render: (_, row) => (
				<Space size={4}>
					<Button
						size="small"
						type="text"
						icon={<InfoCircleOutlined />}
						aria-label={`查看节点 ${row.name}`}
						onClick={(event) => {
							event.stopPropagation();
							onInspectNode(row.node);
						}}
					/>
					<Button
						size="small"
						type="text"
						icon={<FileTextOutlined />}
						aria-label={`查看日志 ${row.name}`}
						onClick={(event) => {
							event.stopPropagation();
							onOpenLogs(row.node);
						}}
					/>
					<Button
						size="small"
						type="text"
						icon={<CloudServerOutlined />}
						aria-label={`查看运行资源 ${row.name}`}
						onClick={(event) => {
							event.stopPropagation();
							onOpenRuntime(row.node);
						}}
					/>
				</Space>
			),
		},
	];

	if (rows.length === 0) {
		return null;
	}

	return (
		<section
			aria-label="节点耗时和花费汇总"
			style={{
				margin: "12px 16px",
				border: "1px solid #e2e8f0",
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
					gap: 16,
					padding: "12px 16px",
					borderBottom: "1px solid #e2e8f0",
					background: "#f8fafc",
				}}
			>
				<div>
					<Typography.Title level={5} style={{ margin: 0 }}>
						节点耗时 / 花费
					</Typography.Title>
					<Typography.Text type="secondary" style={{ fontSize: 12 }}>
						按节点定位最慢、最贵或最需要优化的步骤。
					</Typography.Text>
				</div>
				<Space size={16} wrap>
					<Space size={6}>
						<ClockCircleOutlined />
						<Typography.Text>
							累计节点耗时 {formatDurationSeconds(totalDurationSeconds)}
						</Typography.Text>
					</Space>
					<Space size={6}>
						<CloudServerOutlined />
						<Typography.Text>{podCount} pods</Typography.Text>
					</Space>
					<Space size={6}>
						<DollarOutlined />
						<Typography.Text>
							{hasCost ? formatCost(totalCost) : "成本待接入"}
						</Typography.Text>
					</Space>
				</Space>
			</div>
			<Table
				rowKey="key"
				size="small"
				columns={columns}
				dataSource={rows}
				pagination={false}
				scroll={{ x: 1200, y: 220 }}
				rowClassName={(row) =>
					row.node.id === selectedNodeId ? "ant-table-row-selected" : ""
				}
				onRow={(row) => ({
					onClick: () => onInspectNode(row.node),
					style: { cursor: "pointer" },
				})}
			/>
		</section>
	);
}
