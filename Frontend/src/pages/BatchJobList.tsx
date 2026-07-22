import {
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
	Progress,
	Select,
	Skeleton,
	Space,
	Table,
	Tag,
	Tooltip,
	Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import dayjs from "dayjs";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
	type BatchJob,
	batchJobProgress,
	batchJobProgressStatus,
	listBatchJobs,
	pauseBatchJob,
	resumeBatchJob,
	retryFailedBatchItems,
} from "../api/batchJobApi";
import { listPipelines, type PipelineTemplate } from "../api/pipelineApi";
import { useVisibleInterval } from "../hooks/useVisibleInterval";
import {
	batchJobCompletionAt,
	batchJobCreatedAtMs,
	batchJobRunDurationSeconds,
	copyAssetIdsToClipboard,
	exportAssetIdsCsv,
	fetchAssetIdsForBatches,
	formatDurationSeconds,
	sortBatchJobsByCreatedDesc,
} from "../lib/batchJobs";
import { batchJobDetailLocationState } from "../lib/pipelineNavigation";
import {
	formatBatchJobStatus,
	resolveStatusTagColor,
} from "../lib/statusLabels";

const { Title, Text } = Typography;

interface BatchJobListProps {
	active?: boolean;
}

export function BatchJobList({ active = true }: BatchJobListProps) {
	const { message, modal } = App.useApp();
	const navigate = useNavigate();
	const [jobs, setJobs] = useState<BatchJob[]>([]);
	const [templates, setTemplates] = useState<PipelineTemplate[]>([]);
	const [loading, setLoading] = useState(false);
	const [actionLoading, setActionLoading] = useState<string | null>(null);
	const [nameFilter, setNameFilter] = useState("");
	const [statusFilter, setStatusFilter] = useState<string | undefined>();
	// CYB-3800: batches selected for the multi-batch asset-id export.
	const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);
	const [exportBusy, setExportBusy] = useState(false);

	const templateNameById = Object.fromEntries(
		templates.map((item) => [item.id, item.name]),
	);

	const filteredJobs = useMemo(() => {
		const q = nameFilter.trim().toLowerCase();
		return jobs.filter((job) => {
			if (q && !job.name.toLowerCase().includes(q)) return false;
			if (statusFilter && job.status !== statusFilter) return false;
			return true;
		});
	}, [jobs, nameFilter, statusFilter]);

	const refresh = useCallback(
		async (silent = false) => {
			if (!silent) setLoading(true);
			try {
				const [jobItems, templateItems] = await Promise.all([
					listBatchJobs(),
					listPipelines({ pageSize: 200 })
						.then((r) => r.items)
						.catch(() => []),
				]);
				setJobs(sortBatchJobsByCreatedDesc(jobItems));
				setTemplates(templateItems);
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
			width: 260,
			ellipsis: true,
			render: (name: string, record: BatchJob) => (
				<div style={{ minWidth: 0, maxWidth: 260 }}>
					<Text strong ellipsis={{ tooltip: name }}>
						{name}
					</Text>
					<Text type="secondary" style={{ display: "block", fontSize: 12 }}>
						ID: {record.id.slice(0, 8)}…
					</Text>
				</div>
			),
		},
		{
			title: "模板",
			dataIndex: "templateId",
			key: "templateId",
			width: 200,
			ellipsis: true,
			render: (templateId: string) => {
				const label = templateNameById[templateId] ?? templateId;
				return (
					<Text ellipsis={{ tooltip: label }} style={{ maxWidth: 200 }}>
						{label}
					</Text>
				);
			},
		},
		{
			title: "进度",
			key: "progress",
			width: 220,
			render: (_: unknown, record: BatchJob) => (
				<div>
					<Progress
						percent={batchJobProgress(record)}
						size="small"
						status={batchJobProgressStatus(record)}
					/>
					<Text type="secondary" style={{ fontSize: 12 }}>
						{record.completedCount} 成功 · {record.failedCount} 失败 · 共{" "}
						{record.totalCount}
					</Text>
				</div>
			),
		},
		{
			title: "状态",
			dataIndex: "status",
			key: "status",
			width: 110,
			render: (status: string) => (
				<Tag color={resolveStatusTagColor(status)}>
					{formatBatchJobStatus(status)}
				</Tag>
			),
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
						items: [
							{ key: "copy", label: "复制到剪贴板" },
							{ key: "csv", label: "导出 CSV" },
						],
						onClick: async ({ key }) => {
							if (selectedRowKeys.length === 0) return;
							setExportBusy(true);
							const hide = message.loading(
								`正在拉取 ${selectedRowKeys.length} 个批次的资产 ID…`,
								0,
							);
							try {
								const ids = await fetchAssetIdsForBatches(selectedRowKeys);
								hide();
								if (ids.length === 0) {
									message.info("所选批次没有可导出的资产 ID");
									return;
								}
								if (key === "copy") {
									const ok = await copyAssetIdsToClipboard(ids);
									if (ok) {
										message.success(
											`已复制 ${ids.length} 个资产 ID(来自 ${selectedRowKeys.length} 个批次)到剪贴板`,
										);
									} else {
										message.error("复制失败，请重试");
									}
								} else if (key === "csv") {
									exportAssetIdsCsv(
										ids,
										`batch-multi-${selectedRowKeys.length}`,
									);
									message.success(
										`已导出 ${ids.length} 个资产 ID(来自 ${selectedRowKeys.length} 个批次)`,
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
					scroll={{ x: 1190 }}
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
