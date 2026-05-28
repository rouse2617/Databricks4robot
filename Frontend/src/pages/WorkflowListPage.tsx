import {
	CheckCircleOutlined,
	ClockCircleOutlined,
	CloseCircleOutlined,
	ExclamationCircleOutlined,
	LoadingOutlined,
	ReloadOutlined,
} from "@ant-design/icons";
import { Button, Card, Input, Select, Table, Tag } from "antd";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { listWorkflows, type WorkflowSummary } from "../api/workflowApi";

const WORKFLOW_STATUSES = [
	"Running",
	"Succeeded",
	"Failed",
	"Error",
	"Pending",
] as const;

const STATUS_COLORS: Record<string, string> = {
	Succeeded: "success",
	Running: "processing",
	Pending: "warning",
	Failed: "error",
	Error: "error",
};

const STATUS_ACCENT_COLORS: Record<string, string> = {
	Succeeded: "#52c41a",
	Running: "#1677ff",
	Pending: "#faad14",
	Failed: "#ff4d4f",
	Error: "#cf1322",
};

const STATUS_ICONS: Record<string, React.ReactNode> = {
	Succeeded: <CheckCircleOutlined />,
	Running: <LoadingOutlined />,
	Pending: <ClockCircleOutlined />,
	Failed: <CloseCircleOutlined />,
	Error: <ExclamationCircleOutlined />,
};

export default function WorkflowListPage() {
	const [items, setItems] = useState<WorkflowSummary[]>([]);
	const [loading, setLoading] = useState(false);
	const [statusFilter, setStatusFilter] = useState<string | undefined>();
	const [nameSearch, setNameSearch] = useState("");
	const [debouncedNameSearch, setDebouncedNameSearch] = useState("");
	const navigate = useNavigate();

	const refresh = useCallback(async () => {
		setLoading(true);
		try {
			const res = await listWorkflows();
			setItems(res.items || []);
		} catch (err) {
			console.error(err);
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		refresh();
	}, [refresh]);

	useEffect(() => {
		const timer = window.setTimeout(() => {
			setDebouncedNameSearch(nameSearch.trim().toLowerCase());
		}, 300);

		return () => window.clearTimeout(timer);
	}, [nameSearch]);

	const statusCounts = useMemo(() => {
		const counts = Object.fromEntries(
			WORKFLOW_STATUSES.map((status) => [status, 0]),
		) as Record<(typeof WORKFLOW_STATUSES)[number], number>;

		for (const item of items) {
			if (item.status in counts) {
				counts[item.status as (typeof WORKFLOW_STATUSES)[number]] += 1;
			}
		}

		return counts;
	}, [items]);

	const filtered = useMemo(
		() =>
			items.filter((item) => {
				const matchesStatus = statusFilter
					? item.status === statusFilter
					: true;
				const matchesName = debouncedNameSearch
					? item.name.toLowerCase().includes(debouncedNameSearch)
					: true;

				return matchesStatus && matchesName;
			}),
		[debouncedNameSearch, items, statusFilter],
	);

	const columns = [
		{
			title: "名称",
			dataIndex: "name",
			key: "name",
			ellipsis: true,
		},
		{
			title: "状态",
			dataIndex: "status",
			key: "status",
			width: 120,
			render: (s: string) => (
				<Tag color={STATUS_COLORS[s] || "default"}>{s}</Tag>
			),
		},
		{
			title: "节点数",
			dataIndex: "nodeCount",
			key: "nodeCount",
			width: 100,
		},
		{
			title: "创建时间",
			dataIndex: "createdAt",
			key: "createdAt",
			width: 180,
			render: (t: string) => (t ? new Date(t).toLocaleString() : "-"),
		},
		{
			title: "完成时间",
			dataIndex: "finishedAt",
			key: "finishedAt",
			width: 180,
			render: (t?: string) => (t ? new Date(t).toLocaleString() : "-"),
		},
		{
			title: "操作",
			key: "actions",
			width: 100,
			render: (_: unknown, record: WorkflowSummary) => (
				<Button
					type="link"
					size="small"
					onClick={(e) => {
						e.stopPropagation();
						navigate(`/workflows/${record.name}`);
					}}
				>
					查看
				</Button>
			),
		},
	];

	return (
		<div style={{ padding: 24 }}>
			<div
				style={{
					display: "flex",
					alignItems: "center",
					gap: 12,
					marginBottom: 16,
				}}
			>
				<h2 style={{ margin: 0 }}>流水线运行</h2>
				<Button icon={<ReloadOutlined />} onClick={refresh} loading={loading}>
					刷新
				</Button>
			</div>
			<div
				style={{
					display: "grid",
					gridTemplateColumns: "repeat(auto-fit, minmax(160px, 1fr))",
					gap: 12,
					marginBottom: 16,
				}}
			>
				{WORKFLOW_STATUSES.map((status) => {
					const accentColor = STATUS_ACCENT_COLORS[status];

					return (
						<Card
							key={status}
							size="small"
							styles={{
								body: {
									alignItems: "center",
									display: "flex",
									gap: 10,
									padding: "10px 12px",
								},
							}}
							style={{
								borderColor: accentColor,
								borderLeft: `4px solid ${accentColor}`,
							}}
						>
							<span style={{ color: accentColor, fontSize: 18 }}>
								{STATUS_ICONS[status]}
							</span>
							<span style={{ color: "rgba(0, 0, 0, 0.65)" }}>{status}</span>
							<strong style={{ fontSize: 18, marginLeft: "auto" }}>
								{statusCounts[status]}
							</strong>
						</Card>
					);
				})}
			</div>
			<div
				style={{
					marginBottom: 16,
					display: "flex",
					gap: 8,
					flexWrap: "wrap",
				}}
			>
				<Select
					allowClear
					placeholder="状态筛选"
					style={{ width: 140 }}
					value={statusFilter}
					onChange={(val) => setStatusFilter(val)}
					options={WORKFLOW_STATUSES.map((status) => ({
						label: status,
						value: status,
					}))}
				/>
				<Input.Search
					allowClear
					placeholder="按名称搜索"
					style={{ maxWidth: 320, minWidth: 220 }}
					value={nameSearch}
					onChange={(event) => setNameSearch(event.target.value)}
					onSearch={(value) =>
						setDebouncedNameSearch(value.trim().toLowerCase())
					}
				/>
			</div>
			<Table
				dataSource={filtered}
				columns={columns}
				rowKey="name"
				loading={loading}
				locale={{ emptyText: "暂无流水线运行" }}
				onRow={(record) => ({
					onClick: () => navigate(`/workflows/${record.name}`),
					style: { cursor: "pointer" },
				})}
				pagination={{
					pageSize: 20,
					showSizeChanger: true,
					showTotal: (t) => `共 ${t} 条`,
				}}
			/>
		</div>
	);
}
