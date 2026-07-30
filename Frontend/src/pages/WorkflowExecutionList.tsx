import {
	CopyOutlined,
	EyeOutlined,
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
import { assetsApi } from "../api/assets";
import {
	type BackfillItemAttemptsResult,
	getBatchItemAttempts,
} from "../api/batchJobApi";
import type {
	ExecutionTarget,
	PipelineRun,
	PipelineRunNodeProgress,
} from "../api/pipelineApi";
import { listExecutionTargets } from "../api/pipelineApi";
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
import { HoverActionBar, type HoverAction } from "../components/common/HoverActionBar";
import { WorkflowLabels } from "../components/common/WorkflowLabels";
import { isCanonicalAssetId, isUUID } from "../lib/assetId";
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
import { STATUS_TAG_PALETTE } from "../lib/designTokens";
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
	// CYB-3486: 资源池不再以命名空间示人 —— 列表用 executionTargetId 反查池名。
	executionTargetId?: string;
	videoDurationSec?: number;
	// CYB-4011: source asset ids (often Grace UUIDs) — used to resolve the
	// video duration via grace_video_id when video_durations has no row.
	assetIds?: string[];
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
// 与后端 state_machine.go 的 run 状态词表对齐（Succeeded/Failed/Error/
// Cancelled/Expired）。仅这些明确终态才抑制 runtime_missing；空值、未知或
// 未来新增的状态保持可诊断，不会被“非活动即终态”的反向判断误伤。
const terminalWorkflowStatuses = new Set([
	"Succeeded",
	"Failed",
	"Error",
	"Cancelled",
	"Expired",
]);

interface RunListFilterParams {
	status?: string;
	name?: string;
	label?: string[];
	createdAfter?: string;
	finishedBefore?: string;
}

// CYB-3691: 停滞判定使用 updatedAt (freshness) + 3h 阈值 —— 大任务
// 跑两三小时是常态（容器计算无阶段更新）,30min 过于敏感 (see #417)。
// 仍是纯展示提示,不改 run status —— 原则:任务可以等,不该自动判死。
const STALE_ACTIVE_RUN_MS = 3 * 60 * 60 * 1000;

const isActiveWorkflowStatus = (status?: string): boolean =>
	activeWorkflowStatuses.has(status ?? "");

const isTerminalWorkflowStatus = (status?: string): boolean =>
	terminalWorkflowStatuses.has(status ?? "");

const isStaleRunningWorkflow = (record: WorkflowSummary): boolean => {
	if (!isActiveWorkflowStatus(record.status)) return false;
	const ref = record.updatedAt ?? record.createdAt;
	if (!ref) return false;
	return Date.now() - new Date(ref).getTime() > STALE_ACTIVE_RUN_MS;
};

const parseDate = (value: string | null): Dayjs | null => {
	if (!value) return null;
	const parsed = dayjs(value);
	return parsed.isValid() ? parsed : null;
};

const renderTimestamp = (value?: string) => {
	if (!value) {
		return (
			<span style={{ color: "#bfbfbf", fontStyle: "italic" }}>—</span>
		);
	}
	const parsed = dayjs(value);
	if (!parsed.isValid()) return new Date(value).toLocaleString();
	// CYB-4470 B.4：降行高——相对时间为主（最易扫读），完整时间折叠进 Tooltip，
	// 避免双行堆叠压低一屏可见数据量。
	return (
		<Tooltip title={parsed.format("YYYY-MM-DD HH:mm:ss")}>
			<span>{parsed.fromNow()}</span>
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
	// CYB-4470 B.6：微小成本格式化——0 → "$0.00"，< $0.001 → "<$0.001" 消除
	// "$0.0000 是免费还是算错"的歧义；其它按 < $0.01 用 4 位精度，否则 2 位。
	const display =
		cost === 0
			? "$0.00"
			: cost < 0.001
				? "<$0.001"
				: cost < 0.01
					? `$${cost.toFixed(4)}`
					: `$${cost.toFixed(2)}`;
	return (
		<Tooltip title="估算总成本，非 GCP Billing 最终对账金额">
			<Typography.Text strong>{display}</Typography.Text>
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
	workflow_rbac_forbidden: "工作流权限不足",
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
	workflow_rbac_forbidden: "error",
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

// runtime_missing 是任务在运行中才会有的临时状态（同步断流、ledger 还没更新
// 等），任务一旦进入终态（成功/失败/过期/取消）就不应再展示，否则会误导排障。
const shouldSuppressRuntimeMissing = (
	reason?: string,
	status?: string,
): boolean => reason === "runtime_missing" && isTerminalWorkflowStatus(status);

const renderRunReasonTag = (
	reason?: string,
	message?: string,
	status?: string,
) => {
	if (!reason) return null;
	// 对于已完成的任务（成功/失败/过期/取消等），不显示 runtime_missing
	if (shouldSuppressRuntimeMissing(reason, status)) {
		return null;
	}
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

// CYB-4477 P2-2：行 hover 快捷操作"复制 ID" 复用 navigator.clipboard；
// 老浏览器或非安全上下文可能不可用 —— 失败时静默回退到 textarea + execCommand
// 让操作在本地也能跑通（test 环境没 clipboard API 时同样能 resolve(false)）。
const copyTextToClipboard = async (text: string): Promise<boolean> => {
	if (typeof navigator !== "undefined" && navigator.clipboard?.writeText) {
		try {
			await navigator.clipboard.writeText(text);
			return true;
		} catch {
			// fall through to legacy fallback
		}
	}
	if (typeof document === "undefined") return false;
	try {
		const textarea = document.createElement("textarea");
		textarea.value = text;
		textarea.style.position = "fixed";
		textarea.style.opacity = "0";
		document.body.appendChild(textarea);
		textarea.select();
		const ok = document.execCommand("copy");
		document.body.removeChild(textarea);
		return ok;
	} catch {
		return false;
	}
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
		executionTargetId: run.executionTargetId,
		videoDurationSec: run.videoDurationSec,
		assetIds: run.assetIds,
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
	// CYB-4011: grace UUID → resolved video duration (seconds), used as a
	// fallback when video_durations has no row for the Grace video id but the
	// mirrored DataBrew asset carries duration_sec (resolved via grace_video_id).
	const [resolvedDurationByUUID, setResolvedDurationByUUID] = useState<
		Record<string, number>
	>({});
	// CYB-3486: executionTargetId → 资源池,用于把"命名空间"列换成"资源池"列。
	// 加载失败时保持空表,render 会优雅回退到命名空间显示(不回归)。
	const [targetById, setTargetById] = useState<Map<string, ExecutionTarget>>(
		() => new Map(),
	);
	const [runIdsByExecutionKey, setRunIdsByExecutionKey] = useState<
		Record<string, string>
	>({});
	const [templateVersionsByExecutionKey, setTemplateVersionsByExecutionKey] =
		useState<Record<string, number>>({});
	// CYB-4470 B.7：操作列清理后不再需要 template-id map（"模板"按钮已移除），
	// templateId 现在直接来自 record.labels["template-id"]，按需 lookup。
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
	// CYB-4477 P2-2：行 hover 快捷操作栏 —— 记录当前 hover 行的 executionKey，
	// 配合 <Table onRow> 的 mouse enter/leave 控制 HoverActionBar 的可见性。
	const [hoveredExecutionKey, setHoveredExecutionKey] = useState<string | null>(
		null,
	);
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

	// CYB-3486: 拉取资源池列表,把 executionTargetId 反解成池名。挂载时取一次;
	// 失败时静默 —— map 保持空,列会优雅回退到命名空间显示,不阻塞主列表。
	useEffect(() => {
		let cancelled = false;
		listExecutionTargets()
			.then((targets) => {
				if (cancelled) return;
				setTargetById(new Map(targets.map((t) => [t.id, t])));
			})
			.catch(() => {
				/* 资源池信息非关键路径,失败退回命名空间显示 */
			});
		return () => {
			cancelled = true;
		};
	}, []);

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
				// CYB-3709: the single-execution tab shows only genuine single
				// runs — exclude ALL batch rows (parents AND children). cyb-3392b
				// had used excludeBatchParents to keep children here with a badge,
				// but at 60k+ children that buries the ~4.4k single runs. Batch
				// children remain listed in the batch detail view (batchJobId
				// branch above), so nothing is lost.
				excludeBatch: true,
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

	// CYB-4011: fill the video-duration column for rows whose source asset is a
	// Grace UUID that video_durations has no row for. Resolve the UUID to its
	// DataBrew asset via grace_video_id and use that asset's duration_sec.
	// Read-only, per-UUID, cached; skips UUIDs already resolved (incl. misses).
	useEffect(() => {
		const pending = new Set<string>();
		for (const r of items) {
			if (r.videoDurationSec != null) continue;
			for (const a of r.assetIds ?? []) {
				if (a && isUUID(a) && !(a in resolvedDurationByUUID)) {
					pending.add(a);
				}
			}
		}
		if (pending.size === 0) return;
		let cancelled = false;
		(async () => {
			const entries = await Promise.all(
				[...pending].map(async (uuid) => {
					const asset = await assetsApi.resolveByGraceVideoID(uuid);
					// -1 marks "resolved but no duration" so we don't re-query.
					const sec =
						asset?.duration_sec ??
						(asset?.duration_ms != null ? asset.duration_ms / 1000 : -1);
					return [uuid, sec] as const;
				}),
			);
			if (cancelled) return;
			setResolvedDurationByUUID((prev) => {
				const next = { ...prev };
				for (const [uuid, sec] of entries) next[uuid] = sec;
				return next;
			});
		})();
		return () => {
			cancelled = true;
		};
	}, [items, resolvedDurationByUUID]);

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
				operation.key === "retry" ||
				operation.key === "stop" ||
				operation.key === "suspend"
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
		// CYB-3486: executionTargetId → 资源池名(带回退),供"资源池"列排序/渲染共用。
		const poolLabelFor = (record: ExecutionRecord): string => {
			const id = record.executionTargetId ?? "";
			const target = id ? targetById.get(id) : undefined;
			return target?.name ?? record.argoNamespace ?? "";
		};
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
					// CYB-4334: when a run has neither workflowName nor
					// pipelineName, `name` falls back to the raw run UUID —
					// unreadable and duplicated by the ID line below. Show a
					// human-readable primary label instead (template name, or a
					// placeholder), and suppress the separate 模板 line so the
					// template name is not repeated.
					const nameIsRawId =
						!!name && (name === runId || name === executionKey);
					const primaryLabel = nameIsRawId
						? templateName || "未命名运行"
						: name;
					const showTemplateLine = !!templateName && !nameIsRawId;
					// CYB-4470 B.7：批次名称承担查看入口 —— 让 name 视觉/语义上
					// 都是可点击锚点（蓝色 + cursor pointer + onClick）；同时保留
					// Typography.Text 的 copyable（antd 内部对复制图标 stopPropagation）。
					const handleNameClick = (event: MouseEvent<HTMLElement>) => {
						event.stopPropagation();
						openWorkflowDetail(record);
					};
					return (
						<div style={{ minWidth: 0 }}>
							{/* CYB-4470 B.7：批次名称承担查看入口 —— 用 <a> 包住整个
							    Typography.Text（name + antd 复制图标）。antd 的复制
							    图标 onClick 内部已 stopPropagation，复制不会触发跳转；
							    role="button" + tabIndex=0 让屏幕阅读器把它当按钮。 */}
							<a
								role="button"
								tabIndex={0}
								onClick={handleNameClick}
								onKeyDown={(event) => {
									if (event.key === "Enter" || event.key === " ") {
										event.preventDefault();
										openWorkflowDetail(record);
									}
								}}
								// CYB-4470 B.1：copy-cell 让复制图标 hover 才出现。
								className="copy-cell"
								style={{
									cursor: "pointer",
									color: "#1677ff",
									display: "inline-flex",
									alignItems: "center",
									minWidth: 0,
									maxWidth: "100%",
									overflow: "hidden",
									textDecoration: "none",
								}}
								title="点击查看执行详情"
							>
								<Typography.Text
									strong
									copyable={{ text: nameIsRawId ? copyId : name }}
									style={{
										color: "#1677ff",
										overflow: "hidden",
										textOverflow: "ellipsis",
										whiteSpace: "nowrap",
									}}
									ellipsis={{
										tooltip: nameIsRawId
											? templateName
												? `模板：${templateName}`
												: "未命名运行（无 workflow / pipeline 名）"
											: name,
									}}
								>
									{primaryLabel}
								</Typography.Text>
							</a>
							<span className="copy-cell" style={{ display: "block" }}>
								<Typography.Text
									type="secondary"
									copyable={{ text: copyId }}
									style={{ fontSize: 12 }}
									ellipsis={{
										tooltip: runId ? `完整任务 ID: ${runId}` : name,
									}}
								>
									ID: {displayId}
								</Typography.Text>
							</span>
							{showTemplateLine ? (
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
									// CYB-4470 B.3：模板版本 → system（蓝色，浅蓝底 + 深蓝字）。
									<Tag color={STATUS_TAG_PALETTE.system}>
										模板 v{templateVersion}
									</Tag>
								) : null}
								{scope === "prod" ? (
									// "正式版" 是已发布的稳定态 —— success（绿）。
									<Tag
										color={STATUS_TAG_PALETTE.success}
										style={{ fontSize: 11 }}
									>
										<LockOutlined /> 正式版
									</Tag>
								) : scope ? (
									// "Dev 草稿" 是中性辅助标记 —— paused（默认灰）。
									<Tag
										color={STATUS_TAG_PALETTE.paused}
										style={{ fontSize: 11 }}
									>
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
					// 终态以 failureReason 为准，活动态以 blockingReason 为准
					// （对齐后端 AnnotateRunDiagnostics：终态写 failureReason、
					// 活动态写 blockingReason）。避免残留的 blockingReason（如 stale
					// runtime_missing）在终态遮住真实失败原因。
					const terminal = isTerminalWorkflowStatus(s);
					const reason = terminal
						? record.failureReason || record.blockingReason
						: record.blockingReason || record.failureReason;
					const reasonMessage = terminal
						? record.message || record.blockingMessage
						: record.blockingMessage || record.message;
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
								// CYB-4470 B.3：排队是 warning 语义（不是 success/error 的
								// 终态，而是"等待资源"过渡态），归到 warning。
								<Tooltip title="并发已达上限或等待调度，工作流正在排队，待资源释放后自动开始执行">
									<Tag color={STATUS_TAG_PALETTE.warning}>排队中</Tag>
								</Tooltip>
							) : null}
							{isStaleRunningWorkflow(record) ? (
								// CYB-3491: 展示提示,不再对应"将自动标为失败"—— 任务保持
								// Argo 真值,恢复更新后 tag 自然消失。
								<Tooltip title="任务长时间无状态更新（>3 小时）,可能停滞或平台跟丢了状态回执。刷新页面查看最新状态。">
									<Tag color={STATUS_TAG_PALETTE.warning}>疑似停滞</Tag>
								</Tooltip>
							) : null}
							{redundant ? null : renderRunReasonTag(reason, reasonMessage, s)}
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
											// CYB-4470 B.1：复制图标 hover 才显现。
											className="copy-cell"
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
											// CYB-4470 B.1：复制图标 hover 才显现。
											className="copy-cell"
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
				// CYB-4477 P2-4：按 owner locale-aware 排序 —— 中文用户也走 "zh"
				// 排序规则，中英混排时按拼音首字母归位，比裸 localeCompare 更稳定。
				sorter: (a: WorkflowSummary, b: WorkflowSummary) =>
					((a as ExecutionRecord).owner ?? "").localeCompare(
						(b as ExecutionRecord).owner ?? "",
						"zh",
					),
				render: (owner: string) =>
					owner ? (
						// CYB-4470 B.1：复制图标 hover 才显现。
						<span className="copy-cell">
							<Typography.Text copyable={{ text: owner }}>
								{owner}
							</Typography.Text>
						</span>
					) : (
						<Typography.Text type="secondary">—</Typography.Text>
					),
			},
			{
				// CYB-3486: 产品上不再暴露"命名空间"概念,列表按"资源池"呈现。
				title: "资源池",
				dataIndex: "executionTargetId",
				key: "executionTargetId",
				width: 150,
				sorter: (a: WorkflowSummary, b: WorkflowSummary) =>
					poolLabelFor(a as ExecutionRecord).localeCompare(
						poolLabelFor(b as ExecutionRecord),
					),
				render: (_: unknown, record: ExecutionRecord) => {
					const targetId = record.executionTargetId ?? "";
					const target = targetId ? targetById.get(targetId) : undefined;
					// 资源池列表尚未加载/加载失败 —— 回退到命名空间显示,避免整列空白。
					if (targetById.size === 0) {
						const ns = record.argoNamespace;
						if (!ns)
							return <Typography.Text type="secondary">—</Typography.Text>;
						const nsColor = ns.includes("prod")
							? "red"
							: ns.includes("dev")
								? "blue"
								: "purple";
						return (
							<Tooltip title={ns}>
								<Tag color={nsColor} style={{ fontSize: 11 }}>
									{ns.split("/").pop() || ns}
								</Tag>
							</Tooltip>
						);
					}
					// 有 target id 但资源池已删除,或历史 run 未记录资源池。
					if (!target) {
						return (
							<Tooltip title={targetId ? "资源池已删除" : "未指定资源池"}>
								<Typography.Text type="secondary">—</Typography.Text>
							</Tooltip>
						);
					}
					const ns = target.namespace ?? "";
					const color = ns.includes("prod")
						? "red"
						: ns.includes("dev")
							? "blue"
							: "purple";
					return (
						<Tooltip
							title={`${target.name}（${target.cluster || "默认集群"}${
								ns ? ` / ${ns}` : ""
							}）`}
						>
							<Tag color={color} style={{ fontSize: 11 }}>
								{target.name}
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
				// CYB-4470 B.2：去掉 "asset_id=" 前缀后，每行省 ~60px；
				// 列宽从 240 → 180（batch scope 200 → 160）。
				title: "资产 ID",
				dataIndex: "labels",
				key: "labels",
				width: isBatchScope ? 160 : 180,
				render: (labels?: Record<string, string>) => (
					<WorkflowLabels labels={labels} />
				),
				responsive: isBatchScope ? BATCH_DETAIL_WIDE_ONLY : undefined,
			},
			{
				// CYB-4470 B.5：数值列右对齐 — 数据对齐规范，文本左 / 数值右，
				// 上下扫视时大小对比一目了然。
				title: "耗时",
				key: "duration",
				align: "right" as const,
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
				align: "right" as const,
				width: 100,
				sorter: (a: WorkflowSummary, b: WorkflowSummary) =>
					((a as ExecutionRecord).videoDurationSec ?? -1) -
					((b as ExecutionRecord).videoDurationSec ?? -1),
				render: (_: unknown, record: WorkflowSummary) => {
					const rec = record as ExecutionRecord;
					let sec = rec.videoDurationSec;
					if (sec == null) {
						// CYB-4011: fall back to a Grace-UUID-resolved duration.
						for (const a of rec.assetIds ?? []) {
							const r = resolvedDurationByUUID[a];
							if (r != null && r >= 0) {
								sec = r;
								break;
							}
						}
					}
					return formatVideoDurationSec(sec);
				},
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
				// CYB-4477 P2-2：行 hover 快捷操作栏 — 在"操作"列前面加一列紧凑
				// 的 hover-only 操作组（查看详情 / 复制 ID）。默认 opacity:0，
				// 鼠标进入该行时浮现，无需移动到右侧操作列也能直接命中高频动作。
				title: "快捷操作",
				key: "quickActions",
				width: 90,
				render: (_: unknown, record: ExecutionRecord) => {
					const executionKey = executionKeyForRecord(record);
					const hovered = hoveredExecutionKey === executionKey;
					// 复制 ID：runId 优先，回退到 name（与名称列 copyId 同口径）。
					const copyId = record.runId ?? record.name;
					const actions: HoverAction[] = [
						{
							key: "open-detail",
							label: "查看详情",
							icon: <EyeOutlined />,
							onClick: (event) => {
								event.stopPropagation();
								openWorkflowDetail(record);
							},
						},
						{
							key: "copy-id",
							label: "复制 ID",
							icon: <CopyOutlined />,
							onClick: (event) => {
								event.stopPropagation();
								void copyTextToClipboard(copyId).then((ok) => {
									if (ok) {
										messageApiRef.current.success("已复制");
									} else {
										messageApiRef.current.error("复制失败，请检查浏览器权限");
									}
								});
							},
						},
					];
					return (
						<HoverActionBar actions={actions} hovered={hovered} />
					);
				},
			},
			{
				// CYB-4470 B.7：操作列清理 — "模板 / 查看" 按钮移除（批次名称承担
				// 查看入口，操作 ⁝ 下拉收纳次要动作），仅保留 batch scope 必需的执行
				// 历史快捷按钮。列宽相应从 110 → 100。
				title: "操作",
				key: "actions",
				width: isBatchScope ? 100 : 100,
				render: (_: unknown, record: ExecutionRecord) => {
					const executionKey = executionKeyForRecord(record);
					const assetId = getWorkflowLabel(record.labels, "asset_id");
					const menuItems = getWorkflowOperationMenuItems(record);
					const hasOperationLoading = operationLoading?.startsWith(
						`${executionKey}:`,
					);

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

					// 非 batch scope：单 Dropdown "操作 ⁝" 收纳全部次要动作，
					// "模板 / 查看" 由批次名称点击承担。
					return (
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
		templateVersionsByExecutionKey,
		targetById,
		messageApi,
		navigate,
		resolvedDurationByUUID,
		// CYB-4477 P2-2：hover 状态变化需重建 columns 让快捷操作栏拿到最新值。
		hoveredExecutionKey,
	]);

	const showSkeleton = !initializedOnce;
	const tableScrollX = isBatchScope ? "max-content" : 1500;

	return (
		<>
			{/* CYB-4470 B.1：复制图标 hover 显现。把所有 .copy-cell 内部的
			    antd 复制按钮（.ant-typography-copy）默认 opacity:0，hover 父单元
			    时浮现。focus-within 让键盘 Tab 聚焦时也能看到复制入口（无障碍）。
			    使用 transition 0.15s 防止突变。 */}
			<style>
				{`
				.copy-cell .ant-typography-copy {
					opacity: 0;
					transition: opacity 0.15s ease-in-out;
				}
				.copy-cell:hover .ant-typography-copy,
				.copy-cell:focus-within .ant-typography-copy {
					opacity: 1;
				}
				`}
			</style>
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
				<Tooltip
					title={
						// CYB-4470 B.8：明确触发条件 —— < 2 项 hover 提示，> 3 项
						// 同样提示，让用户一眼看到对比按钮的合法区间。
						selectedExecutionKeys.length < 2
							? "请至少勾选 2 条记录进行对比"
							: selectedExecutionKeys.length > 3
								? "最多选择 3 条运行进行对比"
								: ""
					}
				>
					<Button
						// 满足 2-3 项条件时高亮 type=primary，强化"现在能点"的视觉反馈。
						type={
							selectedExecutionKeys.length >= 2 &&
							selectedExecutionKeys.length <= 3
								? "primary"
								: "default"
						}
						disabled={
							selectedExecutionKeys.length < 2 ||
							selectedExecutionKeys.length > 3
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
						{selectedExecutionKeys.length >= 2
							? ` (${selectedExecutionKeys.length})`
							: ""}
					</Button>
				</Tooltip>
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
					// CYB-3491: 防点击穿透 —— 默认 Select 的下拉浮层挂载在
					// triggerNode 的父节点,当外层有 z-index 更高的浮层
					// (侧栏抽屉/导航)时,点击选项会击穿到底层导航链接,把
					// 用户带到 "MCAP 文件" 之类。挂到 body 上避免此问题。
					getPopupContainer={() => document.body}
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
							// CYB-4477 P2-2：记录当前 hover 行的 executionKey，
							// 供"快捷操作"列的 HoverActionBar 控制 opacity 0/1。
							onMouseEnter: () =>
								setHoveredExecutionKey(executionKeyForRecord(record)),
							onMouseLeave: () => setHoveredExecutionKey(null),
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
		</>
	);
}
