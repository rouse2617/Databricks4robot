import { MoreOutlined, ReloadOutlined } from "@ant-design/icons";
import {
	Button,
	Card,
	Dropdown,
	Input,
	Modal,
	message,
	Select,
	Table,
	Tag,
} from "antd";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { listWorkflows, type WorkflowSummary } from "../api/workflowApi";
import { DurationPanel } from "../components/common/DurationPanel";
import { WorkflowLabels } from "../components/common/WorkflowLabels";
import {
	STATUS_ACCENT_COLORS,
	STATUS_COLORS,
	STATUS_ICONS,
	WORKFLOW_PHASES,
} from "../lib/constants";
import {
	getAvailableWorkflowOperationConfigs,
	getWorkflowOperationMenuItems,
	type WorkflowOperationConfig,
	type WorkflowOperationKey,
} from "../lib/workflow-operations";

export default function WorkflowListPage() {
	const [items, setItems] = useState<WorkflowSummary[]>([]);
	const [loading, setLoading] = useState(false);
	const [statusFilter, setStatusFilter] = useState<string | undefined>();
	const [nameSearch, setNameSearch] = useState("");
	const [debouncedNameSearch, setDebouncedNameSearch] = useState("");
	const [operationLoading, setOperationLoading] = useState<string | null>(null);
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
			WORKFLOW_PHASES.map((status) => [status, 0]),
		) as Record<(typeof WORKFLOW_PHASES)[number], number>;

		for (const item of items) {
			if (item.status in counts) {
				counts[item.status as (typeof WORKFLOW_PHASES)[number]] += 1;
			}
		}

		return counts;
	}, [items]);

	const executeOperation = useCallback(
		async (
			record: WorkflowSummary,
			operation: WorkflowOperationConfig,
		): Promise<void> => {
			const loadingKey = `${record.name}:${operation.key}`;
			setOperationLoading(loadingKey);
			try {
				await operation.run();
				message.success(`${operation.title}已提交`);
				await refresh();
			} catch (err) {
				message.error(`${operation.title}失败: ${String(err)}`);
			} finally {
				setOperationLoading(null);
			}
		},
		[refresh],
	);

	const runOperation = useCallback(
		(record: WorkflowSummary, key: WorkflowOperationKey) => {
			const operation = getAvailableWorkflowOperationConfigs(record).find(
				(item) => item.key === key,
			);
			if (!operation) return;

			if (operation.key === "delete" || operation.key === "terminate") {
				Modal.confirm({
					title: `确认${operation.title} ${record.name}?`,
					okText: operation.title,
					okButtonProps: { danger: operation.danger },
					cancelText: "取消",
					onOk: () => executeOperation(record, operation),
				});
				return;
			}

			executeOperation(record, operation);
		},
		[executeOperation],
	);

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
			title: "标签",
			dataIndex: "labels",
			key: "labels",
			width: 260,
			render: (labels?: Record<string, string>) => (
				<WorkflowLabels labels={labels} />
			),
		},
		{
			title: "耗时",
			key: "duration",
			width: 140,
			render: (_: unknown, record: WorkflowSummary) => (
				<DurationPanel
					phase={record.status}
					startedAt={record.createdAt}
					finishedAt={record.finishedAt}
				/>
			),
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
			width: 150,
			render: (_: unknown, record: WorkflowSummary) => {
				const menuItems = getWorkflowOperationMenuItems(record);
				const hasOperationLoading = operationLoading?.startsWith(
					`${record.name}:`,
				);

				return (
					<div style={{ display: "flex", gap: 4 }}>
						<Button
							type="link"
							size="small"
							onClick={(event) => {
								event.stopPropagation();
								navigate(`/workflows/${record.name}`);
							}}
						>
							查看
						</Button>
						<Dropdown
							menu={{
								items: menuItems,
								onClick: ({ key }) =>
									runOperation(record, key as WorkflowOperationKey),
							}}
							trigger={["click"]}
							disabled={menuItems.length === 0}
						>
							<Button
								size="small"
								icon={<MoreOutlined />}
								loading={hasOperationLoading}
								onClick={(event) => event.stopPropagation()}
							>
								操作
							</Button>
						</Dropdown>
					</div>
				);
			},
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
				{WORKFLOW_PHASES.map((status) => {
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
					options={WORKFLOW_PHASES.map((status) => ({
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
