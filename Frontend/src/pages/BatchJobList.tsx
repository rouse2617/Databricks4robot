import {
	CopyOutlined,
	DownloadOutlined,
	PlayCircleOutlined,
	PoweroffOutlined,
	RedoOutlined,
	ReloadOutlined,
} from "@ant-design/icons";
import {
	App,
	Button,
	Dropdown,
	Empty,
	Input,
	Select,
	Skeleton,
	Space,
	Table,
	Tag,
	Tooltip,
	Typography,
} from "antd";
import { BatchProgressCell } from "../components/pipeline/BatchProgressCell";
import type { ColumnsType } from "antd/es/table";
import dayjs from "dayjs";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
	type BatchJob,
	deriveBatchJobStatus,
	listBatchJobs,
	pauseBatchJob,
	resumeBatchJob,
	retryFailedBatchItems,
} from "../api/batchJobApi";
import {
	type ExecutionTarget,
	listExecutionTargets,
	listPipelines,
	type PipelineTemplate,
} from "../api/pipelineApi";
import { useVisibleInterval } from "../hooks/useVisibleInterval";
import {
	type BatchExportStatusFilter,
	batchJobCompletionAt,
	batchJobCreatedAtMs,
	batchJobRunDurationSeconds,
	copyAssetIdsToClipboard,
	exportAssetIdsCsv,
	fetchAssetIdsForBatches,
	formatDurationMs,
	formatDurationSeconds,
	sortBatchJobsByCreatedDesc,
	statusFilterPredicate,
	statusFilterServerValue,
} from "../lib/batchJobs";
import { batchJobDetailLocationState } from "../lib/pipelineNavigation";
import {
	formatBatchJobStatus,
	resolveStatusTagColor,
} from "../lib/statusLabels";
import {
	batchTargetId,
	effectivePriorityFromFilter,
	priorityBadge,
} from "../lib/workflowPriority";

const { Title, Text } = Typography;

interface BatchJobListProps {
	active?: boolean;
}

export function BatchJobList({ active = true }: BatchJobListProps) {
	const { message, modal } = App.useApp();
	const navigate = useNavigate();
	const [jobs, setJobs] = useState<BatchJob[]>([]);
	const [templates, setTemplates] = useState<PipelineTemplate[]>([]);
	const [targets, setTargets] = useState<ExecutionTarget[]>([]);
	const [loading, setLoading] = useState(false);
	const [actionLoading, setActionLoading] = useState<string | null>(null);
	const [nameFilter, setNameFilter] = useState("");
	const [statusFilter, setStatusFilter] = useState<string | undefined>();
	const [templateFilter, setTemplateFilter] = useState<string | undefined>();
	const [ownerFilter, setOwnerFilter] = useState<string | undefined>();
	// CYB-3800: batches selected for the multi-batch asset-id export.
	const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);
	const [exportBusy, setExportBusy] = useState(false);

	const templateNameById = Object.fromEntries(
		templates.map((item) => [item.id, item.name]),
	);

	const targetMap = useMemo(
		() => new Map(targets.map((t) => [t.id, t])),
		[targets],
	);

	const ownerOptions = useMemo(() => {
		const set = new Set<string>();
		for (const j of jobs) {
			if (j.createdBy) set.add(j.createdBy);
		}
		return [...set].sort().map((v) => ({ value: v, label: v }));
	}, [jobs]);

	const templateOptions = useMemo(() => {
		const used = new Set(jobs.map((j) => j.templateId));
		return templates
			.filter((t) => used.has(t.id))
			.map((t) => ({ value: t.id, label: t.name }));
	}, [jobs, templates]);

	const filteredJobs = useMemo(() => {
		const q = nameFilter.trim().toLowerCase();
		return jobs.filter((job) => {
			if (q && !job.name.toLowerCase().includes(q)) return false;
			if (statusFilter && job.status !== statusFilter) return false;
			if (templateFilter && job.templateId !== templateFilter) return false;
			if (ownerFilter && job.createdBy !== ownerFilter) return false;
			return true;
		});
	}, [jobs, nameFilter, statusFilter, templateFilter, ownerFilter]);

	const refresh = useCallback(
		async (silent = false) => {
			if (!silent) setLoading(true);
			try {
				const [jobItems, templateItems, targetItems] = await Promise.all([
					listBatchJobs(),
					listPipelines({ pageSize: 200 })
						.then((r) => r.items)
						.catch(() => []),
					listExecutionTargets().catch(() => []),
				]);
				setJobs(sortBatchJobsByCreatedDesc(jobItems));
				setTemplates(templateItems);
				setTargets(targetItems);
			} catch (err) {
				message.error(`加载批次任务失败：${String(err)}`);
			} finally {
				if (!silent) setLoading(false);
			}
		},
		[message],
	);

	useEffect(() => {
		if (active) {
			void refresh();
		}
	}, [active, refresh]);

	// Auto-refresh when there are running/paused jobs — but only while the tab
	// is visible (CYB-3486): a backgrounded list used to poll every 10s.
	const hasActiveJobs = jobs.some(
		(j) => j.status === "running" || j.status === "paused",
	);
	useVisibleInterval(
		() => {
			void refresh(true);
		},
		10_000,
		active && hasActiveJobs,
	);

	const runAction = async (
		jobId: string,
		action: "pause" | "resume" | "retry",
		opts?: { stopRunning?: boolean },
	) => {
		setActionLoading(`${jobId}:${action}`);
		try {
			if (action === "pause")
				await pauseBatchJob(jobId, { stopRunning: opts?.stopRunning ?? false });
			if (action === "resume") await resumeBatchJob(jobId);
			if (action === "retry") await retryFailedBatchItems(jobId);
			if (action === "pause") {
				message.success(
					opts?.stopRunning
						? "已停止批次（含运行中的子任务，可恢复）"
						: "已停止下发（运行中的继续执行，可恢复）",
				);
			} else {
				message.success("操作已提交");
			}
			await refresh();
		} catch (err) {
			message.error(`操作失败：${String(err)}`);
		} finally {
			setActionLoading(null);
		}
	};

	const columns: ColumnsType<BatchJob> = [
		{
			title: "批次名称",
			dataIndex: "name",
			key: "name",
			width: 320,
			ellipsis: true,
			render: (name: string, record: BatchJob) => (
				<div style={{ minWidth: 0, maxWidth: 320 }}>
					<Text strong ellipsis={{ tooltip: name }}>
						{name}
					</Text>
					<Space size={4} style={{ display: "flex", alignItems: "center" }}>
						<Text
							type="secondary"
							style={{ fontSize: 12, fontFamily: "monospace" }}
						>
							ID: {record.id.slice(0, 12)}…
						</Text>
						<Tooltip title="复制完整 ID">
							<Button
								type="text"
								size="small"
								icon={<CopyOutlined style={{ fontSize: 12 }} />}
								onClick={(e) => {
									e.stopPropagation();
									void navigator.clipboard
										.writeText(record.id)
										.then(() => message.success("已复制批次 ID"))
										.catch(() => message.error("复制失败"));
								}}
								style={{ padding: "0 4px", height: 20 }}
							/>
						</Tooltip>
					</Space>
				</div>
			),
		},
		{
			title: "模板",
			dataIndex: "templateId",
			key: "templateId",
			width: 260,
			ellipsis: true,
			render: (templateId: string) => {
				const label = templateNameById[templateId] ?? templateId;
				return (
					<Text ellipsis={{ tooltip: label }} style={{ maxWidth: 260 }}>
						{label}
					</Text>
				);
			},
		},
		{
			title: "进度",
			key: "progress",
			width: 200,
			render: (_: unknown, record: BatchJob) => (
				<BatchProgressCell job={record} />
			),
		},
		{
			title: "状态",
			dataIndex: "status",
			key: "status",
			width: 110,
			render: (_status: string, record: BatchJob) => {
				// CYB-4012: derive the tag from item counts, not the raw backend
				// `status` — the backend rollup can report "completed" for an
				// all-failed batch (green "已完成" hiding a 100% failure) and is
				// non-monotonic. "部分失败" (mixed) keeps its light-red informational
				// outline; all-failed (0 completed) now resolves to the plain "失败"
				// tag so operators can distinguish "some worked" from "nothing worked".
				const derived = deriveBatchJobStatus(record);
				if (derived === "partial_failure") {
					return (
						<Tag
							color="warning"
							style={{
								background: "#fff2e8",
								borderColor: "#ffbb96",
								color: "#d4380d",
							}}
						>
							部分失败
						</Tag>
					);
				}
				return (
					<Tag color={resolveStatusTagColor(derived)}>
						{formatBatchJobStatus(derived)}
					</Tag>
				);
			},
		},
		{
			title: "优先级",
			key: "priority",
			width: 90,
			render: (_: unknown, record: BatchJob) => {
				const tid = batchTargetId(record.filterJson);
				const poolDefault = tid
					? targetMap.get(tid)?.resourceDefaults?.priority
					: undefined;
				const badge = priorityBadge(
					effectivePriorityFromFilter(record.filterJson, poolDefault),
				);
				// "普通" is the default and shouldn't compete for attention — render
				// it as a low-contrast outlined chip; "高 / 低" keep their colored
				// tags so they stand out against the dominant "普通" row noise.
				if (badge.label === "普通") {
					return (
						<Tag
							style={{
								background: "#fafafa",
								borderColor: "#d9d9d9",
								color: "#8c8c8c",
							}}
						>
							{badge.label}
						</Tag>
					);
				}
				return <Tag color={badge.color}>{badge.label}</Tag>;
			},
		},
		{
			title: "创建时间",
			dataIndex: "createdAt",
			key: "createdAt",
			width: 180,
			defaultSortOrder: "descend",
			sortDirections: ["descend", "ascend", "descend"],
			sorter: (a, b) => batchJobCreatedAtMs(a) - batchJobCreatedAtMs(b),
			render: (value: string) => dayjs(value).format("YYYY-MM-DD HH:mm:ss"),
		},
		{
			title: "完成时间",
			dataIndex: "finishedAt",
			key: "finishedAt",
			width: 180,
			render: (_: unknown, record: BatchJob) => {
				const finished = batchJobCompletionAt(record);
				return finished ? dayjs(finished).format("YYYY-MM-DD HH:mm:ss") : "—";
			},
		},
		{
			title: (
				<Tooltip title="运行耗时:首个子任务开始 → 最后一个子任务完成,不含提交排队与暂停等待">
					<span>耗时</span>
				</Tooltip>
			),
			key: "duration",
			width: 100,
			render: (_: unknown, record: BatchJob) => {
				const secs = batchJobRunDurationSeconds(record);
				return secs === null ? "—" : formatDurationSeconds(secs);
			},
		},
		{
			title: (
				<Tooltip title="总时长:该批次所有子任务的资源总时长,来自 assets.duration_ms 求和(排除软删除资产;重复出现按次计入)">
					<span>总时长</span>
				</Tooltip>
			),
			key: "totalDurationMs",
			dataIndex: "totalDurationMs",
			width: 110,
			sorter: (a: BatchJob, b: BatchJob) =>
				(a.totalDurationMs ?? 0) - (b.totalDurationMs ?? 0),
			render: (_: unknown, record: BatchJob) => {
				const ms = record.totalDurationMs ?? 0;
				return ms > 0 ? formatDurationMs(ms) : "—";
			},
		},
		{
			title: "所属用户",
			dataIndex: "createdBy",
			key: "createdBy",
			width: 160,
			ellipsis: true,
			render: (value?: string) => value || "—",
		},
		{
			title: "操作",
			key: "actions",
			width: 220,
			render: (_: unknown, record: BatchJob) => (
				<Space size={4} onClick={(event) => event.stopPropagation()}>
					<Button
						type="link"
						size="small"
						onClick={() =>
							navigate(`/pipeline/batch/${record.id}`, {
								state: batchJobDetailLocationState(),
							})
						}
					>
						查看子任务
					</Button>
					{record.status === "running" ? (
						<Dropdown
							trigger={["click"]}
							menu={{
								items: [
									{ key: "soft", label: "停止下发" },
									{ key: "hard", label: "全部停止", danger: true },
								],
								onClick: ({ key }) => {
									// 软停无害(可恢复、不停在飞的)→ 直接执行;
									// 硬停会停掉运行中的子任务 → 二次确认(与详情页一致)。
									if (key === "hard") {
										modal.confirm({
											title: "全部停止该批次？",
											content:
												"会对运行中的子任务发送 Argo 优雅停止信号（非删除），这些子任务恢复时从头重新下发。可稍后点「继续」恢复。",
											okText: "确认停止",
											okButtonProps: { danger: true },
											cancelText: "取消",
											onOk: () =>
												runAction(record.id, "pause", { stopRunning: true }),
										});
									} else {
										void runAction(record.id, "pause", {
											stopRunning: false,
										});
									}
								},
							}}
						>
							<Button
								type="link"
								size="small"
								danger
								icon={<PoweroffOutlined />}
								loading={actionLoading === `${record.id}:pause`}
							>
								停止
							</Button>
						</Dropdown>
					) : null}
					{record.status === "paused" ? (
						<Button
							type="link"
							size="small"
							icon={<PlayCircleOutlined />}
							loading={actionLoading === `${record.id}:resume`}
							onClick={() => void runAction(record.id, "resume")}
						>
							继续
						</Button>
					) : null}
					{record.failedCount > 0 ? (
						<Button
							type="link"
							size="small"
							icon={<RedoOutlined />}
							loading={actionLoading === `${record.id}:retry`}
							onClick={() => void runAction(record.id, "retry")}
						>
							重试失败
						</Button>
					) : null}
				</Space>
			),
		},
	];

	return (
		<div className="pipeline-batch-list">
			<div
				style={{
					display: "flex",
					alignItems: "center",
					gap: 12,
					marginBottom: 12,
					flexWrap: "wrap",
				}}
			>
				<Title level={4} style={{ margin: 0 }}>
					批量任务
				</Title>
				<Input.Search
					allowClear
					placeholder="搜索批次名称"
					value={nameFilter}
					onChange={(e) => setNameFilter(e.target.value)}
					style={{ width: 220 }}
					data-testid="batch-job-name-search"
				/>
				<Select
					allowClear
					placeholder="状态"
					value={statusFilter}
					onChange={(value) => setStatusFilter(value)}
					style={{ width: 120 }}
					data-testid="batch-job-status-filter"
					options={[
						{ value: "running", label: "运行中" },
						{ value: "paused", label: "已暂停" },
						{ value: "completed", label: "已完成" },
						{ value: "failed", label: "失败" },
					]}
				/>
				<Select
					allowClear
					showSearch
					optionFilterProp="label"
					placeholder="模板"
					value={templateFilter}
					onChange={(value) => setTemplateFilter(value)}
					style={{ width: 180 }}
					data-testid="batch-job-template-filter"
					options={templateOptions}
				/>
				<Select
					allowClear
					showSearch
					optionFilterProp="label"
					placeholder="所属用户"
					value={ownerFilter}
					onChange={(value) => setOwnerFilter(value)}
					style={{ width: 160 }}
					data-testid="batch-job-owner-filter"
					options={ownerOptions}
				/>
				<Button
					icon={<ReloadOutlined />}
					onClick={() => void refresh()}
					loading={loading}
				>
					刷新
				</Button>
				<Dropdown
					disabled={selectedRowKeys.length === 0 || exportBusy}
					menu={{
						// CYB-3821: each action gains a status sub-menu so the user can
						// narrow the export to succeeded / failed child runs. Key format
						// `${action}:${filter}` keeps the click handler flat.
						items: [
							{
								key: "copy",
								label: "复制到剪贴板",
								children: [
									{ key: "copy:all", label: "全部" },
									{ key: "copy:succeeded", label: "仅成功" },
									{ key: "copy:failed", label: "仅失败" },
								],
							},
							{
								key: "csv",
								label: "导出 CSV",
								children: [
									{ key: "csv:all", label: "全部" },
									{ key: "csv:succeeded", label: "仅成功" },
									{ key: "csv:failed", label: "仅失败" },
								],
							},
						],
						onClick: async ({ key }) => {
							if (selectedRowKeys.length === 0) return;
							const [action, filterKey] = key.split(":") as [
								"copy" | "csv",
								BatchExportStatusFilter,
							];
							const filterLabel =
								filterKey === "succeeded"
									? "成功"
									: filterKey === "failed"
										? "失败"
										: "全部";
							setExportBusy(true);
							// CYB-3822: progress toast — start at 0/N and rebuild the
							// message each time a batch's fetch completes so a large
							// selection (25k+ children) does not look like a hang. The
							// server-side status filter (statusFilterServerValue) means
							// the payloads themselves are already 5-10× smaller for
							// "仅成功 / 仅失败".
							const total = selectedRowKeys.length;
							let hide = message.loading(
								`正在拉取 0/${total} 个批次的${filterLabel}资产 ID…`,
								0,
							);
							try {
								const ids = await fetchAssetIdsForBatches(selectedRowKeys, {
									status: statusFilterServerValue(filterKey),
									// CYB-3821a: keep the client predicate as a safety net
									// in case a legacy backend ignores ?status=.
									filterFn: statusFilterPredicate(filterKey),
									onBatchDone: (done, tot) => {
										hide();
										hide = message.loading(
											`正在拉取 ${done}/${tot} 个批次的${filterLabel}资产 ID…`,
											0,
										);
									},
								});
								hide();
								if (ids.length === 0) {
									message.info(
										filterKey === "all"
											? "所选批次没有可导出的资产 ID"
											: `所选批次没有${filterLabel}状态的子任务`,
									);
									return;
								}
								if (action === "copy") {
									const ok = await copyAssetIdsToClipboard(ids);
									if (ok) {
										message.success(
											`已复制 ${ids.length} 个${filterLabel}资产 ID(来自 ${selectedRowKeys.length} 个批次)到剪贴板`,
										);
									} else {
										message.error("复制失败，请重试");
									}
								} else if (action === "csv") {
									const filterSuffix =
										filterKey === "all" ? "" : `-${filterKey}`;
									exportAssetIdsCsv(
										ids,
										`batch-multi-${selectedRowKeys.length}${filterSuffix}`,
									);
									message.success(
										`已导出 ${ids.length} 个${filterLabel}资产 ID(来自 ${selectedRowKeys.length} 个批次)`,
									);
								}
							} catch (err) {
								hide();
								message.error(`拉取资产 ID 失败:${String(err)}`);
							} finally {
								setExportBusy(false);
							}
						},
					}}
				>
					<Button
						icon={<DownloadOutlined />}
						disabled={selectedRowKeys.length === 0 || exportBusy}
						loading={exportBusy}
						data-testid="batch-job-bulk-export"
					>
						{selectedRowKeys.length > 0
							? `批量导出资产 ID (${selectedRowKeys.length})`
							: "批量导出资产 ID"}
					</Button>
				</Dropdown>
			</div>

			{loading && jobs.length === 0 ? (
				<Skeleton active paragraph={{ rows: 6 }} />
			) : jobs.length === 0 ? (
				<Empty description="暂无批量任务。在「运行流水线」弹窗中选择 2 个及以上资产后会自动创建。" />
			) : (
				<Table
					rowKey="id"
					loading={loading}
					columns={columns}
					dataSource={filteredJobs}
					scroll={{ x: 1270 }}
					pagination={{ pageSize: 20, showSizeChanger: true }}
					// CYB-3800: multi-select drives the bulk export button above.
					rowSelection={{
						selectedRowKeys,
						onChange: (keys) => setSelectedRowKeys(keys as string[]),
					}}
					onRow={(record) => ({
						// The row-level click drills into the batch detail. The
						// checkbox is rendered outside the row's onClick target so
						// selecting doesn't navigate; explicit action buttons in
						// the operation column stop propagation themselves.
						onClick: () =>
							navigate(`/pipeline/batch/${record.id}`, {
								state: batchJobDetailLocationState(),
							}),
						style: { cursor: "pointer" },
					})}
				/>
			)}
		</div>
	);
}
