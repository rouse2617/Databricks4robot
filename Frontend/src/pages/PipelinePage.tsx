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
	Input,
	Menu,
	Modal,
	message,
	Select,
	Space,
	Spin,
	Tabs,
	Tooltip,
	Typography,
} from "antd";

const { TextArea } = Input;

import {
	type DragEvent,
	type FocusEvent,
	type MutableRefObject,
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
	type ExecutionTarget,
	getPipeline,
	listExecutionTargets,
	listPipelineVersions,
	type PipelineTemplate,
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

function replaceAppendedValue(previous: string, next: string) {
	if (!previous || next === previous) return next;
	if (next.startsWith(previous) && next.length > previous.length) {
		return next.slice(previous.length);
	}
	return next;
}

function PipelineCanvas() {
	const navigate = useNavigate();
	const [searchParams] = useSearchParams();
	const wrapperRef = useRef<HTMLDivElement>(null);
	const editor = useFlowEditor();
	const editorRef = useRef(editor);
	const { modal } = App.useApp();
	const queryAssetIds = useMemo(
		() => parseAssetIds(searchParams.get("asset_ids")),
		[searchParams],
	);
	const [nodes, setNodes] = useState<PipelineFlowNode[]>([]);
	const [edges, setEdges] = useState<PipelineFlowEdge[]>([]);
	const [pipelineName, setPipelineName] = useState("my-pipeline");
	const pipelineNameReplaceRef = useRef(false);
	const workflowNameReplaceRef = useRef(false);
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
	const [templateVersions, setTemplateVersions] = useState<PipelineTemplate[]>(
		[],
	);
	const [selectedTemplateVersionId, setSelectedTemplateVersionId] = useState<
		string | null
	>(null);
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
	// Asset selection for deploy modal
	const [selectedAssetIds, setSelectedAssetIds] =
		useState<string[]>(queryAssetIds);
	const [assetPickerResetKey, setAssetPickerResetKey] = useState(0);
	const [executionTargets, setExecutionTargets] = useState<ExecutionTarget[]>(
		[],
	);
	const [selectedTargetId, setSelectedTargetId] = useState("default");
	const [selectedNodeAsset, setSelectedNodeAsset] = useState<Asset | null>(
		null,
	);
	const [selectedNodeAssetLoading, setSelectedNodeAssetLoading] =
		useState(false);
	const [selectedNodeAssetError, setSelectedNodeAssetError] = useState<
		string | null
	>(null);
	const selectedExecutionTarget = useMemo(
		() =>
			executionTargets.find((target) => target.id === selectedTargetId) ?? null,
		[executionTargets, selectedTargetId],
	);

	const flattenNodes = useMemo(() => toRecord(nodes), [nodes]);
	const flattenEdges = useMemo(() => toRecord(edges), [edges]);
	const nodeTypes = useMemo(() => PIPELINE_NODE_TYPES, []);

	useEffect(() => {
		editorRef.current = editor;
	}, [editor]);

	useEffect(() => {
		setSelectedAssetIds(queryAssetIds);
	}, [queryAssetIds]);

	useEffect(() => {
		let alive = true;
		listExecutionTargets()
			.then((targets) => {
				if (!alive) return;
				setExecutionTargets(targets);
				const defaultTarget =
					targets.find((target) => target.isDefault) ?? targets[0];
				if (defaultTarget) setSelectedTargetId(defaultTarget.id);
			})
			.catch(() => {
				if (!alive) return;
				setExecutionTargets([]);
			});
		return () => {
			alive = false;
		};
	}, []);

	useEffect(() => {
		setSelectedNode((prev: PipelineFlowNode | null) => {
			if (!prev) return null;
			return nodes.find((node) => node.id === prev.id) ?? null;
		});
	}, [nodes]);

	const loadPipelineToCanvas = useCallback((pipeline: Pipeline) => {
		const { nodes: n, edges: e } = fromTranspilerPipeline(pipeline);
		setNodes(n);
		setEdges(e);
		setEditingNodeId(null);
		if (pipeline.name) setPipelineName(pipeline.name);
		setSelectedNode(null);
		editorRef.current.deselectAll();
		setJsonOutput(null);
	}, []);

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
			setTemplateVersions((prev) => (prev.length > 0 ? [] : prev));
			setSelectedTemplateVersionId((prev) => (prev !== null ? null : prev));
			loadPipelineFromSessionStorage();
			return;
		}

		let cancelled = false;
		setTemplateLoading(true);
		Promise.all([getPipeline(templateId), listPipelineVersions(templateId)])
			.then(([template, versions]) => {
				if (cancelled) return;
				loadPipelineToCanvas(template.pipeline);
				setPipelineName(template.name);
				setTemplateVersions(versions);
				setSelectedTemplateVersionId(template.id);
			})
			.catch((err) => {
				if (cancelled) return;
				message.error(`模板加载失败: ${String(err)}`);
				setTemplateVersions([]);
				setSelectedTemplateVersionId(null);
				loadPipelineFromSessionStorage();
			})
			.finally(() => {
				if (!cancelled) setTemplateLoading(false);
			});

		return () => {
			cancelled = true;
		};
	}, [loadPipelineFromSessionStorage, loadPipelineToCanvas, templateId]);

	const handleTemplateVersionChange = useCallback(
		(versionId: string) => {
			const version = templateVersions.find((item) => item.id === versionId);
			if (!version) return;
			setSelectedTemplateVersionId(versionId);
			loadPipelineToCanvas(version.pipeline);
			setPipelineName(version.name);
			navigate(`/pipeline?templateId=${encodeURIComponent(versionId)}`, {
				replace: true,
			});
		},
		[loadPipelineToCanvas, navigate, templateVersions],
	);

	const componentById = useMemo(() => {
		const map = new Map<string, RegisteredComponent>();
		for (const component of registeredComponents) {
			map.set(component.id, component);
		}
		return map;
	}, [registeredComponents]);

	const onDragStart = useCallback((e: DragEvent, comp: RegisteredComponent) => {
		e.dataTransfer.setData("application/databrew-component-id", comp.id);
		e.dataTransfer.setData("text/plain", comp.id);
		e.dataTransfer.effectAllowed = "copy";
	}, []);

	const onDragOver = useCallback((event: DragEvent) => {
		event.preventDefault();
		event.dataTransfer.dropEffect = "copy";
	}, []);

	const addComponentToCanvas = useCallback(
		(comp: RegisteredComponent, clientPosition?: { x: number; y: number }) => {
			const bounds = wrapperRef.current?.getBoundingClientRect();
			const screenPosition =
				clientPosition ??
				(bounds
					? {
							x: bounds.left + bounds.width / 2 + nodes.length * 24,
							y: bounds.top + 140 + nodes.length * 24,
						}
					: { x: 360 + nodes.length * 24, y: 240 + nodes.length * 24 });
			const position = editor.screenToFlowPosition(screenPosition);
			const newNode = createPipelineNode(comp, position.x, position.y);
			editor.addNode(newNode);
			editor.selectElements([newNode.id]);
			setSelectedNode(newNode);
			setEditingNodeId(null);
		},
		[editor, nodes.length],
	);

	const onDrop = useCallback(
		(event: DragEvent) => {
			event.preventDefault();
			event.stopPropagation();
			const componentId =
				event.dataTransfer.getData("application/databrew-component-id") ||
				event.dataTransfer.getData("text/plain");
			if (!componentId) return;
			const comp = componentById.get(componentId);
			if (!comp) return;
			addComponentToCanvas(comp, {
				x: event.clientX,
				y: event.clientY,
			});
		},
		[addComponentToCanvas, componentById],
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
			const saved = await savePipeline(pipelineName, buildPipelineJSON());
			setSelectedTemplateVersionId(saved.id);
			setTemplateVersions((prev) => {
				const withoutSaved = prev.filter((item) => item.id !== saved.id);
				return [saved, ...withoutSaved].sort((a, b) => b.version - a.version);
			});
			navigate(`/pipeline?templateId=${encodeURIComponent(saved.id)}`, {
				replace: true,
			});
			message.success(`已保存为 v${saved.version}，可在「流水线」页签管理`);
		} catch (err) {
			message.error(`保存失败: ${String(err)}`);
		}
	}, [pipelineName, buildPipelineJSON, navigate]);

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
		setAssetPickerResetKey((key) => key + 1);
	}, [pipelineName, nodes.length]);

	const closeDeployDialog = useCallback(() => {
		setDeployDialog({
			open: false,
			deploying: false,
			done: false,
			name: "",
			mode: "edit",
		});
		setSelectedAssetIds([]);
		setAssetPickerResetKey((key) => key + 1);
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
		},
		canDeploy,
		modalOpen: deployDialog.open || importModalOpen,
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
				selectedAssetIds,
				selectedTargetId,
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
	}, [
		buildPipelineJSON,
		deployDialog.name,
		pipelineName,
		selectedAssetIds,
		selectedTargetId,
	]);

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
	const deployDisabledReason = canDeploy
		? "保存并部署为 Argo Workflow (⌘/Ctrl+D)"
		: "请先从左侧拖入至少一个组件到画布，再保存或部署";
	const markReplaceOnNextEdit = (
		event: FocusEvent<HTMLInputElement>,
		replaceRef: MutableRefObject<boolean>,
	) => {
		replaceRef.current = true;
		event.target.select();
	};

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
						onChange={(e) => {
							const next = pipelineNameReplaceRef.current
								? replaceAppendedValue(pipelineName, e.target.value)
								: e.target.value;
							pipelineNameReplaceRef.current = false;
							setPipelineName(next);
						}}
						onFocus={(e) => markReplaceOnNextEdit(e, pipelineNameReplaceRef)}
						onBlur={() => {
							pipelineNameReplaceRef.current = false;
						}}
						maxLength={48}
						placeholder="输入流水线名称"
						aria-label="流水线名称"
						autoComplete="off"
						className="pipeline-toolbar__name-input"
						size="small"
					/>
					{templateVersions.length > 0 ? (
						<Select
							size="small"
							className="pipeline-toolbar__version-select"
							value={selectedTemplateVersionId ?? undefined}
							onChange={handleTemplateVersionChange}
							options={templateVersions.map((version) => ({
								value: version.id,
								label: `v${version.version}`,
							}))}
							aria-label="流水线模板版本"
						/>
					) : null}
				</div>
				<div className="pipeline-toolbar__actions">
					<Tooltip title={deployDisabledReason}>
						<span>
							<Button
								size="small"
								type="primary"
								className="pipeline-toolbar__deploy"
								icon={<PlayCircleOutlined />}
								onClick={openDeployDialog}
								disabled={!canDeploy}
							>
								部署
							</Button>
						</span>
					</Tooltip>
					<Tooltip title="保存 (⌘/Ctrl+S)">
						<Button
							size="small"
							type="primary"
							ghost
							icon={<SaveOutlined />}
							onClick={handleSave}
						>
							保存
						</Button>
					</Tooltip>
					<div className="pipeline-toolbar__secondary-actions">
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
					</div>
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
					onAddComponent={addComponentToCanvas}
					loading={componentsLoading}
					error={componentsError}
					onRetry={reloadComponents}
				/>
				<div
					className={[
						"canvas-wrapper",
						isCanvasEmpty ? "canvas-wrapper--empty" : "",
					]
						.filter(Boolean)
						.join(" ")}
					ref={wrapperRef}
					style={{ flex: 1, height: "100%", position: "relative" }}
					onDrop={onDrop}
					onDragOver={onDragOver}
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
							title="添加组件开始设计"
							description="从左侧组件栏点击或拖拽步骤，连接节点后保存。"
							hint="保存后可在「流水线」页签打开、运行或继续编辑。"
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
				<aside
					className={[
						"config-panel",
						selectedNode ? "config-panel--with-node" : "config-panel--empty",
					]
						.filter(Boolean)
						.join(" ")}
				>
					{selectedNode ? (
						<div className="config-panel__node">
							<div>
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
							</div>
						</div>
					) : (
						<PipelineEmptyState
							variant="config"
							title="节点配置"
							description="选中画布节点后，在此查看关联资产与节点详情。"
							hint="已保存流水线请到「流水线」页签管理。"
						/>
					)}
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
					粘贴 Pipeline
					JSON。导入成功后会覆盖当前画布内容，请先确认当前修改已保存。
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
								选择执行目标和资产后，将流水线转换为 Argo Workflow 并提交到
								Kubernetes 集群。
							</Typography.Paragraph>
							<Alert
								type={selectedAssetIds.length > 0 ? "success" : "warning"}
								showIcon
								message={
									selectedAssetIds.length > 0
										? `将处理 ${selectedAssetIds.length} 个资产`
										: "当前是 no-asset run：不会注入资产环境变量。"
								}
								style={{ marginBottom: 16 }}
							/>
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
												name: workflowNameReplaceRef.current
													? replaceAppendedValue(prev.name, e.target.value)
													: e.target.value,
											}))
										}
										onFocus={(e) =>
											markReplaceOnNextEdit(e, workflowNameReplaceRef)
										}
										onBlur={() => {
											workflowNameReplaceRef.current = false;
										}}
										maxLength={48}
										placeholder="留空则使用当前流水线名称"
										size="small"
									/>
								</label>
								<label
									htmlFor="pp-execution-target"
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
									执行目标
									<Select
										id="pp-execution-target"
										value={selectedTargetId}
										onChange={setSelectedTargetId}
										size="small"
										options={(executionTargets.length > 0
											? executionTargets
											: [
													{
														id: "default",
														name: "Default Argo target",
														cluster: "default",
														namespace: "default",
														status: "unavailable",
														isDefault: true,
														argoServerConfigured: false,
													} satisfies ExecutionTarget,
												]
										).map((target) => ({
											value: target.id,
											label: `${target.isDefault ? "默认目标" : target.name} · ${target.namespace}`,
											disabled: target.status !== "available",
										}))}
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
									<span>{nodes.length} 个步骤</span>
									<span style={{ fontSize: 3, color: "#cbd5e1" }}>•</span>
									<span>{edges.length} 条连线</span>
								</div>
								<AssetPicker
									selectedIds={selectedAssetIds}
									onSelectionChange={setSelectedAssetIds}
									maxHeight={180}
									resetKey={assetPickerResetKey}
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
									{selectedAssetIds.length > 0 ? "运行资产" : "无资产运行"}
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
								先确认运行摘要；需要排查 Argo 配置时再展开原始 YAML。
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
								<Space direction="vertical" size={12} style={{ width: "100%" }}>
									<div
										style={{
											display: "grid",
											gridTemplateColumns:
												"repeat(auto-fit, minmax(150px, 1fr))",
											gap: 8,
										}}
									>
										{[
											["模板", currentTemplateLabel],
											["步骤", `${nodes.length} 个`],
											["连线", `${edges.length} 条`],
											[
												"资产",
												selectedAssetIds.length > 0
													? `${selectedAssetIds.length} 个`
													: "无资产运行",
											],
											[
												"目标",
												selectedExecutionTarget
													? `${selectedExecutionTarget.namespace}`
													: "默认目标",
											],
										].map(([label, value]) => (
											<div
												key={label}
												style={{
													border: "1px solid #e2e8f0",
													borderRadius: 8,
													padding: "10px 12px",
													background: "#f8fafc",
												}}
											>
												<Typography.Text
													type="secondary"
													style={{ display: "block", fontSize: 12 }}
												>
													{label}
												</Typography.Text>
												<Typography.Text strong>{value}</Typography.Text>
											</div>
										))}
									</div>
									{selectedExecutionTarget ? (
										<Typography.Text type="secondary" style={{ fontSize: 12 }}>
											执行目标：{selectedExecutionTarget.cluster}/
											{selectedExecutionTarget.namespace}
										</Typography.Text>
									) : null}
									<Collapse
										size="small"
										items={[
											{
												key: "manifest",
												label: "原始 Argo YAML",
												children: (
													<pre
														style={{
															margin: 0,
															padding: 12,
															background: "#0f172a",
															color: "#e2e8f0",
															borderRadius: 8,
															overflow: "auto",
															maxHeight: 300,
															fontSize: 12,
															fontFamily:
																'"SF Mono", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
															lineHeight: 1.5,
															whiteSpace: "pre",
														}}
													>
														{deployDialog.previewManifest || "（暂无内容）"}
													</pre>
												),
											},
										]}
									/>
								</Space>
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
									{selectedAssetIds.length > 0 ? "运行资产" : "无资产运行"}
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
								<Typography.Text type="secondary" style={{ fontSize: 12 }}>
									{deployDialog.result.assetCount
										? `${deployDialog.result.assetCount} 个资产`
										: "无资产"}
									{deployDialog.result.executionTarget
										? ` · ${deployDialog.result.executionTarget.cluster}/${deployDialog.result.executionTarget.namespace}`
										: ""}
								</Typography.Text>
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
									<span>{deployDialog.result.nodeCount} 个步骤</span>
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
												`/pipeline/executions/${deployDialog.result?.workflowName}`,
											);
										}}
									>
										查看 Workflow
									</Button>
									<Button
										onClick={() => {
											closeDeployDialog();
											navigate("/pipeline?tab=executions");
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

type PipelineTab = "design" | "pipelines" | "executions" | "components";

function resolvePipelineTab(raw: string | null): PipelineTab {
	switch (raw) {
		case "templates":
		case "pipelines":
			return "pipelines";
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
						key: "pipelines",
						label: tabLabel("流水线", "管理已保存的流水线"),
						children: (
							<div className="pipeline-tab-content pipeline-tab-content--panel pipeline-tab-content--management">
								<DeployPanel variant="full" />
							</div>
						),
					},
					{
						key: "executions",
						label: tabLabel("执行记录", "查看和管理流水线运行"),
						children: (
							<div className="pipeline-tab-content pipeline-tab-content--panel pipeline-tab-content--executions">
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
