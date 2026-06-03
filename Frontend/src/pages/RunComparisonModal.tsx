import { Modal, Table, Tag, Typography } from "antd";
import type { WorkflowSummary } from "../api/workflowApi";
import { STATUS_COLORS } from "../lib/constants";
import { formatDuration } from "../lib/workflow-utils";

const { Text } = Typography;

export interface RunComparisonModalProps {
	open: boolean;
	items: WorkflowSummary[];
	onClose: () => void;
}

interface ComparisonRow {
	key: string;
	label: string;
	values: (string | number | React.ReactNode)[];
}

function formatCost(value: number | undefined | null): string {
	if (value == null) return "—";
	return `$${value.toFixed(value < 0.01 ? 4 : 2)}`;
}

export function RunComparisonModal({
	open,
	items,
	onClose,
}: RunComparisonModalProps) {
	if (items.length < 2) return null;

	const rows: ComparisonRow[] = [
		{
			key: "status",
			label: "状态",
			values: items.map((item) => (
				<Tag key={item.name} color={STATUS_COLORS[item.status] || "default"}>
					{item.status}
				</Tag>
			)),
		},
		{
			key: "nodeCount",
			label: "节点数",
			values: items.map((item) => item.nodeCount),
		},
		{
			key: "cost",
			label: "估算成本",
			values: items.map((item) => {
				const cost =
					typeof item.totalEstimatedCost === "number"
						? item.totalEstimatedCost
						: null;
				return formatCost(cost);
			}),
		},
		{
			key: "duration",
			label: "耗时",
			values: items.map((item) =>
				formatDuration(item.createdAt, item.finishedAt),
			),
		},
		{
			key: "created",
			label: "创建时间",
			values: items.map((item) =>
				item.createdAt ? new Date(item.createdAt).toLocaleString() : "—",
			),
		},
		{
			key: "finished",
			label: "完成时间",
			values: items.map((item) =>
				item.finishedAt ? new Date(item.finishedAt).toLocaleString() : "—",
			),
		},
	];

	const columns = [
		{
			title: "指标",
			dataIndex: "label",
			key: "label",
			width: 120,
			fixed: "left" as const,
			render: (label: string) => <Text strong>{label}</Text>,
		},
		...items.map((item, idx) => ({
			title: (
				<div>
					<div>
						<Text strong ellipsis={{ tooltip: item.name }}>
							{item.name}
						</Text>
					</div>
				</div>
			),
			key: `col-${idx}`,
			dataIndex: "values",
			width: 200,
			render: (_: unknown, row: ComparisonRow) => row.values[idx],
		})),
	];

	return (
		<Modal
			title="运行对比"
			open={open}
			onCancel={onClose}
			footer={null}
			width={Math.min(800 + items.length * 220, 1400)}
			data-testid="run-comparison-modal"
		>
			<Table
				dataSource={rows}
				columns={columns}
				pagination={false}
				size="small"
				rowKey="key"
				bordered
				scroll={{ x: "max-content" }}
			/>
		</Modal>
	);
}
