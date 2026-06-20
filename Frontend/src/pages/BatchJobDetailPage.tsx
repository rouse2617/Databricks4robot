import {
	ArrowLeftOutlined,
	DownloadOutlined,
	PauseCircleOutlined,
	PlayCircleOutlined,
	RedoOutlined,
	ReloadOutlined,
} from "@ant-design/icons";
import {
	Alert,
	App,
	Button,
	Card,
	Descriptions,
	Drawer,
	Dropdown,
	Modal,
	Progress,
	Radio,
	Select,
	Skeleton,
	Space,
	Table,
	Tag,
	Typography,
} from "antd";
import dayjs from "dayjs";
import { useCallback, useEffect, useRef, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import {
	type BatchJob,
	type BatchNodeFailureItem,
	type BatchNodeSummary,
	type BatchNodeSummaryNode,
	batchJobProgress,
	continueFullBatchJob,
	getBatchJob,
	getBatchNodeSummary,
	listBatchNodeFailures,
	pauseBatchJob,
	rerunBatchJob,
	resumeBatchJob,
} from "../api/batchJobApi";
import {
	listPipelineVersions,
	type PipelineTemplate,
} from "../api/pipelineApi";
import {
	listRunChildren,
	type RunChildrenResponse,
	type RunChildSummary,
} from "../api/runApi";
import type { WorkflowSummary } from "../api/workflowApi";
import {
	goBackFromBatchJobDetail,
	workflowDetailLocationState,
} from "../lib/pipelineNavigation";
import {
	formatBatchJobStatus,
	formatWorkflowPhaseLabel,
	resolveStatusTagColor,
} from "../lib/statusLabels";
import { WorkflowExecutionList } from "./WorkflowExecutionList";

const { Title, Text } = Typography;

type RerunScope =
	| "failed"
	| "incomplete"
	| "completed"
	| "custom"
	| "node_failed";

type NodeDrawerFilter = "failed" | "running" | "pending";

type SubtaskNodeFilter = {
	pipelineNodeId: string;
	nodeStatus?: string;
	label: string;
};

interface RerunModalState {
	scope: RerunScope;
	extra: Record<string, unknown>;
}

const stableSerialize = (value: unknown): string =>
	JSON.stringify(value, (_key, current) => {
		if (current && typeof current === "object" && !Array.isArray(current)) {
			return Object.fromEntries(
				Object.entries(current as Record<string, unknown>).sort(([a], [b]) =>
					a.localeCompare(b),
				),
			);
		}
		return current;
	});

const sameValue = (left: unknown, right: unknown): boolean =>
	stableSerialize(left) === stableSerialize(right);

type BatchTemplateMeta = {
	name: string;
	versions: PipelineTemplate[];
};

const batchDetailCacheMs = 3000;
const batchJobInFlight = new Map<string, Promise<BatchJob>>();
const batchJobCache = new Map<string, { value: BatchJob; cachedAt: number }>();
const batchNodeSummaryInFlight = new Map<string, Promise<BatchNodeSummary>>();
const batchNodeSummaryCache = new Map<
	string,
	{ value: BatchNodeSummary; cachedAt: number }
>();
const batchTemplateMetaInFlight = new Map<string, Promise<BatchTemplateMeta>>();
const batchRunTreeInFlight = new Map<string, Promise<RunChildrenResponse>>();
const batchRunTreeCache = new Map<
	string,
	{ value: RunChildrenResponse; cachedAt: number }
>();

export function resetBatchJobDetailRequestCacheForTests(): void {
	if (!import.meta.env.TEST) return;
	batchJobInFlight.clear();
	batchJobCache.clear();
	batchNodeSummaryInFlight.clear();
	batchNodeSummaryCache.clear();
	batchTemplateMetaInFlight.clear();
	batchRunTreeInFlight.clear();
	batchRunTreeCache.clear();
}

async function loadBatchJob(
	jobId: string,
	opts?: { useCache?: boolean; force?: boolean },
): Promise<BatchJob> {
	if (opts?.force) {
		batchJobCache.delete(jobId);
	}
	if (opts?.useCache) {
		const cached = batchJobCache.get(jobId);
		if (cached && Date.now() - cached.cachedAt < batchDetailCacheMs) {
			return cached.value;
		}
	}
	const inFlight = batchJobInFlight.get(jobId);
	if (inFlight) return inFlight;

	const request = getBatchJob(jobId)
		.then((value) => {
			batchJobCache.set(jobId, { value, cachedAt: Date.now() });
			return value;
		})
		.finally(() => {
			batchJobInFlight.delete(jobId);
		});

	batchJobInFlight.set(jobId, request);
	return request;
}

async function loadBatchNodeSummary(
	jobId: string,
	opts?: { useCache?: boolean; force?: boolean },
): Promise<BatchNodeSummary> {
	if (opts?.force) {
		batchNodeSummaryCache.delete(jobId);
	}
	if (opts?.useCache) {
		const cached = batchNodeSummaryCache.get(jobId);
		if (cached && Date.now() - cached.cachedAt < batchDetailCacheMs) {
			return cached.value;
		}
	}
	const inFlight = batchNodeSummaryInFlight.get(jobId);
	if (inFlight) return inFlight;

	const request = getBatchNodeSummary(jobId)
		.then((value) => {
			batchNodeSummaryCache.set(jobId, { value, cachedAt: Date.now() });
			return value;
		})
		.finally(() => {
			batchNodeSummaryInFlight.delete(jobId);
		});

	batchNodeSummaryInFlight.set(jobId, request);
	return request;
}

async function loadBatchTemplateMeta(
	templateId: string,
): Promise<BatchTemplateMeta> {
	const inFlight = batchTemplateMetaInFlight.get(templateId);
	if (inFlight) return inFlight;

	const request = listPipelineVersions(templateId)
		.then((versions) => {
			const template = versions.find((item) => item.id === templateId);
			return {
				name: template?.name ?? versions[0]?.name ?? templateId,
				versions,
			};
		})
		.catch(() => ({
			name: templateId,
			versions: [] as PipelineTemplate[],
		}))
		.finally(() => {
			batchTemplateMetaInFlight.delete(templateId);
		});

	batchTemplateMetaInFlight.set(templateId, request);
	return request;
}

async function loadBatchRunTree(
	jobId: string,
	opts?: { useCache?: boolean; force?: boolean },
): Promise<RunChildrenResponse> {
	if (opts?.force) {
		batchRunTreeCache.delete(jobId);
	}
	if (opts?.useCache) {
		const cached = batchRunTreeCache.get(jobId);
		if (cached && Date.now() - cached.cachedAt < batchDetailCacheMs) {
			return cached.value;
		}
	}
	const inFlight = batchRunTreeInFlight.get(jobId);
	if (inFlight) return inFlight;

	const request = listRunChildren(jobId)
		.then((value) => {
			batchRunTreeCache.set(jobId, { value, cachedAt: Date.now() });
			return value;
		})
		.finally(() => {
			batchRunTreeInFlight.delete(jobId);
		});

	batchRunTreeInFlight.set(jobId, request);
	return request;
}

function defaultRerunTemplateVersion(
	job: BatchJob,
	versions: PipelineTemplate[],
): number | undefined {
	if (job.templateVersion && job.templateVersion > 0) {
		return job.templateVersion;
	}
	if (versions.length === 0) {
		return undefined;
	}
	return Math.max(...versions.map((item) => item.version));
}

export function batchJobPollIntervalMs(status?: string | null): number | null {
	switch (status) {
		case "running":
			return 5_000;
		case "paused":
			return 30_000;
		default:
			return null;
	}
}

function nodeStatusForDrawerFilter(
	filter: NodeDrawerFilter,
): string | undefined {
	switch (filter) {
		case "failed":
			return "Failed";
		case "running":
			return "Running";
		case "pending":
			return "Pending";
	}
}

function subtaskNodeFilterLabel(
	node: BatchNodeSummaryNode,
	filter: NodeDrawerFilter,
): string {
	const statusLabel =
		filter === "failed" ? "失败" : filter === "running" ? "运行中" : "等待中";
	return `${node.displayName} · ${statusLabel}`;
}

function exportFailuresCsv(
	items: BatchNodeFailureItem[],
	filename: string,
): void {
	const header = "assetId,status,message,workflowName\n";
	const rows = items
		.map((item) => {
			const message = (item.message ?? "").replace(/"/g, '""');
			return `"${item.assetId}","${item.status}","${message}","${item.workflowName ?? ""}"`;
		})
		.join("\n");
	const blob = new Blob([header + rows], { type: "text/csv;charset=utf-8" });
	const url = URL.createObjectURL(blob);
	const anchor = document.createElement("a");
	anchor.href = url;
	anchor.download = filename;
	anchor.click();
	URL.revokeObjectURL(url);
}

export function formatRerunFeedback(result: {
	status: string;
	matchedCount: number;
	retriedCount?: number;
	skipped?: Array<unknown>;
}): { level: "success" | "warning" | "error"; text: string } {
	if (result.status === "failed") {
		return { level: "error", text: "重跑失败，没有可重新提交的子任务" };
	}
	if (
		result.status === "partial_success" ||
		(result.skipped?.length ?? 0) > 0
	) {
		return {
			level: "warning",
			text: `重跑部分提交成功：${result.retriedCount ?? result.matchedCount} / ${result.matchedCount}`,
		};
	}
	return { level: "success", text: "重跑已提交" };
}

function runTreeSummaryItems(summary: RunChildSummary) {
	return [
		{ label: "总数", value: summary.total },
		{ label: "运行中", value: summary.runningCount },
		{ label: "等待中", value: summary.pendingCount },
		{ label: "成功", value: summary.succeededCount },
		{ label: "失败", value: summary.failedCount },
		{ label: "暂停", value: summary.suspendedCount },
	].filter((item) => item.value > 0 || item.label === "总数");
}

export default function BatchJobDetailPage() {
	const { id = "" } = useParams();
	const navigate = useNavigate();
	const location = useLocation();
	const { message } = App.useApp();
	const messageRef = useRef(message);
	messageRef.current = message;
	const jobRef = useRef<BatchJob | null>(null);
	const templateNameRef = useRef("");
	const templateVersionsRef = useRef<PipelineTemplate[]>([]);
	const [job, setJob] = useState<BatchJob | null>(null);
	const [templateName, setTemplateName] = useState("");
	const [loading, setLoading] = useState(true);
	const [actionLoading, setActionLoading] = useState<string | null>(null);
	const [nodeSummary, setNodeSummary] = useState<BatchNodeSummary | null>(null);
	const [nodeSummaryLoading, setNodeSummaryLoading] = useState(false);
	const [runTree, setRunTree] = useState<RunChildrenResponse | null>(null);
	const [runTreeLoading, setRunTreeLoading] = useState(false);
	const [drawerNode, setDrawerNode] = useState<BatchNodeSummaryNode | null>(
		null,
	);
	const [drawerFilter, setDrawerFilter] = useState<NodeDrawerFilter>("failed");
	const [nodeFailures, setNodeFailures] = useState<BatchNodeFailureItem[]>([]);
	const [nodeFailureTotal, setNodeFailureTotal] = useState(0);
	const [nodeFailureLoading, setNodeFailureLoading] = useState(false);
	const [selectedRuns, setSelectedRuns] = useState<WorkflowSummary[]>([]);
	const [templateVersions, setTemplateVersions] = useState<PipelineTemplate[]>(
		[],
	);
	const [rerunModal, setRerunModal] = useState<RerunModalState | null>(null);
	const [rerunTemplateVersion, setRerunTemplateVersion] = useState<
		number | undefined
	>();
	const [rerunPreviewCount, setRerunPreviewCount] = useState<number | null>(
		null,
	);
	const [rerunPreviewLoading, setRerunPreviewLoading] = useState(false);
	const [pauseModalOpen, setPauseModalOpen] = useState(false);
	const [pauseStopRunning, setPauseStopRunning] = useState(false);
	const [subtaskNodeFilter, setSubtaskNodeFilter] =
		useState<SubtaskNodeFilter | null>(null);
	jobRef.current = job;
	templateNameRef.current = templateName;
	templateVersionsRef.current = templateVersions;

	const refresh = useCallback(
		async (opts?: {
			force?: boolean;
			silent?: boolean;
			useCache?: boolean;
		}) => {
			if (!id) return;
			if (!opts?.silent) {
				setLoading(true);
			}
			const showNodeSummaryLoading = !opts?.silent;
			if (showNodeSummaryLoading) {
				setNodeSummaryLoading(true);
			}
			if (!opts?.silent) {
				setRunTreeLoading(true);
			}
			const requestOptions = {
				force: opts?.force,
				useCache: opts?.useCache,
			};
			const nodeSummaryTask = loadBatchNodeSummary(id, requestOptions)
				.then((summary) =>
					setNodeSummary((current) =>
						sameValue(current, summary) ? current : summary,
					),
				)
				.catch(() =>
					setNodeSummary((current) => (current === null ? current : null)),
				)
				.finally(() => {
					if (showNodeSummaryLoading) {
						setNodeSummaryLoading(false);
					}
				});
			const runTreeTask = loadBatchRunTree(id, requestOptions)
				.then((tree) =>
					setRunTree((current) => (sameValue(current, tree) ? current : tree)),
				)
				.catch(() =>
					setRunTree((current) => (current === null ? current : null)),
				)
				.finally(() => {
					if (!opts?.silent) {
						setRunTreeLoading(false);
					}
				});
			try {
				const jobData = await loadBatchJob(id, requestOptions);
				setJob((current) => (sameValue(current, jobData) ? current : jobData));
				const shouldRefreshTemplateMeta =
					!opts?.silent ||
					jobRef.current?.templateId !== jobData.templateId ||
					templateVersionsRef.current.length === 0 ||
					!templateNameRef.current;
				if (shouldRefreshTemplateMeta) {
					const meta = await loadBatchTemplateMeta(jobData.templateId);
					setTemplateVersions((current) =>
						sameValue(current, meta.versions) ? current : meta.versions,
					);
					setTemplateName((current) =>
						current === meta.name ? current : meta.name,
					);
				}
				void nodeSummaryTask;
				void runTreeTask;
			} catch (err) {
				if (!opts?.silent) {
					messageRef.current.error(`加载批次详情失败：${String(err)}`);
				}
				if (showNodeSummaryLoading) {
					setNodeSummaryLoading(false);
				}
				if (!opts?.silent) {
					setRunTreeLoading(false);
				}
			} finally {
				if (!opts?.silent) {
					setLoading(false);
				}
			}
		},
		[id],
	);

	useEffect(() => {
		void refresh({ useCache: true });
	}, [refresh]);

	const pollIntervalMs = batchJobPollIntervalMs(job?.status);

	useEffect(() => {
		if (pollIntervalMs === null) {
			return;
		}
		const timer = window.setInterval(() => {
			void refresh({ silent: true });
		}, pollIntervalMs);
		return () => window.clearInterval(timer);
	}, [pollIntervalMs, refresh]);

	const backToBatchList = useCallback(() => {
		goBackFromBatchJobDetail(navigate, location.state);
	}, [location.state, navigate]);

	const runAction = async (action: "resume") => {
		if (!job) return;
		setActionLoading(action);
		try {
			await resumeBatchJob(job.id);
			message.success("操作已提交");
			await refresh({ force: true });
		} catch (err) {
			message.error(`操作失败：${String(err)}`);
		} finally {
			setActionLoading(null);
		}
	};

	const submitPause = async () => {
		if (!job) return;
		setActionLoading("pause");
		try {
			const result = await pauseBatchJob(job.id, {
				stopRunning: pauseStopRunning,
			});
			if (pauseStopRunning && (result.stoppedCount ?? 0) > 0) {
				message.success(
					`批次已暂停，已停止 ${result.stoppedCount} 条运行中的子任务`,
				);
			} else if (pauseStopRunning && (result.stopFailedCount ?? 0) > 0) {
				message.warning(
					`批次已暂停，但有 ${result.stopFailedCount} 条子任务停止失败`,
				);
			} else {
				message.success("批次已暂停");
			}
			setPauseModalOpen(false);
			setPauseStopRunning(false);
			await refresh({ force: true });
		} catch (err) {
			message.error(`暂停失败：${String(err)}`);
		} finally {
			setActionLoading(null);
		}
	};

	const applySubtaskNodeFilter = (
		node: BatchNodeSummaryNode,
		filter: NodeDrawerFilter,
	) => {
		setSubtaskNodeFilter({
			pipelineNodeId: node.pipelineNodeId,
			nodeStatus: nodeStatusForDrawerFilter(filter),
			label: subtaskNodeFilterLabel(node, filter),
		});
		setDrawerNode(null);
	};

	const previewRerun = useCallback(
		async (
			scope: RerunScope,
			templateVersion: number | undefined,
			extra: Record<string, unknown> = {},
		) => {
			if (!job) return;
			setRerunPreviewLoading(true);
			try {
				const dryRun = await rerunBatchJob(job.id, {
					scope,
					templateVersion,
					dryRun: true,
					...extra,
				});
				setRerunPreviewCount(dryRun.matchedCount);
			} catch (err) {
				setRerunPreviewCount(null);
				messageRef.current.error(`预览重跑失败：${String(err)}`);
			} finally {
				setRerunPreviewLoading(false);
			}
		},
		[job],
	);

	const openRerunModal = (
		scope: RerunScope,
		extra: Record<string, unknown> = {},
	) => {
		if (!job) return;
		const version = defaultRerunTemplateVersion(job, templateVersions);
		setRerunModal({ scope, extra });
		setRerunTemplateVersion(version);
		setRerunPreviewCount(null);
		void previewRerun(scope, version, extra);
	};

	const closeRerunModal = () => {
		setRerunModal(null);
		setRerunPreviewCount(null);
	};

	const submitRerun = async () => {
		if (!job || !rerunModal) return;
		setActionLoading(`rerun-${rerunModal.scope}`);
		try {
			const result = await rerunBatchJob(job.id, {
				scope: rerunModal.scope,
				templateVersion: rerunTemplateVersion,
				...rerunModal.extra,
			});
			const feedback = formatRerunFeedback(result);
			if (feedback.level === "error") {
				message.error(feedback.text);
			} else if (feedback.level === "warning") {
				message.warning(feedback.text);
			} else {
				message.success(feedback.text);
			}
			closeRerunModal();
			await refresh({ force: true });
		} catch (err) {
			message.error(`重跑失败：${String(err)}`);
		} finally {
			setActionLoading(null);
		}
	};

	const loadNodeFailures = useCallback(
		async (node: BatchNodeSummaryNode, filter: NodeDrawerFilter) => {
			if (!id) return;
			setNodeFailureLoading(true);
			try {
				const statusByFilter: Record<NodeDrawerFilter, string | undefined> = {
					failed: undefined,
					running: "Running",
					pending: "Pending",
				};
				const result = await listBatchNodeFailures(id, {
					pipelineNodeId: node.pipelineNodeId,
					status: statusByFilter[filter],
					page: 1,
					pageSize: 20,
				});
				setNodeFailures(result.items);
				setNodeFailureTotal(result.total);
			} catch (err) {
				messageRef.current.error(`加载节点明细失败：${String(err)}`);
			} finally {
				setNodeFailureLoading(false);
			}
		},
		[id],
	);

	const openNodeDrawer = (
		node: BatchNodeSummaryNode,
		filter: NodeDrawerFilter,
	) => {
		setDrawerNode(node);
		setDrawerFilter(filter);
		void loadNodeFailures(node, filter);
	};

	const selectedAssetIds = selectedRuns
		.map((item) => item.labels?.asset_id ?? item.labels?.assetId ?? "")
		.filter(Boolean);

	if (loading && !job) {
		return <Skeleton active paragraph={{ rows: 8 }} />;
	}

	if (!job) {
		return (
			<div style={{ padding: 24 }}>
				<Button icon={<ArrowLeftOutlined />} onClick={backToBatchList}>
					返回批量任务
				</Button>
				<Typography.Paragraph type="secondary" style={{ marginTop: 16 }}>
					批次任务不存在或已被删除。
				</Typography.Paragraph>
			</div>
		);
	}

	return (
		<div className="pipeline-batch-detail" style={{ padding: "0 0 24px" }}>
			<Space style={{ marginBottom: 16 }}>
				<Button icon={<ArrowLeftOutlined />} onClick={backToBatchList}>
					返回批量任务
				</Button>
				<Button
					icon={<ReloadOutlined />}
					onClick={() => void refresh({ force: true })}
				>
					刷新
				</Button>
			</Space>

			<Card style={{ marginBottom: 16 }}>
				<div
					style={{
						display: "flex",
						justifyContent: "space-between",
						gap: 16,
						flexWrap: "wrap",
					}}
				>
					<div>
						<Title level={4} style={{ margin: 0 }}>
							{job.name}
						</Title>
						<Text type="secondary">批次 ID: {job.id}</Text>
					</div>
					<Space wrap>
						{job.status === "running" ? (
							<Button
								icon={<PauseCircleOutlined />}
								loading={actionLoading === "pause"}
								onClick={() => setPauseModalOpen(true)}
							>
								暂停
							</Button>
						) : null}
						{job.status === "paused" ? (
							<Button
								type="primary"
								icon={<PlayCircleOutlined />}
								loading={actionLoading === "resume"}
								onClick={() => void runAction("resume")}
							>
								继续
							</Button>
						) : null}
						{job.failedCount > 0 ? (
							<Button
								icon={<RedoOutlined />}
								loading={actionLoading === "rerun-failed"}
								onClick={() => openRerunModal("failed")}
							>
								重试失败项
							</Button>
						) : null}
						<Dropdown
							menu={{
								items: [
									{ key: "failed", label: "重试全部失败" },
									{ key: "incomplete", label: "重试未完成" },
									{ key: "completed", label: "重试已完成" },
									{
										key: "custom",
										label: `重试选中项 (${selectedRuns.length})`,
										disabled: selectedAssetIds.length === 0,
									},
								],
								onClick: ({ key }) => {
									if (key === "custom") {
										openRerunModal("custom", {
											assetIds: selectedAssetIds,
										});
										return;
									}
									openRerunModal(key as "failed" | "incomplete" | "completed");
								},
							}}
						>
							<Button
								icon={<RedoOutlined />}
								loading={actionLoading?.startsWith("rerun-")}
							>
								重跑
							</Button>
						</Dropdown>
						{job.pilotPhase === "review" ? (
							<Button
								type="primary"
								icon={<PlayCircleOutlined />}
								loading={actionLoading === "continue-full"}
								onClick={async () => {
									setActionLoading("continue-full");
									try {
										await continueFullBatchJob(job.id);
										message.success("已继续全量");
										await refresh({ force: true });
									} catch (err) {
										message.error(`继续全量失败：${String(err)}`);
									} finally {
										setActionLoading(null);
									}
								}}
							>
								继续全量
							</Button>
						) : null}
					</Space>
				</div>

				<Descriptions
					column={{ xs: 1, sm: 2, md: 3 }}
					style={{ marginTop: 16 }}
					items={[
						{ label: "模板", children: templateName },
						{
							label: "状态",
							children: (
								<Tag color={resolveStatusTagColor(job.status)}>
									{formatBatchJobStatus(job.status)}
								</Tag>
							),
						},
						{
							label: "子任务数",
							children: job.totalCount,
						},
						{
							label: "模板版本",
							children: job.templateVersion
								? `v${job.templateVersion}`
								: "当前",
						},
						{
							label: "Pilot",
							children: job.pilotCount
								? `${job.pilotPhase} · ${job.pilotCount}`
								: "未启用",
						},
						{
							label: "成功 / 失败",
							children: `${job.completedCount} / ${job.failedCount}`,
						},
						{
							label: "创建时间",
							children: dayjs(job.createdAt).format("YYYY-MM-DD HH:mm:ss"),
						},
						{
							label: "更新时间",
							children: dayjs(job.updatedAt).format("YYYY-MM-DD HH:mm:ss"),
						},
					]}
				/>

				<div style={{ marginTop: 16 }}>
					<Progress
						percent={batchJobProgress(job)}
						status={job.failedCount > 0 ? "exception" : "active"}
					/>
				</div>
			</Card>

			{job.pilotPhase === "review" ? (
				<Alert
					type="warning"
					showIcon
					message="Pilot 试跑已完成，等待确认后继续全量"
					style={{ marginBottom: 16 }}
				/>
			) : null}

			{job.failedCount > 0 && nodeSummary ? (
				<Alert
					type="error"
					showIcon
					message={`本批次有 ${job.failedCount} 条子任务失败`}
					description={
						<Space direction="vertical" size={4}>
							<span>
								可在下方「节点概览」点击失败数查看资产明细，或使用「重试失败项」。
							</span>
							{nodeSummary.nodes
								.map((node) => ({
									node,
									failed: (node.counts.Failed ?? 0) + (node.counts.Error ?? 0),
								}))
								.filter((entry) => entry.failed > 0)
								.sort((a, b) => b.failed - a.failed)
								.slice(0, 3)
								.map((entry) => (
									<span key={entry.node.pipelineNodeId}>
										· {entry.node.displayName}：{entry.failed} 失败（
										{(entry.node.failureRate * 100).toFixed(1)}%）
									</span>
								))}
						</Space>
					}
					style={{ marginBottom: 16 }}
				/>
			) : null}

			{runTree || runTreeLoading ? (
				<Card
					className="pipeline-batch-run-tree"
					title="批次运行"
					style={{ marginBottom: 16 }}
					extra={
						<Button
							type="link"
							size="small"
							onClick={() => navigate(`/runs/${encodeURIComponent(job.id)}`)}
						>
							查看批次 Run
						</Button>
					}
				>
					{runTreeLoading && !runTree ? (
						<Skeleton active paragraph={{ rows: 2 }} title={false} />
					) : runTree?.summary ? (
						<Space direction="vertical" size={12} style={{ width: "100%" }}>
							<Space wrap>
								<Tag
									color={resolveStatusTagColor(runTree.summary.aggregateStatus)}
								>
									{formatWorkflowPhaseLabel(runTree.summary.aggregateStatus)}
								</Tag>
								{runTreeSummaryItems(runTree.summary).map((item) => (
									<Text key={item.label} type="secondary">
										{item.label} {item.value}
									</Text>
								))}
								<Text type="secondary">
									子运行 {runTree.relations?.length ?? 0}
								</Text>
							</Space>
							{runTree.items.length > 0 ? (
								<Space wrap size={[8, 8]}>
									{runTree.items.slice(0, 8).map((child) => (
										<Button
											key={child.id}
											size="small"
											onClick={() =>
												navigate(`/runs/${encodeURIComponent(child.id)}`)
											}
										>
											{child.assetIds?.[0] ?? child.id.slice(0, 8)}
										</Button>
									))}
									{runTree.items.length > 8 ? (
										<Text type="secondary">
											+{runTree.items.length - 8} 个子运行
										</Text>
									) : null}
								</Space>
							) : (
								<Text type="secondary">子运行尚未生成</Text>
							)}
						</Space>
					) : null}
				</Card>
			) : null}

			<Card
				className="pipeline-batch-node-overview"
				title="节点概览"
				style={{ marginBottom: 16 }}
			>
				{nodeSummary && !nodeSummary.dataCoverage.complete ? (
					<Alert
						type="info"
						showIcon
						message="节点状态仍在同步中，概览会随刷新更新"
						style={{ marginBottom: 12 }}
					/>
				) : null}
				{nodeSummaryLoading && !nodeSummary ? (
					<Skeleton active paragraph={{ rows: 4 }} title={false} />
				) : (
					<Table
						size="small"
						rowKey="pipelineNodeId"
						dataSource={nodeSummary?.nodes ?? []}
						loading={nodeSummaryLoading}
						locale={{ emptyText: "节点进度尚未生成" }}
						pagination={false}
						columns={[
							{
								title: "节点",
								render: (_, record) =>
									`${record.dagOrder}. ${record.displayName}`,
							},
							{ title: "成功", dataIndex: ["counts", "Succeeded"] },
							{
								title: "失败",
								render: (_, record) => {
									const failed =
										(record.counts.Failed ?? 0) + (record.counts.Error ?? 0);
									return failed > 0 ? (
										<Button
											type="link"
											size="small"
											onClick={() => openNodeDrawer(record, "failed")}
										>
											{failed} 失败
										</Button>
									) : (
										0
									);
								},
							},
							{
								title: "运行中",
								render: (_, record) => {
									const running = record.counts.Running ?? 0;
									return running > 0 ? (
										<Button
											type="link"
											size="small"
											onClick={() => openNodeDrawer(record, "running")}
										>
											{running}
										</Button>
									) : (
										0
									);
								},
							},
							{
								title: "未开始",
								render: (_, record) => {
									const pending = record.counts.Pending ?? 0;
									return pending > 0 ? (
										<Button
											type="link"
											size="small"
											onClick={() => openNodeDrawer(record, "pending")}
										>
											{pending}
										</Button>
									) : (
										0
									);
								},
							},
							{
								title: "失败率",
								render: (_, record) =>
									`${(record.failureRate * 100).toFixed(2)}%`,
							},
						]}
					/>
				)}
			</Card>

			<WorkflowExecutionList
				active
				batchJobId={job.id}
				embedded
				nodeFilter={subtaskNodeFilter ?? undefined}
				onClearNodeFilter={() => setSubtaskNodeFilter(null)}
				onSelectionChange={setSelectedRuns}
				title="子任务执行记录"
			/>

			<Drawer
				title={
					drawerNode
						? `${drawerNode.displayName} · ${
								drawerFilter === "failed"
									? "失败资产"
									: drawerFilter === "running"
										? "运行中资产"
										: "等待中资产"
							}`
						: "节点明细"
				}
				open={Boolean(drawerNode)}
				width={720}
				onClose={() => setDrawerNode(null)}
				extra={
					drawerNode ? (
						<Space>
							<Button
								type="link"
								onClick={() => applySubtaskNodeFilter(drawerNode, drawerFilter)}
							>
								在子任务列表筛选
							</Button>
							{drawerFilter === "failed" ? (
								<>
									<Button
										icon={<DownloadOutlined />}
										disabled={nodeFailures.length === 0}
										onClick={() =>
											exportFailuresCsv(
												nodeFailures,
												`batch-${job.id.slice(0, 8)}-${drawerNode.pipelineNodeId}-failures.csv`,
											)
										}
									>
										导出 CSV
									</Button>
									<Button
										icon={<RedoOutlined />}
										loading={actionLoading === "rerun-node_failed"}
										onClick={() =>
											openRerunModal("node_failed", {
												pipelineNodeId: drawerNode.pipelineNodeId,
											})
										}
									>
										重试这 {nodeFailureTotal} 条
									</Button>
								</>
							) : null}
						</Space>
					) : null
				}
			>
				<Table
					size="small"
					rowKey="backfillItemId"
					loading={nodeFailureLoading}
					dataSource={nodeFailures}
					pagination={false}
					columns={[
						{ title: "资产", dataIndex: "assetId" },
						{
							title: "状态",
							dataIndex: "status",
							render: (status: string) => formatWorkflowPhaseLabel(status),
						},
						{ title: "消息", dataIndex: "message", ellipsis: true },
						{
							title: "运行",
							render: (_, record) => (
								<Button
									type="link"
									size="small"
									onClick={() =>
										navigate(
											record.runId
												? `/runs/${encodeURIComponent(record.runId)}`
												: `/pipeline/executions/${encodeURIComponent(
														record.workflowName,
													)}`,
											{ state: workflowDetailLocationState(job.id) },
										)
									}
								>
									查看
								</Button>
							),
						},
					]}
				/>
			</Drawer>

			<Modal
				title="确认重跑"
				open={Boolean(rerunModal)}
				onCancel={closeRerunModal}
				onOk={() => void submitRerun()}
				okText="确认重跑"
				cancelText="取消"
				confirmLoading={actionLoading?.startsWith("rerun-") ?? false}
				okButtonProps={{
					disabled:
						rerunPreviewLoading ||
						rerunPreviewCount === null ||
						rerunPreviewCount === 0,
				}}
				destroyOnHidden
			>
				<Space direction="vertical" size={12} style={{ width: "100%" }}>
					<div>
						<Text type="secondary">模板版本</Text>
						<Select
							aria-label="重跑模板版本"
							style={{ width: "100%", marginTop: 8 }}
							value={rerunTemplateVersion}
							placeholder="请选择版本"
							loading={templateVersions.length === 0 && rerunPreviewLoading}
							options={templateVersions.map((version) => ({
								value: version.version,
								label: `v${version.version}${
									version.version === job.templateVersion ? "（批次版本）" : ""
								}`,
							}))}
							onChange={(value) => {
								setRerunTemplateVersion(value);
								if (rerunModal) {
									void previewRerun(rerunModal.scope, value, rerunModal.extra);
								}
							}}
						/>
					</div>
					{rerunPreviewLoading ? (
						<Text type="secondary">正在计算将重跑的子任务数…</Text>
					) : rerunPreviewCount !== null ? (
						<Text>
							将重新提交 {rerunPreviewCount} 条子任务，模板版本 v
							{rerunTemplateVersion ?? "当前"}。旧 run 记录会保留。
						</Text>
					) : (
						<Text type="secondary">请选择模板版本后预览重跑范围。</Text>
					)}
				</Space>
			</Modal>

			<Modal
				title="暂停批次"
				open={pauseModalOpen}
				onCancel={() => {
					setPauseModalOpen(false);
					setPauseStopRunning(false);
				}}
				onOk={() => void submitPause()}
				okText="确认暂停"
				cancelText="取消"
				confirmLoading={actionLoading === "pause"}
				destroyOnHidden
			>
				<Radio.Group
					value={pauseStopRunning}
					onChange={(event) => setPauseStopRunning(event.target.value)}
					style={{ display: "flex", flexDirection: "column", gap: 12 }}
				>
					<Radio value={false}>
						仅暂停调度（不再启动新的子任务，运行中的继续执行）
					</Radio>
					<Radio value={true}>
						暂停并停止运行中的子任务（向 Argo 发送停止信号）
					</Radio>
				</Radio.Group>
			</Modal>
		</div>
	);
}
