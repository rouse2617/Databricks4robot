import {
	EyeOutlined,
	ForkOutlined,
	HistoryOutlined,
	LockOutlined,
	MoreOutlined,
	ReloadOutlined,
} from "@ant-design/icons";
import {
	Alert,
	App,
	Button,
	DatePicker,
	Drawer,
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
import type { Breakpoint } from "antd/es/_util/responsiveObserver";
import dayjs, { type Dayjs } from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import {
	type MouseEvent,
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import {
	type BackfillItemAttemptsResult,
	getBatchItemAttempts,
} from "../api/batchJobApi";
import type { PipelineRun, PipelineRunNodeProgress } from "../api/pipelineApi";
import {
	deleteRun,
	listRuns,
	resubmitRun,
	resumeRun,
	retryRun,
	stopRun,
	suspendRun,
	terminateRun,
} from "../api/runApi";
import { deleteWorkflow, type WorkflowSummary } from "../api/workflowApi";
import AssetIdLink from "../components/common/AssetIdLink";
import { DurationPanel } from "../components/common/DurationPanel";
import { WorkflowLabels } from "../components/common/WorkflowLabels";
import { isCanonicalAssetId } from "../lib/assetId";
import { formatPipelineRunNodeProgress } from "../lib/batchNodeProgress";
import {
	STATUS_ACCENT_COLORS,
	STATUS_COLORS,
	WORKFLOW_PHASE_LABELS,
	WORKFLOW_PHASES,
} from "../lib/constants";
import { toAssetStyleId } from "../lib/idDisplay";
import { workflowDetailLocationState } from "../lib/pipelineNavigation";
import { formatWorkflowPhaseLabel } from "../lib/statusLabels";
import { withSelectAllColumn } from "../lib/tableSelection";
import {
	getAvailableWorkflowOperationConfigs,
	getWorkflowOperationConfirmText,
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

const BATCH_DETAIL_WIDE_ONLY: Breakpoint[] = ["xxl"];

interface WorkflowExecutionListProps {
	active?: boolean;
	batchJobId?: string;
	embedded?: boolean;
	nodeFilter?: {
		pipelineNodeId: string;
		nodeStatus?: string;
		label: string;
	};
	onClearNodeFilter?: () => void;
	onSelectionChange?: (items: WorkflowSummary[]) => void;
	title?: string;
}

type ExecutionRecord = WorkflowSummary & {
	runId?: string;
	workflowName?: string;
	pipelineName?: string;
	templateName?: string;
	owner?: string;
	argoNamespace?: string;
	videoDurationSec?: number;
	// CYB-3392: propagate the parent batch id so the row can render a
	// clickable "批次" badge that jumps to BatchJobList detail.
	batchJobId?: string;
};

type WorkflowErrorKind = "network" | "service-unavailable";

type WorkflowErrorState = {
	kind: WorkflowErrorKind;
	message: string;
};

const activeWorkflowStatuses = new Set(["Running", "Pending", "Suspended"]);

interface RunListFilterParams {
	status?: string;
	name?: string;
	label?: string[];
	createdAfter?: string;
	finishedBefore?: string;
}

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
	// 瘦身：相对时间为主（最易扫读），精简绝对时间为辅（去秒，同年省略年份），
	// 完整时间放到 Tooltip，避免在窄列里堆叠两行长数字。
	const sameYear = parsed.year() === dayjs().year();
	const compactAbsolute = parsed.format(
		sameYear ? "MM-DD HH:mm" : "YYYY-MM-DD HH:mm",
	);
	return (
		<Tooltip title={parsed.format("YYYY-MM-DD HH:mm:ss")}>
			<div style={{ lineHeight: 1.35 }}>
				<div>{parsed.fromNow()}</div>
				<Typography.Text type="secondary" style={{ fontSize: 12 }}>
					{compactAbsolute}
				</Typography.Text>
			</div>
		</Tooltip>
	);
};

const renderFinishedTimestamp = (
	value: string | undefined,
	record: WorkflowSummary,
) => {
	if (isActiveWorkflowStatus(record.status)) {
		return "—";
	}
	return renderTimestamp(value);
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

const getTimestampSortValue = (value?: string): number => {
	if (!value) return Number.NEGATIVE_INFINITY;
	const timestamp = new Date(value).getTime();
	return Number.isFinite(timestamp) ? timestamp : Number.NEGATIVE_INFINITY;
};

const getDurationSortValue = (record: WorkflowSummary): number => {
	// 排序口径与展示一致：优先用真实执行起点 startedAt，缺失时回退到 createdAt。
	const startedAt = getTimestampSortValue(record.startedAt ?? record.createdAt);
	if (startedAt === Number.NEGATIVE_INFINITY) {
		return Number.NEGATIVE_INFINITY;
	}
	const finishedAt = getTimestampSortValue(record.finishedAt);
	const endedAt =
		finishedAt === Number.NEGATIVE_INFINITY ? Date.now() : finishedAt;
	if (endedAt < startedAt) {
		return Number.NEGATIVE_INFINITY;
	}
	return endedAt - startedAt;
};

// 与运行详情口径一致：识别因并发限流被 Argo 推迟（postponed）或尚未起跑的排队态。
const isQueuedSummary = (record: WorkflowSummary): boolean => {
	const msg = record.blockingMessage || record.message;
	if (
		msg &&
		/postpone|too many workflows|exceeded.*parallelism|reached.*parallelism|is held by/i.test(
			msg,
		)
	) {
		return true;
	}
	return record.status === "Pending" && !record.startedAt;
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

const RUN_REASON_LABELS: Record<string, string> = {
	unschedulable: "调度失败",
	resource_incompatible: "资源不匹配",
	image_startup: "镜像启动",
	runtime_not_submitted: "等待提交",
	runtime_missing: "Runtime 不可用",
	stale_running: "超时",
	runtime_config_projection_failed: "运行配置投影失败",
	cancelled: "已取消",
	run_failed: "运行失败",
};

const RUN_REASON_COLORS: Record<string, string> = {
	unschedulable: "error",
	resource_incompatible: "error",
	image_startup: "error",
	runtime_not_submitted: "warning",
	runtime_missing: "default",
	stale_running: "error",
	runtime_config_projection_failed: "error",
	cancelled: "default",
	run_failed: "error",
};

const formatRunReasonLabel = (reason?: string): string =>
	reason ? (RUN_REASON_LABELS[reason] ?? reason) : "";

// Generic failure reasons whose tag would only repeat the status tag
// (e.g. status "失败" + reason "运行失败"). Specific reasons such as
// "调度失败 / 资源不匹配 / 超时" still carry extra signal and are kept.
const GENERIC_FAILURE_REASONS = new Set(["run_failed"]);

const isRedundantRunReason = (reason?: string, status?: string): boolean => {
	if (!reason) return false;
	if (!GENERIC_FAILURE_REASONS.has(reason)) return false;
	return status === "Failed" || status === "Error";
};

const renderRunReasonTag = (reason?: string, message?: string) => {
	if (!reason) return null;
	const tag = (
		<Tag
			color={RUN_REASON_COLORS[reason] ?? "default"}
			style={{ marginInlineEnd: 0 }}
		>
			{formatRunReasonLabel(reason)}
		</Tag>
	);
	return message ? <Tooltip title={message}>{tag}</Tooltip> : tag;
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

const executionKeyForRun = (run: PipelineRun): string =>
	run.id || workflowNameForRun(run);

const executionKeyForRecord = (record: ExecutionRecord): string =>
	record.runId || record.name;

const labelsForRun = (run: PipelineRun): Record<string, string> | undefined => {
	const labels: Record<string, string> = {};
	const assetId = run.assetIds?.[0];
	if (assetId) labels.asset_id = assetId;
	return Object.keys(labels).length > 0 ? labels : undefined;
};

const runMatchesFilters = (
	run: PipelineRun,
	params: RunListFilterParams,
): boolean => {
	if (params.status && run.status !== params.status) {
		return false;
	}

	const name = workflowNameForRun(run).toLowerCase();
	const pipelineName = (run.pipelineName ?? "").toLowerCase();
	const runID = (run.id ?? "").toLowerCase();
	const templateName = (run.templateName ?? "").toLowerCase();
	if (
		params.name &&
		!name.includes(params.name) &&
		!pipelineName.includes(params.name) &&
		!runID.includes(params.name) &&
		!templateName.includes(params.name)
	) {
		return false;
	}

	if (params.createdAfter) {
		const createdAt = dayjs(run.createdAt);
		if (createdAt.isValid() && createdAt.isBefore(dayjs(params.createdAfter))) {
			return false;
		}
	}

	if (params.finishedBefore) {
		const finishedAt = dayjs(run.finishedAt);
		if (
			finishedAt.isValid() &&
			finishedAt.isAfter(dayjs(params.finishedBefore))
		) {
			return false;
		}
	}

	if (params.label?.length) {
		const labels = labelsForRun(run) ?? {};
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

// formatVideoDurationSec renders a raw second count as "m分s秒" (or "h时m分s秒"
// for long videos). Returns "-" when the duration is unknown (some videos have
// no recorded duration).
const formatVideoDurationSec = (sec?: number): string => {
	if (sec === undefined || sec === null || !Number.isFinite(sec) || sec < 0) {
		return "-";
	}
	const total = Math.round(sec);
	const h = Math.floor(total / 3600);
	const m = Math.floor((total % 3600) / 60);
	const s = total % 60;
	if (h > 0) return `${h}时${m}分${s}秒`;
	if (m > 0) return `${m}分${s}秒`;
	return `${s}秒`;
};

const workflowSummaryFromRun = (run: PipelineRun): ExecutionRecord => {
	const labels = labelsForRun(run);
	return {
		runId: run.id,
		workflowName: run.workflowName,
		pipelineName: run.pipelineName,
		templateName: run.templateName,
		name: workflowNameForRun(run),
		status: run.status,
		nodeCount: run.nodeCount ?? 0,
		createdAt: run.createdAt,
		startedAt: run.startedAt,
		finishedAt: run.finishedAt,
		labels,
		message: run.message,
		failureReason: run.failureReason,
		blockingReason: run.blockingReason,
		blockingMessage: run.blockingMessage,
		owner: run.owner,
		argoNamespace: run.argoNamespace,
		videoDurationSec: run.videoDurationSec,
		totalEstimatedCost:
			typeof run.totalEstimatedCost === "number"
				? run.totalEstimatedCost
				: undefined,
		batchJobId: run.batchJobId,
	};
};

const runSummariesFromRuns = (
	pipelineRuns: PipelineRun[],
	params: RunListFilterParams,
): ExecutionRecord[] => {
	const ledgerItems = pipelineRuns
		.filter((run) => runMatchesFilters(run, params))
		.map((run) => workflowSummaryFromRun(run));
	return ledgerItems.sort((a, b) => {
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
	nodeFilter,
	onClearNodeFilter,
	onSelectionChange,
	title,
}: WorkflowExecutionListProps) {
	const { message: messageApi } = App.useApp();
	const messageApiRef = useRef(messageApi);
	messageApiRef.current = messageApi;
	const [searchParams, setSearchParams] = useSearchParams();
	const [items, setItems] = useState<ExecutionRecord[]>([]);
	const [runIdsByExecutionKey, setRunIdsByExecutionKey] = useState<
		Record<string, string>
	>({});
	const [templateVersionsByExecutionKey, setTemplateVersionsByExecutionKey] =
		useState<Record<string, number>>({});
	const [templateIdsByExecutionKey, setTemplateIdsByExecutionKey] = useState<
		Record<string, string>
	>({});
	const [nodeCountsByExecutionKey, setNodeCountsByExecutionKey] = useState<
		Record<string, number>
	>({});
	const [scopeByExecutionKey, setScopeByExecutionKey] = useState<
		Record<string, string>
	>({});
	const [nodeProgressByExecutionKey, setNodeProgressByExecutionKey] = useState<
		Record<string, PipelineRunNodeProgress | undefined>
	>({});
	const [attemptsDrawerAssetId, setAttemptsDrawerAssetId] = useState<
		string | null
	>(null);
	const [attemptsLoading, setAttemptsLoading] = useState(false);
	const [attemptsResult, setAttemptsResult] =
		useState<BackfillItemAttemptsResult | null>(null);
	const [loading, setLoading] = useState(false);
	const [initializedOnce, setInitializedOnce] = useState(false);
	const [error, setError] = useState<WorkflowErrorState | null>(null);
	const [statusFilter, setStatusFilter] = useState<string | undefined>(() =>
		batchJobId ? undefined : normalizeStatus(searchParams.get("status")),
	);
	const [draftStatusFilter, setDraftStatusFilter] = useState<
		string | undefined
	>(() =>
		batchJobId ? undefined : normalizeStatus(searchParams.get("status")),
	);
	const [nameSearch, setNameSearch] = useState(
		batchJobId ? "" : (searchParams.get("name")?.trim() ?? ""),
	);
	const [draftNameSearch, setDraftNameSearch] = useState(
		batchJobId ? "" : (searchParams.get("name")?.trim() ?? ""),
	);
	const [labelFilter, setLabelFilter] = useState<string[]>(() => {
		if (batchJobId) return [];
		const labels = searchParams.getAll("label");
		return Array.from(
			new Set(labels.map((label) => label.trim()).filter(Boolean)),
		);
	});
	const [versionFilter, setVersionFilter] = useState<string | undefined>(
		batchJobId
			? undefined
			: searchParams.get("templateVersion")?.trim() || undefined,
	);
	const [draftVersionFilter, setDraftVersionFilter] = useState<
		string | undefined
	>(
		batchJobId
			? undefined
			: searchParams.get("templateVersion")?.trim() || undefined,
	);
	const [draftLabelFilter, setDraftLabelFilter] = useState<string[]>(() => {
		if (batchJobId) return [];
		const labels = searchParams.getAll("label");
		return Array.from(
			new Set(labels.map((label) => label.trim()).filter(Boolean)),
		);
	});
	const [dateRange, setDateRange] = useState<[Dayjs | null, Dayjs | null]>(
		() =>
			batchJobId
				? [null, null]
				: [
						parseDate(searchParams.get("createdAfter")),
						parseDate(searchParams.get("finishedBefore")),
					],
	);
	const [draftDateRange, setDraftDateRange] = useState<
		[Dayjs | null, Dayjs | null]
	>(() =>
		batchJobId
			? [null, null]
			: [
					parseDate(searchParams.get("createdAfter")),
					parseDate(searchParams.get("finishedBefore")),
				],
	);
	const [operationLoading, setOperationLoading] = useState<string | null>(null);
	const [selectedExecutionKeys, setSelectedExecutionKeys] = useState<string[]>(
		[],
	);
	const [bulkDeleteOpen, setBulkDeleteOpen] = useState(false);
	const [bulkDeleting, setBulkDeleting] = useState(false);
	const [compareOpen, setCompareOpen] = useState(false);
	const [compareItems, setCompareItems] = useState<ExecutionRecord[]>([]);
	const [pendingOperation, setPendingOperation] = useState<{
		record: ExecutionRecord;
		operation: WorkflowOperationConfig;
	} | null>(null);
	const [page, setPage] = useState(1);
	const [pageSize, setPageSize] = useState(20);
	const [serverTotal, setServerTotal] = useState(0);
	const navigate = useNavigate();
	const refreshInFlightRef = useRef(false);
	const isBatchScope = Boolean(batchJobId);
	const openWorkflowDetail = useCallback(
		(recordOrName: ExecutionRecord | string) => {
			if (typeof recordOrName !== "string" && recordOrName.runId) {
				navigate(`/runs/${encodeURIComponent(recordOrName.runId)}`, {
					state: workflowDetailLocationState(batchJobId),
				});
				return;
			}
			const name =
				typeof recordOrName === "string" ? recordOrName : recordOrName.name;
			const runId = runIdsByExecutionKey[name];
			navigate(
				runId
					? `/runs/${encodeURIComponent(runId)}`
					: `/pipeline/executions/${encodeURIComponent(name)}`,
				{
					state: workflowDetailLocationState(batchJobId),
				},
			);
		},
		[batchJobId, navigate, runIdsByExecutionKey],
	);

	const runProductOperation = useCallback(
		async (runId: string, key: WorkflowOperationKey) => {
			switch (key) {
				case "delete":
					await deleteRun(runId);
					return;
				case "retry":
					await retryRun(runId);
					return;
				case "resubmit":
					await resubmitRun(runId);
					return;
				case "stop":
					await stopRun(runId);
					return;
				case "suspend":
					await suspendRun(runId);
					return;
				case "resume":
					await resumeRun(runId);
					return;
				case "terminate":
					await terminateRun(runId);
					return;
			}
		},
		[],
	);

	useEffect(() => {
		if (batchJobId) return;
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
	}, [batchJobId, searchParams]);

	const syncAppliedFiltersToUrl = useCallback(() => {
		if (batchJobId) return;
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
		batchJobId,
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
				const runResponse = await listRuns({
					view: "summary",
					batchJobId,
					page,
					pageSize,
					status: statusFilter,
					pipelineNodeId: nodeFilter?.pipelineNodeId,
					nodeStatus: nodeFilter?.nodeStatus,
				});
				const pipelineRuns = runResponse.items ?? [];
				const normalizedNameSearch = nameSearch.trim().toLowerCase();
				const visiblePipelineRuns = normalizedNameSearch
					? pipelineRuns.filter((run) =>
							workflowNameForRun(run)
								.toLowerCase()
								.includes(normalizedNameSearch),
						)
					: pipelineRuns;
				setServerTotal(
					normalizedNameSearch
						? visiblePipelineRuns.length
						: (runResponse.total ?? pipelineRuns.length),
				);
				setRunIdsByExecutionKey(
					Object.fromEntries(
						visiblePipelineRuns
							.filter((run) => run.id)
							.map((run) => [executionKeyForRun(run), run.id] as const),
					),
				);
				setTemplateVersionsByExecutionKey(
					Object.fromEntries(
						visiblePipelineRuns
							.filter((run) => run.templateVersion)
							.map(
								(run) =>
									[
										executionKeyForRun(run),
										run.templateVersion as number,
									] as const,
							),
					),
				);
				setTemplateIdsByExecutionKey(
					Object.fromEntries(
						visiblePipelineRuns
							.filter((run) => run.templateId)
							.map(
								(run) =>
									[executionKeyForRun(run), run.templateId as string] as const,
							),
					),
				);
				setNodeCountsByExecutionKey(
					Object.fromEntries(
						visiblePipelineRuns.map(
							(run) => [executionKeyForRun(run), run.nodeCount] as const,
						),
					),
				);
				setScopeByExecutionKey(
					Object.fromEntries(
						visiblePipelineRuns
							.filter((run) => run.scope)
							.map(
								(run) =>
									[executionKeyForRun(run), run.scope as string] as const,
							),
					),
				);
				setNodeProgressByExecutionKey(
					Object.fromEntries(
						visiblePipelineRuns.map(
							(run) => [executionKeyForRun(run), run.nodeProgress] as const,
						),
					),
				);
				const summaries = visiblePipelineRuns.map((run) =>
					workflowSummaryFromRun(run),
				);
				setItems(summaries);
				setSelectedExecutionKeys((prev) =>
					prev.filter((key) =>
						summaries.some((item) => executionKeyForRecord(item) === key),
					),
				);
				return;
			}

			const params: RunListFilterParams = {
				status: statusFilter,
				label: labelFilter.length ? labelFilter : undefined,
				createdAfter: dateRange[0]?.toISOString(),
				finishedBefore: dateRange[1]?.toISOString(),
			};
			const pipelineRunResponse = await listRuns({
				view: "summary",
				// CYB-3392b: was excludeBatch:true which dropped ALL runs
				// tied to a batch (parent + children), so the CYB-3392 batch
				// badge never had a row to render on. Switch to
				// excludeBatchParents: only the aggregate parent row hides;
				// children stay visible with a clickable badge that jumps
				// to the batch detail page.
				excludeBatchParents: true,
				status: statusFilter,
				q: nameSearch.trim() || undefined,
				page,
				pageSize,
			});
			const pipelineRuns = pipelineRunResponse.items ?? [];
			setServerTotal(pipelineRunResponse.total ?? pipelineRuns.length);
			setRunIdsByExecutionKey(
				Object.fromEntries(
					pipelineRuns
						.filter((run) => run.id)
						.map((run) => [executionKeyForRun(run), run.id] as const),
				),
			);
			setTemplateVersionsByExecutionKey(
				Object.fromEntries(
					pipelineRuns
						.filter((run) => run.templateVersion)
						.map(
							(run) =>
								[
									executionKeyForRun(run),
									run.templateVersion as number,
								] as const,
						),
				),
			);
			setTemplateIdsByExecutionKey(
				Object.fromEntries(
					pipelineRuns
						.filter((run) => run.templateId)
						.map(
							(run) =>
								[executionKeyForRun(run), run.templateId as string] as const,
						),
				),
			);
			setNodeCountsByExecutionKey(
				Object.fromEntries(
					pipelineRuns.map(
						(run) => [executionKeyForRun(run), run.nodeCount] as const,
					),
				),
			);
			setScopeByExecutionKey(
				Object.fromEntries(
					pipelineRuns
						.filter((run) => run.scope)
						.map(
							(run) => [executionKeyForRun(run), run.scope as string] as const,
						),
				),
			);
			const enrichedItems = runSummariesFromRuns(pipelineRuns, params);
			setItems(enrichedItems);
			setSelectedExecutionKeys((prev) =>
				prev.filter((key) =>
					enrichedItems.some((item) => executionKeyForRecord(item) === key),
				),
			);
		} catch (err) {
			console.error(err);
			setError(describeWorkflowError(err));
			// Keep existing items on transient failures instead of clearing
			// the list, which makes it look like no runs exist.
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
		nodeFilter,
		statusFilter,
		page,
		pageSize,
	]);

	useEffect(() => {
		if (active) {
			refresh();
		}
	}, [active, refresh]);

	useEffect(() => {
		setPage(1);
		if (!batchJobId) {
			syncAppliedFiltersToUrl();
		}
	}, [batchJobId, syncAppliedFiltersToUrl]);

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
		setDraftVersionFilter(undefined);
		setDraftNameSearch("");
		setDraftLabelFilter([]);
		setDraftDateRange([null, null]);
		setStatusFilter(undefined);
		setVersionFilter(undefined);
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
			record: ExecutionRecord,
			operation: WorkflowOperationConfig,
		): Promise<void> => {
			const executionKey = executionKeyForRecord(record);
			const loadingKey = `${executionKey}:${operation.key}`;
			setOperationLoading(loadingKey);
			try {
				const runId = record.runId ?? runIdsByExecutionKey[executionKey];
				if (runId) {
					await runProductOperation(runId, operation.key);
				} else {
					await operation.run();
				}
				messageApiRef.current.success(`${operation.title}已提交`);
				await refresh();
			} catch (err) {
				messageApiRef.current.error(`${operation.title}失败: ${String(err)}`);
			} finally {
				setOperationLoading(null);
			}
		},
		[refresh, runIdsByExecutionKey, runProductOperation],
	);

	const runOperation = useCallback(
		(record: ExecutionRecord, key: WorkflowOperationKey) => {
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
		if (selectedExecutionKeys.length === 0) return;
		setBulkDeleting(true);
		try {
			const results = await Promise.allSettled(
				selectedExecutionKeys.map((executionKey) => {
					const runId = runIdsByExecutionKey[executionKey];
					return runId ? deleteRun(runId) : deleteWorkflow(executionKey);
				}),
			);
			const failedCount = results.filter(
				(result) => result.status === "rejected",
			).length;
			const deletedCount = results.length - failedCount;
			if (deletedCount > 0)
				messageApiRef.current.success(`已删除 ${deletedCount} 条执行记录`);
			if (failedCount > 0)
				messageApiRef.current.error(`${failedCount} 条执行记录删除失败`);
			setSelectedExecutionKeys([]);
			setBulkDeleteOpen(false);
			await refresh();
		} finally {
			setBulkDeleting(false);
		}
	}, [refresh, runIdsByExecutionKey, selectedExecutionKeys]);

	const displayItems = useMemo(() => {
		if (isBatchScope) return items;
		if (!versionFilter) return items;
		const targetVersion = Number(versionFilter);
		if (Number.isNaN(targetVersion)) return items;
		return items.filter(
			(item) =>
				templateVersionsByExecutionKey[executionKeyForRecord(item)] ===
				targetVersion,
		);
	}, [isBatchScope, items, versionFilter, templateVersionsByExecutionKey]);

	const tableTotal =
		isBatchScope || labelFilter.length === 0
			? serverTotal
			: displayItems.length;

	const openAttemptsDrawer = useCallback(
		async (assetId: string) => {
			if (!batchJobId) return;
			setAttemptsDrawerAssetId(assetId);
			setAttemptsLoading(true);
			setAttemptsResult(null);
			try {
				const result = await getBatchItemAttempts(batchJobId, { assetId });
				setAttemptsResult(result);
			} catch (err) {
				messageApiRef.current.error(`加载执行历史失败: ${String(err)}`);
			} finally {
				setAttemptsLoading(false);
			}
		},
		[batchJobId],
	);

	const columns = useMemo(() => {
		const baseColumns = [
			{
				title: "名称",
				dataIndex: "name",
				key: "name",
				width: isBatchScope ? 280 : 260,
				sorter: (a: WorkflowSummary, b: WorkflowSummary) =>
					(a.name ?? "").localeCompare(b.name ?? ""),
				render: (name: string, record: ExecutionRecord) => {
					const executionKey = executionKeyForRecord(record);
					const runId = record.runId ?? runIdsByExecutionKey[executionKey];
					const templateVersion = templateVersionsByExecutionKey[executionKey];
					const scope = scopeByExecutionKey[executionKey];
					const displayId = toAssetStyleId(runId ?? name);
					const copyId = runId ?? name;
					const templateName = record.templateName?.trim();
					return (
						<div style={{ minWidth: 0 }}>
							<Typography.Text
								strong
								copyable={{ text: name }}
								ellipsis={{ tooltip: name }}
							>
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
							{templateName ? (
								<Typography.Text
									type="secondary"
									style={{ display: "block", fontSize: 12 }}
									ellipsis={{ tooltip: `模板：${templateName}` }}
								>
									模板：{templateName}
								</Typography.Text>
							) : null}
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
								{scope === "prod" ? (
									<Tag color="green" style={{ fontSize: 11 }}>
										<LockOutlined /> 正式版
									</Tag>
								) : scope ? (
									<Tag color="blue" style={{ fontSize: 11 }}>
										Dev 草稿
									</Tag>
								) : null}
								{/* CYB-3392: 批次徽标 — 单次执行属于批次时显示可跳转 tag,
								    与 BatchJobList 形成双向导航;仅在非-batch scope 下显示,
								    因为 batch scope 页面本身就在批次上下文里. */}
								{!isBatchScope && record.batchJobId ? (
									<Tag
										color="purple"
										style={{ fontSize: 11, cursor: "pointer" }}
										title={`所属批次 ${record.batchJobId} — 点击查看批次详情`}
										onClick={(e) => {
											e.stopPropagation();
											if (record.batchJobId) {
												navigate(
													`/pipeline/batch/${encodeURIComponent(record.batchJobId)}`,
												);
											}
										}}
									>
										批次 {record.batchJobId.slice(0, 8)}
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
				width: isBatchScope ? 110 : 130,
				sorter: (a: WorkflowSummary, b: WorkflowSummary) =>
					(a.status ?? "").localeCompare(b.status ?? ""),
				render: (s: string, record: ExecutionRecord) => {
					const reason = record.blockingReason || record.failureReason;
					const reasonMessage = record.blockingMessage || record.message;
					const redundant = isRedundantRunReason(reason, s);
					const statusTag = (
						<Tag
							color={STATUS_COLORS[s] || STATUS_ACCENT_COLORS[s] || "default"}
							style={{ padding: "2px 8px" }}
						>
							{formatWorkflowPhaseLabel(s)}
						</Tag>
					);
					return (
						<Space size={4} wrap>
							{redundant && reasonMessage ? (
								<Tooltip title={reasonMessage}>{statusTag}</Tooltip>
							) : (
								statusTag
							)}
							{isQueuedSummary(record) ? (
								<Tooltip title="并发已达上限或等待调度，工作流正在排队，待资源释放后自动开始执行">
									<Tag color="gold">排队中</Tag>
								</Tooltip>
							) : null}
							{isStaleRunningWorkflow(record) ? (
								<Tooltip title="运行时间超过 48 小时，同步任务将自动标记为失败">
									<Tag color="warning">疑似僵尸</Tag>
								</Tooltip>
							) : null}
							{redundant ? null : renderRunReasonTag(reason, reasonMessage)}
						</Space>
					);
				},
			},
			...(isBatchScope
				? [
						{
							title: "资产",
							key: "assetId",
							width: 200,
							render: (_: unknown, record: ExecutionRecord) => {
								const assetId = getWorkflowLabel(record.labels, "asset_id");
								if (!assetId) return "—";
								if (isCanonicalAssetId(assetId)) {
									return (
										<span
											style={{
												display: "inline-flex",
												alignItems: "center",
												gap: 4,
											}}
										>
											<AssetIdLink id={assetId} />
											<Typography.Text copyable={{ text: assetId }} />
										</span>
									);
								}
								return (
									<Tooltip title={assetId}>
										<span
											style={{
												display: "inline-flex",
												alignItems: "center",
												gap: 4,
												maxWidth: 190,
											}}
										>
											<Typography.Text
												code
												style={{
													flex: "0 1 auto",
													fontSize: 12,
													overflow: "hidden",
													textOverflow: "ellipsis",
													whiteSpace: "nowrap",
												}}
											>
												{assetId}
											</Typography.Text>
											<Typography.Text copyable={{ text: assetId }} />
										</span>
									</Tooltip>
								);
							},
						},
						{
							title: "节点进度",
							key: "nodeProgress",
							width: 150,
							render: (_: unknown, record: ExecutionRecord) => {
								const executionKey = executionKeyForRecord(record);
								const { text, tooltip } = formatPipelineRunNodeProgress(
									nodeProgressByExecutionKey[executionKey],
								);
								const content = isBatchScope ? (
									<Typography.Text
										style={{
											display: "inline-block",
											maxWidth: 135,
											overflow: "hidden",
											textOverflow: "ellipsis",
											verticalAlign: "bottom",
											whiteSpace: "nowrap",
										}}
									>
										{text}
									</Typography.Text>
								) : (
									text
								);
								if (tooltip) {
									return <Tooltip title={tooltip}>{content}</Tooltip>;
								}
								return content;
							},
						},
					]
				: []),
			{
				title: "所属用户",
				dataIndex: "owner",
				key: "owner",
				width: 160,
				sorter: (a: WorkflowSummary, b: WorkflowSummary) =>
					((a as any).owner ?? "").localeCompare((b as any).owner ?? ""),
				render: (owner: string) =>
					owner ? (
						<Typography.Text copyable={{ text: owner }}>
							{owner}
						</Typography.Text>
					) : (
						<Typography.Text type="secondary">—</Typography.Text>
					),
			},
			{
				title: "命名空间",
				dataIndex: "argoNamespace",
				key: "argoNamespace",
				width: 150,
				sorter: (a: WorkflowSummary, b: WorkflowSummary) =>
					((a as any).argoNamespace ?? "").localeCompare(
						(b as any).argoNamespace ?? "",
					),
				render: (ns: string) => {
					if (!ns) return <Typography.Text type="secondary">—</Typography.Text>;
					const color = ns.includes("prod")
						? "red"
						: ns.includes("dev")
							? "blue"
							: "purple";
					const shortName = ns.split("/").pop() || ns;
					return (
						<Tooltip title={ns}>
							<Tag color={color} style={{ fontSize: 11 }}>
								{shortName}
							</Tag>
						</Tooltip>
					);
				},
			},
			{
				title: "节点数",
				dataIndex: "nodeCount",
				key: "nodeCount",
				width: isBatchScope ? 70 : 90,
				sorter: (a: WorkflowSummary, b: WorkflowSummary) =>
					(a.nodeCount ?? 0) - (b.nodeCount ?? 0),
				render: (nodeCount: number, record: ExecutionRecord) =>
					nodeCountsByExecutionKey[executionKeyForRecord(record)] ?? nodeCount,
				responsive: isBatchScope ? BATCH_DETAIL_WIDE_ONLY : undefined,
			},
			{
				title: "标签",
				dataIndex: "labels",
				key: "labels",
				width: isBatchScope ? 200 : 240,
				render: (labels?: Record<string, string>) => (
					<WorkflowLabels labels={labels} />
				),
				responsive: isBatchScope ? BATCH_DETAIL_WIDE_ONLY : undefined,
			},
			{
				title: "耗时",
				key: "duration",
				width: isBatchScope ? 110 : 140,
				sorter: (a: WorkflowSummary, b: WorkflowSummary) =>
					getDurationSortValue(a) - getDurationSortValue(b),
				render: (_: unknown, record: WorkflowSummary) => (
					<DurationPanel
						phase={record.status}
						createdAt={record.createdAt}
						startedAt={record.startedAt}
						finishedAt={record.finishedAt}
					/>
				),
			},
			{
				title: "视频时长",
				key: "videoDuration",
				width: 100,
				sorter: (a: WorkflowSummary, b: WorkflowSummary) =>
					((a as ExecutionRecord).videoDurationSec ?? -1) -
					((b as ExecutionRecord).videoDurationSec ?? -1),
				render: (_: unknown, record: WorkflowSummary) =>
					formatVideoDurationSec((record as ExecutionRecord).videoDurationSec),
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
				responsive: isBatchScope ? BATCH_DETAIL_WIDE_ONLY : undefined,
			},
			{
				title: "创建时间",
				dataIndex: "createdAt",
				key: "createdAt",
				width: 150,
				defaultSortOrder: "descend" as const,
				sorter: (a: WorkflowSummary, b: WorkflowSummary) =>
					getTimestampSortValue(a.createdAt) -
					getTimestampSortValue(b.createdAt),
				render: renderTimestamp,
				responsive: isBatchScope ? BATCH_DETAIL_WIDE_ONLY : undefined,
			},
			{
				title: "完成时间",
				dataIndex: "finishedAt",
				key: "finishedAt",
				width: 150,
				sorter: (a: WorkflowSummary, b: WorkflowSummary) =>
					getTimestampSortValue(a.finishedAt) -
					getTimestampSortValue(b.finishedAt),
				render: (value: string | undefined, record: ExecutionRecord) =>
					renderFinishedTimestamp(value, record),
				responsive: isBatchScope ? BATCH_DETAIL_WIDE_ONLY : undefined,
			},
			{
				title: "操作",
				key: "actions",
				width: isBatchScope ? 100 : 110,
				render: (_: unknown, record: ExecutionRecord) => {
					const executionKey = executionKeyForRecord(record);
					const templateId =
						templateIdsByExecutionKey[executionKey] ??
						getWorkflowLabel(record.labels, "template-id");
					const templateVersion = templateVersionsByExecutionKey[executionKey];
					const scope = scopeByExecutionKey[executionKey];
					const assetId = getWorkflowLabel(record.labels, "asset_id");
					const menuItems = getWorkflowOperationMenuItems(record);
					const hasOperationLoading = operationLoading?.startsWith(
						`${executionKey}:`,
					);
					const openTemplate = (event: MouseEvent<HTMLElement>) => {
						event.stopPropagation();
						if (!templateId) return;
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
					};
					const viewRun = (event: MouseEvent<HTMLElement>) => {
						event.stopPropagation();
						openWorkflowDetail(record);
					};

					if (isBatchScope) {
						return (
							<Space size={2} wrap={false}>
								{assetId ? (
									<Tooltip title="执行历史">
										<Button
											aria-label="执行历史"
											type="text"
											size="small"
											icon={<HistoryOutlined />}
											onClick={(event) => {
												event.stopPropagation();
												void openAttemptsDrawer(assetId);
											}}
										/>
									</Tooltip>
								) : null}
								{templateId ? (
									<Tooltip title="打开模板">
										<Button
											aria-label="打开模板"
											type="text"
											size="small"
											icon={<ForkOutlined />}
											onClick={openTemplate}
										/>
									</Tooltip>
								) : null}
								<Tooltip title="查看执行">
									<Button
										aria-label="查看执行"
										type="text"
										size="small"
										icon={<EyeOutlined />}
										onClick={viewRun}
									/>
								</Tooltip>
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
										aria-label="更多操作"
										type="text"
										size="small"
										icon={<MoreOutlined />}
										loading={hasOperationLoading}
										onClick={(event) => {
											event.stopPropagation();
											if (menuItems.length === 0) {
												messageApi.info("当前状态暂无可用操作");
											}
										}}
									/>
								</Dropdown>
							</Space>
						);
					}

					return (
						<div style={{ display: "flex", gap: 4, flexWrap: "wrap" }}>
							{isBatchScope && assetId ? (
								<Button
									type="link"
									size="small"
									onClick={(event) => {
										event.stopPropagation();
										void openAttemptsDrawer(assetId);
									}}
								>
									历史
								</Button>
							) : null}
							{templateId ? (
								<Button type="link" size="small" onClick={openTemplate}>
									模板
								</Button>
							) : null}
							<Button type="link" size="small" onClick={viewRun}>
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
		return baseColumns;
	}, [
		isBatchScope,
		nodeCountsByExecutionKey,
		nodeProgressByExecutionKey,
		openAttemptsDrawer,
		openWorkflowDetail,
		operationLoading,
		runIdsByExecutionKey,
		runOperation,
		scopeByExecutionKey,
		templateIdsByExecutionKey,
		templateVersionsByExecutionKey,
		messageApi,
		navigate,
	]);

	const showSkeleton = !initializedOnce;
	const tableScrollX = isBatchScope ? "max-content" : 1500;

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
						disabled={selectedExecutionKeys.length === 0}
						onClick={() => setBulkDeleteOpen(true)}
					>
						批量删除
						{selectedExecutionKeys.length > 0
							? `（${selectedExecutionKeys.length}）`
							: ""}
					</Button>
				) : null}
				<Button icon={<ReloadOutlined />} onClick={refresh} loading={loading}>
					刷新
				</Button>
				{isBatchScope && nodeFilter ? (
					<Tag
						closable={Boolean(onClearNodeFilter)}
						onClose={onClearNodeFilter}
					>
						节点筛选：{nodeFilter.label}
					</Tag>
				) : null}
				<Button
					type="default"
					disabled={
						selectedExecutionKeys.length < 2 || selectedExecutionKeys.length > 3
					}
					title={
						selectedExecutionKeys.length < 2
							? "勾选 2-3 条运行进行对比"
							: selectedExecutionKeys.length > 3
								? "最多选择 3 条运行"
								: undefined
					}
					onClick={() => {
						setCompareItems(
							displayItems.filter((item) =>
								selectedExecutionKeys.includes(executionKeyForRecord(item)),
							),
						);
						setCompareOpen(true);
					}}
				>
					对比选中
					{selectedExecutionKeys.length > 0
						? `（${selectedExecutionKeys.length}）`
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
					// CYB-3391: use the localized zh-CN label from
					// WORKFLOW_PHASE_LABELS so the dropdown matches the table
					// status chips (previously label=status showed the raw English
					// enum while cells rendered "运行中/成功/失败/…", forcing users
					// to translate mentally when filtering).
					options={WORKFLOW_PHASES.map((status) => ({
						label: WORKFLOW_PHASE_LABELS[status],
						value: status,
					}))}
				/>
				{!isBatchScope ? (
					<Select
						id="workflow-execution-version-filter"
						allowClear
						placeholder="模板版本"
						style={{ minWidth: 120, flex: "0 0 120px" }}
						value={draftVersionFilter}
						onChange={(val) => setDraftVersionFilter(val)}
						options={Array.from(
							new Set(
								Object.values(templateVersionsByExecutionKey).filter(
									(v): v is number => typeof v === "number",
								),
							),
						)
							.sort((a, b) => b - a)
							.map((v) => ({ label: `v${v}`, value: String(v) }))}
					/>
				) : null}
				<Input.Search
					id="workflow-execution-name-search"
					allowClear
					placeholder="搜索名称 / ID / 模板 / 用户 / 命名空间 / asset_id"
					style={
						isBatchScope
							? { width: 420, minWidth: 280, maxWidth: 520, flex: "1 1 360px" }
							: { minWidth: 200, flex: "1 1 200px" }
					}
					value={draftNameSearch}
					onChange={(event) => setDraftNameSearch(event.target.value)}
					onSearch={applyFilters}
				/>
				{!isBatchScope ? (
					<>
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
					</>
				) : null}
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
						rowKey={executionKeyForRecord}
						loading={loading}
						size={embedded ? "small" : "middle"}
						rowSelection={withSelectAllColumn<ExecutionRecord>({
							selectedRowKeys: selectedExecutionKeys,
							onChange: (keys) => {
								const executionKeys = keys as string[];
								setSelectedExecutionKeys(executionKeys);
								onSelectionChange?.(
									displayItems.filter((item) =>
										executionKeys.includes(executionKeyForRecord(item)),
									),
								);
							},
						})}
						scroll={{ x: tableScrollX }}
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
								openWorkflowDetail(record);
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
				{pendingOperation
					? getWorkflowOperationConfirmText(pendingOperation.operation.key)
					: null}
			</Modal>
			<Modal
				open={bulkDeleteOpen}
				title={`删除选中的 ${selectedExecutionKeys.length} 条执行记录？`}
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
			<Drawer
				title={
					attemptsDrawerAssetId
						? `执行历史 · ${attemptsDrawerAssetId}`
						: "执行历史"
				}
				open={Boolean(attemptsDrawerAssetId)}
				width={720}
				onClose={() => {
					setAttemptsDrawerAssetId(null);
					setAttemptsResult(null);
				}}
			>
				<Table
					size="small"
					rowKey="runId"
					loading={attemptsLoading}
					dataSource={attemptsResult?.attempts ?? []}
					pagination={false}
					columns={[
						{
							title: "次序",
							dataIndex: "attemptNo",
							width: 60,
						},
						{
							title: "状态",
							dataIndex: "status",
							render: (status: string, record) => (
								<Space size={4}>
									<Tag>{formatWorkflowPhaseLabel(status)}</Tag>
									{record.isCurrent ? <Tag color="blue">当前</Tag> : null}
								</Space>
							),
						},
						{
							title: "节点进度",
							render: (_, record) => {
								const { text, tooltip } = formatPipelineRunNodeProgress(
									record.nodeProgress,
								);
								return tooltip ? (
									<Tooltip title={tooltip}>{text}</Tooltip>
								) : (
									text
								);
							},
						},
						{
							title: "版本",
							dataIndex: "templateVersion",
							render: (version?: number) => (version ? `v${version}` : "—"),
						},
						{
							title: "Run",
							render: (_, record) =>
								record.runId ? (
									<Button
										type="link"
										size="small"
										onClick={() =>
											navigate(`/runs/${encodeURIComponent(record.runId)}`)
										}
									>
										查看
									</Button>
								) : (
									"—"
								),
						},
						{
							title: "创建时间",
							dataIndex: "createdAt",
							render: renderTimestamp,
						},
					]}
				/>
			</Drawer>
		</div>
	);
}
