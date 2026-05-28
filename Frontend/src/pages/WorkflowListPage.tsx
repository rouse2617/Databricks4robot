import { MoreOutlined, ReloadOutlined } from "@ant-design/icons";
import {
	Button,
	Card,
	Checkbox,
	DatePicker,
	Dropdown,
	Input,
	Modal,
	message,
	Select,
	Table,
	Tag,
} from "antd";
import dayjs, { type Dayjs } from "dayjs";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import {
	type ListWorkflowsParams,
	listWorkflows,
	type WorkflowSummary,
} from "../api/workflowApi";
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

const { RangePicker } = DatePicker;

const LABEL_SEPARATOR = "=";

const parseDate = (value: string | null): Dayjs | null => {
	if (!value) return null;
	const parsed = dayjs(value);
	return parsed.isValid() ? parsed : null;
};

const serializeLabel = (key: string, value: string): string =>
	`${key}${LABEL_SEPARATOR}${value}`;

const datesEqual = (a: Dayjs | null, b: Dayjs | null): boolean => {
	if (!a && !b) return true;
	if (!a || !b) return false;
	return a.isSame(b);
};

const arraysEqual = (a: string[], b: string[]): boolean => {
	if (a.length !== b.length) return false;
	return a.every((value, idx) => value === b[idx]);
};

const normalizeStatus = (value: string | null): string | undefined => {
	const trimmed = value?.trim();
	if (!trimmed) return undefined;
	return WORKFLOW_PHASES.includes(trimmed as (typeof WORKFLOW_PHASES)[number])
		? trimmed
		: undefined;
};

export default function WorkflowListPage() {
	const [searchParams, setSearchParams] = useSearchParams();
	const [items, setItems] = useState<WorkflowSummary[]>([]);
	const [loading, setLoading] = useState(false);
	const [statusFilter, setStatusFilter] = useState<string | undefined>(
		normalizeStatus(searchParams.get("status")),
	);
	const [nameSearch, setNameSearch] = useState(
		searchParams.get("name")?.trim() ?? "",
	);
	const [debouncedNameSearch, setDebouncedNameSearch] = useState(
		searchParams.get("name")?.trim().toLowerCase() ?? "",
	);
	const [labelFilter, setLabelFilter] = useState<string[]>(() => {
		const labels = searchParams.getAll("label");
		return Array.from(
			new Set(labels.map((label) => label.trim()).filter(Boolean)),
		);
	});
	const [dateRange, setDateRange] = useState<[Dayjs | null, Dayjs | null]>([
		parseDate(searchParams.get("createdAfter")),
		parseDate(searchParams.get("finishedBefore")),
	]);
	const [operationLoading, setOperationLoading] = useState<string | null>(null);
	const navigate = useNavigate();

	useEffect(() => {
		const nextStatus = normalizeStatus(searchParams.get("status"));
		const nextName = searchParams.get("name")?.trim() ?? "";
		const nextLabelFilter = Array.from(
			new Set(
				searchParams
					.getAll("label")
					.map((label) => label.trim())
					.filter(Boolean),
			),
		);
		const nextCreatedAfter = parseDate(searchParams.get("createdAfter"));
		const nextFinishedBefore = parseDate(searchParams.get("finishedBefore"));

		setStatusFilter((prev) => (prev === nextStatus ? prev : nextStatus));
		setNameSearch((prev) => (prev === nextName ? prev : nextName));
		setDebouncedNameSearch((prev) => {
			const normalized = nextName.toLowerCase();
			return prev === normalized ? prev : normalized;
		});
		setLabelFilter((prev) =>
			arraysEqual(prev, nextLabelFilter) ? prev : nextLabelFilter,
		);
		setDateRange((prev) => {
			if (
				datesEqual(prev[0], nextCreatedAfter) &&
				datesEqual(prev[1], nextFinishedBefore)
			) {
				return prev;
			}
			return [nextCreatedAfter, nextFinishedBefore];
		});
	}, [searchParams]);

	useEffect(() => {
		const next = new URLSearchParams();
		if (statusFilter) next.set("status", statusFilter);
		if (nameSearch) next.set("name", nameSearch.trim());
		for (const label of labelFilter) {
			if (label) next.append("label", label);
		}
		if (dateRange[0]) next.set("createdAfter", dateRange[0].toISOString());
		if (dateRange[1]) next.set("finishedBefore", dateRange[1].toISOString());

		if (next.toString() !== searchParams.toString()) {
			setSearchParams(next, { replace: true });
		}
	}, [
		dateRange,
		labelFilter,
		nameSearch,
		searchParams,
		setSearchParams,
		statusFilter,
	]);

	const refresh = useCallback(async () => {
		setLoading(true);
		try {
			const params: ListWorkflowsParams = {
				status: statusFilter,
				name: debouncedNameSearch || undefined,
				label: labelFilter.length ? labelFilter : undefined,
				createdAfter: dateRange[0]?.toISOString(),
				finishedBefore: dateRange[1]?.toISOString(),
			};
			const res = await listWorkflows(params);
			setItems(res.items || []);
		} catch (err) {
			console.error(err);
		} finally {
			setLoading(false);
		}
	}, [statusFilter, debouncedNameSearch, labelFilter, dateRange]);

	useEffect(() => {
		const timer = window.setTimeout(() => {
			setDebouncedNameSearch(nameSearch.trim().toLowerCase());
		}, 300);

		return () => window.clearTimeout(timer);
	}, [nameSearch]);

	useEffect(() => {
		refresh();
	}, [refresh]);

	const labelCheckboxOptions = useMemo(() => {
		const labels = new Set<string>();
		for (const item of items) {
			for (const [key, value] of Object.entries(item.labels ?? {})) {
				labels.add(serializeLabel(key, value));
			}
		}
		const availableLabelOptions = Array.from(labels).sort((a, b) =>
			a.localeCompare(b),
		);
		return availableLabelOptions.map((label) => ({
			label: <Tag>{label}</Tag>,
			value: label,
		}));
	}, [items]);

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
					display: "flex",
					gap: 8,
					flexWrap: "wrap",
					marginBottom: 16,
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
				<RangePicker
					value={dateRange}
					placeholder={["创建开始时间", "完成截止时间"]}
					onChange={(values) =>
						setDateRange([values?.[0] ?? null, values?.[1] ?? null])
					}
					style={{ width: 320 }}
				/>
			</div>
			<div
				style={{
					marginBottom: 16,
					display: "flex",
					alignItems: "center",
					flexWrap: "wrap",
					gap: 8,
				}}
			>
				<span style={{ color: "rgba(0,0,0,0.65)" }}>标签筛选：</span>
				<Checkbox.Group
					options={labelCheckboxOptions}
					value={labelFilter}
					onChange={(values) => setLabelFilter(values as string[])}
				/>
			</div>

			<Table
				dataSource={items}
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
