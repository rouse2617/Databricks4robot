import { MoreOutlined, ReloadOutlined } from "@ant-design/icons";
import {
	Button,
	Card,
	DatePicker,
	Dropdown,
	Input,
	Modal,
	message,
	Select,
	Space,
	Table,
	Tag,
	Tooltip,
	Typography,
} from "antd";
import dayjs, { type Dayjs } from "dayjs";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { listWorkflows, type WorkflowSummary } from "../api/workflowApi";
import { DurationPanel } from "../components/common/DurationPanel";
import { WorkflowLabels } from "../components/common/WorkflowLabels";
import {
	STATUS_ACCENT_COLORS,
	STATUS_COLORS,
	STATUS_ICONS,
	WORKFLOW_PHASE_LABELS,
	WORKFLOW_PHASES,
	WORKFLOW_SUMMARY_ALWAYS_VISIBLE,
} from "../lib/constants";
import {
	getAvailableWorkflowOperationConfigs,
	getWorkflowOperationMenuItems,
	type WorkflowOperationConfig,
	type WorkflowOperationKey,
} from "../lib/workflow-operations";
import {
	formatWorkflowLabelKey,
	getDisplayLabelEntries,
	serializeWorkflowLabel,
} from "../lib/workflowLabels";

const { RangePicker } = DatePicker;
const { Title, Text } = Typography;

const parseDate = (value: string | null): Dayjs | null => {
	if (!value) return null;
	const parsed = dayjs(value);
	return parsed.isValid() ? parsed : null;
};

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

interface WorkflowListFilters {
	status?: string;
	name: string;
	label: string[];
	createdAfter: Dayjs | null;
	finishedBefore: Dayjs | null;
}

function filterWorkflows(
	items: WorkflowSummary[],
	filters: WorkflowListFilters,
): WorkflowSummary[] {
	const nameNeedle = filters.name.trim().toLowerCase();

	return items.filter((item) => {
		if (filters.status && item.status !== filters.status) {
			return false;
		}
		if (nameNeedle && !item.name.toLowerCase().includes(nameNeedle)) {
			return false;
		}
		if (filters.createdAfter) {
			const created = dayjs(item.createdAt);
			if (!created.isValid() || created.isBefore(filters.createdAfter)) {
				return false;
			}
		}
		if (filters.finishedBefore && item.finishedAt) {
			const finished = dayjs(item.finishedAt);
			if (finished.isValid() && finished.isAfter(filters.finishedBefore)) {
				return false;
			}
		}
		for (const raw of filters.label) {
			const [key, value] = raw.split("=", 2);
			if (!key || !value || item.labels?.[key] !== value) {
				return false;
			}
		}
		return true;
	});
}

export default function WorkflowListPage() {
	const [searchParams, setSearchParams] = useSearchParams();
	const [allItems, setAllItems] = useState<WorkflowSummary[]>([]);
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
			const res = await listWorkflows();
			setAllItems(res.items || []);
		} catch (err) {
			console.error(err);
			message.error("加载流水线运行列表失败");
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		const timer = window.setTimeout(() => {
			setDebouncedNameSearch(nameSearch.trim().toLowerCase());
		}, 300);

		return () => window.clearTimeout(timer);
	}, [nameSearch]);

	useEffect(() => {
		refresh();
	}, [refresh]);

	const baseFilters = useMemo<WorkflowListFilters>(
		() => ({
			name: debouncedNameSearch,
			label: labelFilter,
			createdAfter: dateRange[0],
			finishedBefore: dateRange[1],
		}),
		[dateRange, debouncedNameSearch, labelFilter],
	);

	const itemsBeforeStatus = useMemo(
		() => filterWorkflows(allItems, baseFilters),
		[allItems, baseFilters],
	);

	const items = useMemo(
		() =>
			filterWorkflows(allItems, {
				...baseFilters,
				status: statusFilter,
			}),
		[allItems, baseFilters, statusFilter],
	);

	const statusCounts = useMemo(() => {
		const counts = Object.fromEntries(
			WORKFLOW_PHASES.map((status) => [status, 0]),
		) as Record<(typeof WORKFLOW_PHASES)[number], number>;

		for (const item of itemsBeforeStatus) {
			if (item.status in counts) {
				counts[item.status as (typeof WORKFLOW_PHASES)[number]] += 1;
			}
		}

		return counts;
	}, [itemsBeforeStatus]);

	const labelOptions = useMemo(() => {
		const labels = new Set<string>();
		for (const item of itemsBeforeStatus) {
			for (const [key, value] of getDisplayLabelEntries(item.labels)) {
				labels.add(serializeWorkflowLabel(key, value));
			}
		}
		return Array.from(labels)
			.sort((a, b) => a.localeCompare(b))
			.map((label) => {
				const [key, ...rest] = label.split("=");
				const value = rest.join("=");
				return {
					label: `${formatWorkflowLabelKey(key)} · ${value}`,
					value: label,
				};
			});
	}, [itemsBeforeStatus]);

	const hasActiveFilters =
		Boolean(statusFilter) ||
		Boolean(debouncedNameSearch) ||
		labelFilter.length > 0 ||
		dateRange[0] != null ||
		dateRange[1] != null;

	const clearFilters = useCallback(() => {
		setStatusFilter(undefined);
		setNameSearch("");
		setDebouncedNameSearch("");
		setLabelFilter([]);
		setDateRange([null, null]);
	}, []);

	const toggleStatusFilter = useCallback((status: string) => {
		setStatusFilter((prev) => (prev === status ? undefined : status));
	}, []);

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
			render: (name: string) => (
				<Text code style={{ fontSize: 12 }}>
					{name}
				</Text>
			),
		},
		{
			title: "状态",
			dataIndex: "status",
			key: "status",
			width: 108,
			render: (status: string) => (
				<Tag
					color={STATUS_COLORS[status] || "default"}
					icon={STATUS_ICONS[status]}
				>
					{WORKFLOW_PHASE_LABELS[status as (typeof WORKFLOW_PHASES)[number]] ??
						status}
				</Tag>
			),
		},
		{
			title: (
				<Tooltip title="Argo 节点总数，含 DAG 根节点">
					<span>节点</span>
				</Tooltip>
			),
			dataIndex: "nodeCount",
			key: "nodeCount",
			width: 72,
			align: "center" as const,
		},
		{
			title: "标签",
			dataIndex: "labels",
			key: "labels",
			width: 220,
			render: (labels?: Record<string, string>) => (
				<WorkflowLabels labels={labels} />
			),
		},
		{
			title: "耗时",
			key: "duration",
			width: 120,
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
			width: 168,
			render: (t: string) => (t ? new Date(t).toLocaleString() : "—"),
		},
		{
			title: "操作",
			key: "actions",
			width: 132,
			fixed: "right" as const,
			render: (_: unknown, record: WorkflowSummary) => {
				const menuItems = getWorkflowOperationMenuItems(record);
				const hasOperationLoading = operationLoading?.startsWith(
					`${record.name}:`,
				);

				return (
					<Space size={4}>
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
					</Space>
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
					justifyContent: "space-between",
					gap: 12,
					marginBottom: 16,
					flexWrap: "wrap",
				}}
			>
				<div>
					<Title level={4} style={{ margin: 0 }}>
						流水线运行
					</Title>
					<Text type="secondary" style={{ fontSize: 12 }}>
						共 {allItems.length} 条记录
						{hasActiveFilters ? ` · 当前筛选 ${items.length} 条` : ""}
					</Text>
				</div>
				<Space wrap>
					{hasActiveFilters && <Button onClick={clearFilters}>清除筛选</Button>}
					<Button icon={<ReloadOutlined />} onClick={refresh} loading={loading}>
						刷新
					</Button>
				</Space>
			</div>

			<div
				style={{
					display: "grid",
					gridTemplateColumns: "repeat(auto-fit, minmax(148px, 1fr))",
					gap: 12,
					marginBottom: 16,
				}}
			>
				{WORKFLOW_PHASES.map((status) => {
					const count = statusCounts[status];
					if (count === 0 && !WORKFLOW_SUMMARY_ALWAYS_VISIBLE.has(status)) {
						return null;
					}

					const accentColor = STATUS_ACCENT_COLORS[status];
					const selected = statusFilter === status;

					return (
						<Card
							key={status}
							size="small"
							hoverable
							onClick={() => toggleStatusFilter(status)}
							styles={{
								body: {
									alignItems: "center",
									display: "flex",
									gap: 10,
									padding: "10px 12px",
								},
							}}
							style={{
								borderColor: selected ? accentColor : undefined,
								borderLeft: `4px solid ${accentColor}`,
								boxShadow: selected ? `0 0 0 1px ${accentColor}` : undefined,
								cursor: "pointer",
							}}
						>
							<span style={{ color: accentColor, fontSize: 18 }}>
								{STATUS_ICONS[status]}
							</span>
							<span style={{ color: "rgba(0, 0, 0, 0.65)" }}>
								{WORKFLOW_PHASE_LABELS[status]}
							</span>
							<strong style={{ fontSize: 18, marginLeft: "auto" }}>
								{count}
							</strong>
						</Card>
					);
				})}
			</div>

			<Card size="small" style={{ marginBottom: 16 }}>
				<Space wrap style={{ width: "100%" }}>
					<Select
						allowClear
						placeholder="状态"
						style={{ width: 128 }}
						value={statusFilter}
						onChange={(val) => setStatusFilter(val)}
						options={WORKFLOW_PHASES.map((status) => ({
							label: WORKFLOW_PHASE_LABELS[status],
							value: status,
						}))}
					/>
					<Input.Search
						allowClear
						placeholder="按名称搜索"
						style={{ width: 280 }}
						value={nameSearch}
						onChange={(event) => setNameSearch(event.target.value)}
						onSearch={(value) =>
							setDebouncedNameSearch(value.trim().toLowerCase())
						}
					/>
					<RangePicker
						value={dateRange}
						placeholder={["创建起始", "完成截止"]}
						onChange={(values) =>
							setDateRange([values?.[0] ?? null, values?.[1] ?? null])
						}
						style={{ width: 300 }}
					/>
					{labelOptions.length > 0 && (
						<Select
							mode="multiple"
							allowClear
							placeholder="标签"
							style={{ minWidth: 220, maxWidth: 420 }}
							value={labelFilter}
							onChange={(values) => setLabelFilter(values)}
							options={labelOptions}
							maxTagCount="responsive"
						/>
					)}
				</Space>
			</Card>

			<Table
				dataSource={items}
				columns={columns}
				rowKey="name"
				loading={loading}
				scroll={{ x: 980 }}
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
