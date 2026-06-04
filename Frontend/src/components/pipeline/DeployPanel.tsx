import {
	DeleteOutlined,
	EditOutlined,
	HistoryOutlined,
	LinkOutlined,
	LockOutlined,
	PlayCircleOutlined,
	ReloadOutlined,
	RocketOutlined,
} from "@ant-design/icons";
import {
	Alert,
	App,
	Button,
	Checkbox,
	Input,
	Modal,
	Popconfirm,
	Segmented,
	Select,
	Skeleton,
	Space,
	Tag,
} from "antd";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import {
	batchDeployTemplate,
	type Deployment,
	deletePipeline,
	deployTemplate,
	type ExecutionTarget,
	getPipeline,
	listDeployments,
	listExecutionTargets,
	listPipelines,
	listPipelineVersions,
	type PipelineTemplate,
	promotePipeline,
} from "../../api/pipelineApi";
import { request } from "../../api/pipelineClient";
import { toAssetStyleId } from "../../lib/idDisplay";
import { buildWorkflowExecutionUrl } from "../../lib/workflowNavigation";
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
import { VersionHistoryDrawer } from "./VersionHistoryDrawer";

const STATUS_COLORS: Record<string, string> = {
	Succeeded: "success",
	Running: "processing",
	Pending: "warning",
	Failed: "error",
	Error: "error",
	Expired: "default",
};

export type DeployPanelVariant = "full" | "compact" | "sidebar";

function parseAssetIdsParam(raw: string | null): string[] {
	if (!raw) return [];
	return raw
		.split(",")
		.map((item) => item.trim())
		.filter(Boolean);
}

function parsePastedAssetIds(raw: string): {
	ids: string[];
	unique: number;
	duplicates: number;
} {
	const lines = raw
		.split(/[\n,;\t ]+/)
		.map((s) => s.trim())
		.filter(Boolean);
	const seen = new Set<string>();
	const ids: string[] = [];
	let duplicates = 0;
	for (const line of lines) {
		if (line.toLowerCase() === "asset_id" || line.toLowerCase() === "video_id")
			continue;
		if (seen.has(line)) {
			duplicates++;
			continue;
		}
		seen.add(line);
		ids.push(line);
	}
	return { ids, unique: ids.length, duplicates };
}

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

function AssetRunSummary({
	assetIds,
	onClear,
}: {
	assetIds: string[];
	onClear: () => void;
}) {
	if (assetIds.length === 0) {
		return (
			<Alert
				type="warning"
				showIcon
				message="无资产运行"
				description="本次运行不会注入资产环境变量，适合调试不依赖资产输入的流水线。"
				style={{ fontSize: 12 }}
			/>
		);
	}

	const visibleIds = assetIds.slice(0, 8);
	const hiddenCount = Math.max(assetIds.length - visibleIds.length, 0);

	return (
		<Alert
			type="success"
			showIcon
			message={
				assetIds.length > 1
					? `将为 ${assetIds.length} 个资产各创建 1 个独立 Workflow`
					: `将处理 ${assetIds.length} 个资产`
			}
			description={
				<div style={{ display: "grid", gap: 8 }}>
					<div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
						{visibleIds.map((assetId) => (
							<Tag key={assetId} color="blue" style={{ marginInlineEnd: 0 }}>
								{assetId}
							</Tag>
						))}
						{hiddenCount > 0 ? <Tag>+{hiddenCount}</Tag> : null}
					</div>
					<Button size="small" onClick={onClear}>
						转为无资产运行
					</Button>
				</div>
			}
			style={{ fontSize: 12 }}
		/>
	);
}

function TemplateCard({
	template,
	onRun,
	onEdit,
	onDelete,
	onVersionHistory,
	onPromote,
	activeVersion,
	compactActions,
	selectable,
	selected,
	onSelect,
}: {
	template: PipelineTemplate;
	onRun: (id: string) => void;
	onEdit: (id: string) => void;
	onDelete: (id: string) => void;
	onVersionHistory?: (template: PipelineTemplate) => void;
	onPromote?: (template: PipelineTemplate) => void;
	activeVersion?: number;
	compactActions?: boolean;
	selectable?: boolean;
	selected?: boolean;
	onSelect?: (id: string, selected: boolean) => void;
}) {
	return (
		<div
			key={template.id}
			className={["dep-card", compactActions ? "dep-card--compact" : ""]
				.filter(Boolean)
				.join(" ")}
		>
			{selectable ? (
				<Checkbox
					checked={selected}
					aria-label={`选择流水线 ${template.name}`}
					onChange={(event) => onSelect?.(template.id, event.target.checked)}
				/>
			) : null}
			<div className="dep-card-info">
				<div className="dep-card-name" title={template.name}>
					{template.name}
				</div>
				<div className="dep-card-id" title={`完整 ID: ${template.id}`}>
					ID: {toAssetStyleId(template.id)}
				</div>
				{!compactActions ? (
					<div className="dep-card-meta">
						<Tag
							color="blue"
							style={{ cursor: "pointer" }}
							onClick={(e) => {
								e.stopPropagation();
								onVersionHistory?.(template);
							}}
							title="点击查看版本历史"
						>
							v{template.version} <HistoryOutlined />
						</Tag>
						{template.scope === "prod" ? (
							<Tag color="green" style={{ fontSize: 11 }}>
								<LockOutlined /> 正式版
							</Tag>
						) : (
							<Tag color="blue" style={{ fontSize: 11 }}>
								Dev 草稿
							</Tag>
						)}
						{activeVersion != null && activeVersion < template.version ? (
							<Tag color="orange" style={{ fontSize: 11, marginLeft: 4 }}>
								活跃: v{activeVersion}
							</Tag>
						) : null}
						{template.versionCount && template.versionCount > 1 ? (
							<span>{template.versionCount} 个版本</span>
						) : null}
						<span>{template.nodeCount} 个步骤</span>
						<span className="dot">•</span>
						<span>{new Date(template.createdAt).toLocaleString()}</span>
					</div>
				) : (
					<div className="dep-card-meta dep-card-meta--compact">
						<Tag
							color="blue"
							style={{ cursor: "pointer" }}
							onClick={(e) => {
								e.stopPropagation();
								onVersionHistory?.(template);
							}}
							title="点击查看版本历史"
						>
							v{template.version} <HistoryOutlined />
						</Tag>
						{template.scope === "prod" ? (
							<Tag color="green" style={{ fontSize: 11 }}>
								<LockOutlined /> 正式版
							</Tag>
						) : (
							<Tag color="blue" style={{ fontSize: 11 }}>
								Dev 草稿
							</Tag>
						)}
						{activeVersion != null && activeVersion < template.version ? (
							<Tag color="orange" style={{ fontSize: 11, marginLeft: 4 }}>
								活跃: v{activeVersion}
							</Tag>
						) : null}
						{template.versionCount && template.versionCount > 1 ? (
							<span>{template.versionCount} 版</span>
						) : null}
						<span>{template.nodeCount} 个步骤</span>
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
					{template.scope !== "prod" ? (
						<Button
							size="small"
							icon={<RocketOutlined />}
							onClick={() => onPromote?.(template)}
							title="发布到正式版"
						>
							发布
						</Button>
					) : null}
					{template.scope !== "prod" ? (
						<Button
							size="small"
							icon={<EditOutlined />}
							onClick={() => onEdit(template.id)}
						>
							编辑
						</Button>
					) : null}
					{template.scope !== "prod" ? (
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
					) : null}
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
	const { message: messageApi } = App.useApp();
	void _refreshKey;
	const resolvedVariant: DeployPanelVariant =
		variant ?? (compact ? "compact" : "full");
	const navigate = useNavigate();
	const [searchParams, setSearchParams] = useSearchParams();
	const queryAssetIds = useMemo(
		() => parseAssetIdsParam(searchParams.get("asset_ids")),
		[searchParams],
	);
	const [templates, setTemplates] = useState<PipelineTemplate[]>([]);
	const [deployments, setDeployments] = useState<Deployment[]>([]);
	const [targets, setTargets] = useState<ExecutionTarget[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [templateVisibleCount, setTemplateVisibleCount] =
		useState(TEMPLATE_PAGE_SIZE);
	const [selectedTemplateIds, setSelectedTemplateIds] = useState<string[]>([]);
	const [bulkDeletingTemplates, setBulkDeletingTemplates] = useState(false);

	const [assetModalOpen, setAssetModalOpen] = useState(false);
	const [deployTargetId, setDeployTargetId] = useState<string | null>(null);
	const [selectedAssetIds, setSelectedAssetIds] = useState<string[]>([]);
	const [selectedTargetId, setSelectedTargetId] = useState<string>("default");
	const [deploying, setDeploying] = useState(false);
	const [assetPickerResetKey, setAssetPickerResetKey] = useState(0);
	const [deployVersions, setDeployVersions] = useState<PipelineTemplate[]>([]);
	const [selectedDeployVersion, setSelectedDeployVersion] = useState<
		number | undefined
	>();
	const [versionDrawerOpen, setVersionDrawerOpen] = useState(false);
	const [versionDrawerTemplate, setVersionDrawerTemplate] =
		useState<PipelineTemplate | null>(null);
	const [activeVersionByTemplate, setActiveVersionByTemplate] = useState<
		Record<string, number>
	>({});
	const [scopeTab, setScopeTab] = useState<string>("all");
	const [assetMode, setAssetMode] = useState<"search" | "paste">("search");
	const [pasteText, setPasteText] = useState("");
	const [pasteFileKey] = useState(0);

	const updateSelectedAssetIds = useCallback(
		(nextIds: string[]) => {
			setSelectedAssetIds(nextIds);
			const nextParams = new URLSearchParams(searchParams);
			if (nextIds.length > 0) {
				nextParams.set("asset_ids", nextIds.join(","));
			} else {
				nextParams.delete("asset_ids");
			}
			setSearchParams(nextParams, { replace: true });
		},
		[searchParams, setSearchParams],
	);

	const displayTemplates = useMemo(() => {
		const deduped = dedupeTemplatesByName(templates);
		if (scopeTab === "dev") return deduped.filter((t) => t.scope !== "prod");
		if (scopeTab === "prod") return deduped.filter((t) => t.scope === "prod");
		return deduped;
	}, [templates, scopeTab]);
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
			const activeMap: Record<string, number> = {};
			for (const tmpl of t) {
				const av = tmpl.activeVersion;
				if (av != null && av > 0 && av < tmpl.version) {
					activeMap[tmpl.name] = av;
				}
			}
			setActiveVersionByTemplate(activeMap);
			setSelectedTemplateIds((prev) =>
				prev.filter((id) => t.some((template) => template.id === id)),
			);
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
		const currentTemplate = templates.find(
			(template) => template.id === templateId,
		);
		setDeployTargetId(templateId);
		setDeployVersions(currentTemplate ? [currentTemplate] : []);
		const activeVersion = activeVersionByTemplate[currentTemplate?.name ?? ""];
		setSelectedDeployVersion(activeVersion ?? currentTemplate?.version);
		setSelectedAssetIds(queryAssetIds);
		setAssetPickerResetKey((key) => key + 1);
		const defaultTarget =
			targets.find((target) => target.isDefault) ?? targets[0];
		setSelectedTargetId(defaultTarget?.id ?? "default");
		setAssetModalOpen(true);
		listPipelineVersions(templateId)
			.then((versions) => {
				setDeployVersions(versions);
				const av = activeVersionByTemplate[currentTemplate?.name ?? ""];
				setSelectedDeployVersion(
					av ?? currentTemplate?.version ?? versions[0]?.version,
				);
			})
			.catch((err) => {
				messageApi.warning(`版本列表加载失败，将运行当前版本: ${String(err)}`);
			});
	};

	const closeAssetModal = () => {
		setAssetModalOpen(false);
		setSelectedAssetIds(queryAssetIds);
		setDeployVersions([]);
		setSelectedDeployVersion(undefined);
		setAssetPickerResetKey((key) => key + 1);
	};

	const handleDeployConfirm = async () => {
		if (!deployTargetId) return;
		setDeploying(true);
		try {
			if (selectedAssetIds.length > 1) {
				const result = await batchDeployTemplate(
					deployTargetId,
					selectedAssetIds,
					selectedTargetId,
					selectedDeployVersion,
				);
				const failedCount = result.failed?.length ?? 0;
				if (failedCount > 0) {
					messageApi.warning(
						`已提交 ${result.items.length} 个 Workflow，${failedCount} 个资产失败`,
					);
				} else {
					messageApi.success(
						`已提交 ${result.items.length} 个独立 Workflow（批次 ${result.batchId.slice(0, 8)}）`,
					);
				}
			} else {
				await deployTemplate(
					deployTargetId,
					selectedAssetIds,
					selectedTargetId,
					selectedDeployVersion,
				);
				messageApi.success("部署成功");
			}
			closeAssetModal();
			setDeploying(false);
			void refresh();
		} catch (err) {
			messageApi.error(`部署失败: ${String(err)}`);
			setDeploying(false);
		}
	};

	const handleDeleteTemplate = async (id: string) => {
		try {
			await deletePipeline(id);
			messageApi.success("已删除流水线模板");
			refresh();
		} catch (err) {
			messageApi.error(`删除失败: ${String(err)}`);
		}
	};

	const handleBulkDeleteTemplates = async () => {
		if (selectedTemplateIds.length === 0) return;
		setBulkDeletingTemplates(true);
		try {
			const results = await Promise.allSettled(
				selectedTemplateIds.map((id) => deletePipeline(id)),
			);
			const failedCount = results.filter(
				(result) => result.status === "rejected",
			).length;
			const deletedCount = results.length - failedCount;
			if (deletedCount > 0) {
				messageApi.success(`已删除 ${deletedCount} 条流水线`);
			}
			if (failedCount > 0) {
				messageApi.error(`${failedCount} 条流水线删除失败`);
			}
			setSelectedTemplateIds([]);
			await refresh();
		} finally {
			setBulkDeletingTemplates(false);
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
			messageApi.error(`加载模板失败: ${String(err)}`);
		}
	};

	const handleVersionHistory = (template: PipelineTemplate) => {
		setVersionDrawerTemplate(template);
		setVersionDrawerOpen(true);
	};

	const handlePromote = async (template: PipelineTemplate) => {
		try {
			const promoted = await promotePipeline(template.id);
			messageApi.success(
				`已发布 ${template.name} v${promoted.version} 到正式版（prod）`,
			);
			refresh();
		} catch (err) {
			messageApi.error(`发布失败: ${String(err)}`);
		}
	};

	const handleSetActiveVersion = async (
		template: PipelineTemplate,
		version: number,
	) => {
		try {
			await request("PATCH", `/pipelines/${template.id}/active-version`, {
				activeVersion: version,
			});
			setActiveVersionByTemplate((prev) => ({
				...prev,
				[template.name]: version,
			}));
			messageApi.success(
				`已将 ${template.name} 的活跃版本设为 v${version}，下次运行将默认使用此版本`,
			);
		} catch (err) {
			messageApi.error(`设置活跃版本失败: ${String(err)}`);
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
				onVersionHistory={handleVersionHistory}
				onPromote={handlePromote}
				activeVersion={activeVersionByTemplate[t.name]}
				compactActions={options?.compactActions}
				selectable={resolvedVariant === "full"}
				selected={selectedTemplateIds.includes(t.id)}
				onSelect={(id, checked) =>
					setSelectedTemplateIds((prev) =>
						checked
							? Array.from(new Set([...prev, id]))
							: prev.filter((item) => item !== id),
					)
				}
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
										{d.scope === "prod" ? (
											<Tag color="green" style={{ fontSize: 11 }}>
												<LockOutlined /> 正式版
											</Tag>
										) : d.scope ? (
											<Tag color="blue" style={{ fontSize: 11 }}>
												Dev 草稿
											</Tag>
										) : null}
									</div>
								</div>
								<Button
									size="small"
									icon={<LinkOutlined />}
									onClick={() => {
										navigate(buildWorkflowExecutionUrl(d.workflowName, d.id));
									}}
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
	const selectableTemplateIds = displayTemplates.map((template) => template.id);
	const selectedTemplateIdSet = new Set(selectedTemplateIds);
	const allTemplatesSelected =
		selectableTemplateIds.length > 0 &&
		selectableTemplateIds.every((id) => selectedTemplateIdSet.has(id));
	const someTemplatesSelected =
		selectedTemplateIds.length > 0 && !allTemplatesSelected;
	const visibleQueryAssetIds = queryAssetIds.slice(0, 6);
	const hiddenQueryAssetCount = Math.max(
		queryAssetIds.length - visibleQueryAssetIds.length,
		0,
	);

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
			{queryAssetIds.length > 0 ? (
				<Alert
					type="info"
					showIcon
					message={`已选择 ${queryAssetIds.length} 个资产`}
					description={
						<div className="deploy-panel__asset-context-body">
							<span>请选择要运行的流水线和版本，确认后即可提交运行。</span>
							<div className="deploy-panel__asset-context-assets">
								{visibleQueryAssetIds.map((assetId) => (
									<Tag
										key={assetId}
										color="blue"
										style={{ marginInlineEnd: 0 }}
									>
										{assetId}
									</Tag>
								))}
								{hiddenQueryAssetCount > 0 ? (
									<Tag>+{hiddenQueryAssetCount}</Tag>
								) : null}
							</div>
							<Button size="small" onClick={() => updateSelectedAssetIds([])}>
								转为无资产运行
							</Button>
						</div>
					}
					style={{ marginBottom: 12 }}
				/>
			) : null}

			<div className="deploy-panel__section-card">
				<div
					style={{
						marginBottom: 12,
						display: "flex",
						justifyContent: "center",
					}}
				>
					<Segmented
						options={[
							{ value: "all", label: "全部" },
							{ value: "dev", label: "我的 Dev" },
							{ value: "prod", label: "共享正式版" },
						]}
						value={scopeTab}
						onChange={(v) => setScopeTab(v as string)}
					/>
				</div>
				<div className="deploy-section-title">
					<div>
						已保存的流水线
						{!loading ? (
							<span className="count">{displayTemplates.length}</span>
						) : null}
					</div>
					{!loading && displayTemplates.length > 0 ? (
						<Space>
							<Checkbox
								checked={allTemplatesSelected}
								indeterminate={someTemplatesSelected}
								onChange={(event) =>
									setSelectedTemplateIds(
										event.target.checked ? selectableTemplateIds : [],
									)
								}
							>
								全选
							</Checkbox>
							<Popconfirm
								title={`删除选中的 ${selectedTemplateIds.length} 条流水线？`}
								description="删除后不可恢复"
								okText="删除"
								cancelText="取消"
								disabled={selectedTemplateIds.length === 0}
								onConfirm={handleBulkDeleteTemplates}
							>
								<Button
									size="small"
									danger
									icon={<DeleteOutlined />}
									disabled={selectedTemplateIds.length === 0}
									loading={bulkDeletingTemplates}
								>
									批量删除
									{selectedTemplateIds.length > 0
										? `（${selectedTemplateIds.length}）`
										: ""}
								</Button>
							</Popconfirm>
						</Space>
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
				onCancel={closeAssetModal}
				onOk={handleDeployConfirm}
				confirmLoading={deploying}
				okText={selectedAssetIds.length > 0 ? "运行资产" : "无资产运行"}
				width={640}
			>
				<div style={{ marginBottom: 16 }}>
					<AssetRunSummary
						assetIds={selectedAssetIds}
						onClear={() => updateSelectedAssetIds([])}
					/>
				</div>
				<div className="deploy-run-field">
					<div className="deploy-run-field__label">模板版本</div>
					<Select
						aria-label="模板版本"
						value={selectedDeployVersion}
						onChange={setSelectedDeployVersion}
						style={{ width: "100%", marginBottom: 12 }}
						options={deployVersions.map((version) => ({
							value: version.version,
							label: `版本 v${version.version} · ${version.nodeCount} 个步骤 · 保存于 ${new Date(
								version.createdAt,
							).toLocaleString()}`,
						}))}
					/>
				</div>
				<div className="deploy-run-field">
					<div className="deploy-run-field__label">执行目标</div>
					<Select
						aria-label="执行目标"
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
				<div style={{ marginBottom: 12 }}>
					<Segmented
						options={[
							{ value: "search", label: "搜索资产" },
							{ value: "paste", label: "粘贴资产 ID" },
						]}
						value={assetMode}
						onChange={(v) => {
							setAssetMode(v as "search" | "paste");
							if (v === "paste") {
								const parsed = parsePastedAssetIds(pasteText);
								updateSelectedAssetIds(parsed.ids);
							}
						}}
					/>
				</div>
				{assetMode === "paste" ? (
					<div style={{ display: "grid", gap: 8 }}>
						<Input.TextArea
							rows={6}
							value={pasteText}
							onChange={(e) => {
								setPasteText(e.target.value);
								const parsed = parsePastedAssetIds(e.target.value);
								updateSelectedAssetIds(parsed.ids);
							}}
							placeholder="粘贴 asset ID，每行一个，或粘贴 CSV"
							style={{ fontFamily: "monospace", fontSize: 12 }}
						/>
						<div style={{ display: "flex", gap: 8, alignItems: "center" }}>
							<label
								className="ant-btn ant-btn-default"
								style={{ cursor: "pointer" }}
							>
								上传 CSV
								<input
									type="file"
									accept=".csv,.txt"
									key={pasteFileKey}
									style={{ display: "none" }}
									onChange={(e) => {
										const file = e.target.files?.[0];
										if (!file) return;
										file.text().then((text) => {
											setPasteText(text);
											const parsed = parsePastedAssetIds(text);
											updateSelectedAssetIds(parsed.ids);
										});
									}}
								/>
							</label>
							{pasteText
								? (() => {
										const parsed = parsePastedAssetIds(pasteText);
										return (
											<span
												style={{
													fontSize: 12,
													color: parsed.unique > 0 ? "#16a34a" : "#999",
												}}
											>
												{parsed.unique} 个资产
												{parsed.duplicates > 0
													? `（${parsed.duplicates} 个重复已移除）`
													: ""}
											</span>
										);
									})()
								: null}
						</div>
					</div>
				) : (
					<AssetPicker
						selectedIds={selectedAssetIds}
						onSelectionChange={updateSelectedAssetIds}
						maxHeight={300}
						resetKey={assetPickerResetKey}
					/>
				)}
			</Modal>
			{versionDrawerTemplate ? (
				<VersionHistoryDrawer
					open={versionDrawerOpen}
					pipelineName={versionDrawerTemplate.name}
					templateId={versionDrawerTemplate.id}
					activeVersion={activeVersionByTemplate[versionDrawerTemplate.name]}
					onClose={() => {
						setVersionDrawerOpen(false);
						setVersionDrawerTemplate(null);
					}}
					onSetActive={(version) =>
						handleSetActiveVersion(versionDrawerTemplate, version)
					}
				/>
			) : null}
		</div>
	);
}
