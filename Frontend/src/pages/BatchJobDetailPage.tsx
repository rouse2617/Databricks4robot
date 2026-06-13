import {
	ArrowLeftOutlined,
	PauseCircleOutlined,
	PlayCircleOutlined,
	RedoOutlined,
	ReloadOutlined,
} from "@ant-design/icons";
import {
	App,
	Button,
	Card,
	Descriptions,
	Progress,
	Skeleton,
	Space,
	Tag,
	Typography,
} from "antd";
import dayjs from "dayjs";
import { useCallback, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import {
	batchJobProgress,
	type BatchJob,
	getBatchJob,
	pauseBatchJob,
	resumeBatchJob,
	retryFailedBatchItems,
} from "../api/batchJobApi";
import { listPipelines, type PipelineTemplate } from "../api/pipelineApi";
import { WorkflowExecutionList } from "./WorkflowExecutionList";

const { Title, Text } = Typography;

export default function BatchJobDetailPage() {
	const { id = "" } = useParams();
	const navigate = useNavigate();
	const { message } = App.useApp();
	const [job, setJob] = useState<BatchJob | null>(null);
	const [templateName, setTemplateName] = useState("");
	const [loading, setLoading] = useState(true);
	const [actionLoading, setActionLoading] = useState<string | null>(null);

	const refresh = useCallback(async () => {
		if (!id) return;
		setLoading(true);
		try {
			const [jobData, templates] = await Promise.all([
				getBatchJob(id),
				listPipelines({ pageSize: 200 }).then((r) => r.items).catch(() => [] as PipelineTemplate[]),
			]);
			setJob(jobData);
			const template = templates.find((item) => item.id === jobData.templateId);
			setTemplateName(template?.name ?? jobData.templateId);
		} catch (err) {
			message.error(`加载批次详情失败：${String(err)}`);
		} finally {
			setLoading(false);
		}
	}, [id, message]);

	useEffect(() => {
		void refresh();
	}, [refresh]);

	const runAction = async (action: "pause" | "resume" | "retry") => {
		if (!job) return;
		setActionLoading(action);
		try {
			if (action === "pause") await pauseBatchJob(job.id);
			if (action === "resume") await resumeBatchJob(job.id);
			if (action === "retry") await retryFailedBatchItems(job.id);
			message.success("操作已提交");
			await refresh();
		} catch (err) {
			message.error(`操作失败：${String(err)}`);
		} finally {
			setActionLoading(null);
		}
	};

	if (loading && !job) {
		return <Skeleton active paragraph={{ rows: 8 }} />;
	}

	if (!job) {
		return (
			<div style={{ padding: 24 }}>
				<Button
					icon={<ArrowLeftOutlined />}
					onClick={() =>
						navigate("/pipeline?tab=executions&executionView=batch")
					}
				>
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
				<Button
					icon={<ArrowLeftOutlined />}
					onClick={() =>
						navigate("/pipeline?tab=executions&executionView=batch")
					}
				>
					返回批量任务
				</Button>
				<Button icon={<ReloadOutlined />} onClick={() => void refresh()}>
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
								onClick={() => void runAction("pause")}
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
								loading={actionLoading === "retry"}
								onClick={() => void runAction("retry")}
							>
								重试失败项
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
							children: <Tag>{job.status}</Tag>,
						},
						{
							label: "子任务数",
							children: job.totalCount,
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

			<WorkflowExecutionList
				active
				batchJobId={job.id}
				batchListKey={job.updatedAt}
				embedded
				title="子任务执行记录"
			/>
		</div>
	);
}
