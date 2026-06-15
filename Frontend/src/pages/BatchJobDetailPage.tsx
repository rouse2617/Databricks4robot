import {
	ArrowLeftOutlined,
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
	Skeleton,
	Space,
	Table,
	Tag,
	Typography,
} from "antd";
import dayjs from "dayjs";
import { useCallback, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
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
	retryFailedBatchItems,
} from "../api/batchJobApi";
import { listPipelines, type PipelineTemplate } from "../api/pipelineApi";
import type { WorkflowSummary } from "../api/workflowApi";
import { BackfillResultsUpload } from "../components/backfill/BackfillResultsUpload";
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
	const [nodeSummary, setNodeSummary] = useState<BatchNodeSummary | null>(null);
	const [drawerNode, setDrawerNode] = useState<BatchNodeSummaryNode | null>(
		null,
	);
	const [nodeFailures, setNodeFailures] = useState<BatchNodeFailureItem[]>([]);
	const [nodeFailureTotal, setNodeFailureTotal] = useState(0);
	const [nodeFailureLoading, setNodeFailureLoading] = useState(false);
	const [selectedRuns, setSelectedRuns] = useState<WorkflowSummary[]>([]);

	const refresh = useCallback(async () => {
		if (!id) return;
		setLoading(true);
		try {
			const [jobData, templates] = await Promise.all([
				getBatchJob(id),
				listPipelines({ pageSize: 200 })
					.then((r) => r.items)
					.catch(() => [] as PipelineTemplate[]),
			]);
			setJob(jobData);
			getBatchNodeSummary(id)
				.then(setNodeSummary)
				.catch(() => setNodeSummary(null));
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

	const runRerun = async (
		scope: "failed" | "incomplete" | "completed" | "custom" | "node_failed",
		extra: Record<string, unknown> = {},
	) => {
		if (!job) return;
		setActionLoading(`rerun-${scope}`);
		try {
			const dryRun = await rerunBatchJob(job.id, {
				scope,
				templateVersion: job.templateVersion,
				dryRun: true,
				...extra,
			});
			await new Promise<void>((resolve, reject) => {
				Modal.confirm({
					title: "确认重跑",
					content: `将重新提交 ${dryRun.matchedCount} 条子任务，模板版本 v${dryRun.templateVersion ?? "当前"}。旧 run 记录会保留。`,
					okText: "确认重跑",
					cancelText: "取消",
					onOk: async () => {
						try {
							await rerunBatchJob(job.id, {
								scope,
								templateVersion: job.templateVersion,
								...extra,
							});
							resolve();
						} catch (err) {
							reject(err);
						}
					},
					onCancel: () => resolve(),
				});
			});
			message.success("重跑已提交");
			await refresh();
		} catch (err) {
			message.error(`重跑失败：${String(err)}`);
		} finally {
			setActionLoading(null);
		}
	};

	const loadNodeFailures = useCallback(
		async (node: BatchNodeSummaryNode) => {
			if (!id) return;
			setNodeFailureLoading(true);
			try {
				const result = await listBatchNodeFailures(id, {
					pipelineNodeId: node.pipelineNodeId,
					page: 1,
					pageSize: 20,
				});
				setNodeFailures(result.items);
				setNodeFailureTotal(result.total);
			} catch (err) {
				message.error(`加载节点明细失败：${String(err)}`);
			} finally {
				setNodeFailureLoading(false);
			}
		},
		[id, message],
	);

	const selectedAssetIds = selectedRuns
		.map((item) => item.labels?.asset_id ?? item.labels?.assetId ?? "")
		.filter(Boolean);

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
										void runRerun("custom", { assetIds: selectedAssetIds });
										return;
									}
									void runRerun(key as "failed" | "incomplete" | "completed");
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
										await refresh();
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
							children: <Tag>{job.status}</Tag>,
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

			<Card title="回填结果上传" style={{ marginBottom: 16 }}>
				<BackfillResultsUpload
					defaultReportId={
						typeof job.filterJson?.expectedReportId === "string"
							? job.filterJson.expectedReportId
							: undefined
					}
					defaultVersion={
						typeof job.filterJson?.expectedReportVersion === "string"
							? job.filterJson.expectedReportVersion
							: undefined
					}
					onUploaded={() => {
						void refresh();
					}}
				/>
			</Card>

			<Card title="节点概览" style={{ marginBottom: 16 }}>
				{nodeSummary && !nodeSummary.dataCoverage.complete ? (
					<Alert
						type="info"
						showIcon
						message="节点状态仍在同步中，概览会随刷新更新"
						style={{ marginBottom: 12 }}
					/>
				) : null}
				<Table
					size="small"
					rowKey="pipelineNodeId"
					dataSource={nodeSummary?.nodes ?? []}
					locale={{ emptyText: "暂无节点账本数据，请稍后刷新" }}
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
										onClick={() => {
											setDrawerNode(record);
											void loadNodeFailures(record);
										}}
									>
										{failed} 失败
									</Button>
								) : (
									0
								);
							},
						},
						{ title: "运行中", dataIndex: ["counts", "Running"] },
						{ title: "未开始", dataIndex: ["counts", "Pending"] },
						{
							title: "失败率",
							render: (_, record) =>
								`${(record.failureRate * 100).toFixed(2)}%`,
						},
					]}
				/>
			</Card>

			<WorkflowExecutionList
				active
				batchJobId={job.id}
				_batchListKey={job.updatedAt}
				embedded
				onSelectionChange={setSelectedRuns}
				title="子任务执行记录"
			/>

			<Drawer
				title={drawerNode ? `${drawerNode.displayName} 失败资产` : "节点明细"}
				open={Boolean(drawerNode)}
				width={720}
				onClose={() => setDrawerNode(null)}
				extra={
					drawerNode ? (
						<Button
							icon={<RedoOutlined />}
							loading={actionLoading === "rerun-node_failed"}
							onClick={() =>
								void runRerun("node_failed", {
									pipelineNodeId: drawerNode.pipelineNodeId,
								})
							}
						>
							重试这 {nodeFailureTotal} 条
						</Button>
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
						{ title: "状态", dataIndex: "status" },
						{ title: "消息", dataIndex: "message", ellipsis: true },
						{
							title: "Workflow",
							render: (_, record) => (
								<Button
									type="link"
									size="small"
									onClick={() =>
										navigate(`/pipeline/executions/${record.workflowName}`)
									}
								>
									查看
								</Button>
							),
						},
					]}
				/>
			</Drawer>
		</div>
	);
}
