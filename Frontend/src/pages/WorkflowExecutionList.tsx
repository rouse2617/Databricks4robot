import { MoreOutlined, ReloadOutlined } from "@ant-design/icons";
import {
	Alert,
	Button,
	Card,
	Checkbox,
	DatePicker,
	Dropdown,
	Empty,
	Input,
	Modal,
	Skeleton,
	Tag,
	Tooltip,
	Typography,
	Table,
	message,
	Select,
} from "antd";
import dayjs, { type Dayjs } from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
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
dayjs.extend(relativeTime);

const LABEL_SEPARATOR = "=";

interface WorkflowExecutionListProps {
	active?: boolean;
}

type WorkflowErrorKind = "network" | "service-unavailable";

type WorkflowErrorState = {
	kind: WorkflowErrorKind;
	message: string;
};

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

const describeWorkflowError = (err: unknown): WorkflowErrorState => {
	const message =
		err instanceof Error
			? err.message
			: typeof err === "string"
				? err
				: "请求失败，请稍后重试";

	const normalized = message.toLowerCase();
	if (
		normalized.includes("failed to fetch") ||
		normalized.includes("network") ||
		normalized.includes("econnrefused") ||
		normalized.includes("enotfound") ||
		normalized.includes("timeout") ||
		normalized.includes("typeerror")
	) {
		return {
			kind: "network",
			message,
		};
	}

	return {
		kind: "service-unavailable",
		message,
	};
};

const getErrorTitle = (kind: WorkflowErrorKind): string =>
	kind === "network" ? "网络错误" : "服务不可用";

export function WorkflowExecutionList({
	active = true,
}: WorkflowExecutionListProps) {
	const [searchParams, setSearchParams] = useSearchParams();
	const [items, setItems] = useState<WorkflowSummary[]>([]);
	const [loading, setLoading] = useState(false);
	const [initializedOnce, setInitializedOnce] = useState(false);
	const [error, setError] = useState<WorkflowErrorState | null>(null);
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
	const [page, setPage] = useState(1);
	const [pageSize, setPageSize] = useState(20);
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
		const next = new URLSearchParams(searchParams);
		if (statusFilter) next.set("status", statusFilter);
		else next.delete("status");
		if (nameSearch) next.set("name", nameSearch.trim());
		else next.delete("name");
		next.delete("label");
		for (const label of labelFilter) {
			if (label) next.append("label", label);
		}
		if (dateRange[0]) next.set("createdAfter", dateRange[0].toISOString());
		else next.delete("createdAfter");
		if (dateRange[1]) next.set("finishedBefore", dateRange[1].toISOString());
		else next.delete("finishedBefore");

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
		setError(null);
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
			setError(describeWorkflowError(err));
			setItems([]);
		} finally {
			setLoading(false);
			setInitializedOnce(true);
		}
	}, [statusFilter, debouncedNameSearch, labelFilter, dateRange]);

	useEffect(() => {
		if (active) {
			refresh();
		}
	}, [active, refresh]);

	useEffect(() => {
		const timer = window.setTimeout(() => {
			setDebouncedNameSearch(nameSearch.trim().toLowerCase());
		}, 300);

		return () => window.clearTimeout(timer);
	}, [nameSearch]);

	useEffect(() => {
		setPage(1);
	}, [statusFilter, debouncedNameSearch, labelFilter, dateRange]);

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
			width: 260,
			ellipsis: true,
		},
		{
			title: "状态",
			dataIndex: "status",
			key: "status",
			width: 130,
			render: (s: string) => (
				<Tag
					color={STATUS_COLORS[s] || STATUS_ACCENT_COLORS[s] || "default"}
					style={{ padding: "2px 8px" }}
				>
					{s}
				</Tag>
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
			width: 240,
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
			render: (t: string) => {
				if (!t) return "-";
				const created = dayjs(t);
				if (!created.isValid()) return new Date(t).toLocaleString();
				return (
					<Tooltip title={created.toLocaleString()}>{created.fromNow()}</Tooltip>
				);
			},
		},
		{
			title: "完成时间",
			dataIndex: "finishedAt",
			key: "finishedAt",
			width: 180,
			render: (t?: string) => {
				if (!t) return "-";
				const finished = dayjs(t);
				if (!finished.isValid()) return new Date(t).toLocaleString();
				return (
					<Tooltip title={finished.toLocaleString()}>
						{finished.fromNow()}
					</Tooltip>
				);
			},
		},
		{
			title: "操作",
			key: "actions",
			width: 110,
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

	const showSkeleton = loading && !initializedOnce;

	return (
		<div className="pipeline-execution-list">
			<div
				style={{
					display: "flex",
					alignItems: "center",
					gap: 12,
					marginBottom: 16,
				}}
			>
				<Typography.Title level={4} style={{ margin: 0 }}>
					流水线执行记录
				</Typography.Title>
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
				{showSkeleton
					? WORKFLOW_PHASES.map((status) => <Card key={status} loading />)
					: WORKFLOW_PHASES.map((status) => {
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
								<span style={{ color: "rgba(0, 0, 0, 0.65)" }}>
									{status}
								</span>
								<strong style={{ fontSize: 18, marginLeft: "auto" }}>
									{statusCounts[status]}
								</strong>
							</Card>
						);
					})}
			</div>

			<div className="pipeline-execution-filters" style={{ gap: 8 }}>
				<Select
					allowClear
					placeholder="状态筛选"
					style={{ minWidth: 140, flex: "1 1 160px" }}
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
					style={{ minWidth: 220, flex: "1 1 220px" }}
					value={nameSearch}
					onChange={(event) => setNameSearch(event.target.value)}
					onSearch={(value) => setDebouncedNameSearch(value.trim().toLowerCase())}
				/>
				<RangePicker
					value={dateRange}
					placeholder={["创建开始时间", "完成截止时间"]}
					onChange={(values) =>
						setDateRange([values?.[0] ?? null, values?.[1] ?? null])
					}
					style={{ minWidth: 320, flex: "1 1 260px" }}
				/>
			</div>

			<div
				style={{
					display: "flex",
					alignItems: "center",
					flexWrap: "wrap",
					gap: 8,
					marginBottom: 16,
				}}
			>
				<Typography.Text type="secondary">标签筛选：</Typography.Text>
				<Checkbox.Group
					options={labelCheckboxOptions}
					value={labelFilter}
					onChange={(values) => setLabelFilter(values as string[])}
				/>
			</div>

			{error ? (
				<Alert
					type="error"
					showIcon
					message={getErrorTitle(error.kind)}
					description={error.message}
					action={
						<Button size="small" onClick={refresh} loading={loading}>
							重试
						</Button>
					}
					style={{ marginBottom: 16 }}
				/>
			) : null}

			{showSkeleton ? (
				<Skeleton
					active
					paragraph={{ rows: 8 }}
					title={false}
					style={{ width: "100%" }}
				/>
				) : items.length === 0 ? (
					<Empty description="暂无执行记录，部署流水线后将自动生成" />
				) : (
				<div className="pipeline-execution-table">
					<Table
						dataSource={items}
						columns={columns}
						rowKey="name"
						loading={loading}
						rowClassName={() => "pipeline-execution-table-row"}
						onRow={(record) => ({
							onClick: (event) => {
								const el = event.target as HTMLElement;
								if (
									el.closest(
										"a, button, input, textarea, select, label, [role='checkbox'], .ant-pagination, .ant-pagination-item, .ant-dropdown",
									)
								) {
									return;
								}
								navigate(`/workflows/${record.name}`);
							},
							style: { cursor: "pointer" },
						})}
						pagination={{
								current: page,
								pageSize,
								showSizeChanger: true,
								pageSizeOptions: ["10", "20", "50", "100"],
								showTotal: (total) => `共 ${total} 条`,
								onChange: (nextPage, nextPageSize) => {
									setPage(nextPage);
									setPageSize(nextPageSize);
								},
							}}
					/>
				</div>
			)}
		</div>
	);
}
