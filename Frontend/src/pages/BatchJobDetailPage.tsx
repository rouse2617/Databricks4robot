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
	Select,
	Skeleton,
	Space,
	Table,
	Tag,
	Typography,
} from "antd";
import dayjs from "dayjs";
import { useCallback, useEffect, useState } from "react";
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
	listPipelines,
	type PipelineTemplate,
} from "../api/pipelineApi";
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

type RerunScope = "failed" | "incomplete" | "completed" | "custom" | "node_failed";

interface RerunModalState {
	scope: RerunScope;
	extra: Record<string, unknown>;
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

export default function BatchJobDetailPage() {
	const { id = "" } = useParams();
	const navigate = useNavigate();
	const location = useLocation();
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

	const refresh = useCallback(async (opts?: { silent?: boolean }) => {
		if (!id) return;
		if (!opts?.silent) {
			setLoading(true);
		}
		try {
			const jobData = await getBatchJob(id);
			const [templates, versions] = await Promise.all([
				listPipelines({ pageSize: 200 })
					.then((r) => r.items)
					.catch(() => [] as PipelineTemplate[]),
				listPipelineVersions(jobData.templateId).catch(
					() => [] as PipelineTemplate[],
				),
			]);
			setJob(jobData);
			setTemplateVersions(versions);
			getBatchNodeSummary(id)
				.then(setNodeSummary)
				.catch(() => setNodeSummary(null));
			const template = templates.find((item) => item.id === jobData.templateId);
			setTemplateName(template?.name ?? jobData.templateId);
		} catch (err) {
			if (!opts?.silent) {
				message.error(`加载批次详情失败：${String(err)}`);
			}
		} finally {
			if (!opts?.silent) {
				setLoading(false);
			}
		}
	}, [id, message]);

	useEffect(() => {
		void refresh();
	}, [refresh]);

	useEffect(() => {
		if (!job || !["running", "paused"].includes(job.status)) {
			return;
		}
		const timer = window.setInterval(() => {
			void refresh({ silent: true });
		}, 5000);
		return () => window.clearInterval(timer);
	}, [job?.id, job?.status, refresh]);

	const backToBatchList = useCallback(() => {
		goBackFromBatchJobDetail(navigate, location.state);
	}, [location.state, navigate]);

	const runAction = async (action: "pause" | "resume") => {
		if (!job) return;
		setActionLoading(action);
		try {
			if (action === "pause") await pauseBatchJob(job.id);
			if (action === "resume") await resumeBatchJob(job.id);
			message.success("操作已提交");
			await refresh();
		} catch (err) {
			message.error(`操作失败：${String(err)}`);
		} finally {
			setActionLoading(null);
		}
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
				message.error(`预览重跑失败：${String(err)}`);
			} finally {
				setRerunPreviewLoading(false);
			}
		},
		[job, message],
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
			await rerunBatchJob(job.id, {
				scope: rerunModal.scope,
				templateVersion: rerunTemplateVersion,
				...rerunModal.extra,
			});
			message.success("重跑已提交");
			closeRerunModal();
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
									openRerunModal(
										key as "failed" | "incomplete" | "completed",
									);
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
						<Space>
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
							title: "Workflow",
							render: (_, record) => (
								<Button
									type="link"
									size="small"
									onClick={() =>
										navigate(`/pipeline/executions/${record.workflowName}`, {
											state: workflowDetailLocationState(job.id),
										})
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
		</div>
	);
}
