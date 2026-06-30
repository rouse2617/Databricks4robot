import {
	PauseCircleOutlined,
	PlayCircleOutlined,
	RedoOutlined,
	ReloadOutlined,
} from "@ant-design/icons";
import {
	App,
	Button,
	Empty,
	Input,
	Progress,
	Select,
	Skeleton,
	Space,
	Table,
	Tag,
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
import {
	batchJobCreatedAtMs,
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
	const { message } = App.useApp();
	const navigate = useNavigate();
	const [jobs, setJobs] = useState<BatchJob[]>([]);
	const [templates, setTemplates] = useState<PipelineTemplate[]>([]);
	const [loading, setLoading] = useState(false);
	const [actionLoading, setActionLoading] = useState<string | null>(null);
	const [nameFilter, setNameFilter] = useState("");
	const [statusFilter, setStatusFilter] = useState<string | undefined>();

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

	// Auto-refresh when there are running/paused jobs
	useEffect(() => {
		if (!active) return;
		const hasActive = jobs.some(
			(j) => j.status === "running" || j.status === "paused",
		);
		if (!hasActive) return;
		const id = setInterval(() => {
			void refresh(true);
		}, 10_000);
		return () => clearInterval(id);
	}, [active, jobs, refresh]);

	const runAction = async (
		jobId: string,
		action: "pause" | "resume" | "retry",
	) => {
		setActionLoading(`${jobId}:${action}`);
		try {
			if (action === "pause") await pauseBatchJob(jobId);
			if (action === "resume") await resumeBatchJob(jobId);
			if (action === "retry") await retryFailedBatchItems(jobId);
			message.success("操作已提交");
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
						<Button
							type="link"
							size="small"
							icon={<PauseCircleOutlined />}
							loading={actionLoading === `${record.id}:pause`}
							onClick={() => void runAction(record.id, "pause")}
						>
							暂停
						</Button>
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
					onRow={(record) => ({
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
