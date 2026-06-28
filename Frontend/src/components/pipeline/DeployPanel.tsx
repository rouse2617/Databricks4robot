import {
	DeleteOutlined,
	EditOutlined,
	EyeOutlined,
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
	Card,
	Checkbox,
	Input,
	Modal,
	Popconfirm,
	Radio,
	Select,
	Skeleton,
	Space,
	Tag,
	Tooltip,
} from "antd";
import {
	type ChangeEvent,
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { deployPipelineForAssets } from "../../api/deployPipelineRun";
import {
	type DeployConfigSelection,
	type Deployment,
	deletePipeline,
	type ExecutionTarget,
	getPipeline,
	type ListPipelinesParams,
	listDeployments,
	listExecutionTargets,
	listRuntimeMounts,
	type RuntimeMountCatalog,
	listPipelines,
	listPipelineVersions,
	type PipelineTemplate,
	promotePipeline,
} from "../../api/pipelineApi";
import { request } from "../../api/pipelineClient";
import {
	type PipelineConfig,
	type PipelineConfigVersion,
	pipelineConfigApi,
} from "../../api/pipelineConfigs";
import { toAssetStyleId } from "../../lib/idDisplay";
import { batchJobDetailLocationState } from "../../lib/pipelineNavigation";
import AssetPicker, { type AssetPickerHandle } from "./AssetPicker";
import {
	COMPACT_TEMPLATE_LIMIT,
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
type DeployConfigSourceMode = "saved" | "upload" | "inline";

interface UploadDraftFile {
	name: string;
	size: number;
	content: string;
}

export function buildDeployConfigSelection(input: {
	configSourceMode: DeployConfigSourceMode;
	selectedSavedConfig: PipelineConfig | null;
	uploadDraftFile: UploadDraftFile | null;
	inlineDraftName: string;
	inlineDraftContent: string;
	selectedConfigVersion?: number;
	configMountPath: string;
	configTargetFilename: string;
}): DeployConfigSelection | undefined {
	const mountPath = input.configMountPath.trim();
	const targetFilename = input.configTargetFilename.trim();
	if (!mountPath || !targetFilename) {
		return undefined;
	}
	if (input.configSourceMode === "saved" && input.selectedSavedConfig) {
		return {
			mode: "saved",
			configId: input.selectedSavedConfig.id,
			version: input.selectedConfigVersion ?? input.selectedSavedConfig.currentVersion,
			fileName: input.selectedSavedConfig.name,
			mountPath,
			targetFilename,
		};
	}
	if (input.configSourceMode === "upload" && input.uploadDraftFile) {
		return {
			mode: "upload",
			fileName: input.uploadDraftFile.name,
			content: input.uploadDraftFile.content,
			mountPath,
			targetFilename,
		};
	}
	if (input.configSourceMode === "inline" && input.inlineDraftContent.trim()) {
		return {
			mode: "inline",
			fileName: input.inlineDraftName.trim() || "runtime-config.yaml",
			content: input.inlineDraftContent,
			mountPath,
			targetFilename,
		};
	}
	return undefined;
}

function formatFileSize(bytes: number) {
	if (bytes < 1024) return `${bytes} B`;
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
	return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function parseAssetIdsParam(raw: string | null): string[] {
	if (!raw) return [];
	return raw
		.split(",")
		.map((item) => item.trim())
		.filter(Boolean);
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
			type={assetIds.length >= 2 ? "info" : "success"}
			showIcon
			message={
				assetIds.length >= 2
					? `将创建批量任务，共 ${assetIds.length} 个子任务`
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

// Auto-generated throwaway drafts use a timestamp default name like
// `pipeline-1782564077864` and usually carry a single step. They flood the
// list and bury the curated pipelines, so we let users hide them in one click.
const AUTO_DRAFT_NAME_RE = /^pipeline-\d{10,}$/;
function isAutoNamedDraft(template: PipelineTemplate): boolean {
	return (
		template.scope !== "prod" &&
		AUTO_DRAFT_NAME_RE.test(template.name) &&
		(template.nodeCount == null || template.nodeCount <= 1)
	);
}

function TemplateCard({
	template,
	onRun,
	onEdit,
	onDelete,
	onVersionHistory,
	onPromote,
	onView,
	activeVersion,
	recommended,
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
	onView?: (id: string) => void;
	activeVersion?: number;
	recommended?: boolean;
	compactActions?: boolean;
	selectable?: boolean;
	selected?: boolean;
	onSelect?: (id: string, selected: boolean) => void;
}) {
	// The version actually running in production may lag behind the latest
	// saved draft. Surface the *active* version as the primary badge so users
	// don't mistake the latest edit for what's live; the latest is demoted to
	// a muted secondary tag.
	const hasDistinctActive =
		activeVersion != null && activeVersion < template.version;
	const versionTags = (
		<>
			<Tag
				color={hasDistinctActive ? "green" : "blue"}
				style={{ cursor: "pointer" }}
				onClick={(e) => {
					e.stopPropagation();
					onVersionHistory?.(template);
				}}
				title="点击查看版本历史"
			>
				{hasDistinctActive ? "活跃 " : ""}v
				{hasDistinctActive ? activeVersion : template.version}{" "}
				<HistoryOutlined />
			</Tag>
			{hasDistinctActive ? (
				<Tag
					style={{ fontSize: 11, color: "#94a3b8", borderColor: "#e2e8f0" }}
					title="最新已保存版本（尚未设为活跃）"
				>
					最新 v{template.version}
				</Tag>
			) : null}
		</>
	);
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
						{versionTags}
						{template.scope === "prod" ? (
							<Tag color="green" style={{ fontSize: 11 }}>
								<LockOutlined /> 正式版
							</Tag>
						) : (
							<Tag color="blue" style={{ fontSize: 11 }}>
								Dev 草稿
							</Tag>
						)}
						{template.owner ? (
							<Tag style={{ fontSize: 11, marginLeft: 4 }}>{template.owner}</Tag>
						) : null}
						{recommended ? (
							<Tag color="gold" style={{ fontSize: 11, marginLeft: 4 }}>
								推荐
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
						{versionTags}
						{template.scope === "prod" ? (
							<Tag color="green" style={{ fontSize: 11 }}>
								<LockOutlined /> 正式版
							</Tag>
						) : (
							<Tag color="blue" style={{ fontSize: 11 }}>
								Dev 草稿
							</Tag>
						)}
						{template.owner ? (
							<Tag style={{ fontSize: 11, marginLeft: 4 }}>{template.owner}</Tag>
						) : null}
						{recommended ? (
							<Tag color="gold" style={{ fontSize: 11, marginLeft: 4 }}>
								推荐
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
					{template.scope === "prod" ? (
						<Button
							size="small"
							icon={<EyeOutlined />}
							onClick={() => onView?.(template.id) ?? onEdit(template.id)}
						>
							查看定义
						</Button>
					) : (
						<Button
							size="small"
							icon={<EditOutlined />}
							onClick={() => onEdit(template.id)}
						>
							打开
						</Button>
					)}
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
					) : (
						<Button
							size="small"
							icon={<EyeOutlined />}
							onClick={() => onView?.(template.id) ?? onEdit(template.id)}
						>
							查看定义
						</Button>
					)}
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
	const { message: messageApi, modal } = App.useApp();
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
	const [templateTotal, setTemplateTotal] = useState(0);
	const [templatePage, setTemplatePage] = useState(1);
	const [templateQueryDraft, setTemplateQueryDraft] = useState("");
	const [templateQuery, setTemplateQuery] = useState("");
	const [templateScope, setTemplateScope] = useState<string | undefined>();
	const [templateSort, setTemplateSort] =
		useState<ListPipelinesParams["sort"]>("updated_at_desc");
	const [loadingMoreTemplates, setLoadingMoreTemplates] = useState(false);
	const [hideAutoDrafts, setHideAutoDrafts] = useState<boolean>(
		() => localStorage.getItem("db.pipeline.hideAutoDrafts") === "1",
	);
	const [deployments, setDeployments] = useState<Deployment[]>([]);
	const [targets, setTargets] = useState<ExecutionTarget[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [selectedTemplateIds, setSelectedTemplateIds] = useState<string[]>([]);
	const [bulkDeletingTemplates, setBulkDeletingTemplates] = useState(false);

	const [assetModalOpen, setAssetModalOpen] = useState(false);
	const [deployTargetId, setDeployTargetId] = useState<string | null>(null);
	const [selectedAssetIds, setSelectedAssetIds] = useState<string[]>([]);
	const [selectedTargetId, setSelectedTargetId] = useState<string>("default");
	const [targetWarning, setTargetWarning] = useState<string | null>(null);
	const [deploying, setDeploying] = useState(false);
	const [assetPickerResetKey, setAssetPickerResetKey] = useState(0);
	const assetPickerRef = useRef<AssetPickerHandle>(null);
	const fileInputRef = useRef<HTMLInputElement | null>(null);
	const [deployVersions, setDeployVersions] = useState<PipelineTemplate[]>([]);
	const [selectedDeployVersion, setSelectedDeployVersion] = useState<
		number | undefined
	>();
	const [versionDrawerOpen, setVersionDrawerOpen] = useState(false);
	const [versionDrawerTemplate, setVersionDrawerTemplate] =
		useState<PipelineTemplate | null>(null);
	const refreshInFlightRef = useRef(false);
	const templatesInFlightRef = useRef(false);
	const templateSearchDebounceRef = useRef<ReturnType<
		typeof setTimeout
	> | null>(null);
	const prevTemplateFiltersRef = useRef({
		query: templateQuery,
		scope: templateScope,
		sort: templateSort,
	});
	const [activeVersionByTemplate, setActiveVersionByTemplate] = useState<
		Record<string, number>
	>({});
	const [globalConfigEnabled, setGlobalConfigEnabled] = useState(false);
	const [configSourceMode, setConfigSourceMode] =
		useState<DeployConfigSourceMode>("saved");
	const [savedConfigs, setSavedConfigs] = useState<PipelineConfig[]>([]);
	const [savedConfigsLoading, setSavedConfigsLoading] = useState(false);
	const [savedConfigsError, setSavedConfigsError] = useState<string | null>(
		null,
	);
	const [selectedSavedConfigId, setSelectedSavedConfigId] = useState<
		string | undefined
	>();
	const [selectedConfigVersion, setSelectedConfigVersion] = useState<number>(1);
	const [configVersions, setConfigVersions] = useState<PipelineConfigVersion[]>([]);
	const [uploadDraftFile, setUploadDraftFile] =
		useState<UploadDraftFile | null>(null);
	const [inlineDraftName, setInlineDraftName] = useState("runtime-config.yaml");
	const [inlineDraftContent, setInlineDraftContent] = useState("");
	const [configMountPath, setConfigMountPath] = useState("/app/configs");
	const [configTargetFilename, setConfigTargetFilename] = useState("");
	const recommendedTemplateId = useMemo(() => {
		if (queryAssetIds.length === 0 || templates.length === 0) return null;
		const prodTemplates = templates.filter(
			(template) => template.scope === "prod",
		);
		if (prodTemplates.length === 1) return prodTemplates[0].id;
		const activeProd = prodTemplates.find(
			(template) => activeVersionByTemplate[template.name] === template.version,
		);
		return activeProd?.id ?? prodTemplates[0]?.id ?? templates[0]?.id ?? null;
	}, [activeVersionByTemplate, queryAssetIds.length, templates]);

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

	const displayTemplates = templates;
	const selectedSavedConfig = useMemo(
		() =>
			savedConfigs.find((config) => config.id === selectedSavedConfigId) ??
			null,
		[savedConfigs, selectedSavedConfigId],
	);
	const selectedConfigSummary = useMemo(() => {
		if (configSourceMode === "saved" && selectedSavedConfig) {
			return {
				sourceLabel: "平台配置",
				name: selectedSavedConfig.name,
				meta: `${selectedSavedConfig.owner} · ${selectedSavedConfig.lifecycle} · v${selectedSavedConfig.currentVersion}`,
			};
		}
		if (configSourceMode === "upload" && uploadDraftFile) {
			return {
				sourceLabel: "本地上传",
				name: uploadDraftFile.name,
				meta: `${formatFileSize(uploadDraftFile.size)} · deploy 草稿`,
			};
		}
		if (configSourceMode === "inline" && inlineDraftContent.trim()) {
			return {
				sourceLabel: "在线编辑",
				name: inlineDraftName.trim() || "runtime-config.yaml",
				meta: `${inlineDraftContent.split("\n").length} 行 · deploy 草稿`,
			};
		}
		return null;
	}, [
		configSourceMode,
		inlineDraftContent,
		inlineDraftName,
		selectedSavedConfig,
		uploadDraftFile,
	]);
	const deployTargetTemplate = useMemo(
		() => templates.find((item) => item.id === deployTargetId),
		[deployTargetId, templates],
	);
	const displayDeployments = useMemo(
		() => prepareDeployments(deployments),
		[deployments],
	);

	const applyTemplateFiltersFromTemplates = useCallback(
		(items: PipelineTemplate[]) => {
			const activeMap: Record<string, number> = {};
			for (const tmpl of items) {
				const av = tmpl.activeVersion;
				if (av != null && av > 0 && av < tmpl.version) {
					activeMap[tmpl.name] = av;
				}
			}
			setActiveVersionByTemplate(activeMap);
			setSelectedTemplateIds((prev) =>
				prev.filter((id) => items.some((template) => template.id === id)),
			);
		},
		[],
	);

	const fetchTemplates = useCallback(
		async (page: number, append: boolean) => {
			if (append) {
				setLoadingMoreTemplates(true);
			}
			try {
				const resp = await listPipelines({
					page,
					pageSize: TEMPLATE_PAGE_SIZE,
					q: templateQuery || undefined,
					scope: templateScope,
					sort: templateSort,
				});
				setTemplateTotal(resp.total);
				setTemplatePage(resp.page);
				setTemplates((prev) => {
					const next = append ? [...prev, ...resp.items] : resp.items;
					applyTemplateFiltersFromTemplates(next);
					return next;
				});
			} catch {
				if (!append) {
					setTemplates([]);
					setTemplateTotal(0);
				}
				throw new Error("流水线模板加载失败");
			} finally {
				if (append) {
					setLoadingMoreTemplates(false);
				}
			}
		},
		[
			applyTemplateFiltersFromTemplates,
			templateQuery,
			templateScope,
			templateSort,
		],
	);

	const refreshTemplates = useCallback(async () => {
		if (templatesInFlightRef.current) return;
		templatesInFlightRef.current = true;
		setLoading(true);
		try {
			const resp = await listPipelines({
				page: 1,
				pageSize: TEMPLATE_PAGE_SIZE,
				q: templateQuery || undefined,
				scope: templateScope,
				sort: templateSort,
			});
			setTemplates(resp.items);
			setTemplateTotal(resp.total);
			setTemplatePage(resp.page);
			applyTemplateFiltersFromTemplates(resp.items);
			setError((prev) => (prev?.includes("流水线模板加载失败") ? null : prev));
		} catch {
			setTemplates([]);
			setTemplateTotal(0);
			setError((prev) => {
				const parts = (prev ?? "")
					.split("；")
					.filter((part) => part && part !== "流水线模板加载失败");
				parts.push("流水线模板加载失败");
				return parts.join("；");
			});
		} finally {
			setLoading(false);
			templatesInFlightRef.current = false;
		}
	}, [
		applyTemplateFiltersFromTemplates,
		templateQuery,
		templateScope,
		templateSort,
	]);

	const refreshAll = useCallback(async () => {
		if (refreshInFlightRef.current) return;
		refreshInFlightRef.current = true;
		setLoading(true);
		setError(null);
		const shouldLoadDeployments = resolvedVariant === "compact";
		const { query, scope, sort } = prevTemplateFiltersRef.current;
		try {
			const [deploymentsResult, templatesResult, targetsResult] =
				await Promise.allSettled([
					shouldLoadDeployments ? listDeployments() : Promise.resolve([]),
					listPipelines({
						page: 1,
						pageSize: TEMPLATE_PAGE_SIZE,
						q: query || undefined,
						scope,
						sort,
					}),
					listExecutionTargets(),
				]);
			const partialErrors: string[] = [];

			if (deploymentsResult.status === "fulfilled") {
				setDeployments(deploymentsResult.value);
			} else if (shouldLoadDeployments) {
				partialErrors.push("执行记录加载失败");
				setDeployments([]);
			}

			if (templatesResult.status === "fulfilled") {
				const resp = templatesResult.value;
				const templateItems = resp.items ?? [];
				setTemplates(templateItems);
				setTemplateTotal(resp.total ?? templateItems.length);
				setTemplatePage(resp.page ?? 1);
				applyTemplateFiltersFromTemplates(templateItems);
			} else {
				partialErrors.push("流水线模板加载失败");
				setTemplates([]);
				setTemplateTotal(0);
			}

			if (targetsResult.status === "fulfilled") {
				setTargets(targetsResult.value);
				const defaultTarget =
					targetsResult.value.find((target) => target.isDefault) ??
					targetsResult.value[0];
				if (defaultTarget) setSelectedTargetId(defaultTarget.id);
			} else {
				partialErrors.push("运行目标加载失败");
			}

			setError(partialErrors.length > 0 ? partialErrors.join("；") : null);
		} finally {
			setLoading(false);
			refreshInFlightRef.current = false;
		}
	}, [applyTemplateFiltersFromTemplates, resolvedVariant]);

	useEffect(() => {
		void refreshAll();
	}, [refreshAll]);

	useEffect(() => {
		const prev = prevTemplateFiltersRef.current;
		const filtersChanged =
			prev.query !== templateQuery ||
			prev.scope !== templateScope ||
			prev.sort !== templateSort;
		prevTemplateFiltersRef.current = {
			query: templateQuery,
			scope: templateScope,
			sort: templateSort,
		};
		if (!filtersChanged) return;
		void refreshTemplates();
	}, [templateQuery, templateScope, templateSort, refreshTemplates]);

	const loadMoreTemplates = useCallback(async () => {
		try {
			await fetchTemplates(templatePage + 1, true);
		} catch (err) {
			messageApi.error(String(err));
		}
	}, [fetchTemplates, messageApi, templatePage]);

	const handleTemplateSearchChange = useCallback(
		(event: ChangeEvent<HTMLInputElement>) => {
			const value = event.target.value;
			setTemplateQueryDraft(value);
			if (templateSearchDebounceRef.current) {
				clearTimeout(templateSearchDebounceRef.current);
			}
			templateSearchDebounceRef.current = setTimeout(() => {
				setTemplateQuery(value.trim());
				setTemplatePage(1);
			}, 300);
		},
		[],
	);

	const applyTemplateSearch = useCallback(
		(value?: string) => {
			setTemplateQuery((value ?? templateQueryDraft).trim());
			setTemplatePage(1);
		},
		[templateQueryDraft],
	);

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
		setConfigSourceMode("saved");
		setSelectedSavedConfigId(undefined);
		setSavedConfigsError(null);
		setUploadDraftFile(null);
		setInlineDraftName("runtime-config.yaml");
		setInlineDraftContent("");
		setConfigMountPath("/app/configs");
		setConfigTargetFilename("");
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
		setGlobalConfigEnabled(false);
		setAssetPickerResetKey((key) => key + 1);
		if (fileInputRef.current) {
			fileInputRef.current.value = "";
		}
	};

	useEffect(() => {
		if (
			!assetModalOpen ||
			!globalConfigEnabled ||
			configSourceMode !== "saved" ||
			savedConfigsLoading
		) {
			return;
		}
		if (savedConfigs.length > 0) {
			return;
		}
		let cancelled = false;
		setSavedConfigsLoading(true);
		setSavedConfigsError(null);
		void pipelineConfigApi
			.list()
			.then((response) => {
				if (cancelled) return;
				setSavedConfigs(response.items);
			})
			.catch(() => {
				if (cancelled) return;
				setSavedConfigsError("平台配置列表加载失败");
			})
			.finally(() => {
				if (cancelled) return;
				setSavedConfigsLoading(false);
			});
		return () => {
			cancelled = true;
		};
	}, [
		assetModalOpen,
		configSourceMode,
		globalConfigEnabled,
		savedConfigs.length,
		savedConfigsLoading,
	]);

	const handleUploadDraftChange = useCallback(
		async (event: ChangeEvent<HTMLInputElement>) => {
			const file = event.target.files?.[0];
			if (!file) {
				setUploadDraftFile(null);
				return;
			}
			try {
				const content = await file.text();
				setUploadDraftFile({
					name: file.name,
					size: file.size,
					content,
				});
				if (!configTargetFilename.trim()) {
					setConfigTargetFilename(file.name);
				}
			} catch {
				messageApi.error("读取本地文件失败");
			}
		},
		[configTargetFilename, messageApi],
	);

	const handleDeployConfirm = async () => {
		if (!deployTargetId) return;
		const resolved = assetPickerRef.current?.resolveSelectionForRun() ?? {
			assetIds: selectedAssetIds,
		};
		if (resolved.error) {
			messageApi.error(resolved.error);
			return;
		}
		const assetIds = resolved.assetIds;
		const configSelection = globalConfigEnabled
			? buildDeployConfigSelection({
					configSourceMode,
					selectedSavedConfig,
					selectedConfigVersion,
					uploadDraftFile,
					inlineDraftName,
					inlineDraftContent,
					configMountPath,
					configTargetFilename,
				})
			: undefined;
		setDeploying(true);
		try {
			const template = templates.find((item) => item.id === deployTargetId);
			const result = await deployPipelineForAssets(deployTargetId, assetIds, {
				targetId: selectedTargetId,
				version: selectedDeployVersion,
				batchName: template ? `${template.name}-${Date.now()}` : undefined,
				configSelection,
			});
			if (result.mode === "batch") {
				messageApi.success(
					`已创建批量任务，共 ${result.batchJob.totalCount} 个子任务`,
				);
				closeAssetModal();
				navigate(`/pipeline/batch/${result.batchJob.id}`, {
					state: batchJobDetailLocationState(),
				});
			} else {
				messageApi.success(
					result.runs.length > 1
						? `已下发 ${result.runs.length} 个任务`
						: "部署成功",
				);
				closeAssetModal();
				void refreshAll();
			}
			setDeploying(false);
		} catch (err) {
			messageApi.error(`部署失败: ${String(err)}`);
			setDeploying(false);
		}
	};

	const handleDeleteTemplate = async (id: string) => {
		try {
			await deletePipeline(id);
			messageApi.success("已删除流水线模板");
			refreshAll();
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
			const succeeded: string[] = [];
			const failed: { id: string; reason: string }[] = [];
			results.forEach((result, i) => {
				const id = selectedTemplateIds[i];
				if (result.status === "fulfilled") {
					succeeded.push(id);
				} else {
					const reason =
						result.reason instanceof Error
							? result.reason.message
							: String(result.reason);
					failed.push({ id, reason });
				}
			});
			const deletedCount = succeeded.length;
			if (deletedCount > 0) {
				messageApi.success(`已删除 ${deletedCount} 条流水线`);
			}
			if (failed.length > 0) {
				const name = (fid: string) =>
					displayTemplates.find((t) => t.id === fid)?.name ?? fid;
				const detail = failed
					.slice(0, 3)
					.map((f) => `"${name(f.id)}"`)
					.join("、");
				const overflow = failed.length > 3 ? ` 等 ${failed.length} 条` : "";
				messageApi.error(
					`删除失败：${detail}${overflow}（${failed[0].reason}）`,
				);
			}
			setSelectedTemplateIds([]);
			await refreshAll();
		} finally {
			setBulkDeletingTemplates(false);
		}
	};

	const confirmBulkDeleteTemplates = () => {
		if (selectedTemplateIds.length === 0) return;
		const count = selectedTemplateIds.length;
		let confirmInput = "";
		modal.confirm({
			title: `删除选中的 ${count} 条流水线？`,
			content: (
				<div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
					<span>正式版（prod）已从选择中排除。此操作不可恢复。</span>
					<span>
						请输入 <strong>{count}</strong> 以确认删除：
					</span>
					<Input
						placeholder={String(count)}
						onChange={(event) => {
							confirmInput = event.target.value;
						}}
					/>
				</div>
			),
			okText: "删除",
			okButtonProps: { danger: true },
			cancelText: "取消",
			onOk: async () => {
				if (confirmInput.trim() !== String(count)) {
					messageApi.error("确认数字不匹配，已取消删除");
					return Promise.reject(new Error("confirm mismatch"));
				}
				await handleBulkDeleteTemplates();
			},
		});
	};

	const handleEditTemplate = async (id: string) => {
		try {
			const t = await getPipeline(id);
			if (t.scope === "prod") {
				handleViewTemplate(id);
				return;
			}
			if (onEditTemplate) {
				onEditTemplate(t.pipeline);
			} else {
				navigate(`/pipeline?templateId=${encodeURIComponent(id)}&tab=design`);
			}
		} catch (err) {
			messageApi.error(`加载模板失败: ${String(err)}`);
		}
	};

	const handleViewTemplate = (id: string) => {
		navigate(
			`/pipeline?templateId=${encodeURIComponent(id)}&readonly=1&tab=design`,
		);
	};

	const handleVersionHistory = (template: PipelineTemplate) => {
		setVersionDrawerTemplate(template);
		setVersionDrawerOpen(true);
	};

	const handlePromote = (template: PipelineTemplate) => {
		modal.confirm({
			title: `发布 ${template.name} v${template.version} 到正式版？`,
			content:
				"将生成 prod 正式版模板（只读、不可删除）。dev 草稿仍可继续编辑；「活跃版本」仅影响默认运行版本，不等于 prod。",
			okText: "发布",
			cancelText: "取消",
			onOk: async () => {
				try {
					const promoted = await promotePipeline(template.id);
					messageApi.success(
						`已发布 ${template.name} v${promoted.version} 到正式版（prod）`,
					);
					await refreshAll();
				} catch (err) {
					messageApi.error(`发布失败: ${String(err)}`);
				}
			},
		});
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
				onView={handleViewTemplate}
				activeVersion={activeVersionByTemplate[t.name]}
				recommended={t.id === recommendedTemplateId}
				compactActions={options?.compactActions}
				selectable={resolvedVariant === "full" && t.scope !== "prod"}
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
					<Link to="/runs" className="deploy-panel-compact__link">
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
									onClick={() =>
										navigate(
											d.id
												? `/runs/${encodeURIComponent(d.id)}`
												: `/pipeline/executions/${encodeURIComponent(
														d.workflowName,
													)}`,
										)
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

	const autoDraftCount = displayTemplates.filter(isAutoNamedDraft).length;
	const visibleTemplates = hideAutoDrafts
		? displayTemplates.filter((template) => !isAutoNamedDraft(template))
		: displayTemplates;
	const deletableVisibleTemplates = visibleTemplates.filter(
		(template) => template.scope !== "prod",
	);
	const selectableTemplateIds = deletableVisibleTemplates.map(
		(template) => template.id,
	);
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
						onClick={() => navigate("/runs")}
					>
						执行记录
					</Button>
					<Button
						size="small"
						icon={<ReloadOutlined />}
						onClick={refreshAll}
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
							<span>
								请选择要运行的流水线和版本，确认后即可提交运行。
								{recommendedTemplateId ? " 已为你标出一个推荐模板。" : ""}
							</span>
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
				<div className="deploy-section-title">
					<div>
						已保存的流水线
						{!loading ? <span className="count">{templateTotal}</span> : null}
					</div>
					{!loading && displayTemplates.length > 0 ? (
						<Space>
							<Tooltip title="仅选择当前页可删除的 dev 流水线，正式版不会被选中">
								<Checkbox
									checked={allTemplatesSelected}
									indeterminate={someTemplatesSelected}
									onChange={(event) =>
										setSelectedTemplateIds(
											event.target.checked ? selectableTemplateIds : [],
										)
									}
								>
									全选当前页
								</Checkbox>
							</Tooltip>
							<Popconfirm
								title={`删除选中的 ${selectedTemplateIds.length} 条流水线？`}
								description="正式版已排除；将要求输入数量确认"
								okText="继续"
								cancelText="取消"
								disabled={selectedTemplateIds.length === 0}
								onConfirm={confirmBulkDeleteTemplates}
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
				<div
					className="deploy-panel__template-filters"
					style={{
						display: "flex",
						gap: 8,
						flexWrap: "wrap",
						marginBottom: 12,
					}}
				>
					<Input.Search
						allowClear
						placeholder="搜索流水线名称"
						value={templateQueryDraft}
						onChange={handleTemplateSearchChange}
						onSearch={(value) => applyTemplateSearch(value)}
						style={{ width: 220 }}
						data-testid="pipeline-template-search"
					/>
					<Select
						allowClear
						placeholder="范围"
						value={templateScope}
						onChange={(value) => {
							setTemplateScope(value);
							setTemplatePage(1);
						}}
						style={{ width: 120 }}
						options={[
							{ value: "dev", label: "dev" },
							{ value: "prod", label: "prod" },
						]}
						data-testid="pipeline-template-scope"
					/>
					<Select
						value={templateSort}
						onChange={(value) => {
							setTemplateSort(value);
							setTemplatePage(1);
						}}
						style={{ width: 160 }}
						options={[
							{ value: "updated_at_desc", label: "最近更新" },
							{ value: "created_at_desc", label: "最近创建" },
							{ value: "name_asc", label: "名称 A-Z" },
							{ value: "name_desc", label: "名称 Z-A" },
						]}
						data-testid="pipeline-template-sort"
					/>
					<Tooltip title="隐藏形如 pipeline-1782564077864 的自动命名单步草稿，只看正式/已命名流水线">
						<Checkbox
							checked={hideAutoDrafts}
							onChange={(event) => {
								const next = event.target.checked;
								setHideAutoDrafts(next);
								localStorage.setItem(
									"db.pipeline.hideAutoDrafts",
									next ? "1" : "0",
								);
								setTemplatePage(1);
							}}
							data-testid="pipeline-hide-auto-drafts"
							style={{ alignSelf: "center" }}
						>
							隐藏自动命名草稿
							{autoDraftCount > 0 ? ` (${autoDraftCount})` : ""}
						</Checkbox>
					</Tooltip>
				</div>
				<div className="deploy-section">
					{renderTemplateSection(visibleTemplates)}
				</div>
				{displayTemplates.length < templateTotal ? (
					<Button
						type="link"
						size="small"
						className="deploy-panel__load-more"
						loading={loadingMoreTemplates}
						onClick={() => void loadMoreTemplates()}
						data-testid="pipeline-template-load-more"
					>
						加载更多模板（还剩 {templateTotal - displayTemplates.length} 条）
					</Button>
				) : null}
			</div>

			<Modal
				title={selectedAssetIds.length > 0 ? `运行流水线（${selectedAssetIds.length} 个资产）` : "运行流水线（无资产）"}
				open={assetModalOpen}
				onCancel={closeAssetModal}
				onOk={handleDeployConfirm}
				confirmLoading={deploying}
				okText={selectedAssetIds.length > 0 ? "运行资产" : "无资产运行"}
				width={640}
			>
				{deployTargetTemplate?.scope === "prod" ? (
					<Alert
						type="success"
						showIcon
						message="正在运行正式版流水线"
						description="本次运行使用 prod 模板定义，适合生产数据处理；如需修改流程，请先回到 dev 草稿保存并发布。"
						style={{ marginBottom: 16 }}
					/>
				) : null}
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
					<div className="deploy-run-field__label">存储池</div>
					<Select
						aria-label="存储池"
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
							label: target.description ? `${target.name}  ${target.description}` : target.name,
							disabled: target.status !== "available",
						}))}
					/>
				</div>
				<div className="deploy-run-field" data-testid="deploy-config-panel">
					<div className="deploy-run-field__label">高级全局配置（可选）</div>
					<Space direction="vertical" size={12} style={{ width: "100%" }}>
						<Checkbox
							checked={globalConfigEnabled}
							onChange={(event) => setGlobalConfigEnabled(event.target.checked)}
						>
							启用全局配置
						</Checkbox>
						{globalConfigEnabled ? (
							<Space direction="vertical" size={12} style={{ width: "100%" }}>
								<Radio.Group
									value={configSourceMode}
									onChange={(event) =>
										setConfigSourceMode(
											event.target.value as DeployConfigSourceMode,
										)
									}
									optionType="button"
									buttonStyle="solid"
								>
									<Radio.Button value="saved">选择已保存配置</Radio.Button>
									<Radio.Button value="upload">上传本地文件</Radio.Button>
									<Radio.Button value="inline">在线编辑</Radio.Button>
								</Radio.Group>
								{configSourceMode === "saved" ? (
									<Space
										direction="vertical"
										size={8}
										style={{ width: "100%" }}
									>
										<Select
											aria-label="选择已保存配置"
											placeholder="选择平台已保存的配置"
											loading={savedConfigsLoading}
											value={selectedSavedConfigId}
											onChange={(value) => {
												setSelectedSavedConfigId(value);
												const selected = savedConfigs.find(
													(config) => config.id === value,
												);
												if (selected) {
													if (!configTargetFilename.trim()) {
														setConfigTargetFilename(selected.name);
													}
													setSelectedConfigVersion(selected.currentVersion);
													// Fetch versions
													pipelineConfigApi.get(value).then((cfg) => {
														setConfigVersions(cfg.versions || []);
													}).catch(() => {});
												}
											}}
											options={savedConfigs.map((config) => ({
												value: config.id,
												label: `${config.name} · ${config.owner} · ${config.lifecycle} · v${config.currentVersion}`,
											}))}
										/>
										{selectedSavedConfig && configVersions.length > 0 ? (
											<Select
												aria-label="选择配置版本"
												size="small"
												value={selectedConfigVersion}
												onChange={(v) => setSelectedConfigVersion(v)}
												style={{ width: "100%" }}
												options={configVersions.map((ver) => ({
													value: ver.version,
													label: `v${ver.version} · ${ver.status} · ${ver.author} · ${ver.createdAt?.slice(0, 10)}`,
												}))}
											/>
										) : null}
										{savedConfigsError ? (
											<Alert
												type="error"
												showIcon
												message={savedConfigsError}
											/>
										) : null}
										{selectedSavedConfig ? (
											<Alert
												type="info"
												showIcon
												message={selectedSavedConfig.name}
												description={`owner: ${selectedSavedConfig.owner} · ${selectedSavedConfig.description || "无描述"} · 当前版本 v${selectedSavedConfig.currentVersion}`}
											/>
										) : (
											<Alert
												type="warning"
												showIcon
												message="还未选择平台配置"
												description="这里会显示当前用户可用的已保存配置。"
											/>
										)}
									</Space>
								) : null}
								{configSourceMode === "upload" ? (
									<Space
										direction="vertical"
										size={8}
										style={{ width: "100%" }}
									>
										<input
											ref={fileInputRef}
											aria-label="上传配置文件"
											type="file"
											accept=".yaml,.yml,.json,.txt,.conf,.cfg"
											onChange={(event) => {
												void handleUploadDraftChange(event);
											}}
										/>
										{uploadDraftFile ? (
											<Alert
												type="success"
												showIcon
												message={uploadDraftFile.name}
												description={`${formatFileSize(uploadDraftFile.size)} · 本地文件仅作为本次 deploy 草稿，不会自动保存到平台配置库`}
											/>
										) : (
											<Alert
												type="warning"
												showIcon
												message="还未上传本地文件"
												description="选择一个 yaml / json 文件作为本次 deploy 的临时配置。"
											/>
										)}
									</Space>
								) : null}
								{configSourceMode === "inline" ? (
									<Space
										direction="vertical"
										size={8}
										style={{ width: "100%" }}
									>
										<Input
											aria-label="在线编辑文件名"
											placeholder="runtime-config.yaml"
											value={inlineDraftName}
											onChange={(event) => {
												setInlineDraftName(event.target.value);
												if (!configTargetFilename.trim()) {
													setConfigTargetFilename(event.target.value);
												}
											}}
										/>
										<Input.TextArea
											aria-label="在线编辑配置内容"
											rows={8}
											placeholder={"threshold: 0.82\nwindow: 5\n"}
											value={inlineDraftContent}
											onChange={(event) =>
												setInlineDraftContent(event.target.value)
											}
										/>
										<Alert
											type="info"
											showIcon
											message="在线编辑内容只作为本次 deploy 草稿"
											description="这部分内容不会自动写回平台配置库，后续如需沉淀为长期配置，再单独保存到配置中心。"
										/>
									</Space>
								) : null}
								<div
									style={{
										display: "grid",
										gridTemplateColumns: "1fr 1fr",
										gap: 8,
									}}
								>
									<Input
										aria-label="挂载目录"
										placeholder="/app/configs"
										value={configMountPath}
										onChange={(event) => setConfigMountPath(event.target.value)}
									/>
									<Input
										aria-label="目标文件名"
										placeholder="runtime-config.yaml"
										value={configTargetFilename}
										onChange={(event) =>
											setConfigTargetFilename(event.target.value)
										}
									/>
								</div>
								{selectedConfigSummary ? (
									<Alert
										type={
											configMountPath.trim() && configTargetFilename.trim()
												? "success"
												: "warning"
										}
										showIcon
										message={`来源：${selectedConfigSummary.sourceLabel} · ${selectedConfigSummary.name}`}
										description={`摘要：${selectedConfigSummary.meta} · 挂载到 ${configMountPath.trim() || "（未填写目录）"}/${configTargetFilename.trim() || "（未填写文件名）"}`}
									/>
								) : (
									<Alert
										type="warning"
										showIcon
										message="还未完成配置文件选择"
										description="请选择一种文件来源，并补充挂载目录与目标文件名。"
									/>
								)}
							</Space>
						) : null}
					</Space>
				</div>
				<AssetPicker
					ref={assetPickerRef}
					selectedIds={selectedAssetIds}
					onSelectionChange={updateSelectedAssetIds}
					maxHeight={300}
					resetKey={assetPickerResetKey}
				/>
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
