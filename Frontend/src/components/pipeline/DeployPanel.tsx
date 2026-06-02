import {
	DeleteOutlined,
	EditOutlined,
	LinkOutlined,
	PlayCircleOutlined,
	ReloadOutlined,
} from "@ant-design/icons";
import {
	Alert,
	Button,
	Modal,
	message,
	Popconfirm,
	Select,
	Skeleton,
	Space,
	Tag,
} from "antd";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import {
	type Deployment,
	deletePipeline,
	deployTemplate,
	type ExecutionTarget,
	getPipeline,
	listDeployments,
	listExecutionTargets,
	listPipelines,
	type PipelineTemplate,
} from "../../api/pipelineApi";
import AssetPicker from "./AssetPicker";
import {
	COMPACT_TEMPLATE_LIMIT,
	dedupeTemplatesByName,
	prepareDeployments,
	SIDEBAR_TEMPLATE_LIMIT,
	TEMPLATE_PAGE_SIZE,
} from "./deployPanelUtils";
import { PipelineEmptyState } from "./PipelineEmptyState";
import type { Pipeline } from "./types";

const STATUS_COLORS: Record<string, string> = {
	Succeeded: "success",
	Running: "processing",
	Pending: "warning",
	Failed: "error",
	Error: "error",
	Expired: "default",
};

export type DeployPanelVariant = "full" | "compact" | "sidebar";

function PanelSkeleton({ rows = 3 }: { rows?: number }) {
	const keys = Array.from({ length: rows }, (_, i) => `skel-${i}`);
	return (
		<div className="deploy-panel-skeleton" data-testid="deploy-panel-loading">
			{keys.map((key) => (
				<Skeleton key={key} active paragraph={{ rows: 1 }} title={false} />
			))}
		</div>
	);
}

function TemplateCard({
	template,
	onRun,
	onEdit,
	onDelete,
	compactActions,
}: {
	template: PipelineTemplate;
	onRun: (id: string) => void;
	onEdit: (id: string) => void;
	onDelete: (id: string) => void;
	compactActions?: boolean;
}) {
	return (
		<div
			key={template.id}
			className={["dep-card", compactActions ? "dep-card--compact" : ""]
				.filter(Boolean)
				.join(" ")}
		>
			<div className="dep-card-info">
				<div className="dep-card-name" title={template.name}>
					{template.name}
				</div>
				{!compactActions ? (
					<div className="dep-card-meta">
						<span>{template.nodeCount} 个节点</span>
						<span className="dot">•</span>
						<span>{new Date(template.createdAt).toLocaleString()}</span>
					</div>
				) : (
					<div className="dep-card-meta dep-card-meta--compact">
						<span>{template.nodeCount} 个节点</span>
						<span className="dot">•</span>
						<span>{new Date(template.createdAt).toLocaleDateString()}</span>
					</div>
				)}
			</div>
			{compactActions ? (
				<Space.Compact>
					<Button
						size="small"
						icon={<EditOutlined />}
						onClick={() => onEdit(template.id)}
					>
						打开
					</Button>
					<Popconfirm
						title="删除此流水线？"
						description="删除后不可恢复"
						okText="删除"
						cancelText="取消"
						onConfirm={() => onDelete(template.id)}
					>
						<Button
							size="small"
							danger
							icon={<DeleteOutlined />}
							aria-label="删除流水线"
						/>
					</Popconfirm>
				</Space.Compact>
			) : (
				<div className="deploy-btn-list">
					<Space.Compact>
						<Button
							size="small"
							type="primary"
							icon={<PlayCircleOutlined />}
							onClick={() => onRun(template.id)}
						>
							运行
						</Button>
					</Space.Compact>
					<Button
						size="small"
						icon={<EditOutlined />}
						onClick={() => onEdit(template.id)}
					>
						编辑
					</Button>
					<Popconfirm
						title="删除此流水线？"
						description="删除后不可恢复"
						okText="删除"
						cancelText="取消"
						onConfirm={() => onDelete(template.id)}
					>
						<Button
							size="small"
							danger
							icon={<DeleteOutlined />}
							aria-label="删除流水线"
						/>
					</Popconfirm>
				</div>
			)}
		</div>
	);
}

export function DeployPanel({
	onEditTemplate,
	_refreshKey,
	compact,
	variant,
	onViewAll,
}: {
	onEditTemplate?: (pipeline: Pipeline) => void;
	_refreshKey?: number;
	compact?: boolean;
	variant?: DeployPanelVariant;
	onViewAll?: () => void;
}) {
	void _refreshKey;
	const resolvedVariant: DeployPanelVariant =
		variant ?? (compact ? "compact" : "full");
	const navigate = useNavigate();
	const [templates, setTemplates] = useState<PipelineTemplate[]>([]);
	const [deployments, setDeployments] = useState<Deployment[]>([]);
	const [targets, setTargets] = useState<ExecutionTarget[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [templateVisibleCount, setTemplateVisibleCount] =
		useState(TEMPLATE_PAGE_SIZE);

	const [assetModalOpen, setAssetModalOpen] = useState(false);
	const [deployTargetId, setDeployTargetId] = useState<string | null>(null);
	const [selectedAssetIds, setSelectedAssetIds] = useState<string[]>([]);
	const [selectedTargetId, setSelectedTargetId] = useState<string>("default");
	const [deploying, setDeploying] = useState(false);

	const displayTemplates = useMemo(
		() => dedupeTemplatesByName(templates),
		[templates],
	);
	const displayDeployments = useMemo(
		() => prepareDeployments(deployments),
		[deployments],
	);

	const refresh = useCallback(async () => {
		setLoading(true);
		setError(null);
		try {
			const shouldLoadDeployments = resolvedVariant === "compact";
			const [d, t, executionTargets] = await Promise.all([
				shouldLoadDeployments ? listDeployments() : Promise.resolve([]),
				listPipelines(),
				listExecutionTargets(),
			]);
			setDeployments(d);
			setTemplates(t);
			setTargets(executionTargets);
			const defaultTarget =
				executionTargets.find((target) => target.isDefault) ??
				executionTargets[0];
			if (defaultTarget) setSelectedTargetId(defaultTarget.id);
		} catch (err) {
			const detail = err instanceof Error ? err.message : String(err);
			setError(detail);
			setDeployments([]);
			setTemplates([]);
			setTargets([]);
		} finally {
			setLoading(false);
		}
	}, [resolvedVariant]);

	useEffect(() => {
		refresh();
	}, [refresh]);

	useEffect(() => {
		setTemplateVisibleCount(TEMPLATE_PAGE_SIZE);
	}, []);

	const handleDeployClick = (templateId: string) => {
		setDeployTargetId(templateId);
		setSelectedAssetIds([]);
		const defaultTarget =
			targets.find((target) => target.isDefault) ?? targets[0];
		setSelectedTargetId(defaultTarget?.id ?? "default");
		setAssetModalOpen(true);
	};

	const handleDeployConfirm = async () => {
		if (!deployTargetId) return;
		setDeploying(true);
		try {
			await deployTemplate(
				deployTargetId,
				selectedAssetIds.length > 0 ? selectedAssetIds : undefined,
				selectedTargetId,
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
				navigate(`/pipeline?templateId=${encodeURIComponent(id)}`);
			}
		} catch (err) {
			message.error(`加载模板失败: ${String(err)}`);
		}
	};

	const renderTemplateSection = (
		items: PipelineTemplate[],
		options?: { compactActions?: boolean; emptyLabel?: string },
	) => {
		if (loading) {
			return <PanelSkeleton rows={resolvedVariant === "sidebar" ? 2 : 3} />;
		}
		if (items.length === 0) {
			return (
				<PipelineEmptyState
					variant="deploy"
					title={options?.emptyLabel ?? "暂无已保存的流水线模板"}
				/>
			);
		}
		return items.map((t) => (
			<TemplateCard
				key={t.id}
				template={t}
				onRun={handleDeployClick}
				onEdit={handleEditTemplate}
				onDelete={handleDeleteTemplate}
				compactActions={options?.compactActions}
			/>
		));
	};

	if (resolvedVariant === "sidebar") {
		const sidebarTemplates = displayTemplates.slice(0, SIDEBAR_TEMPLATE_LIMIT);
		const hiddenCount = Math.max(
			displayTemplates.length - sidebarTemplates.length,
			0,
		);

		return (
			<div className="deploy-panel deploy-panel--sidebar">
				<div className="deploy-panel-sidebar__header">
					<div className="deploy-panel-sidebar__title">已保存</div>
					{!loading ? (
						<span className="count">{displayTemplates.length}</span>
					) : null}
				</div>
				{error ? (
					<Alert
						type="error"
						showIcon
						message="加载失败"
						description={error}
						style={{ marginBottom: 12, fontSize: 12 }}
					/>
				) : null}
				<div className="deploy-section deploy-section--sidebar">
					{renderTemplateSection(sidebarTemplates, {
						compactActions: true,
					})}
				</div>
				{hiddenCount > 0 || onViewAll ? (
					<div className="deploy-panel-sidebar__footer">
						{hiddenCount > 0 ? (
							<span className="deploy-panel-sidebar__hint">
								还有 {hiddenCount} 个模板未显示
							</span>
						) : null}
						{onViewAll ? (
							<Button type="link" size="small" onClick={onViewAll}>
								查看全部
							</Button>
						) : null}
					</div>
				) : null}
			</div>
		);
	}

	if (resolvedVariant === "compact") {
		const recentDeployments = displayDeployments.slice(0, 3);
		const compactTemplates = displayTemplates.slice(0, COMPACT_TEMPLATE_LIMIT);
		const hiddenTemplates = Math.max(
			displayTemplates.length - compactTemplates.length,
			0,
		);

		return (
			<div className="deploy-panel deploy-panel--compact">
				{error ? (
					<Alert
						type="error"
						showIcon
						message="加载失败"
						description={error}
						style={{ marginBottom: 12, fontSize: 12 }}
					/>
				) : null}
				<div className="deploy-panel-compact__section-header">
					<div className="deploy-section-title">最近执行</div>
					<Link
						to="/pipeline?tab=executions"
						className="deploy-panel-compact__link"
					>
						查看全部
					</Link>
				</div>

				<div className="deploy-section">
					{loading ? (
						<PanelSkeleton rows={2} />
					) : recentDeployments.length === 0 ? (
						<PipelineEmptyState variant="deploy" title="暂无执行记录" />
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
									onClick={() =>
										navigate(`/pipeline/executions/${d.workflowName}`)
									}
								>
									查看
								</Button>
							</div>
						))
					)}
				</div>

				<div className="deploy-section-title deploy-panel-compact__templates-title">
					模板
					{!loading ? (
						<span className="count">{displayTemplates.length}</span>
					) : null}
				</div>
				<div className="deploy-section">
					{renderTemplateSection(compactTemplates, {
						compactActions: true,
						emptyLabel: "暂无已保存的流水线模板",
					})}
				</div>
				{hiddenTemplates > 0 ? (
					<div className="deploy-panel-compact__hint">
						还有 {hiddenTemplates} 个同名模板已折叠
					</div>
				) : null}
			</div>
		);
	}

	const visibleTemplates = displayTemplates.slice(0, templateVisibleCount);

	return (
		<div className="deploy-panel">
			<div className="deploy-panel__toolbar">
				<h3>流水线管理</h3>
				<Space>
					<Button
						size="small"
						icon={<LinkOutlined />}
						onClick={() => navigate("/pipeline?tab=executions")}
					>
						执行记录
					</Button>
					<Button
						size="small"
						icon={<ReloadOutlined />}
						onClick={refresh}
						loading={loading}
					>
						刷新
					</Button>
				</Space>
			</div>

			{error ? (
				<Alert
					type="error"
					showIcon
					message="加载失败"
					description={error}
					style={{ marginBottom: 16, fontSize: 12 }}
				/>
			) : null}

			<div className="deploy-panel__section-card">
				<div className="deploy-section-title">
					已保存的流水线
					{!loading ? (
						<span className="count">{displayTemplates.length}</span>
					) : null}
				</div>
				<div className="deploy-section">
					{renderTemplateSection(visibleTemplates)}
				</div>
				{displayTemplates.length > visibleTemplates.length ? (
					<Button
						type="link"
						size="small"
						className="deploy-panel__load-more"
						onClick={() =>
							setTemplateVisibleCount((count) => count + TEMPLATE_PAGE_SIZE)
						}
					>
						加载更多模板（还剩{" "}
						{displayTemplates.length - visibleTemplates.length} 条）
					</Button>
				) : null}
			</div>

			<Modal
				title="运行流水线"
				open={assetModalOpen}
				onCancel={() => setAssetModalOpen(false)}
				onOk={handleDeployConfirm}
				confirmLoading={deploying}
				okText={selectedAssetIds.length > 0 ? "运行资产" : "无资产运行"}
				width={640}
			>
				<Alert
					type={selectedAssetIds.length > 0 ? "success" : "warning"}
					message={
						selectedAssetIds.length > 0
							? `将处理 ${selectedAssetIds.length} 个资产`
							: "当前是 no-asset run：不会注入资产环境变量。"
					}
					showIcon
					style={{ marginBottom: 16, fontSize: 12 }}
				/>
				<div className="deploy-run-field">
					<div className="deploy-run-field__label">执行目标</div>
					<Select
						value={selectedTargetId}
						onChange={setSelectedTargetId}
						style={{ width: "100%" }}
						options={(targets.length > 0
							? targets
							: [
									{
										id: "default",
										name: "Default Argo target",
										namespace: "default",
										cluster: "default",
										status: "unavailable",
										isDefault: true,
										argoServerConfigured: false,
									} satisfies ExecutionTarget,
								]
						).map((target) => ({
							value: target.id,
							label: `${target.name} · ${target.cluster}/${target.namespace}`,
							disabled: target.status !== "available",
						}))}
					/>
				</div>
				<AssetPicker
					selectedIds={selectedAssetIds}
					onSelectionChange={setSelectedAssetIds}
					maxHeight={300}
				/>
			</Modal>
		</div>
	);
}
