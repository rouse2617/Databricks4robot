import {
	DeleteOutlined,
	LinkOutlined,
	DownOutlined,
	EditOutlined,
	EyeOutlined,
	PlayCircleOutlined,
	ReloadOutlined,
} from "@ant-design/icons";
import { Alert, Button, Dropdown, Modal, message, Space, Tag } from "antd";
import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
	type Deployment,
	deleteDeployment,
	deletePipeline,
	deployTemplate,
	getPipeline,
	listDeployments,
	listPipelines,
	type PipelineTemplate,
} from "../../api/pipelineApi";
import AssetPicker from "./AssetPicker";
import type { Pipeline } from "./types";

const STATUS_COLORS: Record<string, string> = {
	Succeeded: "success",
	Running: "processing",
	Pending: "warning",
	Failed: "error",
	Error: "error",
};

function toStringArray(value: unknown): string[] {
	if (typeof value === "string") return [value].filter(Boolean);
	if (!Array.isArray(value)) return [];
	return value
		.map((item) => (typeof item === "string" ? item.trim() : ""))
		.filter((item) => item.length > 0);
}

function extractPipelineAssetIds(pipelineJSON: Pipeline | undefined): string[] {
	const contract = (pipelineJSON ?? {}) as Record<string, unknown>;
	return [
		...toStringArray(contract._input_asset_ids),
		...toStringArray(contract.input_asset_ids),
		...toStringArray(
			(contract.input as Record<string, unknown> | undefined)?.asset_ids,
		),
		...toStringArray(
			(contract.assetSelection as Record<string, unknown> | undefined)
				?.assetIds,
		),
	]
		.filter(Boolean)
		.filter((value, index, values) => values.indexOf(value) === index)
		.sort();
}

export function DeployPanel({
	onEditTemplate,
	compact,
}: {
	onEditTemplate?: (pipeline: Pipeline) => void;
	compact?: boolean;
}) {
	const navigate = useNavigate();
	const [templates, setTemplates] = useState<PipelineTemplate[]>([]);
	const [deployments, setDeployments] = useState<Deployment[]>([]);
	const [loading, setLoading] = useState(false);

	// Asset selection modal state
	const [assetModalOpen, setAssetModalOpen] = useState(false);
	const [deployTargetId, setDeployTargetId] = useState<string | null>(null);
	const [selectedAssetIds, setSelectedAssetIds] = useState<string[]>([]);
	const [deploying, setDeploying] = useState(false);

	const refresh = useCallback(async () => {
		setLoading(true);
		try {
			const [d, t] = await Promise.all([listDeployments(), listPipelines()]);
			setDeployments(d);
			setTemplates(t);
		} catch {
			/* server not available */
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		refresh();
	}, [refresh]);

	const handleDeployClick = (templateId: string) => {
		setDeployTargetId(templateId);
		setSelectedAssetIds([]);
		setAssetModalOpen(true);
	};

	const handleDirectRun = async (templateId: string) => {
		try {
			await deployTemplate(templateId);
			message.success("部署成功");
			refresh();
		} catch (err) {
			message.error(`部署失败: ${String(err)}`);
		}
	};

	const handleDeployConfirm = async () => {
		if (!deployTargetId) return;
		setDeploying(true);
		try {
			await deployTemplate(
				deployTargetId,
				selectedAssetIds.length > 0 ? selectedAssetIds : undefined,
			);
			message.success("部署成功");
			setAssetModalOpen(false);
			refresh();
		} catch (err) {
			message.error(`部署失败: ${String(err)}`);
		} finally {
			setDeploying(false);
		}
	};

	const handleDeleteDeployment = async (id: string) => {
		try {
			await deleteDeployment(id);
			message.success("已删除部署记录");
			refresh();
		} catch (err) {
			message.error(`删除失败: ${String(err)}`);
		}
	};

	const handleDeleteTemplate = async (id: string) => {
		try {
			await deletePipeline(id);
			message.success("已删除流水线模板");
			refresh();
		} catch (err) {
			message.error(`删除失败: ${String(err)}`);
		}
	};

	const handleEditTemplate = async (id: string) => {
		try {
			const t = await getPipeline(id);
			if (onEditTemplate) {
				onEditTemplate(t.pipeline);
			} else {
				sessionStorage.setItem("pipeline-edit", JSON.stringify(t.pipeline));
				navigate("/pipeline");
			}
		} catch (err) {
			message.error(`加载模板失败: ${String(err)}`);
		}
	};

	if (compact) {
		const recentDeployments = deployments.slice(0, 3);
		return (
			<div className="deploy-panel">
				<div
					style={{
						display: "flex",
						justifyContent: "space-between",
						alignItems: "center",
						marginBottom: 12,
					}}
				>
					<div className="deploy-section-title">最近部署</div>
					<a
						href="/workflows"
						style={{
							color: "#2563eb",
							fontSize: 12,
							textDecoration: "underline",
						}}
					>
						查看全部
					</a>
				</div>

				<div className="deploy-section">
					{recentDeployments.length === 0 ? (
						<div className="dep-empty">暂无部署记录</div>
					) : (
						recentDeployments.map((d) => (
							<div key={d.id} className="dep-card">
								<div className="dep-card-info">
									<div className="dep-card-name">{d.pipelineName}</div>
									<div className="dep-card-meta">
										<Tag color={STATUS_COLORS[d.status] || "default"}>
											{d.status}
										</Tag>
									</div>
								</div>
								<Button
									size="small"
									icon={<LinkOutlined />}
									onClick={() => navigate(`/workflows/${d.workflowName}`)}
								>
									查看
								</Button>
							</div>
						))
					)}
				</div>

				<div className="deploy-section-title" style={{ marginTop: 18 }}>
					模板
				</div>
				<div className="deploy-section">
					{templates.length === 0 ? (
						<div className="dep-empty">暂无已保存的流水线模板</div>
					) : (
						templates.map((t) => (
							<div key={t.id} className="dep-card">
								<div className="dep-card-info">
									<div className="dep-card-name">{t.name}</div>
								</div>
								<Button
									size="small"
									icon={<EditOutlined />}
									onClick={() => handleEditTemplate(t.id)}
								>
									加载到画布
								</Button>
							</div>
						))
					)}
				</div>
			</div>
		);
	}

	return (
		<div className="deploy-panel">
			<div
				style={{
					display: "flex",
					alignItems: "center",
					gap: 12,
					marginBottom: 16,
				}}
			>
				<h3>部署记录</h3>
				<Button
					size="small"
					icon={<ReloadOutlined />}
					onClick={refresh}
					loading={loading}
				>
					刷新
				</Button>
			</div>

			<div className="deploy-section-title">
				已保存的流水线
				<span className="count">{templates.length}</span>
			</div>
			<div className="deploy-section">
				{templates.length === 0 ? (
					<div className="dep-empty">暂无已保存的流水线模板</div>
				) : (
					templates.map((t) => (
						<div key={t.id} className="dep-card">
							<div className="dep-card-info">
								<div className="dep-card-name">{t.name}</div>
								<div className="dep-card-meta">
									<span>{t.nodeCount} 个节点</span>
									<span className="dot">•</span>
									<span>{new Date(t.createdAt).toLocaleString()}</span>
								</div>
							</div>
							<div className="deploy-btn-list">
								<Space.Compact>
									<Button
										size="small"
										type="primary"
										icon={<PlayCircleOutlined />}
										onClick={() => handleDirectRun(t.id)}
									>
										运行
									</Button>
									<Dropdown
										menu={{
											items: [
												{
													key: "assets",
													label: "选择资产运行",
													onClick: () => handleDeployClick(t.id),
												},
											],
										}}
										trigger={["click"]}
									>
										<Button
											size="small"
											type="primary"
											style={{ padding: "0 4px" }}
										>
											<DownOutlined style={{ fontSize: 10 }} />
										</Button>
									</Dropdown>
								</Space.Compact>
								<Button
									size="small"
									icon={<EditOutlined />}
									onClick={() => handleEditTemplate(t.id)}
								>
									编辑
								</Button>
								<Button
									size="small"
									danger
									icon={<DeleteOutlined />}
									onClick={() => handleDeleteTemplate(t.id)}
								/>
							</div>
						</div>
					))
				)}
			</div>

			{/* Asset selection modal */}
			<Modal
				title="可选：绑定处理资产"
				open={assetModalOpen}
				onCancel={() => setAssetModalOpen(false)}
				onOk={handleDeployConfirm}
				confirmLoading={deploying}
				okText="部署"
				width={640}
			>
				<Alert
					type="info"
					message="不选择则直接部署，不注入资产环境变量。"
					showIcon
					style={{ marginBottom: 16, fontSize: 12 }}
				/>
				<AssetPicker
					selectedIds={selectedAssetIds}
					onSelectionChange={setSelectedAssetIds}
					maxHeight={300}
				/>
			</Modal>

			<div className="deploy-section-title" style={{ marginTop: 20 }}>
				运行历史
				<span className="count">{deployments.length}</span>
			</div>
			<div className="deploy-section">
				{deployments.length === 0 ? (
					<div className="dep-empty">暂无部署记录</div>
				) : (
					deployments.map((d) => {
						const pipelineAssetIds = extractPipelineAssetIds(d.pipelineJSON);
						const pipelineAssetId = pipelineAssetIds[0];
						return (
							<div key={d.id} className="dep-card">
								<div className="dep-card-info">
									<div className="dep-card-name">{d.pipelineName}</div>
									<div className="dep-card-meta">
										<Tag color={STATUS_COLORS[d.status] || "default"}>
											{d.status}
										</Tag>
										<span>{d.nodeCount} 个节点</span>
										<span className="dot">•</span>
										<span>{new Date(d.createdAt).toLocaleString()}</span>
										{d.finishedAt && (
											<>
												<span className="dot">•</span>
												<span>
													完成: {new Date(d.finishedAt).toLocaleString()}
												</span>
											</>
										)}
									</div>
								</div>
								<div className="deploy-btn-list">
									{pipelineAssetId ? (
										<Button
											size="small"
											onClick={() => navigate(`/assets/${pipelineAssetId}`)}
										>
											查看关联资产
										</Button>
									) : null}
									<Button
										size="small"
										icon={<EyeOutlined />}
										onClick={() => navigate(`/workflows/${d.workflowName}`)}
									>
										查看
									</Button>
									<Button
										size="small"
										danger
										icon={<DeleteOutlined />}
										onClick={() => handleDeleteDeployment(d.id)}
									/>
								</div>
							</div>
						);
					})
				)}
			</div>
		</div>
	);
}
