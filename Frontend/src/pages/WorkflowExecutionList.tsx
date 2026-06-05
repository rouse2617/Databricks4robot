import { MoreOutlined, ReloadOutlined } from "@ant-design/icons";
import {
	Alert,
	Button,
	Checkbox,
	DatePicker,
	Dropdown,
	Empty,
	Input,
	Modal,
	message,
	Select,
	Skeleton,
	Space,
	Table,
	Tag,
	Tooltip,
	Typography,
} from "antd";
import dayjs, { type Dayjs } from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { listPipelineRuns, type PipelineRun } from "../api/pipelineApi";
import {
	deleteWorkflow,
	type ListWorkflowsParams,
	listWorkflows,
	type WorkflowSummary,
} from "../api/workflowApi";
import { DurationPanel } from "../components/common/DurationPanel";
import { WorkflowLabels } from "../components/common/WorkflowLabels";
import {
	STATUS_ACCENT_COLORS,
	STATUS_COLORS,
	WORKFLOW_PHASES,
} from "../lib/constants";
import { toAssetStyleId } from "../lib/idDisplay";
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
import { buildWorkflowExecutionUrl } from "../lib/workflowNavigation";

const { RangePicker } = DatePicker;
dayjs.extend(relativeTime);

interface WorkflowExecutionListProps {
	active?: boolean;
}

interface ExecutionListRow extends WorkflowSummary {
	key: string;
	runId?: string;
	templateId?: string;
	templateName?: string;
	templateVersion?: number;
	triggerSource?: string;
	failureSummary?: string;
	liveAvailable: boolean;
	historyOnly: boolean;
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

const renderTimestamp = (value?: string) => {
	if (!value) return "-";
	const parsed = dayjs(value);
	if (!parsed.isValid()) return new Date(value).toLocaleString();
	const absolute = parsed.format("YYYY-MM-DD HH:mm:ss");
	return (
		<Tooltip title={parsed.toDate().toLocaleString()}>
			<div style={{ lineHeight: 1.35 }}>
				<div>{absolute}</div>
				<Typography.Text type="secondary" style={{ fontSize: 12 }}>
					{parsed.fromNow()}
				</Typography.Text>
			</div>
		</Tooltip>
	);
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

const normalizeSearchValue = (value: string) => value.trim().toLowerCase();

const isFailedExecutionStatus = (status?: string) => {
	const normalized = (status || "").trim().toLowerCase();
	return normalized === "failed" || normalized === "error";
};

const formatTriggerSource = (value?: string) => {
	switch ((value || "").trim()) {
		case "manual":
			return "手动运行";
		case "asset_run":
			return "资产运行";
		case "batch":
			return "批量运行";
		case "api":
			return "API 运行";
		default:
			return "";
	}
};

const renderPipelineIdentity = (record: ExecutionListRow) => {
	if (!record.templateId && !record.templateName && !record.templateVersion) {
		return <Typography.Text type="secondary">-</Typography.Text>;
	}

	const versionLabel =
		typeof record.templateVersion === "number"
			? `v${record.templateVersion}`
			: "-";
	const rawId = record.templateId || record.templateName || "";
	const displayId = record.templateId
		? toAssetStyleId(record.templateId)
		: record.templateName || "-";
	const copyText = rawId
		? `${rawId}:${versionLabel}`
		: versionLabel !== "-"
			? versionLabel
			: "";

	return (
		<div style={{ minWidth: 0 }}>
			<Typography.Text
				strong
				className="pipeline-execution-pipeline-id"
				copyable={copyText ? { text: copyText } : false}
				ellipsis={{
					tooltip: rawId
						? `流水线 ID: ${rawId}，版本: ${versionLabel}`
						: "未关联流水线模板",
				}}
			>
				{displayId} : {versionLabel}
			</Typography.Text>
			{record.templateName ? (
				<Typography.Text
					type="secondary"
					style={{ display: "block", fontSize: 12 }}
					ellipsis={{ tooltip: record.templateName }}
				>
					{record.templateName}
				</Typography.Text>
			) : null}
		</div>
	);
};

const buildExecutionRows = (
	runs: PipelineRun[],
	workflows: WorkflowSummary[],
): ExecutionListRow[] => {
	const workflowByName = new Map(
		workflows.map((workflow) => [workflow.name, workflow] as const),
	);
	const seenWorkflowNames = new Set<string>();
	const rows: ExecutionListRow[] = [];

	for (const run of runs) {
		const workflowName = run.workflowName || run.pipelineName || run.id;
		const live = workflowByName.get(workflowName);
		if (workflowName) {
			seenWorkflowNames.add(workflowName);
		}
		rows.push({
			key: run.id || workflowName,
			name: workflowName,
			status: live?.status || run.status,
			nodeCount: live?.nodeCount ?? run.nodeCount,
			assetCount: run.assetCount,
			createdAt: run.createdAt,
			finishedAt: run.finishedAt,
			labels: live?.labels,
			totalEstimatedCost:
				run.totalEstimatedCost ?? live?.totalEstimatedCost ?? null,
			runId: run.id,
			templateId: run.templateId,
			templateName: run.templateName || run.pipelineName,
			templateVersion: run.templateVersion ?? undefined,
			triggerSource: run.triggerSource,
			failureSummary: run.message || undefined,
			liveAvailable: !!live,
			historyOnly: !live,
		});
	}

	for (const workflow of workflows) {
		if (seenWorkflowNames.has(workflow.name)) continue;
		rows.push({
			key: workflow.name,
			name: workflow.name,
			status: workflow.status,
			nodeCount: workflow.nodeCount,
			assetCount: workflow.assetCount,
			createdAt: workflow.createdAt,
			finishedAt: workflow.finishedAt,
			labels: workflow.labels,
			totalEstimatedCost:
				workflow.totalEstimatedCost ?? workflow.estimatedCostUsd ?? null,
			liveAvailable: true,
			historyOnly: false,
		});
	}

	rows.sort((a, b) => {
		const left = new Date(a.createdAt).getTime();
		const right = new Date(b.createdAt).getTime();
		return right - left;
	});
	return rows;
};

const filterExecutionRows = (
	rows: ExecutionListRow[],
	params: {
		status?: string;
		name?: string;
		label?: string[];
		createdAfter?: string;
		finishedBefore?: string;
	},
) => {
	const status = params.status?.trim();
	const name = normalizeSearchValue(params.name || "");
	const createdAfter = params.createdAfter
		? new Date(params.createdAfter)
		: null;
	const finishedBefore = params.finishedBefore
		? new Date(params.finishedBefore)
		: null;

	return rows.filter((row) => {
		if (status && row.status !== status) return false;
		if (name && !row.name.toLowerCase().includes(name)) {
			const templateName = (row.templateName || "").toLowerCase();
			if (!templateName.includes(name)) return false;
		}
		if (createdAfter && new Date(row.createdAt) < createdAfter) return false;
		if (finishedBefore) {
			if (!row.finishedAt) return false;
			if (new Date(row.finishedAt) > finishedBefore) return false;
		}
		if (params.label && params.label.length > 0) {
			const rowLabels = new Set(
				getDisplayLabelEntries(row.labels).map(([key, value]) =>
					serializeWorkflowLabel(key, value),
				),
			);
			for (const label of params.label) {
				if (!rowLabels.has(label)) return false;
			}
		}
		return true;
	});
};

export function WorkflowExecutionList({
	active = true,
}: WorkflowExecutionListProps) {
	const [searchParams, setSearchParams] = useSearchParams();
	const [items, setItems] = useState<ExecutionListRow[]>([]);
	const [loading, setLoading] = useState(false);
	const [initializedOnce, setInitializedOnce] = useState(false);
	const [error, setError] = useState<WorkflowErrorState | null>(null);
	const [statusFilter, setStatusFilter] = useState<string | undefined>(
		normalizeStatus(searchParams.get("status")),
	);
	const [draftStatusFilter, setDraftStatusFilter] = useState<
		string | undefined
	>(normalizeStatus(searchParams.get("status")));
	const [nameSearch, setNameSearch] = useState(
		searchParams.get("name")?.trim() ?? "",
	);
	const [draftNameSearch, setDraftNameSearch] = useState(
		searchParams.get("name")?.trim() ?? "",
	);
	const [labelFilter, setLabelFilter] = useState<string[]>(() => {
		const labels = searchParams.getAll("label");
		return Array.from(
			new Set(labels.map((label) => label.trim()).filter(Boolean)),
		);
	});
	const [draftLabelFilter, setDraftLabelFilter] = useState<string[]>(() => {
		const labels = searchParams.getAll("label");
		return Array.from(
			new Set(labels.map((label) => label.trim()).filter(Boolean)),
		);
	});
	const [dateRange, setDateRange] = useState<[Dayjs | null, Dayjs | null]>([
		parseDate(searchParams.get("createdAfter")),
		parseDate(searchParams.get("finishedBefore")),
	]);
	const [draftDateRange, setDraftDateRange] = useState<
		[Dayjs | null, Dayjs | null]
	>([
		parseDate(searchParams.get("createdAfter")),
		parseDate(searchParams.get("finishedBefore")),
	]);
	const [operationLoading, setOperationLoading] = useState<string | null>(null);
	const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);
	const [bulkDeleteOpen, setBulkDeleteOpen] = useState(false);
	const [bulkDeleting, setBulkDeleting] = useState(false);
	const [pendingOperation, setPendingOperation] = useState<{
		record: ExecutionListRow;
		operation: WorkflowOperationConfig;
	} | null>(null);
	const [page, setPage] = useState(1);
	const [pageSize, setPageSize] = useState(20);
	const [openDropdown, setOpenDropdown] = useState<string | null>(null);
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
		setDraftStatusFilter((prev) => (prev === nextStatus ? prev : nextStatus));
		setNameSearch((prev) => (prev === nextName ? prev : nextName));
		setDraftNameSearch((prev) => (prev === nextName ? prev : nextName));
		setLabelFilter((prev) =>
			arraysEqual(prev, nextLabelFilter) ? prev : nextLabelFilter,
		);
		setDraftLabelFilter((prev) =>
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
		setDraftDateRange((prev) => {
			if (
				datesEqual(prev[0], nextCreatedAfter) &&
				datesEqual(prev[1], nextFinishedBefore)
			) {
				return prev;
			}
			return [nextCreatedAfter, nextFinishedBefore];
		});
	}, [searchParams]);

	const syncAppliedFiltersToUrl = useCallback(() => {
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
				name: normalizeSearchValue(nameSearch) || undefined,
				label: labelFilter.length ? labelFilter : undefined,
				createdAfter: dateRange[0]?.toISOString(),
				finishedBefore: dateRange[1]?.toISOString(),
			};
			const [workflowsResult, runsResult] = await Promise.allSettled([
				listWorkflows(params),
				listPipelineRuns(),
			]);
			const workflows =
				workflowsResult.status === "fulfilled"
					? workflowsResult.value.items || []
					: [];
			const runs = runsResult.status === "fulfilled" ? runsResult.value : [];
			const merged = buildExecutionRows(runs, workflows);
			const filtered = filterExecutionRows(merged, params);
			if (
				workflowsResult.status === "rejected" &&
				runsResult.status === "rejected"
			) {
				throw workflowsResult.reason;
			}
			setItems(filtered);
			setSelectedRowKeys((prev) =>
				prev.filter((key) => filtered.some((item) => item.key === key)),
			);
		} catch (err) {
			console.error(err);
			setError(describeWorkflowError(err));
			setItems([]);
		} finally {
			setLoading(false);
			setInitializedOnce(true);
		}
	}, [statusFilter, nameSearch, labelFilter, dateRange]);

	useEffect(() => {
		if (active) {
			refresh();
		}
	}, [active, refresh]);

	useEffect(() => {
		setPage(1);
		syncAppliedFiltersToUrl();
	}, [syncAppliedFiltersToUrl]);

	const filtersDirty =
		draftStatusFilter !== statusFilter ||
		draftNameSearch !== nameSearch ||
		!arraysEqual(draftLabelFilter, labelFilter) ||
		!datesEqual(draftDateRange[0], dateRange[0]) ||
		!datesEqual(draftDateRange[1], dateRange[1]);

	const applyFilters = useCallback(() => {
		setStatusFilter(draftStatusFilter);
		setNameSearch(draftNameSearch.trim());
		setLabelFilter(draftLabelFilter);
		setDateRange(draftDateRange);
		setPage(1);
	}, [draftDateRange, draftLabelFilter, draftNameSearch, draftStatusFilter]);

	const resetFilters = useCallback(() => {
		setDraftStatusFilter(undefined);
		setDraftNameSearch("");
		setDraftLabelFilter([]);
		setDraftDateRange([null, null]);
		setStatusFilter(undefined);
		setNameSearch("");
		setLabelFilter([]);
		setDateRange([null, null]);
		setPage(1);
	}, []);

	const labelCheckboxOptions = useMemo(() => {
		const labels = new Set<string>();
		for (const item of items) {
			for (const [key, value] of getDisplayLabelEntries(item.labels)) {
				labels.add(serializeWorkflowLabel(key, value));
			}
		}
		const availableLabelOptions = Array.from(labels).sort((a, b) =>
			a.localeCompare(b),
		);
		return availableLabelOptions.map((label) => ({
			label: (() => {
				const separatorIndex = label.indexOf("=");
				const key =
					separatorIndex >= 0 ? label.slice(0, separatorIndex) : label;
				const value =
					separatorIndex >= 0 ? label.slice(separatorIndex + 1) : "";
				return (
					<Tag>
						{formatWorkflowLabelKey(key)}
						{value ? ` · ${value}` : ""}
					</Tag>
				);
			})(),
			value: label,
		}));
	}, [items]);

	const selectedRows = useMemo(
		() => items.filter((item) => selectedRowKeys.includes(item.key)),
		[items, selectedRowKeys],
	);
	const selectedLiveRows = useMemo(
		() => selectedRows.filter((item) => item.liveAvailable),
		[selectedRows],
	);
	const executionSummary = useMemo(() => {
		const counts = new Map<string, number>();
		for (const item of items) {
			counts.set(item.status, (counts.get(item.status) || 0) + 1);
		}
		return Array.from(counts.entries()).sort((a, b) =>
			a[0].localeCompare(b[0]),
		);
	}, [items]);

	const executeOperation = useCallback(
		async (
			record: ExecutionListRow,
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
		(record: ExecutionListRow, key: WorkflowOperationKey) => {
			const operation = getAvailableWorkflowOperationConfigs(record).find(
				(item) => item.key === key,
			);
			if (!operation) return;

			if (
				operation.key === "delete" ||
				operation.key === "terminate" ||
				operation.key === "resubmit" ||
				operation.key === "retry"
			) {
				setPendingOperation({ record, operation });
				return;
			}

			void executeOperation(record, operation);
		},
		[executeOperation],
	);

	const confirmPendingOperation = useCallback(async () => {
		if (!pendingOperation) return;
		const { record, operation } = pendingOperation;
		setPendingOperation(null);
		await executeOperation(record, operation);
	}, [executeOperation, pendingOperation]);

	const confirmBulkDelete = useCallback(async () => {
		if (selectedLiveRows.length === 0) return;
		setBulkDeleting(true);
		try {
			const results = await Promise.allSettled(
				selectedLiveRows.map((row) => deleteWorkflow(row.name)),
			);
			const failedCount = results.filter(
				(result) => result.status === "rejected",
			).length;
			const deletedCount = results.length - failedCount;
			if (deletedCount > 0)
				message.success(`已删除 ${deletedCount} 条执行记录`);
			if (failedCount > 0) message.error(`${failedCount} 条执行记录删除失败`);
			setSelectedRowKeys([]);
			setBulkDeleteOpen(false);
			await refresh();
		} finally {
			setBulkDeleting(false);
		}
	}, [refresh, selectedLiveRows]);

	const columns = [
		{
			title: "名称",
			dataIndex: "name",
			key: "name",
			width: 300,
			fixed: "left" as const,
			render: (name: string, record: ExecutionListRow) => {
				const displayId = toAssetStyleId(record.runId ?? name);
				const copyId = record.runId ?? name;
				const triggerLabel = formatTriggerSource(record.triggerSource);
				const failedSummary =
					isFailedExecutionStatus(record.status) && record.failureSummary
						? record.failureSummary
						: undefined;
				const detailHref = buildWorkflowExecutionUrl(record.name, record.runId);
				return (
					<div style={{ minWidth: 0 }}>
						<Typography.Text strong ellipsis={{ tooltip: name }}>
							{name}
						</Typography.Text>
						<Typography.Text
							type="secondary"
							copyable={{ text: copyId }}
							style={{ display: "block", fontSize: 12 }}
							ellipsis={{
								tooltip: record.runId ? `完整任务 ID: ${record.runId}` : name,
							}}
						>
							ID: {displayId}
						</Typography.Text>
						<Space size={[6, 6]} wrap style={{ marginTop: 4 }}>
							{triggerLabel ? <Tag color="gold">{triggerLabel}</Tag> : null}
							{record.historyOnly ? <Tag>已归档</Tag> : null}
						</Space>
						{failedSummary ? (
							<div className="pipeline-execution-failure">
								<Typography.Text
									type="danger"
									style={{ fontSize: 12 }}
									ellipsis={{ tooltip: failedSummary }}
								>
									失败原因：{failedSummary}
								</Typography.Text>
								<Button
									type="link"
									danger
									size="small"
									className="pipeline-execution-failure__link"
									onClick={(event) => {
										event.stopPropagation();
										navigate(detailHref);
									}}
								>
									查看失败详情
								</Button>
							</div>
						) : isFailedExecutionStatus(record.status) ? (
							<Button
								type="link"
								danger
								size="small"
								className="pipeline-execution-failure__link pipeline-execution-failure__link--standalone"
								onClick={(event) => {
									event.stopPropagation();
									navigate(detailHref);
								}}
							>
								查看失败详情
							</Button>
						) : null}
					</div>
				);
			},
		},
		{
			title: "流水线",
			key: "pipeline",
			width: 240,
			render: (_: unknown, record: ExecutionListRow) =>
				renderPipelineIdentity(record),
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
			render: (nodeCount: number) => nodeCount,
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
			width: 220,
			render: renderTimestamp,
		},
		{
			title: "完成时间",
			dataIndex: "finishedAt",
			key: "finishedAt",
			width: 220,
			render: renderTimestamp,
		},
		{
			title: "操作",
			key: "actions",
			width: 128,
			fixed: "right" as const,
			render: (_: unknown, record: ExecutionListRow) => {
				const menuItems = record.liveAvailable
					? getWorkflowOperationMenuItems(record)
					: [];
				const hasOperationLoading = operationLoading?.startsWith(
					`${record.name}:`,
				);
				const detailHref = record.runId
					? `/pipeline/executions/${encodeURIComponent(record.name)}?runId=${encodeURIComponent(record.runId)}`
					: `/pipeline/executions/${encodeURIComponent(record.name)}`;

				return (
					<div className="pipeline-execution-actions">
						<Button
							type="link"
							size="small"
							onClick={(event) => {
								event.stopPropagation();
								navigate(detailHref);
							}}
						>
							查看
						</Button>
						<Dropdown
							open={openDropdown === record.name}
							onOpenChange={(open) =>
								setOpenDropdown(open ? record.name : null)
							}
							menu={{
								items: menuItems,
								onClick: ({ key, domEvent }) => {
									domEvent.stopPropagation();
									setOpenDropdown(null);
									runOperation(record, key as WorkflowOperationKey);
								},
							}}
							trigger={["click"]}
						>
							<Button
								size="small"
								icon={<MoreOutlined />}
								loading={hasOperationLoading}
								aria-label="更多操作"
								onClick={(event) => {
									event.stopPropagation();
									if (menuItems.length === 0) {
										message.info(
											record.historyOnly
												? "该记录已归档，Argo Workflow 已清理或不可用，暂无可用运行操作"
												: "当前状态暂无可用操作",
										);
									}
								}}
							/>
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
				<Tooltip
					title={
						selectedRowKeys.length === 0
							? "请先选择执行记录"
							: selectedLiveRows.length === 0
								? "当前选中项包含已归档记录，无法直接删除 Argo Workflow"
								: undefined
					}
				>
					<span>
						<Button
							danger
							disabled={selectedLiveRows.length === 0}
							onClick={() => setBulkDeleteOpen(true)}
						>
							批量删除
							{selectedLiveRows.length > 0
								? `（${selectedLiveRows.length}）`
								: ""}
						</Button>
					</span>
				</Tooltip>
				<Button icon={<ReloadOutlined />} onClick={refresh} loading={loading}>
					刷新
				</Button>
			</div>

			{items.length > 0 ? (
				<div
					style={{
						display: "flex",
						flexWrap: "wrap",
						gap: 8,
						marginBottom: 16,
					}}
				>
					<Tag>共 {items.length} 条</Tag>
					{executionSummary.map(([status, count]) => (
						<Tag
							key={status}
							color={
								STATUS_COLORS[status] ||
								STATUS_ACCENT_COLORS[status] ||
								"default"
							}
						>
							{status} {count}
						</Tag>
					))}
				</div>
			) : null}

			<div className="pipeline-execution-filters" style={{ gap: 8 }}>
				<Select
					id="workflow-execution-status-filter"
					allowClear
					placeholder="状态筛选"
					style={{ minWidth: 140, flex: "1 1 160px" }}
					value={draftStatusFilter}
					onChange={(val) => setDraftStatusFilter(val)}
					options={WORKFLOW_PHASES.map((status) => ({
						label: status,
						value: status,
					}))}
				/>
				<Input.Search
					id="workflow-execution-name-search"
					name="workflowExecutionName"
					allowClear
					placeholder="按名称搜索"
					style={{ minWidth: 220, flex: "1 1 220px" }}
					value={draftNameSearch}
					onChange={(event) => setDraftNameSearch(event.target.value)}
					onSearch={applyFilters}
				/>
				<RangePicker
					id="workflow-execution-date-range"
					value={draftDateRange}
					placeholder={["创建开始时间", "完成截止时间"]}
					onChange={(values) =>
						setDraftDateRange([values?.[0] ?? null, values?.[1] ?? null])
					}
					style={{ minWidth: 320, flex: "1 1 260px" }}
				/>
				<Button type="primary" onClick={applyFilters} disabled={!filtersDirty}>
					应用
				</Button>
				<Button onClick={resetFilters}>重置</Button>
			</div>

			{labelCheckboxOptions.length > 0 ? (
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
						value={draftLabelFilter}
						onChange={(values) => setDraftLabelFilter(values as string[])}
					/>
				</div>
			) : null}

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
			) : !error && items.length === 0 ? (
				<Empty description="暂无执行记录，部署流水线后将自动生成" />
			) : (
				<div className="pipeline-execution-table">
					<Table
						dataSource={items}
						columns={columns}
						rowKey="key"
						loading={loading}
						rowSelection={{
							selectedRowKeys,
							onChange: (keys) => setSelectedRowKeys(keys as string[]),
							getCheckboxProps: (record) => ({
								disabled: !record.liveAvailable,
								title: record.liveAvailable
									? undefined
									: "该记录已归档，当前不支持直接删除 Argo Workflow",
							}),
						}}
						scroll={{ x: 1660 }}
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
								navigate(buildWorkflowExecutionUrl(record.name, record.runId));
							},
							style: { cursor: "pointer" },
						})}
						pagination={{
							current: page,
							pageSize,
							showSizeChanger: true,
							showQuickJumper: true,
							pageSizeOptions: ["10", "20", "50", "100"],
							showTotal: (total, range) =>
								`第 ${range[0]}-${range[1]} 条 / 共 ${total} 条 · 共 ${Math.max(1, Math.ceil(total / pageSize))} 页`,
							onChange: (nextPage, nextPageSize) => {
								setPage(nextPage);
								setPageSize(nextPageSize);
							},
						}}
					/>
				</div>
			)}
			<Modal
				open={!!pendingOperation}
				title={
					pendingOperation
						? `确认${pendingOperation.operation.title} ${pendingOperation.record.name}?`
						: ""
				}
				okText={pendingOperation?.operation.title}
				cancelText="取消"
				okButtonProps={{ danger: pendingOperation?.operation.danger }}
				onOk={confirmPendingOperation}
				onCancel={() => setPendingOperation(null)}
			>
				{pendingOperation?.operation.key === "resubmit" ||
				pendingOperation?.operation.key === "retry" ? (
					<p>将基于当前工作流再次提交执行。</p>
				) : null}
			</Modal>
			<Modal
				open={bulkDeleteOpen}
				title={`删除选中的 ${selectedLiveRows.length} 条执行记录？`}
				okText="删除"
				cancelText="取消"
				okButtonProps={{ danger: true, loading: bulkDeleting }}
				onOk={confirmBulkDelete}
				onCancel={() => setBulkDeleteOpen(false)}
			>
				<p>删除后不可恢复。正在运行的工作流请先确认不再需要。</p>
			</Modal>
		</div>
	);
}
