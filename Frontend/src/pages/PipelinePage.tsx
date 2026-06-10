import {
	DeleteOutlined,
	ExportOutlined,
	ImportOutlined,
	PlayCircleOutlined,
	SaveOutlined,
} from "@ant-design/icons";
import {
	FlowEditor,
	FlowEditorProvider,
	type Node,
	useFlowEditor,
} from "@ant-design/pro-flow";
import {
	Alert,
	App,
	Button,
	Collapse,
	Drawer,
	Input,
	Menu,
	Modal,
	message,
	Spin,
	Tabs,
	Tooltip,
	Typography,
} from "antd";

const { TextArea } = Input;

import {
	type DragEvent,
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { assetsApi } from "../api/assets";
import {
	type Deployment,
	deployTemplate,
	getPipeline,
	previewDeploy,
	savePipeline,
} from "../api/pipelineApi";
import type { Asset } from "../api/types";
import ErrorBoundary from "../components/ErrorBoundary";
import AssetPicker from "../components/pipeline/AssetPicker";
import { ComponentPalette } from "../components/pipeline/ComponentPalette";
import { DeployPanel } from "../components/pipeline/DeployPanel";
import { NodeConfigPanel } from "../components/pipeline/NodeConfigPanel";
import { PipelineEmptyState } from "../components/pipeline/PipelineEmptyState";
import { PipelineStepNode } from "../components/pipeline/PipelineNode";
import type {
	Pipeline,
	PipelineNodeData,
	RegisteredComponent,
} from "../components/pipeline/types";
import { usePipelineComponents } from "../hooks/usePipelineComponents";
import { usePipelineKeyboardShortcuts } from "../hooks/usePipelineKeyboardShortcuts";
import {
	fromTranspilerPipeline,
	toTranspilerPipeline,
} from "../lib/pipelineContract";
import { ComponentManager } from "./ComponentManager";
import {
	apiToRegistered,
	createPipelineNode,
	dedupeComponentsByName,
	extractAssetSizeBytes,
	extractNodeAssetIds,
	extractPipelineAssetIds,
	formatBytes,
	type PipelineFlowEdge,
	type PipelineFlowNode,
	parseAssetIds,
	parseAssetName,
	parseAssetType,
	toRecord,
} from "./pipeline/pipelinePageHelpers";
import { WorkflowExecutionList } from "./WorkflowExecutionList";

import "../styles/pipeline.css";

const PIPELINE_NODE_TYPES = { pipelineStep: PipelineStepNode };

type DeployMode = "edit" | "preview";

type CanvasMenuState = {
	open: boolean;
	x: number;
	y: number;
	node: PipelineFlowNode | null;
};

function PipelineCanvas() {
	const navigate = useNavigate();
	const [searchParams] = useSearchParams();
	const wrapperRef = useRef<HTMLDivElement>(null);
	const editor = useFlowEditor();
	const { modal } = App.useApp();
	const queryAssetIds = useMemo(
		() => parseAssetIds(searchParams.get("asset_ids")),
		[searchParams],
	);
	const [nodes, setNodes] = useState<PipelineFlowNode[]>([]);
	const [edges, setEdges] = useState<PipelineFlowEdge[]>([]);
	const [pipelineName, setPipelineName] = useState("my-pipeline");
	const [selectedNode, setSelectedNode] = useState<PipelineFlowNode | null>(
		null,
	);
	const [contextMenu, setContextMenu] = useState<CanvasMenuState>({
		open: false,
		x: 0,
		y: 0,
		node: null,
	});
	const [editingNodeId, setEditingNodeId] = useState<string | null>(null);
	const {
		components: registeredComponents,
		loading: componentsLoading,
		error: componentsError,
		reload: reloadComponents,
	} = usePipelineComponents(apiToRegistered, dedupeComponentsByName);
	const [templateLoading, setTemplateLoading] = useState(false);
	const [savedDrawerOpen, setSavedDrawerOpen] = useState(false);
	const templateId = useMemo(
		() => searchParams.get("templateId") || null,
		[searchParams],
	);
	const [deployDialog, setDeployDialog] = useState<{
		open: boolean;
		deploying: boolean;
		done: boolean;
		name: string;
		mode: DeployMode;
		result?: Deployment;
		error?: string;
		previewManifest?: string;
		previewLoading?: boolean;
		previewError?: string;
	}>({
		open: false,
		deploying: false,
		done: false,
		name: "",
		mode: "edit",
	});
	const [jsonOutput, setJsonOutput] = useState<string | null>(null);
	const [importModalOpen, setImportModalOpen] = useState(false);
	const [importText, setImportText] = useState("");
	const importTextRef = useRef<string>("");
	const [templateRefreshKey, setTemplateRefreshKey] = useState(0);
	// Asset selection for deploy modal
	const [selectedAssetIds, setSelectedAssetIds] =
		useState<string[]>(queryAssetIds);
	const [selectedNodeAsset, setSelectedNodeAsset] = useState<Asset | null>(
		null,
	);
	const [selectedNodeAssetLoading, setSelectedNodeAssetLoading] =
		useState(false);
	const [selectedNodeAssetError, setSelectedNodeAssetError] = useState<
		string | null
	>(null);

	const flattenNodes = useMemo(() => toRecord(nodes), [nodes]);
	const flattenEdges = useMemo(() => toRecord(edges), [edges]);
	const nodeTypes = useMemo(() => PIPELINE_NODE_TYPES, []);

	useEffect(() => {
		setSelectedAssetIds(queryAssetIds);
	}, [queryAssetIds]);

	useEffect(() => {
		setSelectedNode((prev: PipelineFlowNode | null) => {
			if (!prev) return null;
			return nodes.find((node) => node.id === prev.id) ?? null;
		});
	}, [nodes]);

	const loadPipelineToCanvas = useCallback(
		(pipeline: Pipeline) => {
			const { nodes: n, edges: e } = fromTranspilerPipeline(pipeline);
			setNodes(n);
			setEdges(e);
			setEditingNodeId(null);
			if (pipeline.name) setPipelineName(pipeline.name);
			setSelectedNode(null);
			editor.deselectAll();
			setJsonOutput(null);
		},
		[editor],
	);

	const loadPipelineFromSessionStorage = useCallback(() => {
		const raw = sessionStorage.getItem("pipeline-edit");
		if (!raw) return;
		sessionStorage.removeItem("pipeline-edit");
		try {
			loadPipelineToCanvas(JSON.parse(raw));
		} catch {
			/* ignore */
		}
	}, [loadPipelineToCanvas]);

	useEffect(() => {
		if (!templateId) {
			loadPipelineFromSessionStorage();
			return;
		}

		let cancelled = false;
		setTemplateLoading(true);
		getPipeline(templateId)
			.then((template) => {
				if (cancelled) return;
				loadPipelineToCanvas(template.pipeline);
			})
			.catch((err) => {
				if (cancelled) return;
				message.error(`模板加载失败: ${String(err)}`);
				loadPipelineFromSessionStorage();
			})
			.finally(() => {
				if (!cancelled) setTemplateLoading(false);
			});

		return () => {
			cancelled = true;
		};
	}, [loadPipelineFromSessionStorage, loadPipelineToCanvas, templateId]);

	const onDragStart = useCallback((e: DragEvent, comp: RegisteredComponent) => {
		e.dataTransfer.setData("application/reactflow", JSON.stringify(comp));
		e.dataTransfer.effectAllowed = "move";
	}, []);

	const onDragOver = useCallback((event: DragEvent) => {
		event.preventDefault();
		event.dataTransfer.dropEffect = "move";
	}, []);

	const onDrop = useCallback(
		(event: DragEvent) => {
			event.preventDefault();
			const raw = event.dataTransfer.getData("application/reactflow");
			if (!raw) return;
			try {
				const comp: RegisteredComponent = JSON.parse(raw);
				const bounds = wrapperRef.current?.getBoundingClientRect();
				if (!bounds) return;
				const position = editor.screenToFlowPosition({
					x: event.clientX,
					y: event.clientY,
				});
				const newNode = createPipelineNode(comp, position.x, position.y);
				editor.addNode(newNode);
				editor.selectElements([newNode.id]);
				setSelectedNode(newNode);
				setEditingNodeId(null);
			} catch {
				/* ignore */
			}
		},
		[editor],
	);

	const selectNodeWithEdges = useCallback(
		(node: PipelineFlowNode) => {
			const connectedEdgeIds = edges
				.filter((edge) => edge.source === node.id || edge.target === node.id)
				.map((edge) => edge.id);
			setSelectedNode(node);
			editor.selectElements([node.id, ...connectedEdgeIds]);
		},
		[edges, editor],
	);

	const onNodeClick = useCallback(
		(event: React.MouseEvent, node: Node) => {
			setContextMenu((prev) => ({ ...prev, open: false }));
			const pipelineNode = node as PipelineFlowNode;
			selectNodeWithEdges(pipelineNode);
			if (event.detail > 1) {
				setEditingNodeId(pipelineNode.id);
			}
		},
		[selectNodeWithEdges],
	);
	const onPaneClick = useCallback(() => {
		setSelectedNode(null);
		setContextMenu((prev) => ({ ...prev, open: false }));
		setEditingNodeId(null);
		editor.deselectAll();
	}, [editor]);

	const onPaneContextMenu = useCallback(
		(event: React.MouseEvent) => {
			event.preventDefault();
			setSelectedNode(null);
			setEditingNodeId(null);
			editor.deselectAll();
			setContextMenu({
				open: true,
				x: event.clientX,
				y: event.clientY,
				node: null,
			});
		},
		[editor],
	);

	const onNodeContextMenu = useCallback(
		(event: React.MouseEvent, node: Node) => {
			event.preventDefault();
			event.stopPropagation();
			const pipelineNode = node as PipelineFlowNode;
			selectNodeWithEdges(pipelineNode);
			setContextMenu({
				open: true,
				x: event.clientX,
				y: event.clientY,
				node: pipelineNode,
			});
		},
		[selectNodeWithEdges],
	);

	const updateNodeData = useCallback(
		(id: string, data: Record<string, unknown>) => {
			editor.updateNodeData(id, data);
			setSelectedNode((prev: PipelineFlowNode | null) =>
				prev?.id === id ? { ...prev, data: { ...prev.data, ...data } } : prev,
			);
		},
		[editor],
	);

	const closeNodeConfig = useCallback(() => {
		setEditingNodeId(null);
	}, []);

	const saveNodeConfig = useCallback(
		(id: string, data: Partial<PipelineNodeData>) => {
			updateNodeData(id, data as Record<string, unknown>);
			closeNodeConfig();
		},
		[closeNodeConfig, updateNodeData],
	);

	const editingNode = useMemo(
		() => nodes.find((node) => node.id === editingNodeId) ?? null,
		[nodes, editingNodeId],
	);

	const handleContextMenuClick = useCallback(
		({ key }: { key: string }) => {
			const menuNode = contextMenu.node;
			setContextMenu((prev) => ({ ...prev, open: false }));

			if (menuNode) {
				if (key === "configure") {
					selectNodeWithEdges(menuNode);
					setEditingNodeId(menuNode.id);
					return;
				}
				if (key === "copy") {
					editor.selectElements([menuNode.id]);
					window.setTimeout(() => {
						void editor
							.copySelection()
							.catch(() => message.warning("浏览器未允许读取剪贴板"));
					}, 0);
					return;
				}
				if (key === "delete") {
					editor.selectElements([menuNode.id]);
					window.setTimeout(() => {
						editor.deleteSelection();
						setSelectedNode((prev: PipelineFlowNode | null) =>
							prev?.id === menuNode.id ? null : prev,
						);
						setEditingNodeId((prev) => (prev === menuNode.id ? null : prev));
					}, 0);
					return;
				}
			}

			if (key === "paste") {
				void editor
					.paste()
					.catch(() => message.warning("浏览器未允许读取剪贴板"));
			}
			if (key === "selectAll") {
				editor.selectAll();
			}
			if (key === "zoomIn") {
				editor.reactflow?.zoomIn();
			}
			if (key === "zoomOut") {
				editor.reactflow?.zoomOut();
			}
			if (key === "fitView") {
				editor.reactflow?.fitView();
			}
		},
		[contextMenu.node, editor, selectNodeWithEdges],
	);

	const buildPipelineJSON = useCallback(
		(): Pipeline => toTranspilerPipeline(nodes, edges, { name: pipelineName }),
		[nodes, edges, pipelineName],
	);

	const pipelineAssetIds = useMemo(
		() => extractPipelineAssetIds(buildPipelineJSON()),
		[buildPipelineJSON],
	);

	const selectedNodeAssetId = useMemo(() => {
		const nodeAssetIds = extractNodeAssetIds(selectedNode);
		if (nodeAssetIds.length > 0) {
			return nodeAssetIds[0];
		}
		if (pipelineAssetIds.length > 0) {
			return pipelineAssetIds[0];
		}
		return selectedAssetIds[0] ?? "";
	}, [pipelineAssetIds, selectedAssetIds, selectedNode]);

	useEffect(() => {
		if (!selectedNodeAssetId) {
			setSelectedNodeAsset(null);
			setSelectedNodeAssetError(null);
			setSelectedNodeAssetLoading(false);
			return;
		}

		let alive = true;
		setSelectedNodeAssetLoading(true);
		setSelectedNodeAssetError(null);

		assetsApi
			.get(selectedNodeAssetId)
			.then((asset) => {
				if (!alive) return;
				setSelectedNodeAsset(asset);
			})
			.catch(() => {
				if (!alive) return;
				setSelectedNodeAsset(null);
				setSelectedNodeAssetError("加载关联资产失败");
			})
			.finally(() => {
				if (!alive) return;
				setSelectedNodeAssetLoading(false);
			});

		return () => {
			alive = false;
		};
	}, [selectedNodeAssetId]);

	const selectedNodeAssetInfo = useMemo(() => {
		if (!selectedNodeAsset) {
			return null;
		}

		const raw = selectedNodeAsset as unknown as Record<string, unknown>;
		return {
			name: parseAssetName(raw, selectedNodeAsset.asset_id),
			type: parseAssetType(raw, selectedNodeAsset.type ?? "—"),
			size: formatBytes(extractAssetSizeBytes(raw)),
		};
	}, [selectedNodeAsset]);

	const exportPipeline = useCallback(() => {
		setJsonOutput(JSON.stringify(buildPipelineJSON(), null, 2));
	}, [buildPipelineJSON]);

	const importPipeline = useCallback(() => {
		setImportText("");
		importTextRef.current = "";
		setImportModalOpen(true);
	}, []);

	const applyImportedPipeline = useCallback(() => {
		const text =
			(
				document.querySelector(
					".ant-modal textarea",
				) as HTMLTextAreaElement | null
			)?.value?.trim() || importTextRef.current.trim();
		if (!text) {
			message.warning("请粘贴 Pipeline JSON");
			return;
		}
		try {
			const pipeline: Pipeline = JSON.parse(text);
			const { nodes: importedNodes, edges: importedEdges } =
				fromTranspilerPipeline(pipeline);
			setNodes(importedNodes);
			setEdges(importedEdges);
			setEditingNodeId(null);
			if (pipeline.name) {
				setPipelineName(pipeline.name);
			}
			setJsonOutput(null);
			setImportModalOpen(false);
			setImportText("");
			message.success("导入成功");
		} catch {
			message.error("无效的 JSON");
		}
	}, []);

	const clearCanvas = useCallback(() => {
		modal.confirm({
			title: "清空画布",
			content: "将删除当前所有节点与连线，此操作不可撤销。",
			okText: "清空",
			okType: "danger",
			cancelText: "取消",
			onOk: () => {
				setNodes([]);
				setEdges([]);
				setSelectedNode(null);
				setEditingNodeId(null);
				editor.deselectAll();
				setJsonOutput(null);
			},
		});
	}, [editor, modal.confirm]);

	const handleSave = useCallback(async () => {
		try {
			await savePipeline(pipelineName, buildPipelineJSON());
			message.success("已保存，可在右侧「已保存」查看");
			setTemplateRefreshKey((k) => k + 1);
		} catch (err) {
			message.error(`保存失败: ${String(err)}`);
		}
	}, [pipelineName, buildPipelineJSON]);

	const canDeploy = nodes.length > 0;

	const openDeployDialog = useCallback(() => {
		if (nodes.length === 0) {
			message.warning("请先从左侧拖入至少一个组件到画布");
			return;
		}
		setDeployDialog({
			open: true,
			deploying: false,
			done: false,
			mode: "edit",
			name: pipelineName,
		});
		setSelectedAssetIds([]);
	}, [pipelineName, nodes.length]);

	const closeDeployDialog = useCallback(() => {
		setDeployDialog({
			open: false,
			deploying: false,
			done: false,
			name: "",
			mode: "edit",
		});
	}, []);

	usePipelineKeyboardShortcuts({
		onSave: handleSave,
		onDeploy: openDeployDialog,
		onClear: clearCanvas,
		onCloseModal: () => {
			if (deployDialog.open) closeDeployDialog();
			if (importModalOpen) {
				setImportModalOpen(false);
				setImportText("");
			}
			if (savedDrawerOpen) setSavedDrawerOpen(false);
		},
		canDeploy,
		modalOpen: deployDialog.open || importModalOpen || savedDrawerOpen,
	});

	const deployDialogAssetId = useMemo(() => {
		if (!deployDialog.result) return "";
		return (
			extractPipelineAssetIds(deployDialog.result.pipelineJSON ?? {})[0] ?? ""
		);
	}, [deployDialog.result]);

	const handleDeploy = useCallback(async () => {
		setDeployDialog((prev) => ({ ...prev, deploying: true, done: false }));
		try {
			const pipeline = buildPipelineJSON();
			const name = deployDialog.name || pipelineName;
			const saved = await savePipeline(name, pipeline);
			const result = await deployTemplate(
				saved.id,
				selectedAssetIds.length > 0 ? selectedAssetIds : undefined,
			);
			setDeployDialog((prev) => ({
				...prev,
				deploying: false,
				done: true,
				result,
			}));
		} catch (err) {
			setDeployDialog((prev) => ({
				...prev,
				deploying: false,
				done: true,
				error: String(err),
			}));
		}
	}, [buildPipelineJSON, deployDialog.name, pipelineName, selectedAssetIds]);

	const handlePreviewDeploy = useCallback(async () => {
		setDeployDialog((prev) => ({
			...prev,
			previewLoading: true,
			previewError: undefined,
			previewManifest: undefined,
			mode: "preview",
		}));
		try {
			const pipeline = buildPipelineJSON();
			const { manifest } = await previewDeploy(pipeline);
			setDeployDialog((prev) => ({
				...prev,
				previewLoading: false,
				previewManifest: manifest,
			}));
		} catch (err) {
			setDeployDialog((prev) => ({
				...prev,
				previewLoading: false,
				previewError: String(err),
			}));
		}
	}, [buildPipelineJSON]);

	const isCanvasEmpty = nodes.length === 0;
	const currentTemplateLabel = pipelineName || "未命名流水线";

	return (
		<div className="pipeline-page">
			{/* Header */}
			<div className="pipeline-toolbar">
				<div className="pipeline-toolbar__left">
					<Typography.Title level={5} className="pipeline-toolbar__title">
						流水线设计
					</Typography.Title>
				</div>

				<div className="pipeline-toolbar__name">
					<span className="pipeline-toolbar__name-label">名称</span>
					<Input
						id="pipeline-name-input"
						name="pipelineName"
						value={pipelineName}
						onChange={(e) => setPipelineName(e.target.value)}
						placeholder="输入流水线名称"
						aria-label="流水线名称"
						className="pipeline-toolbar__name-input"
						size="small"
					/>
				</div>
				<div className="pipeline-toolbar__actions">
					<Tooltip
						title={
							canDeploy
								? "保存并部署为 Argo Workflow (⌘/Ctrl+D)"
								: "请先从左侧拖入至少一个组件到画布"
						}
					>
						<span>
							<Button
								size="small"
								type="primary"
								icon={<PlayCircleOutlined />}
								onClick={openDeployDialog}
								disabled={!canDeploy}
							>
								部署
							</Button>
						</span>
					</Tooltip>
					<Tooltip title="保存 (⌘/Ctrl+S)">
						<Button size="small" icon={<SaveOutlined />} onClick={handleSave}>
							保存
						</Button>
					</Tooltip>
					<Button
						size="small"
						icon={<ExportOutlined />}
						onClick={exportPipeline}
					>
						导出
					</Button>
					<Button
						size="small"
						icon={<ImportOutlined />}
						onClick={importPipeline}
					>
						导入
					</Button>
					<Button
						size="small"
						danger
						icon={<DeleteOutlined />}
						onClick={clearCanvas}
					>
						清空
					</Button>
				</div>
			</div>

			{/* Body */}
			<div className="pipeline-body">
				<ComponentPalette
					components={registeredComponents}
					onDragStart={onDragStart}
					loading={componentsLoading}
					error={componentsError}
					onRetry={reloadComponents}
				/>
				<div
					className="canvas-wrapper"
					ref={wrapperRef}
					style={{ flex: 1, height: "100%", position: "relative" }}
					role="application"
					aria-label="流水线画布"
				>
					{templateLoading ? (
						<div className="pipeline-canvas-loading" aria-busy="true">
							<Spin tip="正在加载模板..." />
						</div>
					) : null}
					{isCanvasEmpty && !templateLoading ? (
						<PipelineEmptyState
							variant="canvas"
							title="拖入组件开始设计"
							hint="从左侧拖入步骤，连线后点「保存」；已保存的在右侧查看"
						/>
					) : null}
					<ErrorBoundary
						title="画布渲染错误"
						subTitle="流水线画布出现异常，可重试或刷新页面"
					>
						<FlowEditor
							nodeTypes={nodeTypes}
							flattenNodes={flattenNodes}
							flattenEdges={flattenEdges}
							onFlattenNodesChange={(nextNodes: Record<string, unknown>) =>
								setNodes(Object.values(nextNodes) as PipelineFlowNode[])
							}
							onFlattenEdgesChange={(nextEdges: Record<string, unknown>) =>
								setEdges(Object.values(nextEdges) as PipelineFlowEdge[])
							}
							contextMenuEnabled={false}
							flowProps={{
								onDrop,
								onDragOver,
								onNodeClick,
								onNodeContextMenu,
								onPaneClick,
								onPaneContextMenu,
								onlyRenderVisibleElements: true,
								minZoom: 0.2,
								maxZoom: 2,
								proOptions: { hideAttribution: true },
							}}
						/>
					</ErrorBoundary>
					{contextMenu.open && (
						<div
							className="pipeline-context-menu"
							style={{
								left: contextMenu.x,
								top: contextMenu.y,
							}}
						>
							<Menu
								selectable={false}
								onClick={handleContextMenuClick}
								items={
									contextMenu.node
										? [
												{ key: "configure", label: "配置节点" },
												{ key: "copy", label: "复制节点" },
												{ type: "divider" },
												{
													key: "delete",
													label: "删除节点",
													danger: true,
												},
											]
										: [
												{ key: "paste", label: "粘贴" },
												{ key: "selectAll", label: "选择全部" },
												{ type: "divider" },
												{ key: "zoomIn", label: "放大" },
												{ key: "zoomOut", label: "缩小" },
												{ key: "fitView", label: "适应画布" },
											]
								}
							/>
						</div>
					)}
				</div>
				<aside className="config-panel">
					<div
						className="config-panel__saved"
						data-testid="saved-pipelines-panel"
					>
						<DeployPanel
							variant="sidebar"
							_refreshKey={templateRefreshKey}
							onEditTemplate={loadPipelineToCanvas}
							onViewAll={() => setSavedDrawerOpen(true)}
						/>
					</div>
					<div className="config-panel__node">
						{selectedNode ? (
							<>
								<div className="config-panel-header">
									<div className="config-panel-header__info">
										<span className="config-panel-label">
											{selectedNode.data?.label || selectedNode.id}
										</span>
										<span className="config-panel-type">
											{selectedNode.data?.image || "未设置镜像"}
										</span>
									</div>
									<Button
										size="small"
										type="primary"
										onClick={() => setEditingNodeId(selectedNode.id)}
									>
										配置节点
									</Button>
								</div>
								<div className="config-content">
									<div className="config-section-title">关联资产</div>
									{selectedNodeAssetLoading ? (
										<div
											style={{
												fontSize: 12,
												color: "#64748b",
											}}
										>
											{selectedNodeAssetId
												? "正在加载关联资产..."
												: "未关联资产"}
										</div>
									) : selectedNodeAssetError ? (
										<div
											style={{
												fontSize: 12,
												color: "#dc2626",
											}}
										>
											{selectedNodeAssetError}
										</div>
									) : selectedNodeAssetInfo ? (
										<div
											style={{
												display: "grid",
												gap: 6,
											}}
										>
											<div className="config-field">
												<span
													style={{
														fontSize: 10,
														color: "var(--color-text-secondary, #64748b)",
														textTransform: "uppercase",
														letterSpacing: "0.8px",
													}}
												>
													名称
												</span>
												<Typography.Text>
													{selectedNodeAssetInfo.name}
												</Typography.Text>
											</div>
											<div className="config-field">
												<span
													style={{
														fontSize: 10,
														color: "var(--color-text-secondary, #64748b)",
														textTransform: "uppercase",
														letterSpacing: "0.8px",
													}}
												>
													类型
												</span>
												<Typography.Text>
													{selectedNodeAssetInfo.type}
												</Typography.Text>
											</div>
											<div className="config-field">
												<span
													style={{
														fontSize: 10,
														color: "var(--color-text-secondary, #64748b)",
														textTransform: "uppercase",
														letterSpacing: "0.8px",
													}}
												>
													大小
												</span>
												<Typography.Text>
													{selectedNodeAssetInfo.size}
												</Typography.Text>
											</div>
										</div>
									) : (
										<div
											style={{
												fontSize: 12,
												color: "#64748b",
											}}
										>
											未选择关联资产
										</div>
									)}
								</div>
							</>
						) : (
							<PipelineEmptyState
								variant="config"
								title="节点配置"
								description="选中画布节点后，在此查看详情。"
							/>
						)}
					</div>
				</aside>
				{editingNode && (
					<NodeConfigPanel
						open={Boolean(editingNode)}
						node={editingNode}
						onCancel={closeNodeConfig}
						onSave={saveNodeConfig}
					/>
				)}
			</div>

			<Drawer
				title="已保存与运行记录"
				open={savedDrawerOpen}
				onClose={() => setSavedDrawerOpen(false)}
				width={720}
				destroyOnHidden
			>
				<DeployPanel
					variant="full"
					_refreshKey={templateRefreshKey}
					onEditTemplate={(pipeline) => {
						loadPipelineToCanvas(pipeline);
						setSavedDrawerOpen(false);
					}}
				/>
			</Drawer>

			{/* JSON Output */}
			{jsonOutput && <pre className="json-output">{jsonOutput}</pre>}

			{/* Deploy Dialog */}
			<Modal
				open={importModalOpen}
				title="导入流水线"
				okText="导入"
				cancelText="取消"
				onOk={applyImportedPipeline}
				onCancel={() => {
					setImportModalOpen(false);
					setImportText("");
				}}
				destroyOnHidden
			>
				<p style={{ marginTop: 0, color: "#64748b", fontSize: 13 }}>
					粘贴 Pipeline JSON，将替换当前画布内容。
				</p>
				<TextArea
					value={importText}
					onChange={(event) => {
						setImportText(event.target.value);
						importTextRef.current = event.target.value;
					}}
					placeholder='{"name":"my-pipeline","nodes":[],"edges":[]}'
					autoSize={{ minRows: 10, maxRows: 18 }}
					style={{
						fontFamily: '"SF Mono", "Fira Code", monospace',
						fontSize: 12,
					}}
				/>
			</Modal>

			<Modal
				title="部署流水线"
				open={deployDialog.open}
				onCancel={closeDeployDialog}
				footer={null}
				width={780}
				destroyOnHidden
			>
				{!deployDialog.deploying &&
					!deployDialog.done &&
					deployDialog.mode === "edit" && (
						<>
							<Typography.Text
								type="secondary"
								style={{ fontSize: 12, display: "block", marginBottom: 4 }}
							>
								当前模板：{currentTemplateLabel}
							</Typography.Text>
							<Typography.Paragraph
								type="secondary"
								style={{ fontSize: 12, marginBottom: 16 }}
							>
								将流水线转换为 Argo Workflow 并提交到 Kubernetes 集群。
							</Typography.Paragraph>
							{!canDeploy && (
								<Alert
									type="warning"
									showIcon
									message="画布为空"
									description="请先从左侧拖入至少一个组件，或导入带节点的 Pipeline JSON。"
									style={{ marginBottom: 16 }}
								/>
							)}
							<div
								style={{
									display: "flex",
									flexDirection: "column",
									gap: 12,
									marginBottom: 16,
								}}
							>
								<label
									htmlFor="pp-workflow-name"
									style={{
										display: "flex",
										flexDirection: "column",
										gap: 4,
										fontSize: 10,
										textTransform: "uppercase",
										letterSpacing: "0.8px",
										color: "#64748b",
									}}
								>
									工作流名称
									<Input
										id="pp-workflow-name"
										value={deployDialog.name}
										onChange={(e) =>
											setDeployDialog((prev) => ({
												...prev,
												name: e.target.value,
											}))
										}
										placeholder="留空则使用当前流水线名称"
										size="small"
									/>
								</label>
								<div
									style={{
										fontSize: 10,
										color: "#64748b",
										display: "flex",
										gap: 8,
									}}
								>
									<span>{nodes.length} 个节点</span>
									<span style={{ fontSize: 3, color: "#cbd5e1" }}>•</span>
									<span>{edges.length} 条连线</span>
								</div>
								<Collapse
									ghost
									size="small"
									items={[
										{
											key: "assets",
											label: (
												<span style={{ fontSize: 12, color: "#64748b" }}>
													高级：绑定资产（可选）
												</span>
											),
											children: (
												<AssetPicker
													selectedIds={selectedAssetIds}
													onSelectionChange={setSelectedAssetIds}
													maxHeight={180}
												/>
											),
										},
									]}
								/>
							</div>
							<div
								style={{
									display: "flex",
									gap: 8,
									justifyContent: "flex-end",
									borderTop: "1px solid var(--color-border, #e2e8f0)",
									paddingTop: 14,
								}}
							>
								<Button onClick={closeDeployDialog}>取消</Button>
								<Button onClick={handlePreviewDeploy} disabled={!canDeploy}>
									预览
								</Button>
								<Button
									type="primary"
									onClick={handleDeploy}
									disabled={!canDeploy}
								>
									部署
								</Button>
							</div>
						</>
					)}
				{!deployDialog.deploying &&
					!deployDialog.done &&
					deployDialog.mode === "preview" && (
						<>
							<Typography.Text
								type="secondary"
								style={{ fontSize: 12, display: "block", marginBottom: 4 }}
							>
								当前模板：{currentTemplateLabel}
							</Typography.Text>
							<Typography.Paragraph
								type="secondary"
								style={{ fontSize: 12, marginBottom: 12 }}
							>
								以下为 dry-run 结果，仅用于确认。
							</Typography.Paragraph>
							{deployDialog.previewLoading ? (
								<div style={{ textAlign: "center", padding: 20 }}>
									<Typography.Text type="secondary">
										正在生成预览内容...
									</Typography.Text>
								</div>
							) : deployDialog.previewError ? (
								<Alert
									type="error"
									showIcon
									message="预览失败"
									description={deployDialog.previewError}
									style={{ marginBottom: 16 }}
								/>
							) : (
								<pre
									style={{
										margin: 0,
										padding: 12,
										background: "#0f172a",
										color: "#e2e8f0",
										borderRadius: 8,
										overflow: "auto",
										maxHeight: 360,
										fontSize: 12,
										fontFamily:
											'"SF Mono", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
										lineHeight: 1.5,
										whiteSpace: "pre",
									}}
								>
									{deployDialog.previewManifest || "（暂无内容）"}
								</pre>
							)}
							<div
								style={{
									display: "flex",
									gap: 8,
									justifyContent: "flex-end",
									borderTop: "1px solid var(--color-border, #e2e8f0)",
									paddingTop: 14,
									marginTop: 12,
								}}
							>
								<Button
									onClick={() =>
										setDeployDialog((prev) => ({ ...prev, mode: "edit" }))
									}
								>
									返回编辑
								</Button>
								<Button
									type="primary"
									onClick={handleDeploy}
									disabled={!canDeploy}
								>
									确认部署
								</Button>
							</div>
						</>
					)}
				{deployDialog.deploying && (
					<div style={{ textAlign: "center", padding: 20 }}>
						<Typography.Text type="secondary">
							正在部署流水线...
						</Typography.Text>
					</div>
				)}
				{deployDialog.done && (
					<div style={{ textAlign: "center", padding: 20 }}>
						{deployDialog.error ? (
							<>
								<Typography.Text type="danger" strong>
									部署失败
								</Typography.Text>
								<pre
									style={{
										fontSize: 11,
										color: "#ef4444",
										marginTop: 8,
										whiteSpace: "pre-wrap",
									}}
								>
									{deployDialog.error}
								</pre>
							</>
						) : deployDialog.result ? (
							<>
								<Typography.Text type="secondary" style={{ fontSize: 12 }}>
									当前模板：{currentTemplateLabel}
								</Typography.Text>
								<br />
								<Typography.Text type="success" strong>
									部署成功
								</Typography.Text>
								<p
									style={{
										fontFamily: '"SF Mono",monospace',
										fontSize: 12,
										color: "#64748b",
										marginTop: 8,
									}}
								>
									{deployDialog.result.workflowName}
								</p>
								{deployDialogAssetId ? (
									<Button
										type="link"
										onClick={() => navigate(`/assets/${deployDialogAssetId}`)}
									>
										查看关联资产
									</Button>
								) : null}
								<div
									style={{
										fontSize: 10,
										color: "#64748b",
										display: "flex",
										gap: 8,
										justifyContent: "center",
										marginTop: 8,
									}}
								>
									<span>{deployDialog.result.nodeCount} 个节点</span>
									<span style={{ fontSize: 3, color: "#cbd5e1" }}>•</span>
									<span>
										{new Date(deployDialog.result.createdAt).toLocaleString()}
									</span>
								</div>
								<div
									style={{
										marginTop: 16,
										display: "flex",
										gap: 8,
										justifyContent: "center",
									}}
								>
									<Button
										type="primary"
										onClick={() => {
											closeDeployDialog();
											navigate(
												`/workflows/${deployDialog.result?.workflowName}`,
											);
										}}
									>
										查看 Workflow
									</Button>
									<Button
										onClick={() => {
											closeDeployDialog();
											setSavedDrawerOpen(true);
										}}
									>
										查看记录
									</Button>
									<Button onClick={closeDeployDialog}>关闭</Button>
								</div>
							</>
						) : null}
					</div>
				)}
			</Modal>
		</div>
	);
}

type PipelineTab = "design" | "executions" | "components";

function resolvePipelineTab(raw: string | null): PipelineTab {
	switch (raw) {
		case "executions":
			return "executions";
		case "components":
			return "components";
		default:
			return "design";
	}
}

export default function PipelinePage() {
	const [searchParams, setSearchParams] = useSearchParams();
	const activeTab = useMemo(
		() => resolvePipelineTab(searchParams.get("tab")),
		[searchParams],
	);
	const onTabChange = useCallback(
		(nextTab: string) => {
			const next = new URLSearchParams(searchParams);
			if (nextTab === "design") {
				next.delete("tab");
			} else {
				next.set("tab", nextTab);
			}
			setSearchParams(next, { replace: true });
		},
		[searchParams, setSearchParams],
	);

	const tabLabel = useCallback((title: string, subtitle: string) => {
		return (
			<div className="pipeline-tab-label">
				<span className="pipeline-tab-label__title">{title}</span>
				<span className="pipeline-tab-label__subtitle">{subtitle}</span>
			</div>
		);
	}, []);

	return (
		<div className="pipeline-tabs-page">
			<Tabs
				activeKey={activeTab}
				onChange={onTabChange}
				destroyOnHidden={false}
				animated={{ inkBar: true, tabPane: true }}
				items={[
					{
						key: "design",
						label: tabLabel("设计", "编辑流水线画布"),
						children: (
							<div className="pipeline-tab-content pipeline-tab-content--canvas">
								<ErrorBoundary
									title="流水线设计错误"
									subTitle="设计画布加载失败，可重试或刷新页面"
								>
									<FlowEditorProvider>
										<PipelineCanvas />
									</FlowEditorProvider>
								</ErrorBoundary>
							</div>
						),
					},
					{
						key: "executions",
						label: tabLabel("执行记录", "查看和管理流水线运行"),
						children: (
							<div className="pipeline-tab-content pipeline-tab-content--panel">
								<WorkflowExecutionList active={activeTab === "executions"} />
							</div>
						),
					},
					{
						key: "components",
						label: tabLabel("组件", "管理可复用的步骤定义"),
						children: (
							<div className="pipeline-tab-content pipeline-tab-content--panel">
								<ComponentManager />
							</div>
						),
					},
				]}
			/>
		</div>
	);
}
