import { LockOutlined, MoreOutlined, ReloadOutlined } from "@ant-design/icons";
import {
	Alert,
	App,
	Button,
	DatePicker,
	Dropdown,
	Empty,
	Input,
	Modal,
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
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import {
	listPipelineRuns,
	listPipelines,
	type PipelineRun,
} from "../api/pipelineApi";
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
import { RunComparisonModal } from "./RunComparisonModal";

const { RangePicker } = DatePicker;
dayjs.extend(relativeTime);

interface WorkflowExecutionListProps {
	active?: boolean;
	batchJobId?: string;
	/** Bumps embedded batch subtask lists after parent job metadata loads. */
	_batchListKey?: string | number;
	embedded?: boolean;
	onSelectionChange?: (items: WorkflowSummary[]) => void;
	title?: string;
}

type WorkflowErrorKind = "network" | "service-unavailable";

type WorkflowErrorState = {
	kind: WorkflowErrorKind;
	message: string;
};

const activeWorkflowStatuses = new Set(["Running", "Pending", "Suspended"]);

const STALE_ACTIVE_RUN_MS = 48 * 60 * 60 * 1000;

const isActiveWorkflowStatus = (status?: string): boolean =>
	activeWorkflowStatuses.has(status ?? "");

const isStaleRunningWorkflow = (record: WorkflowSummary): boolean => {
	if (!isActiveWorkflowStatus(record.status)) return false;
	if (!record.createdAt) return false;
	return (
		Date.now() - new Date(record.createdAt).getTime() > STALE_ACTIVE_RUN_MS
	);
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

const getWorkflowLabel = (
	labels: Record<string, string> | undefined,
	key: string,
): string | undefined => {
	if (!labels) return undefined;
	const normalizedKey = formatWorkflowLabelKey(key);
	for (const [labelKey, value] of Object.entries(labels)) {
		if (formatWorkflowLabelKey(labelKey) === normalizedKey && value) {
			return value;
		}
	}
	return undefined;
};

const getWorkflowEstimatedCost = (record: WorkflowSummary): number | null => {
	if (typeof record.estimatedCostUsd === "number") {
		return record.estimatedCostUsd;
	}
	if (typeof record.totalEstimatedCost === "number") {
		return record.totalEstimatedCost;
	}
	return null;
};

const getEstimatedCostTooltip = (record: WorkflowSummary): string => {
	if (isActiveWorkflowStatus(record.status)) {
		return "运行完成并写入节点快照后会显示估算成本";
	}
	if (record.nodeCount === 0) {
		return "本次运行没有可计费节点";
	}
	return "本次运行尚未生成成本快照，可能是历史运行、无可计费 Pod，或节点资源耗时未回填";
};

const renderEstimatedCost = (_: unknown, record: WorkflowSummary) => {
	const cost = getWorkflowEstimatedCost(record);
	if (cost == null) {
		return (
			<Tooltip title={getEstimatedCostTooltip(record)}>
				<Typography.Text type="secondary">—</Typography.Text>
			</Tooltip>
		);
	}
	return (
		<Tooltip title="估算总成本，非 GCP Billing 最终对账金额">
			<Typography.Text strong>
				${cost.toFixed(cost < 0.01 ? 4 : 2)}
			</Typography.Text>
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

const workflowNameForRun = (run: PipelineRun): string =>
	run.workflowName || run.pipelineName || run.id;

const runMatchesFilters = (
	run: PipelineRun,
	liveWorkflow: WorkflowSummary | undefined,
	params: ListWorkflowsParams,
): boolean => {
	const status = liveWorkflow?.status ?? run.status;
	if (params.status && status !== params.status) {
		return false;
	}

	const name = workflowNameForRun(run).toLowerCase();
	const pipelineName = (run.pipelineName ?? "").toLowerCase();
	const runID = (run.id ?? "").toLowerCase();
	if (
		params.name &&
		!name.includes(params.name) &&
		!pipelineName.includes(params.name) &&
		!runID.includes(params.name)
	) {
		return false;
	}

	if (params.createdAfter) {
		const createdAt = dayjs(liveWorkflow?.createdAt ?? run.createdAt);
		if (createdAt.isValid() && createdAt.isBefore(dayjs(params.createdAfter))) {
			return false;
		}
	}

	if (params.finishedBefore) {
		const finishedAt = dayjs(liveWorkflow?.finishedAt ?? run.finishedAt);
		if (
			finishedAt.isValid() &&
			finishedAt.isAfter(dayjs(params.finishedBefore))
		) {
			return false;
		}
	}

	if (params.label?.length) {
		const labels = liveWorkflow?.labels ?? {};
		return params.label.every((filter) => {
			const separatorIndex = filter.indexOf("=");
			if (separatorIndex < 0) {
				return filter in labels;
			}
			const key = filter.slice(0, separatorIndex);
			const value = filter.slice(separatorIndex + 1);
			return labels[key] === value;
		});
	}

	return true;
};

const workflowSummaryFromRun = (
	run: PipelineRun,
	liveWorkflow?: WorkflowSummary,
): WorkflowSummary => {
	const assetId = run.assetIds?.[0];
	const mergedLabels = {
		...(liveWorkflow?.labels ?? {}),
		...(assetId ? { asset_id: assetId } : {}),
	};
	return {
		name: workflowNameForRun(run),
		status: liveWorkflow?.status ?? run.status,
		nodeCount: run.nodeCount ?? liveWorkflow?.nodeCount ?? 0,
		createdAt: liveWorkflow?.createdAt ?? run.createdAt,
		finishedAt: liveWorkflow?.finishedAt ?? run.finishedAt,
		labels: Object.keys(mergedLabels).length > 0 ? mergedLabels : undefined,
		estimatedCostUsd: liveWorkflow?.estimatedCostUsd,
		totalEstimatedCost:
			typeof run.totalEstimatedCost === "number"
				? run.totalEstimatedCost
				: liveWorkflow?.totalEstimatedCost,
	};
};

const mergeLedgerRunsWithLiveWorkflows = (
	liveWorkflows: WorkflowSummary[],
	pipelineRuns: PipelineRun[],
	params: ListWorkflowsParams,
): WorkflowSummary[] => {
	const liveByName = new Map(liveWorkflows.map((item) => [item.name, item]));
	const seen = new Set<string>();
	const ledgerItems = pipelineRuns
		.filter((run) =>
			runMatchesFilters(run, liveByName.get(run.workflowName), params),
		)
		.map((run) => {
			const name = workflowNameForRun(run);
			seen.add(name);
			if (run.workflowName) {
				seen.add(run.workflowName);
			}
			return workflowSummaryFromRun(run, liveByName.get(run.workflowName));
		});
	const liveOnlyItems = liveWorkflows.filter((item) => !seen.has(item.name));
	return [...ledgerItems, ...liveOnlyItems].sort((a, b) => {
		const left = dayjs(a.createdAt).valueOf();
		const right = dayjs(b.createdAt).valueOf();
		return right - left;
	});
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
	batchJobId,
	embedded = false,
	onSelectionChange,
	title,
}: WorkflowExecutionListProps) {
	const { message: messageApi } = App.useApp();
	const [searchParams, setSearchParams] = useSearchParams();
	const [items, setItems] = useState<WorkflowSummary[]>([]);
	const [runIdsByWorkflowName, setRunIdsByWorkflowName] = useState<
		Record<string, string>
	>({});
	const [templateVersionsByWorkflowName, setTemplateVersionsByWorkflowName] =
		useState<Record<string, number>>({});
	const [templateIdsByWorkflowName, setTemplateIdsByWorkflowName] = useState<
		Record<string, string>
	>({});
	const [nodeCountsByWorkflowName, setNodeCountsByWorkflowName] = useState<
		Record<string, number>
	>({});
	const [scopeByWorkflowName, setScopeByWorkflowName] = useState<
		Record<string, string>
	>({});
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
	const [versionFilter, setVersionFilter] = useState<string | undefined>(
		searchParams.get("templateVersion")?.trim() || undefined,
	);
	const [draftVersionFilter, setDraftVersionFilter] = useState<
		string | undefined
	>(searchParams.get("templateVersion")?.trim() || undefined);
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
	const [selectedWorkflowNames, setSelectedWorkflowNames] = useState<string[]>(
		[],
	);
	const [bulkDeleteOpen, setBulkDeleteOpen] = useState(false);
	const [bulkDeleting, setBulkDeleting] = useState(false);
	const [compareOpen, setCompareOpen] = useState(false);
	const [compareItems, setCompareItems] = useState<WorkflowSummary[]>([]);
	const [pendingOperation, setPendingOperation] = useState<{
		record: WorkflowSummary;
		operation: WorkflowOperationConfig;
	} | null>(null);
	const [page, setPage] = useState(1);
	const [pageSize, setPageSize] = useState(20);
	const [serverTotal, setServerTotal] = useState(0);
	const navigate = useNavigate();
	const refreshInFlightRef = useRef(false);
	const isBatchScope = Boolean(batchJobId);

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
		const nextVersion =
			searchParams.get("templateVersion")?.trim() || undefined;
		setVersionFilter((prev) => (prev === nextVersion ? prev : nextVersion));
		setDraftVersionFilter((prev) =>
			prev === nextVersion ? prev : nextVersion,
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
		if (versionFilter) next.set("templateVersion", versionFilter);
		else next.delete("templateVersion");
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
		versionFilter,
		searchParams,
		setSearchParams,
		statusFilter,
	]);

	const refresh = useCallback(async () => {
		if (refreshInFlightRef.current) return;
		refreshInFlightRef.current = true;
		setLoading(true);
		setError(null);
		try {
			if (isBatchScope && batchJobId) {
				const runResponse = await listPipelineRuns({
					view: "summary",
					batchJobId,
					page,
					pageSize,
					status: statusFilter,
				});
				const pipelineRuns = runResponse.items ?? [];
				setServerTotal(runResponse.total ?? pipelineRuns.length);
				setRunIdsByWorkflowName(
					Object.fromEntries(
						pipelineRuns
							.filter((run) => run.workflowName && run.id)
							.map((run) => [run.workflowName, run.id] as const),
					),
				);
				setTemplateVersionsByWorkflowName(
					Object.fromEntries(
						pipelineRuns
							.filter((run) => run.workflowName && run.templateVersion)
							.map(
								(run) =>
									[run.workflowName, run.templateVersion as number] as const,
							),
					),
				);
				setTemplateIdsByWorkflowName(
					Object.fromEntries(
						pipelineRuns
							.filter((run) => run.workflowName && run.templateId)
							.map(
								(run) => [run.workflowName, run.templateId as string] as const,
							),
					),
				);
				setNodeCountsByWorkflowName(
					Object.fromEntries(
						pipelineRuns
							.filter((run) => run.workflowName)
							.map((run) => [run.workflowName, run.nodeCount] as const),
					),
				);
				setScopeByWorkflowName(
					Object.fromEntries(
						pipelineRuns
							.filter((run) => run.workflowName && run.scope)
							.map((run) => [run.workflowName, run.scope as string] as const),
					),
				);
				const summaries = pipelineRuns.map((run) =>
					workflowSummaryFromRun(run),
				);
				setItems(summaries);
				setSelectedWorkflowNames((prev) =>
					prev.filter((name) => summaries.some((item) => item.name === name)),
				);
				return;
			}

			const params: ListWorkflowsParams = {
				status: statusFilter,
				name: nameSearch.trim().toLowerCase() || undefined,
				label: labelFilter.length ? labelFilter : undefined,
				createdAfter: dateRange[0]?.toISOString(),
				finishedBefore: dateRange[1]?.toISOString(),
			};
			const shouldLoadLiveWorkflows = labelFilter.length > 0;
			const [liveWorkflowResponse, pipelineRunResponse, templates] =
				await Promise.all([
					shouldLoadLiveWorkflows
						? listWorkflows(params).catch((err) => {
								console.warn("live workflow list unavailable", err);
								return { items: [] };
							})
						: Promise.resolve({ items: [] }),
				listPipelineRuns({ view: "summary", excludeBatch: true }).catch(() => ({
					items: [],
					total: 0,
				})),
				listPipelines({ pageSize: 200 })
					.then((r) => r.items)
					.catch(() => []),
			]);
			const liveWorkflows = liveWorkflowResponse.items || [];
			const pipelineRuns = pipelineRunResponse.items ?? [];
			setServerTotal(pipelineRunResponse.total ?? pipelineRuns.length);
			setRunIdsByWorkflowName(
				Object.fromEntries(
					pipelineRuns
						.filter((run) => run.workflowName && run.id)
						.map((run) => [run.workflowName, run.id] as const),
				),
			);
			setTemplateVersionsByWorkflowName(
				Object.fromEntries([
					...pipelineRuns
						.filter((run) => run.workflowName && run.templateVersion)
						.map(
							(run) =>
								[run.workflowName, run.templateVersion as number] as const,
						),
					...templates
						.filter((t) => t.name)
						.flatMap((t) =>
							liveWorkflows
								.filter((item) => item.name.startsWith(`${t.name}-`))
								.map((item) => [item.name, t.version] as const),
						),
				]),
			);
			setTemplateIdsByWorkflowName(
				Object.fromEntries(
					pipelineRuns
						.filter((run) => run.workflowName && run.templateId)
						.map(
							(run) => [run.workflowName, run.templateId as string] as const,
						),
				),
			);
			setNodeCountsByWorkflowName(
				Object.fromEntries(
					pipelineRuns
						.filter((run) => run.workflowName)
						.map((run) => [run.workflowName, run.nodeCount] as const),
				),
			);
			setScopeByWorkflowName(
				Object.fromEntries(
					pipelineRuns
						.filter((run) => run.workflowName && run.scope)
						.map((run) => [run.workflowName, run.scope as string] as const),
				),
			);
			const enrichedItems = mergeLedgerRunsWithLiveWorkflows(
				liveWorkflows,
				pipelineRuns,
				params,
			);
			setItems(enrichedItems);
			setSelectedWorkflowNames((prev) =>
				prev.filter((name) => enrichedItems.some((item) => item.name === name)),
			);
		} catch (err) {
			console.error(err);
			setError(describeWorkflowError(err));
			setItems([]);
		} finally {
			setLoading(false);
			setInitializedOnce(true);
			refreshInFlightRef.current = false;
		}
	}, [
		batchJobId,
		dateRange,
		isBatchScope,
		labelFilter,
		nameSearch,
		page,
		pageSize,
		statusFilter,
	]);

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
		draftVersionFilter !== versionFilter ||
		draftStatusFilter !== statusFilter ||
		draftNameSearch !== nameSearch ||
		!arraysEqual(draftLabelFilter, labelFilter) ||
		!datesEqual(draftDateRange[0], dateRange[0]) ||
		!datesEqual(draftDateRange[1], dateRange[1]);

	const applyFilters = useCallback(() => {
		setVersionFilter(draftVersionFilter);
		setStatusFilter(draftStatusFilter);
		setNameSearch(draftNameSearch.trim());
		setLabelFilter(draftLabelFilter);
		setDateRange(draftDateRange);
		setPage(1);
	}, [
		draftDateRange,
		draftLabelFilter,
		draftNameSearch,
		draftStatusFilter,
		draftVersionFilter,
	]);

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

	const labelSelectOptions = useMemo(() => {
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
					<Tooltip title={value ? `${key}=${value}` : key}>
						<Tag>
							{formatWorkflowLabelKey(key)}
							{value ? ` · ${value}` : ""}
						</Tag>
					</Tooltip>
				);
			})(),
			value: label,
		}));
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
				messageApi.success(`${operation.title}已提交`);
				await refresh();
			} catch (err) {
				messageApi.error(`${operation.title}失败: ${String(err)}`);
			} finally {
				setOperationLoading(null);
			}
		},
		[messageApi, refresh],
	);

	const runOperation = useCallback(
		(record: WorkflowSummary, key: WorkflowOperationKey) => {
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
		if (selectedWorkflowNames.length === 0) return;
		setBulkDeleting(true);
		try {
			const results = await Promise.allSettled(
				selectedWorkflowNames.map((name) => deleteWorkflow(name)),
			);
			const failedCount = results.filter(
				(result) => result.status === "rejected",
			).length;
			const deletedCount = results.length - failedCount;
			if (deletedCount > 0)
				messageApi.success(`已删除 ${deletedCount} 条执行记录`);
			if (failedCount > 0)
				messageApi.error(`${failedCount} 条执行记录删除失败`);
			setSelectedWorkflowNames([]);
			setBulkDeleteOpen(false);
			await refresh();
		} finally {
			setBulkDeleting(false);
		}
	}, [messageApi, refresh, selectedWorkflowNames]);

	const displayItems = useMemo(() => {
		if (isBatchScope) return items;
		if (!versionFilter) return items;
		const targetVersion = Number(versionFilter);
		if (Number.isNaN(targetVersion)) return items;
		return items.filter(
			(item) => templateVersionsByWorkflowName[item.name] === targetVersion,
		);
	}, [isBatchScope, items, versionFilter, templateVersionsByWorkflowName]);

	const tableTotal = isBatchScope ? serverTotal : displayItems.length;

	const columns = [
		{
			title: "名称",
			dataIndex: "name",
			key: "name",
			width: 260,
			render: (name: string, record: WorkflowSummary) => {
				const runId = runIdsByWorkflowName[record.name];
				const templateId =
					templateIdsByWorkflowName[record.name] ??
					getWorkflowLabel(record.labels, "template-id");
				const templateVersion = templateVersionsByWorkflowName[record.name];
				const scope = scopeByWorkflowName[record.name];
				const displayId = toAssetStyleId(runId ?? name);
				const copyId = runId ?? name;
				return (
					<div style={{ minWidth: 0 }}>
						<Typography.Text strong ellipsis={{ tooltip: name }}>
							{name}
						</Typography.Text>
						<Typography.Text
							type="secondary"
							copyable={{ text: copyId }}
							style={{ display: "block", fontSize: 12 }}
							ellipsis={{ tooltip: runId ? `完整任务 ID: ${runId}` : name }}
						>
							ID: {displayId}
						</Typography.Text>
						<div
							style={{
								marginTop: 4,
								display: "flex",
								flexWrap: "wrap",
								gap: 4,
							}}
						>
							{templateVersion ? (
								<Tag color="blue">模板 v{templateVersion}</Tag>
							) : null}
							{templateId ? (
								<Tag style={{ fontSize: 11 }}>可回到模板</Tag>
							) : null}
							{scope === "prod" ? (
								<Tag color="green" style={{ fontSize: 11 }}>
									<LockOutlined /> 正式版
								</Tag>
							) : scope ? (
								<Tag color="blue" style={{ fontSize: 11 }}>
									Dev 草稿
								</Tag>
							) : null}
						</div>
					</div>
				);
			},
		},
		{
			title: "状态",
			dataIndex: "status",
			key: "status",
			width: 130,
			render: (s: string, record: WorkflowSummary) => (
				<Space size={4} wrap>
					<Tag
						color={STATUS_COLORS[s] || STATUS_ACCENT_COLORS[s] || "default"}
						style={{ padding: "2px 8px" }}
					>
						{s}
					</Tag>
					{isStaleRunningWorkflow(record) ? (
						<Tooltip title="运行时间超过 48 小时，同步任务将自动标记为失败">
							<Tag color="warning">疑似僵尸</Tag>
						</Tooltip>
					) : null}
				</Space>
			),
		},
		{
			title: "节点数",
			dataIndex: "nodeCount",
			key: "nodeCount",
			width: 100,
			render: (nodeCount: number, record: WorkflowSummary) =>
				nodeCountsByWorkflowName[record.name] ?? nodeCount,
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
			title: (
				<Tooltip title="按节点成本汇总的估算总成本">
					<span>总成本</span>
				</Tooltip>
			),
			key: "estimatedCostUsd",
			width: 120,
			align: "right" as const,
			render: renderEstimatedCost,
			sorter: (a: WorkflowSummary, b: WorkflowSummary) =>
				(getWorkflowEstimatedCost(a) ?? -1) -
				(getWorkflowEstimatedCost(b) ?? -1),
		},
		{
			title: "创建时间",
			dataIndex: "createdAt",
			key: "createdAt",
			width: 190,
			render: renderTimestamp,
		},
		{
			title: "完成时间",
			dataIndex: "finishedAt",
			key: "finishedAt",
			width: 190,
			render: renderTimestamp,
		},
		{
			title: "操作",
			key: "actions",
			width: 110,
			render: (_: unknown, record: WorkflowSummary) => {
				const templateId =
					templateIdsByWorkflowName[record.name] ??
					getWorkflowLabel(record.labels, "template-id");
				const templateVersion = templateVersionsByWorkflowName[record.name];
				const scope = scopeByWorkflowName[record.name];
				const menuItems = getWorkflowOperationMenuItems(record);
				const hasOperationLoading = operationLoading?.startsWith(
					`${record.name}:`,
				);

				return (
					<div style={{ display: "flex", gap: 4 }}>
						{templateId ? (
							<Button
								type="link"
								size="small"
								onClick={(event) => {
									event.stopPropagation();
									const params = new URLSearchParams({
										templateId,
										tab: "design",
									});
									if (templateVersion) {
										params.set("templateVersion", String(templateVersion));
									}
									if (scope === "prod") {
										params.set("readonly", "1");
									}
									navigate(`/pipeline?${params.toString()}`);
								}}
							>
								模板
							</Button>
						) : null}
						<Button
							type="link"
							size="small"
							onClick={(event) => {
								event.stopPropagation();
								navigate(`/pipeline/executions/${record.name}`);
							}}
						>
							查看
						</Button>
						<Dropdown
							menu={{
								items: menuItems,
								onClick: ({ key, domEvent }) => {
									domEvent.stopPropagation();
									runOperation(record, key as WorkflowOperationKey);
								},
							}}
							trigger={["click"]}
						>
							<Button
								size="small"
								icon={<MoreOutlined />}
								loading={hasOperationLoading}
								onClick={(event) => {
									event.stopPropagation();
									if (menuItems.length === 0) {
										messageApi.info("当前状态暂无可用操作");
									}
								}}
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
					{title ?? "流水线执行记录"}
				</Typography.Title>
				{!embedded ? (
					<Button
						danger
						disabled={selectedWorkflowNames.length === 0}
						onClick={() => setBulkDeleteOpen(true)}
					>
						批量删除
						{selectedWorkflowNames.length > 0
							? `（${selectedWorkflowNames.length}）`
							: ""}
					</Button>
				) : null}
				<Button icon={<ReloadOutlined />} onClick={refresh} loading={loading}>
					刷新
				</Button>
				<Button
					type="default"
					disabled={
						selectedWorkflowNames.length < 2 || selectedWorkflowNames.length > 3
					}
					title={
						selectedWorkflowNames.length < 2
							? "勾选 2-3 条运行进行对比"
							: selectedWorkflowNames.length > 3
								? "最多选择 3 条运行"
								: undefined
					}
					onClick={() => {
						setCompareItems(
							displayItems.filter((item) =>
								selectedWorkflowNames.includes(item.name),
							),
						);
						setCompareOpen(true);
					}}
				>
					对比选中
					{selectedWorkflowNames.length > 0
						? `（${selectedWorkflowNames.length}）`
						: ""}
				</Button>
			</div>
			<div className="pipeline-execution-filters" style={{ gap: 8 }}>
				<Select
					id="workflow-execution-status-filter"
					allowClear
					placeholder="状态"
					style={{ minWidth: 130, flex: "0 0 130px" }}
					value={draftStatusFilter}
					onChange={(val) => setDraftStatusFilter(val)}
					options={WORKFLOW_PHASES.map((status) => ({
						label: status,
						value: status,
					}))}
				/>
				<Select
					id="workflow-execution-version-filter"
					allowClear
					placeholder="模板版本"
					style={{ minWidth: 120, flex: "0 0 120px" }}
					value={draftVersionFilter}
					onChange={(val) => setDraftVersionFilter(val)}
					options={Array.from(
						new Set(
							Object.values(templateVersionsByWorkflowName).filter(
								(v): v is number => typeof v === "number",
							),
						),
					)
						.sort((a, b) => b - a)
						.map((v) => ({ label: `v${v}`, value: String(v) }))}
				/>
				<Input.Search
					id="workflow-execution-name-search"
					allowClear
					placeholder="按名称搜索"
					style={{ minWidth: 200, flex: "1 1 200px" }}
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
					style={{ minWidth: 300, flex: "1 1 280px" }}
				/>
				<Select
					id="workflow-execution-label-filter"
					mode="multiple"
					allowClear
					maxTagCount="responsive"
					placeholder="标签筛选"
					style={{ minWidth: 220, flex: "1 1 240px" }}
					value={draftLabelFilter}
					onChange={(values) => setDraftLabelFilter(values)}
					options={labelSelectOptions}
				/>
				<Button type="primary" onClick={applyFilters} disabled={!filtersDirty}>
					应用
				</Button>
				<Button onClick={resetFilters}>重置</Button>
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
			) : !error && items.length === 0 ? (
				<Empty description="暂无执行记录，部署流水线后将自动生成" />
			) : (
				<div className="pipeline-execution-table">
					<Table
						dataSource={displayItems}
						columns={columns}
						rowKey="name"
						loading={loading}
						rowSelection={{
							selectedRowKeys: selectedWorkflowNames,
							onChange: (keys) => {
								const names = keys as string[];
								setSelectedWorkflowNames(names);
								onSelectionChange?.(
									displayItems.filter((item) => names.includes(item.name)),
								);
							},
						}}
						scroll={{ x: 1200 }}
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
								navigate(`/pipeline/executions/${record.name}`);
							},
							style: { cursor: "pointer" },
						})}
						pagination={{
							current: page,
							pageSize,
							total: tableTotal,
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
				title={`删除选中的 ${selectedWorkflowNames.length} 条执行记录？`}
				okText="删除"
				cancelText="取消"
				okButtonProps={{ danger: true, loading: bulkDeleting }}
				onOk={confirmBulkDelete}
				onCancel={() => setBulkDeleteOpen(false)}
			>
				<p>删除后不可恢复。正在运行的工作流请先确认不再需要。</p>
			</Modal>
			<RunComparisonModal
				open={compareOpen}
				items={compareItems}
				onClose={() => setCompareOpen(false)}
			/>
		</div>
	);
}
