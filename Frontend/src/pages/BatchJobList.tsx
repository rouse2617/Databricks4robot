import {
	PauseCircleOutlined,
	PlayCircleOutlined,
	PlusOutlined,
	ReloadOutlined,
	RedoOutlined,
} from "@ant-design/icons";
import {
	App,
	Button,
	Empty,
	Progress,
	Skeleton,
	Space,
	Table,
	Tag,
	Typography,
} from "antd";
import dayjs from "dayjs";
import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
	batchJobProgress,
	type BatchJob,
	listBatchJobs,
	pauseBatchJob,
	resumeBatchJob,
	retryFailedBatchItems,
} from "../api/batchJobApi";
import { listPipelines, type PipelineTemplate } from "../api/pipelineApi";
import { STATUS_COLORS } from "../lib/constants";
import { CreateBatchJobModal } from "../components/pipeline/CreateBatchJobModal";

const { Title, Text } = Typography;

const BATCH_STATUS_COLORS: Record<string, string> = {
	running: "processing",
	paused: "warning",
	completed: "success",
	failed: "error",
};

interface BatchJobListProps {
	active?: boolean;
}

export function BatchJobList({ active = true }: BatchJobListProps) {
	const { message } = App.useApp();
	const navigate = useNavigate();
	const [jobs, setJobs] = useState<BatchJob[]>([]);
	const [templates, setTemplates] = useState<PipelineTemplate[]>([]);
	const [loading, setLoading] = useState(false);
	const [createOpen, setCreateOpen] = useState(false);
	const [actionLoading, setActionLoading] = useState<string | null>(null);

	const templateNameById = Object.fromEntries(
		templates.map((item) => [item.id, item.name]),
	);

	const refresh = useCallback(async () => {
		setLoading(true);
		try {
			const [jobItems, templateItems] = await Promise.all([
				listBatchJobs(),
				listPipelines().catch(() => []),
			]);
			setJobs(jobItems);
			setTemplates(templateItems);
		} catch (err) {
			message.error(`加载批次任务失败：${String(err)}`);
		} finally {
			setLoading(false);
		}
	}, [message]);

	useEffect(() => {
		if (active) {
			void refresh();
		}
	}, [active, refresh]);

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

	const columns = [
		{
			title: "批次名称",
			dataIndex: "name",
			key: "name",
			render: (name: string, record: BatchJob) => (
				<div>
					<Text strong>{name}</Text>
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
			render: (templateId: string) => templateNameById[templateId] ?? templateId,
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
						status={record.failedCount > 0 ? "exception" : "active"}
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
				<Tag color={BATCH_STATUS_COLORS[status] ?? STATUS_COLORS[status] ?? "default"}>
					{status}
				</Tag>
			),
		},
		{
			title: "创建时间",
			dataIndex: "createdAt",
			key: "createdAt",
			width: 180,
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
						onClick={() => navigate(`/pipeline/batch/${record.id}`)}
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
					marginBottom: 16,
				}}
			>
				<Title level={4} style={{ margin: 0 }}>
					批次任务
				</Title>
				<Button
					type="primary"
					icon={<PlusOutlined />}
					onClick={() => setCreateOpen(true)}
				>
					新建批次
				</Button>
				<Button icon={<ReloadOutlined />} onClick={() => void refresh()} loading={loading}>
					刷新
				</Button>
			</div>

			{loading && jobs.length === 0 ? (
				<Skeleton active paragraph={{ rows: 6 }} />
			) : jobs.length === 0 ? (
				<Empty description="暂无批次任务，可新建批次对多个资产批量跑流水线">
					<Button type="primary" onClick={() => setCreateOpen(true)}>
						新建批次
					</Button>
				</Empty>
			) : (
				<Table
					rowKey="id"
					loading={loading}
					columns={columns}
					dataSource={jobs}
					pagination={{ pageSize: 20, showSizeChanger: true }}
					onRow={(record) => ({
						onClick: () => navigate(`/pipeline/batch/${record.id}`),
						style: { cursor: "pointer" },
					})}
				/>
			)}

			<CreateBatchJobModal
				open={createOpen}
				templates={templates}
				onClose={() => setCreateOpen(false)}
				onCreated={(job) => {
					setCreateOpen(false);
					message.success(`批次「${job.name}」已创建，共 ${job.totalCount} 个子任务`);
					void refresh();
					navigate(`/pipeline/batch/${job.id}`);
				}}
			/>
		</div>
	);
}
